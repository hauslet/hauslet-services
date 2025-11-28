# Hauslet Authentication System Documentation

This guide covers the implementation, usage, and management of the Hauslet authentication system. It includes setup instructions, client-side integration, server-side handlers, RBAC (Role-Based Access Control) policies, and Root user management.

-----

## 1\. System Architecture & Roles

The system implements a hierarchical role structure with specific permission boundaries.

### Role Definitions

Defined in `schema/gorm.go`:

| Role | Description |
| :--- | :--- |
| **root** | Superuser. Cannot be deleted or demoted. Full system access. |
| **admin** | Administrative access. Can manage users/content. |
| **moderator** | Content moderation capabilities. |
| **staff** | Internal staff access. |
| **support** | Customer support capabilities. |
| **user** | Default role for standard registered users. |

### Permission Matrix (Role Changes)

The `ChangeUserRole` service enforces strict permission logic:

| Actor Role | Can Change Target To | Cannot Change Target To |
| :--- | :--- | :--- |
| **Root** | Any role | None |
| **Admin** | user, staff, moderator, support | admin, root |
| **Other** | None | All |

> **Safeguards:**
>
>   * Admins cannot modify other Admins.
>   * No one can modify a Root user's role.
>   * Users cannot change their own roles (prevents self-promotion).

-----

## 2\. Server-Side Setup

### Initialize Auth Service

Bootstrapping the service with GORM, Redis, and Config.

```go
import (
	"hauslet/config"
	"hauslet/internal/auth/service"
	"hauslet/internal/auth/repository"
	"hauslet/platform/redis"
	"gorm.io/gorm"
)

func setupAuth(db *gorm.DB, redisClient redis.RedisClient, cfg *config.AuthConfig) *auth.Service {
	// 1. Create session store (Redis)
	sessionStore := session.NewSessionStore(redisClient)

	// 2. Create auth repository
	authRepo := repository.NewAuthRepositoryImpl(db, sessionStore)

	// 3. Create auth service
	authService := service.NewAuthService(cfg, authRepo)

	// 4. Return the service interface
	return authService.OAuthService()
}
```

### Mount Routes & Middleware

Setting up the Chi router with protected and public groups.

```go
func setupRoutes(authSvc *auth.Service) http.Handler {
	router := chi.NewRouter()
	authMiddleware := authSvc.Middleware()

	// -- Public Routes --
	// Mounts /auth/google/login, /auth/password/login, etc.
	authRoutes, avatarRoutes := authSvc.Handlers()
	router.Mount("/auth", authRoutes)
	router.Mount("/avatar", avatarRoutes)
	router.Post("/register", registerHandler)

	// -- Protected Routes (Requires Login) --
	router.Group(func(r chi.Router) {
		r.Use(authMiddleware.Auth) // JWT Validation

		r.Get("/profile", profileHandler)
		r.Get("/dashboard", dashboardHandler)
	})

	// -- RBAC: Admin Routes --
	router.Group(func(r chi.Router) {
		r.Use(authMiddleware.Auth)
		r.Use(authmiddleware.RBACMiddleware("admin", "root"))

		r.Get("/admin/users", listUsersHandler)
		r.Delete("/admin/users/{id}", deleteUserHandler)
		r.Put("/users/{userID}/role", changeUserRoleHandler) // See Section 5
	})

	return router
}
```

-----

## 3\. Authentication Flows

### OAuth Flow (Google)

1.  **Trigger:** User visits `/auth/google/login`.
2.  **Consent:** Redirected to Google.
3.  **Callback:** Google redirects to `/auth/google/callback`.
4.  **Enrichment:** System links identity, creates Redis session, and generates JWT.
5.  **Result:** JWT set in HTTP-only cookie.

### Password Flow

1.  **Trigger:** POST to `/auth/password/login`.
2.  **Validation:** `CredCheckerFunc` verifies bcrypt hash.
3.  **Enrichment:** Same as OAuth (Redis session + JWT).
4.  **Result:** JWT set in HTTP-only cookie.

-----

## 4\. Client-Side Usage Examples

### Registration (Password)

```go
// Handler
func registerHandler(w http.ResponseWriter, r *http.Request) {
    // Decode body...
    // Create password user
    user, err := authService.CreatePasswordUser(r.Context(), req.Email, req.Password, req.Name)
    // Handle response...
}
```

### Logging In

**HTML/OAuth:**

```html
<a href="/auth/google/login">Login with Google</a>
```

