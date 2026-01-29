# Hauslet Services

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8.svg)](https://golang.org/)

Backend for **Hauslet**, a hybrid real estate platform (Zillow + Airbnb) for the Nigerian market. Modular monolith in Go with an async worker for heavy/background tasks.

## Architecture
- **Modular Monolith + Worker:** HTTP/GraphQL API (`cmd/api`) and a background worker (`cmd/worker`) sharing the same codebase and database. Async tasks go through Google Cloud Tasks.
- **Hexagonal (Ports & Adapters):** Domain modules live in `internal/modules/*` with adapters for HTTP/GraphQL, queue handlers, and external services.
- **Domain Modules:**
  - **Core:** `auth`, `profile`, `business`
  - **Property:** `property`, `calendar`
  - **Transactions:** `payments`, `finance`, `booking`, `payout`
  - **Monetization:** `promotions` (subscriptions & listing promotions)
  - **Social:** `wishlist`, `review`, `interactions`, `leads`
  - **Content:** `moderation`, `discovery`, `verification`
- **Platform Abstractions:** `internal/platform` for DB, queue (Cloud Tasks), storage (R2), AI providers, email, Redis, payment providers (Paystack), FX rates.
- **Async Jobs:** Email sending, media thumbnail/cleanup, AI moderation (Gemini primary, Anthropic fallback), payment webhook processing handled via Cloud Tasks.
- **Scheduled Jobs:** Cloud Scheduler triggers periodic tasks (cleanup, reconciliation, notifications).

## Tech Stack
- **Runtime:** Go 1.24+
- **API:** chi (REST) + gqlgen (GraphQL)
- **Database:** PostgreSQL via GORM
- **Migrations:** goose
- **Queue:** Google Cloud Tasks
- **Scheduler:** Google Cloud Scheduler
- **Cache/Sessions:** Redis
- **Auth:** go-pkgz/auth (OAuth2/JWT) with password login and refresh
- **Payments:** Paystack (Nigerian payment gateway)
- **AI Moderation:** Gemini primary with Anthropic fallback
- **Storage:** Cloudflare R2 (S3-compatible)
- **Email:** SMTP or Resend
- **FX Rates:** exchangerate-api.com
- **Infrastructure:** Terraform (GCP deployment)
- **Rate Limiting:** Redis-backed per-IP rate limiting in production

## Process Overview
- **API Server (`cmd/api`):** Serves REST/GraphQL, enqueues tasks to Cloud Tasks, and orchestrates domain services.
- **Worker (`cmd/worker`):** Processes Cloud Tasks for AI moderation, media processing, email jobs, and cleanup.
- **Scheduler:** Cloud Scheduler triggers periodic jobs for maintenance, reconciliation, and notifications.

## Getting Started

### Prerequisites
- Go 1.24+
- Docker & docker-compose
- PostgreSQL 14+
- Redis 7+
- Google Cloud account (for Cloud Tasks, Cloud Scheduler)
- Cloudflare R2 credentials
- AI API keys (Gemini, Anthropic)
- Paystack API keys

### Setup
1. **Clone & Install:**
   ```bash
   git clone <repo>
   cd hauslet-services
   ```

2. **Configuration:**
   - Copy `.env.example` to `.env`
   - Fill in required environment variables
   - YAML queue subjects config under `config/`

3. **Infrastructure:**
   ```bash
   make compose-up  # Starts Postgres, Redis
   ```

4. **Database:**
   ```bash
   export DATABASE_URL="postgres://user:pass@localhost:5432/hauslet?sslmode=disable"
   make migrate     # Run migrations
   make seed        # Seed dev data (optional)
   ```

5. **Run Services:**
   ```bash
   # Terminal 1: API Server
   make run-api

   # Terminal 2: Worker
   make run-worker
   ```

6. **Development:**
   - API: http://localhost:8080
   - GraphQL Playground (dev only): http://localhost:8080/playground

## API Endpoints

### REST Routes (Consistent Namespacing)
- **`/auth/*`** - Authentication operations (register, login, password reset, email verification, OAuth)
- **`/me/*`** - Current user profile management (profile, identities, sessions, change password)
- **`/listings/*`** - Property listings (media uploads, calendar blocks)
- **`/webhooks/*`** - External integrations (Paystack payment webhooks)
- **`/avatar/*`** - User avatar images

### GraphQL
- **`/query`** - Main GraphQL endpoint
- **`/playground`** - GraphQL Playground (development only)

### Rate Limiting (Production Only)
- **Auth endpoints:** 3-5 req/15min (registration, OTP)
- **User endpoints:** 20-60 req/min (profile updates, queries)
- **Webhooks:** 100 req/min (payment provider callbacks)
- **GraphQL:** 60 req/min (general queries)
- **Media uploads:** 20 req/min

## Project Structure
```
cmd/
  api/             # API server entrypoint
  worker/          # Background worker entrypoint
internal/
  modules/         # Domain modules (auth, property, payments, etc.)
    auth/          # Authentication & authorization
    profile/       # User profiles & preferences
    property/      # Property listings & media
    calendar/      # Availability & blocking
    payments/      # Payment processing (Paystack)
    finance/       # Financial transactions & ledger
    booking/       # Reservation management
    payout/        # Host payouts & disbursements
    wishlist/      # User favorites
    review/        # Property & host reviews
    business/      # Business entities & teams
    moderation/    # Content moderation (AI-powered)
  platform/        # Platform services
    ai/            # Gemini/Anthropic integration
    queue/         # Cloud Tasks integration
    storage/       # Cloudflare R2
    email/         # SMTP/Resend
    payment/       # Paystack client
    redis/         # Redis client & rate limiting
    xchange/       # FX rate service
  transport/       # API layer
    graph/         # GraphQL resolvers & schema
config/            # Configuration management
db/
  migrations/      # Database migrations (goose)
  seeds/           # Development seed data
deploy/            # Deployment scripts & Terraform
```

## Key Workflows

### Content Moderation Flow
1. Property publish request enqueues text and per-media moderation jobs to Cloud Tasks
2. Worker pulls tasks from Cloud Tasks queue
3. AI moderation: Gemini (primary) → Anthropic (fallback after threshold)
4. Updates moderation records and aggregates by content type
5. Triggers property hooks to update listing status
6. Notifies property owners of moderation results

### Payment Flow (Paystack)
1. User initiates booking payment
2. API creates payment record and returns Paystack checkout URL
3. User completes payment on Paystack
4. Paystack sends webhook to `/webhooks/paystack`
5. Webhook handler validates signature and enqueues processing job
6. Worker processes payment, updates booking status
7. Finance module records transaction in ledger
8. Notifications sent to guest and host

### Booking Lifecycle
1. Guest creates booking (payment required)
2. Payment processed via Paystack
3. Finance ledger updated (guest charge)
4. Host notified of new booking
5. On checkout: Booking completed
6. Payout scheduled for host (minus platform fee)
7. Finance ledger updated (host payout)
8. Review flow initiated for guest and host

### Subscription & Promotion Flow
1. **Auto-Subscription**: User selects supply role (Agent/Landlord) → FREE subscription auto-created
2. **Supply Gate**: User creates listing → System checks subscription & limits → Allow/Deny
3. **Listing Creation**: User creates listing (under FREE plan: max 3 listings, 20 photos each)
4. **Promotion (Optional)**:
   - Option A: Use subscription quota (if available on paid plans)
   - Option B: Purchase pay-per-promotion (₦50k-₦180k for Featured)
5. **Search Boost**: Active promotions apply 3x-10x multiplier to search ranking
6. **Usage Tracking**: Monthly quotas tracked and reset on billing cycle
7. **Upgrade**: User can upgrade plan instantly (with proration) for more limits/quotas
8. **See**: [`docs/PROMOTIONS_VS_SUBSCRIPTIONS.md`](docs/PROMOTIONS_VS_SUBSCRIPTIONS.md) for details

## Development

### Makefile Commands
```bash
make help              # Show available commands
make run-api           # Run API server
make run-worker        # Run background worker
make build-api         # Build API binary
make build-worker      # Build worker binary
make test              # Run tests
make migrate           # Run database migrations
make seed              # Seed development data
make fresh             # Reset DB, migrate, and seed
make compose-up        # Start Docker services
make compose-down      # Stop Docker services
make gql-gen           # Generate GraphQL code
```

### Code Organization Principles
- **Clean Architecture:** Service layer handles ALL business logic and authorization
- **Thin API Layer:** GraphQL/REST only validate and delegate to services
- **DDD:** Rich domain models with behavior, separate from DB schemas
- **Dependency Rule:** Domain → Repository → Service → Port (adapters)
- **Testing:** `go test ./...`

### Module Structure
Each module follows hexagonal architecture:
```
internal/modules/<module>/
  domain/          # Pure business entities (no DB/JSON tags)
  repository/      # Data access (GORM) & DB schemas
  service/         # Business logic & authorization (SINGLE SOURCE OF TRUTH)
  port/            # External adapters (HTTP, GraphQL, Queue handlers)
```

### Adding New Modules
```bash
make scaffold MODULE=mymodule
# Creates: internal/modules/mymodule/{service,repository/schema,domain,docs,templates}
```

## Deployment

### Infrastructure
- **Provider:** Google Cloud Platform (GCP)
- **IaC:** Terraform configurations in `/deploy/terraform`
- **Environments:** Development, Staging, Production
- **Compute:** Cloud Run (API & Worker)
- **Database:** Cloud SQL (PostgreSQL)
- **Caching:** Cloud Memorystore (Redis)
- **Queue:** Cloud Tasks (async job processing)
- **Scheduler:** Cloud Scheduler (periodic jobs)
- **Storage:** Cloudflare R2

### CI/CD
- Automated builds on push to `staging` and `main` branches
- Database migrations run automatically on deployment
- Environment-specific configurations via `.env` files
- Deployment scripts and infrastructure in `/deploy`
- Terraform manages Cloud Tasks queues, Cloud Scheduler jobs, and infrastructure

## Security Features
- **Authentication:** JWT-based with refresh tokens
- **OAuth2:** Google login integration
- **Rate Limiting:** Redis-backed IP-based rate limiting (production)
- **CSRF Protection:** go-pkgz/auth XSRF tokens
- **Webhook Validation:** Signature verification for payment webhooks
- **Input Validation:** Request validation at API layer
- **Authorization:** Service-layer permission checks (never trust the API layer)
- **Content Moderation:** AI-powered text and image moderation

## Contributing
This is a proprietary codebase. For internal contributors:
1. Follow the Clean Architecture patterns defined in `CLAUDE.md`
2. Keep service layer as single source of truth for business logic
3. Write tests for new features
4. Run `make test` before committing
5. Use conventional commit messages

## License
Proprietary – internal use for Hauslet.





