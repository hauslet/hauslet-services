# REST API Reference

All endpoints are prefixed with `/verification` and require a valid Bearer token
in the `Authorization` header. Responses use standard JSON with consistent error
format.

## Authentication

Every request must include:

```
Authorization: Bearer <jwt_token>
```

The authenticated user ID is extracted automatically from the token.

## Error Response Format

All errors return this structure:

```json
{
  "error": "Bad Request",
  "message": "Invalid session_id",
  "field": "session_id"
}
```

| HTTP Code | Meaning |
|-----------|---------|
| 400 | Validation error / bad input |
| 401 | Missing or invalid token |
| 403 | Not authorized (e.g. wrong user for session) |
| 404 | Session / evidence not found |
| 409 | Session in invalid state for this operation |
| 410 | Session has expired |
| 429 | Rate limit / max attempts exceeded |
| 500 | Internal server error |

---

## Rate Limits (Production)

| Category | Limit |
|----------|-------|
| Submit endpoints (identity, address, business, listing) | 10 requests/min per IP |
| OTP endpoints (generate, verify) | 5 requests/min per IP |
| Read endpoints (sessions, evidence, attempts) | 60 requests/min per IP |

---

## Endpoints

### Session Management

#### Create Session

```
POST /verification/sessions
```

Creates a new verification session. This is the first step in any verification flow.

**Request Body:**

```json
{
  "type": "identity",
  "tier": "standard",
  "country": "NG",
  "target_id": "uuid-string (optional, for listing verification)",
  "ip_address": "1.2.3.4 (optional)",
  "user_agent": "Mozilla/... (optional)",
  "data": {
    // Type-specific data, see below
  }
}
```

**Verification Types:** `identity`, `phone`, `address`, `business`, `listing`

**Tiers:** `basic`, `standard`, `enhanced`

**Type-specific `data` payloads:**

For `identity`:

```json
{
  "identity": {
    "applicant_info": {
      "first_name": "John",
      "last_name": "Doe",
      "date_of_birth": "1990-01-15T00:00:00Z",
      "nationality": "NG"
    }
  }
}
```

For `phone`:

```json
{
  "phone": {
    "phone_number": "+2348012345678",
    "country_code": "NG"
  }
}
```

For `address`:

```json
{
  "address": {
    "full_address": "123 Example Street",
    "city": "Lagos",
    "state": "Lagos",
    "postal_code": "100001",
    "country": "NG"
  }
}
```

For `business`:

```json
{
  "business": {
    "business_name": "Acme Ltd",
    "registration_number": "RC123456",
    "business_type": "llc",
    "country": "NG",
    "business_address": {
      "full_address": "1 Marina",
      "city": "Lagos",
      "state": "Lagos",
      "country": "NG"
    }
  }
}
```

For `listing`:

```json
{
  "listing": {
    "listing_id": "uuid-string",
    "property_id": "uuid-string"
  }
}
```

**Response:** `201 Created`

```json
{
  "id": "uuid",
  "userId": "uuid",
  "type": "identity",
  "tier": "standard",
  "status": "pending",
  "country": "NG",
  "attemptsUsed": 0,
  "maxAttempts": 3,
  "createdAt": "2026-02-08T...",
  "expiresAt": "2026-02-09T..."
}
```

---

#### Get Session by ID

```
GET /verification/sessions/{sessionID}
```

Returns a session the authenticated user owns (or is admin).

**Response:** `200 OK` — `VerificationSession` object

---

#### Get Session by Type

```
GET /verification/sessions?type={verificationType}
```

Looks up the current user's active session for the given type.

**Query Parameters:**

| Name | Required | Values |
|------|----------|--------|
| `type` | Yes | `identity`, `phone`, `address`, `business`, `listing` |

**Response:** `200 OK` — `VerificationSession` object

---

### Identity Verification

#### Submit Identity Verification

```
POST /verification/identity
```

Submits a selfie and identity document for automated KYC processing. The system
selects the appropriate provider (Dojah for African countries, Veriff for others)
based on the session's `country` field.

**Request Body:**

```json
{
  "session_id": "uuid-string",
  "selfie_image": "base64-encoded-image",
  "document_image": "base64-encoded-image",
  "document_type": "passport",
  "document_number": "A12345678 (optional)",
  "ip_address": "1.2.3.4 (optional)"
}
```

**Document Types:** `passport`, `drivers_license`, `national_id`, `voters_card`,
`residence_permit`, `nin`, `bvn`, `vin`, `cac`

