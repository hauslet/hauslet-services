package setup

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-pkgz/lgr"
)

// HandleServerLifecycle manages the lifecycle of the HTTP server.
func HandleServerLifecycle(srv *http.Server, log *lgr.Logger) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() {
		log.Logf("INFO 🌐 HTTP server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		log.Logf("ERROR Server error: %v", err)

	case <-stop:
		log.Logf("WARN Shutdown signal received. Shutting down server...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Logf("ERROR HTTP server shutdown error: %v", err)
		} else {
			log.Logf("INFO HTTP server shutdown cleanly")
		}
	}

	log.Logf("INFO Graceful shutdown complete.")
}
