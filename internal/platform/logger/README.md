# Platform: Logger

This module provides a centralized, structured logging solution for the application with Google Cloud Logging compatibility.

## Purpose

The `logger` platform module is responsible for initializing and configuring the application's logger. It provides:

- **Structured JSON logging** for machine-readable logs
- **Google Cloud Logging compatibility** with proper field mapping
- **Consistent log format** across the entire application
- **Production-ready configuration** for Cloud Run deployments

By centralizing the logger setup, we can easily adjust log levels, change output formats, or modify cloud provider integrations from one place.

## Technology

- **Library**: [`log/slog`](https://pkg.go.dev/log/slog) - Go's standard structured logging library (Go 1.21+)
- **Handler**: `JSONHandler` for structured output
- **Output**: Writes to `os.Stdout` for container/Cloud Run compatibility

## Implementation Details

### Field Mapping for Google Cloud

The logger automatically renames standard `slog` fields to match Google Cloud Logging schema:

- `level` → `severity` (Google Cloud standard)
- `msg` → `message` (consistency and readability)

### Configuration

- **Default Level**: `INFO` (filters out DEBUG logs)
- **Format**: JSON (structured logging)
- **Output**: stdout (Cloud Run/container-friendly)

## Usage

A logger instance is created at application startup using `logger.NewLogger()` and passed to components via dependency injection.

### Example Initialization

```go
// In main.go or cmd/api/main.go
import (
    "hauslet/internal/platform/logger"
)

func main() {
    // Initialize logger
    log := logger.NewLogger()
    
    // Pass to services
    paymentService := payment.NewService(repo, log)
    server := server.New(paymentService, log)
}
```

### Example Usage in a Service

```go
import "log/slog"

type MyService struct {
    log *slog.Logger 
}

func NewMyService(log *slog.Logger) *MyService {
    return &MyService{log: log}
}

func (s *MyService) DoSomething(userID string) error {
    // ✅ CORRECT: Use key-value pairs for structured logging
    s.log.Info("starting operation", "user_id", userID)

    // ... business logic ...

    if err != nil {
        s.log.Error("operation failed", "user_id", userID, "error", err)
        return err
    }

    s.log.Info("operation completed successfully", "user_id", userID)
    return nil
}

func (s *MyService) ProcessPayment(amount int64, currency string) {
    // Multiple attributes
    s.log.Info("processing payment",
        "amount", amount,
        "currency", currency,
        "timestamp", time.Now())
}
```

### ❌ Common Mistakes (Printf-Style)

**Don't use printf-style formatting with slog:**

```go
// ❌ WRONG: Printf-style formatting
s.log.Info("starting operation for user %s", userID)
s.log.Error("operation failed: %v", err)

// ✅ CORRECT: Key-value pairs
s.log.Info("starting operation", "user_id", userID)
s.log.Error("operation failed", "error", err)
```

## Log Levels

- `Debug()` - Development debugging (filtered out by default)
- `Info()` - General informational messages
- `Warn()` - Warning messages for recoverable issues
- `Error()` - Error messages for failures

## Google Cloud Integration

When deployed to Cloud Run, logs are automatically:

1. **Parsed as JSON** by Google Cloud Logging
2. **Indexed by fields** (user_id, error, etc.)
3. **Filterable** in Cloud Console using structured queries
4. **Correlated** with trace IDs and request contexts

### Example Cloud Logging Query

```txt
severity="ERROR"
jsonPayload.user_id="123e4567-e89b-12d3-a456-426614174000"
```

## Best Practices

1. **Always use key-value pairs** instead of string formatting
2. **Use consistent key names** across the codebase (e.g., `user_id` not `userId` or `userID`)
3. **Include context** in logs (IDs, timestamps, states)
4. **Log errors with the error object** for stack traces
5. **Avoid logging sensitive data** (passwords, tokens, credit cards)
