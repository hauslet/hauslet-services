package service

import (
	"context"
	"fmt"
	"time"

	"hauslet/internal/modules/property/domain"
	"hauslet/internal/modules/property/repository/schema"
	listingJob "hauslet/internal/queue/jobs/listing"

	"github.com/google/uuid"
)

// UploadListingMedia adds media to a listing.
func (s *ServiceImpl) UploadListingMedia(ctx context.Context,
	listingID uuid.UUID,
	media []domain.ListingMediaInput,
) ([]domain.ListingMediaResult, error) {
	if listingID == uuid.Nil {
		return nil, domain.ErrInvalidListingID
	}

	if len(media) == 0 {
		return nil, fmt.Errorf("no media input found")
	}

	listing, err := s.ensureListing(ctx, listingID, true)
	if err != nil {
		s.log.Error("listing not found for media upload", "listing_id", listingID, "error", err)
		return nil, err
	}

	// Check subscription limits - count existing photos + new photos
	existingMedia, err := s.repo.ListListingMedia(ctx, listingID)
	if err != nil {
		s.log.Error("failed to get existing media for quota check", "listing_id", listingID, "error", err)
		return nil, err
	}

	// Count existing photos
	existingPhotoCount := 0
	for _, m := range existingMedia {
		if m.Type == schema.MediaTypeImage {
			existingPhotoCount++
		}
	}

	// Count new photos being uploaded
	newPhotoCount := 0
	for _, m := range media {
		if m.Type == domain.MediaTypeImage {
			newPhotoCount++
		}
	}

	totalPhotoCount := existingPhotoCount + newPhotoCount

	// Check if user can add this many photos
	canAdd, err := s.subscriptionService.CanAddPhotos(ctx, listing.OwnerID, listingID, totalPhotoCount)
	if err != nil {
		s.log.Error("failed to check photo limit", "owner_id", listing.OwnerID, "listing_id", listingID, "error", err)
		return nil, err
	}
	if !canAdd {
		s.log.Warn("photo limit exceeded",
			"owner_id", listing.OwnerID,
			"listing_id", listingID,
			"existing_photos", existingPhotoCount,
			"new_photos", newPhotoCount,
			"total", totalPhotoCount,
		)
		return nil, fmt.Errorf("photo limit exceeded - your plan allows fewer photos per listing (total would be %d)", totalPhotoCount)
	}

	s.log.Info("uploading media items for listing",
		"count", len(media),
		"listing_id", listingID,
		"existing_photos", existingPhotoCount,
		"new_photos", newPhotoCount,
		"total_photos", totalPhotoCount,
	)

	mapped := make([]schema.ListingMedia, len(media))
	result := make([]domain.ListingMediaResult, len(media))

	for i, m := range media {
		objKey := fmt.Sprintf("listings/%s/media/%s/%s", listingID.String(),
			m.Type, time.Now().Format("20060102T150405")+"_"+m.Filename)
		d := 15 * time.Minute
		if m.Type == domain.MediaTypeVideo {
			d = 30 * time.Minute
		}
		url, err := s.storage.GenerateSignedUploadURL(ctx, objKey, string(m.Type), d)
		if err != nil {
			s.log.Error("failed to generate signed URL", "listing_id", listingID, "key", objKey, "error", err)
			return nil, err
		}
		mapped[i] = schema.ListingMedia{
			ID:             uuid.New(),
			ListingID:      listingID,
			Group:          m.Group,
			Caption:        m.Caption,
			MimeType:       m.MimeType,
			Duration:       m.Duration,
			SizeBytes:      m.SizeBytes,
			IsPrimary:      m.IsPrimary,
			IsGroupCover:   m.IsGroupCover,
			Order:          m.Order,
			Key:            objKey,
			Type:           schema.MediaType(m.Type),
			UrlGeneratedAt: time.Now(),
		}
		result[i] = domain.ListingMediaResult{
			ID:       mapped[i].ID,
			URL:      url,
			Filename: m.Filename,
			Key:      objKey,
		}
	}

	if err := s.repo.AddListingMedia(ctx, listingID, mapped); err != nil {
		s.log.Error("failed to add media to listing", "listing_id", listingID, "error", err)
		return nil, err
	}

	// Auto-unpublish if listing is live (media addition is material).
	if listing.Published && listing.Status == domain.StatusActive {
		if err := s.unpublishAndEnqueueModeration(ctx, listing); err != nil {
			return nil, err
		}
	}

	s.log.Info("successfully prepared media uploads", "count", len(result), "listing_id", listingID)
	s.invalidateListingCache(ctx, listingID, listing.Slug, s.getPropertyPublicID(ctx, listing.PropertyID))
	return result, nil
}

