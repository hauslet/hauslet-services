# Platform: Logger

This module provides a centralized logging solution for the application.

## Purpose

The `logger` platform module is responsible for initializing and configuring the application's logger. It ensures that all parts of the application use a consistent logging format and level.

By centralizing the logger setup, we can easily switch the logging library, change log formats (e.g., from plain text to JSON), or adjust log levels across the entire application from one place.

## Technology

- **Library**: [`github.com/go-pkgz/lgr`](https://github.com/go-pkgz/lgr) - A flexible logging library for Go.

## Usage

A logger instance is typically created at application startup and passed down to components that require logging.

### Example Initialization

```go
// In main.go
import (
    "hauslet/config"
    "hauslet/internal/platform/logger"
)

// ...

cfg := config.Load()

// Initialize the logger based on environment
log := logger.New()
if cfg.App.Env != "development" {
    log = logger.NewProduction()
}

// Inject the logger instance
server := server.New(..., log, ...)
```

### Example Usage in a Service

```go
import "github.com/go-pkgz/lgr"

type MyService struct {
    log *lgr.Logger
}

func (s *MyService) DoSomething(userID string) {
    s.log.Logf("INFO Starting operation for user %s", userID)

    // ... logic ...

    if err != nil {
        s.log.Logf("ERROR Operation failed for user %s: %v", userID, err)
        return
    }

    s.log.Logf("DEBUG Operation finished successfully for user %s", userID)
}
```
