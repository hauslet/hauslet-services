package setup

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

// HandleServerLifecycle manages the lifecycle of the HTTP server.
func HandleServerLifecycle(srv *http.Server, log *slog.Logger) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	defer close(serverErr)

	go func() {
		log.Info("🌐 HTTP server listening on " + srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		log.Error("Server error", "error", err)

	case <-stop:
		log.Warn("Shutdown signal received. Shutting down server...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("HTTP server shutdown error", "error", err)
		} else {
			log.Info("HTTP server shutdown cleanly")
		}
	}

	log.Info("Graceful shutdown complete.")
}
