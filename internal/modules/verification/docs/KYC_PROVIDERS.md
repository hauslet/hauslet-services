# KYC Providers

The verification module uses two KYC providers selected automatically based on
the session's `country` field.

## Provider Selection

| Region | Provider | Coverage |
|--------|----------|----------|
| Nigeria (NG) | **Dojah** | NIN, BVN, VIN, drivers license, voters card, CAC |
| Ghana (GH), Kenya (KE), South Africa (ZA) | **Dojah** | Photo ID + liveness |
| All other countries | **Veriff** | Passport, drivers license, national ID |

Selection is handled by the `ProviderFactory` in `internal/platform/kyc/factory.go`.
The factory reads the country code and returns the appropriate adapter.

---

## Dojah

**Base URL:** `https://api.dojah.io`

**Authentication:**
- `Authorization: {secretKey}`
- `AppId: {appID}`

### Supported Operations

#### 1. Photo ID Verification

Endpoint: `POST /api/v1/kyc/photo_id`

Submits a selfie + document image for face comparison and document validation.
The response is **synchronous** — the decision comes in the HTTP response.

#### 2. Government Data Lookup

| Document Type | Endpoint |
|--------------|----------|
| NIN | `GET /api/v1/kyc/nin?nin={number}` |
| BVN | `GET /api/v1/kyc/bvn/full?bvn={number}` |
| VIN | `GET /api/v1/kyc/vin?vin={number}` |
| Drivers License | `GET /api/v1/kyc/dl?license_number={number}` |

These endpoints return rich identity data (name, DOB, gender, photo, address)
from Nigeria's government databases. The module:

1. Extracts the returned entity fields into an `ExtractedData` map
2. Cross-validates the first/last name against the applicant's submitted name
3. Returns `success` if both first and last names match

#### 3. CAC Business Lookup

Endpoint: `GET /api/v1/kyc/cac/advance?rc_number={rcNumber}&company_type={type}`

Returns company registration details from the Corporate Affairs Commission.

**Company Types:** `BUSINESS_NAME`, `COMPANY`, `INCORPORATED_TRUSTEES`,
`LIMITED_PARTNERSHIP`, `LIMITED_LIABILITY_PARTNERSHIP`

**Response Data:**
- Company name, RC number, type
- Registration date, status
- Address, state, city, LGA
- Email
- Affiliates (directors/proprietors with names and positions)

---

## Veriff

**Base URL:** `https://stationapi.veriff.com`

**Authentication:**
- `X-AUTH-CLIENT: {publicKey}`
- `X-HMAC-SIGNATURE: HMAC-SHA256(payload, secretKey)`

### Supported Operations

#### 1. Create Session

Endpoint: `POST /v1/sessions`

Creates a Veriff verification session. The response includes a `sessionUrl` that
can be used for redirect-based verification, or the images can be submitted via API.

#### 2. Upload Media

Endpoint: `POST /v1/sessions/{sessionId}/media`

Uploads selfie and document images to an existing Veriff session.

#### 3. Submit for Processing

After uploading media, the session is submitted for processing. Veriff processes
asynchronously and sends the result via webhook.

### Webhook Flow

Veriff sends webhook callbacks when verification is complete:

```
Veriff → POST /webhooks/verification/veriff → WebhookHandler
```

The webhook payload is verified using HMAC-SHA256 signature validation.

---

## Provider Adapter Interface

Both providers implement the `KYCProvider` interface:

```go
type KYCProvider interface {
    SubmitVerification(ctx context.Context, req *VerificationRequest) (*VerificationResponse, error)
    ParseWebhook(payload []byte, headers map[string]string) (*WebhookEvent, error)
    VerifySignature(payload []byte, signature string) bool
    EstimateCost(country string) (float64, error)
    VerifyBusiness(ctx context.Context, registrationNumber, businessType string) (*BusinessVerificationResponse, error)
}
```

The `Client` struct wraps `ProviderFactory` and adds country-based routing:

```go
client.SubmitVerification(ctx, req)  // routes to Dojah or Veriff
client.VerifyBusiness(ctx, "NG", "RC123456", "COMPANY")  // routes to Dojah
```

---

## Cost Configuration

Per-country verification costs are configured in `config/defaults/kyc.yaml`:

```yaml
kyc:
  costs:
    dojah:
      NG: 0.15
      GH: 0.50
      KE: 0.50
      ZA: 0.50
      default: 1.00
    veriff:
      US: 2.50
      GB: 2.50
      EU: 2.00
      default: 3.00
```

These costs are used for billing estimation when submitting verifications.
