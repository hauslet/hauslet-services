package http

import (
	"encoding/json"
	authmiddleware "hauslet/internal/modules/auth/middleware"
	"hauslet/internal/modules/messaging/service"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// GenerateUploadURLRequest captures the payload required to sign an attachment upload.
type GenerateUploadURLRequest struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	AttachmentID   uuid.UUID `json:"attachment_id"`
	ContentType    string    `json:"content_type"`
	Filename       string    `json:"filename"`
	SizeBytes      int64     `json:"size_bytes"`
	ExpiresIn      int       `json:"expires_in_seconds"`
}

// GenerateUploadURLResponse returns the presigned URL and storage key.
type GenerateUploadURLResponse struct {
	URL string `json:"url"`
	Key string `json:"key"`
}

// GenerateUploadURL hands a signed PUT URL to the caller after ensuring conversation access.
func (h *HTTPHandler) GenerateUploadURL(w http.ResponseWriter, r *http.Request) {
	userID := authmiddleware.GetUserID(r)
	if userID == "" {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}
	var payload GenerateUploadURLRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.sendError(w, "invalid request payload", http.StatusBadRequest, "")
		return
	}

	if payload.ConversationID == uuid.Nil || payload.AttachmentID == uuid.Nil {
		h.sendError(w, "conversation_id and attachment_id are required", http.StatusBadRequest, "")
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		h.log.Error("invalid user ID", "user_id", userID, "error", err)
		h.sendError(w, "Invalid user ID", http.StatusInternalServerError, "")
		return
	}

	if _, err := h.messagingSvc.GetConversation(r.Context(), payload.ConversationID, userUUID); err != nil {
		h.log.Warn("failed to authorize upload url request", "error", err)
		h.handleError(w, err)
		return
	}

	req := service.AttachmentUploadRequest{
		ConversationID: payload.ConversationID,
		AttachmentID:   payload.AttachmentID,
		ContentType:    payload.ContentType,
		Filename:       payload.Filename,
		SizeBytes:      payload.SizeBytes,
	}
	if payload.ExpiresIn > 0 {
		req.ExpiresIn = time.Duration(payload.ExpiresIn) * time.Second
	}

	result, err := h.messagingSvc.GenerateUploadURL(r.Context(), req)
	if err != nil {
		h.log.Warn("failed to generate messaging upload url", "error", err)
		h.handleError(w, err)
		return
	}

	h.sendSuccess(w, GenerateUploadURLResponse{
		URL: result.URL,
		Key: result.Key,
	}, http.StatusOK)
}
