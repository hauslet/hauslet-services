package middleware

import (
	"fmt"
	"hauslet/internal/auth/service"
	"net/http"
	"strings"
)

// CaptureAuthMetadata wraps go-pkgz/auth login handlers to capture IP and User-Agent
// This middleware should be applied BEFORE the auth handlers
func CaptureAuthMetadata(authService service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only capture for login endpoints
			if isLoginEndpoint(r.URL.Path) {
				// Extract IP
				ip := extractIP(r)

				// Extract User-Agent
				userAgent := r.Header.Get("User-Agent")

				// Extract email from request
				// For password login: user parameter contains email
				// For OAuth: we can't capture here (no email yet)
				email := r.FormValue("user")
				if email == "" {
					email = r.PostFormValue("user")
				}

				// Store metadata if email is available
				if email != "" {
					authService.StoreRequestMetadata(email, ip, userAgent)
				}
			}

			// Continue to auth handler
			next.ServeHTTP(w, r)
		})
	}
}

// isLoginEndpoint checks if the path is a login endpoint
func isLoginEndpoint(path string) bool {
	loginPaths := []string{
		"/auth/direct/login",
		"/auth/password/login",
		"/auth/login",
	}

	for _, loginPath := range loginPaths {
		if strings.HasSuffix(path, loginPath) {
			return true
		}
	}

	return false
}

var logRequestHeaders = func(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("== Incoming Headers ==")
		for name, values := range r.Header {
			for _, v := range values {
				fmt.Printf("%s: %s\n", name, v)
			}
		}
		fmt.Println("======================")
		next.ServeHTTP(w, r)
	})
}
