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

	if err := s.ensureListingExists(ctx, listingID); err != nil {
		s.log.Logf("ERROR listing not found for media upload listing=%s: %v", listingID, err)
		return nil, err
	}

	s.log.Logf("INFO uploading %d media items for listing=%s", len(media), listingID)

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
			s.log.Logf("ERROR failed to generate signed URL for listing=%s key=%s: %v", listingID, objKey, err)
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
		s.log.Logf("ERROR failed to add media to listing=%s: %v", listingID, err)
		return nil, err
	}

	s.log.Logf("INFO successfully prepared %d media uploads for listing=%s", len(result), listingID)
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

	if err := s.ensureListingExists(ctx, listingID); err != nil {
		s.log.Logf("ERROR listing not found for media update listing=%s: %v", listingID, err)
		return err
	}

	if err := s.repo.UpdateListingMedia(ctx, listingID, mediaID, updateMap); err != nil {
		s.log.Logf("ERROR failed to update media listing=%s media=%s: %v", listingID, mediaID, err)
		return err
	}

	s.log.Logf("INFO updated media listing=%s media=%s", listingID, mediaID)
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

	if err := s.ensureListingExists(ctx, listingID); err != nil {
		s.log.Logf("ERROR listing not found for media deletion listing=%s: %v", listingID, err)
		return err
	}

	s.log.Logf("INFO deleting %d media items from listing=%s", len(media), listingID)

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
		s.log.Logf("ERROR failed to delete media from listing=%s: %v", listingID, err)
		return err
	}

	s.log.Logf("INFO deleted %d media items from listing=%s", len(mediaIDs), listingID)
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

	s.log.Logf("INFO finalizing %d media items for listing=%s", len(data.MediaKeys), data.ListingID)

	if err := s.ensureListingExists(ctx, data.ListingID); err != nil {
		s.log.Logf("ERROR listing not found for media finalization listing=%s: %v", data.ListingID, err)
		return err
	}

	media, err := s.repo.ListListingMedia(ctx, data.ListingID)
	if err != nil {
		s.log.Logf("ERROR failed to list media for finalization listing=%s: %v", data.ListingID, err)
		return err
	}
	if len(media) == 0 {
		s.log.Logf("ERROR no media found for finalization listing=%s", data.ListingID)
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
			s.log.Logf("ERROR media key not found for finalization listing=%s key=%s", data.ListingID, key)
			return domain.ErrMediaNotFound
		}

		exists, err := s.storage.ObjectExists(ctx, key)
		if err != nil {
			s.log.Logf("ERROR failed to check object existence listing=%s key=%s: %v", data.ListingID, key, err)
			return err
		}
		if !exists {
			s.log.Logf("ERROR media object not found in storage listing=%s key=%s", data.ListingID, key)
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
			s.log.Logf("ERROR failed to mark media as uploaded listing=%s media=%s: %v", data.ListingID, m.ID, err)
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
			s.log.Logf("ERROR failed to enqueue thumbnail job listing=%s: %v", data.ListingID, err)
			return err
		}
		s.log.Logf("INFO enqueued thumbnail generation for listing=%s count=%d", data.ListingID, len(data.MediaKeys))
	}

	s.log.Logf("INFO finalized %d media items for listing=%s", len(data.MediaKeys), data.ListingID)
	return nil
}
