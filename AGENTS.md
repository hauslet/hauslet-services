This document serves as the **Technical Architecture & Design Specification** for **Hauslet Services**. It outlines the system architecture, technology choices, module boundaries, and data design for a scalable, production-grade Go application.

-----

# Hauslet Services: Technical Design Document (v1.0)

## 1\. Executive Summary

**Hauslet Services** is a hybrid real estate platform tailored for the Nigerian market. It bridges the gap between long-term property acquisition (**Zillow model**) and short-term rental bookings (**Airbnb model**).

The system is architected as a **Modular Monolith** using **Go**. This ensures high performance and type safety while maintaining the flexibility to extract specific modules (like Payments or Notifications) into microservices in the future without a total rewrite.

-----

## 2\. High-Level Architecture

The system follows a **Hexagonal Architecture (Ports and Adapters)** style within a monolithic structure.

  * **The Core:** All business logic lives in `internal/`.
  * **The Ports:** Traffic enters via HTTP (Chi Router) or GraphQL (Gqlgen).
  * **The Async Layer:** Heavy tasks (media processing, AI moderation) are offloaded to background workers via **NATS JetStream**.

[Image of Hexagonal Architecture Diagram for Go Application]

-----

## 3\. Technology Stack

### Core Backend

  * **Language:** Go (Golang) 1.22+
  * **Router:** `go-chi/chi` (Lightweight, idiomatic HTTP routing).
  * **API Layer:** `99designs/gqlgen` (Schema-first GraphQL) & REST for Webhooks.
  * **Database ORM:** `GORM` (Object Relational Mapping for PostgreSQL).
  * **Dependency Injection:** Manual injection (Constructor-based) for strict modularity.

### Data & Infrastructure

  * **Primary Database:** **PostgreSQL** (Relational data).
  * **Message Broker:** **NATS JetStream** (Work queues for moderation/emails).
  * **Caching/Session:** **Redis** (Session storage & rate limiting).
  * **Authentication:** `go-pkgz/auth` (OAuth2 & JWT management).
  * **AI/LLM:** OpenAI API (via internal worker) for listing moderation.

-----

## 4\. Module Breakdown (Vertical Slices)

The application is divided into five distinct business domains.

### A. Auth Module (`internal/auth`)

Responsible for identity and access management.

  * **Features:** Google/Email Login, JWT issuance, Role Management (Landlord vs. Tenant vs. Admin).
  * **Nigerian Context:** Basic KYC verification (NIN/BVN placeholder fields) for Landlords.

### B. Property Module (`internal/property`)

Handles the "Zillow" and "Airbnb" asset data.

  * **Features:** CRUD for properties, Media management (images/videos), Geolocation search.
  * **Logic:** Handles "Listing Type" (Sale vs. Rent vs. Short-let).

### C. Booking Module (`internal/booking`)

Handles the transactional logic for short-term stays.

  * **Features:** Calendar availability checks, Reservation state machine (Pending -\> Paid -\> Confirmed).
  * **Dependencies:** Injects `PropertyModule` interface to check availability.

### D. Payment Module (`internal/payment`)

Abstracts the payment gateway logic.

  * **Features:** Integration with **Paystack** or **Flutterwave**, Wallet management, Payout processing.
  * **Logic:** Handles webhooks from providers to update Booking status securely.

### E. Moderation Module (`internal/moderation`)

The AI safety layer.

  * **Features:** Consumes NATS messages, checks text/images against OpenAI, flags inappropriate listings.

-----

## 5\. Database Schema Design (GORM Models)

The database relates users to properties and bookings.

[Image of Entity Relationship Diagram for Real Estate App]

### Key GORM Structs

