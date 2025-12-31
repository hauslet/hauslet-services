# Platform: Queue

This module provides the low-level client for interacting with the NATS JetStream message broker.

## Purpose

The `queue` platform module is a technical abstraction responsible for:
- Establishing a connection to the NATS server.
- Creating a JetStream context.
- Providing a simple client that can be used to publish messages to NATS subjects.

This module deals with the infrastructure-level concerns of connecting to the queue, while the `internal/queue` module provides the higher-level abstractions for jobs and handlers.

## Usage

The queue client is initialized once at application startup. It connects to NATS, ensures the specified stream exists, and provides a `Client` instance that can be used to publish jobs.

### Example Initialization

```go
// In main.go
import (
    "context"
    "hauslet/config"
    "hauslet/internal/platform/queue"
)

// ...

cfg := config.Load()
ctx := context.Background()

// Define the subjects this client will publish to
queueSubjects := []string{cfg.YAML.Queue.Subjects["email"]}

// Initialize the queue client
queueClient, err := queue.New(ctx, cfg.Infra.NATS.URL, cfg.YAML.Queue.StreamName, queueSubjects)
if err != nil {
    log.Warn("⚠️ failed to initialize NATS queue: %v", err)
    // Handle error, perhaps by using a fallback mechanism
} else {
    defer queueClient.Close()
    log.Info(" ✅ NATS queue initialized")
}


// Inject the 'queueClient' into services that need to dispatch jobs
emailService := service.NewEmailService(..., queueClient)
```
