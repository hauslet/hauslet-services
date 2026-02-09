# Verification Module

The verification module handles all identity, phone, address, business, and listing
verification flows for the Hauslet platform. It supports automated KYC via external
providers (Dojah for Africa, Veriff for global coverage) and manual document review.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [REST API Reference](./REST_API.md)
- [Verification Flows](./FLOWS.md)
- [KYC Providers](./KYC_PROVIDERS.md)
- [Webhooks](./WEBHOOKS.md)
- [Domain Model](./DOMAIN.md)

## Architecture Overview

The module follows the hexagonal (ports & adapters) architecture used throughout Hauslet:

```txt
┌──────────────────────────────────────────────────────┐
│                   Port Layer (Inbound)               │
│  ┌──────────────────┐  ┌──────────────────────────┐  │
│  │   HTTP Handler   │  │   Webhook Handler        │  │
│  │  (REST API)      │  │  (Dojah/Veriff callbacks)│  │
│  └────────┬─────────┘  └──────────┬───────────────┘  │
│           │                       │                  │
├───────────▼───────────────────────▼──────────────────┤
│                 Service Layer                        │
│  ┌──────────────────────────────────────────────┐    │
│  │          VerificationService                 │    │
│  │  identity · phone · address · business ·     │    │
│  │  listing · session · evidence · webhook ·    │    │
│  │  admin · manual_review                       │    │
│  └───────────┬──────────────────────┬───────────┘    │
│              │                      │                │
├──────────────▼──────────────────────▼────────────────┤
│          Adapter Layer (Outbound)                    │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────┐  │
│  │KYC Client│ │  Repo    │ │ Evidence │ │Notifier│  │
│  │Dojah/    │ │(Postgres)│ │  Store   │ │(SMS/   │  │
│  │Veriff    │ │          │ │ (GCS)    │ │Queue)  │  │
│  └──────────┘ └──────────┘ └──────────┘ └────────┘  │
└──────────────────────────────────────────────────────┘
```

### Directory Structure

```txt
internal/modules/verification/
├── docs/               # This documentation
├── domain/             # Pure domain model (no external dependencies)
│   ├── enums.go        # VerificationType, SessionStatus, DocumentType, etc.
│   ├── session.go      # VerificationSession aggregate root
│   ├── attempt.go      # VerificationAttempt value object
│   ├── evidence.go     # Evidence entity (uploaded documents)
│   ├── errors.go       # Domain error sentinels
│   ├── value_objects.go
│   └── verification_data.go  # Type-specific data (IdentityData, PhoneData, etc.)
├── service/            # Business logic
│   ├── interface.go    # VerificationService interface
│   ├── service.go      # Constructor + request/response DTOs
│   ├── session.go      # Session CRUD operations
│   ├── identity.go     # KYC identity flow (Dojah/Veriff)
│   ├── phone.go        # OTP phone verification
│   ├── address.go      # Address document review
│   ├── business.go     # Business verification (CAC + manual)
│   ├── listing.go      # Listing document review
│   ├── manual_review.go # Shared manual review helper
│   ├── evidence.go     # Evidence upload/retrieval
│   ├── webhook.go      # Provider webhook processing
│   ├── admin.go        # Admin approval/expiry operations
│   └── helpers.go      # Rate limiting, KYC request building, OTP
├── port/
│   └── http/
│       ├── handler.go         # REST API handler (all endpoints)
│       ├── webhook_handler.go # Webhook receiver (Dojah/Veriff)
│       ├── process.go         # Webhook processing logic
│       └── helpers.go         # Signature extraction
├── repository/
│   ├── interface.go    # VerificationRepo interface
│   └── mapper.go       # GORM ↔ domain mapping
├── notification/
│   └── service.go      # SMS/queue notification adapter
└── templates/          # Email/SMS templates
```

## Quick Start

All verification endpoints require authentication via Bearer token. The typical
flow is:

1. **Create a session** → `POST /verification/sessions`
2. **Submit verification** → `POST /verification/{type}` (type-specific endpoint)
3. **Check status** → `GET /verification/sessions/{id}`

See [REST API Reference](./REST_API.md) for complete endpoint documentation and
[Verification Flows](./FLOWS.md) for detailed flow diagrams.
