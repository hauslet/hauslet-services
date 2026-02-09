# Verification Flows

This document describes the complete user-facing flows for each verification type.
Every flow starts with session creation and ends with an approved or rejected session.

---

## Session Lifecycle

All verification types share the same session state machine:

```txt
                ┌──────────┐
                │ pending  │ ← session created
                └────┬─────┘
                     │ user submits verification
                     ▼
               ┌───────────┐
               │in_progress│ ← provider processing / manual review
               └─────┬─────┘
                     │
          ┌──────────┼──────────┐
          ▼          ▼          ▼
    ┌──────────┐ ┌──────────┐ ┌──────────┐
    │ approved │ │ rejected │ │ expired  │
    └──────────┘ └──────────┘ └──────────┘
         (final states — no further transitions)
```

Sessions have a configurable expiry (default 24h). The `ExpireOldSessions` admin
job runs periodically to transition stale sessions to `expired`.

Each session tracks:

- `attemptsUsed` / `maxAttempts` — users get up to 3 attempts per session
- `rejectionReason` / `rejectionNotes` — if rejected, explains why

---

## 1. Identity Verification (KYC)

The identity flow uses automated KYC providers. Dojah handles African countries;
Veriff covers the rest of the world.

### Photo ID Mode (passport, drivers_license, national_id, etc.)

```txt
Client                         API                          KYC Provider
  │                              │                              │
  │ POST /verification/sessions  │                              │
  │ type=identity, tier=standard │                              │
  │─────────────────────────────>│                              │
  │         201 { session }      │                              │
  │<─────────────────────────────│                              │
  │                              │                              │
  │ POST /verification/identity  │                              │
  │ { selfie + document image }  │                              │
  │─────────────────────────────>│  SubmitVerification(...)     │
  │                              │─────────────────────────────>│
  │                              │    { decision: approved }    │
  │                              │<─────────────────────────────│
  │   200 { status: approved }   │                              │
  │<─────────────────────────────│                              │
```

For Dojah, photo IDs are submitted synchronously — the response comes inline.
For Veriff, a webhook callback arrives asynchronously.

### Government Data Mode (NIN, BVN, VIN)

When the document type is `nin`, `bvn`, or `vin`, Dojah performs a **government
database lookup** instead of photo ID comparison. No selfie is required for matching
but is still uploaded as evidence. The system:

1. Queries Dojah's `/api/v1/kyc/{type}` endpoint with the document number
2. Extracts rich identity data (full name, DOB, gender, address, photo URL)
3. Cross-validates the returned name against the applicant's submitted name
4. Auto-approves if names match, rejects if they don't

### Extracted Data

On success, the system parses and stores the following from provider responses:

| Field | Source |
|-------|--------|
| `full_name` | NIN/BVN entity response |
| `first_name`, `last_name`, `middle_name` | Parsed from entity |
| `date_of_birth` | Entity DOB field |
| `gender` | Entity gender field |
| `phone_number` | Entity phone (if available) |
| `address` | Entity residential address |
| `photo_url` | Entity image/photo URL |
| `nationality` | Entity nationality |

---

## 2. Phone Verification (OTP)

Phone verification uses a standard OTP flow via SMS.

```
Client                         API                      SMS Provider
  │                              │                           │
  │ POST /verification/sessions  │                           │
  │ type=phone, data.phone=...   │                           │
  │─────────────────────────────>│                           │
  │         201 { session }      │                           │
  │<─────────────────────────────│                           │
  │                              │                           │
  │ POST /verification/phone/otp │                           │
  │ { session_id }               │                           │
  │─────────────────────────────>│  Send SMS(code)           │
  │                              │──────────────────────────>│
  │  200 { otpSent: true }       │                           │
  │<─────────────────────────────│                           │
  │                              │                           │
  │  User receives SMS           │                           │
  │                              │                           │
  │ POST /verification/phone/verify                          │
  │ { session_id, otp_code }     │                           │
  │─────────────────────────────>│                           │
  │  200 { verified: true }      │                           │
  │<─────────────────────────────│                           │
```

**OTP Details:**

- 6-digit cryptographically secure code
- Stored in Redis with a 5-minute TTL (`verification:otp:{sessionID}`)
- Maximum 3 verification attempts per OTP generation
- Can regenerate OTP (resets attempt counter)

