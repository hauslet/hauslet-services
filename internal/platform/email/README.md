# Platform: Email

This module provides a generic interface for sending emails, with concrete adapters for different email providers.

## Purpose

The `email` platform module abstracts the details of sending transactional emails. It defines a common `EmailClient` interface and provides implementations (adapters) for different services like SMTP or Resend.

This allows the application's business logic to send emails without being coupled to a specific provider.

## Key Components

- `interface.go`: Defines the `EmailClient` interface with methods like `Send`.
- `smtp_adapter.go`: An implementation of `EmailClient` that sends emails via a standard SMTP server.
- `resend_adapter.go`: An implementation of `EmailClient` that uses the Resend API.
- `client.go`: A factory function (`NewClient`) that returns the appropriate email client based on the application's configuration.

## Usage

The email client is typically initialized at application startup and injected as a dependency into services that need to send emails.

### Example Initialization & Usage

```go
// In main.go
import "hauslet/internal/platform/email"

// ...

// Create a new email client based on config
emailClient := email.NewClient(cfg.Services.Email)

// Inject the client into a service
authService := service.NewAuthService(..., emailClient)


// --- In a service method ---

// Use the client to send an email
err := s.emailClient.Send(ctx, "recipient@example.com", "Welcome!", "<h1>Hello</h1>")
if err != nil {
    // Handle error
}
```
