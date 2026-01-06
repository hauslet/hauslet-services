# Platform: Queue

This module provides the Google Cloud Tasks client used by services to enqueue background jobs targeted at the worker deployment.

## Purpose

The `internal/platform/queue` package owns the infrastructure-facing concerns of Cloud Tasks:

- Establishing a Cloud Tasks client with project, region, and service-account details.
- Resolving queue routes (`QueueRoute`) from the YAML configuration.
- Publishing HTTP tasks to the worker with consistent timeouts and authorization headers.

Domain modules interact with a higher-level service API (see `internal/queue`) that delegates the actual transport work to this package.

## Usage

The queue client is created during application startup and shared with services that need to dispatch work to the worker. Routes are derived from `cfg.YAML.Queue.Subjects` and must match the worker’s registered handlers.

### Example Initialization

```go
import (
    "context"
    "log/slog"

    "hauslet/config"
    platformqueue "hauslet/internal/platform/queue"
)

func initQueue(ctx context.Context, cfg *config.GlobalConfig, log *slog.Logger) (*platformqueue.Client, error) {
    qCfg := platformqueue.Config{
        ProjectID:           cfg.Infra.CloudTasks.ProjectID,
        Location:            cfg.Infra.CloudTasks.Location,
        WorkerBaseURL:       cfg.Infra.CloudTasks.WorkerBaseURL,
        ServiceAccountEmail: cfg.Infra.CloudTasks.ServiceAccountEmail,
        Environment:         cfg.App.Env,
    }

    return platformqueue.New(ctx, qCfg, cfg.YAML.Queue.Subjects, log)
}
```

Once instantiated, services call `Publish` with the resolved queue name. The client handles JSON encoding, attaches the correct HTTP endpoint, and injects the optional OIDC token.

```go
if err := queueClient.Publish(ctx, cfg.YAML.Queue.Subjects["email"], payload); err != nil {
    log.Error("failed to enqueue email task", "error", err)
}
```

`Client.AllowFallback()` signals whether it is acceptable to run inline fallbacks (disabled in production to avoid double execution). Use this when deciding whether to execute work synchronously after a publish failure.