// UpdateListingMedia updates specific media for a listing.
func (s *ServiceImpl) UpdateListingMedia(ctx context.Context,
	listingID uuid.UUID,
	mediaID uuid.UUID,
	updates domain.ListingMediaUpdateInput) error {
	if listingID == uuid.Nil {
		return domain.ErrInvalidListingID
	}
	if mediaID == uuid.Nil {
		return domain.ErrMediaNotFound
	}
	// Ensure listing exists
	listing, err := s.ensureListing(ctx, listingID, true)
	if err != nil {
		s.log.Error("listing not found for media update", "listing_id", listingID, "error", err)
		return err
	}
	updateMap := make(map[string]any)

	if updates.Caption != nil {
		updateMap["caption"] = *updates.Caption
	}
	if updates.IsPrimary != nil {
		updateMap["is_primary"] = *updates.IsPrimary
	}
	if updates.IsGroupCover != nil {
		updateMap["is_group_cover"] = *updates.IsGroupCover
	}
	if updates.Order != nil {
		updateMap["order"] = *updates.Order
	}

	if err := s.repo.UpdateListingMedia(ctx, listingID, mediaID, updateMap); err != nil {
		s.log.Error("failed to update media", "listing_id", listingID, "media_id", mediaID, "error", err)
		return err
	}

	// Auto-unpublish if listing is live (media change is material).
	if listing.Published && listing.Status == domain.StatusActive {
		if err := s.unpublishAndEnqueueModeration(ctx, listing); err != nil {
			return err
		}
	}

	s.log.Info("updated media", "listing_id", listingID, "media_id", mediaID)
	s.invalidateListingCache(ctx, listingID, listing.Slug, s.getPropertyPublicID(ctx, listing.PropertyID))
	return nil
}

// DeleteListingMedia deletes media from a listing.
func (s *ServiceImpl) DeleteListingMedia(ctx context.Context,
	listingID uuid.UUID, media []domain.ListingMediaDeleteInput) error {

	if listingID == uuid.Nil {
		return domain.ErrInvalidListingID
	}

	if len(media) == 0 {
		return domain.ErrMediaNotFound
	}

	listing, err := s.ensureListing(ctx, listingID, true)
	if err != nil {
		s.log.Error("listing not found for media deletion", "listing_id", listingID, "error", err)
		return err
	}

	s.log.Info("deleting media items from listing", "count", len(media), "listing_id", listingID)

	var mediaIDs []uuid.UUID
	for _, m := range media {
		if m.MediaID == uuid.Nil {
			return domain.ErrMediaNotFound
		}
		mediaIDs = append(mediaIDs, m.MediaID)

		// Delete from storage asynchronously
		if s.storage != nil && m.Key != "" {
			go func(key string) {
				_ = s.storage.DeleteObject(context.Background(), key)
			}(m.Key)
		}
	}

	if err := s.repo.DeleteListingMedia(ctx, listingID, mediaIDs); err != nil {
		s.log.Error("failed to delete media from listing", "listing_id", listingID, "error", err)
		return err
	}

	// Auto-unpublish if listing is live (media removal is material).
	if listing.Published && listing.Status == domain.StatusActive {
		if err := s.unpublishAndEnqueueModeration(ctx, listing); err != nil {
			return err
		}
	}

	s.log.Info("deleted media items from listing", "count", len(mediaIDs), "listing_id", listingID)
	s.invalidateListingCache(ctx, listingID, listing.Slug, s.getPropertyPublicID(ctx, listing.PropertyID))
	return nil
}

// ListListingMedia retrieves all media for a listing.
func (s *ServiceImpl) ListListingMedia(ctx context.Context,
	listingID uuid.UUID) ([]domain.ListingMedia, error) {
	if listingID == uuid.Nil {
		return nil, domain.ErrInvalidListingID
	}

	if err := s.ensureListingExists(ctx, listingID); err != nil {
		return nil, err
	}

	media, err := s.repo.ListListingMedia(ctx, listingID)
	if err != nil {
		return nil, err
	}

	if len(media) == 0 {
		return []domain.ListingMedia{}, nil
	}

	result := make([]domain.ListingMedia, len(media))
	for i, m := range media {
		result[i] = *domain.MapListingMediaFromSchema(&m)
	}
	return result, nil
}

