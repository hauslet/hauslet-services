# Worker Implementation

This module contains the concrete implementation of the background worker, including the generic processor and the specific job handlers.

## Purpose

While `internal/queue` defines the *framework*, this `workers` module provides the *implementation*. It is responsible for the actual processing of jobs.

## Key Components

- `processor.go`: This file contains the generic `Processor`. It is the core of the worker application. It subscribes to NATS subjects, receives messages, and uses the `Registry` (from `internal/queue`) to delegate the message to the appropriate handler. It also contains logic for message acknowledgement and retries.

- `processor_test.go`: An integration test for the worker processor, ensuring the flow from message reception to handler execution works correctly.

- `handlers/`: This directory contains the concrete implementations of `JobHandler` for each job type.
    - `handlers/email.go`: The handler responsible for processing `EmailJob` types. It deserializes the job payload and uses the email platform service to send the email.

## How to Add a New Job Handler

1.  **Create the Job**: Define your new job struct in `internal/queue/jobs/` (e.g., `MyJob`).
2.  **Create the Handler**: Create a new file in `internal/workers/handlers/` (e.g., `my_job.go`).
3.  **Implement the Handler**: In the new file, create a struct (e.g., `MyJobHandler`) and implement the `Handle(ctx context.Context, data []byte) error` method from the `JobHandler` interface. This method will contain the business logic for processing `MyJob`.
4.  **Register the Handler**: In `cmd/worker/main.go`, instantiate your new `MyJobHandler` and register it with the `Registry`.

This structure ensures that all business logic for handling specific jobs is located here, while the core processing and queueing infrastructure remains separate.
