# Verification System Implementation Plan v2.0

**Status:** Design  
**Owner:** Backend Team  
**Last Updated:** January 1, 2026  
**Scope:** Server-side implementation only (micro-frontend client is out of scope)

---

## Table of Contents
1. [Executive Summary](#executive-summary)
2. [Architecture Overview](#architecture-overview)
3. [Domain Model](#domain-model)
4. [Workflow Specifications](#workflow-specifications)
5. [Module Structure](#module-structure)
6. [Database Schema](#database-schema)
7. [API Specification](#api-specification)
8. [Queue Integration](#queue-integration)
9. [Security & Compliance](#security--compliance)
10. [Integration Points](#integration-points)
11. [Testing Strategy](#testing-strategy)
12. [Deployment Plan](#deployment-plan)
13. [Open Questions](#open-questions)

---

## Executive Summary

This document defines the backend architecture for **Hauslet's Identity & Phone Verification System**, supporting:
- **ID Verification** via a micro-frontend ("KYC Handshake" pattern) hosted on `verification.hauslet.com`
- **Phone Number Verification** via OTP (SMS)
- **Liveness Detection** (face capture & AI validation)

The system is designed as a **dedicated module** (`internal/modules/verification`) following Hauslet's **Hexagonal Architecture** principles. It integrates with existing `profile`, `auth`, and `authorization` modules while maintaining clear domain boundaries.

### Key Design Principles
1. **Stateless Token Handshake:** No shared cookies/localStorage between domains
2. **Service Layer Authority:** All business logic in service layer, thin adapters
3. **Async Processing:** Heavy operations (AI validation, 3rd-party KYC APIs) via Cloud Tasks
4. **Audit Trail:** Every state transition logged in `verification_events`
5. **Single-Use Tokens:** Short-lived (10 min), SHA-256 hashed, invalidated on first use

---

## Architecture Overview

### The "KYC Handshake" Flow

```
┌─────────────────┐         ┌──────────────────┐         ┌─────────────────────┐
│  Main Web App   │         │  Hauslet Backend │         │ Verification Micro- │
│ (hauslet.com)   │         │    (Go API)      │         │   Frontend (React)  │
│                 │         │                  │         │ verification.       │
│                 │         │                  │         │   hauslet.com       │
└────────┬────────┘         └────────┬─────────┘         └──────────┬──────────┘
         │                           │                              │
         │ 1. User clicks "Verify ID"│                              │
         │ ──────────────────────────>                              │
         │                           │                              │
         │ 2. POST /verification/sessions                           │
         │    (Auth: Bearer JWT)     │                              │
         │ ────────────────────────> │                              │
         │                           │                              │
         │ 3. Generate token + session                              │
         │    (Store SHA256(token))  │                              │
         │                           │                              │
         │ 4. Return token + URL     │                              │
         │ <──────────────────────── │                              │
         │                           │                              │
         │ 5. Redirect user to verification.hauslet.com?token=XYZ   │
         │ ─────────────────────────────────────────────────────────>
         │                           │                              │
         │                           │ 6. GET /verification/sessions/validate?token=XYZ
         │                           │ <────────────────────────────│
         │                           │                              │
         │                           │ 7. Validate token (hash match)
         │                           │                              │
         │                           │ 8. Return session metadata   │
         │                           │ ─────────────────────────────>
         │                           │                              │
         │                           │                   9. User captures liveness
         │                           │                      (face detection, blink)
         │                           │                              │
         │                           │ 10. POST /verification/sessions/{id}/submit
         │                           │     (token + base64 image)   │
         │                           │ <────────────────────────────│
         │                           │                              │
         │                           │ 11. Store evidence (R2)      │
         │                           │     Enqueue job (Cloud Tasks)│
         │                           │                              │
         │                           │ 12. Return success           │
         │                           │ ─────────────────────────────>
         │                           │                              │
         │ 13. Redirect user back with status=pending               │
         │ <─────────────────────────────────────────────────────────
         │                           │                              │
         │                      14. Worker processes job            │
         │                          (AI validation, Dojah API)      │
         │                           │                              │
         │                      15. Update session status           │
         │                          Update profile.IDVerified=true  │
         │                           │                              │
```

### Components

| Component | Technology | Responsibility |
|-----------|-----------|----------------|
| **Main App** | Next.js/React | Initiate verification, display status |
| **API Server** | Go (Chi + gqlgen) | Token generation, session management, evidence storage |
| **Worker** | Go | Async processing (AI validation, 3rd-party KYC APIs) |
| **Verification Frontend** | React (Vite) on Cloudflare Pages | Liveness capture, evidence submission |
| **Queue** | Google Cloud Tasks | Async job processing |
| **Storage** | Cloudflare R2 | Store selfies & ID documents |
| **SMS Provider** | Termii / Twilio | OTP delivery |
| **KYC Provider** | Dojah / Smile ID | ID verification & BVN/NIN lookup |

---

## Domain Model

### Core Entities

#### `VerificationSession`
The primary aggregate representing a verification attempt.

```go
type VerificationSession struct {
    ID            uuid.UUID
    UserID        uuid.UUID
    Method        VerificationMethod  // Enum: id_liveness, id_document, phone_otp
    Status        VerificationStatus  // Enum: pending, in_review, verified, failed, expired, cancelled
    TokenHash     string              // SHA256 hash of plaintext token
    ExpiresAt     time.Time           // Token expiration (10 minutes from creation)
    ReturnURL     string              // Where to redirect after completion
    Provider      *string             // Optional: dojah, smile_id, etc.
    Attempts      int                 // Number of submission attempts
    EvidenceURL   *string             // R2 object key for uploaded evidence
    ProviderRef   *string             // External provider's verification ID
    FailureReason *string             // Error message if failed
    CompletedAt   *time.Time
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

// Enums
type VerificationMethod string
const (
    MethodIDLiveness   VerificationMethod = "id_liveness"
    MethodIDDocument   VerificationMethod = "id_document"
    MethodPhoneOTP     VerificationMethod = "phone_otp"
)

type VerificationStatus string
const (
    StatusPending    VerificationStatus = "pending"
    StatusInReview   VerificationStatus = "in_review"
    StatusVerified   VerificationStatus = "verified"
    StatusFailed     VerificationStatus = "failed"
    StatusExpired    VerificationStatus = "expired"
    StatusCancelled  VerificationStatus = "cancelled"
)
```

#### `VerificationEvent`
Immutable audit log for all session state changes.

```go
type VerificationEvent struct {
    ID        uuid.UUID
    SessionID uuid.UUID
    EventType string                 // session_created, evidence_submitted, provider_called, status_updated, etc.
    Payload   map[string]interface{} // JSON metadata
    CreatedAt time.Time
}
```

#### `PhoneVerificationAttempt`
Tracks OTP verification attempts (separate from sessions for simplicity).

```go
type PhoneVerificationAttempt struct {
    ID             uuid.UUID
    UserID         uuid.UUID
    PhoneNumber    string
    OTPHash        string       // SHA256 hash of plaintext OTP
    ExpiresAt      time.Time    // 5 minutes from creation
    Verified       bool
    AttemptCount   int
    VerifiedAt     *time.Time
    CreatedAt      time.Time
}
```

---

## Workflow Specifications

### Workflow 1: ID Verification with Liveness Check

#### Step 1: Session Creation (Main App)
**Endpoint:** `POST /verification/sessions`  
**Auth:** Required (JWT)  
**Request Body:**
```json
{
  "method": "id_liveness",
  "return_url": "https://hauslet.com/dashboard?tab=verification"
}
```

**Backend Logic:**
1. Validate user authentication
2. Check if user already has a verified session (prevent duplicates)
3. Generate cryptographically secure token (32 bytes)
4. Hash token with SHA-256, store hash in DB
5. Create `VerificationSession` record:
   - `status=pending`
   - `expires_at=now + 10 minutes`
   - `token_hash=SHA256(token)`
6. Log event: `session_created`
7. Return response

**Response (200):**
```json
{
  "session_id": "123e4567-e89b-12d3-a456-426614174000",
  "token": "abc123def456...",  // Plaintext (never stored)
  "verification_url": "https://verification.hauslet.com?token=abc123def456...",
  "expires_at": "2026-01-01T12:10:00Z",
  "return_url": "https://hauslet.com/dashboard?tab=verification"
}
```

#### Step 2: Token Validation (Verification Frontend)
**Endpoint:** `GET /verification/sessions/validate?token=abc123...`  
**Auth:** Public (token-based)

**Backend Logic:**
1. Hash incoming token with SHA-256
2. Query DB for `verification_sessions` where `token_hash=hashed_token` AND `status=pending` AND `expires_at > now()`
3. If not found → return `401 Unauthorized` (invalid/expired token)
4. If found → mark token as used (update `status=in_review` or set flag)
5. Log event: `token_validated`
6. Return session metadata

**Response (200):**
```json
{
  "session_id": "123e4567-e89b-12d3-a456-426614174000",
  "method": "id_liveness",
  "status": "pending",
  "user_id": "user123",
  "allowed_actions": ["submit_evidence"]
}
```

#### Step 3: Evidence Submission (Verification Frontend)
**Endpoint:** `POST /verification/sessions/{session_id}/submit`  
**Auth:** Token required (in request body)  
**Request Body:**
```json
{
  "token": "abc123def456...",
  "evidence": {
    "type": "selfie",
    "image": "data:image/jpeg;base64,/9j/4AAQSkZJRg...",
    "metadata": {
      "face_detected": true,
      "blink_detected": true,
      "device_info": "Chrome 120.0 / Android 13"
    }
  }
}
```

**Backend Logic:**
1. Validate token (hash match + session_id match)
2. Decode base64 image
3. Validate image format & size (max 5MB)
4. Upload image to R2:
   - Bucket: `hauslet-verification-evidence`
   - Path: `{user_id}/{session_id}/selfie-{timestamp}.jpg`
   - Permissions: Private (no public access)
5. Update session:
   - `evidence_url=R2_object_key`
   - `status=in_review`
   - `attempts++`
6. Log event: `evidence_submitted`
7. Enqueue job to Cloud Tasks:
   - Queue: `verification-processing`
   - Job type: `VerifyLivenessJob`
   - Payload: `{session_id, evidence_url, user_id}`
8. Return success

**Response (200):**
```json
{
  "status": "in_review",
  "message": "Evidence submitted successfully. Verification in progress."
}
```

#### Step 4: Async Processing (Worker)
**Queue:** `verification-processing`  
**Job:** `VerifyLivenessJob`

**Worker Logic:**
1. Pull job from Cloud Tasks
2. Download image from R2
3. **AI Validation (Gemini/Anthropic):**
   - Prompt: "Analyze this image. Is it a real human face (not a photo of a photo or screen)? Is the person alive (eyes open, natural pose)?"
   - If AI rejects → mark session as `failed`, update profile (keep `IDVerified=false`)
4. **Optional: Call 3rd-party KYC (Dojah):**
   - If user uploaded ID document separately, send both to Dojah API
   - Dojah returns: `{verified: true, nin: "12345678901", name: "John Doe"}`
5. **Update Session:**
   - If verified: `status=verified`, `completed_at=now()`, store `provider_ref`
   - If failed: `status=failed`, `failure_reason="AI detected fake image"`
6. **Update Profile (via Profile Service):**
   ```go
   profileSvc.SetVerificationStatus(ctx, userID, "identity", true, &now)
   // This sets: IDVerified=true, VerificationLevel="identity", VerificationDate=now
   ```
7. **Add Badge (optional):**
   ```go
   profileSvc.AddBadge(ctx, userID, profile.BadgeIdentityVerified)
   ```
8. Log event: `verification_completed` or `verification_failed`
9. **Send Notification (optional):**
   - Email: "Your ID verification is complete!"
   - Push notification (if supported)

#### Step 5: Redirect & Status Check (Main App)
After verification frontend redirects user back to `return_url`, main app can query status:

**Endpoint:** `GET /verification/sessions/{session_id}`  
**Auth:** Required (JWT) - only session owner can view

**Response (200):**
```json
{
  "session_id": "123e4567-e89b-12d3-a456-426614174000",
  "status": "verified",
  "method": "id_liveness",
  "completed_at": "2026-01-01T12:05:30Z"
}
```

---

### Workflow 2: Phone Number Verification (OTP)

#### Step 1: Start Verification
**Endpoint:** `POST /verification/phone/start`  
**Auth:** Required (JWT)  
**Request Body:**
```json
{
  "phone_number": "+2348012345678"
}
```

**Backend Logic:**
1. Validate phone number format (E.164)
2. Check rate limit: max 3 OTP requests per phone number per hour (Redis)
3. Generate 6-digit OTP
4. Hash OTP with SHA-256
5. Create `PhoneVerificationAttempt` record:
   - `otp_hash=SHA256(otp)`
   - `expires_at=now + 5 minutes`
6. Send SMS via provider (Termii/Twilio):
   - Message: "Your Hauslet verification code is: {otp}. Valid for 5 minutes."
7. Enqueue SMS job to Cloud Tasks (async, non-blocking)
8. Log event: `otp_sent`

**Response (200):**
```json
{
  "phone_number": "+234801*****78",  // Masked
  "expires_at": "2026-01-01T12:05:00Z",
  "retry_after": 60  // Seconds until next OTP allowed
}
```

#### Step 2: Verify OTP
**Endpoint:** `POST /verification/phone/verify`  
**Auth:** Required (JWT)  
**Request Body:**
```json
{
  "phone_number": "+2348012345678",
  "otp": "123456"
}
```

**Backend Logic:**
1. Hash incoming OTP
2. Query `PhoneVerificationAttempt` where:
   - `user_id=current_user`
   - `phone_number=provided_number`
   - `verified=false`
   - `expires_at > now()`
3. Compare hashed OTP
4. Increment `attempt_count` (max 3 attempts before expiry)
5. If match:
   - Update attempt: `verified=true`, `verified_at=now()`
   - Update profile: `PhoneVerified=true`
   - Add badge: `phone_verified`
   - Log event: `phone_verified`
6. If failed after 3 attempts → mark expired, require new OTP

**Response (200):**
```json
{
  "verified": true,
  "message": "Phone number verified successfully"
}
```

---

## Module Structure

Following Hauslet's hexagonal architecture:

```
internal/modules/verification/
├── domain/
│   ├── session.go           # VerificationSession entity + business rules
│   ├── event.go             # VerificationEvent entity
│   ├── phone.go             # PhoneVerificationAttempt entity
│   ├── enums.go             # Method, Status enums
│   └── errors.go            # Domain-specific errors
│
├── repository/
│   ├── schema/
│   │   └── gorm.go          # GORM models (DB layer)
│   ├── interface.go         # Repository contract
│   ├── session_repo.go      # CRUD for sessions
│   └── event_repo.go        # CRUD for events
│
├── service/
│   ├── interface.go         # Service contract
│   ├── service.go           # Core business logic
│   ├── token.go             # Token generation/validation
│   ├── session_lifecycle.go # Session state machine
│   └── phone_verification.go # OTP logic
│
├── port/
│   ├── http/
│   │   ├── routes.go        # Chi router setup
│   │   ├── session_handlers.go
│   │   ├── phone_handlers.go
│   │   └── webhook_handlers.go
│   │
│   └── worker/
│       ├── liveness_handler.go   # Process liveness verification
│       └── kyc_handler.go        # Call Dojah/Smile ID
│
├── templates/
│   └── verification_complete.html  # Email template (optional)
│
└── docs/
    └── README.md
```

---

## Database Schema

### Migration: `010_create_verification_tables.sql`

```sql
-- Verification Sessions Table
CREATE TABLE verification_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    method VARCHAR(50) NOT NULL CHECK (method IN ('id_liveness', 'id_document', 'phone_otp')),
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_review', 'verified', 'failed', 'expired', 'cancelled')),
    token_hash VARCHAR(64) NOT NULL UNIQUE,  -- SHA256 hash (hex string = 64 chars)
    expires_at TIMESTAMP NOT NULL,
    return_url TEXT NOT NULL,
    provider VARCHAR(100),                   -- e.g., 'dojah', 'smile_id'
    attempts INT NOT NULL DEFAULT 0,
    evidence_url TEXT,                       -- R2 object key
    provider_ref VARCHAR(255),               -- External provider's ID
    failure_reason TEXT,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_verification_sessions_user_id ON verification_sessions(user_id);
CREATE INDEX idx_verification_sessions_status ON verification_sessions(status);
CREATE INDEX idx_verification_sessions_expires_at ON verification_sessions(expires_at);
CREATE INDEX idx_verification_sessions_token_hash ON verification_sessions(token_hash);

-- Verification Events Table (Audit Log)
CREATE TABLE verification_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES verification_sessions(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_verification_events_session_id ON verification_events(session_id);
CREATE INDEX idx_verification_events_created_at ON verification_events(created_at);

-- Phone Verification Attempts Table
CREATE TABLE phone_verification_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    phone_number VARCHAR(20) NOT NULL,
    otp_hash VARCHAR(64) NOT NULL,           -- SHA256 hash
    expires_at TIMESTAMP NOT NULL,
    verified BOOL NOT NULL DEFAULT false,
    attempt_count INT NOT NULL DEFAULT 0,
    verified_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_phone_verification_user_id ON phone_verification_attempts(user_id);
CREATE INDEX idx_phone_verification_phone ON phone_verification_attempts(phone_number);
CREATE INDEX idx_phone_verification_expires_at ON phone_verification_attempts(expires_at);
```

---

## API Specification

### REST Endpoints

#### 1. Create Verification Session
```http
POST /verification/sessions
Authorization: Bearer {jwt_token}
Content-Type: application/json

{
  "method": "id_liveness",
  "return_url": "https://hauslet.com/dashboard"
}

Response 201:
{
  "session_id": "uuid",
  "token": "plaintext_token",
  "verification_url": "https://verification.hauslet.com?token=...",
  "expires_at": "2026-01-01T12:10:00Z",
  "return_url": "https://hauslet.com/dashboard"
}

Response 400: Invalid method
Response 429: Too many verification attempts
```

#### 2. Validate Verification Token
```http
GET /verification/sessions/validate?token={token}
(No auth required - token is the auth mechanism)

Response 200:
{
  "session_id": "uuid",
  "method": "id_liveness",
  "status": "pending",
  "user_id": "uuid",
  "allowed_actions": ["submit_evidence"]
}

Response 401: Invalid or expired token
```

#### 3. Submit Evidence
```http
POST /verification/sessions/{session_id}/submit
Content-Type: application/json

{
  "token": "plaintext_token",
  "evidence": {
    "type": "selfie",
    "image": "base64_encoded_string",
    "metadata": {
      "face_detected": true,
      "blink_detected": true
    }
  }
}

Response 200:
{
  "status": "in_review",
  "message": "Evidence submitted successfully"
}

Response 400: Invalid image format
Response 401: Invalid token
Response 409: Session already completed
Response 413: Image too large (>5MB)
```

#### 4. Get Session Status
```http
GET /verification/sessions/{session_id}
Authorization: Bearer {jwt_token}

Response 200:
{
  "session_id": "uuid",
  "method": "id_liveness",
  "status": "verified",
  "created_at": "2026-01-01T12:00:00Z",
  "completed_at": "2026-01-01T12:05:30Z",
  "failure_reason": null
}

Response 404: Session not found
Response 403: Not authorized to view this session
```

#### 5. Start Phone Verification
```http
POST /verification/phone/start
Authorization: Bearer {jwt_token}
Content-Type: application/json

{
  "phone_number": "+2348012345678"
}

Response 200:
{
  "phone_number": "+234801*****78",
  "expires_at": "2026-01-01T12:05:00Z",
  "retry_after": 60
}

Response 429: Rate limit exceeded
```

#### 6. Verify Phone OTP
```http
POST /verification/phone/verify
Authorization: Bearer {jwt_token}
Content-Type: application/json

{
  "phone_number": "+2348012345678",
  "otp": "123456"
}

Response 200:
{
  "verified": true,
  "message": "Phone number verified successfully"
}

Response 400: Invalid OTP
Response 410: OTP expired
Response 429: Too many attempts
```

#### 7. Provider Webhook (Optional)
```http
POST /webhooks/verification/dojah
Content-Type: application/json
X-Dojah-Signature: {hmac_signature}

{
  "verification_id": "external_id",
  "status": "verified",
  "nin": "12345678901",
  "bvn": "22334455667",
  "name": "John Doe"
}

Response 200: OK
Response 401: Invalid signature
```

### Rate Limits (Production)

| Endpoint | Limit | Window |
|----------|-------|--------|
| `POST /verification/sessions` | 5 requests | 1 hour |
| `POST /verification/phone/start` | 3 requests | 1 hour |
| `POST /verification/phone/verify` | 10 requests | 15 minutes |
| `POST /verification/sessions/{id}/submit` | 3 requests | 10 minutes |

---

## Queue Integration

### Cloud Tasks Queues

#### 1. `verification-processing` Queue
Processes verification evidence and calls external providers.

**Job Type:** `VerifyLivenessJob`

```go
type VerifyLivenessJob struct {
    BaseJob
    SessionID   string `json:"session_id"`
    EvidenceURL string `json:"evidence_url"`
    UserID      string `json:"user_id"`
}

func (j VerifyLivenessJob) Type() string {
    return "verification:liveness"
}
```

**Worker Handler Logic:**
```go
func (h *VerificationWorkerHandler) HandleVerifyLiveness(ctx context.Context, job VerifyLivenessJob) error {
    // 1. Download image from R2
    image, err := h.storageClient.Download(ctx, job.EvidenceURL)
    
    // 2. Call AI provider (Gemini/Anthropic)
    aiResult, err := h.aiClient.ValidateLiveness(ctx, image)
    if !aiResult.IsLive {
        return h.failSession(ctx, job.SessionID, "AI detected non-live image")
    }
    
    // 3. Optional: Call KYC provider (Dojah)
    kycResult, err := h.kycProvider.VerifyIdentity(ctx, image)
    if err != nil {
        return fmt.Errorf("KYC provider error: %w", err)
    }
    
    // 4. Update session
    if kycResult.Verified {
        err = h.verificationService.CompleteSession(ctx, job.SessionID, VerificationResult{
            Verified:    true,
            ProviderRef: kycResult.ID,
        })
    } else {
        err = h.verificationService.FailSession(ctx, job.SessionID, kycResult.Reason)
    }
    
    // 5. Update profile
    if kycResult.Verified {
        err = h.profileService.SetVerificationStatus(ctx, job.UserID, "identity", true, timePtr(time.Now()))
    }
    
    return err
}
```

#### 2. `verification-notifications` Queue
Sends SMS and email notifications.

**Job Type:** `SendVerificationSMSJob`

```go
type SendVerificationSMSJob struct {
    BaseJob
    PhoneNumber string `json:"phone_number"`
    OTP         string `json:"otp"`
    UserID      string `json:"user_id"`
}

func (j SendVerificationSMSJob) Type() string {
    return "verification:send_sms"
}
```

---

## Security & Compliance

### Token Security
1. **Generation:** Use `crypto/rand` for cryptographically secure tokens (32 bytes minimum)
2. **Storage:** Never store plaintext tokens - only SHA-256 hashes
3. **Single Use:** Mark token as used after first validation (set `token_hash=NULL` or status flag)
4. **Short TTL:** 10 minutes for ID verification, 5 minutes for phone OTP
5. **Timing Attack Prevention:** Use constant-time comparison (`subtle.ConstantTimeCompare`) for token validation

### Data Protection
1. **Evidence Encryption:** Store sensitive images in R2 with server-side encryption
2. **Access Control:** Evidence URLs signed with time-limited presigned URLs
3. **Audit Trail:** Log every state transition in `verification_events`
4. **Retention Policy:** Delete verification evidence after 90 days (GDPR compliance)
5. **PII Handling:** Never log full phone numbers or tokens in plain text

### Rate Limiting (Redis)
```go
// Example rate limit key structure
key := fmt.Sprintf("ratelimit:verification:session:user:%s", userID)
ttl := 1 * time.Hour
limit := 5

// Increment counter
count, err := redisClient.Incr(ctx, key).Result()
if count == 1 {
    redisClient.Expire(ctx, key, ttl)
}
if count > limit {
    return ErrRateLimitExceeded
}
```

### Provider Webhook Validation
```go
func validateDojahWebhook(req *http.Request, secret string) error {
    signature := req.Header.Get("X-Dojah-Signature")
    body, _ := io.ReadAll(req.Body)
    
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    expectedMAC := hex.EncodeToString(mac.Sum(nil))
    
    if !hmac.Equal([]byte(signature), []byte(expectedMAC)) {
        return errors.New("invalid webhook signature")
    }
    return nil
}
```

---

## Integration Points

### 1. Profile Module
**Interface Contract:**
```go
type ProfileService interface {
    // Update verification status and date
    SetVerificationStatus(ctx context.Context, userID string, level string, verified bool, date *time.Time) error
    
    // Add badge to user profile
    AddBadge(ctx context.Context, userID string, badge profile.Badge) error
    
    // Get user profile (to check existing verification)
    GetProfileByUserID(ctx context.Context, userID string) (*profile.Profile, error)
}
```

**Usage:**
```go
// After successful ID verification
err := profileService.SetVerificationStatus(ctx, userID, "identity", true, &now)
if err == nil {
    profileService.AddBadge(ctx, userID, profile.BadgeIdentityVerified)
}
```

### 2. Authorization Module (Supply Gate)
No changes required. Supply gate already checks `profile.IDVerified`:

```go
// In authorization/supply_gate.go
if !userProfile.IDVerified {
    return authorization.ErrIDVerificationRequired
}
```

**Ensure:** Sessions with `status=in_review` or `status=pending` do NOT set `profile.IDVerified=true`.

### 3. Platform: Storage (R2)
**Interface:**
```go
type StorageClient interface {
    Upload(ctx context.Context, bucket, key string, data []byte) error
    Download(ctx context.Context, bucket, key string) ([]byte, error)
    GeneratePresignedURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error)
    Delete(ctx context.Context, bucket, key string) error
}
```

**Bucket:** `hauslet-verification-evidence`  
**Path Pattern:** `{user_id}/{session_id}/{type}-{timestamp}.{ext}`

### 4. Platform: AI Provider
**Interface:**
```go
type AIProvider interface {
    ValidateLiveness(ctx context.Context, image []byte) (*LivenessResult, error)
}

type LivenessResult struct {
    IsLive      bool
    Confidence  float64
    FaceDetected bool
    Reason      string
}
```

**Implementation:** Use existing `internal/platform/ai` with Gemini (primary) and Anthropic (fallback).

### 5. Platform: SMS Provider
**New Interface:** `internal/platform/notifications/sms.go`

```go
type SMSProvider interface {
    SendSMS(ctx context.Context, to, message string) error
}

// Implementation: Termii
type TermiiClient struct {
    apiKey string
    senderID string
}

func (c *TermiiClient) SendSMS(ctx context.Context, to, message string) error {
    // Call Termii API
    // POST https://api.ng.termii.com/api/sms/send
}
```

### 6. Platform: KYC Provider (Optional)
**New Interface:** `internal/platform/kyc/provider.go`

```go
type KYCProvider interface {
    VerifyIdentity(ctx context.Context, image []byte, metadata map[string]string) (*KYCResult, error)
}

type KYCResult struct {
    Verified    bool
    ID          string  // Provider's verification ID
    NIN         string  // Nigerian National ID
    BVN         string  // Bank Verification Number
    Name        string
    Reason      string  // Failure reason if not verified
}

// Implementation: Dojah
type DojahClient struct {
    apiKey    string
    publicKey string
    baseURL   string
}
```

---

## Testing Strategy

### Unit Tests
1. **Token Generation/Validation**
   - Test SHA-256 hashing consistency
   - Test expired token rejection
   - Test single-use enforcement
   - Test timing attack resistance

2. **Session Lifecycle**
   - Test state machine transitions (pending → in_review → verified)
   - Test invalid state transitions (cannot go from verified → failed)
   - Test expiration handling

3. **Phone OTP**
   - Test OTP generation randomness
   - Test attempt count enforcement (max 3)
   - Test expiration (5 minutes)

### Integration Tests
1. **Full ID Verification Flow**
   - Create session → Validate token → Submit evidence → Process async → Check profile update
   
2. **Phone Verification Flow**
   - Start verification → Receive OTP → Verify → Check profile

3. **Rate Limiting**
   - Test Redis rate limit enforcement
   - Test rate limit expiration

4. **Webhook Processing**
   - Test HMAC signature validation
   - Test webhook payload processing
   - Test duplicate webhook handling (idempotency)

### End-to-End Tests
1. **Micro-frontend Handshake**
   - Simulate full redirect flow from main app → verification app → back
   - Verify token passing across domains

2. **Error Scenarios**
   - Expired session
   - Invalid image format
   - AI rejection
   - Provider API failure

### Load Tests
1. **Concurrent Session Creation**
   - 100 users creating sessions simultaneously
   - Verify no race conditions in token generation

2. **OTP Flood Protection**
   - Test rate limiting holds under load
   - Verify Redis performance

---

## Deployment Plan

### Phase 1: Infrastructure Setup
1. **Database Migrations**
   - Run migration `010_create_verification_tables.sql`
   - Verify indexes created successfully

2. **R2 Bucket Creation**
   - Create bucket: `hauslet-verification-evidence`
   - Set CORS policy (allow uploads from `verification.hauslet.com`)
   - Enable server-side encryption

3. **Cloud Tasks Queues**
   - Create queue: `verification-processing`
   - Create queue: `verification-notifications`
   - Configure retry policies (max 3 retries, exponential backoff)

4. **SMS Provider Setup**
   - Register with Termii (or Twilio)
   - Configure sender ID ("Hauslet")
   - Test SMS delivery to Nigerian numbers

### Phase 2: Module Implementation
1. **Week 1: Domain + Repository**
   - Implement domain models
   - Implement GORM repositories
   - Write unit tests

2. **Week 2: Service Layer**
   - Implement session lifecycle logic
   - Implement token generation/validation
   - Implement phone OTP logic
   - Write unit tests

3. **Week 3: API Endpoints**
   - Implement HTTP handlers
   - Add rate limiting middleware
   - Write integration tests

4. **Week 4: Worker Integration**
   - Implement Cloud Tasks handlers
   - Integrate AI provider
   - Integrate KYC provider (optional)
   - Write end-to-end tests

### Phase 3: Client Integration (Out of Scope, But Noted)
1. **Main App Changes**
   - Add "Verify Identity" button in dashboard
   - Implement session creation API call
   - Handle redirect to verification subdomain

2. **Verification Frontend Deployment**
   - Deploy React app to Cloudflare Pages
   - Configure custom domain: `verification.hauslet.com`
   - Test liveness detection on various devices

### Phase 4: Rollout
1. **Feature Flag:** Deploy behind feature flag `enable_verification`
2. **Internal Testing:** Enable for staff accounts only (1 week)
3. **Beta Testing:** Enable for 100 selected users (2 weeks)
4. **Public Launch:** Enable for all supply-side users (hosts, agents, landlords)

### Phase 5: Monitoring
1. **Metrics to Track:**
   - Session creation rate
   - Verification success rate
   - Average verification time (session created → verified)
   - AI rejection rate
   - Provider API error rate
   - SMS delivery rate

2. **Alerts:**
   - Verification success rate drops below 80%
   - Provider API errors exceed 5%
   - Queue processing backlog exceeds 1000 jobs

---

## Open Questions

### Technical
1. **Which KYC provider will we use?**
   - Dojah (Nigerian-focused, NIN/BVN verification)
   - Smile ID (Pan-African, supports multiple countries)
   - Decision: Pending cost analysis

2. **How do we handle ID verification for non-Nigerians?**
   - Currently scoped for Nigeria only
   - Future: Integrate additional providers for Ghana, Kenya, etc.

3. **Should we support document upload (passport/driver's license)?**
   - Phase 1: Liveness check only (selfie)
   - Phase 2: Add document upload + OCR

4. **How long do we retain verification evidence (images)?**
   - Proposal: 90 days (GDPR compliance)
   - Option: Immediate deletion after verification (only keep provider reference)

### Business
1. **Do we charge for verification?**
   - Proposal: Free for first verification, ₦500 for re-verification
   - Finance module already supports `ResourceTypeIDVerification`

2. **Which user types require verification?**
   - Hosts: Required (to list properties)
   - Agents: Required (to list on behalf of landlords)
   - Landlords: Required (to create business entities)
   - Guests: Optional (but unlocks "verified guest" badge)

3. **What happens if verification fails?**
   - User can retry (max 3 times per day)
   - After 3 failures: Manual review required (support ticket)

4. **Do we offer manual verification as fallback?**
   - Yes: If AI/provider fails, user can submit to support for manual review

### Compliance
1. **Data Residency:** Where do we store verification images?
   - Currently: Cloudflare R2 (global)
   - Future: May need Nigerian data center for compliance

2. **Consent:** Do we need explicit consent for biometric data?
   - Yes: Add consent checkbox before liveness check
   - Update privacy policy to cover facial recognition

3. **NDPR Compliance (Nigeria Data Protection Regulation):**
   - Ensure user can request deletion of verification data
   - Implement data export (GDPR Article 20)

---

## Appendix

### Example Token Generation (Go)

```go
package service

import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
)

func generateToken() (plaintext string, hash string, err error) {
    // Generate 32 random bytes
    bytes := make([]byte, 32)
    if _, err := rand.Read(bytes); err != nil {
        return "", "", fmt.Errorf("failed to generate random token: %w", err)
    }
    
    // Encode to base64 for URL-safe transmission
    plaintext = base64.URLEncoding.EncodeToString(bytes)
    
    // Hash with SHA-256
    h := sha256.Sum256([]byte(plaintext))
    hash = fmt.Sprintf("%x", h)
    
    return plaintext, hash, nil
}

func validateToken(plaintext, storedHash string) bool {
    h := sha256.Sum256([]byte(plaintext))
    computedHash := fmt.Sprintf("%x", h)
    
    // Use constant-time comparison to prevent timing attacks
    return subtle.ConstantTimeCompare([]byte(computedHash), []byte(storedHash)) == 1
}
```

### Example SMS Template

```
Hauslet Verification

Your verification code is: {otp}

This code expires in 5 minutes.

Do not share this code with anyone.
```

### Example AI Prompt (Gemini)

```
Analyze this image and determine if it shows a live human face (not a photo of a photo, screen, or mask).

Criteria:
1. Is there a real human face present? (not a printed photo or screen)
2. Are the eyes open and naturally positioned?
3. Is the face centered and clearly visible?
4. Are there signs of life (natural skin texture, proper lighting)?

Return JSON:
{
  "is_live": true/false,
  "confidence": 0.0-1.0,
  "face_detected": true/false,
  "reason": "Brief explanation"
}
```

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 2.0 | 2026-01-01 | Backend Team | Complete rewrite based on micro-frontend architecture |
| 1.0 | 2025-12-15 | Backend Team | Initial draft |

---

**End of Document**