**JavaScript/Fetch (Password):**

```javascript
const response = await fetch('/auth/password/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ user: 'john@example.com', passwd: 'mypassword123' }),
  credentials: 'include' // Important: receives cookie
});
```

**Curl:**

```bash
curl -X POST http://localhost:8080/auth/password/login \
  -H "Content-Type: application/json" \
  -d '{"user":"john@example.com","passwd":"mypassword123"}' \
  -c cookies.txt
```

-----

## 5\. Implementation Guide: Protected Handlers

### Extracting User Context

How to get user data inside a protected handler.

```go
func profileHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Get User ID from Context (set by middleware)
	userID := authmiddleware.GetUserID(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Get User Role
	role := authmiddleware.GetUserRole(r)

	// 3. Fetch Full User Object
	user, _ := authService.GetUser(r.Context(), userID)

    // ... response logic
}
```

### Session Management

Manage active devices and sessions.

```go
// Revoke all sessions (Logout all devices)
func logoutAllDevicesHandler(w http.ResponseWriter, r *http.Request) {
	userID := authmiddleware.GetUserID(r)
	err := authService.RevokeAllUserSessions(r.Context(), userID)
    // ...
}
```

-----

## 6\. Role Management (RBAC)

This section details how to programmatically change user roles securely.

### Service Method: `ChangeUserRole`

```go
// Method Signature
ChangeUserRole(ctx context.Context, actorID, targetUserID string, newRole domain.UserRole) error
```

### Handler Implementation

```go
// PUT /api/users/:userID/role
func changeUserRoleHandler(w http.ResponseWriter, r *http.Request) {
	actorID := middleware.GetUserID(r)
	targetUserID := chi.URLParam(r, "userID")
    
    // Parse role from body...
    var req struct { Role string `json:"role"` }
    
    // Call Service
	err := authService.ChangeUserRole(r.Context(), actorID, targetUserID, domain.UserRole(req.Role))

	if err != nil {
        // Map errors to status codes (403 for permission issues, 404 for not found)
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
    // Success...
}
```

### Usage Examples (Client)

```bash
# Admin promotes user to staff (Allowed)
curl -X PUT /api/users/user-123/role -d '{"role": "staff"}' ...

# Admin tries to promote to admin (Forbidden - requires Root)
curl -X PUT /api/users/user-123/role -d '{"role": "admin"}' ...
# Response: 403 "only root users can promote to admin"
```

-----

## 7\. Root User Management

The **Root** user is the bootstrap administrator for the system.

### Creating a Root User

**Method A: Idempotent Startup (Recommended)**

```go
// In main.go
if cfg.RootEmail != "" {
    authService.EnsureRootUserExists(context.Background(), cfg.RootEmail, cfg.RootPassword, "Root Admin")
}
```

**Method B: CLI Command**

```go
// cmd/admin.go
var createRootCmd = &cobra.Command{
    Use: "create-root",
    Run: func(cmd *cobra.Command, args []string) {
        // ... setup service ...
        authService.CreateRootUser(ctx, email, password, name)
    },
}
```

### Root Capabilities & Protections

Root users have special privileges hardcoded into the service logic:

1.  **Promote to Admin:** Only Root can promote a user to `RoleAdmin`.
2.  **Demote Admin:** Only Root can demote an `RoleAdmin`.
3.  **Immunity:**
      * Cannot be deactivated (`DeactivateUser` fails).
      * Cannot be deleted (`DeleteUser` fails).
      * Cannot be demoted via standard update (`UpdateUser` fails).

### Managing Admins (Root Only)

```go
func promoteToAdmin(ctx context.Context, authService service.AuthService, rootID, targetUserID string) {
    // This calls internal logic to verify rootID actually belongs to a root user
	err := authService.PromoteToAdmin(ctx, rootID, targetUserID)
}
```

-----

## 8\. Database Schema Reference

Key tables involved in this system:

  * `users`: Stores profile, `role` (enum), and `is_active` status.
  * `user_identities`: Stores auth provider data (Password hash, Google sub, etc.).
  * `sessions` (Redis): Stores active JWT validity and expiry.

### Migration SQL (Manual Root Creation)

If CLI access is unavailable, you can seed a root user via SQL:

```sql
INSERT INTO users (id, name, primary_email, role, is_active, created_at)
VALUES (gen_random_uuid(), 'Root', 'root@local', 'root', true, NOW());
-- Note: Requires matching user_identities entry for password login
```

