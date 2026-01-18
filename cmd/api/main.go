package main

import (
	"context"
	"hauslet/cmd/api/server"
	"hauslet/cmd/api/setup"
	"hauslet/config"
	"hauslet/internal/platform/logger"
)

// @title           Hauslet API
// @version         1.0
// @description     Hauslet Services API Server.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@hauslet.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      https://dev-api.hauslet.com
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg := config.Load()
	log := logger.NewLogger()
	log.Info("🚀 Starting Hauslet server", "mode", cfg.App.Env)

	initCtx, cancelInit := context.WithTimeout(context.Background(), setup.InitTimeout)
	defer cancelInit()

	infra, err := setup.InitInfrastructure(initCtx, cfg, log)
	if err != nil {
		log.Error("failed to initialize infrastructure", "error", err)
		return
	}
	defer infra.CloseDB()
	defer infra.CloseCache()

	if err := setup.RunMigrations(infra.DB, log); err != nil {
		log.Error("failed to run migrations", "error", err)
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
		infra.KYC,
		infra.SMS,
		infra.Evidence,
		infra.RateLimiter,
		infra.CircuitBreaker,
	)
	setup.HandleServerLifecycle(srv, log)
}
