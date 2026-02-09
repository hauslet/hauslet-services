# Webhooks

The verification module receives asynchronous callbacks from KYC providers (Dojah
and Veriff) when verification processing completes. These webhooks are handled by
the `WebhookHandler` which is separate from the main REST `HTTPHandler`.

## Webhook Endpoints

| Provider | Endpoint | Auth |
|----------|----------|------|
| Dojah | `POST /webhooks/verification/dojah` | `x-dojah-signature` header |
| Veriff | `POST /webhooks/verification/veriff` | `x-hmac-signature` header |

These endpoints are **not** behind user authentication. They are protected by
provider-specific signature verification.

## Rate Limits

Webhook endpoints have a higher rate limit than user-facing endpoints:
**100 requests/min per IP** in production.

## Processing Flow

```
Provider                    WebhookHandler              VerificationService
   │                             │                              │
   │  POST /webhooks/...         │                              │
   │  { payload + signature }    │                              │
   │────────────────────────────>│                              │
   │                             │  1. Read body                │
   │                             │  2. Extract signature        │
   │                             │  3. Collect headers          │
   │                             │                              │
   │                             │  ProcessWebhook(req)         │
   │                             │─────────────────────────────>│
   │                             │                              │
   │                             │  a. Verify signature         │
   │                             │  b. Parse webhook payload    │
   │                             │  c. Find matching session    │
   │                             │  d. Update attempt status    │
   │                             │  e. Approve/reject session   │
   │                             │  f. Notify downstream        │
   │                             │                              │
   │                             │         nil / error          │
   │                             │<─────────────────────────────│
   │     200 { status: received }│                              │
   │<────────────────────────────│                              │
```

## Signature Verification

### Dojah

Dojah sends a signature in one of two headers:

- `x-dojah-signature` (v1)
- `x-dojah-signature-v2` (v2)

The adapter verifies the signature by computing an HMAC of the payload using the
configured secret key and comparing.

### Veriff

Veriff sends an HMAC-SHA256 signature in the `x-hmac-signature` header. The
adapter computes the expected signature using the secret key and verifies it.

## Error Handling

- **Invalid signature** → Returns `401 Unauthorized`
- **Processing error** → Returns `200 OK` (to prevent provider retries) but logs
  the error as a warning
- **Successful processing** → Returns `200 OK` with `{"status": "received"}`

Returning 200 even on business logic errors is intentional — it prevents providers
from retrying webhooks that will consistently fail (e.g., session not found, already
in final state).

## Downstream Notifications

When a webhook results in session approval, the service notifies downstream modules:

- **Identity verification approved** → Updates user profile verification status
- **Business verification approved** → Updates business entity verification status

These notifications are sent via the `notifyVerificationSuccess` method which
dispatches to the appropriate module adapter (Profile, Business).
