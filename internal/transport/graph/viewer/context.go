package viewer

import (
	"context"
	"fmt"
	"net/http"

	authmiddleware "hauslet/internal/modules/auth/middleware"
	"hauslet/internal/platform/authz"

	"github.com/go-pkgz/auth/v2/token"
	"github.com/google/uuid"
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

// Authorization errors
var (
	ErrUnauthorized      = fmt.Errorf("unauthorized: authentication required")
	ErrAdminRequired     = fmt.Errorf("forbidden: admin or root access required")
	ErrSupportRequired   = fmt.Errorf("forbidden: support, admin, or root access required")
	ErrOwnershipRequired = fmt.Errorf("forbidden: resource ownership or admin access required")
)

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

// GetUserIDFromContext extracts user ID from context
func GetUserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	v := FromContext(ctx)
	if v == nil || v.UserID == "" {
		return uuid.Nil, ErrUnauthorized
	}

	userID, err := uuid.Parse(v.UserID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID in context")
	}

	return userID, nil
}

// GetOptionalUserIDFromContext extracts user ID from context without error
func GetOptionalUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	v := FromContext(ctx)
	if v == nil || v.UserID == "" {
		return uuid.Nil, false
	}

	userID, err := uuid.Parse(v.UserID)
	if err != nil {
		return uuid.Nil, false
	}

	return userID, true
}

// RequireOwnership ensures user owns the resource
func RequireOwnership(ctx context.Context, resourceOwnerID uuid.UUID) error {
	v := FromContext(ctx)
	if v == nil || v.UserID == "" {
		return ErrUnauthorized
	}

	userID, err := uuid.Parse(v.UserID)
	if err != nil {
		return fmt.Errorf("invalid user ID")
	}

	if userID != resourceOwnerID {
		return ErrOwnershipRequired
	}

	return nil
}

// RequireSupport ensures user has support, admin, or root role
func RequireSupport(ctx context.Context) error {
	v := FromContext(ctx)
	if v == nil || v.UserID == "" {
		return ErrUnauthorized
	}

	if !isSupportRole(v.Role) {
		return ErrSupportRequired
	}

	return nil
}

// isSupportRole checks if role is support, admin, or root
func isSupportRole(role string) bool {
	return role == "support" || role == "admin" || role == "root"
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
