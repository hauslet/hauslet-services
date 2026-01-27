# Device Token Management Plan (Simplified)

**Purpose**: Store device tokens to enable push notifications beyond session lifetime.

**Target Audience**: Backend development team  
**Related Modules**: Auth, Push Platform  
**Database**: PostgreSQL

---

## Goals (MVP)

1. **Store device tokens** for FCM push notifications
2. **Associate tokens with users** to send targeted notifications
3. **Support token updates** when FCM tokens refresh
4. **Enable cross-module push** - any module can send notifications

## Out of Scope (Future Work)

- ❌ Device fingerprinting
- ❌ Session-device validation middleware
- ❌ Anomaly/location detection
- ❌ IP geolocation service
- ❌ Security audit events table
- ❌ Trusted device management
- ❌ Device management UI / "My Devices" screens
- ❌ New device login alerts

---

## Database Schema

### `user_devices` Table

```sql
CREATE TABLE user_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Ownership
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Device Identity
    device_token TEXT NOT NULL,
    device_platform TEXT NOT NULL CHECK (device_platform IN ('ios', 'android', 'web')),
    device_name TEXT,               -- "iPhone 13 Pro", "Chrome on Windows"
    
    -- Activity Tracking  
    last_active_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Lifecycle
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Constraints
    UNIQUE (user_id, device_token)
);

-- Indexes
CREATE INDEX idx_user_devices_user_id ON user_devices(user_id);
CREATE INDEX idx_user_devices_token ON user_devices(device_token);
CREATE INDEX idx_user_devices_last_active ON user_devices(last_active_at DESC);

-- Auto-update updated_at
CREATE TRIGGER update_user_devices_updated_at
    BEFORE UPDATE ON user_devices
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

---

## Module Structure

```
internal/modules/devices/
├── domain/
│   ├── device.go           # Core domain model
│   └── errors.go           # Device-specific errors
├── repository/
│   ├── schema/
│   │   ├── device.go       # Postgres schema
│   │   └── mapper.go       # Schema ↔ Domain mapping
│   ├── interface.go        # Repository interface
│   └── device_repo.go      # Implementation
├── service/
│   ├── interface.go        # Service interface
│   └── device_service.go   # Core device operations
└── port/
    └── graphql/
        ├── schema.graphqls # GraphQL types
        └── resolvers.go    # Resolvers
```

---

## Domain Model

```go
// internal/modules/devices/domain/device.go
package domain

import (
    "time"
    "github.com/google/uuid"
)

