# Background Worker

This directory contains the main entry point for the Hauslet background worker.

## Purpose

The `worker` is a standalone application responsible for processing asynchronous jobs from the NATS JetStream queue. It uses a generic processing framework that allows new job types to be added with minimal effort.

Key responsibilities include:
- Connecting to NATS JetStream.
- Initializing a `Registry` of job handlers.
- Subscribing to relevant NATS subjects.
- Fetching jobs, dispatching them to the correct handler, and managing their lifecycle (e.g., retries, acknowledgements).

## How to Run

To start the background worker, run the following command from the project root:

```bash
go run ./cmd/worker/main.go
```

The worker will connect to NATS and begin processing jobs from the configured stream.

## Architecture

The worker uses a handler registry pattern:
1.  **Job Handlers**: Each job type (e.g., `email`, `notification`) has a corresponding `Handler` in `internal/workers/handlers/`.
2.  **Registry**: In `main.go`, each handler is instantiated and registered with the `Registry` from `internal/queue/`.
3.  **Processor**: The `Processor` in `internal/workers/` subscribes to the queue and uses the registry to route incoming jobs to the correct handler.

This design decouples the worker infrastructure from the business logic of the jobs themselves. For more details, see the [queue module documentation](../../internal/queue/README.md).
