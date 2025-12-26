package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-pkgz/auth/token"
	"github.com/google/uuid"
)

type blockRequest struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Reason    string `json:"reason"`
	OwnerStay *bool  `json:"owner_stay,omitempty"`
}

type blockResponse struct {
	EventID   uuid.UUID `json:"event_id"`
	ListingID uuid.UUID `json:"listing_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Reason    string    `json:"reason"`
}

func (h *HTTPHandler) createBlock(w http.ResponseWriter, r *http.Request) {
	listingID, err := uuid.Parse(chi.URLParam(r, "listingId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid listing id")
		return
	}

	ownerID, err := extractUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req blockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	start, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid start_time; must be RFC3339")
		return
	}

	end, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid end_time; must be RFC3339")
		return
	}

	if !end.After(start) {
		writeError(w, http.StatusBadRequest, "end_time must be after start_time")
		return
	}

	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "manual_block"
	}

	ownerStay := true
	if req.OwnerStay != nil {
		ownerStay = *req.OwnerStay
	}

	event, err := h.calendarService.BlockDates(h.ctx, listingID, start, end, reason, ownerID, ownerStay)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp := blockResponse{
		EventID:   event.ID,
		ListingID: listingID,
		StartTime: event.StartTime,
		EndTime:   event.EndTime,
		Reason:    reason,
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *HTTPHandler) deleteBlock(w http.ResponseWriter, r *http.Request) {
	blockID, err := uuid.Parse(chi.URLParam(r, "blockId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid block id")
		return
	}

	ownerID, err := extractUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.calendarService.UnblockDates(h.ctx, blockID, ownerID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func extractUserID(r *http.Request) (uuid.UUID, error) {
	userInfo, err := token.GetUserInfo(r)
	if err != nil {
		return uuid.Nil, err
	}

	userID := userInfo.StrAttr("uid")
	if userID == "" {
		userID = userInfo.ID
		if parts := strings.SplitN(userID, "_", 2); len(parts) == 2 {
			userID = parts[1]
		}
	}

	return uuid.Parse(userID)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error":   http.StatusText(status),
		"message": message,
	})
}
