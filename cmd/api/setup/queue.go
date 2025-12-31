package setup

import (
	"context"
	"log/slog"

	"hauslet/config"
	"hauslet/internal/platform/queue"
)

// InitQueue initializes the Cloud Tasks client.
func InitQueue(ctx context.Context, cfg *config.GlobalConfig, log *slog.Logger) (*queue.Client, error) {
	queueCfg := queue.Config{
		ProjectID:           cfg.Infra.CloudTasks.ProjectID,
		Location:            cfg.Infra.CloudTasks.Location,
		WorkerBaseURL:       cfg.Infra.CloudTasks.WorkerBaseURL,
		ServiceAccountEmail: cfg.Infra.CloudTasks.ServiceAccountEmail,
		Environment:         cfg.App.Env,
	}
	return queue.New(ctx, queueCfg, cfg.YAML.Queue.Subjects, log)
}