**Response:** `200 OK`

```json
{
  "attempt": { "id": "uuid", "status": "processing", ... },
  "session": { "id": "uuid", "status": "in_progress", ... },
  "providerName": "dojah",
  "status": "processing",
  "message": "Verification submitted successfully"
}
```

---

### Phone Verification

#### Generate OTP

```
POST /verification/phone/otp
```

Sends a one-time password to the phone number stored in the session.

**Request Body:**

```json
{
  "session_id": "uuid-string",
  "ip_address": "1.2.3.4 (optional)"
}
```

**Response:** `200 OK`

```json
{
  "session": { ... },
  "otpSent": true,
  "expiresAt": "2026-02-08T12:05:00Z",
  "smsProvider": "twilio",
  "message": "OTP sent successfully"
}
```

#### Verify OTP

```
POST /verification/phone/verify
```

Verifies the OTP code submitted by the user.

**Request Body:**

```json
{
  "session_id": "uuid-string",
  "otp_code": "123456",
  "ip_address": "1.2.3.4 (optional)"
}
```

**Response:** `200 OK`

```json
{
  "session": { ... },
  "verified": true,
  "remaining": 2,
  "message": "Phone verified successfully"
}
```

---

### Address Verification

#### Submit Address Verification

```
POST /verification/address
```

Submits an address proof document for manual review.

**Request Body:**

```json
{
  "session_id": "uuid-string",
  "proof_document": "base64-encoded-pdf-or-image",
  "document_type": "utility_bill",
  "ip_address": "1.2.3.4 (optional)"
}
```

**Document Types:** `utility_bill`, `bank_statement`, `lease`

**Response:** `200 OK` — `SubmitVerificationResponse`

---

### Business Verification

#### Submit Business Verification

```
POST /verification/business
```

Submits business documents. For **Nigerian businesses** (`country = "NG"`), the system
first attempts an automated CAC (Corporate Affairs Commission) lookup via Dojah. If
the lookup succeeds and business details match, the session is auto-approved. If the
CAC lookup fails or the country is not Nigeria, the submission falls through to
manual review.

**Request Body:**

```json
{
  "session_id": "uuid-string",
  "registration_document": "base64-encoded-pdf",
  "tax_id_document": "base64-encoded-pdf (optional)",
  "business_license_document": "base64-encoded-pdf (optional)",
  "ip_address": "1.2.3.4 (optional)"
}
```

**CAC Auto-Approval Criteria:**

1. Business found in CAC registry by RC number
2. Company name matches (case-insensitive)
3. Business status is "Active"

If any criterion fails, the session is rejected with a detailed reason.

**Response:** `200 OK` — `SubmitVerificationResponse`

---

### Listing Verification

#### Submit Listing Verification

```
POST /verification/listing
```

Submits listing ownership proof for manual review.

**Request Body:**

```json
{
  "session_id": "uuid-string",
  "proof_document": "base64-encoded-pdf-or-image",
  "document_type": "title_deed",
  "ip_address": "1.2.3.4 (optional)"
}
```

**Document Types:** `title_deed`, `property_tax`, `geo_tagged_photo`

**Response:** `200 OK` — `SubmitVerificationResponse`

---

### Evidence Management

#### List Session Evidence

```
GET /verification/sessions/{sessionID}/evidence
```

Returns all uploaded evidence items for a session.

**Response:** `200 OK` — `Evidence[]`

---

#### Get Evidence

```
GET /verification/evidence/{evidenceID}
```

Returns metadata for a specific evidence item.

**Response:** `200 OK` — `Evidence`

---

#### Generate Evidence Download URL

```
GET /verification/evidence/{evidenceID}/url
```

Generates a temporary signed URL to download the evidence file from cloud storage.

**Response:** `200 OK`

```json
{
  "url": "https://storage.googleapis.com/..."
}
```

---

### Attempts

#### List Attempts

```
GET /verification/sessions/{sessionID}/attempts
```

Returns all verification attempts for a session.

**Response:** `200 OK` — `VerificationAttempt[]`

```json
[
  {
    "id": "uuid",
    "sessionId": "uuid",
    "status": "success",
    "providerName": "dojah",
    "providerSessionId": "dojah-ref-123",
    "processingTimeMs": 2500,
    "createdAt": "2026-02-08T...",
    "completedAt": "2026-02-08T..."
  }
]
```
