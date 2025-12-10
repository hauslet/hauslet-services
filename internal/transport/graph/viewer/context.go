package viewer

import (
	"context"
	"net/http"

	"github.com/go-pkgz/auth/token"
)

type contextKey struct{}

// Viewer represents the authenticated user making the GraphQL request.
type Viewer struct {
	UserID string
	Role   string
}

// WithContext captures token.User (if present) and exposes a lightweight viewer
// in the request context for downstream resolvers. Safe for public routes as it
// simply skips when no auth token is provided.
func WithContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			v := &Viewer{
				UserID: userID,
				Role:   role,
			}
			ctx := context.WithValue(r.Context(), contextKey{}, v)
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
