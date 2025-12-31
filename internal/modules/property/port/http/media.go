package http

import (
	"encoding/json"
	"hauslet/internal/modules/property/domain"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// UploadListingMedia handles the upload of media for a listing
// @Summary Upload listing media
// @Description Generate presigned upload URLs for listing media
// @Tags listings
// @Accept json
// @Produce json
// @Param id path string true "Listing ID"
// @Param request body domain.UploadMediaRequest true "Media upload details"
// @Success 201 {object} domain.UploadMediaResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/listings/{id}/media [post]
// @Security BearerAuth
func (h *HTTPHandler) UploadListingMedia(w http.ResponseWriter, r *http.Request) {
	// Extract listing ID from URL
	listingIDStr := chi.URLParam(r, "id")
	listingID, err := uuid.Parse(listingIDStr)
	if err != nil {
		h.log.Error("Invalid listing ID: %v", err)
		h.sendError(w, "Invalid listing ID", http.StatusBadRequest, "id")
		return
	}

	// Extract user ID from JWT token
	userID, err := h.extractUserID(r)
	if err != nil {
		h.log.Error("Failed to extract user ID: %v", err)
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	// Verify listing ownership
	if err := h.verifyListingOwnership(r.Context(), listingID, userID); err != nil {
		switch err {
		case domain.ErrListingNotFound:
			h.sendError(w, "Listing not found", http.StatusNotFound, "")
		case domain.ErrForbidden:
			h.sendError(w, "You do not have permission to upload media for this listing", http.StatusForbidden, "")
		default:
			h.sendError(w, "Failed to verify listing ownership", http.StatusInternalServerError, "")
		}
		return
	}

	// Decode request body
	var req domain.UploadMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("Failed to decode upload request: %v", err)
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.log.Warn("Upload validation failed: %v", err)
		if valErr, ok := err.(*domain.ValidationError); ok {
			h.sendError(w, valErr.Message, http.StatusBadRequest, valErr.Field)
			return
		}
		h.sendError(w, err.Error(), http.StatusBadRequest, "")
		return
	}

	// Convert DTOs to domain input
	mediaInput := domain.ToListingMediaInputList(req.Media)

	// Call service to upload media
	results, err := h.propertyService.UploadListingMedia(r.Context(), listingID, mediaInput)
	if err != nil {
		h.log.Error("Failed to upload media for listing %s: %v", listingID, err)
		h.sendError(w, "Failed to generate upload URLs", http.StatusInternalServerError, "")
		return
	}

	h.log.Info(" Generated upload URLs for %d media items for listing %s", len(results), listingID)

	// Send response
	response := domain.UploadMediaResponse{
		Media: domain.ToMediaUploadResultList(results),
		Total: len(results),
	}

	h.sendSuccess(w, response, http.StatusCreated)
}

// UpdateListingMedia handles updating media metadata for a listing
// @Summary Update listing media
// @Description Update metadata for a specific media item
// @Tags listings
// @Accept json
// @Produce json
// @Param id path string true "Listing ID"
// @Param mediaId path string true "Media ID"
// @Param request body domain.UpdateMediaRequest true "Media update details"
// @Success 200 {object} domain.SuccessResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/listings/{id}/media/{mediaId} [patch]
// @Security BearerAuth
func (h *HTTPHandler) UpdateListingMedia(w http.ResponseWriter, r *http.Request) {
	// Extract listing ID from URL
	listingIDStr := chi.URLParam(r, "id")
	listingID, err := uuid.Parse(listingIDStr)
	if err != nil {
		h.log.Error("Invalid listing ID: %v", err)
		h.sendError(w, "Invalid listing ID", http.StatusBadRequest, "id")
		return
	}

	// Extract media ID from URL
	mediaIDStr := chi.URLParam(r, "mediaId")
	mediaID, err := uuid.Parse(mediaIDStr)
	if err != nil {
		h.log.Error("Invalid media ID: %v", err)
		h.sendError(w, "Invalid media ID", http.StatusBadRequest, "mediaId")
		return
	}

	// Extract user ID from JWT token
	userID, err := h.extractUserID(r)
	if err != nil {
		h.log.Error("Failed to extract user ID: %v", err)
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	// Verify listing ownership
	if err := h.verifyListingOwnership(r.Context(), listingID, userID); err != nil {
		switch err {
		case domain.ErrListingNotFound:
			h.sendError(w, "Listing not found", http.StatusNotFound, "")
		case domain.ErrForbidden:
			h.sendError(w, "You do not have permission to update media for this listing", http.StatusForbidden, "")
		default:
			h.sendError(w, "Failed to verify listing ownership", http.StatusInternalServerError, "")
		}
		return
	}

	// Decode request body
	var req domain.UpdateMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("Failed to decode update request: %v", err)
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.log.Warn("Update validation failed: %v", err)
		if valErr, ok := err.(*domain.ValidationError); ok {
			h.sendError(w, valErr.Message, http.StatusBadRequest, valErr.Field)
			return
		}
		h.sendError(w, err.Error(), http.StatusBadRequest, "")
		return
	}

	// Convert to domain update input
	updates := domain.ListingMediaUpdateInput(req)

	// Call service to update media
	if err := h.propertyService.UpdateListingMedia(r.Context(), listingID, mediaID, updates); err != nil {
		h.log.Error("Failed to update media %s for listing %s: %v", mediaID, listingID, err)

		if err == domain.ErrMediaNotFound {
			h.sendError(w, "Media not found", http.StatusNotFound, "")
		} else {
			h.sendError(w, "Failed to update media", http.StatusInternalServerError, "")
		}
		return
	}

	h.log.Info(" Updated media %s for listing %s", mediaID, listingID)

	// Send success response
	response := domain.SuccessResponse{
		Message: "Media updated successfully",
	}

	h.sendSuccess(w, response, http.StatusOK)
}

// DeleteListingMedia handles deleting media for a listing
// @Summary Delete listing media
// @Description Delete one or more media items from a listing
// @Tags listings
// @Accept json
// @Produce json
// @Param id path string true "Listing ID"
// @Param request body domain.DeleteMediaRequest true "Media delete details"
// @Success 200 {object} domain.SuccessResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/listings/{id}/media [delete]
// @Security BearerAuth
func (h *HTTPHandler) DeleteListingMedia(w http.ResponseWriter, r *http.Request) {
	// Extract listing ID from URL
	listingIDStr := chi.URLParam(r, "id")
	listingID, err := uuid.Parse(listingIDStr)
	if err != nil {
		h.log.Error("Invalid listing ID: %v", err)
		h.sendError(w, "Invalid listing ID", http.StatusBadRequest, "id")
		return
	}

	// Extract user ID from JWT token
	userID, err := h.extractUserID(r)
	if err != nil {
		h.log.Error("Failed to extract user ID: %v", err)
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	// Verify listing ownership
	if err := h.verifyListingOwnership(r.Context(), listingID, userID); err != nil {
		switch err {
		case domain.ErrListingNotFound:
			h.sendError(w, "Listing not found", http.StatusNotFound, "")
		case domain.ErrForbidden:
			h.sendError(w, "You do not have permission to delete media for this listing", http.StatusForbidden, "")
		default:
			h.sendError(w, "Failed to verify listing ownership", http.StatusInternalServerError, "")
		}
		return
	}

	// Decode request body
	var req domain.DeleteMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("Failed to decode delete request: %v", err)
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.log.Warn("Delete validation failed: %v", err)
		if valErr, ok := err.(*domain.ValidationError); ok {
			h.sendError(w, valErr.Message, http.StatusBadRequest, valErr.Field)
			return
		}
		h.sendError(w, err.Error(), http.StatusBadRequest, "")
		return
	}

	// Convert to domain delete input
	mediaInput := domain.ToListingMediaDeleteInputList(req.Media)

	// Call service to delete media
	if err := h.propertyService.DeleteListingMedia(r.Context(), listingID, mediaInput); err != nil {
		h.log.Error("Failed to delete media for listing %s: %v", listingID, err)

		if err == domain.ErrMediaNotFound {
			h.sendError(w, "One or more media items not found", http.StatusNotFound, "")
		} else {
			h.sendError(w, "Failed to delete media", http.StatusInternalServerError, "")
		}
		return
	}

	h.log.Info(" Deleted %d media items for listing %s", len(req.Media), listingID)

	// Send success response
	response := domain.SuccessResponse{
		Message: "Media deleted successfully",
	}

	h.sendSuccess(w, response, http.StatusOK)
}

// FinalizeListingMedia handles finalizing uploaded media
// @Summary Finalize listing media
// @Description Mark media as uploaded and trigger thumbnail generation
// @Tags listings
// @Accept json
// @Produce json
// @Param id path string true "Listing ID"
// @Param request body domain.FinalizeMediaRequest true "Media finalization details"
// @Success 200 {object} domain.SuccessResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/listings/{id}/media/finalize [post]
// @Security BearerAuth
func (h *HTTPHandler) FinalizeListingMedia(w http.ResponseWriter, r *http.Request) {
	// Extract listing ID from URL
	listingIDStr := chi.URLParam(r, "id")
	listingID, err := uuid.Parse(listingIDStr)
	if err != nil {
		h.log.Error("Invalid listing ID: %v", err)
		h.sendError(w, "Invalid listing ID", http.StatusBadRequest, "id")
		return
	}

	// Extract user ID from JWT token
	userID, err := h.extractUserID(r)
	if err != nil {
		h.log.Error("Failed to extract user ID: %v", err)
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	// Verify listing ownership
	if err := h.verifyListingOwnership(r.Context(), listingID, userID); err != nil {
		switch err {
		case domain.ErrListingNotFound:
			h.sendError(w, "Listing not found", http.StatusNotFound, "")
		case domain.ErrForbidden:
			h.sendError(w, "You do not have permission to finalize media for this listing", http.StatusForbidden, "")
		default:
			h.sendError(w, "Failed to verify listing ownership", http.StatusInternalServerError, "")
		}
		return
	}

	// Decode request body
	var req domain.FinalizeMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("Failed to decode finalize request: %v", err)
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.log.Warn("Finalize validation failed: %v", err)
		if valErr, ok := err.(*domain.ValidationError); ok {
			h.sendError(w, valErr.Message, http.StatusBadRequest, valErr.Field)
			return
		}
		h.sendError(w, err.Error(), http.StatusBadRequest, "")
		return
	}

	// Build domain object
	finalizeData := domain.FinalizedListingMedia{
		ListingID: listingID,
		MediaKeys: req.MediaKeys,
	}

	// Call service to finalize media
	if err := h.propertyService.FinalizeListingMedia(r.Context(), finalizeData); err != nil {
		h.log.Error("Failed to finalize media for listing %s: %v", listingID, err)

		if err == domain.ErrMediaNotFound {
			h.sendError(w, "One or more media items not found", http.StatusNotFound, "")
		} else {
			h.sendError(w, "Failed to finalize media", http.StatusInternalServerError, "")
		}
		return
	}

	h.log.Info(" Finalized %d media items for listing %s", len(req.MediaKeys), listingID)

	// Send success response
	response := domain.SuccessResponse{
		Message: "Media finalized successfully. Thumbnail generation in progress.",
	}

	h.sendSuccess(w, response, http.StatusOK)
}
