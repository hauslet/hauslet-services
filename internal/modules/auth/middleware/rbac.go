package middleware

import (
	"net/http"
	"slices"

	"github.com/go-pkgz/auth/v2/token"
)

// RBAC provides role-based access control
// Usage example in your HTTP handlers:
//
//	router.With(authMiddleware.Auth, RBAC("admin")).Get("/admin/users", listUsersHandler)
//	router.With(authMiddleware.Auth, RBAC("user", "admin")).Get("/profile", profileHandler)
//
// The role is read from the JWT token's user attributes set during ClaimsUpdater
// NOTE: Root users have access to ALL routes, regardless of specified roles
func RBAC(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := token.GetUserInfo(r)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			userRole := user.StrAttr("role")
			if userRole == "" {
				http.Error(w, "Forbidden: no role assigned", http.StatusForbidden)
				return
			}

			// ROOT PRIVILEGE: Root users have access to everything
			if userRole == "root" {
				next.ServeHTTP(w, r)
				return
			}

			// Check if user role matches any of the allowed roles
			if slices.Contains(allowedRoles, userRole) {
				next.ServeHTTP(w, r)
				return
			}

			http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
		})
	}
}

// GetUserRole extracts the role from token claims
// Use this in handlers to check user role manually if needed
func GetUserRole(r *http.Request) string {
	user, err := token.GetUserInfo(r)
	if err != nil {
		return ""
	}
	return user.StrAttr("role")
}

// GetUserID extracts the user ID from token claims
func GetUserID(r *http.Request) string {
	user, err := token.GetUserInfo(r)
	if err != nil {
		return ""
	}
	return user.ID
}
