package viewer

import (
	"context"
	"net/http"

	authmiddleware "hauslet/internal/modules/auth/middleware"
	"hauslet/internal/platform/authz"

	"github.com/go-pkgz/auth/token"
)

type contextKey struct{}

// Viewer represents the authenticated user making the GraphQL request.
type Viewer struct {
	UserID    string
	Role      string
	SessionID string
	UserAgent string
	IPAddress string
}

// WithContext captures token.User (if present) and exposes a lightweight viewer
// in the request context for downstream resolvers. Safe for public routes as it
// simply skips when no auth token is provided.
func WithContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract request metadata (IP, User-Agent) from auth middleware
		requestMeta := authmiddleware.GetRequestMeta(r.Context())

		user, err := token.GetUserInfo(r)
		if err == nil {
			userID := user.StrAttr("uid")
			if userID == "" {
				userID = user.ID
			}

			role := user.Role
			if role == "" {
				role = user.StrAttr("role")
			}

			// Extract session ID from token attrs
			sessionID := user.StrAttr("sid")

			v := &Viewer{
				UserID:    userID,
				Role:      role,
				SessionID: sessionID,
				UserAgent: requestMeta.UserAgent,
				IPAddress: requestMeta.IP,
			}
			ctx := context.WithValue(r.Context(), contextKey{}, v)
			ctx = authz.ContextWithActor(ctx, &authz.Actor{
				UserID: userID,
				Role:   role,
			})
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

// FromContext extracts the viewer info if present.
func FromContext(ctx context.Context) *Viewer {
	viewer, _ := ctx.Value(contextKey{}).(*Viewer)
	return viewer
}

// ContextWith sets a viewer on a context (useful outside graph adapter).
func ContextWith(ctx context.Context, v *Viewer) context.Context {
	return context.WithValue(ctx, contextKey{}, v)
}
