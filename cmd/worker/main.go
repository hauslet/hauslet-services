package main

import (
	"context"

	"hauslet/cmd/worker/setup"
	"hauslet/config"
	"hauslet/internal/transport/worker"
)

func main() {
	// 1) Config + Logger
	cfg := config.Load()
	log := setup.SetupLogger(cfg.App.Env)
	log.Logf("INFO 🔧 Starting worker in %s mode", cfg.App.Env)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2) Core infrastructure (DB, storage, AI, email, cache)
	infra, err := setup.InitInfrastructure(ctx, cfg, log)
	if err != nil {
		log.Logf("CRITICAL failed to initialize infrastructure: %v", err)
		return
	}
	defer infra.CloseDB()
	defer infra.CloseCache()

	// 3) Queue client
	queueSubjects := setup.GetActiveSubjects(cfg)
	qClient, err := setup.InitQueue(ctx, cfg, queueSubjects)
	if err != nil {
		log.Logf("CRITICAL failed to initialize queue: %v", err)
		return
	}
	infra.Queue = qClient
	defer infra.CloseQueue()

	// 4) Handlers
	registry := setup.RegisterHandlers(infra, cfg, log)
	log.Logf("INFO ✅ Registered %d job handlers", registry.HandlerCount())

	// 5) Start worker processor
	processor := worker.NewProcessor(infra.Queue, registry, log, cfg)
	if err := processor.Start(ctx); err != nil {
		log.Logf("CRITICAL failed to start processor: %v", err)
		return
	}

	// 6) Background publishers
	setup.StartPeriodicCleanup(ctx, infra.Queue, cfg, log)

	// 7) Graceful shutdown
	setup.HandleShutdown(log, processor)
}
