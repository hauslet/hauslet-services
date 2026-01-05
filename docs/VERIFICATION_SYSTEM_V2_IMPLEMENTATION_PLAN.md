# Verification System Implementation Plan v5.0

**Status:** Design  
**Owner:** Backend Team  
**Last Updated:** January 5, 2026  
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
11. [Monitoring & Observability](#monitoring--observability)
12. [Testing Strategy](#testing-strategy)
13. [Deployment Plan](#deployment-plan)
14. [Disaster Recovery](#disaster-recovery)
15. [Cost Optimization](#cost-optimization)
16. [Open Questions (Resolved)](#open-questions-resolved)

---

## Executive Summary

This document defines the complete production-ready backend architecture for **Hauslet's Verification System**, covering:

* **ID + Liveness Verification** (single flow, liveness is part of ID verification)
* **Phone Number Verification** via OTP
* **Provider routing** by country (Dojah for Nigeria + supported countries; Veriff for unsupported countries)
* **Hybrid Async Architecture:** Uses worker jobs for submission and **Webhooks** for final status updates
* **Quality Pre-check:** Uses AI (Gemini) for image quality/face detection before incurring provider costs

### Key Design Principles

1. **Stateless Token Handshake:** Tokens are short-lived, single-use, and never stored in plaintext with replay attack protection
2. **Module Boundaries:** Verification is a dedicated module with its own data and services
3. **Async Processing:** Heavy operations are handled by worker jobs via Cloud Tasks
4. **Audit Trail:** All verification steps are recorded in `verification_events`
5. **No Profile Expansion:** Profile stores only high-level verification state
6. **Evidence Integrity:** All uploaded evidence is hashed and verified for tamper detection
7. **Circuit Breaker Pattern:** External provider calls are protected with automatic failover

---

## Architecture Overview

### The Complete Verification Flow

The system follows a "Hybrid Async" flow that acknowledges provider processing is often asynchronous while maintaining user experience.

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
         │                           │ 7. Validate token (hash match) │
         │                           │    Mark token as used        │
         │                           │                              │
         │                           │ 8. Return session metadata   │
         │                           │ ─────────────────────────────>
         │                           │                              │
         │                           │                   9. User captures liveness
         │                           │                      (face capture)
         │                           │                              │
         │                           │ 10. POST /verification/sessions/{id}/submit
         │                           │     (token + base64 image)   │
         │                           │ <────────────────────────────│
         │                           │                              │
         │                           │ 11. Store evidence (R2)      │
         │                           │     Calculate SHA256 hash    │
         │                           │     Enqueue job (Cloud Tasks)│
         │                           │                              │
         │                           │ 12. Return success           │
         │                           │ ─────────────────────────────>
         │                           │                              │
         │ 13. Redirect user back with status=pending               │
         │ <─────────────────────────────────────────────────────────
         │                           │                              │
         │                      14. Worker processes job            │
         │                          (AI Quality Check)              │
         │                           │                              │
         │                      15. Quality check passes            │
         │                           │                              │
         │                      16. Submit to provider              │
         │                          (Dojah/Veriff based on country) │
         │                           │                              │
         │                      17. Provider processes async        │
         │                           │                              │
         │                      18. Provider webhook received       │
         │                           │                              │
         │                      19. Update session status           │
         │                          Update profile.IDVerified=true  │
         │                           │                              │
```

### Components

| Component | Technology | Responsibility |
|-----------|------------|----------------|
| **Main App** | Next.js/React | Initiate verification, display status |
| **API Server** | Go (Chi + gqlgen) | Session management, evidence upload, phone OTP, webhook ingestion |
| **Worker** | Go | **Quality Pre-check** (Gemini) + Provider Submission |
| **Verification Frontend** | React (Vite) on Cloudflare Pages | Liveness capture, evidence submission |
| **Webhook Handler** | Go | Receives final decision from providers |
| **Queue** | Google Cloud Tasks | Async job processing |
| **Storage** | Cloudflare R2 | Evidence storage (private, 90-day retention) |
| **SMS Provider** | Termii (primary), Twilio (fallback) | OTP delivery |
| **KYC Providers** | Dojah (Nigeria), Veriff (Global) | ID + liveness verification |
| **Reconciler** | Go (Cron) | Fixes drift between Verification and Profile modules |
| **Circuit Breaker** | Go + Redis | Provider health monitoring |
| **Rate Limiter** | Go + Redis | Multi-dimensional rate limiting |

---

## Domain Model

### Core Entities

#### `VerificationSession`

Primary aggregate representing a verification attempt. Liveness is part of the ID verification flow.

```go
type VerificationSession struct {
    ID            uuid.UUID
    UserID        uuid.UUID
    Method        VerificationMethod  // id_liveness, phone_otp
    Status        VerificationStatus  // pending, in_review, provider_processing, verified, failed, expired, cancelled
    
    // Security
    TokenHash     string              // SHA256 hash of plaintext token
    TokenUsedAt   *time.Time          // When token was validated (prevents replay)
    ExpiresAt     time.Time           // Token expiration (10 minutes from creation)
    IPAddress     string              // Client IP for fraud detection
    UserAgent     string              // User agent for device fingerprinting
    
    // Evidence
    EvidenceURL   *string             // R2 object key
    EvidenceHash  *string             // SHA256 of uploaded image bytes
    EvidenceSize  *int64              // File size in bytes
    
    // Provider Details
    Provider      *string             // dojah, veriff
    ProviderRef   *string             // External provider ID (crucial for webhooks)
    ProviderCountry *string           // ISO country code
    ProviderStatus *string            // provider-specific status (raw)
    ProviderPayload map[string]any    // provider response metadata (JSON)
    ProviderCost  *float64            // Cost incurred for accounting
    ProviderLatency *int64            // Response time in ms
    
    // Quality Scoring
    QualityScore  *float64            // 0-1 confidence from Gemini pre-check
    QualityFlags  []string            // ["face_detected", "no_blur", "good_lighting"]
    
    // Outcome
    Attempts      int                 // Number of attempts this session
    RetryCount    int                 // Number of retries across sessions
    FailureReason *string             // Human readable failure reason
    FailureCode   *string             // Machine readable: image_blurry, doc_expired, face_mismatch
    
    // Metadata
    CreatedBy     string              // "user" or "system" or "admin"
    ReturnURL     string              // Redirect after completion
    Metadata      map[string]any      // Flexible extension field
    
    // Timestamps
    CreatedAt     time.Time
    UpdatedAt     time.Time
    CompletedAt   *time.Time
}
```

#### `VerificationEvent`

Immutable audit log of session changes with enhanced tracking.

```go
type VerificationEvent struct {
    ID          uuid.UUID
    SessionID   uuid.UUID
    EventType   string                 // session_created, evidence_submitted, provider_called, webhook_received
    Severity    string                 // info, warn, error, critical
    Actor       string                 // user_id or system component
    IPAddress   string                 // Source IP for audit trail
    Payload     map[string]interface{} // JSON metadata
    CreatedAt   time.Time
}
```

#### `PhoneVerificationAttempt`

Tracks OTP attempts separately from sessions with enhanced security.

```go
type PhoneVerificationAttempt struct {
    ID             uuid.UUID
    UserID         uuid.UUID
    PhoneNumber    string
    OTPHash        string       // SHA256 hash of plaintext OTP
    ExpiresAt      time.Time    // 5 minutes
    Verified       bool
    AttemptCount   int          // Updated atomically
    IPAddress      string       // Client IP for security
    VerifiedAt     *time.Time
    CreatedAt      time.Time
}
```

#### `ProviderCircuitBreaker` (NEW)

Tracks health of external providers for automatic failover.

```go
type ProviderCircuitBreaker struct {
    ProviderName   string
    State          string           // closed, open, half_open
    FailureCount   int
    SuccessCount   int
    LastFailureAt  *time.Time
    LastSuccessAt  *time.Time
    OpenedAt       *time.Time
    Config         CircuitBreakerConfig
    UpdatedAt      time.Time
}
```

### Enums

```go
type VerificationMethod string
const (
    MethodIDLiveness   VerificationMethod = "id_liveness"
    MethodPhoneOTP     VerificationMethod = "phone_otp"
)

type VerificationStatus string
const (
    StatusPending             VerificationStatus = "pending"
    StatusInReview            VerificationStatus = "in_review"           // Submitted to our worker
    StatusProviderProcessing  VerificationStatus = "provider_processing" // Handed off to external provider
    StatusVerified            VerificationStatus = "verified"
    StatusFailed              VerificationStatus = "failed"
    StatusExpired             VerificationStatus = "expired"
    StatusCancelled           VerificationStatus = "cancelled"
)

// Failure Codes (Actionable for UI)
const (
    FailureImageBlurry        = "image_blurry"
    FailureNoFaceDetected     = "no_face_detected"
    FailureDocumentExpired    = "document_expired"
    FailureFaceMismatch       = "face_mismatch"
    FailureProviderTimeout    = "provider_timeout"
    FailureProviderError      = "provider_error"
    FailureRateLimited        = "rate_limited"
    FailureSuspectedFraud     = "suspected_fraud"
)

// Quality Flags from AI Pre-check
const (
    QualityFaceDetected    = "face_detected"
    QualityNoBlur          = "no_blur"
    QualityGoodLighting    = "good_lighting"
    QualityNoGlare         = "no_glare"
    QualityDocumentVisible = "document_visible"
)

// Circuit Breaker States
const (
    CircuitClosed   = "closed"
    CircuitOpen     = "open"
    CircuitHalfOpen = "half_open"
)
```

---

## Workflow Specifications

### Workflow 1: ID Verification with Liveness (Enhanced)

#### Step 1: Session Creation with Concurrency Control

**Endpoint:** `POST /verification/sessions`

**Backend Logic:**

1. Validate auth (JWT)
2. **Rate Limit Check:** Check Redis for user/IP/country limits
3. **Concurrency Guard:** Check for existing active session using partial unique index (prevents multiple simultaneous verifications)
4. Generate secure token (32 bytes random)
5. Hash token (SHA-256) and store with `token_used_at = NULL`
6. Create session with `status = pending`, `expires_at = now + 10m`
7. Log `session_created` event with IP and user agent
8. Return session + redirect URL to verification micro-frontend

#### Step 2: Token Validation with Replay Protection

**Endpoint:** `GET /verification/sessions/validate?token=...`

**Backend Logic:**

1. Hash token and fetch session with optimistic lock
2. Ensure status is `pending`, not expired, and `token_used_at IS NULL`
3. **Atomic Update:** Set `token_used_at = NOW()` to prevent replay
4. Mark session `in_review`
5. Log `token_validated` event
6. Return session metadata to micro-frontend

#### Step 3: Evidence Submission with Integrity Verification

**Endpoint:** `POST /verification/sessions/{id}/submit`

**Backend Logic:**

1. **Replay Check:** Verify token was marked as used
2. Decode and validate base64 image (size, format)
3. **Calculate Hash:** SHA-256 of image bytes for integrity
4. Upload to R2 with metadata (user_id, session_id, hash)
5. Update session: `evidence_url`, `evidence_hash`, `status = in_review`, increment attempts
6. **Evidence Integrity Record:** Store hash in evidence chain
7. Log `evidence_submitted` event with hash
8. Enqueue `verification:process_submission` job with evidence hash
9. Return success to micro-frontend (user redirected back to main app)

#### Step 4: Worker Processing - Quality Pre-check

**Job:** `verification:process_submission`

**Worker Logic:**

1. **Integrity Verification:** Download image from R2, verify SHA-256 matches stored hash
2. **AI Quality Check (Gemini):**
   * Check for: Face detection, blur, glare, document visibility
   * Generate confidence score (0-1) and quality flags
   * *If Fail:* Update session `status = failed`, `failure_code = image_quality`. Stop.
   * *If Pass:* Proceed with provider submission
3. **Circuit Breaker Check:** Check if provider is healthy
4. **Provider Routing:**
   * If `Country == "NG"` → **Dojah** (Nigeria focus)
   * Else → **Veriff** (Global coverage)
5. **Provider Submission:**
   * Call Provider API with timeout (30s)
   * Save `provider_ref` (External ID)
   * Update session `status = provider_processing`
   * Record provider latency and estimated cost
6. **Circuit Breaker Update:** Record success/failure
7. Log `provider_submitted` event

#### Step 5: Webhook Processing (Async Finalization)

**Endpoint:** `POST /webhooks/verification/{provider}`

**Backend Logic:**

1. **Security:** Verify `X-Signature` header (HMAC validation)
2. **Lookup:** Find session by `provider_ref` (indexed query)
3. **Parse Decision:** Map provider status to our status
   * *Approved:* `status = verified`
   * *Rejected:* `status = failed` with mapped failure code
4. **Transaction:** Update session and profile atomically
5. **Profile Update:** Call `profile.SetVerificationStatus(userID, "identity", true)`
6. **Notification:** Send real-time update to user
7. Log `webhook_received` event

#### Step 6: Status Polling (Fallback)

**Endpoint:** `GET /verification/sessions/{id}`

**Backend Logic:**

1. Return session status for UI to show progress
2. Include `failure_code` for actionable UI feedback
3. Include `can_retry` boolean based on retry count and failure type
4. Include `estimated_completion_time` for provider_processing status

---

### Workflow 2: Phone Verification (OTP) - Enhanced Atomic

#### Step 1: Start Verification with Rate Limiting

**Endpoint:** `POST /verification/phone/start`

**Backend Logic:**

1. Validate E.164 phone number format
2. **Rate Limit (Redis):** Per user (5/day), per IP (50/day), per number (3/hour)
3. Generate 6-digit OTP
4. **Atomic Upsert:** Create or reset `PhoneVerificationAttempt` (reset attempts if expired)
5. Store hashed OTP (SHA-256)
6. Enqueue SMS job with provider fallback (Termii → Twilio)
7. Log `phone_verification_started` event

#### Step 2: Verify OTP with Atomic Operations

**Endpoint:** `POST /verification/phone/verify`

**Backend Logic:**

1. **DB Transaction with Lock:**
   * Select row `FOR UPDATE SKIP LOCKED`
   * Check `attempts < 3` and not expired
   * Verify hash match
   * Atomically increment attempt count
2. **On Success:**
   * Mark as verified
   * `profile.SetPhoneVerified(userID, true)`
   * Profile.VerificationLevel auto-updates via `SetVerificationStatus`
   * Add `phone_verified` badge
3. **On Failure:**
   * Return specific error (wrong_otp, too_many_attempts, expired)
4. Log `phone_verification_result` event

---

### Workflow 3: Manual Fallback (Disaster Recovery)

**Trigger:** Provider outage, circuit breaker open, or multiple failures

**Workflow:**

1. System detects failure and enqueues manual review
2. User notified: "Verification requires manual review (24-48 hours)"
3. Admin portal shows pending manual reviews
4. Admin reviews evidence, approves/rejects with reason
5. System updates session and profile accordingly
6. User notified of outcome

---

## Module Structure

```
internal/modules/verification/
├── domain/
│   ├── session.go              # VerificationSession with methods
│   ├── event.go                # VerificationEvent
│   ├── phone.go                # PhoneVerificationAttempt
│   ├── circuit_breaker.go      # ProviderCircuitBreaker (NEW)
│   ├── enums.go                # All enums and constants
│   ├── errors.go               # Domain-specific errors
│   └── interfaces.go           # Repository and service interfaces
├── repository/
│   ├── schema/
│   │   └── gorm.go             # GORM models and migrations
│   ├── session_repo.go         # Session repository with locking
│   ├── event_repo.go           # Event repository
│   ├── phone_repo.go           # Phone verification repository
│   ├── circuit_breaker_repo.go # Circuit breaker repository (NEW)
│   └── interface.go            # Repository interfaces
├── service/
│   ├── verification_service.go # Main service orchestration
│   ├── token_service.go        # Token generation/validation
│   ├── evidence_service.go     # Evidence upload/integrity (NEW)
│   ├── quality_service.go      # AI quality checking (NEW)
│   ├── provider_service.go     # Provider routing and submission
│   ├── circuit_breaker.go      # Circuit breaker logic (NEW)
│   ├── phone_service.go        # Phone OTP service
│   ├── webhook_service.go      # Webhook processing
│   └── reconciliation_service.go # Data consistency
├── providers/
│   ├── interface.go            # VerificationProvider interface
│   ├── dojah_adapter.go        # Dojah implementation
│   ├── veriff_adapter.go       # Veriff implementation
│   └── factory.go              # Provider factory
├── ports/
│   ├── http/
│   │   ├── routes.go           # Chi router setup
│   │   ├── middleware.go       # Auth, rate limiting
│   │   ├── session_handlers.go # Session endpoints
│   │   ├── phone_handlers.go   # Phone OTP endpoints
│   │   └── webhook_handlers.go # Provider webhooks
│   └── worker/
│       ├── submission_handler.go # Process verification job
│       ├── sms_handler.go      # Send SMS job
│       └── reconciliation_handler.go # Cron job
└── monitoring/
    ├── metrics.go              # OpenTelemetry metrics
    ├── logging.go              # Structured logging
    └── alerts.go               # Alert definitions
```

---

## Database Schema

### Migration: `create_verification_tables_v5.sql`

```sql
-- Core Tables
CREATE TABLE verification_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    method VARCHAR(50) NOT NULL CHECK (method IN ('id_liveness', 'phone_otp')),
    status VARCHAR(50) NOT NULL DEFAULT 'pending' 
        CHECK (status IN ('pending', 'in_review', 'provider_processing', 'verified', 'failed', 'expired', 'cancelled')),
    
    -- Security
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    token_used_at TIMESTAMP,                      -- NEW: Replay protection
    expires_at TIMESTAMP NOT NULL,
    ip_address INET,                              -- NEW: Fraud detection
    user_agent TEXT,                              -- NEW: Device fingerprinting
    
    -- Evidence Integrity
    evidence_url TEXT,
    evidence_hash VARCHAR(64),                    -- NEW: SHA256 of image
    evidence_size BIGINT,                         -- NEW: File size
    
    -- Provider Details
    provider VARCHAR(100),
    provider_ref VARCHAR(255),                    -- Indexed for webhooks
    provider_country VARCHAR(10),
    provider_cost DECIMAL(10, 4),                 -- NEW: Cost tracking
    provider_latency INTEGER,                     -- NEW: Performance tracking
    
    -- Quality Metrics
    quality_score DECIMAL(3, 2) 
        CHECK (quality_score >= 0 AND quality_score <= 1),
    quality_flags TEXT[],                         -- NEW: AI quality flags
    
    -- Outcome
    failure_reason TEXT,
    failure_code VARCHAR(50),                     -- Machine-readable code
    retry_count INTEGER DEFAULT 0,                -- NEW: Cross-session retries
    
    -- Metadata
    attempts INT NOT NULL DEFAULT 0,
    created_by VARCHAR(50) DEFAULT 'user',
    return_url TEXT NOT NULL,
    metadata JSONB,                               -- NEW: Flexible extensions
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    completed_at TIMESTAMP
);

-- Concurrency Guard: Only one active session per user
CREATE UNIQUE INDEX idx_verification_sessions_active_user 
ON verification_sessions(user_id) 
WHERE status IN ('pending', 'in_review', 'provider_processing');

-- Performance Indexes
CREATE INDEX idx_verification_sessions_user_status ON verification_sessions(user_id, status);
CREATE INDEX idx_verification_sessions_provider_ref ON verification_sessions(provider_ref);
CREATE INDEX idx_verification_sessions_evidence_hash ON verification_sessions(evidence_hash);
CREATE INDEX idx_verification_sessions_created_at ON verification_sessions(created_at DESC);
CREATE INDEX idx_verification_sessions_expires_at ON verification_sessions(expires_at) WHERE status = 'pending';

-- Audit Log
CREATE TABLE verification_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES verification_sessions(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    severity VARCHAR(20) DEFAULT 'info'           -- NEW: Severity levels
        CHECK (severity IN ('info', 'warn', 'error', 'critical')),
    actor VARCHAR(255),                           -- NEW: Who performed action
    ip_address INET,                              -- NEW: Source IP
    payload JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_verification_events_session_id ON verification_events(session_id);
CREATE INDEX idx_verification_events_created_at ON verification_events(created_at);
CREATE INDEX idx_verification_events_severity ON verification_events(severity) WHERE severity IN ('error', 'critical');

-- Phone Verification
CREATE TABLE phone_verification_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    phone_number VARCHAR(20) NOT NULL,
    otp_hash VARCHAR(64) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    verified BOOL NOT NULL DEFAULT false,
    attempt_count INT NOT NULL DEFAULT 0,
    ip_address INET,                              -- NEW: Security tracking
    verified_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_phone_verification_user_id ON phone_verification_attempts(user_id);
CREATE INDEX idx_phone_verification_phone ON phone_verification_attempts(phone_number);
CREATE INDEX idx_phone_verification_expires_at ON phone_verification_attempts(expires_at) WHERE verified = false;

-- NEW: Circuit Breaker State
CREATE TABLE provider_circuit_breakers (
    provider_name VARCHAR(100) PRIMARY KEY,
    state VARCHAR(20) NOT NULL CHECK (state IN ('closed', 'open', 'half_open')),
    failure_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    last_failure_at TIMESTAMP,
    last_success_at TIMESTAMP,
    opened_at TIMESTAMP,
    config JSONB NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- NEW: Rate Limit Tracking (Redis is primary, this is for audit)
CREATE TABLE verification_rate_limit_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_type VARCHAR(50) NOT NULL,  -- 'user', 'ip', 'phone'
    key_value VARCHAR(255) NOT NULL,
    count INTEGER NOT NULL,
    window_start TIMESTAMP NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    action VARCHAR(50) NOT NULL,    -- 'allowed', 'blocked'
    reason TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_rate_limit_audit_key ON verification_rate_limit_audit(key_type, key_value);
CREATE INDEX idx_rate_limit_audit_created ON verification_rate_limit_audit(created_at DESC);
```

---

## API Specification

### REST Endpoints

#### 1. **Create Verification Session**

```
POST /verification/sessions
Authorization: Bearer {jwt}
X-Forwarded-For: {client_ip}

{
  "method": "id_liveness",
  "return_url": "https://hauslet.com/dashboard",
  "metadata": {
    "device_id": "abc123",
    "platform": "web"
  }
}

Response:
{
  "session_id": "uuid",
  "token": "plaintext_token",  // Single use, 10min expiry
  "redirect_url": "https://verification.hauslet.com?token=...",
  "expires_in": 600
}
```

#### 2. **Validate Token**

```
GET /verification/sessions/validate?token={token}

Response:
{
  "session_id": "uuid",
  "status": "pending",
  "method": "id_liveness",
  "expires_at": "2026-01-05T14:30:00Z"
}
```

#### 3. **Submit Evidence**

```
POST /verification/sessions/{id}/submit
{
  "token": "plaintext_token",
  "evidence": {
    "type": "selfie",
    "image": "base64_jpeg_data",
    "metadata": {
      "device": "Chrome",
      "os": "Android",
      "timestamp": "2026-01-05T14:25:00Z"
    }
  }
}

Response:
{
  "status": "in_review",
  "message": "Verification submitted",
  "estimated_completion": "5-10 minutes"
}
```

#### 4. **Get Session Status**

```
GET /verification/sessions/{id}
Authorization: Bearer {jwt}

Response:
{
  "session_id": "uuid",
  "status": "provider_processing",
  "failure_code": null,
  "can_retry": true,
  "retry_count": 0,
  "quality_score": 0.85,
  "provider": "veriff",
  "created_at": "2026-01-05T14:20:00Z",
  "updated_at": "2026-01-05T14:25:00Z"
}
```

#### 5. **Retry Verification** (NEW)

```
POST /verification/sessions/{id}/retry
Authorization: Bearer {jwt}

Response:
{
  "retry_initiated": true,
  "new_session_id": "uuid",
  "estimated_cost": 1.50
}
```

#### 6. **Start Phone Verification**

```
POST /verification/phone/start
Authorization: Bearer {jwt}

{
  "phone_number": "+2348012345678"
}

Response:
{
  "message": "OTP sent",
  "expires_in": 300,
  "retry_available_in": 60
}
```

#### 7. **Verify Phone OTP**

```
POST /verification/phone/verify
Authorization: Bearer {jwt}

{
  "phone_number": "+2348012345678",
  "otp": "123456"
}

Response:
{
  "verified": true,
  "message": "Phone number verified"
}
```

#### 8. **Provider Webhooks**

```
POST /webhooks/verification/{provider}
Headers:
  X-Signature: {hmac_sha256}
  X-Timestamp: {unix_timestamp}

Body: (Provider-specific JSON)
```

#### 9. **Export Verification Data** (GDPR/NDPR Compliance) (NEW)

```
GET /verification/sessions/{id}/export
Authorization: Bearer {jwt}
Accept: application/json

Response:
{
  "session_data": { ... },
  "events": [ ... ],
  "evidence_url": "https://r2.signed-url.com/...",
  "provider_responses": [ ... ]
}
```

---

## Queue Integration

The system uses the existing queue infrastructure with enhanced job definitions:

### Queues

1. `verification-processing` - Core verification jobs
2. `verification-notifications` - SMS and user notifications
3. `verification-maintenance` - Cron jobs and cleanup

### Enhanced Job Definitions

```go
// Enhanced verification job with retry tracking
type VerifyLivenessJob struct {
    queue.BaseJob
    SessionID     string    `json:"session_id"`
    EvidenceURL   string    `json:"evidence_url"`
    EvidenceHash  string    `json:"evidence_hash"`  // NEW: Integrity verification
    Priority      string    `json:"priority"`       // high, normal, low
    RetryAttempt  int       `json:"retry_attempt"`  // 0 for first attempt
    ScheduledFor  time.Time `json:"scheduled_for"`  // For delayed retries
}

func (j *VerifyLivenessJob) CalculateDelay() time.Duration {
    // Exponential backoff with jitter
    baseDelay := time.Duration(math.Pow(2, float64(j.RetryAttempt))) * time.Minute
    jitter := time.Duration(rand.Intn(30)) * time.Second
    return baseDelay + jitter
}

type SendVerificationSMSJob struct {
    queue.BaseJob
    PhoneNumber string `json:"phone_number"`
    OTP         string `json:"otp"`
    UserID      string `json:"user_id"`
    Provider    string `json:"provider"`    // termii or twilio
    RetryCount  int    `json:"retry_count"` // For provider fallback
}

// NEW: Cost optimization job (runs every 6 hours)
type OptimizeProviderRoutingJob struct {
    queue.BaseJob
    WindowStart time.Time `json:"window_start"`
    WindowEnd   time.Time `json:"window_end"`
}

// NEW: Data retention job (runs daily)
type CleanupVerificationDataJob struct {
    queue.BaseJob
    RetentionDays int `json:"retention_days"`
}

// NEW: Reconciliation job (runs every 15 minutes)
type ReconcileVerificationDataJob struct {
    queue.BaseJob
    BatchSize int `json:"batch_size"`
}
```

### Queue Configuration

```yaml
queues:
  verification-processing:
    max_concurrent: 10
    max_retries: 3
    retry_delay: exponential
    dead_letter_queue: verification-failed
    priorities:
      - high: 5
      - normal: 3
      - low: 1
  
  verification-notifications:
    max_concurrent: 5
    max_retries: 2
  
  verification-maintenance:
    schedule:
      - name: optimize-provider-routing
        cron: "0 */6 * * *"        # Every 6 hours
        job: OptimizeProviderRoutingJob
      - name: cleanup-old-sessions
        cron: "0 2 * * *"          # Daily at 2 AM
        job: CleanupVerificationDataJob
        args: {retention_days: 90}
      - name: reconcile-data
        cron: "*/15 * * * *"       # Every 15 minutes
        job: ReconcileVerificationDataJob
        args: {batch_size: 100}
```

---

## Security & Compliance

### Enhanced Security Measures

#### 1. **Token Security with Replay Protection**

```go
// Token lifecycle states tracked in database
type TokenLifecycle struct {
    GeneratedAt  time.Time
    ValidatedAt  *time.Time  // When token_used_at is set
    SubmittedAt  *time.Time  // When evidence is submitted
    ExpiresAt    time.Time
}

// Redis cache for fast validation
func ValidateToken(tokenHash string) error {
    key := fmt.Sprintf("verification:token:%s", tokenHash)
    
    // Check if already used (fast path)
    used, err := redis.Get(ctx, key).Result()
    if err == nil && used == "1" {
        return ErrTokenAlreadyUsed
    }
    
    // Database check with optimistic locking
    result := db.Model(&VerificationSession{}).
        Where("token_hash = ? AND expires_at > ? AND token_used_at IS NULL", 
              tokenHash, time.Now()).
        Update("token_used_at", time.Now())
    
    if result.RowsAffected == 0 {
        return ErrInvalidToken
    }
    
    // Mark as used in Redis (short TTL)
    redis.Set(ctx, key, "1", 10*time.Minute)
    
    return nil
}
```

#### 2. **Evidence Integrity Chain**

```go
// Store evidence hash chain for tamper detection
type EvidenceChain struct {
    SessionID    uuid.UUID
    PreviousHash string      // Hash of previous evidence or session
    CurrentHash  string      // SHA256 of current evidence
    Timestamp    time.Time
    Signature    string      // HMAC of (SessionID + PreviousHash + CurrentHash + Timestamp)
}

func VerifyEvidenceChain(sessionID uuid.UUID, evidenceData []byte) error {
    // Calculate hash of new evidence
    newHash := sha256Hash(evidenceData)
    
    // Get previous hash from database
    prevHash := getPreviousEvidenceHash(sessionID)
    
    // Create and sign new chain link
    chain := EvidenceChain{
        SessionID:    sessionID,
        PreviousHash: prevHash,
        CurrentHash:  newHash,
        Timestamp:    time.Now(),
    }
    chain.Signature = signChain(chain)
    
    // Store in database
    return storeEvidenceChain(chain)
}

func DetectTampering(sessionID uuid.UUID) bool {
    chains := getEvidenceChains(sessionID)
    
    for i := 1; i < len(chains); i++ {
        if chains[i].PreviousHash != chains[i-1].CurrentHash {
            return true // Tampering detected
        }
        if !verifySignature(chains[i]) {
            return true // Signature invalid
        }
    }
    return false
}
```

#### 3. **Rate Limiting with Multiple Dimensions**

```go
type RateLimitConfig struct {
    UserLimit      int           // per day
    IPLimit        int           // per day  
    CountryLimit   int           // per day per country
    PhoneLimit     int           // per hour per phone
    Window         time.Duration // Sliding window
}

func CheckRateLimit(ctx context.Context, userID, ip, country, phone string) error {
    limits := []RateLimitCheck{
        {Key: fmt.Sprintf("user:%s", userID), Limit: config.UserLimit},
        {Key: fmt.Sprintf("ip:%s", ip), Limit: config.IPLimit},
        {Key: fmt.Sprintf("country:%s", country), Limit: config.CountryLimit},
        {Key: fmt.Sprintf("phone:%s", phone), Limit: config.PhoneLimit},
    }
    
    for _, limit := range limits {
        count, err := redis.IncrWithWindow(ctx, limit.Key, config.Window)
        if err != nil {
            return err
        }
        if count > limit.Limit {
            metrics.Increment("verification.rate_limited", "key", limit.Key)
            return ErrRateLimited
        }
    }
    
    return nil
}
```

#### 4. **Circuit Breaker for External Providers**

```go
type CircuitBreaker struct {
    Name          string
    FailureThreshold int      // e.g., 5 failures
    SuccessThreshold int      // e.g., 3 successes to close
    Timeout       time.Duration // e.g., 30 seconds open state
    LastCheck     time.Time
    State         string     // closed, open, half_open
}

func (cb *CircuitBreaker) AllowRequest() bool {
    switch cb.State {
    case "closed":
        return true
    case "open":
        if time.Since(cb.LastCheck) > cb.Timeout {
            cb.State = "half_open"
            cb.LastCheck = time.Now()
            return true // Allow one request to test
        }
        return false
    case "half_open":
        // Allow only one request at a time
        return time.Since(cb.LastCheck) > cb.Timeout
    }
    return false
}

func (cb *CircuitBreaker) RecordSuccess() {
    if cb.State == "half_open" {
        cb.State = "closed"
        cb.ResetCounters()
    }
}

func (cb *CircuitBreaker) RecordFailure() {
    cb.FailureCount++
    if cb.FailureCount >= cb.FailureThreshold {
        cb.State = "open"
        cb.LastCheck = time.Now()
    }
}
```

### Compliance Framework

#### GDPR/NDPR Compliance

```go
// Right to erasure implementation
func DeleteUserVerificationData(userID uuid.UUID) error {
    // 1. Get all sessions for user
    sessions := sessionRepo.FindByUserID(userID)
    
    // 2. Delete evidence from R2 (90-day lifecycle handles this)
    for _, session := range sessions {
        if session.EvidenceURL != nil {
            r2Client.Delete(ctx, *session.EvidenceURL)
        }
    }
    
    // 3. Anonymize database records (keep for audit)
    err := db.Exec(`
        UPDATE verification_sessions 
        SET ip_address = NULL, 
            user_agent = NULL,
            evidence_url = NULL,
            evidence_hash = NULL,
            metadata = jsonb_set(metadata, '{gdpr_anonymized}', 'true')
        WHERE user_id = ?`, userID).Error
    
    // 4. Log deletion for audit
    auditLogger.LogDeletion(userID, "verification_data", "GDPR_request")
    
    return err
}

// Biometric consent tracking
type BiometricConsent struct {
    UserID      uuid.UUID
    Purpose     string     // "identity_verification"
    LegalBasis  string     // "explicit_consent"
    GrantedAt   time.Time
    ExpiresAt   time.Time  // Consent expires after 1 year
    WithdrawnAt *time.Time
    RecordID    string     // Link to consent management system
}

func CheckBiometricConsent(userID uuid.UUID) (bool, error) {
    consent := BiometricConsent{}
    err := db.Where("user_id = ? AND purpose = ? AND withdrawn_at IS NULL AND expires_at > ?",
        userID, "identity_verification", time.Now()).
        First(&consent).Error
    
    return err == nil, err
}
```

#### R2 Lifecycle Policy

```yaml
# Cloudflare R2 lifecycle configuration
lifecycle_rules:
  - id: delete_evidence_after_90_days
    enabled: true
    prefix: "evidence/"
    expiration_days: 90
    abort_incomplete_multipart_upload_days: 7
  
  - id: transition_to_archive
    enabled: false  # Optional for cost savings
    prefix: "evidence/"
    transitions:
      - days: 30
        storage_class: "GLACIER"
```

---

## Integration Points

### Profile Module Integration

**No new fields added to `profile`**. We use existing fields:
* `PhoneVerified`
* `IDVerified`
* `VerificationLevel`
* `VerificationDate`

**Integration Pattern:**

```go
// Profile service method
func (s *ProfileService) SetVerificationStatus(ctx context.Context, userID uuid.UUID, verificationType string, verified bool) error {
    tx := s.db.Begin()
    
    // Update profile based on verification type
    switch verificationType {
    case "identity":
        tx.Model(&Profile{}).
            Where("user_id = ?", userID).
            Updates(map[string]interface{}{
                "IDVerified": verified,
                "VerificationDate": time.Now(),
            })
        
        // Recalculate verification level
        profile := &Profile{}
        tx.Where("user_id = ?", userID).First(profile)
        
        level := CalculateVerificationLevel(profile)
        tx.Model(&Profile{}).
            Where("user_id = ?", userID).
            Update("VerificationLevel", level)
            
    case "phone":
        tx.Model(&Profile{}).
            Where("user_id = ?", userID).
            Update("PhoneVerified", verified)
    }
    
    return tx.Commit().Error
}

// Resource Gate integration
type ResourceGate struct {
    profileSvc ProfileService
}

func (g *ResourceGate) CheckAccess(ctx context.Context, userID uuid.UUID, resourceType string, resourceValue interface{}) bool {
    profile, err := g.profileSvc.Get(ctx, userID)
    if err != nil {
        return false
    }
    
    switch resourceType {
    case "booking":
        bookingValue := resourceValue.(float64)
        return g.RequiresIDVerification(bookingValue, profile)
        
    case "roomie_request":
        return profile.VerificationLevel >= "level2"
        
    case "legal_document":
        return profile.IDVerified && profile.PhoneVerified
    }
    
    return false
}
```

### Authorization Module Integration

* **Supply Gate:** Already checks `profile.IDVerified` for host-side actions
* **Resource Gate:** Generic gate (`authorization/resource_gate.go`) for guest/resource gating in future modules
* **Verification Checks:** Real-time verification status checking via profile or session status

### Provider Adapter Pattern

```go
// Unified provider interface
type VerificationProvider interface {
    // Returns provider name (e.g., "veriff")
    Name() string
    
    // Returns supported countries
    SupportedCountries() []string
    
    // Submits verification data
    SubmitVerification(ctx context.Context, data SubmissionData) (string, string, error)
    
    // Parses incoming webhook
    ParseWebhook(r *http.Request) (WebhookResult, error)
    
    // Verifies webhook signature
    VerifySignature(r *http.Request) bool
    
    // Estimates cost for this verification
    EstimateCost(ctx context.Context, data SubmissionData) (float64, error)
    
    // Checks service health
    HealthCheck(ctx context.Context) error
}

// Provider factory with circuit breaker
type ProviderFactory struct {
    providers map[string]VerificationProvider
    breakers  map[string]*CircuitBreaker
}

func (f *ProviderFactory) GetProvider(country string) (VerificationProvider, error) {
    // Routing logic
    if country == "NG" || slices.Contains(dojahSupportedCountries, country) {
        return f.getWithBreaker("dojah")
    }
    return f.getWithBreaker("veriff")
}

func (f *ProviderFactory) getWithBreaker(name string) (VerificationProvider, error) {
    breaker := f.breakers[name]
    if !breaker.AllowRequest() {
        return nil, ErrProviderUnavailable
    }
    return f.providers[name], nil
}
```

---

## Monitoring & Observability

### Key Metrics Dashboard

```go
// OpenTelemetry metrics definition
var (
    verificationCounter = otel.Meter("verification").
        NewInt64Counter("verification.attempts",
            metric.WithDescription("Total verification attempts"))
    
    verificationDuration = otel.Meter("verification").
        NewFloat64Histogram("verification.duration_ms",
            metric.WithDescription("Verification processing duration"))
    
    qualityScoreGauge = otel.Meter("verification").
        NewFloat64Gauge("verification.quality_score",
            metric.WithDescription("AI quality score distribution"))
    
    providerLatency = otel.Meter("verification").
        NewFloat64Histogram("verification.provider_latency_ms",
            metric.WithDescription("Provider API latency"))
    
    costCounter = otel.Meter("verification").
        NewFloat64Counter("verification.cost",
            metric.WithDescription("Verification costs by provider"))
)

// Usage in code
func ProcessVerification(ctx context.Context, sessionID string) error {
    start := time.Now()
    verificationCounter.Add(ctx, 1, metric.WithAttributes(
        attribute.String("method", "id_liveness"),
    ))
    
    defer func() {
        duration := float64(time.Since(start).Milliseconds())
        verificationDuration.Record(ctx, duration, metric.WithAttributes(
            attribute.String("status", "success"),
        ))
    }()
    
    // ... processing logic
    
    return nil
}
```

### Business Metrics

| Metric | Description | Target | Alert Threshold |
|--------|-------------|--------|-----------------|
| Verification Success Rate | % of verifications that succeed | > 95% | < 90% for 15min |
| Time to Verify | End-to-end verification time | < 5min (p95) | > 10min (p95) |
| Quality Rejection Rate | % rejected by AI pre-check | < 10% | > 25% |
| Provider Success Rate | Success rate by provider | > 90% each | < 85% |
| Cost per Verification | Average cost per attempt | < $2.50 | > $3.50 |
| Evidence Upload Success | % of evidence uploads that succeed | > 99% | < 95% |
| Webhook Processing Time | Time to process provider webhook | < 1s (p99) | > 5s (p99) |

### Alerting Rules

```yaml
alerts:
  - name: high_verification_failure_rate
    condition: verification_success_rate < 90%
    for: 15m
    severity: critical
    annotations:
      summary: "High verification failure rate"
      description: "Verification success rate is {{ $value }}% (< 90%)"
    notifications:
      - pagerduty
      - slack#engineering
  
  - name: provider_circuit_breaker_open
    condition: circuit_breaker_state{state="open"} > 0
    for: 5m
    severity: warning
    annotations:
      summary: "Provider circuit breaker is open"
      description: "{{ $labels.provider }} circuit breaker is open"
    notifications:
      - slack#engineering
  
  - name: evidence_tampering_detected
    condition: tamper_detection_count > 5
    for: 1h
    severity: high
    annotations:
      summary: "Evidence tampering detected"
      description: "{{ $value }} tampering attempts detected"
    notifications:
      - pagerduty
      - slack#security
  
  - name: verification_cost_spike
    condition: verification_cost_per_user > 10
    for: 1h
    severity: warning
    annotations:
      summary: "Verification cost spike"
      description: "Cost per verification is ${{ $value }} (> $10)"
    notifications:
      - slack#finance
```

### Structured Logging

```go
type VerificationLog struct {
    Timestamp    time.Time              `json:"timestamp"`
    Level        string                 `json:"level"`
    SessionID    string                 `json:"session_id,omitempty"`
    UserID       string                 `json:"user_id,omitempty"`
    CorrelationID string                `json:"correlation_id"`
    Component    string                 `json:"component"` // "api", "worker", "webhook"
    Message      string                 `json:"message"`
    DurationMS   int64                  `json:"duration_ms,omitempty"`
    Error        string                 `json:"error,omitempty"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Example log entry
{
  "timestamp": "2026-01-05T14:25:00Z",
  "level": "info",
  "session_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "user_id": "user_123",
  "correlation_id": "corr_abc123",
  "component": "worker",
  "message": "Provider submission completed",
  "duration_ms": 1250,
  "metadata": {
    "provider": "veriff",
    "country": "US",
    "quality_score": 0.85,
    "estimated_cost": 1.75
  }
}
```

---

## Testing Strategy

### Comprehensive Test Suite

```
tests/
├── unit/
│   ├── token_validation_test.go      # Token security tests
│   ├── evidence_integrity_test.go    # Hash verification tests
│   ├── quality_engine_test.go        # AI pre-check tests
│   ├── circuit_breaker_test.go       # Circuit breaker logic
│   └── rate_limiter_test.go          # Rate limiting logic
├── integration/
│   ├── session_lifecycle_test.go     # Complete session flow
│   ├── provider_routing_test.go      # Provider selection logic
│   ├── webhook_handling_test.go      # Webhook processing
│   ├── data_consistency_test.go      # Profile-session sync
│   └── queue_integration_test.go     # Job processing
├── e2e/
│   ├── complete_flow_test.go         # Full verification flow
│   ├── failure_scenarios_test.go     # Error handling
│   └── load_test.go                  # Performance under load
├── security/
│   ├── replay_attack_test.go         # Token replay tests
│   ├── tamper_detection_test.go      # Evidence tampering tests
│   └── rate_limit_bypass_test.go     # Rate limit evasion tests
└── compliance/
    ├── gdpr_test.go                  # Data deletion tests
    └── consent_test.go               # Consent management tests
```

### Security Testing Examples

```go
func TestReplayAttackProtection(t *testing.T) {
    // Create session and token
    session := createTestSession()
    token := generateTestToken()
    
    // First validation should succeed
    err := validateToken(token)
    assert.NoError(t, err)
    
    // Second validation should fail (replay attack)
    err = validateToken(token)
    assert.Error(t, err)
    assert.Equal(t, ErrTokenAlreadyUsed, err)
}

func TestEvidenceTamperingDetection(t *testing.T) {
    originalImage := loadTestImage("valid_selfie.jpg")
    tamperedImage := tamperImage(originalImage) // Modify pixels
    
    // Calculate hashes
    originalHash := sha256Hash(originalImage)
    tamperedHash := sha256Hash(tamperedImage)
    
    // Simulate upload with original hash
    session := &VerificationSession{
        EvidenceHash: &originalHash,
    }
    
    // Attempt to verify tampered image
    err := session.VerifyEvidence(tamperedImage)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "hash mismatch")
    assert.Equal(t, ErrEvidenceTampered, err)
}

func TestCircuitBreakerPattern(t *testing.T) {
    breaker := NewCircuitBreaker("test-provider", 3, 2, 30*time.Second)
    
    // Simulate failures
    for i := 0; i < 3; i++ {
        assert.True(t, breaker.AllowRequest())
        breaker.RecordFailure()
    }
    
    // Should be open after 3 failures
    assert.False(t, breaker.AllowRequest())
    
    // Wait for timeout
    time.Sleep(35 * time.Second)
    
    // Should allow one request in half-open
    assert.True(t, breaker.AllowRequest())
}
```

### Load Testing Scenarios

```yaml
load_tests:
  - name: peak_registration_load
    scenario: "1000 users verify simultaneously"
    users: 1000
    ramp_up: 1m
    hold_for: 5m
    metrics:
      - p95_response_time: < 2s
      - error_rate: < 1%
      - evidence_upload_success: > 99%
    endpoints:
      - POST /verification/sessions
      - POST /verification/sessions/{id}/submit
  
  - name: provider_outage_recovery
    scenario: "Primary provider fails during load"
    users: 500
    ramp_up: 30s
    steps:
      - run: 2m normal_load
      - simulate: provider_failure(dojah)
      - run: 5m with_failure
      - restore: provider_recovery
      - run: 2m recovery
    metrics:
      - automatic_fallback_time: < 30s
      - success_rate_during_outage: > 80%
      - no_data_loss: true
  
  - name: rate_limit_enforcement
    scenario: "Malicious user attempts to bypass rate limits"
    users: 10
    strategy:
      - 1 legitimate_user: normal_behavior
      - 9 malicious_users: spam_attempts
    metrics:
      - legitimate_users_unaffected: true
      - malicious_users_blocked: within 5_requests
      - false_positives: 0
```

---

## Deployment Plan

### Phase 1: Infrastructure Setup (Week 1-2)

| Day | Task | Owner | Success Criteria |
|-----|------|-------|------------------|
| 1-2 | Database migrations | DBA | Tables created, indexes optimized, constraints validated |
| 3-4 | R2 bucket with lifecycle rules | DevOps | 90-day auto-deletion working, CORS configured |
| 5 | Redis cluster for rate limiting | DevOps | < 5ms p95 latency, persistence configured |
| 6-7 | Cloud Tasks queues | DevOps | Dead-letter queues working, alerting configured |
| 8-9 | OpenTelemetry setup | DevOps | Metrics visible in Grafana, logs in Loki |
| 10 | Provider credentials | Security | Keys in Secret Manager, rotation scheduled |

### Phase 2: Core Implementation (Week 3-4)

| Component | Priority | Dependencies | Test Coverage | Owner |
|-----------|----------|--------------|---------------|-------|
| Session Management | P0 | Database, Redis | 95% | Backend Team |
| Evidence Service | P0 | R2, Redis | 90% | Backend Team |
| Quality Engine | P1 | Gemini API | 85% | AI Team |
| Provider Adapters | P0 | Dojah/Veriff APIs | 90% | Backend Team |
| Webhook Handlers | P0 | Provider APIs | 95% | Backend Team |
| Circuit Breakers | P1 | Redis | 90% | Backend Team |
| Rate Limiting | P1 | Redis | 95% | Backend Team |
| Reconciliation Service | P2 | Database | 85% | Backend Team |

### Phase 3: Safety Features (Week 5)

1. **Rate Limiting Rollout**
   * Day 1: Deploy in monitor-only mode
   * Day 2: Analyze logs for false positives
   * Day 3: Enable enforcement with conservative limits
   * Day 4-5: Adjust based on traffic patterns

2. **Circuit Breaker Tuning**
   * Start with conservative thresholds (10 failures, 60s timeout)
   * Monitor provider health metrics
   * Adjust based on real failure patterns

3. **Monitoring Enablement**
   * Deploy Grafana dashboards
   * Configure alerts in warning-only mode
   * Train team on alert response procedures

### Phase 4: Gradual Rollout (Week 6)

| Cohort | Size | Duration | Validation Criteria |
|--------|------|----------|---------------------|
| Internal Staff | 100% | 2 days | Manual verification succeeds, no data loss |
| Beta Users | 10% | 3 days | Compare success rates with manual process |
| Nigeria Users | 50% | 2 days | Monitor Dojah integration, cost tracking |
| Global Users | 25% | 2 days | Monitor Veriff integration, latency |
| All Users | 100% | Ongoing | Continuous monitoring of all metrics |

### Rollback Strategy

```yaml
rollback_triggers:
  - verification_success_rate < 80% for 1 hour
  - evidence_upload_failure_rate > 10%
  - provider_cost > 200% of expected
  - critical_security_issue_detected

rollback_steps:
  1. Disable new verification sessions (feature flag)
  2. Route all traffic to manual review workflow
  3. If data-safe: revert database migrations
  4. Restore previous verification endpoints
  5. Notify users of temporary service interruption

rollback_time_target: < 15 minutes from trigger to completion
rollback_owner: On-call engineer + Backend lead
```

---

## Disaster Recovery

### Manual Fallback Workflow

When automated verification fails (provider outage, circuit breaker open, multiple failures):

```go
// Manual review workflow implementation
type ManualReviewWorkflow struct {
    ticketRepo    ManualReviewTicketRepository
    notificationSvc NotificationService
    adminPortalURL string
}

func (w *ManualReviewWorkflow) Initiate(sessionID uuid.UUID, reason string) error {
    // 1. Create manual review ticket
    ticket := &ManualReviewTicket{
        SessionID: sessionID,
        Reason:    reason,
        Priority:  "high",
        Status:    "pending",
        CreatedAt: time.Now(),
    }
    
    if err := w.ticketRepo.Create(ticket); err != nil {
        return err
    }
    
    // 2. Notify admin team
    w.notificationSvc.NotifyAdmins("manual_review_needed", map[string]any{
        "session_id": sessionID.String(),
        "reason":     reason,
        "priority":   "high",
        "link":       fmt.Sprintf("%s/admin/review/%s", w.adminPortalURL, sessionID),
        "timestamp":  time.Now().Format(time.RFC3339),
    })
    
    // 3. Update user
    w.notificationSvc.NotifyUser(session.UserID, "verification_manual_review", map[string]any{
        "estimated_time": "24-48 hours",
        "reference_id":   sessionID.String(),
    })
    
    // 4. Log for audit
    auditLogger.LogManualReviewInitiated(sessionID, reason)
    
    return nil
}

// Admin review endpoint (internal)
func (w *ManualReviewWorkflow) Review(ticketID uuid.UUID, adminID uuid.UUID, decision ReviewDecision) error {
    // 1. Get ticket with lock
    ticket := w.ticketRepo.GetWithLock(ticketID)
    
    // 2. Get session
    session := sessionRepo.Get(ticket.SessionID)
    
    // 3. Update based on decision
    tx := db.Begin()
    defer tx.RollbackUnlessCommitted()
    
    switch decision.Action {
    case "approve":
        session.Status = StatusVerified
        session.CompletedAt = &time.Now()
        session.Provider = &"manual_review"
        session.FailureReason = nil
        
        // Update profile
        if err := profileSvc.SetVerificationStatus(ctx, session.UserID, "identity", true); err != nil {
            return err
        }
        
    case "reject":
        session.Status = StatusFailed
        session.CompletedAt = &time.Now()
        session.FailureReason = &decision.Reason
        session.FailureCode = &"manual_rejection"
    }
    
    // 4. Update ticket
    ticket.Status = "completed"
    ticket.ReviewedBy = &adminID
    ticket.ReviewedAt = &time.Now()
    ticket.Decision = decision.Action
    ticket.Notes = decision.Notes
    
    // 5. Commit
    if err := tx.Commit().Error; err != nil {
        return err
    }
    
    // 6. Notify user
    w.notificationSvc.NotifyUser(session.UserID, "verification_completed", map[string]any{
        "status":   session.Status,
        "reason":   session.FailureReason,
        "reviewer": "Hauslet Trust & Safety Team",
    })
    
    return nil
}
```

### Provider Outage Response Playbook

```markdown
# Provider Outage Response Playbook

## Detection
1. Circuit breaker opens for provider
2. Alert fires to on-call engineer via PagerDuty
3. Dashboard shows 100% failure rate for provider
4. Check provider status page for confirmed outage

## Immediate Actions (First 5 minutes)
1. Acknowledge alert and create incident ticket
2. Verify outage via provider status page and health checks
3. Check if fallback provider is available and healthy
4. If yes: Confirm routing engine automatically switched
5. If no: Enable manual review workflow automatically

## Communication
1. Update internal status page with incident details
2. Notify customer support team with template message
3. Prepare customer communication if outage > 30 minutes
4. Post updates every 30 minutes until resolved

## Recovery
1. Monitor provider status for restoration
2. When restored: Gradually reintroduce traffic
   - 10% traffic for 5 minutes
   - 50% traffic for 10 minutes  
   - 100% traffic if success rate normal
3. Verify success rate returns to baseline (> 95%)
4. Close circuit breaker if manually opened
5. Close incident ticket

## Post-Mortem (Within 24 hours)
1. Analyze impact: Number of users affected, business impact
2. Review automatic failover effectiveness
3. Identify detection time improvements
4. Update playbook if needed
5. Share learnings with team
```

### Data Export for Regulatory Requests

```go
// GDPR/NDPR compliant data export
type VerificationExport struct {
    UserID       uuid.UUID
    Sessions     []VerificationSession
    Events       []VerificationEvent
    EvidenceURLs []SignedURL
    GeneratedAt  time.Time
    RequestID    string
    ExpiresAt    time.Time
}

func ExportUserVerificationData(userID uuid.UUID) (*VerificationExport, error) {
    // 1. Get all sessions for user (with pagination)
    sessions, err := sessionRepo.FindByUserID(userID, 1000)
    if err != nil {
        return nil, err
    }
    
    // 2. Get all events for those sessions
    sessionIDs := make([]uuid.UUID, len(sessions))
    for i, s := range sessions {
        sessionIDs[i] = s.ID
    }
    events, err := eventRepo.FindBySessionIDs(sessionIDs)
    if err != nil {
        return nil, err
    }
    
    // 3. Generate signed URLs for evidence (short-lived)
    evidenceURLs := make([]SignedURL, 0)
    for _, session := range sessions {
        if session.EvidenceURL != nil {
            url, err := r2Client.GenerateSignedURL(*session.EvidenceURL, 5*time.Minute)
            if err != nil {
                // Log but continue - don't fail entire export
                logger.Warn("Failed to generate signed URL", "session_id", session.ID)
                continue
            }
            evidenceURLs = append(evidenceURLs, SignedURL{
                URL:       url,
                ExpiresAt: time.Now().Add(5 * time.Minute),
            })
        }
    }
    
    // 4. Create export package
    export := &VerificationExport{
        UserID:       userID,
        Sessions:     sessions,
        Events:       events,
        EvidenceURLs: evidenceURLs,
        GeneratedAt:  time.Now(),
        RequestID:    generateRequestID(),
        ExpiresAt:    time.Now().Add(7 * 24 * time.Hour), // Keep for 7 days
    }
    
    // 5. Store export metadata for audit trail
    err = exportRepo.StoreMetadata(export)
    if err != nil {
        return nil, err
    }
    
    // 6. Log export for compliance audit
    auditLogger.LogExport(userID, export.RequestID, "GDPR_request")
    
    return export, nil
}
```

---

## Cost Optimization

### Smart Provider Routing Engine

```go
// Multi-factor provider selection with cost optimization
type RoutingEngine struct {
    providers map[string]ProviderStats
    costDB    CostDatabase
    geoIP     GeoIPService
}

type ProviderStats struct {
    SuccessRate   float64   // Last 24 hours
    AvgLatency    float64   // milliseconds
    AvgCost       float64   // USD
    Coverage      map[string]bool // Country coverage
    LastUpdated   time.Time
}

func (e *RoutingEngine) SelectProvider(ctx context.Context, userID uuid.UUID, country string, qualityScore float64) (string, error) {
    availableProviders := e.GetAvailableProviders(country)
    
    if len(availableProviders) == 0 {
        return "", ErrNoProviderAvailable
    }
    
    // Calculate scores for each provider
    scores := make(map[string]float64)
    for _, provider := range availableProviders {
        stats := e.providers[provider]
        
        // Base score components (0-1 each)
        costScore := 1.0 - (stats.AvgCost / 5.0) // Normalize to $5 max
        successScore := stats.SuccessRate
        latencyScore := 1.0 - (stats.AvgLatency / 10000.0) // Normalize to 10s max
        
        // Quality-based weighting
        // High quality images can use cheaper providers
        var weightCost, weightSuccess float64
        if qualityScore > 0.8 {
            weightCost = 0.6    // Prefer cheaper for high quality
            weightSuccess = 0.4
        } else if qualityScore > 0.5 {
            weightCost = 0.4
            weightSuccess = 0.6 // Prefer reliable for medium quality
        } else {
            weightCost = 0.2
            weightSuccess = 0.8 // Prefer reliable for low quality
        }
        
        // Calculate weighted score
        score := (costScore * weightCost) + 
                (successScore * weightSuccess) + 
                (latencyScore * 0.2) // 20% weight for latency
        
        scores[provider] = score
    }
    
    // Select highest scoring provider
    var bestProvider string
    bestScore := -1.0
    for provider, score := range scores {
        if score > bestScore {
            bestScore = score
            bestProvider = provider
        }
    }
    
    return bestProvider, nil
}

// Cost-aware retry logic
func (e *RoutingEngine) ShouldRetry(session *VerificationSession, failureCode string) (bool, string) {
    // Don't retry certain failure types
    nonRetryable := []string{FailureDocumentExpired, FailureSuspectedFraud}
    if slices.Contains(nonRetryable, failureCode) {
        return false, ""
    }
    
    // Check retry count
    if session.RetryCount >= 2 {
        return false, ""
    }
    
    // Check if cheaper alternative exists
    currentProvider := session.Provider
    if currentProvider != nil {
        alternatives := e.GetCheaperAlternatives(*currentProvider, session.ProviderCountry)
        if len(alternatives) > 0 {
            // Select cheapest alternative that meets minimum success rate
            for _, alt := range alternatives {
                if e.providers[alt].SuccessRate > 0.8 {
                    return true, alt
                }
            }
        }
    }
    
    return false, ""
}
```

### Cost Monitoring Dashboard

| Metric | Calculation | Target | Alert Threshold |
|--------|-------------|--------|-----------------|
| Cost per verification | Total cost / Total verifications | < $2.50 | > $3.50 |
| Monthly verification cost | Sum of all provider costs | < $10,000 | > $15,000 |
| Cost per successful verification | Cost / Successful verifications | < $2.00 | > $3.00 |
| Provider cost variance | (Max cost - Min cost) / Avg cost | < 20% | > 50% |
| Failed verification cost | Cost of failed verifications / Total cost | < 10% | > 20% |
| Quality-based cost savings | Cost with routing vs without | > 15% savings | < 5% savings |

### Optimization Strategies

1. **Geographic Routing Table**

   ```go
   var providerRouting = map[string][]string{
       "NG": {"dojah", "veriff"},          // Nigeria: Dojah first (cheaper)
       "KE": {"dojah", "veriff"},          // Kenya: Dojah if supported
       "GH": {"dojah", "veriff"},          // Ghana: Dojah if supported
       "US": {"veriff", "dojah"},          // US: Veriff only (Dojah may not support)
       "GB": {"veriff"},                   // UK: Veriff only
       "*":  {"veriff"},                   // Default: Veriff global
   }
   ```

2. **Time-Based Optimization**

   ```go
   // Run expensive verifications during off-peak hours
   func GetProcessingPriority(session *VerificationSession) string {
       hour := time.Now().UTC().Hour()
       
       // Peak hours (9 AM - 5 PM UTC): Process high-quality quickly
       if hour >= 9 && hour <= 17 {
           if session.QualityScore != nil && *session.QualityScore > 0.8 {
               return "high"
           }
           return "normal"
       }
       
       // Off-peak: Process everything, including retries
       return "normal"
   }
   ```

3. **Volume Discount Tracking**

   ```go
   // Track volume for discount negotiations
   type ProviderVolume struct {
       Provider    string
       Month       time.Month
       Year        int
       Count       int
       TotalCost   float64
       AvgCost     float64
   }
   
   func CheckVolumeDiscount(provider string, count int) float64 {
       volume := getMonthlyVolume(provider)
       projected := volume.Count + count
       
       // Negotiated discounts
       if projected > 10000 {
           return 0.20 // 20% discount
       } else if projected > 5000 {
           return 0.10 // 10% discount
       } else if projected > 1000 {
           return 0.05 // 5% discount
       }
       return 0.0
   }
   ```

---

## Open Questions (Resolved)

| Question | Resolution in v5.0 | Implementation |
|----------|-------------------|----------------|
| **Async vs Sync architecture?** | **Hybrid Approach:** Sync submission, async provider processing with webhooks | Webhook handlers + status polling fallback |
| **Provider selection for non-Nigeria?** | **Veriff for global coverage** with Dojah for supported African countries | Country-based routing in provider service |
| **Manual review workflow?** | **User-triggered retry first**, then manual review after failures | Automatic manual review after 3 failures |
| **Data retention policy?** | **90 days for evidence** (R2 lifecycle), metadata kept longer | R2 lifecycle rules + anonymization after retention |
| **Concurrency control?** | **Partial unique index** prevents multiple active sessions | Database constraint + application logic |
| **Replay attack protection?** | **Token usage tracking** with atomic updates | `token_used_at` column + Redis cache |
| **Evidence integrity?** | **SHA-256 hashing** with chain verification | Store hash, verify on download, tamper detection |
| **Provider failure handling?** | **Circuit breaker pattern** + automatic fallback | Redis-based circuit breaker with configurable thresholds |
| **Data consistency?** | **Reconciliation service** with cron job | Hourly job fixes profile-verification drift |
| **Monitoring gaps?** | **Comprehensive metrics** + business KPIs | OpenTelemetry integration with 4 golden signals |
| **Compliance requirements?** | **GDPR/NDPR specific implementations** | Right to delete, data export, consent tracking |
| **Rate limiting?** | **Multi-dimensional rate limiting** | User + IP + country + phone limits |
| **Disaster recovery?** | **Manual fallback workflow** + admin portal | Automatic escalation to manual review |
| **Cost optimization?** | **Smart routing engine** with quality-based weights | Multi-factor scoring with cost awareness |
| **Performance bottlenecks?** | **Enhanced indexes** + Redis caching | Composite indexes + query optimization |

---

## Revision History

| Version | Date | Author | Key Changes |
|---------|------|--------|-------------|
| 5.0 | 2026-01-05 | Backend Team | Production hardening: security enhancements, monitoring, disaster recovery, cost optimization |
| 4.0 | 2026-01-04 | Backend Team | Async webhook architecture, quality pre-check, concurrency control |
| 3.0 | 2026-01-04 | Backend Team | Micro-frontend handshake, no profile changes, resource gate integration |
| 2.0 | 2026-01-01 | Backend Team | Complete architecture rewrite based on micro-frontend approach |
| 1.0 | 2025-12-15 | Backend Team | Initial draft |

---

**End of Document**
