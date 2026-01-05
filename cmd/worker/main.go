package main

import (
	"context"
	"sync/atomic"

	"hauslet/cmd/worker/setup"
	"hauslet/config"
	"hauslet/internal/platform/logger"
)

func main() {
	// 1) Config + Logger
	cfg := config.Load()
	log := logger.NewLogger()
	log.Info("🔧 Starting worker", "mode", cfg.App.Env)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2) Core infrastructure (DB, storage, AI, email, cache)
	infra, err := setup.InitInfrastructure(ctx, cfg, log)
	if err != nil {
		log.Warn("failed to initialize infrastructure", "error", err)
		return
	}
	defer infra.CloseDB()
	defer infra.CloseCache()

	// 3) Queue client
	qClient, err := setup.InitQueue(ctx, cfg, log)
	if err != nil {
		log.Warn("failed to initialize queue", "error", err)
	}
	infra.Queue = qClient
	defer infra.CloseQueue()

	// 4) Handlers
	registry := setup.RegisterHandlers(infra, cfg, log)
	log.Info("✅ Registered job handlers", "count", registry.HandlerCount())

	ready := atomic.Bool{}
	workerServer := startWorkerServer(cfg, registry, log, &ready)

	// 6) Background publishers - Handled by Cloud Scheduler
	// Cloud Scheduler jobs trigger these endpoints directly:
	// - /tasks/media/cleanup (every 15 min)
	// - /tasks/booking/expiry (every 2 min)
	// - /tasks/booking/completion (every hour)
	// - /tasks/booking/checkin-out (every hour)
	// - /tasks/finance/payout/process (every hour)
	// - /tasks/finance/payout/retry (every 15 min)
	// - /tasks/finance/reconciliation (daily at 2 AM UTC)
	// - /tasks/review/standoff/publish (daily at midnight UTC)
	// - /tasks/review/reminders (daily at 10 AM UTC)
	// - /tasks/promotion/expiry (daily at 1 AM UTC)
	// - /tasks/promotion/billing (daily at 3 AM UTC)
	// - /tasks/calendar/showing/reminders (hourly)
	// - /tasks/calendar/open-house/reminders (hourly)

	ready.Store(true)

	// 7) Graceful shutdown
	setup.HandleShutdown(log, workerServer)
}
