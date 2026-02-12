package middleware

import (
	"hauslet/config"

	"net/http"

	"github.com/go-chi/cors"
)

// CORSMiddleware returns a CORS middleware configured based on the application environment.
func CORSMiddleware(cfg *config.AppConfig) func(http.Handler) http.Handler {
	opts := cors.Options{
		AllowedOrigins: func() []string {
			allowed := []string{cfg.Client}
			if cfg.Env != "production" {
				allowed = append(allowed,
					"http://localhost:3000",
					"https://dev-client.hauslet.com",
					"https://hauslet-test-client.onrender.com",
					"https://hauslet-client-admin.vercel.app",
				)
			}
			return allowed
		}(),
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-XSRF-TOKEN"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true, // Always allow credentials for cookie-based auth
		MaxAge:           300,
	}

	// In development, accept any origin so frontend devs on other devices
	// can connect via http://<your-local-ip>:3000
	if cfg.Env == "development" {
		opts.AllowOriginFunc = func(r *http.Request, origin string) bool {
			return true
		}
	}

	return cors.Handler(opts)
}
