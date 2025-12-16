package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// --- Request DTOs ---

// MediaUploadInput represents a single media item to upload
type MediaUploadInput struct {
	Type         MediaType `json:"type" validate:"required"`
	Group        *string   `json:"group,omitempty"`
	Caption      *string   `json:"caption,omitempty"`
	MimeType     *string   `json:"mime_type,omitempty"`
	SizeBytes    int64     `json:"size_bytes" validate:"required,gt=0"`
	IsPrimary    bool      `json:"is_primary"`
	IsGroupCover bool      `json:"is_group_cover"`
	Order        int       `json:"order"`
	Filename     string    `json:"filename" validate:"required"`
	Duration     *int      `json:"duration,omitempty"`
}

// UploadMediaRequest represents the request to upload media
type UploadMediaRequest struct {
	Media []MediaUploadInput `json:"media" validate:"required,min=1,max=50"`
}

// Validate validates the upload media request
func (r *UploadMediaRequest) Validate() error {
	if len(r.Media) == 0 {
		return NewValidationError("media", "at least one media item is required")
	}

	if len(r.Media) > 50 {
		return NewValidationError("media", "cannot upload more than 50 media items at once")
	}

	for i, m := range r.Media {
		if m.Filename == "" {
			return NewValidationError(fmt.Sprintf("media[%d].filename", i), "filename is required")
		}

		if m.SizeBytes <= 0 {
			return NewValidationError(fmt.Sprintf("media[%d].size_bytes", i), "file size must be greater than 0")
		}

		if m.Type == "" {
			return NewValidationError(fmt.Sprintf("media[%d].type", i), "media type is required")
		}

		// Validate media type
		if m.Type != MediaTypeImage && m.Type != MediaTypeVideo && m.Type != MediaTypeDocument && m.Type != MediaTypeTour360 {
			return NewValidationError(fmt.Sprintf("media[%d].type", i), "invalid media type")
		}
	}

	return nil
}

// UpdateMediaRequest represents the request to update media metadata
type UpdateMediaRequest struct {
	Caption      *string `json:"caption,omitempty"`
	IsPrimary    *bool   `json:"is_primary,omitempty"`
	IsGroupCover *bool   `json:"is_group_cover,omitempty"`
	Order        *int    `json:"order,omitempty"`
}

// Validate validates the update media request
func (r *UpdateMediaRequest) Validate() error {
	// At least one field must be provided
	if r.Caption == nil && r.IsPrimary == nil && r.IsGroupCover == nil && r.Order == nil {
		return NewValidationError("", "at least one field must be provided for update")
	}

	// Validate order if provided
	if r.Order != nil && *r.Order < 0 {
		return NewValidationError("order", "order must be non-negative")
	}

	return nil
}

// MediaDeleteInput represents a single media item to delete
type MediaDeleteInput struct {
	MediaID uuid.UUID `json:"media_id" validate:"required"`
	Key     string    `json:"key" validate:"required"`
}

// DeleteMediaRequest represents the request to delete media
type DeleteMediaRequest struct {
	Media []MediaDeleteInput `json:"media" validate:"required,min=1,max=50"`
}

// Validate validates the delete media request
func (r *DeleteMediaRequest) Validate() error {
	if len(r.Media) == 0 {
		return NewValidationError("media", "at least one media item is required")
	}

	if len(r.Media) > 50 {
		return NewValidationError("media", "cannot delete more than 50 media items at once")
	}

	for i, m := range r.Media {
		if m.MediaID == uuid.Nil {
			return NewValidationError(fmt.Sprintf("media[%d].media_id", i), "media ID is required")
		}

		if m.Key == "" {
			return NewValidationError(fmt.Sprintf("media[%d].key", i), "media key is required")
		}
	}

	return nil
}

// FinalizeMediaRequest represents the request to finalize uploaded media
type FinalizeMediaRequest struct {
	MediaKeys []string `json:"media_keys" validate:"required,min=1"`
}

// Validate validates the finalize media request
func (r *FinalizeMediaRequest) Validate() error {
	if len(r.MediaKeys) == 0 {
		return NewValidationError("media_keys", "at least one media key is required")
	}

	for i, key := range r.MediaKeys {
		if key == "" {
			return NewValidationError(fmt.Sprintf("media_keys[%d]", i), "media key cannot be empty")
		}
	}

	return nil
}