```go
// internal/auth/models.go
type User struct {
    gorm.Model
    Email    string `gorm:"uniqueIndex"`
    FullName string
    Role     string // 'admin', 'host', 'guest'
    Listings []property.Property
    Bookings []booking.Booking
}

// internal/property/models.go
type Property struct {
    gorm.Model
    UserID      uint   // Foreign Key
    Title       string
    Description string
    Price       float64
    Type        string // 'sale', 'rent', 'shortlet'
    Status      string // 'pending_moderation', 'active', 'rejected'
    Images      []string `gorm:"serializer:json"` // Store URLs as JSON
}

// internal/booking/models.go
type Booking struct {
    gorm.Model
    PropertyID uint
    GuestID    uint
    CheckIn    time.Time
    CheckOut   time.Time
    Status     string // 'confirmed', 'cancelled'
    TotalPaid  float64
}
```

-----

## 6\. Key System Workflows

### Flow 1: New Listing & AI Moderation (Async)

This flow ensures no illegal content appears on Hauslet services.

1.  **Client:** Sends GraphQL Mutation `createProperty(...)`.
2.  **API (Go):** Saves Property to DB with status `PENDING_MODERATION`.
3.  **API (Go):** Publishes Event `mod.property.created` to NATS JetStream.
4.  **API (Go):** Returns "Success" to user immediately.
5.  **Worker (Go):** Pulls message from NATS.
6.  **Worker (Go):** Sends description/images to **LLM API**.
7.  **Worker (Go):**
      * *If Safe:* Updates DB status to `ACTIVE`.
      * *If Unsafe:* Updates DB status to `REJECTED` and emails user.

### Flow 2: Booking with Payment

1.  **Client:** Mutation `bookProperty(id, dates)`.
2.  **Booking Module:** Checks `PropertyModule` interface for date conflicts.
3.  **Booking Module:** Creates `Booking` record (Status: `PENDING_PAYMENT`).
4.  **Payment Module:** Generates a Paystack Checkout Link.
5.  **Client:** User pays on Paystack.
6.  **Paystack:** Sends Webhook (POST) to `/api/webhooks/paystack`.
7.  **Payment Module:** Verifies signature $\rightarrow$ Updates Booking to `CONFIRMED`.

-----

## 7\. Project Directory Structure

This structure enforces the Modular Monolith principles.

```text
/hauslet-services
├── /cmd
│   ├── /api                # Main entry point for HTTP/GraphQL API
│   │   └── main.go
│   └── /worker             # Main entry point for NATS Background Worker
│       └── main.go
│
├── /internal
│   ├── /auth               # Logic: User Identity
│   ├── /property           # Logic: Real Estate Listings
│   ├── /booking            # Logic: Reservations
│   ├── /payment            # Logic: Paystack/Flutterwave
│   ├── /moderation         # Logic: AI Safety
│   │
│   ├── /graph              # GraphQL Presentation Layer
│   │   ├── schema.graphqls
│   │   └── resolvers.go    # Glues Modules to GraphQL
│   │
│   └── /platform           # Technical foundation (Not business logic)
│       ├── /database       # GORM connection setup
│       ├── /queue          # NATS JetStream setup
│       └── /router         # Chi Router configuration
│
├── /migrations             # Raw SQL for DB versioning
├── docker-compose.yml      # Postgres, Redis, NATS
├── Makefile                # Build automation
└── go.mod
```

-----

## 8\. Deployment Strategy

  * **Containerization:** The app compiles into a single Docker image.
  * **Orchestration:**
      * *Development:* `docker-compose`.
      * *Production:* AWS ECS or DigitalOcean App Platform.
  * **Environment Variables:** All secrets (DB Passwords, OpenAI Keys, Paystack Secrets) are injected via `.env`.

-----

## Next Steps for Implementation

1.  **Initialize the Repo:** Set up the `go.mod` and the `internal/platform` packages.
2.  **Docker Setup:** Create the `docker-compose.yml` to spin up Postgres and NATS.
3.  **Core Domain:** Implement the `User` and `Property` GORM models and the basic Service/Repository layers.
4.  **API Layer:** Wire up `go-chi` and generate the GraphQL resolvers.