type Device struct {
    ID             uuid.UUID
    UserID         uuid.UUID
    DeviceToken    string
    DevicePlatform Platform
    DeviceName     string
    LastActiveAt   time.Time
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

type Platform string

const (
    PlatformIOS     Platform = "ios"
    PlatformAndroid Platform = "android"
    PlatformWeb     Platform = "web"
)
```

---

## Service Interface

```go
// internal/modules/devices/service/interface.go
package service

import (
    "context"
    "hauslet/internal/modules/devices/domain"
    "github.com/google/uuid"
    "time"
)

type DeviceService interface {
    // Device Registration - upserts device token
    RegisterDevice(ctx context.Context, req RegisterDeviceRequest) (*domain.Device, error)
    
    // Update token (FCM token refresh)
    UpdateDeviceToken(ctx context.Context, userID uuid.UUID, oldToken, newToken string) error
    
    // Get tokens for push notifications
    GetUserDeviceTokens(ctx context.Context, userID uuid.UUID) ([]string, error)
    GetActiveDeviceTokens(ctx context.Context, userID uuid.UUID, activeWithin time.Duration) ([]string, error)
    
    // Cleanup - remove stale tokens
    RemoveDeviceToken(ctx context.Context, userID uuid.UUID, token string) error
    RemoveInvalidToken(ctx context.Context, token string) error  // Called by FCM on invalid tokens
}

type RegisterDeviceRequest struct {
    UserID     uuid.UUID
    Token      string
    Platform   domain.Platform
    DeviceName string
}
```

---

## Auth Integration

### Register Device at Login

Modify login flows to accept optional device token:

```go
// When user logs in (OAuth or password), capture and register device token
func (s *AuthService) handleSuccessfulLogin(ctx context.Context, userID uuid.UUID, deviceInfo *DeviceInfo) {
    if deviceInfo != nil && deviceInfo.Token != "" {
        _, err := s.deviceService.RegisterDevice(ctx, service.RegisterDeviceRequest{
            UserID:     userID,
            Token:      deviceInfo.Token,
            Platform:   deviceInfo.Platform,
            DeviceName: deviceInfo.DeviceName,
        })
        if err != nil {
            s.log.Warn("Failed to register device token", "error", err)
            // Non-blocking - don't fail login
        }
    }
}
```

### Update GraphQL Login Mutations

```graphql
input DeviceTokenInput {
  token: String!
  platform: DevicePlatform!
  deviceName: String
}

enum DevicePlatform {
  IOS
  ANDROID
  WEB
}

extend type Mutation {
  # Existing login mutations - add optional device input
  loginWithPassword(email: String!, password: String!, device: DeviceTokenInput): AuthPayload!
  
  # Standalone device registration (for registering after login)
  registerDeviceToken(input: DeviceTokenInput!): Boolean!
  
  # Update token when FCM refreshes
  updateDeviceToken(oldToken: String!, newToken: String!): Boolean!
}
```

---

## Push Notification Usage

### Example: Booking Module

```go
// internal/modules/booking/service/notifications.go
func (s *BookingService) NotifyHostOfNewBooking(ctx context.Context, booking *domain.Booking) error {
    // Get host's device tokens (active in last 30 days)
    tokens, err := s.deviceService.GetActiveDeviceTokens(ctx, booking.HostID, 30*24*time.Hour)
    if err != nil || len(tokens) == 0 {
        s.log.Info("No active devices for host", "hostID", booking.HostID)
        return nil
    }
    
    notification := push.NotificationPayload{
        Title: "New Booking Request",
        Body:  fmt.Sprintf("%s wants to book %s", booking.GuestName, booking.PropertyTitle),
    }
    
    data := map[string]string{
        "type":       "new_booking",
        "booking_id": booking.ID.String(),
    }
    
    resp, err := s.pushClient.SendToTokens(ctx, tokens, notification, data)
    if err != nil {
        return fmt.Errorf("failed to send push: %w", err)
    }
    
    // Handle invalid tokens from response
    for _, invalidToken := range resp.InvalidTokens {
        _ = s.deviceService.RemoveInvalidToken(ctx, invalidToken)
    }
    
    return nil
}
```

---

## FCM Invalid Token Handling

When FCM returns errors for invalid tokens, clean them up:

```go
// Enhance push platform to return invalid tokens
type MulticastResponse struct {
    SuccessCount  int
    FailureCount  int
    InvalidTokens []string  // Tokens that are no longer valid
}

// In FCM adapter, after sending:
func (f *FCMProvider) parseMulticastResponse(tokens []string, resp *messaging.BatchResponse) *MulticastResponse {
    result := &MulticastResponse{
        SuccessCount: resp.SuccessCount,
        FailureCount: resp.FailureCount,
    }
    
    for i, sendResp := range resp.Responses {
        if sendResp.Error != nil {
            errMsg := sendResp.Error.Error()
            if strings.Contains(errMsg, "registration-token-not-registered") ||
               strings.Contains(errMsg, "invalid-registration-token") {
                result.InvalidTokens = append(result.InvalidTokens, tokens[i])
            }
        }
    }
    
    return result
}
```

---

## Implementation Timeline

### Week 1: Core Implementation (3-4 days)

| Day | Task |
|-----|------|
| 1 | Database migration + domain models |
| 2 | Repository implementation |
| 3 | Service layer + GraphQL API |
| 4 | Auth integration (login flows) |

### Week 2: Integration + Testing (2-3 days)

| Day | Task |
|-----|------|
| 1 | FCM invalid token handling |
| 2 | Integration with 1-2 modules (booking, messages) |
| 3 | Testing + deployment |

---

## Files to Create

| Path | Description |
|------|-------------|
| `db/migrations/XXXXXX_create_user_devices.up.sql` | Migration |
| `db/migrations/XXXXXX_create_user_devices.down.sql` | Rollback |
| `internal/modules/devices/domain/device.go` | Domain model |
| `internal/modules/devices/domain/errors.go` | Error types |
| `internal/modules/devices/repository/schema/device.go` | DB schema |
| `internal/modules/devices/repository/schema/mapper.go` | Mappers |
| `internal/modules/devices/repository/interface.go` | Repo interface |
| `internal/modules/devices/repository/device_repo.go` | Repo impl |
| `internal/modules/devices/service/interface.go` | Service interface |
| `internal/modules/devices/service/device_service.go` | Service impl |
| `internal/modules/devices/port/graphql/schema.graphqls` | GraphQL |
| `internal/modules/devices/port/graphql/resolvers.go` | Resolvers |

---

## Future Enhancements (Post-MVP)

When needed, these can be added:

1. **Device Management UI** - Let users see/revoke their devices
2. **Security Alerts** - Notify on new device logins
3. **Stale Token Cleanup Job** - Remove devices inactive for 90+ days
4. **Device Limits** - Cap devices per user (e.g., max 10)

---

**Document Version**: 2.0 (Simplified)  
**Last Updated**: 2026-01-27  
**Status**: Ready for Implementation
