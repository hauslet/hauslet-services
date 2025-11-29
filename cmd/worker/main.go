package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hauslet/config"
	"hauslet/internal/platform/email"
	"hauslet/internal/platform/logger"
	platformQueue "hauslet/internal/platform/queue"
	"hauslet/internal/queue"
	"hauslet/internal/workers"
	"hauslet/internal/workers/handlers"

	"github.com/go-pkgz/lgr"
)

const ShutdownTimeout = 30 * time.Second

func main() {
	cfg := config.Load()
	log := logger.New()

	if cfg.App.Env != "development" {
		log = logger.NewProduction()
	}

	log.Logf("INFO 🔧 Starting worker in %s mode", cfg.App.Env)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize queue client
	emailSubject := cfg.YAML.Queue.Subjects["email"]
	queueClient, err := platformQueue.New(
		ctx,
		cfg.Infra.NATS.URL,
		cfg.YAML.Queue.StreamName,
		[]string{emailSubject}, // All subjects we'll handle
	)
	if err != nil {
		log.Logf("ERROR Failed to initialize queue: %v", err)
		return
	}
	defer queueClient.Close()

	// Initialize email client
	emailClient := initializeEmailClient(cfg, log)

	// Setup handler registry
	registry := queue.NewRegistry()

	// Register email handler
	emailHandler := handlers.NewEmailHandler(emailClient, log, emailSubject)
	registry.Register(emailHandler)

	// TODO: Register other handlers as they're implemented
	// registry.Register(notificationHandler)
	// registry.Register(moderationHandler)

	log.Logf("INFO ✅ Registered %d job handlers", registry.HandlerCount())

	// Start worker processor
	processor := workers.NewProcessor(queueClient, registry, log, cfg)

	if err := processor.Start(ctx); err != nil {
		log.Logf("ERROR Failed to start processor: %v", err)
		return
	}

	// Graceful shutdown
	handleShutdown(log, processor)
}

// handleShutdown manages graceful shutdown on signals
func handleShutdown(log *lgr.Logger, processor *workers.Processor) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	log.Logf("WARN Shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()

	if err := processor.Stop(shutdownCtx); err != nil {
		log.Logf("ERROR Worker shutdown error: %v", err)
	} else {
		log.Logf("INFO Worker shutdown cleanly")
	}
}

// initializeEmailClient creates the appropriate email client based on environment
func initializeEmailClient(cfg *config.GlobalConfig, log *lgr.Logger) *email.Client {
	var emailSender email.Sender

	switch cfg.App.Env {
	case "development", "testing":
		emailSender = email.NewSMTPAdapter(
			cfg.Services.Email.SMTP.Host,
			log,
			cfg.Services.Email.SMTP.Port,
			cfg.Services.Email.SMTP.User,
			cfg.Services.Email.SMTP.Pass,
			cfg.Services.Email.From,
		)
		log.Logf("INFO ✅ SMTP email adapter initialized")
	default:
		emailSender = email.NewResendAdapter(
			cfg.Services.Email.Resend.APIKey,
			cfg.Services.Email.From,
		)
		log.Logf("INFO ✅ Resend email adapter initialized")
	}

	return email.New(emailSender)
}
