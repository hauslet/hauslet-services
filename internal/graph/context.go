package graph

import (
	"context"
	"net/http"

	"github.com/go-pkgz/auth/token"
)

type viewerContextKey struct{}

// Viewer represents the authenticated user making the GraphQL request.
type Viewer struct {
	UserID string
	Role   string
}

// WithViewerContext captures token.User (if present) and exposes a lightweight viewer
// in the request context for downstream resolvers. Safe for public routes as it
// simply skips when no auth token is provided.
func WithViewerContext(next http.Handler) http.Handler {
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
			viewer := &Viewer{
				UserID: userID,
				Role:   role,
			}
			ctx := context.WithValue(r.Context(), viewerContextKey{}, viewer)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

// viewerFromContext extracts the viewer info if present.
func viewerFromContext(ctx context.Context) *Viewer {
	viewer, _ := ctx.Value(viewerContextKey{}).(*Viewer)
	return viewer
}
