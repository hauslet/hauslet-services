package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"path/filepath"
	"strings"
	"sync"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"hauslet/internal/platform/storage"
	"hauslet/internal/property/repository"
	"hauslet/internal/property/repository/schema"
	"hauslet/internal/queue/jobs"

	"github.com/disintegration/imaging"
	"github.com/go-pkgz/lgr"
	"github.com/google/uuid"
	"github.com/rwcarlsen/goexif/exif"
)

// ListingMediaThumbnailHandler generates thumbnails for listing media.
type ListingMediaThumbnailHandler struct {
	repo    repository.Repository
	storage *storage.R2Storage
	log     *lgr.Logger
	subject string
}

// NewListingMediaThumbnailHandler constructs the handler.
func NewListingMediaThumbnailHandler(repo repository.Repository, 
	storage *storage.R2Storage, 
	log *lgr.Logger,
	subject string) *ListingMediaThumbnailHandler {
	return &ListingMediaThumbnailHandler{
		repo:    repo,
		storage: storage,
		log:     log,
		subject: subject,
	}
}

// JobType returns the job type processed.
func (h *ListingMediaThumbnailHandler) JobType() string {
	return jobs.ListingMediaThumbnailJobType
}

// Subject returns the subscribed subject.
func (h *ListingMediaThumbnailHandler) Subject() string {
	return h.subject
}

// Handle generates thumbnails for each finalized media key.
func (h *ListingMediaThumbnailHandler) Handle(ctx context.Context, data []byte) error {
	var job jobs.ListingMediaThumbnailJob
	if err := json.Unmarshal(data, &job); err != nil {
		return fmt.Errorf("failed to unmarshal thumbnail job: %w", err)
	}
	if err := job.Validate(); err != nil {
		return fmt.Errorf("invalid thumbnail job: %w", err)
	}

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

		if err := h.processThumbnail(ctx, job.ListingID, m); err != nil {
			return err
		}
	}

	return nil
}

func (h *ListingMediaThumbnailHandler) processThumbnail(ctx context.Context, 
	listingID uuid.UUID, 
	media *schema.ListingMedia) error {
	// Avoid regeneration if thumbnails already exist
	if len(media.Thumbnails) > 0 {
		h.log.Logf("INFO thumbnails already present for %s, skipping", media.Key)
		return nil
	}

	raw, _, err := h.storage.GetObject(ctx, media.Key)
	if err != nil {
		return fmt.Errorf("fetch object: %w", err)
	}

	// Decode image with orientation correction
	img, err := decodeImageWithOrientation(raw)
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}

	// Define thumbnail targets with aspect ratio preservation
	targets := []struct {
		name   string
		width  int
		height int
	}{
		{"small", 320, 240},
		{"medium", 640, 480},
		{"large", 1280, 960},
	}

	// Generate thumbnails concurrently
	var wg sync.WaitGroup
	thumbChan := make(chan schema.Thumbnail, len(targets))
	errChan := make(chan error, len(targets))

	for _, target := range targets {
		wg.Add(1)
		go func(name string, width, height int) {
			defer wg.Done()

			// Preserve aspect ratio
			thumb := imaging.Fit(img, width, height, imaging.Lanczos)

			// Encode as JPEG with optimized quality
			var buf bytes.Buffer
			if err := imaging.Encode(&buf, thumb, imaging.JPEG, imaging.JPEGQuality(85)); err != nil {
				errChan <- fmt.Errorf("encode thumbnail %s: %w", name, err)
				return
			}

			key := thumbnailKey(media.Key, name)
			if err := h.storage.UploadObject(ctx, key, bytes.NewReader(buf.Bytes()), "image/jpeg"); err != nil {
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

	// Wait for all goroutines to complete
	wg.Wait()
	close(thumbChan)
	close(errChan)

	// Check for errors
	for err := range errChan {
		return err
	}

	// Collect thumbnails
	thumbs := make(schema.ThumbnailMap)
	for thumb := range thumbChan {
		thumbs[strings.TrimPrefix(filepath.Base(thumb.Key), fmt.Sprintf("%s_thumb_", strings.TrimSuffix(filepath.Base(media.Key), filepath.Ext(media.Key))))] = thumb
	}

	return h.repo.UpdateListingMedia(ctx, listingID, media.ID, map[string]any{
		"thumbnails": thumbs,
	})
}

// decodeImageWithOrientation decodes an image and corrects its orientation based on EXIF data
func decodeImageWithOrientation(data []byte) (image.Image, error) {
	// First decode to get the basic image
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("image decode: %w", err)
	}

	// Only JPEG and TIFF have EXIF orientation
	if format != "jpeg" && format != "tiff" {
		return img, nil
	}

	// Try to extract EXIF orientation
	exifData, err := exif.Decode(bytes.NewReader(data))
	if err != nil {
		// No EXIF data or can't decode, return original image
		return img, nil
	}

	orientation, err := exifData.Get(exif.Orientation)
	if err != nil {
		// No orientation tag, return original image
		return img, nil
	}

	val, err := orientation.Int(0)
	if err != nil {
		return img, nil
	}

	// Apply orientation correction
	return applyOrientation(img, val), nil
}

// applyOrientation applies EXIF orientation to an image
func applyOrientation(img image.Image, orientation int) image.Image {
	switch orientation {
	case 1:
		// Normal
		return img
	case 2:
		// Flip horizontal
		return imaging.FlipH(img)
	case 3:
		// Rotate 180
		return imaging.Rotate180(img)
	case 4:
		// Flip vertical
		return imaging.FlipV(img)
	case 5:
		// Rotate 90 clockwise and flip horizontal
		img = imaging.Rotate90(img)
		return imaging.FlipH(img)
	case 6:
		// Rotate 270
		return imaging.Rotate270(img)
	case 7:
		// Rotate 270 and flip horizontal
		img = imaging.Rotate270(img)
		return imaging.FlipH(img)
	case 8:
		// Rotate 90
		return imaging.Rotate90(img)
	default:
		return img
	}
}

func thumbnailKey(baseKey, size string) string {
	ext := filepath.Ext(baseKey)
	base := strings.TrimSuffix(baseKey, ext)
	return fmt.Sprintf("%s_thumb_%s.jpg", base, size)
}
