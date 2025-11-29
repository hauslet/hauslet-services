# Generic Queue System

This module defines the interfaces and core logic for a generic, type-safe job queue system built on top of NATS JetStream.

## Purpose

The `queue` module provides a framework for defining, dispatching, and handling asynchronous jobs. It decouples the job definition from the handling logic, making it easy to add new job types without modifying the core worker infrastructure.

This system was created in a major refactor to replace a single-purpose email queue.

## Key Components

- `job.go`: Defines the `Job` interface. Any struct that wants to be a processable job must implement this interface, which includes methods for defining the job `Type()` and `Validate()`-ing its payload.

- `handler.go`: Defines the `JobHandler` interface. A handler is responsible for the business logic of processing a specific job type.

- `registry.go`: Implements the `Registry`. This is a crucial component that maps a job's subject to its corresponding `JobHandler`. The worker uses the registry to route incoming messages to the correct handler.

- `jobs/`: This directory contains the concrete implementations of different `Job` types (e.g., `jobs/email.go`).

## Architecture & Flow

1.  **Job Definition**: A new job type is created in `internal/queue/jobs/` (e.g., `NotificationJob`). It must implement the `Job` interface.
2.  **Handler Definition**: A corresponding handler is created in `internal/workers/handlers/` (e.g., `NotificationHandler`). It implements the `JobHandler` interface.
3.  **Registration**: In `cmd/worker/main.go`, the `NotificationHandler` is instantiated and registered with the `Registry`.
4.  **Dispatch**: A service somewhere in the application creates an instance of `NotificationJob`, serializes it to JSON, and publishes it to the appropriate NATS subject.
5.  **Processing**: The generic worker (`internal/workers/processor.go`) receives the message from NATS, finds the correct handler using the `Registry`, and invokes its `Handle` method with the message payload.

This pattern ensures a clean separation of concerns and makes the worker system highly extensible.
