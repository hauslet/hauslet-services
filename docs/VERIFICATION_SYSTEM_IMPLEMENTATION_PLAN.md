# Verification System Implementation Plan (ID + Phone)

Status: Draft
Owner: Backend
Scope: Server-side only (client implementation is out of scope)

## Goals
- Provide a secure, auditable verification pipeline for ID verification and phone verification.
- Support the "KYC handshake" flow for the verification micro-frontend without sharing cookies/localStorage across domains.
- Update `profile` verification fields in a consistent, service-layer-first way.
- Integrate cleanly with existing supply-side access gating.

## Non-Goals
- Client-side verification UI and camera/liveness logic.
- Full vendor-specific KYC UX or SDK embedding.
- Reworking existing auth email verification flow.

## Current State (Relevant Code)
- `internal/modules/profile/domain/profile.go` already has `PhoneVerified`, `IDVerified`, `VerificationLevel`, `VerificationDate`.
- `internal/modules/profile/service/profile_verification.go` exposes `SetVerificationStatus`.
- Supply access gate checks `profile.IDVerified` for supply-side actions.
- Payments/finance domain already has resource types for verification (`id_verification`).

## Proposed Module: `internal/modules/verification`
Add a dedicated module with clear domain boundaries to avoid coupling verification workflows to auth/profile/service internals.

### Domain Model (examples)
- `VerificationSession`
  - `ID`, `UserID`, `Method` (enum), `Status` (enum), `TokenHash`, `ExpiresAt`
  - `RedirectURL`, `ReturnURL`, `Provider` (optional), `Attempts`
  - `CreatedAt`, `UpdatedAt`
- `VerificationMethod`
  - `id_liveness`, `id_document`, `phone_otp`
- `VerificationStatus`
  - `pending`, `in_review`, `verified`, `failed`, `expired`, `cancelled`
- `VerificationEvent`
  - `SessionID`, `Type`, `Payload`, `CreatedAt` (audit trail)

### Ports & Adapters
- `port/http`: public endpoints for handshake, evidence submission, and provider webhooks.
- `port/worker`: NATS consumers for async verification processing.
- Optional `port/graphql`: status queries (e.g., `myVerificationStatus`) if needed later.

## Persistence
### Tables (GORM + migrations)
- `verification_sessions`
  - `id`, `user_id`, `method`, `status`, `token_hash`, `expires_at`, `return_url`, `provider`, `attempts`, timestamps.
  - Indexes on `user_id`, `status`, `expires_at`.
- `verification_events`
  - `id`, `session_id`, `type`, `payload`, timestamps.
  - Index on `session_id`.

### Token Storage
- Generate a random token (32+ bytes), store only `SHA256(token)` as `token_hash`.
- Token TTL ~10 minutes; invalidate on first use.
- Optional Redis cache for token lookups (keep DB as source of truth).

## Core Workflows

### 1) ID Verification Handshake (Micro-frontend)
1. Main app calls `POST /verification/sessions` (auth required).
2. Backend creates `verification_session` with `status=pending`, returns:
   - `token` (plaintext, one-time)
   - `verification_url` (`https://verification.hauslet.com?token=...`)
   - `return_url` (for redirect after completion)
3. Verification app calls `GET /verification/sessions/validate?token=...`.
4. Backend validates token -> returns session info and allowed actions.
5. Verification app uploads evidence: `POST /verification/sessions/{id}/submit` with `token` + payload.
6. Backend stores evidence (R2) and publishes `verification.id.submitted` to NATS.
7. Worker calls provider (e.g., Dojah), then updates session status:
   - On success: mark `verified`, update profile `IDVerified=true`, `VerificationLevel=identity`, `VerificationDate=now`.
   - On failure: mark `failed`, keep profile unchanged.
8. Optional webhook `POST /webhooks/verification/provider` for provider callbacks.

### 2) Phone Verification (OTP)
1. Main app calls `POST /verification/phone/start` with phone number.
2. Backend generates OTP (or reuses auth OTP logic) and sends via SMS provider.
3. User submits OTP: `POST /verification/phone/verify`.
4. Backend verifies OTP, updates profile:
   - `PhoneVerified=true`, optionally add badge `phone_verified`.
   - Update `VerificationLevel` if applicable (e.g., `phone`).

## Integration Points

### Profile Module
- Verification service calls `ProfileService.SetVerificationStatus` for ID verification.
- For phone verification: either use `SetVerificationStatus` with `level="phone"` or a dedicated `SetPhoneVerified` helper.
- Preserve trust score updates via profile service.

### Access Gate
- Supply-side gate already checks `profile.IDVerified`. No changes required, but verify behavior for:
  - `pending`/`in_review` sessions (should not grant access).
  - Admin/support bypass remains.

### Notifications
- Add SMS provider interface under `internal/platform/notifications` or a new `internal/modules/notifications` adapter.
- Use async worker for SMS delivery to avoid blocking API.

### Storage
- Store evidence (selfie, ID document) in R2 with restricted access.
- Keep only object keys in session metadata.

### Payments (optional)
- If charging for verification, create a payment with `ResourceTypeIDVerification`.

## API Surface (HTTP)
- `POST /verification/sessions`
  - Auth required, returns token + redirect URL.
- `GET /verification/sessions/validate?token=...`
  - Public, token required.
- `POST /verification/sessions/{id}/submit`
  - Public, token required. Stores evidence + enqueues worker job.
- `POST /verification/phone/start`
  - Auth required; sends OTP.
- `POST /verification/phone/verify`
  - Auth required; verifies OTP.
- `POST /webhooks/verification/provider`
  - Public, HMAC signature required.

## NATS Subjects & Workers
- `verification.id.submitted` -> worker sends evidence to provider.
- `verification.id.completed` -> worker updates profile + session.
- Add retry logic for transient provider failures.

## Security & Compliance
- HMAC signatures for provider webhooks.
- Rate limit session creation and OTP requests (Redis).
- Short TTL for verification tokens; single-use.
- Encrypt sensitive payloads at rest where possible.
- Audit all state transitions in `verification_events`.

## Testing Strategy
- Unit tests for session creation, token validation, OTP flow.
- Integration tests for webhook processing and profile updates.
- Negative tests: expired token, reused token, invalid OTP.

## Rollout Plan
1. Ship module + DB migrations, endpoints behind feature flag.
2. Enable ID verification for internal users only.
3. Gradually roll out to supply-side users.
4. Add monitoring/alerts on provider error rates and webhook failures.

## Open Questions
- Which SMS provider will be used?
- Do we require ID verification for hosts only, or also for agents/landlords?
- Should verification data be retained permanently or for a fixed retention window?
