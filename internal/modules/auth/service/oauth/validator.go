package oauth

import (
	"context"
	"strings"

	"hauslet/internal/modules/auth/repository"

	"github.com/go-pkgz/auth/token"
	"github.com/go-pkgz/lgr"
)

func NewValidator(repo repository.AuthRepository, log *lgr.Logger) token.ValidatorFunc {
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
			log.Logf("DEBUG Validator using fallback user ID from token: %s", userID)
		}

		sessionID := claims.User.StrAttr("sid")
		if sessionID == "" {
			log.Logf("WARN Token rejected - missing session ID for user %s", userID)
			return false
		}

		session, err := repo.GetSessionByID(ctx, sessionID)
		if err != nil {
			log.Logf("ERROR Session validation failed (Redis unavailable): %v - rejecting token", err)
			return false
		}
		if session == nil {
			log.Logf("INFO Token rejected - session %s revoked for user %s", sessionID, userID)
			return false
		}

		user, err := repo.GetUserByID(ctx, userID)
		if err != nil || user == nil {
			return false
		}

		return user.IsActive
	}
}
