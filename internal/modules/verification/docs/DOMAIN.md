# Domain Model

This document describes the core domain types used throughout the verification
module. All types are in `internal/modules/verification/domain/`.

---

## Enums

### VerificationType

The type of verification being performed.

| Value | Description |
|-------|-------------|
| `identity` | KYC identity verification (selfie + document) |
| `phone` | Phone number verification via OTP |
| `address` | Physical address proof verification |
| `business` | Business/entity registration verification |
| `listing` | Listing ownership proof verification |

### SessionStatus

The lifecycle state of a verification session.

| Value | Description | Final? |
|-------|-------------|--------|
| `pending` | Session created, awaiting submission | No |
| `in_progress` | Verification submitted, processing | No |
| `approved` | Verification passed | Yes |
| `rejected` | Verification failed | Yes |
| `expired` | Session timed out | Yes |

### AttemptStatus

The state of an individual verification attempt.

| Value | Description | Final? |
|-------|-------------|--------|
| `pending` | Queued for submission or manual review | No |
| `processing` | Submitted to KYC provider | No |
| `success` | Provider returned success | Yes |
| `failed` | Provider returned failure | Yes |

### VerificationTier

The level of verification required.

| Value | Description |
|-------|-------------|
| `basic` | Document + selfie only |
| `standard` | Basic + liveness check |
| `enhanced` | Standard + address verification |

### DocumentType

The type of identity document submitted.

| Value | Description | Provider |
|-------|-------------|----------|
| `passport` | International passport | Both |
| `drivers_license` | Driving license | Both |
| `national_id` | National identity card | Both |
| `voters_card` | Voter registration card | Dojah |
| `residence_permit` | Residence permit | Veriff |
| `nin` | Nigeria National ID Number (govt lookup) | Dojah |
| `bvn` | Bank Verification Number (govt lookup) | Dojah |
| `vin` | Voter ID Number (govt lookup) | Dojah |
| `cac` | CAC registration (business lookup) | Dojah |

### RejectionReason

Why a verification was rejected.

| Value | Description |
|-------|-------------|
| `document_not_readable` | Document image too blurry or unclear |
| `document_expired` | Document has passed its expiry date |
| `document_fraudulent` | Document appears tampered or fake |
| `selfie_mismatch` | Face in selfie doesn't match document |
| `liveness_failed` | Liveness check failed |
| `data_mismatch` | Submitted data doesn't match document |
| `age_requirement` | Applicant below minimum age |
| `blacklisted` | User on a sanctions/deny list |
| `invalid_address` | Address could not be verified |
| `business_not_found` | Business not found in registry |
| `sanctioned_country` | Country under sanctions |
| `duplicate_account` | User already verified with another account |
| `max_attempts_exceeded` | Used all verification attempts |
| `provider_error` | KYC provider returned an error |
| `insufficient_data` | Not enough data for verification |
| `other` | Other reason (see notes) |

---

## Core Entities

### VerificationSession

The aggregate root. Represents one verification process for a user.

```go
type VerificationSession struct {
    ID              uuid.UUID
    UserID          uuid.UUID
    Type            VerificationType
    Tier            VerificationTier
    Status          SessionStatus
    TargetID        *uuid.UUID          // e.g., listing ID for listing verification
    TargetType      TargetType          // e.g., "listing"
    Country         string              // ISO 3166-1 alpha-2
    Data            VerificationData    // Type-specific data (JSONB)
    AttemptsUsed    int
    MaxAttempts     int
    LastAttemptAt   *time.Time
    ApprovedAt      *time.Time
    RejectedAt      *time.Time
    RejectionReason *RejectionReason
    RejectionNotes  *string
    ExpiresAt       time.Time
    CompletedAt     *time.Time
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

**Key Methods:**

- `CanSubmitAttempt()` — checks max attempts, expiry, and status
- `StartAttempt()` — transitions status and increments counter
- `Approve()` — transitions to approved (if not already final)
- `Reject(reason, notes)` — transitions to rejected
- `GetIdentityData()` / `GetPhoneData()` / `GetBusinessData()` / etc. — type-safe accessors

### VerificationAttempt

Represents a single submission to a KYC provider or manual review queue.

```go
type VerificationAttempt struct {
    ID               uuid.UUID
    SessionID        uuid.UUID
    Status           AttemptStatus
    ProviderName     string           // "dojah", "veriff", "manual_review"
    ProviderSessionID *string
    EvidenceIDs      []uuid.UUID
    ProcessingTime   *time.Duration
    WebhookReceivedAt *time.Time
    Result           *VerificationResult
    CreatedAt        time.Time
    CompletedAt      *time.Time
}
```

### Evidence

An uploaded file linked to a verification attempt.

```go
type Evidence struct {
    ID          uuid.UUID
    SessionID   uuid.UUID
    UserID      uuid.UUID
    Type        EvidenceType        // selfie, id_document, utility_bill, etc.
    MimeType    string
    Hash        string              // SHA-256
    StoragePath string              // GCS path
    UploadedAt  time.Time
    IPAddress   *string
}
```

### VerificationData

Type-specific data stored as JSONB on the session. Only one field is populated
at a time, matching the session's `Type`.

```go
type VerificationData struct {
    Identity *IdentityData `json:"identity,omitempty"`
    Phone    *PhoneData    `json:"phone,omitempty"`
    Address  *AddressData  `json:"address,omitempty"`
    Business *BusinessData `json:"business,omitempty"`
    Listing  *ListingData  `json:"listing,omitempty"`
}
```

---

## Error Sentinels

All domain errors are defined as `errors.New(...)` sentinels in `errors.go`:

| Error | HTTP Status | Description |
|-------|-------------|-------------|
| `ErrSessionNotFound` | 404 | Session does not exist |
| `ErrSessionExpired` | 410 | Session has expired |
| `ErrInvalidSessionStatus` | 409 | Session in wrong state for operation |
| `ErrSessionTypeMismatch` | 400 | Wrong verification type for endpoint |
| `ErrMaxAttemptsExceeded` | 429 | All attempts used |
| `ErrUnauthorized` | 403 | User doesn't own this session |
| `ErrEvidenceNotFound` | 404 | Evidence item not found |
| `ErrRateLimitExceeded` | 429 | Too many requests |
