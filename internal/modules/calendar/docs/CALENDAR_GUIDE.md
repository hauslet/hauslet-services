# Hauslet Calendar System - Complete Guide

## Table of Contents

1. [Overview](#overview)
2. [Event Types](#event-types)
3. [Availability & Conflicts](#availability--conflicts)
4. [Showing Workflow](#showing-workflow)
5. [Open House Management](#open-house-management)
6. [Maintenance & Blocks](#maintenance--blocks)
7. [Recurring Patterns](#recurring-patterns)
8. [Configuration](#configuration)
9. [GraphQL API Reference](#graphql-api-reference)
10. [Integration Hooks](#integration-hooks)

---

## Overview

The **Hauslet Calendar System** is the central time management engine for properties. It handles availability for bookings, schedules property viewings (showings), manages open houses, tracks maintenance work, and allows owners to block dates for personal use.

### Key Features

- **Unified Timeline**: Single source of truth for all property-related events.
- **Polymorphic Events**: Supports diverse event types (Bookings, Showings, Maintenance, etc.) with specific data structures.
- **Conflict Resolution**: Intelligent availability checking that distinguishes between "blocking" and "non-blocking" events.
- **Viewing Management**: specialized workflows for scheduling and confirming property tours for Rent/Sale listings.
- **Open House Registration**: Capacity-managed event registration for public viewings.
- **Recurring Events**: Support for repeating availability patterns or maintenance schedules.

### System Architecture

```
┌──────────────┐
│    User      │
│ (Host/Guest) │
└──────┬───────┘
       │
       ├──→ checkAvailability (Bookings)
       │
       ├──→ requestShowing (Rent/Sale)
       │
       ▼
┌──────────────────────────────────────────────────────────┐
│                  Calendar Service                        │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌─────────┐│
│  │ Event    │  │ Conflict │  │ Recurring│  │ Config  ││
│  │ Manager  │  │ Detector │  │ Engine   │  │ Service ││
│  └──────────┘  └──────────┘  └──────────┘  └─────────┘│
└──────────────────────────────────────────────────────────┘
       │
       ├──→ Listing Module (Validation, Owner checks)
       │
       ├──→ Profile Module (User details)
       │
       ▼
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Database   │     │ Notifications│     │    Redis     │
│  (Events)    │     │   (Email)    │     │   (Cache)    │
└──────────────┘     └──────────────┘     └──────────────┘
```

---

## Event Types

The system uses a single `CalendarEvent` entity with a polymorphic design to handle different activities.

### 1. Booking (`booking`)
- **Source**: Created by the Booking Module.
- **Blocking**: YES (Prevents other bookings).
- **Details**: Linked via `booking_id`.
- **Status**: Draft, Confirmed, Cancelled.

### 2. Showing (`showing`)
- **Purpose**: A prospect visits a property for sale or rent.
- **Blocking**: NO (Usually short duration, doesn't prevent bookings unless overlapping).
- **Details**: `ShowingDetail` (Prospect info, Agent info, Feedback).
- **Status**: Pending, Confirmed, Completed, No-Show.

### 3. Maintenance (`maintenance`)
- **Purpose**: Repairs, cleaning, or renovations.
- **Blocking**: Conditional (Disruptive work blocks bookings; minor repairs might not).
- **Details**: `MaintenanceDetail` (Vendor, Cost, Disruptive flag).
- **Subtypes**: Routine, Emergency, Repair, Renovation.

### 4. Block (`block`)
- **Purpose**: Owner manually blocks dates (e.g., personal stay).
- **Blocking**: YES (Always).
- **Details**: `BlockDetail` (Reason, OwnerStay flag).

### 5. Open House (`open_house`)
- **Purpose**: Public viewing event for multiple attendees.
- **Blocking**: NO (Doesn't block bookings, but might overlap).
- **Details**: `OpenHouseDetail` (Max attendees, Registered list).

---

## Availability & Conflicts

### Conflict Logic

Availability is determined by checking if a requested time range overlaps with any "Blocking" event.

**Blocking Rules**:
| Event Type | Blocks Bookings? | Logic |
|------------|------------------|-------|
| Booking | ✅ Yes | If status != Cancelled |
| Block | ✅ Yes | Always |
| Maintenance| ⚠️ Conditional | Only if `Disruptive == true` |
| Showing | ❌ No | Viewings don't prevent overnight stays |
| Open House | ❌ No | Public events don't prevent stays |

### Availability Check Algorithm

```go
func CheckAvailability(listingID, start, end) {
    // 1. Get all events overlapping [start, end]
    events = repo.GetOverlappingEvents(listingID, start, end)
    
    // 2. Check for blockers
    for event in events {
        if event.BlocksBookings() {
            return Unavailable (Reason: event.Type)
        }
    }
    
    // 3. Check Listing Constraints (Min nights, buffers)
    // ... logic handled by Listing Service ...
    
    return Available
}
```

---

## Showing Workflow

Used primarily for long-term **Rent** and **Sale** listings where potential tenants/buyers need to inspect the property.

### Flow Diagram

```
Prospect                 System                  Host/Agent
   │                        │                        │
   ├── Request Showing ────→│                        │
   │   (Time, Details)      ├─── Validate Request    │
   │                        │   (Availability)       │
   │                        │                        │
   │                        ├─── Notify Host ───────→│
   │                        │                        │
   │                        │←── Confirm Request ────┤
   │←── Send Confirmation ──┤                        │
   │                        │                        │
   │      [Viewing Day]     │                        │
   │                        │                        │
   │←─── Feedback Request ──┤←── Mark Attended ──────┤
   │                        │                        │
```

### Constraints & Validation
- **Listing Type**: Only valid for `rent` or `sale` listings.
- **Availability Windows**: Hosts define specific windows (e.g., "Mon-Fri, 9am-5pm") via `ShowingAvailability` configuration.
- **Lead Time**: Requests must respect the `lead_time_hours` setting.

### Rescheduling
- Both Prospect and Host can propose a reschedule.
- Rescheduling resets status to `Pending` requiring re-confirmation.
- Tracks `RescheduleCount` and reasons.

---

## Open House Management

Organized events where multiple prospects can visit simultaneously.

### Key Features
- **Capacity Control**: `MaxAttendees` limits registrations.
- **Registration Deadline**: Optional cutoff time for sign-ups.
- **Idempotency**: Duplicate registrations by email are handled gracefully.
- **Tokenized Access**: Attendees get a token for self-cancellation.

### Registration Process
1. **Create Event**: Agent schedules Open House (e.g., "Saturday 10am-2pm").
2. **Register**: Users call `registerOpenHouseAttendee`.
   - System checks capacity.
   - System checks deadline.
   - Adds to `Attendees` list (JSONB).
3. **Attendance**: Agent marks `Attended = true` for check-ins.

---

## Maintenance & Blocks

### Blocking Dates (Owner/Host)
Owners can block dates for any reason.
- **Owner Stay**: `OwnerStay = true` (Used for analytics/reporting).
- **Reason**: Required field (e.g., "Painting", "Family Visit").

### Maintenance Tracking
Tracks the lifecycle of property upkeep.
1. **Schedule**: Define vendor and estimated cost.
2. **Execute**: Mark `WorkCompleted = true`.
3. **Finalize**: Record `ActualCost` and notes.
- **Financial Integration**: Completed maintenance costs can be pulled by the Finance module for expense reporting.

---

## Recurring Patterns

Allows setting up repeating events (e.g., "Cleaner comes every Monday at 10 AM").

### Pattern Structure (`RecurringEventPattern`)
- **Frequency**: Daily, Weekly, Monthly.
- **Interval**: Every X (days/weeks).
- **DaysOfWeek**: Bitmask or string (e.g., "Mon,Wed,Fri").
- **Exceptions**: Specific dates excluded from the pattern.

**Note**: The system currently "expands" patterns into individual `CalendarEvent` instances for a look-ahead period (e.g., 3 months) via a background job, rather than calculating them strictly on-the-fly. This simplifies collision detection.

---

## Configuration

Each listing has a `CalendarConfig` that controls behavior.

```json
{
  "buffer_hours": 2,          // Time gap required between bookings
  "lead_time_hours": 24,      // Minimum notice before check-in
  "booking_window_months": 6, // How far in future can guests book
  "instant_booking": true,    // Auto-confirm bookings
  "same_day_booking": false,  // Allow booking for today?
  "timezone": "Africa/Lagos"  // Critical for accurate local time calculations
}
```

---

## GraphQL API Reference

### Queries

#### Check Availability
```graphql
query {
  checkAvailability(
    listingId: "uuid", 
    startTime: "2026-02-01T00:00:00Z", 
    endTime: "2026-02-05T00:00:00Z"
  ) {
    available
    reason
    conflicts {
      id
      eventType
      startTime
      endTime
    }
  }
}
```

#### Get Calendar Events
```graphql
query {
  calendarEvents(
    listingId: "uuid",
    startTime: "2026-02-01T00:00:00Z",
    endTime: "2026-03-01T00:00:00Z"
  ) {
    id
    eventType
    status
    startTime
    endTime
    # Polymorphic details
    showingDetails { prospectName }
    blockDetails { reason }
  }
}
```

### Mutations

#### Request Showing
```graphql
mutation {
  requestShowing(input: {
    listingId: "uuid",
    startTime: "2026-02-10T14:00:00Z",
    endTime: "2026-02-10T14:30:00Z",
    note: "I love the kitchen photos!"
  }) {
    id
    status # Pending
  }
}
```

#### Block Dates
```graphql
mutation {
  blockDates(input: {
    listingId: "uuid",
    startTime: "2026-05-01T00:00:00Z",
    endTime: "2026-05-07T00:00:00Z",
    reason: "Renovation",
    disruptive: true
  }) {
    id
    status
  }
}
```

---

## Integration Hooks

The Calendar module interacts with others via defined interfaces:

### `ListingHooks`
- **CanUseCalendar**: Checks subscription/feature flags.
- **GetListingOwner**: Verifies permissions.
- **GetShowingAvailability**: Fetches valid showing hours.
- **GetListingType**: Ensures showings only for Rent/Sale.

### `ProfileHooks`
- **GetUserProfile**: Enriches events with user names/emails for display.

### `NotificationService`
- Sends emails for: Showing Requests, Confirmations, Cancellations, Open House Registrations.

---

## Best Practices

1. **Timezones**: Always store times in UTC but perform logic (like "Same Day" checks) using the listing's configured `Timezone`.
2. **Buffers**: Respect `BufferHours` when calculating availability to ensure turnover time.
3. **Idempotency**: Use idempotent operations for attendee registration to prevent duplicates.
4. **Caching**: Utilize `invalidateAvailabilityCache` aggressively on any event modification to ensure search results are accurate.
5. **Soft Deletes**: Never hard delete events; use `DeletedAt` to maintain history for analytics.
