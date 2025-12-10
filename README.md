# Hauslet Services

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8.svg)](https://golang.org/)

This repository contains the backend services for **Hauslet**, a hybrid real estate platform tailored for the Nigerian market, combining long-term property acquisition (Zillow model) and short-term rentals (Airbnb model).

## Architecture

The system is architected as a **Modular Monolith** using **Go**, following a **Hexagonal Architecture (Ports and Adapters)** pattern. This approach ensures high performance and type safety while allowing for future scalability into microservices.

- **Core Business Logic**: `internal/`
- **Entrypoints**: `cmd/` (HTTP API & Background Worker)
- **Platform Abstractions**: `internal/platform/` (Database, Queue, etc.)

For a detailed architecture overview, see the original [Technical Design Document](./AGENTS.md).

## Technology Stack

- **Language**: Go (1.22+)
- **API**: REST (`go-chi/chi`) & GraphQL (`99designs/gqlgen`)
- **Database**: PostgreSQL with `GORM`
- **Message Broker**: NATS JetStream
- **Cache & Sessions**: Redis
- **Authentication**: `go-pkgz/auth` (OAuth2 & JWT)

## Getting Started

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- A running PostgreSQL, Redis, and NATS instance.

### 1. Configuration

All configuration is managed via environment variables and YAML files. Start by setting up your local environment:

```bash
# Copy the example environment file
cp .env.example .env
```

Now, fill in the required secrets and configuration values in the `.env` file (e.g., database credentials, API keys). For more details on service-level configuration, see the [Configuration README](./config/README.md).

### 2. Run Infrastructure

The easiest way to run the required infrastructure (PostgreSQL, Redis, NATS) is with Docker Compose:

```bash
docker-compose up -d
```

### 3. Run the Application

You can run the API server and the background worker in separate terminals:

**Run the API Server:**
```bash
go run ./cmd/api/main.go
```

**Run the Background Worker:**
```bash
go run ./cmd/worker/main.go
```

## Project Structure

```text
/hauslet-services
├── /cmd                # Application entry points
│   ├── /api            # HTTP/GraphQL API Server
│   └── /worker         # NATS Background Worker
│
├── /internal           # Core business logic (private)
│   ├── /auth           # User identity & access management
│   ├── /queue          # Generic job queue system
│   ├── /workers        # Job handlers for the worker
│   └── /platform       # Technical abstractions (DB, cache, etc.)
│
├── /config             # Service configuration (YAML)
├── go.mod
└── README.md
```
