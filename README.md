# Hauslet Services

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8.svg)](https://golang.org/)

Backend for **Hauslet**, a hybrid real estate platform (Zillow + Airbnb) for the Nigerian market. Modular monolith in Go with an async worker for heavy/background tasks.

## Architecture
- **Modular Monolith + Worker:** HTTP/GraphQL API (`cmd/api`) and a background worker (`cmd/worker`) sharing the same codebase and database. Async tasks go through NATS JetStream.
- **Hexagonal (Ports & Adapters):** Domain modules live in `internal/modules/*` with adapters for HTTP/GraphQL, queue handlers, and external services.
- **Domain Modules:** `auth`, `profile`, `property`, `business`, `moderation`.
- **Platform Abstractions:** `internal/platform` for DB, queue, storage (R2), AI providers, email, Redis.
- **Async Jobs:** Email sending, media thumbnail/cleanup, AI moderation (Gemini primary, Anthropic fallback) handled by the worker.

## Tech Stack
- Go 1.24+
- API: chi (REST) + gqlgen (GraphQL)
- DB: PostgreSQL via GORM
- Queue: NATS JetStream
- Cache/Sessions: Redis
- Auth: go-pkgz/auth (OAuth2/JWT) with password login and refresh
- AI Moderation: Gemini primary with Anthropic fallback
- Storage: Cloudflare R2 (S3-compatible)
- Email: SMTP or Resend

## Process Overview
- **API Server (`cmd/api`):** Serves REST/GraphQL, enqueues moderation, and orchestrates domain services.
- **Worker (`cmd/worker`):** Subscribes to NATS subjects for AI moderation, media processing, email jobs, and cleanup.

## Getting Started
1) **Prerequisites:** Go 1.24+, Docker, docker-compose, Postgres, Redis, NATS, R2 credentials, AI keys.
2) **Config:** Copy `.env.example` to `.env` and fill required envs. YAML queue subjects are under `config`/`YAML`.
3) **Infrastructure:** `docker-compose up -d` (for Postgres/Redis/NATS).
4) **Run API:** `go run ./cmd/api`
5) **Run Worker:** `go run ./cmd/worker`

## Key Paths
```
cmd/api            # API entrypoint and route wiring
cmd/worker         # Worker entrypoint and setup
internal/modules
  auth             # Identity/auth flows, sessions
  profile          # User profiles, notifications
  property         # Listings, media, publish flow, moderation hook handling
  business         # Business entities, invitations, notifications
  moderation       # Moderation records, aggregation, AI enqueue/process
internal/platform  # ai (Gemini/Anthropic + fallback), queue, storage (R2), email, database, redis
config             # Config structs and YAML loader
```

## Moderation Flow
- Publish request enqueues text and per-media moderation jobs.
- Worker pulls jobs, calls AI (Gemini → Anthropic fallback after threshold; no fallback for video), updates moderation records, aggregates latest per content type, and triggers property hooks to update listing status and notify owners.

## Notes for Developers
- Modular boundaries are respected via ports/adapters; keep domain logic in `internal/modules/*`.
- Worker wiring is in `cmd/worker/setup/*`; API wiring in `cmd/api/server/routes.go`.
- Tests: standard Go tooling; run `go test ./...`.

## License
Proprietary – internal use for Hauslet.
