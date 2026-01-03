# Roomies and Lawyers Implementation Plan

**Status:** Design  
**Owner:** Backend Team  
**Last Updated:** 2026-01-01  
**Scope:** Server-side implementation only (web and mobile UI out of scope)

---

## Executive Summary

This plan introduces two new vertical slices aligned with Hauslet's modular monolith and hexagonal architecture:

- Roomies: a lightweight roommate discovery module with one-time payments for promotions (no subscriptions).
- Lawyers: a verified property lawyer marketplace with hybrid monetization (subscription for providers plus per-booking fees).

Both modules follow the same patterns already used in the codebase: service-layer enforcement, thin transport adapters, GORM repositories, queue-driven async work, and explicit integration adapters for cross-module dependencies.

---

## Key Business Decisions

1. Roomies monetization uses one-time payments for promotion windows (7, 14, 30 days). No recurring plans.
2. Lawyers monetization is hybrid: provider subscription plus per-booking/engagement fees.
3. Verification is required before supply-side actions. Lawyers require ID verification and credential verification. Roomies require phone verification and ID verification for offers or contact unlocks.

---

## Goals

- Provide a safe roommate discovery flow without building a full messaging system on day one.
- Allow verified lawyers to be discovered and booked with clear pricing and availability.
- Reuse payments, notifications, queue, moderation, and profile verification systems.
- Enforce access rules in service layers, not GraphQL or HTTP handlers.

## Non-goals

- No in-app chat or real-time messaging in v1.
- No escrow or milestone payments in v1 (lawyers only).
- No client-side flows in this document.

---

## Architecture Fit (Current Codebase)

- New modules live under `internal/modules/roomies` and `internal/modules/lawyers`.
- Service layer enforces access rules and subscription/payment state.
- Adapters in `port/hooks` expose only required cross-module data.
- Async tasks use `internal/queue` and worker handlers.
- Email templates use the existing `internal/platform/email` infrastructure.
- Moderation uses the existing AI moderation pipeline.

---

## Module Structure

### Roomies

```text
internal/modules/roomies/
  domain/
  repository/
  service/
  notification/
  port/
    graphql/
    http/
    hooks/
  templates/
```

### Lawyers

```text
internal/modules/lawyers/
  domain/
  repository/
  service/
  notification/
  port/
    graphql/
    http/
    hooks/
  templates/
```

---

## Domain Model

### Roomies

Core entities are intentionally compact for a v1 launch.

```go
type RoomiePost struct {
    ID             uuid.UUID
    OwnerID        uuid.UUID
    PostType       RoomiePostType // offer_room, seek_room
    Title          string
    Description    string
    City           string
    State          string
    ListingID      *uuid.UUID     // optional link to property listing
    BudgetMin      *int
    BudgetMax      *int
    MoveInDate     *time.Time
    Preferences    map[string]any // JSONB: gender, pets, smoking, etc
    Status         RoomiePostStatus // draft, pending_moderation, active, paused, expired, rejected
    PromotedUntil  *time.Time
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

type RoomieInterest struct {
    ID        uuid.UUID
    PostID    uuid.UUID
    SenderID  uuid.UUID
    Message   string
    Status    RoomieInterestStatus // pending, accepted, declined, withdrawn
    CreatedAt time.Time
}

type RoomiePromotion struct {
    ID        uuid.UUID
    PostID    uuid.UUID
    Tier      RoomiePromotionTier // boost, featured
    StartsAt  time.Time
    EndsAt    time.Time
    PaymentID uuid.UUID
    Status    PromotionStatus // pending_payment, active, expired, cancelled
}
```

### Lawyers

