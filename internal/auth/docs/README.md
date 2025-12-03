# Hauslet Authentication System

This document provides a comprehensive guide to the Hauslet authentication system. It covers everything from the overall architecture and API reference to integration examples and security features.

## Table of Contents

1.  [**Overview**](#1-overview)
    -   [System Architecture](#system-architecture)
    -   [Role Definitions](#role-definitions)
2.  [**Getting Started**](#2-getting-started)
    -   [Prerequisites](#prerequisites)
    -   [Running the Service](#running-the-service)
    -   [Testing with Postman](#testing-with-postman)
3.  [**API Reference**](#3-api-reference)
    -   [Public Endpoints](#public-endpoints)
    -   [Protected Endpoints](#protected-endpoints)
    -   [Error Responses](#error-responses)
4.  [**Authentication Flows**](#4-authentication-flows)
    -   [Password-Based Authentication](#password-based-authentication)
    -   [Google OAuth2 Flow](#google-oauth2-flow)
5.  [**Advanced Topics**](#5-advanced-topics)
    -   [Rate Limiting](#rate-limiting)
    -   [Session Management](#session-management)
    -   [Role-Based Access Control (RBAC)](#role-based-access-control-rbac)
    -   [Root User Management](#root-user-management)
6.  [**Integration Guide**](#6-integration-guide)
    -   [Server-Side Setup](#server-side-setup)
    -   [Client-Side Usage](#client-side-usage)

---

## 1. Overview

### System Architecture

The Hauslet authentication system is a robust, secure, and feature-rich module designed to handle user authentication, session management, and access control. It is built with a modular architecture, making it easy to integrate into the main application.

-   **Language**: Go
-   **Framework**: Chi (for routing)
-   **Database**: PostgreSQL (with GORM)
-   **Cache**: Redis (for session management and rate limiting)
-   **Dependencies**: `go-pkgz/auth` for core authentication logic.

### Role Definitions

The system uses a hierarchical role-based access control (RBAC) model to manage user permissions.

| Role      | Description                                                  |
| :-------- | :----------------------------------------------------------- |
| **root**  | Superuser with unrestricted access. Cannot be deleted or demoted. |
| **admin**     | Administrative access. Can manage users and content.         |
| **moderator** | Content moderation capabilities.                             |
| **staff**     | Internal staff access.                                       |
| **support**   | Customer support capabilities.                               |
| **user**      | Default role for standard registered users.                  |

---

## 2. Getting Started

### Prerequisites

-   Go 1.18+
-   Docker and Docker Compose
-   Postman (optional, for API testing)

### Running the Service

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/your-repo/hauslet-services.git
    cd hauslet-services
    ```

2.  **Set up environment variables:**
    ```bash
    cp .env.example .env
    # Fill in the required variables in .env
    ```

3.  **Start the services:**
    ```bash
    docker-compose up -d
    ```

4.  **Run the API server:**
    ```bash
    go run ./cmd/api/main.go
    ```
    The API will be available at `http://localhost:3000`.

### Testing with Postman

1.  **Import the Postman Collection:**
    -   Open Postman and click the "Import" button.
    -   Select the `Hauslet_Auth_API.postman_collection.json` file from the `internal/auth/docs` directory.

2.  **Configure the Base URL:**
    -   The collection uses a variable `{{base_url}}` which defaults to `http://localhost:3000`.
    -   You can change this by editing the collection's variables.

3.  **Start Testing:**
    -   Use the requests in the collection to test the API endpoints.
    -   The collection includes requests for registration, login, profile management, and more.

---

## 3. API Reference

### Public Endpoints

#### **`POST /register`**

Create a new user account.

-   **Request Body:**
    ```json
    {
      "email": "user@example.com",
      "password": "securePassword123",
      "name": "John Doe"
    }
    ```
-   **Success Response (201):**
    ```json
    {
      "id": "user-uuid",
      "email": "user@example.com",
      "name": "John Doe",
      "message": "Registration successful. Please verify your email to activate your account."
    }
    ```

#### **`POST /auth/login`**

Authenticate with email and password.

-   **Request Body (form-data or JSON):**
    ```json
    {
      "user": "user@example.com",
      "passwd": "securePassword123"
    }
    ```
-   **Success Response (200):**
    -   Sets a `JWT` HTTP-only cookie.

#### **`GET /auth/google/login`**

Initiates the Google OAuth2 flow.

-   **Action:** Redirects the user to the Google login page.

#### **`GET /auth/logout`**

Clears the authentication cookie and ends the session.

#### **`POST /auth/verify-email`**

Verify a user's email address with an OTP.

-   **Request Body:**
    ```json
    {
      "email": "user@example.com",
      "otp": "123456"
    }
    ```
-   **Success Response (200):**
    ```json
    {
      "message": "Email verified successfully"
    }
    ```

#### **`POST /auth/resend-otp`**

Resend the email verification OTP.

-   **Request Body:**
    ```json
    {
      "email": "user@example.com"
    }
    ```
-   **Success Response (200):**
    ```json
    {
      "message": "OTP has been resent to your email"
    }
    ```

### Protected Endpoints

All protected endpoints require a valid `JWT` cookie.

#### **`GET /me`**

Get the current user's profile.

-   **Success Response (200):**
    ```json
    {
      "id": "user-uuid",
      "name": "John Doe",
      "primary_email": "user@example.com",
      "avatar_url": "https://example.com/avatar.png",
      "role": "user",
      "is_active": true
    }
    ```

#### **`PUT /me`**

Update the current user's profile.

-   **Request Body:**
    ```json
    {
      "name": "Jane Doe",
      "email": "new.email@example.com"
    }
    ```

#### **`POST /change-password`**

Change the current user's password.

-   **Request Body:**
    ```json
    {
      "old_password": "currentPassword123",
      "new_password": "newSecurePassword456"
    }
    ```

#### **`GET /me/identities`**

List all authentication methods (e.g., password, Google) linked to the user's account.

-   **Success Response (200):**
    ```json
    [
      {
        "id": "identity-uuid",
        "provider": "password",
        "email": "user@example.com",
        "email_verified": true
      },
      {
        "id": "identity-uuid-2",
        "provider": "google",
        "email": "user@gmail.com",
        "email_verified": true
      }
    ]
    ```

#### **`DELETE /me/identities?id=<identity-id>`**

Unlink an authentication method from the user's account.

-   **Security:** Cannot unlink the last remaining identity (prevents account lockout).

#### **`GET /auth/link/{provider}`**

Initiate linking an OAuth provider to the current user's account.

-   **Providers:** Currently supports `google`.
-   **Query Parameters:**
    -   `redirect_uri` (optional): URI to redirect after linking completes. Defaults to `/settings`.
-   **Authentication:** Required (protected endpoint).
-   **Success:** Redirects to OAuth provider for authorization.
-   **Response (302):** Redirects to Google OAuth consent screen.
-   **After Authorization:** User is redirected to `redirect_uri` with status.
-   **Example:**
    ```
    GET /auth/link/google?redirect_uri=/settings
    ```
-   **Security Features:**
    -   Validates OAuth state using HMAC-signed tokens (10-minute expiration).
    -   Prevents duplicate linking (rejects if provider already linked to another user).
    -   Sends email notification when identity is successfully linked.
    -   Idempotent: Returns success if provider already linked to the same user.

**Error Scenarios:**

| Error                                       | Status Code | Message                                       |
| :------------------------------------------ | :---------- | :-------------------------------------------- |
| Provider already linked to different user   | 409         | "This provider is already linked to another account" |
| Invalid state token                         | 400         | "Invalid or expired linking state"            |
| User not authenticated                      | 401         | "Unauthorized"                                |
| OAuth provider error                        | 500         | "Failed to initiate linking"                  |

#### **`GET /me/sessions`**

List all active sessions for the current user.

-   **Success Response (200):**
    ```json
    [
      {
        "id": "session-uuid",
        "provider": "password",
        "created_at": "2023-10-27T10:00:00Z",
        "expires_at": "2023-10-28T10:00:00Z",
        "user_agent": "Chrome/118.0.0.0",
        "ip": "192.168.1.1"
      }
    ]
    ```

#### **`DELETE /me/sessions`**

Revoke all active sessions for the user (logout everywhere).

#### **`DELETE /me/session?id=<session-id>`**

Revoke a specific session.

### Error Responses

The API uses a standard error format:

```json
{
  "error": "Error Type",
  "message": "A descriptive error message.",
  "field": "field_name (optional)"
}
```

**Common Status Codes:**

-   `400 Bad Request`: Invalid input.
-   `401 Unauthorized`: Missing or invalid authentication.
-   `403 Forbidden`: Insufficient permissions.
-   `404 Not Found`: Resource not found.
-   `409 Conflict`: The resource already exists.
-   `429 Too Many Requests`: Rate limit exceeded.

---

## 4. Authentication Flows

### Password-Based Authentication

1.  The user submits their email and password to the `POST /auth/login` endpoint.
2.  The server verifies the credentials against the hashed password in the database.
3.  If successful, the server creates a session in Redis and returns a `JWT` token as an HTTP-only cookie.
4.  The client sends this cookie with all subsequent requests to protected endpoints.

### Google OAuth2 Flow

1.  The user clicks a "Login with Google" button, which directs them to `GET /auth/google/login`.
2.  The server redirects the user to Google's authentication page.
3.  After the user grants permission, Google redirects them back to the application's callback URL (`/auth/google/callback`).
4.  The server handles the callback, retrieves the user's information from Google, creates a user account if one doesn't exist, and issues a `JWT` cookie.

### Identity Linking Flow

Identity linking allows users to connect multiple OAuth providers (e.g., Google) to their existing account.

**Important:** The system does NOT automatically link OAuth providers when emails match. All linking must be explicit for security reasons.

**Manual Linking Flow:**

1.  The user must be logged in to their account.
2.  From the settings page, the user clicks "Link Google Account".
3.  The client redirects to `GET /auth/link/google?redirect_uri=/settings`.
4.  The server validates the user's authentication and generates a signed OAuth state token containing:
    -   User ID
    -   Provider (`google`)
    -   Redirect URI
    -   Nonce (for CSRF protection)
    -   Expiration timestamp (10 minutes)
5.  The user is redirected to Google's OAuth consent screen.
6.  After the user grants permission, Google redirects to `/auth/google/callback?code=xxx&state=link.xxx`.
7.  The server detects the linking state (state starts with `link.`), validates the state token, and verifies:
    -   The state signature is valid (HMAC).
    -   The state has not expired.
    -   The Google account is not already linked to a different user.
8.  If validation passes, the server links the Google identity to the authenticated user and sends a security notification email.
9.  The user is redirected back to the original `redirect_uri` with a success indicator.

**Security Considerations:**

-   **Stateless Security:** State tokens are HMAC-signed and self-contained (no database storage required).
-   **Duplicate Prevention:** The system rejects linking if the OAuth provider is already connected to another user (prevents account takeover).
-   **No Auto-Linking:** If a user logs in via OAuth with an email that matches an existing account but the provider isn't linked, the login is rejected. Users must explicitly link from their account settings.
-   **Notifications:** Users receive an email alert whenever a new identity is linked to their account.

---

## 5. Advanced Topics

### Rate Limiting

-   **Production-Aware:** Rate limiting is automatically **enabled** in `production` and **disabled** in `development`.
-   **Configuration:** Set the `APP_ENV` environment variable to `production` to enable it.
-   **Limits:**
    -   `POST /register`: 3 requests per 15 minutes.
    -   `POST /change-password`: 5 requests per 1 hour.
    -   `PUT /me`: 20 requests per 1 minute.
-   **Response:** When a rate limit is exceeded, the API returns a `429 Too Many Requests` error.

### Session Management

-   **Storage:** User sessions are stored in Redis for scalability and persistence.
-   **Endpoints:**
    -   `GET /me/sessions`: List all active sessions for the user.
    -   `DELETE /me/sessions`: Revoke all active sessions (logout everywhere).
    -   `DELETE /me/session?id=<session-id>`: Revoke a specific session.

### Role-Based Access Control (RBAC)

-   **Middleware:** The `RBAC` middleware is used to protect endpoints based on user roles.
-   **Example:**
    ```go
    router.Group(func(r chi.Router) {
        r.Use(authMiddleware.Auth)
        r.Use(authmiddleware.RBAC("admin", "root")) // Only admins and root can access

        r.Get("/admin/users", listUsersHandler)
    })
    ```
-   **Permission Matrix:**
    -   **Root:** Can change any user's role.
    -   **Admin:** Can change users to `user`, `staff`, `moderator`, or `support`. Cannot create or demote other admins.
    -   **Other roles:** Cannot change any user's role.

### Root User Management

-   **Creation:** A root user can be created at startup by setting the `ROOT_EMAIL` and `ROOT_PASSWORD` environment variables.
    ```go
    // In main.go
    if cfg.RootEmail != "" {
        authService.EnsureRootUserExists(context.Background(), cfg.RootEmail, cfg.RootPassword, "Root Admin")
    }
    ```
-   **Capabilities:**
    -   Promote users to `admin`.
    -   Demote `admin` users.
-   **Protection:** Root users are immune to deletion, deactivation, and demotion.

---

## 6. Integration Guide

### Server-Side Setup

1.  **Initialize the Auth Service:**
    ```go
    import (
        "hauslet/config"
        "hauslet/internal/auth/service"
        "hauslet/internal/auth/repository"
        "gorm.io/gorm"
    )

    func setupAuth(db *gorm.DB, redisClient *redis.Client, cfg *config.GlobalConfig) service.AuthService {
        authRepo := repository.NewAuthRepositoryImpl(db, session.NewSessionStore(redisClient))
        authService := service.NewAuthService(&cfg.Auth, authRepo, lgr.Default(), nil, redisClient, nil, "")
        return authService
    }
    ```

2.  **Mount Routes and Middleware:**
    ```go
    import (
        "hauslet/internal/auth/port"
        "github.com/go-chi/chi/v5"
    )

    func SetupRoutes(r *chi.Mux, authService service.AuthService, redisClient *redis.Client, appEnv string) {
        authHandler := authhttp.NewHTTPHandler(r.Context(), authService, lgr.Default())

        // With rate limiting (recommended for production)
        authHandler.SetupRoutesWithRateLimiting(r, redisClient, appEnv)

        // Without rate limiting (for development)
        // authHandler.SetupRoutes(r)
    }
    ```

### Client-Side Usage

#### **Extracting User Context in a Handler**

```go
import "hauslet/internal/auth/middleware"

func profileHandler(w http.ResponseWriter, r *http.Request) {
    // Get user ID and role from the request context
    userID := middleware.GetUserID(r)
    role := middleware.GetUserRole(r)

    // Fetch the full user object
    user, err := authService.GetUser(r.Context(), userID)
    if err != nil {
        // Handle error
        return
    }

    // ...
}
```

#### **JavaScript Example (Login)**

```javascript
async function login(email, password) {
  const response = await fetch('/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ user: email, passwd: password }),
    credentials: 'include' // Important for receiving the cookie
  });

  if (response.ok) {
    // Login successful
  } else {
    // Handle error
  }
}
```