// --- Response DTOs ---

// MediaUploadResult represents the result of uploading a single media item
type MediaUploadResult struct {
	ID       uuid.UUID `json:"id"`
	Filename string    `json:"filename"`
	URL      string    `json:"url"`
	Key      string    `json:"key"`
}

// UploadMediaResponse represents the response after uploading media
type UploadMediaResponse struct {
	Media []MediaUploadResult `json:"media"`
	Total int                 `json:"total"`
}

// MediaResponse represents a media item in responses
type MediaResponse struct {
	ID           uuid.UUID    `json:"id"`
	ListingID    uuid.UUID    `json:"listing_id"`
	URL          string       `json:"url"`
	Type         MediaType    `json:"type"`
	Thumbnails   ThumbnailMap `json:"thumbnails,omitempty"`
	Group        *string      `json:"group,omitempty"`
	Caption      *string      `json:"caption,omitempty"`
	MimeType     *string      `json:"mime_type,omitempty"`
	SizeBytes    int64        `json:"size_bytes"`
	IsPrimary    bool         `json:"is_primary"`
	IsGroupCover bool         `json:"is_group_cover"`
	Order        int          `json:"order"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// ListMediaResponse represents the response when listing media
type ListMediaResponse struct {
	Media []MediaResponse `json:"media"`
	Total int             `json:"total"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Message string `json:"message"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Field   string `json:"field,omitempty"`
}

// --- Helper Functions ---

// ToMediaResponse converts a ListingMedia domain object to MediaResponse DTO
func ToMediaResponse(m *ListingMedia) MediaResponse {
	return MediaResponse{
		ID:           m.ID,
		ListingID:    m.ListingID,
		URL:          m.URL,
		Type:         m.Type,
		Thumbnails:   m.Thumbnails,
		Group:        m.Group,
		Caption:      m.Caption,
		MimeType:     m.MimeType,
		SizeBytes:    m.SizeBytes,
		IsPrimary:    m.IsPrimary,
		IsGroupCover: m.IsGroupCover,
		Order:        m.Order,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

// ToMediaResponseList converts a list of ListingMedia to MediaResponse DTOs
func ToMediaResponseList(media []ListingMedia) []MediaResponse {
	result := make([]MediaResponse, len(media))
	for i, m := range media {
		result[i] = ToMediaResponse(&m)
	}
	return result
}

// ToListingMediaInput converts MediaUploadInput to ListingMediaInput
func (m MediaUploadInput) ToListingMediaInput() ListingMediaInput {
	return ListingMediaInput(m)
}

// ToListingMediaInputList converts a list of MediaUploadInput to ListingMediaInput
func ToListingMediaInputList(media []MediaUploadInput) []ListingMediaInput {
	result := make([]ListingMediaInput, len(media))
	for i, m := range media {
		result[i] = m.ToListingMediaInput()
	}
	return result
}

// ToListingMediaDeleteInput converts MediaDeleteInput to ListingMediaDeleteInput
// ToListingMediaDeleteInput converts MediaDeleteInput to ListingMediaDeleteInput
func (m MediaDeleteInput) ToListingMediaDeleteInput() ListingMediaDeleteInput {
	return ListingMediaDeleteInput(m)
}

// ToListingMediaDeleteInputList converts a list of MediaDeleteInput to ListingMediaDeleteInput
func ToListingMediaDeleteInputList(media []MediaDeleteInput) []ListingMediaDeleteInput {
	result := make([]ListingMediaDeleteInput, len(media))
	for i, m := range media {
		result[i] = m.ToListingMediaDeleteInput()
	}
	return result
}

// ToMediaUploadResult converts ListingMediaResult to MediaUploadResult
func (r ListingMediaResult) ToMediaUploadResult() MediaUploadResult {
	return MediaUploadResult(r)
}

// ToMediaUploadResultList converts a list of ListingMediaResult to MediaUploadResult
func ToMediaUploadResultList(results []ListingMediaResult) []MediaUploadResult {
	result := make([]MediaUploadResult, len(results))
	for i, r := range results {
		result[i] = r.ToMediaUploadResult()
	}
	return result
}

// ValidationError represents a field-specific validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}