```go
type LawyerProfile struct {
    ID              uuid.UUID
    UserID          uuid.UUID
    DisplayName     string
    Bio             string
    PracticeAreas   []string
    YearsExperience int
    BarNumber       string
    Status          LawyerProfileStatus // draft, pending_verification, verified, suspended
    VerifiedAt      *time.Time
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type LawyerService struct {
    ID            uuid.UUID
    LawyerID      uuid.UUID
    Name          string
    Description   string
    PriceMinor    int64
    DurationMins  int
    Enabled       bool
}

type LawyerAvailability struct {
    ID        uuid.UUID
    LawyerID  uuid.UUID
    DayOfWeek int
    StartTime string // HH:MM
    EndTime   string // HH:MM
    Timezone  string
}

type LawyerBooking struct {
    ID          uuid.UUID
    LawyerID    uuid.UUID
    ClientID    uuid.UUID
    ServiceID   uuid.UUID
    ScheduledAt time.Time
    Status      LawyerBookingStatus // pending_payment, confirmed, completed, cancelled
    PaymentID   *uuid.UUID
    CreatedAt   time.Time
}

type LawyerSubscription struct {
    ID                 uuid.UUID
    LawyerID           uuid.UUID
    Plan               string
    Status             SubscriptionStatus // active, past_due, cancelled
    CurrentPeriodStart time.Time
    CurrentPeriodEnd   time.Time
    PaymentID          *uuid.UUID
}
```

---

## Workflow Specifications

### Roomies: Create and Promote a Post

1. User submits `CreateRoomiePost`.
2. Service enforces verification:
   - Phone verified required for all posts.
   - ID verified required for `offer_room` posts or when enabling contact sharing.
3. Post is saved as `pending_moderation`.
4. Moderation worker reviews content and sets `active` or `rejected`.
5. User optionally purchases a promotion tier:
   - Service creates a payment via `payments` module and returns checkout URL.
   - On webhook success, promotion activates and `promoted_until` is set.
6. Scheduled job expires promotions and posts at their TTL.

### Roomies: Express Interest

1. User submits `ExpressRoomieInterest` with optional message.
2. Service verifies sender is not the post owner.
3. Creates `RoomieInterest` with `pending` status.
4. Notification emails are sent to the post owner.
5. Owner accepts or declines; acceptance can unlock contact details.

### Lawyers: Onboarding and Verification

1. User creates a `LawyerProfile` and uploads credentials.
2. Profile is `pending_verification` until ID and credentials are approved.
3. Verification uses the verification module for ID and a manual admin review for credentials.
4. Verified profiles can subscribe and be listed publicly.

### Lawyers: Subscription and Booking

1. Lawyer selects a plan (free or paid). Subscription is created via `payments`.
2. Client chooses a lawyer and service, then creates a booking.
3. Booking is `pending_payment` until checkout success.
4. On payment success:
   - Booking moves to `confirmed`.
   - A calendar event is created (new event type: `consultation`).
5. Reminders are sent via queue jobs.

---

## Database Schema (GORM)

Roomies:

- `roomie_posts`
- `roomie_interests`
- `roomie_promotions`

Lawyers:

- `lawyer_profiles`
- `lawyer_services`
- `lawyer_availability`
- `lawyer_bookings`
- `lawyer_subscriptions`
- `lawyer_verifications` (optional, if not reusing verification module)

Key indexes:

- `roomie_posts` on `(status, city, state, promoted_until)`
- `roomie_interests` on `(post_id, status, created_at)`
- `lawyer_profiles` on `(status, city, practice_area)`
- `lawyer_bookings` on `(lawyer_id, scheduled_at)`
- `lawyer_subscriptions` on `(lawyer_id, status)`

---

## API Specification (GraphQL)

Roomies:

- Queries:
  - `roomiePost(id)`
  - `roomiePosts(filter, paging)`
  - `myRoomiePosts`
  - `roomieInterests(postID)`
- Mutations:
  - `createRoomiePost(input)`
  - `updateRoomiePost(id, input)`
  - `publishRoomiePost(id)`
  - `expressRoomieInterest(postID, message)`
  - `acceptRoomieInterest(id)`
  - `declineRoomieInterest(id)`
  - `promoteRoomiePost(postID, tier, duration)`