// ensureListingExists is a lightweight existence check for listing-scoped operations.
func (s *ServiceImpl) ensureListingExists(ctx context.Context, id uuid.UUID) error {
	exists, err := s.repo.ListingExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return domain.ErrListingNotFound
	}
	return nil
}

// FinalizeListingMedia updates media records after upload is complete.
func (s *ServiceImpl) FinalizeListingMedia(ctx context.Context, data domain.FinalizedListingMedia) error {
	if data.ListingID == uuid.Nil {
		return domain.ErrInvalidListingID
	}
	if len(data.MediaKeys) == 0 {
		return fmt.Errorf("media keys are required")
	}

	s.log.Info("finalizing media items for listing", "count", len(data.MediaKeys), "listing_id", data.ListingID)

	listing, err := s.ensureListing(ctx, data.ListingID, true)
	if err != nil {
		s.log.Error("listing not found for media finalization", "listing_id", data.ListingID, "error", err)
		return err
	}

	media, err := s.repo.ListListingMedia(ctx, data.ListingID)
	if err != nil {
		s.log.Error("failed to list media for finalization", "listing_id", data.ListingID, "error", err)
		return err
	}
	if len(media) == 0 {
		s.log.Error("no media found for finalization", "listing_id", data.ListingID)
		return domain.ErrMediaNotFound
	}

	byKey := make(map[string]*schema.ListingMedia, len(media))
	for i := range media {
		byKey[media[i].Key] = &media[i]
	}

	now := time.Now()
	for _, key := range data.MediaKeys {
		m, ok := byKey[key]
		if !ok {
			s.log.Error("media key not found for finalization", "listing_id", data.ListingID, "key", key)
			return domain.ErrMediaNotFound
		}

		exists, err := s.storage.ObjectExists(ctx, key)
		if err != nil {
			s.log.Error("failed to check object existence", "listing_id", data.ListingID, "key", key, "error", err)
			return err
		}
		if !exists {
			s.log.Error("media object not found in storage", "listing_id", data.ListingID, "key", key)
			return fmt.Errorf("media object not found")
		}

		updates := map[string]any{
			"uploaded":    true,
			"uploaded_at": now,
		}
		if m.Key == "" {
			updates["key"] = key
		}

		if err := s.repo.UpdateListingMedia(ctx, data.ListingID, m.ID, updates); err != nil {
			s.log.Error("failed to mark media as uploaded", "listing_id", data.ListingID, "media_id", m.ID, "error", err)
			return err
		}

		m.Uploaded = true
	}

	// Ensure only one primary across listing
	var primarySet bool
	for _, m := range media {
		if !m.IsPrimary {
			continue
		}
		if !primarySet {
			primarySet = true
			continue
		}
		_ = s.repo.UpdateListingMedia(ctx, data.ListingID, m.ID, map[string]any{"is_primary": false})
	}

	// Ensure only one group cover per group
	groupPrimary := make(map[string]bool)
	for _, m := range media {
		if !m.IsGroupCover {
			continue
		}
		groupKey := ""
		if m.Group != nil {
			groupKey = *m.Group
		}

		if !groupPrimary[groupKey] {
			groupPrimary[groupKey] = true
			continue
		}

		_ = s.repo.UpdateListingMedia(ctx, data.ListingID, m.ID, map[string]any{"is_group_cover": false})
	}

	if s.queue != nil && s.thumbnailSubject != "" {
		job := listingJob.ListingMediaThumbnailJob{
			ListingID: data.ListingID,
			MediaKeys: data.MediaKeys,
		}
		if err := s.queue.Publish(ctx, s.thumbnailSubject, job); err != nil {
			s.log.Error("failed to enqueue thumbnail job", "listing_id", data.ListingID, "error", err)
			return err
		}
		s.log.Info("enqueued thumbnail generation", "listing_id", data.ListingID, "count", len(data.MediaKeys))
	}

	s.log.Info("finalized media items", "count", len(data.MediaKeys), "listing_id", data.ListingID)

	// Auto-unpublish if listing is live (finalizing media is material).
	if listing.Published && listing.Status == domain.StatusActive {
		if err := s.unpublishAndEnqueueModeration(ctx, listing); err != nil {
			return err
		}
	}

	s.invalidateListingCache(ctx, data.ListingID, listing.Slug, s.getPropertyPublicID(ctx, listing.PropertyID))
	return nil
}
