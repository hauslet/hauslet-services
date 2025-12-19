package setup

import (
	"context"

	"hauslet/config"
	"hauslet/internal/platform/ai"
	"hauslet/internal/platform/database"
	"hauslet/internal/platform/email"
	"hauslet/internal/platform/logger"
	"hauslet/internal/platform/queue"
	"hauslet/internal/platform/storage"

	"github.com/go-pkgz/lgr"
	"gorm.io/gorm"
)

// Infrastructure holds shared platform dependencies for the worker.
type Infrastructure struct {
	DB      *gorm.DB
	Storage *storage.R2Storage
	AI      *ai.Client
	Email   *email.Client
	Queue   *queue.Client
}

// CloseDB closes the DB connection.
func (i *Infrastructure) CloseDB() {
	if i != nil && i.DB != nil {
		database.Close(i.DB)
	}
}

// CloseQueue closes the queue client.
func (i *Infrastructure) CloseQueue() {
	if i != nil && i.Queue != nil {
		i.Queue.Close()
	}
}

// SetupLogger initializes the logger based on environment.
func SetupLogger(env string) *lgr.Logger {
	if env != "development" {
		return logger.NewProduction()
	}
	return logger.New()
}

// InitInfrastructure establishes DB, storage, AI, and email clients.
func InitInfrastructure(ctx context.Context, cfg *config.GlobalConfig, log *lgr.Logger) (*Infrastructure, error) {
	// Database
	db, err := database.NewPostgresWithContext(ctx, &cfg.Storage.DB, cfg.App.Env)
	if err != nil {
		return nil, err
	}

	// Storage (R2)
	storageClient, err := storage.NewR2S3Client(cfg.Storage.R2)
	if err != nil {
		return nil, err
	}
	r2Storage := storage.NewR2Storage(storageClient, &cfg.Storage.R2)

	// AI providers
	geminiClient, err := ai.NewGeminiClient(ctx, cfg.Services.Gemini, r2Storage)
	if err != nil {
		return nil, err
	}
	anthropicClient, err := ai.NewAnthropicClient(ctx, cfg.Services.Anthropic, r2Storage)
	if err != nil {
		return nil, err
	}

	fallbackOpts := ai.FallbackOptions{
		Enabled:          cfg.Services.Fallback.Enable,
		AttemptThreshold: cfg.Services.Fallback.AttemptThreshold,
	}

	fallbackProvider := ai.NewFallbackClient(geminiClient, anthropicClient, fallbackOpts, log)

	if err := fallbackProvider.HealthCheck(ctx); err != nil {
		return nil, err
	}

	aiClient := ai.New(fallbackProvider)

	// Email
	emailClient := InitializeEmailClient(cfg, log)

	return &Infrastructure{
		DB:      db,
		Storage: r2Storage,
		AI:      aiClient,
		Email:   emailClient,
	}, nil
}
