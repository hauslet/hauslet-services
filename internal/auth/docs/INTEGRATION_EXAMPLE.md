# Integration Example

## How to Use Rate-Limited Auth Routes in Your Server

### Step 1: In your `cmd/api/server/routes.go` or similar file

```go
package server

import (
    "hauslet/internal/auth/port"
    "hauslet/internal/auth/service"
    "hauslet/internal/auth/repository"
    "hauslet/config"

    "github.com/go-chi/chi/v5"
    "github.com/go-pkgz/lgr"
    "github.com/redis/go-redis/v9"
    "gorm.io/gorm"
)

func SetupRoutes(
    r *chi.Mux,
    db *gorm.DB,
    redisClient *redis.Client,
    log *lgr.Logger,
    cfg *config.GlobalConfig,
) {
    // Initialize auth components
    authRepo := repository.NewAuthRepository(db, redisClient)
    authService := service.NewAuthService(&cfg.Auth, authRepo)
    authHandler := port.NewHTTPHandler(r.Context(), authService, log)

    // Choose one of these options:

    // OPTION 1: With production-aware rate limiting (RECOMMENDED)
    authHandler.SetupRoutesWithRateLimiting(r, redisClient, cfg.App.Env)

    // OPTION 2: Without rate limiting (simple, for development only)
    // authHandler.SetupRoutes(r)
}
```

### Step 2: Verify your .env file

```bash
# Development - Rate limiting DISABLED
APP_ENV=development

# Production - Rate limiting ENABLED
APP_ENV=production
```

### Step 3: Test it

**Development (no limits):**
```bash
APP_ENV=development go run cmd/api/main.go

# Register as many times as you want - no blocking
curl -X POST http://localhost:3000/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123","name":"Test"}'
```

**Production (limits active):**
```bash
APP_ENV=production go run cmd/api/main.go

# Try registering 5 times - gets blocked after 3
for i in {1..5}; do
  curl -X POST http://localhost:3000/register \
    -H "Content-Type: application/json" \
    -d '{"email":"test'$i'@example.com","password":"password123","name":"Test"}'
  echo "\n"
done
```

## That's it!

Rate limiting is now:
- ✅ Automatically enabled in production
- ✅ Automatically disabled in development
- ✅ No complex configuration needed
- ✅ Works with your existing Redis setup