Lawyers:

- Queries:
  - `lawyer(id)`
  - `lawyers(filter, paging)`
  - `lawyerServices(lawyerID)`
  - `lawyerAvailability(lawyerID)`
  - `myLawyerBookings`
  - `myLawyerSubscription`
- Mutations:
  - `createLawyerProfile(input)`
  - `updateLawyerProfile(input)`
  - `submitLawyerVerification(input)`
  - `setLawyerAvailability(input)`
  - `createLawyerBooking(input)`
  - `cancelLawyerBooking(id)`
  - `createLawyerSubscription(plan)`
  - `changeLawyerPlan(plan)`

Resolvers follow existing patterns:

- Module schema in `internal/modules/<module>/port/graphql/schema.graphqls`
- Module resolvers in `internal/modules/<module>/port/graphql/resolvers.go`
- Transport wiring in `internal/transport/graph/schema.resolvers.go` and `gqlgen.yml`

---

## Authorization and Verification

Roomies:

- Require phone verification for all posts and interests.
- Require ID verification for `offer_room` posts and contact unlocks.
- Service layer enforces rules, with admin bypass via RBAC.

Lawyers:

- Require ID verification and credential verification before listing.
- Require active subscription to be discoverable.
- Restrict booking creation to verified lawyers only.

Implementation pattern:

- Add `Lawyer` as a `profile.UserType` for supply-side gating.
- Add a `RoomieGate` or extend `SupplyGate` with new actions if needed.

---

## Payments and Pricing

Roomies (one-time payments only):

- Promotion tiers configured in a new `config/defaults/roomies.yaml`.
- Service creates payment via `payments` module.
- Webhook activates promotion and sets expiration.

Lawyers (hybrid):

- Subscription plans defined in `config/defaults/lawyers.yaml`.
- Booking fees are charged per consultation.
- Platform fee applied via `finance` module (ledger entries).

---

## Queue and Notifications

Queue subjects to add in `config/defaults/queue.yaml`:

- `roomies.post.moderation`
- `roomies.post.expiry`
- `roomies.promotion.expiry`
- `lawyers.booking.reminder`
- `lawyers.subscription.billing`
- `lawyers.verification.review`

Handlers:

- Create jobs in `internal/queue/jobs/<module>/`
- Worker handlers in `internal/transport/worker/handlers/<module>/`

Email templates:

- Roomie interest received, interest accepted, post expiring, promotion active
- Lawyer booking confirmed, booking reminder, subscription renewal

---

## Integration Points

- Profile module: user types, phone verification, ID verification status.
- Verification module: identity and credential verification workflows.
- Payments module: checkout and webhook handling.
- Finance module: fees and payouts to lawyers.
- Moderation module: AI content review for roomie posts and lawyer bios.
- Calendar module: consultation events and reminders.
- Review module: optional ratings for lawyers after completed bookings.

---

## Testing Strategy

- Unit tests for service-layer gating and state transitions.
- Repository tests for JSONB fields and status filters.
- GraphQL resolver tests for permissions and error handling.
- Queue handler tests for expiry and reminder jobs.

---

## Phased Delivery Plan

Phase 1: Roomies MVP

- Roomie posts with moderation
- Express interest and notifications
- Search and basic filters

Phase 2: Roomies Monetization

- One-time promotion payments
- Promotion expiry jobs

Phase 3: Lawyers Onboarding

- Lawyer profile CRUD
- ID verification + credential review
- Subscription creation

Phase 4: Lawyers Booking

- Service offerings and availability
- Booking, payment, and calendar integration
- Reminders and email templates

Phase 5: Reviews and Analytics

- Lawyer reviews
- Roomies engagement analytics

---

## Open Questions

- Should roomie posts be linkable to an existing listing or be fully independent?
- Do we require ID verification for all roomie posts, or only for offers?
- Should lawyer subscriptions gate visibility or also gate booking acceptance?
