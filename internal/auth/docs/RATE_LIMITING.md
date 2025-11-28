# Rate Limiting Implementation

## Overview

Rate limiting is **automatically enabled in production** and **disabled in development** for easier testing.

## Usage

### Option 1: Routes WITHOUT Rate Limiting (Development)

```go
// In your server setup
authHandler := port.NewHTTPHandler(ctx, authService, log)
authHandler.SetupRoutes(r)
```

### Option 2: Routes WITH Rate Limiting (Production-aware)

```go
// In your server setup
authHandler := port.NewHTTPHandler(ctx, authService, log)
authHandler.SetupRoutesWithRateLimiting(r, redisClient, cfg.App.Env)
```

## Rate Limit Configuration

When `APP_ENV=production`, these rate limits apply:

| Endpoint | Rate Limit | Window | Purpose |
|----------|-----------|--------|---------|
| `POST /register` | 3 requests | 15 minutes | Prevent bot registrations |
| `POST /change-password` | 5 requests | 1 hour | Prevent brute force |
| `PUT /me` | 20 requests | 1 minute | Prevent abuse |

All other endpoints have no specific rate limits but share the global Redis-based limiting.

## Testing

### Development (Rate Limiting Disabled)
```bash
APP_ENV=development go run cmd/api/main.go
# No rate limits - test freely
```

### Production (Rate Limiting Enabled)
```bash
APP_ENV=production go run cmd/api/main.go
# Rate limits active

# Test registration limit (should block after 3 attempts)
for i in {1..5}; do
  curl -X POST http://localhost:3000/register \
    -H "Content-Type: application/json" \
    -d '{"email":"test'$i'@example.com","password":"password123","name":"Test"}'
  echo "\n---"
done
```

## Customizing Rate Limits

Edit the rate limit configs in `internal/auth/port/http.go`:

```go
// Adjust these values
r.With(applyRateLimit(middleware.RateLimitConfig{
    Requests: 3,              // Change to your desired limit
    Window:   15 * time.Minute, // Change to your desired window
})).Post("/register", h.Register)
```

## Rate Limit Response

When rate limited, clients receive:

```json
{
  "message": "Rate limit exceeded. Try again in 14m30s",
  "title": "Too Many Requests"
}
```

HTTP Status: `429 Too Many Requests`

## Architecture

- **IP-based limiting**: Uses `X-Forwarded-For` header or `RemoteAddr`
- **Redis-backed**: Distributed rate limiting across multiple instances
- **Graceful degradation**: If Redis fails, requests are allowed through
- **Environment-aware**: Automatically disabled in development
