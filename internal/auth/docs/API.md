# Authentication API Documentation

Base URL: `http://localhost:3000`

## Table of Contents
1. [Public Endpoints](#public-endpoints)
2. [Protected Endpoints](#protected-endpoints)
3. [Error Responses](#error-responses)
4. [Rate Limiting](#rate-limiting)

---

## Public Endpoints

### 1. User Registration

**POST** `/register`

Create a new user account with email and password.

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "securePassword123",
  "name": "John Doe"
}
```

**Success Response (201):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "name": "John Doe",
  "message": "Registration successful. Please verify your email to activate your account."
}
```

**Error Response (400):**
```json
{
  "error": "Bad Request",
  "message": "Password must be at least 8 characters",
  "field": "password"
}
```

**Validation Rules:**
- Email: Required, valid email format
- Password: Required, minimum 8 characters
- Name: Required, 2-100 characters

---

### 2. Password Login

**POST** `/auth/login`

Authenticate with email and password. Returns JWT token as HTTP-only cookie.

**Request Body (form-data or x-www-form-urlencoded):**
```
user=user@example.com
passwd=securePassword123
```

**OR JSON:**
```json
{
  "user": "user@example.com",
  "passwd": "securePassword123"
}
```

**Success Response (200):**
- Sets `JWT` cookie (HTTP-only)
- Response body varies (handled by go-pkgz/auth)

**Error Response (401):**
```json
{
  "error": "Unauthorized",
  "message": "Invalid credentials"
}
```

---

### 3. Google OAuth Login

**GET** `/auth/google/login`

Initiates Google OAuth flow. Redirects to Google login.

**Usage:**
1. Navigate to: `http://localhost:3000/auth/google/login`
2. User logs in with Google
3. Redirects back to `/auth/google/callback`
4. Sets JWT cookie and redirects to app

---

### 4. Logout

**GET** `/auth/logout`

Clears authentication cookie and ends session.

**Success Response (200):**
- Clears `JWT` cookie
- Redirects or returns success message

---

## Protected Endpoints

All endpoints below require authentication via JWT cookie (obtained from login).

### 5. Get Current User Profile

**GET** `/me`

**Headers:**
```
Cookie: JWT=<your-jwt-token>
```

**Success Response (200):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "name": "John Doe",
  "role": "user"
}
```

---

### 6. Update User Profile

**PUT** `/me`

**Headers:**
```
Cookie: JWT=<your-jwt-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "name": "Jane Doe",
  "email": "newemail@example.com"
}
```

**Success Response (200):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "newemail@example.com",
  "name": "Jane Doe",
  "role": "user"
}
```

**Notes:**
- Both fields are optional
- At least one field must be provided
- Email must be valid format if provided

---

### 7. Change Password

**POST** `/change-password`

**Headers:**
```
Cookie: JWT=<your-jwt-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "old_password": "currentPassword123",
  "new_password": "newSecurePassword456"
}
```

**Success Response (200):**
```json
{
  "message": "Password changed successfully"
}
```

**Error Response (400):**
```json
{
  "error": "Bad Request",
  "message": "Current password is incorrect",
  "field": "old_password"
}
```

**Validation:**
- Old password: Required
- New password: Required, minimum 8 characters
- New password must differ from old password

---

### 8. List Authentication Methods

**GET** `/me/identities`

Returns all authentication methods linked to the account (OAuth providers, password).

**Headers:**
```
Cookie: JWT=<your-jwt-token>
```

**Success Response (200):**
```json
[
  {
    "id": "identity-uuid-1",
    "provider": "password",
    "email": "user@example.com",
    "email_verified": false
  },
  {
    "id": "identity-uuid-2",
    "provider": "google",
    "email": "user@gmail.com",
    "email_verified": true
  }
]
```

---

### 9. Unlink Authentication Method

**DELETE** `/me/identities?id=<identity-id>`

Remove an OAuth provider or password authentication from your account.

**Headers:**
```
Cookie: JWT=<your-jwt-token>
```

**Query Parameters:**
```
id=identity-uuid-to-remove
```

**Success Response (200):**
```json
{
  "message": "Identity unlinked successfully"
}
```

**Error Response (400):**
```json
{
  "error": "Bad Request",
  "message": "Cannot remove last authentication method"
}
```

**Note:** Cannot unlink the last authentication method (user must have at least one way to login).

---

### 10. List Active Sessions

**GET** `/me/sessions`

Returns all active sessions for the current user.

**Headers:**
```
Cookie: JWT=<your-jwt-token>
```

**Success Response (200):**
```json
[
  {
    "id": "session-uuid-1",
    "provider": "password",
    "created_at": "2025-11-28T10:30:00Z",
    "expires_at": "2025-11-29T10:30:00Z"
  },
  {
    "id": "session-uuid-2",
    "provider": "google",
    "created_at": "2025-11-28T12:00:00Z",
    "expires_at": "2025-11-29T12:00:00Z"
  }
]
```

---

### 11. Revoke Specific Session

**DELETE** `/me/session?id=<session-id>`

Revoke a specific session (e.g., logout from a specific device).

**Headers:**
```
Cookie: JWT=<your-jwt-token>
```

**Query Parameters:**
```
id=session-uuid-to-revoke
```

**Success Response (200):**
```json
{
  "message": "Session revoked successfully"
}
```

---

### 12. Revoke All Sessions

**DELETE** `/me/sessions`

Logout from all devices (useful for "Logout Everywhere" feature).

**Headers:**
```
Cookie: JWT=<your-jwt-token>
```

**Success Response (200):**
```json
{
  "message": "All sessions revoked successfully"
}
```

**Note:** This will invalidate all active sessions including the current one.

---

## Error Responses

### Standard Error Format

```json
{
  "error": "HTTP Status Text",
  "message": "Detailed error message",
  "field": "field_name (optional)"
}
```

### Common HTTP Status Codes

| Code | Meaning | Description |
|------|---------|-------------|
| 200 | OK | Request succeeded |
| 201 | Created | Resource created successfully |
| 400 | Bad Request | Invalid request data or validation error |
| 401 | Unauthorized | Missing or invalid authentication |
| 404 | Not Found | Resource not found |
| 409 | Conflict | Resource already exists (e.g., duplicate email) |
| 429 | Too Many Requests | Rate limit exceeded (production only) |
| 500 | Internal Server Error | Server error |

---

## Rate Limiting

**Production Only** - Rate limiting is disabled in development.

| Endpoint | Limit | Window | Status Code |
|----------|-------|--------|-------------|
| `POST /register` | 3 requests | 15 minutes | 429 |
| `POST /change-password` | 5 requests | 1 hour | 429 |
| `PUT /me` | 20 requests | 1 minute | 429 |

**Rate Limit Response (429):**
```json
{
  "message": "Rate limit exceeded. Try again in 14m30s",
  "title": "Too Many Requests"
}
```

---

## Authentication Flow

### Registration → Login Flow
```
1. POST /register (create account)
   ↓
2. POST /auth/login (get JWT cookie)
   ↓
3. Access protected endpoints with JWT cookie
```

### OAuth Flow
```
1. GET /auth/google/login (redirect to Google)
   ↓
2. User authenticates with Google
   ↓
3. GET /auth/google/callback (returns with JWT cookie)
   ↓
4. Access protected endpoints with JWT cookie
```

---

## JWT Token

- **Storage:** HTTP-only cookie named `JWT`
- **Duration:** 5 minutes (token), 24 hours (cookie)
- **Renewal:** Automatically renewed by go-pkgz/auth
- **Validation:** Checked on every protected endpoint

---

## Security Notes

1. **Password Requirements:** Minimum 8 characters
2. **JWT Cookies:** HTTP-only, secure in production
3. **Rate Limiting:** Enabled in production to prevent abuse
4. **Session Management:** Redis-backed for distributed systems
5. **Multi-Auth:** Users can link multiple OAuth providers
