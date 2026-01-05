package oauth

import (
	"context"
	"time"

	"hauslet/internal/modules/auth/domain"
	"hauslet/internal/modules/auth/repository/schema"

	"github.com/google/uuid"
)

func createSession(deps Dependencies, user *schema.User, provider string, metadataKey string) string {
	ctx := context.Background()

	_ = deps.Repository.UpdateUserLastLogin(ctx, user.ID.String())

	var ip, userAgent string
	if deps.MetadataFetcher != nil {
		if metadata := deps.MetadataFetcher(metadataKey); metadata != nil {
			ip = metadata.IP
			userAgent = metadata.UserAgent
		}
	}

	sessionID := uuid.New().String()
	session := &domain.Session{
		ID:        sessionID,
		UserID:    user.ID,
		Provider:  provider,
		ExpiresAt: time.Now().Add(deps.Config.SessionDuration),
		CreatedAt: time.Now(),
		IP:        ip,
		UserAgent: userAgent,
	}

	if err := deps.Repository.CreateSession(ctx, session); err != nil {
		deps.Log.Error("Auth: Error creating session", "error", err)
	} else {
		deps.Log.Info(" Auth: Created session", "sessionID", sessionID, "userID", user.ID)
	}

	return sessionID
}
