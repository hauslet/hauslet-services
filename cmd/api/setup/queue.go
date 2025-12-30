package setup

import (
	"context"

	"hauslet/config"
	"hauslet/internal/platform/queue"

	"github.com/go-pkgz/lgr"
)

// InitQueue initializes the Cloud Tasks client.
func InitQueue(ctx context.Context, cfg *config.GlobalConfig, log *lgr.Logger) (*queue.Client, error) {
	queueCfg := queue.Config{
		ProjectID:           cfg.Infra.CloudTasks.ProjectID,
		Location:            cfg.Infra.CloudTasks.Location,
		WorkerBaseURL:       cfg.Infra.CloudTasks.WorkerBaseURL,
		ServiceAccountEmail: cfg.Infra.CloudTasks.ServiceAccountEmail,
		Environment:         cfg.App.Env,
	}
	return queue.New(ctx, queueCfg, cfg.YAML.Queue.Subjects, log)
}
