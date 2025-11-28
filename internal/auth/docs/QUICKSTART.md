# Quick Start Guide

## Setup

### 1. Import Postman Collection

1. Open Postman
2. Click **Import** button
3. Select `Hauslet_Auth_API.postman_collection.json`
4. Collection will appear in your sidebar

### 2. Configure Base URL

The collection uses a variable `{{base_url}}` which defaults to `http://localhost:3000`.

To change it:
1. Click on the collection name
2. Go to **Variables** tab
3. Update `base_url` value
4. Click **Save**

---

## Testing Flow

### Scenario 1: Register and Login

```bash
# 1. Start your server
APP_ENV=development go run cmd/api/main.go

# 2. In Postman, run these requests in order:
```

**Step 1:** Register User
- Request: `POST /register`
- Body:
  ```json
  {
    "email": "test@example.com",
    "password": "password123",
    "name": "Test User"
  }
  ```
- Expected: `201 Created`

**Step 2:** Login
- Request: `POST /auth/login`
- Body (form-data):
  ```
  user: test@example.com
  passwd: password123
  ```
- Expected: `200 OK` + JWT cookie set

**Step 3:** Get Profile
- Request: `GET /me`
- Expected: `200 OK` with user profile

---

### Scenario 2: Update Profile

**Step 1:** Login (if not already)
- Request: `POST /auth/login`

**Step 2:** Update Profile
- Request: `PUT /me`
- Body:
  ```json
  {
    "name": "Updated Name"
  }
  ```
- Expected: `200 OK` with updated profile

---

### Scenario 3: Change Password

**Step 1:** Login
- Request: `POST /auth/login`

**Step 2:** Change Password
- Request: `POST /change-password`
- Body:
  ```json
  {
    "old_password": "password123",
    "new_password": "newPassword456"
  }
  ```
- Expected: `200 OK`

**Step 3:** Login with New Password
- Request: `POST /auth/login`
- Use `newPassword456`
- Expected: `200 OK`

---

### Scenario 4: Session Management

**Step 1:** Login
- Creates a new session

**Step 2:** List Sessions
- Request: `GET /me/sessions`
- Expected: Array of active sessions

**Step 3:** Revoke All Sessions
- Request: `DELETE /me/sessions`
- Expected: `200 OK`

**Step 4:** Try to Access Protected Endpoint
- Request: `GET /me`
- Expected: `401 Unauthorized` (session revoked)

---

## Common Use Cases

### 1. Testing Registration Validation

**Invalid Email:**
```json
{
  "email": "invalid-email",
  "password": "password123",
  "name": "Test"
}
```
Expected: `400 Bad Request`

**Short Password:**
```json
{
  "email": "test@example.com",
  "password": "pass",
  "name": "Test"
}
```
Expected: `400 Bad Request` with field: "password"

**Duplicate Email:**
Register same email twice.
Expected: `409 Conflict`

---

### 2. Testing Protected Endpoints Without Auth

**Without Login:**
- Request: `GET /me`
- Expected: `401 Unauthorized`

**With Login:**
- First: `POST /auth/login`
- Then: `GET /me`
- Expected: `200 OK`

---

### 3. Testing Rate Limiting (Production Only)

```bash
# Start server in production mode
APP_ENV=production go run cmd/api/main.go
```

**Test Registration Rate Limit:**
1. Register user 1: `POST /register` → Success
2. Register user 2: `POST /register` → Success
3. Register user 3: `POST /register` → Success
4. Register user 4: `POST /register` → **429 Too Many Requests**

Wait 15 minutes, then try again.

---

## Postman Tips

### 1. Automatic Cookie Handling

Postman automatically stores and sends cookies from the `/auth/login` response.

To verify:
1. Login via `POST /auth/login`
2. Go to **Cookies** (below Send button)
3. You should see `JWT` cookie
4. Protected endpoints will automatically use this cookie

### 2. Using Variables

Create environment variables for common values:

```
email = test@example.com
password = password123
user_id = <set after registration>
```

Use in requests: `{{email}}`, `{{password}}`

### 3. Running Multiple Requests

Use **Collection Runner**:
1. Click collection → Run
2. Select requests to run
3. Click **Run**
4. View results

---

## cURL Examples

### Register
```bash
curl -X POST http://localhost:3000/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "name": "Test User"
  }'
```

### Login
```bash
curl -X POST http://localhost:3000/auth/login \
  -c cookies.txt \
  -d "user=test@example.com&passwd=password123"
```

### Get Profile (with cookies)
```bash
curl -X GET http://localhost:3000/me \
  -b cookies.txt
```

### Update Profile
```bash
curl -X PUT http://localhost:3000/me \
  -b cookies.txt \
  -H "Content-Type: application/json" \
  -d '{"name": "Updated Name"}'
```

### Change Password
```bash
curl -X POST http://localhost:3000/change-password \
  -b cookies.txt \
  -H "Content-Type: application/json" \
  -d '{
    "old_password": "password123",
    "new_password": "newPassword456"
  }'
```

### Logout
```bash
curl -X GET http://localhost:3000/auth/logout \
  -b cookies.txt
```

---

## Troubleshooting

### Issue: 401 Unauthorized on Protected Endpoints

**Solution:**
1. Ensure you logged in first (`POST /auth/login`)
2. Check cookies are enabled in Postman
3. Verify JWT cookie is present (check Cookies section)

### Issue: 429 Too Many Requests

**Solution:**
1. Check if running in production mode (`APP_ENV=production`)
2. Wait for rate limit window to expire
3. Switch to development mode for testing (`APP_ENV=development`)

### Issue: 409 Conflict on Registration

**Solution:**
- Email already registered
- Use a different email address
- Or login with existing credentials

### Issue: 400 Bad Request

**Solution:**
- Check request body format
- Verify all required fields are present
- Check validation rules (password length, email format, etc.)

---

## Next Steps

1. ✅ Test all public endpoints
2. ✅ Test authentication flow
3. ✅ Test protected endpoints
4. ✅ Test error scenarios
5. ✅ Test rate limiting (production mode)
6. ✅ Integrate with your frontend application

For complete API documentation, see [API.md](./API.md)
