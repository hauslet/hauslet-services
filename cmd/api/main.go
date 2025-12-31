package main

import (
	"context"
	"hauslet/cmd/api/server"
	"hauslet/cmd/api/setup"
	"hauslet/config"
	"hauslet/internal/platform/logger"
)

func main() {
	cfg := config.Load()
	log := logger.NewLogger()
	log.Info("🚀 Starting Hauslet server", "mode", cfg.App.Env)

	initCtx, cancelInit := context.WithTimeout(context.Background(), setup.InitTimeout)
	defer cancelInit()

	infra, err := setup.InitInfrastructure(initCtx, cfg, log)
	if err != nil {
		log.Error("failed to initialize infrastructure: %v", err)
		return
	}
	defer infra.CloseDB()
	defer infra.CloseCache()

	if err := setup.RunMigrations(infra.DB, log); err != nil {
		log.Error("failed to run migrations: %v", err)
		return
	}

	queueClient, err := setup.InitQueue(initCtx, cfg, log)
	if err != nil {
		log.Warn("⚠️ failed to initialize Cloud Tasks queue, direct send will be used", "error", err)
	}
	infra.Queue = queueClient
	defer infra.CloseQueue()
	if queueClient != nil {
		log.Info("✅ Cloud Tasks queue initialized")
	}

	redisClient := infra.Cache
	srv := server.NewHTTPServer(
		initCtx,
		infra.DB,
		&redisClient,
		log,
		cfg,
		infra.Email,
		infra.Queue,
		infra.Storage,
	)
	setup.HandleServerLifecycle(srv, log)
}
