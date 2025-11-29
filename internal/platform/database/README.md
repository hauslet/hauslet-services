# Platform: Database

This module provides the core functionality for connecting to and interacting with the primary database.

## Purpose

The `database` platform module is a technical abstraction responsible for:
- Establishing and managing the connection pool to the PostgreSQL database using `GORM`.
- Providing a single, initialized `*gorm.DB` instance for the rest of the application to use.
- Handling database migrations.

This abstraction centralizes database connection logic, ensuring that the rest of the application does not need to be aware of the underlying driver or connection details.

## Usage

The database connection is typically initialized once at application startup (in `cmd/api/main.go` or `cmd/worker/main.go`) and then injected as a dependency into repositories or services that require database access.

### Example Initialization

```go
// In main.go
import "hauslet/internal/platform/database"

// ...

// Load application configuration
cfg := config.Load()

// Establish database connection
db, err := database.NewConnection(cfg.Storage.DB)
if err != nil {
    log.Fatalf("Failed to connect to database: %v", err)
}

// Inject the 'db' instance into repositories
userRepo := repository.NewUserRepository(db)
```
