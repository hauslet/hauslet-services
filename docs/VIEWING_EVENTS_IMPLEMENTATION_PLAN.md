# Viewing and Open House Events Implementation Plan

## Overview
This plan adds viewing and open house events for rent and sale listings. Agents and landlords can create events, and prospects can request or register. The implementation builds on the existing calendar module so the domain stays consistent and reusable.

## Goals
- Enable hosts/agents to create showings and open houses for rent and sale listings.
- Allow prospects to request a viewing or register for open houses.
- Respect agent/listing availability windows before accepting requests.
- Track attendance, feedback, and outcomes for follow-up.
- Reuse calendar conflict detection and availability logic.
- Support notifications and analytics for the viewing funnel.
- Sync open house registrations into leads/CRM workflows.

## Non-goals
- No payment, escrow, or booking workflow changes.
- No shortlet viewings (shortlet uses booking flow).
- No public event discovery beyond listing context.

## Architecture Fit (Current Codebase)
- Core logic in `internal/modules/calendar`.
- Use existing `EventTypeShowing` and `EventTypeOpenHouse`.
- Keep data in `calendar_event` with JSONB details (same as current models).
- Add ports for GraphQL and optional HTTP for public registration.
- Use existing queue + worker pattern for reminders and maintenance tasks.
- Emit calendar events through the queue for leads/analytics consumers.

## Showing Availability Windows
Avoid “blind requests” by validating requested times against explicit showing windows.

Recommended V1 approach:
- Store `ShowingAvailability` in listing metadata for rent/sale (JSON in `RentalDetail` and `SaleDetail`).
- Use `listingHooks` to expose showing windows to the calendar service.
- Validate `RequestShowing` and `RescheduleViewing` against those windows before creating/updating events.

Optional future extension:
- Add agent-level availability (profile or business schedules) and merge with listing windows.

Checklist:
- [ ] Define `ShowingAvailability` (day-of-week, start/end, timezone).
- [ ] Add `showing_availability` to rental/sale details and update mappers.
- [ ] Extend `ListingHooks` with `GetShowingAvailability`.
- [ ] Reject requests outside availability windows with a clear error.

## Domain Model Adjustments
Keep `CalendarEvent` as the canonical record. Extend details with minimal fields:
- `ShowingDetail`:
  - `ProspectID` and `ProspectEmail` already exist.
  - Add `RequestedBy` (user ID) or reuse `ProspectID` when logged in.
  - Add `RequestedAt` (time) for SLA and reminders.
  - Add `RescheduledAt` and `RescheduleReason` for audit trail.
  - Add `Status` or reuse `CalendarEvent.Status` as source of truth.
- `OpenHouseDetail`:
  - Add `RegistrationDeadline` (optional).
  - Add `Attendee.UserID` (optional) for registered users.
  - Add `RegistrationToken` (optional) for public cancel links.

## Data and Schema
Update `internal/modules/calendar/repository/schema/gorm.go` to include new JSON fields for showing and open house details. Keep JSONB serializer for details to match existing patterns. Add showing availability to listing metadata (rent/sale) so it can be validated before requests are accepted.

Checklist:
- [ ] Extend `ShowingDetail` with request metadata fields.
- [ ] Extend `OpenHouseDetail` and `Attendee` with optional user linkage.
- [ ] Add `showing_availability` to rental and sale detail schemas.
- [ ] Add data validation for new fields in GORM hooks where needed.

## Service Layer (Calendar)
Add explicit service methods to align with showing and open house workflows:
- Viewings:
  - `RequestShowing` -> create event with `EventStatusPending`.
  - `ConfirmShowing` -> update status to confirmed.
  - `CancelShowing` -> status cancelled and notify.
  - `RescheduleViewing` -> updates time without losing event context.
  - `MarkShowingAttendance` already exists; keep it.
- Open houses:
  - `CreateOpenHouse` already exists; ensure listing type rules apply.
  - `RegisterOpenHouseAttendee` (idempotent, capacity-aware).
  - `RemoveOpenHouseAttendee` (self or host removal).

Business rules:
- Only `ListingType` rent and sale allowed.
- Listing must be published and have calendar enabled.
- Use `listingHooks` to verify ownership and listing type.
- Validate requested times against showing availability windows.
- Use `CreateEvent` conflict checks to avoid overlaps.

Checklist:
- [ ] Extend `CalendarService` interface for request/confirm/cancel flows.
- [ ] Implement new service methods in `internal/modules/calendar/service/service.go`.
- [ ] Add `RescheduleViewing` workflow that preserves event history.
- [ ] Add listing type checks through `listingHooks` and property module.
- [ ] Enforce showing availability windows before creating/updating events.
- [ ] Add idempotency checks for attendee registration.
- [ ] Use atomic JSONB updates for attendee list writes.

## Repository Changes
JSONB attendee lists need concurrency safety:
- Prefer atomic JSONB updates using Postgres functions to avoid race conditions.
- Use SQL helpers (`jsonb_set`, `||`, `jsonb_path_exists`) to append/remove attendees with guards.
- Use transactions only when atomic SQL cannot express the update.

Checklist:
- [ ] Add repository method for atomic attendee append/remove.
- [ ] Add SQL guards for capacity and duplicate prevention.

## GraphQL API
Add GraphQL mutations and queries in a new calendar GraphQL port, consistent with other modules.