---

## 3. Address Verification (Manual Review)

Address verification requires uploading proof documents that an admin reviews.

```txt
Client                         API                      Admin
  │                              │                        │
  │ POST /verification/sessions  │                        │
  │ type=address, data.address=..│                        │
  │─────────────────────────────>│                        │
  │         201 { session }      │                        │
  │<─────────────────────────────│                        │
  │                              │                        │
  │ POST /verification/address   │                        │
  │ { proof_document, type }     │                        │
  │─────────────────────────────>│                        │
  │  200 { status: pending }     │  Notification →        │
  │<─────────────────────────────│───────────────────────>│
  │                              │                        │
  │                              │  Admin reviews & approves
  │                              │<───────────────────────│
  │                              │                        │
  │ GET /verification/sessions/{id}                       │
  │─────────────────────────────>│                        │
  │  200 { status: approved }    │                        │
  │<─────────────────────────────│                        │
```

**Accepted Document Types:**

- `utility_bill` — Recent electricity/water/gas bill
- `bank_statement` — Bank statement showing address
- `lease` — Rental agreement or lease

---

## 4. Business Verification (CAC + Manual Review)

Business verification supports automated lookup for Nigerian businesses and
falls back to manual review for other countries or if the lookup fails.

### Nigerian Businesses (Automated CAC Lookup)

```txt
Client                         API                      Dojah CAC
  │                              │                         │
  │ POST /verification/sessions  │                         │
  │ type=business, country=NG    │                         │
  │ data.business.rc_number=...  │                         │
  │─────────────────────────────>│                         │
  │         201 { session }      │                         │
  │<─────────────────────────────│                         │
  │                              │                         │
  │ POST /verification/business  │                         │
  │ { registration_document }    │                         │
  │─────────────────────────────>│  GET /kyc/cac/advance   │
  │                              │ ?rc_number=...          │
  │                              │────────────────────────>│
  │                              │  { entity: {...} }      │
  │                              │<────────────────────────│
  │                              │                         │
  │                              │  Name matches + Active?
  │                              │  → Auto-approve         │
  │  200 { status: approved }    │                         │
  │<─────────────────────────────│                         │
```

**CAC Company Type Mapping:**

| Business Type Input | Dojah company_type |
|--------------------|--------------------|
| `sole_proprietorship` | `BUSINESS_NAME` |
| `llc`, `limited_company` | `COMPANY` |
| `ngo`, `nonprofit`, `trust` | `INCORPORATED_TRUSTEES` |
| `lp`, `limited_partnership` | `LIMITED_PARTNERSHIP` |
| `llp` | `LIMITED_LIABILITY_PARTNERSHIP` |
| Default | `BUSINESS_NAME` |

**Auto-Rejection Reasons:**

- Business not found in CAC registry
- Company name does not match (shows both names)
- Business status is not "Active"

### Non-Nigerian Businesses (Manual Review)

Falls through to the same manual review flow as address verification — documents
are uploaded, evidence is stored, and an admin reviews.

---

## 5. Listing Verification (Manual Review)

Listing verification proves property ownership or access. The flow is identical
to address verification but with listing-specific document types.

**Accepted Document Types:**

- `title_deed` — Certificate of Occupancy or title deed
- `property_tax` — Property tax receipt
- `geo_tagged_photo` — Geotagged photo at the property

---

## Evidence Storage

All uploaded documents are:

1. Hashed (SHA-256) for integrity verification
2. Stored in Google Cloud Storage with unique keys
3. Accessible only via time-limited signed URLs
4. Linked to the session's attempt record

Evidence metadata includes:

- `type` — What kind of evidence (selfie, id_document, utility_bill, etc.)
- `mimeType` — Detected content type
- `hash` — SHA-256 integrity hash
- `storagePath` — GCS object path
- `uploadedAt` — Timestamp
- `ipAddress` — Uploader's IP (for audit)

---

## Webhook Processing

KYC provider webhooks (Dojah, Veriff) arrive at dedicated endpoints handled by
the `WebhookHandler`. See [Webhooks](./WEBHOOKS.md) for details.
