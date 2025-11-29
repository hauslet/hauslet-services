# Platform: Redis

This module provides the core functionality for connecting to and interacting with the Redis server.

## Purpose

The `redis` platform module is a technical abstraction responsible for:
- Establishing and managing the connection pool to the Redis server.
- Providing a single, initialized `*redis.Client` instance for the rest of the application to use.

This centralizes the Redis connection logic, making it easy to manage and inject into various parts of the application that require caching, session storage, or other Redis-based functionality.

## Usage

The Redis client is typically initialized once at application startup using `InitRedis`. The client instance can then be retrieved with `GetRedis` and injected as a dependency into components that require it, such as session stores or rate limiters.

### Example Initialization

```go
// In main.go
import (
    "context"
    "hauslet/config"
    "hauslet/internal/platform/redis"
)

// ...

// Load application configuration
cfg := config.Load()
ctx := context.Background()

// Initialize the Redis connection pool
err := redis.InitRedis(&cfg.Storage.Redis, ctx)
if err != nil {
    log.Fatalf("Failed to initialize Redis: %v", err)
}
defer redis.CloseRedis()

// Retrieve the client to inject into other services
redisClient, err := redis.GetRedis()
if err != nil {
    log.Fatalf("Failed to get Redis client: %v", err)
}

// Inject the 'redisClient' instance
sessionStore := session.NewStore(redisClient)
```
