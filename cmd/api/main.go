package main

import (
	"context"
	"hauslet/cmd/api/server"
	"hauslet/config"
	"hauslet/internal/auth/repository/schema"
	"hauslet/pkg/database"
	"hauslet/pkg/logger"
	"hauslet/platform/redis"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-pkgz/lgr"
)

const (
	ShutdownTimeout = 60 * time.Second
	InitTimeout     = 30 * time.Second
)

func main() {
	cfg := config.Load()
	log := logger.New()
	if cfg.App.Env != "development" {
		log = logger.NewProduction()
	}

	log.Logf("INFO 🚀 Starting Hauslet server in %s mode", cfg.App.Env)

	// Context for initialization steps
	initCtx, cancelInit := context.WithTimeout(context.Background(), InitTimeout)
	defer cancelInit()

	// Connect to the database
	db, err := database.NewPostgresWithContext(initCtx, &cfg.Storage.DB, cfg.App.Env)
	if err != nil {
		log.Logf("ERROR failed to connect to database: %v", err)
		return
	}
	defer database.Close(db)

	log.Logf("INFO ✅ Database connected successfully")

	// Run migrations
	if err := database.RunMigrations(db, &schema.User{}, &schema.UserIdentity{}); err != nil {
		log.Logf("ERROR failed to run migrations: %v", err)
		return
	}
	log.Logf("INFO ✅ Database migrations completed")

	err = redis.InitRedis(&cfg.Storage.Redis, initCtx)
	if err != nil {
		log.Logf("ERROR failed to initialize Redis: %v", err)
		return
	}
	defer redis.CloseRedis()

	redisClient, err := redis.GetRedis()
	if err != nil {
		log.Logf("ERROR failed to get Redis client: %v", err)
		return
	}
	_, err = redisClient.Ping(initCtx).Result()
	if err != nil {
		log.Logf("ERROR failed to connect to Redis: %v", err)
		return
	}
	log.Logf("INFO ✅ Redis connected successfully")

	srv := server.NewHTTPServer(initCtx, db, &redisClient, log, cfg)
	handleServerLifecycle(srv, log)
}

// handleServerLifecycle manages the lifecycle of the HTTP server
func handleServerLifecycle(srv *http.Server, log *lgr.Logger) {
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
