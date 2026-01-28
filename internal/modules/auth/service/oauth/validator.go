package oauth

import (
	"context"
	"log/slog"
	"strings"

	"hauslet/internal/modules/auth/repository"

	"github.com/go-pkgz/auth/v2/token"
)

func NewValidator(repo repository.AuthRepository, log *slog.Logger) token.ValidatorFunc {
	return func(_ string, claims token.Claims) bool {
		ctx := context.Background()
		if claims.User == nil || claims.User.ID == "" {
			return false
		}

		// Prefer canonical user ID stored in uid; fallback to ID for older tokens
		userID := claims.User.StrAttr("uid")
		if userID == "" {
			userID = claims.User.ID
			// If the ID is provider-prefixed, strip the prefix as a last resort
			if idx := strings.Index(userID, "_"); idx > 0 {
				userID = userID[idx+1:]
			}
			log.Debug("Validator using fallback user ID from token", "userID", userID)
		}

		// Allow 2FA pending state to pass validation (restricted access checked by handlers)
		if claims.User.StrAttr("login_state") == "2fa_pending" {
			return true
		}

		sessionID := claims.User.StrAttr("sid")
		if sessionID == "" {
			log.Warn("Token rejected - missing session ID for user", "userID", userID)
			return false
		}

		session, err := repo.GetSessionByID(ctx, sessionID)
		if err != nil {
			log.Error("Session validation failed (Redis unavailable) - rejecting token", "error", err)
			return false
		}
		if session == nil {
			log.Info("Token rejected session revoked", "session_id", sessionID, "user_id", userID)
			return false
		}

		user, err := repo.GetUserByID(ctx, userID)
		if err != nil || user == nil {
			return false
		}

		return user.IsActive
	}
}
