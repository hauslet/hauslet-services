package main

import (
	"context"

	"hauslet/cmd/api/server"
	"hauslet/cmd/api/setup"
	"hauslet/config"
)

func main() {
	cfg := config.Load()
	log := setup.SetupLogger(cfg.App.Env)
	log.Logf("INFO 🚀 Starting Hauslet server in %s mode", cfg.App.Env)

	initCtx, cancelInit := context.WithTimeout(context.Background(), setup.InitTimeout)
	defer cancelInit()

	infra, err := setup.InitInfrastructure(initCtx, cfg, log)
	if err != nil {
		log.Logf("ERROR failed to initialize infrastructure: %v", err)
		return
	}
	defer infra.CloseDB()
	defer infra.CloseCache()

	if err := setup.RunMigrations(infra.DB, log); err != nil {
		log.Logf("ERROR failed to run migrations: %v", err)
		return
	}

	queueClient, err := setup.InitQueue(initCtx, cfg, log)
	if err != nil {
		log.Logf("WARN ⚠️ failed to initialize Cloud Tasks queue, direct send will be used: %v", err)
	}
	infra.Queue = queueClient
	defer infra.CloseQueue()
	if queueClient != nil {
		log.Logf("INFO ✅ Cloud Tasks queue initialized")
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
