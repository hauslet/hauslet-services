package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"hauslet/internal/modules/property/repository"
	"hauslet/internal/modules/property/repository/schema"
	"hauslet/internal/platform/storage"
	listingJob "hauslet/internal/queue/jobs/listing"

	"github.com/disintegration/imaging"
	"github.com/go-pkgz/lgr"
	"github.com/google/uuid"
)

// ListingMediaThumbnailHandler generates thumbnails for listing media.
type ListingMediaThumbnailHandler struct {
	repo    repository.Repository
	storage *storage.R2Storage
	log     *lgr.Logger
	subject string
}

// NewListingMediaThumbnailHandler constructs the handler.
func NewListingMediaThumbnailHandler(
	repo repository.Repository,
	storage *storage.R2Storage,
	log *lgr.Logger,
	subject string,
) *ListingMediaThumbnailHandler {
	return &ListingMediaThumbnailHandler{
		repo:    repo,
		storage: storage,
		log:     log,
		subject: subject,
	}
}

// JobType returns the job type processed.
func (h *ListingMediaThumbnailHandler) JobType() string {
	return listingJob.ListingMediaThumbnailJobType
}

// Subject returns the subscribed subject.
func (h *ListingMediaThumbnailHandler) Subject() string {
	return h.subject
}

// Handle generates thumbnails for each finalized media key.
func (h *ListingMediaThumbnailHandler) Handle(ctx context.Context, data []byte) error {
	var job listingJob.ListingMediaThumbnailJob
	if err := json.Unmarshal(data, &job); err != nil {
		return fmt.Errorf("failed to unmarshal thumbnail job: %w", err)
	}
	if err := job.Validate(); err != nil {
		return fmt.Errorf("invalid thumbnail job: %w", err)
	}

	h.log.Logf(
		"INFO listing thumbnail job started listing=%s media_keys=%d",
		job.ListingID,
		len(job.MediaKeys),
	)

	media, err := h.repo.ListListingMedia(ctx, job.ListingID)
	if err != nil {
		return fmt.Errorf("list listing media: %w", err)
	}
	if len(media) == 0 {
		return fmt.Errorf("no media found for listing %s", job.ListingID)
	}

	byKey := make(map[string]*schema.ListingMedia, len(media))
	for i := range media {
		byKey[media[i].Key] = &media[i]
	}

	for _, key := range job.MediaKeys {
		m, ok := byKey[key]
		if !ok {
			return fmt.Errorf("media key not found: %s", key)
		}

		if m.Type != schema.MediaTypeImage {
			h.log.Logf("INFO skipping non-image media %s (%s)", key, m.Type)
			continue
		}

		h.log.Logf(
			"INFO processing media thumbnail listing=%s media=%s",
			job.ListingID,
			m.Key,
		)

		if err := h.processThumbnail(ctx, job.ListingID, m); err != nil {
			return err
		}
	}

	h.log.Logf(
		"INFO listing thumbnail job completed listing=%s",
		job.ListingID,
	)

	return nil
}

func (h *ListingMediaThumbnailHandler) processThumbnail(
	ctx context.Context,
	listingID uuid.UUID,
	media *schema.ListingMedia,
) error {
	// Avoid regeneration if thumbnails already exist
	if len(media.Thumbnails) > 0 {
		h.log.Logf("INFO thumbnails already present for %s, skipping", media.Key)
		return nil
	}

	raw, _, err := h.storage.GetObject(ctx, media.Key)
	if err != nil {
		return fmt.Errorf("fetch object: %w", err)
	}

	// Decode image with automatic EXIF orientation correction
	img, err := imaging.Decode(bytes.NewReader(raw), imaging.AutoOrientation(true))
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}

	targets := []struct {
		name   string
		width  int
		height int
	}{
		{"small", 320, 240},
		{"medium", 640, 480},
		{"large", 1280, 960},
	}

	var wg sync.WaitGroup
	thumbChan := make(chan schema.Thumbnail, len(targets))
	errChan := make(chan error, len(targets))

	for _, target := range targets {
		wg.Add(1)
		go func(name string, width, height int) {
			defer wg.Done()

			thumb := imaging.Fit(img, width, height, imaging.Lanczos)

			var buf bytes.Buffer
			if err := imaging.Encode(&buf, thumb, imaging.JPEG, imaging.JPEGQuality(85)); err != nil {
				errChan <- fmt.Errorf("encode thumbnail %s: %w", name, err)
				return
			}

			key := thumbnailKey(media.Key, name)
			if err := h.storage.UploadObject(
				ctx,
				key,
				bytes.NewReader(buf.Bytes()),
				"image/jpeg",
			); err != nil {
				errChan <- fmt.Errorf("upload thumbnail %s: %w", name, err)
				return
			}

			thumbChan <- schema.Thumbnail{
				Key:      key,
				Width:    thumb.Bounds().Dx(),
				Height:   thumb.Bounds().Dy(),
				Size:     int64(buf.Len()),
				MimeType: "image/jpeg",
			}
		}(target.name, target.width, target.height)
	}

	wg.Wait()
	close(thumbChan)
	close(errChan)

	for err := range errChan {
		return err
	}

	thumbs := make(schema.ThumbnailMap)
	for thumb := range thumbChan {
		thumbs[strings.TrimPrefix(
			filepath.Base(thumb.Key),
			fmt.Sprintf(
				"%s_thumb_",
				strings.TrimSuffix(filepath.Base(media.Key), filepath.Ext(media.Key)),
			),
		)] = thumb
	}

	h.log.Logf(
		"INFO thumbnails generated media=%s count=%d",
		media.Key,
		len(thumbs),
	)

	return h.repo.UpdateListingMedia(ctx, listingID, media.ID, map[string]any{
		"thumbnails": thumbs,
	})
}

func thumbnailKey(baseKey, size string) string {
	ext := filepath.Ext(baseKey)
	base := strings.TrimSuffix(baseKey, ext)
	return fmt.Sprintf("%s_thumb_%s.jpg", base, size)
}
