# Push Notification Platform

FCM (Firebase Cloud Messaging) push notification support for Hauslet Services.

## Overview

The `internal/platform/push` module provides a clean interface for sending push notifications using Firebase Cloud Messaging. It follows the same patterns as other platform modules (email, SMS) for consistency.

## Features

- ✅ Send to single device token
- ✅ Send to multiple device tokens (multicast)
- ✅ Send to topics
- ✅ Topic subscription management
- ✅ Comprehensive error handling
- ✅ Structured logging
- ✅ Platform-specific options (Android priority, iOS badge/sound)

## Usage

### Initialize the Client

```go
import (
    "context"
    "log/slog"
    "hauslet/internal/platform/push"
)

func main() {
    ctx := context.Background()
    logger := slog.Default()
    
    // Create FCM provider
    provider, err := push.NewFCMProvider(
        ctx,
        "path/to/firebase-credentials.json",
        "your-project-id",
        logger,
    )
    if err != nil {
        panic(err)
    }
    
    // Create push client
    client := push.New(provider, logger)
}
```

### Send to Single Device

```go
notification := push.NotificationPayload{
    Title: "New Booking",
    Body:  "You have a new booking for Unit 3A",
    ImageURL: "https://example.com/image.jpg",
}

data := map[string]string{
    "booking_id": "booking-123",
    "type": "new_booking",
}

resp, err := client.SendToToken(ctx, deviceToken, notification, data)
if err != nil {
    log.Printf("Failed to send notification: %v", err)
    return
}

log.Printf("Notification sent: %s", resp.MessageID)
```

### Send to Multiple Devices

```go
tokens := []string{"token1", "token2", "token3"}

resp, err := client.SendToTokens(ctx, tokens, notification, data)
if err != nil {
    log.Printf("Failed to send notifications: %v", err)
    return
}

log.Printf("Sent: %d successful, %d failed", resp.SuccessCount, resp.FailureCount)
```

### Send to Topic

```go
resp, err := client.SendToTopic(ctx, "new-listings", notification, data)
if err != nil {
    log.Printf("Failed to send topic notification: %v", err)
    return
}

log.Printf("Topic notification sent: %s", resp.MessageID)
```

### Manage Topic Subscriptions

```go
// Subscribe devices to a topic
err := client.SubscribeToTopic(ctx, tokens, "booking-updates")
if err != nil {
    log.Printf("Failed to subscribe to topic: %v", err)
}

// Unsubscribe devices from a topic
err = client.UnsubscribeFromTopic(ctx, tokens, "booking-updates")
if err != nil {
    log.Printf("Failed to unsubscribe from topic: %v", err)
}
```

### Advanced Options

```go
req := push.PushRequest{
    Tokens: []string{deviceToken},
    Notification: push.NotificationPayload{
        Title: "Payment Received",
        Body:  "Your payment has been processed",
    },
    Data: map[string]string{
        "payment_id": "pay-456",
    },
    AndroidPriority: "high",
    IOSBadge: intPtr(1),
    IOSSound: "default",
    TTL: 3600, // 1 hour
}

resp, err := client.Send(ctx, req)
```

## Device Token Management

This platform module handles **sending** notifications. You need to manage device tokens separately:

### Option 1: Store in User Profile

Add a `device_tokens` JSONB column to your users/profiles table:

```sql
ALTER TABLE profiles ADD COLUMN device_tokens JSONB DEFAULT '[]'::jsonb;
```

### Option 2: Separate Device Table

Create a dedicated devices table:

```sql
CREATE TABLE user_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    device_token TEXT NOT NULL UNIQUE,
    platform TEXT NOT NULL, -- 'ios' or 'android'
    app_version TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### Option 3: Application Layer

Pass tokens directly when calling the push service from your modules.

## Error Handling

The module provides specific errors for different failure scenarios:

```go
import "errors"

resp, err := client.SendToToken(ctx, token, notification, data)
if err != nil {
    switch {
    case errors.Is(err, push.ErrInvalidToken):
        // Token is invalid or expired - remove from database
    case errors.Is(err, push.ErrRateLimited):
        // Rate limited - retry with backoff
    case errors.Is(err, push.ErrProviderUnavailable):
        // FCM service is down - queue for retry
    default:
        // Other error
    }
}
```

## Firebase Setup

1. Create a Firebase project at https://console.firebase.google.com
2. Go to **Project Settings** → **Service Accounts**
3. Click **Generate New Private Key**
4. Save the JSON file to your server
5. Set the path in your configuration

## Configuration

Add to your config structure:

```go
type Config struct {
    Push PushConfig
}

type PushConfig struct {
    Provider           string // "fcm"
    FCMCredentialsPath string
    FCMProjectID       string
}
```

## Example: Booking Notification

```go
// In your booking service
func (s *BookingService) notifyHostOfNewBooking(ctx context.Context, booking *domain.Booking) error {
    // Get host's device tokens from profile
    tokens, err := s.profileRepo.GetUserDeviceTokens(ctx, booking.HostID)
    if err != nil || len(tokens) == 0 {
        return nil // Skip if no tokens
    }
    
    // Prepare notification
    notification := push.NotificationPayload{
        Title: "New Booking Request",
        Body:  fmt.Sprintf("New booking from %s for %s", booking.GuestName, booking.Property.Title),
    }
    
    data := map[string]string{
        "type": "new_booking",
        "booking_id": booking.ID.String(),
        "action": "view_booking",
    }
    
    // Send notification
    _, err = s.pushClient.SendToTokens(ctx, tokens, notification, data)
    return err
}
```

## Testing

Run tests:

```bash
go test ./internal/platform/push/... -v
```

## Future Enhancements

- [ ] Support for additional providers (OneSignal, APNS directly)
- [ ] Notification templates
- [ ] Scheduled notifications
- [ ] Analytics and delivery tracking
- [ ] Multi-language support