Proposed mutations:
- `requestViewing(listingId, startTime, endTime, message, contactInfo)`
- `confirmViewing(eventId)`
- `cancelViewing(eventId, reason)`
- `rescheduleViewing(eventId, startTime, endTime, reason)`
- `createOpenHouse(listingId, startTime, endTime, details)`
- `registerOpenHouse(eventId, attendee)`
- `markOpenHouseAttendance(eventId, attendeeId, attended)`

Proposed queries:
- `listingEvents(listingId, startTime, endTime, types)`
- `openHouseAttendees(eventId)` (host only)

Checklist:
- [ ] Create `internal/modules/calendar/port/graphql/schema.graphqls`.
- [ ] Add resolvers in `internal/modules/calendar/port/graphql/resolvers.go`.
- [ ] Wire resolvers into `internal/transport/graph`.
- [ ] Add auth checks (owner, agent, business roles).
- [ ] Add `showing_availability` fields to listing inputs (rent/sale).
- [ ] Validate requested times against availability windows in resolvers.

## HTTP API (Public Registration)
If public registration is required, add HTTP routes under calendar:
- `POST /listings/{listingId}/open-houses/{eventId}/register`
- Optional: `POST /listings/{listingId}/viewings/request`

Checklist:
- [ ] Add handlers in `internal/modules/calendar/port/http`.
- [ ] Apply Redis rate limiting and input validation.
- [ ] Return minimal data and avoid exposing attendee lists.

## Notifications and Reminders
Use queue + worker handlers for reminders and follow-ups:
- Confirmation emails for viewing requests and open house registration.
- Reminder 24h and 1h before event start.
- Post-event follow-up to mark attendance or collect feedback.
- Attach `.ics` calendar files to confirmations for agents and prospects.

Checklist:
- [ ] Add queue subjects in `config/defaults/queue.yaml`.
- [ ] Add routes in `internal/platform/queue/routes.go`.
- [ ] Create worker handlers in `internal/transport/worker/handlers/calendar`.
- [ ] Add Cloud Scheduler jobs for reminders if needed.
- [ ] Extend email platform to support attachments.
- [ ] Generate `.ics` files for viewing and open house confirmations.
- [ ] Update email templates to include “Add to Calendar” CTA.

## Lead/CRM Sync (Event-Driven)
Open house attendees are leads. Avoid siloing them in JSONB by emitting domain events and letting the leads module ingest them.

Approach:
- Emit `calendar.open_house_attendee_registered` and `calendar.open_house_attended` events.
- Payload should include listing ID, event ID, prospect contact info, and owner/agent IDs.
- Leads module listens and upserts a Lead record linked to the listing.

Checklist:
- [ ] Define calendar event payloads in `internal/queue/jobs/calendar`.
- [ ] Emit events from `RegisterOpenHouseAttendee` and attendance updates.
- [ ] Add a leads/CRM consumer to upsert lead records.

## Analytics and Event Tracking
Track viewing funnel events for listings and business dashboards:
- `viewing_requested`
- `viewing_confirmed`
- `viewing_rescheduled`
- `viewing_completed`
- `open_house_registered`
- `open_house_attended`

Checklist:
- [ ] Emit events into analytics module or event ingestion flow.
- [ ] Add daily rollups for viewing conversion if analytics module is active.
- [ ] Ensure event payloads include prospect info required by CRM.

## Security Considerations
- Authorization: restrict creation and updates to listing owners or business members with edit permissions.
- Data exposure: only owners/agents can see attendee lists and prospect details.
- Input validation: validate time windows, contact fields, and max attendees.
- Abuse prevention: rate-limit public registration by IP/email and dedupe registrations.
- Idempotency: use registration keys to prevent duplicates and race conditions.
- PII retention: archive or mask attendee details after a retention window.
- Auditability: log event lifecycle changes without sensitive data.
- Calendar safety: enforce availability windows server-side for all viewing requests.
- Public endpoints: use non-guessable IDs or signed tokens for register/cancel actions.

Checklist:
- [ ] Enforce owner or business membership checks in calendar service.
- [ ] Add rate limiting middleware for public endpoints.
- [ ] Add registration dedupe and capacity checks.
- [ ] Add PII retention policy via archive job.
- [ ] Restrict access to `.ics` links or avoid embedding sensitive details.
- [ ] Require signed tokens for public register/cancel actions.

## Testing Strategy
- Unit tests for service validation and conflict detection.
- Concurrency tests for attendee registration.
- GraphQL and HTTP endpoint tests for access control.
- Worker handler tests for reminders and post-event actions.
- Tests for showing availability validation and reschedule flows.
- Tests for `.ics` generation and attachment handling.

Checklist:
- [ ] Add service tests for viewing/open house flows.
- [ ] Add API tests for auth and validation.
- [ ] Add worker handler tests for reminder payloads.
- [ ] Add concurrency tests for atomic attendee updates.

## Rollout Plan
- Phase 1: internal API only for hosts and agents, with showing availability windows.
- Phase 2: enable public registration with rate limiting and `.ics` attachments.
- Phase 3: add analytics, lead sync automation, and follow-ups.

Checklist:
- [ ] Deploy phase 1 behind a feature flag if needed.
- [ ] Monitor errors and conflict rates.
- [ ] Enable public registration after validation.
