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
			if cfg.Env == "development" {
				return []string{"http://localhost:*", "http://127.0.0.1:*"}
			}
			return []string{cfg.Client}
		}(),
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: cfg.Env != "development",
		MaxAge:           300,
	})
}
