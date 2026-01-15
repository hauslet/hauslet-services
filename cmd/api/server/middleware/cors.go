package middleware

import (
	"hauslet/config"

	"net/http"

	"github.com/go-chi/cors"
)

// CORSMiddleware returns a CORS middleware configured based on the application environment.
func CORSMiddleware(cfg *config.AppConfig) func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins: func() []string {
			if cfg.Env == "production" {
				return []string{cfg.Client}
			}
			return []string{
				"http://localhost:8080",
				"https://dev-client.hauslet.com",
				"https://hauslet-test-client.onrender.com",
			}
		}(),
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-XSRF-TOKEN"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true, // Always allow credentials for cookie-based auth
		MaxAge:           300,
	})
}
