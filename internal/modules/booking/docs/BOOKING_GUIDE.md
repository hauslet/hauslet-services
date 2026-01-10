# Hauslet Booking System - Complete Guide

## Table of Contents

1. [Overview](#overview)
2. [Booking Lifecycle](#booking-lifecycle)
3. [Booking Types](#booking-types)
4. [Status States](#status-states)
5. [Pricing & Quotes](#pricing--quotes)
6. [Reservation Flow](#reservation-flow)
7. [Payment Integration](#payment-integration)
8. [Cancellation & Refunds](#cancellation--refunds)
9. [Check-In & Check-Out](#check-in--check-out)
10. [GraphQL API Reference](#graphql-api-reference)
11. [Usage Examples](#usage-examples)
12. [Best Practices](#best-practices)

---

## Overview

The **Hauslet Booking System** manages the complete reservation lifecycle for short-term property rentals in Nigeria. It orchestrates calendar availability, dynamic pricing, payment processing, host communications, and financial settlements.

### Key Features

- **Dual Booking Models**: Instant booking (auto-accept) + Request booking (host approval)
- **Dynamic Pricing**: Real-time price calculation with discounts, fees, and seasonal rates
- **Integrated Calendar**: Automatic blocking of dates, check-in/out time enforcement
- **Payment Processing**: Paystack integration with escrow, refunds, and payouts
- **Multi-Module Coordination**: Links calendar, pricing, payments, finance, and notifications
- **Smart Cancellation**: Policy-based refunds (flexible, moderate, strict)
- **Automated Lifecycle**: Cron jobs for booking completion, expiration, and settlements

### System Architecture

```
┌──────────────┐
│    Guest     │
│ (Frontend)   │
└──────┬───────┘
       │
       ├──→ quoteBooking (Check price & availability)
       │
       ├──→ reserveBooking (Instant) or requestBooking (Manual)
       │
       ▼
┌──────────────────────────────────────────────────────────┐
│                  Booking Service                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌─────────┐│
│  │ Calendar │  │ Pricing  │  │ Payments │  │ Finance ││
│  │ Gateway  │  │ Service  │  │ Gateway  │  │  Hooks  ││
│  └──────────┘  └──────────┘  └──────────┘  └─────────┘│
└──────────────────────────────────────────────────────────┘
       │
       ▼
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Database   │     │ Notifications│     │ Cloud Tasks  │
│  (Bookings)  │     │   (Email)    │     │  (Refunds)   │
└──────────────┘     └──────────────┘     └──────────────┘
```

---

## Booking Lifecycle

### Complete Journey: Search → Book → Stay → Complete

```
Day -7: Guest Searches
   ↓
   ├─→ quoteBooking()
   │   ├─ Check calendar availability
   │   ├─ Calculate pricing
   │   └─ Return quote (available: true/false)
   │
Day 0: Guest Books
   ↓
   ├─→ INSTANT BOOKING (Auto-Accept Listings)
   │   └─→ reserveBooking()
   │       ├─ Create booking (status: draft)
   │       ├─ Block calendar dates
   │       ├─ Initiate payment (Paystack)
   │       └─ Payment success
   │           ├─ Status: draft → confirmed
   │           ├─ Finance records charge
   │           ├─ Email confirmations sent
   │           └─ Calendar event confirmed
   │
   ├─→ MANUAL BOOKING (Approval Required)
   │   └─→ requestBooking()
   │       ├─ Create booking (status: pending_host_approval)
   │       ├─ Hold calendar dates (24h)
   │       └─ Email host for approval
   │           │
   │           ├─→ HOST APPROVES (within 24h)
   │           │   └─→ confirmBooking()
   │           │       ├─ Status: pending → awaiting_payment
   │           │       └─ Guest pays via payForBooking()
   │           │           └─ Status: awaiting_payment → confirmed
   │           │
   │           └─→ HOST DECLINES or TIMEOUT
   │               └─ Status: pending → archived
   │
Day 0 (Check-in Day): Guest Arrives
   ↓
   └─→ checkInBooking() [Optional - Host records]
       ├─ Status: confirmed → active
       ├─ Records actual check-in timestamp
       └─ Sends welcome message
   
Day 7 (Check-out Day): Guest Departs
   ↓
   └─→ checkOutBooking() [Optional - Host records]
       ├─ Status: active → completed
       ├─ Records actual check-out timestamp
       └─ Starts 48h payout window
   
Day 9 (48h After Check-out): Settlement
   ↓
   └─→ Cron: CompleteBookings()
       ├─ Auto-mark as completed if not already
       ├─ Finance triggers payout (escrow → host)
       ├─ Status: completed → settled
       ├─ Send payout notification
       └─ Send review invitations (guest & host)
```

---

## Booking Types

### 1. Instant Booking (Auto-Accept)

**What**: Guest books immediately without host approval  
**When**: Listing has `auto_accept_bookings = true`  
**Flow**: Reserve → Pay → Confirmed (instant)

**Advantages**:

- ✅ Immediate confirmation (higher conversion)
- ✅ Faster checkout experience
- ✅ Better for competitive markets
- ✅ Higher search ranking

**Use Cases**:

- Professional hosts with consistent availability
- High-trust properties (verified hosts)
- Standardized listings (apartments, condos)

**GraphQL Mutation**:

```graphql
mutation {
  reserveBooking(input: {
    listingId: "listing-uuid"
    checkIn: "2026-02-01T14:00:00Z"
    checkOut: "2026-02-08T10:00:00Z"
    guestCount: 2
    paymentMethodId: "card-uuid"
    specialRequests: "Early check-in if possible"
  }) {
    booking {
      id
      status  # confirmed
    }
    paymentId
    paymentStatus
  }
}
```

### 2. Request Booking (Manual Approval)

**What**: Host reviews and approves/declines booking request  
**When**: Listing has `auto_accept_bookings = false`  
**Flow**: Request → Host Approves → Guest Pays → Confirmed (multi-step)

**Advantages**:

- ✅ Host vets guests (reduces bad bookings)
- ✅ Flexible for unique properties
- ✅ Control over calendar availability
- ✅ Negotiate special requests

**Use Cases**:

- Luxury/high-value properties (₦500k+/night)
- Shared spaces (host lives on-site)
- Properties requiring special care
- New hosts building trust

**GraphQL Mutations**:

```graphql
# Step 1: Guest requests booking
mutation {
  requestBooking(input: {
    listingId: "listing-uuid"
    checkIn: "2026-02-01T14:00:00Z"
    checkOut: "2026-02-08T10:00:00Z"
    guestCount: 4
    specialRequests: "Hosting 10th anniversary celebration"
  }) {
    id
    status  # pending_host_approval
    holdExpiresAt  # 24h from now
  }
}

# Step 2: Host approves (within 24h)
mutation {
  confirmBooking(bookingId: "booking-uuid") {
    id
    status  # awaiting_payment
    paymentDueAt  # 24h from approval
  }
}

# Step 3: Guest pays
mutation {
  payForBooking(input: {
    bookingId: "booking-uuid"
    paymentMethodId: "card-uuid"
  }) {
    booking {
      id
      status  # confirmed
      confirmedAt
    }
    paymentId
    authorizationUrl
  }
}
```

---

## Status States

### State Machine (11 States)

```
┌─────────────────────────────────────────────────────────┐
│                    BOOKING STATUS                       │
└─────────────────────────────────────────────────────────┘

1. DRAFT
   ├─ Initial state for instant bookings
   ├─ Held for payment processing (5-10 minutes)
   └─→ confirmed (payment success) or cancelled (payment failed)

2. PENDING_HOST_APPROVAL
   ├─ Initial state for request bookings
   ├─ Host has 24 hours to respond
   └─→ awaiting_payment (approved) or archived (declined/timeout)

3. AWAITING_PAYMENT
   ├─ Host approved, waiting for guest payment
   ├─ Guest has 24 hours to pay
   └─→ confirmed (paid) or archived (timeout)

4. PAYMENT_FAILED
   ├─ Payment attempt failed
   └─→ awaiting_payment (retry) or cancelled

5. CONFIRMED
   ├─ Booking paid and confirmed
   ├─ Before check-in date
   └─→ active (check-in) or cancelled

6. ACTIVE
   ├─ Guest currently staying
   ├─ Check-in completed, before check-out
   └─→ completed (check-out) or disputed

7. COMPLETED
   ├─ Check-out completed
   ├─ Waiting 48h for dispute window
   └─→ settled (payout processed)

8. SETTLED
   ├─ Final state (successful booking)
   ├─ Host received payout
   └─ Review invitations sent

9. CANCELLED
   ├─ Booking cancelled (guest/host/admin)
   ├─ Refund processed if applicable
   └─ Terminal state

10. DISPUTED
    ├─ Finance dispute filed
    ├─ Escrow frozen
    └─→ settled (resolved) or cancelled

11. ARCHIVED
    ├─ Expired draft/request
    └─ Terminal state
```

### Status Transitions

| From State | To State | Trigger | Conditions |
|------------|----------|---------|------------|
| draft | confirmed | Payment success | Paystack webhook |
| draft | cancelled | Payment failed | Paystack webhook or timeout |
| pending_host_approval | awaiting_payment | Host confirms | Within 24h hold |
| pending_host_approval | archived | Host declines or timeout | > 24h |
| awaiting_payment | confirmed | Guest pays | Payment success |
| awaiting_payment | archived | Timeout | > 24h payment window |
| payment_failed | awaiting_payment | Retry payment | Guest retries |
| confirmed | active | Check-in | Host records or auto |
| confirmed | cancelled | Guest/host cancels | Before check-in |
| active | completed | Check-out | Host records or auto |
| active | disputed | Dispute filed | Guest/host action |
| completed | settled | Payout processed | 48h + finance settled |
| disputed | settled | Dispute resolved | Admin resolution |

---

## Pricing & Quotes

### Quote Calculation

Before booking, guests get a real-time price quote:

```graphql
query {
  quoteBooking(
    listingId: "listing-uuid"
    checkIn: "2026-02-01T14:00:00Z"
    checkOut: "2026-02-08T10:00:00Z"
    guestCount: 3
  ) {
    available
    totalPrice
    currency
    priceBreakdown {
      baseTotal
      cleaningFee
      serviceFee
      cautionFee
      subtotal
      total
      nightlyRates {
        date
        baseRate
        finalRate
      }
      discounts {
        name
        amount
        type
      }
    }
    instantBooking
    minNights
    maxNights
    maxGuests
    unavailabilityReason
  }
}
```

### Price Breakdown Structure

**Example**: 7-night stay in Lekki apartment

```txt
Base Rate: ₦25,000/night × 7 nights = ₦175,000
├─ Night 1 (Fri): ₦30,000 (weekend rate)
├─ Night 2 (Sat): ₦30,000 (weekend rate)
├─ Night 3 (Sun): ₦25,000
├─ Night 4 (Mon): ₦22,000 (weekday discount)
├─ Night 5 (Tue): ₦22,000 (weekday discount)
├─ Night 6 (Wed): ₦22,000 (weekday discount)
└─ Night 7 (Thu): ₦22,000 (weekday discount)

Extra Guest Fee: 1 extra guest × ₦5,000 = ₦5,000

Discounts:
├─ Weekly discount (7+ nights): -₦8,750 (5%)
└─ Early bird (30+ days advance): -₦5,250 (3%)

Fees:
├─ Cleaning fee: ₦10,000 (one-time)
├─ Service fee (guest): ₦19,125 (11% of subtotal)
└─ Caution fee: ₦25,000 (refundable)

Subtotal: ₦175,000 + ₦5,000 - ₦14,000 = ₦166,000
Fees: ₦10,000 + ₦19,125 + ₦25,000 = ₦54,125
TOTAL: ₦220,125

Platform Commission (paid by host):
Host receives: ₦166,000 × 0.90 = ₦149,400 (10% commission)
Platform keeps: ₦16,600
```

### Dynamic Pricing Factors

1. **Time-Based**:
   - Weekday vs. Weekend rates
   - Seasonal pricing (December = peak, June = low season)
   - Special events (Lagos Carnival, New Year)

2. **Booking Window**:
   - Last-minute (< 7 days): +20%
   - Standard (7-30 days): Base rate
   - Early bird (30+ days): -3% discount

3. **Length of Stay**:
   - 1-6 nights: Base rate
   - 7-29 nights: -5% weekly discount
   - 30+ nights: -15% monthly discount

4. **Guest Count**:
   - Up to base guests (e.g., 2): Base rate
   - Extra guests: +₦3,000-₦8,000/night/guest

5. **Occupancy**:
   - Low demand (< 30% booked): -10%
   - High demand (> 80% booked): +15%

---

## Reservation Flow

Same-day bookings: allowed when a listing's calendar config sets `same_day_booking = true` and the configured `lead_time_hours` window is respected (computed in the listing timezone). Otherwise, same-day requests are rejected during validation.

### Instant Booking Flow (Fast Path)

```txt
1. GET QUOTE
   ↓
   query { quoteBooking(...) }
   ├─ Response: { available: true, totalPrice: 220125 }
   │
2. RESERVE & PAY (Single Mutation)
   ↓
   mutation { reserveBooking(...) }
   ├─ Creates booking (status: draft)
   ├─ Blocks calendar dates
   ├─ Initiates Paystack payment
   ├─ Returns authorization URL
   │
3. GUEST PAYS
   ↓
   Guest redirected to Paystack checkout
   ├─ Enters card details
   ├─ Completes payment
   │
4. WEBHOOK (Paystack → Hauslet)
   ↓
   POST /api/webhooks/paystack
   ├─ Verify signature
   ├─ HandlePaymentSuccess()
   │   ├─ Update booking: draft → confirmed
   │   ├─ Finance.RecordCharge(₦220,125)
   │   ├─ Calendar.ConfirmEvent()
   │   ├─ Send confirmation emails (guest + host)
   │   └─ Return success
   │
5. CONFIRMATION
   ↓
   Guest receives:
   ├─ Booking confirmation email
   ├─ Receipt (₦220,125)
   ├─ Host contact info
   └─ Check-in instructions
   
   Host receives:
   ├─ New booking notification
   ├─ Guest details
   └─ Payout estimate (₦149,400 on Day 9)
```

**Timeline**: 2-5 minutes from quote to confirmation

### Request Booking Flow (Approval Path)

```txt
1. GET QUOTE
   ↓
   query { quoteBooking(...) }
   ├─ Response: { available: true, instantBooking: false }
   │
2. REQUEST BOOKING (No Payment Yet)
   ↓
   mutation { requestBooking(...) }
   ├─ Creates booking (status: pending_host_approval)
   ├─ Soft-holds calendar (24h)
   ├─ Sets holdExpiresAt: now + 24h
   └─ Sends email to host
   │
3. HOST REVIEWS (Within 24h)
   ↓
   Host receives email with guest details
   ├─ Guest profile (reviews, verification)
   ├─ Special requests
   └─ Decision buttons [APPROVE] [DECLINE]
   │
   ├─→ HOST APPROVES
   │   ↓
   │   mutation { confirmBooking(...) }
   │   ├─ Status: pending → awaiting_payment
   │   ├─ Sets paymentDueAt: now + 24h
   │   ├─ Confirms calendar block
   │   └─ Emails guest to pay
   │   │
   │   4. GUEST PAYS (Within 24h)
   │   ↓
   │   mutation { payForBooking(...) }
   │   ├─ Initiates payment
   │   ├─ Paystack webhook
   │   ├─ Status: awaiting_payment → confirmed
   │   └─ Same as instant booking flow
   │
   └─→ HOST DECLINES or TIMEOUT (> 24h)
       ↓
       ├─ Status: pending → archived
       ├─ Releases calendar hold
       └─ Emails guest (declined or expired)
```

**Timeline**: 1-48 hours from request to confirmation

---

## Payment Integration

### Payment Providers

**Primary**: Paystack (Nigerian payments)  
**Supported**: Flutterwave (backup/alternative)

**Accepted Methods**:

- Card (Visa, Mastercard, Verve)
- Bank transfer
- USSD
- QR code

### Payment Flow

#### 1. Initiate Payment

```go
paymentInput := PaymentInput{
    BookingID:       booking.ID,
    Amount:          220125 * 100,  // ₦220,125 in kobo
    Currency:        "NGN",
    PayerID:         guestID,
    PayerEmail:      "guest@example.com",
    PayerName:       "Chidi Okafor",
    PaymentMethodID: &savedCardID,  // Optional: Use saved card
    CallbackURL:     "https://hauslet.com/bookings/confirm",
    Description:     "7-night stay in Lekki Phase 1",
}

result := paymentGateway.InitiatePayment(ctx, paymentInput)
// Returns: { paymentID, authorizationURL, reference }
```

#### 2. Guest Completes Payment

Guest redirected to `authorizationURL`:

```sh
https://checkout.paystack.com/xxxxxx
```

#### 3. Webhook Processing

```txt
Paystack → POST /api/webhooks/paystack

Headers:
  x-paystack-signature: sha512(body + secret)

Body:
{
  "event": "charge.success",
  "data": {
    "reference": "hauslet_booking_abc123",
    "amount": 22012500,  // kobo
    "status": "success",
    "metadata": {
      "booking_id": "booking-uuid"
    }
  }
}

Handler:
1. Verify signature (HMAC SHA-512)
2. Extract booking_id from metadata
3. Call HandlePaymentSuccess(bookingID, paymentID)
   ├─ Update booking status
   ├─ Finance.RecordCharge()
   ├─ Calendar.ConfirmEvent()
   └─ Send notifications
```

### Payment Retry

If payment fails, guest can retry:

```graphql
mutation {
  payForBooking(input: {
    bookingId: "booking-uuid"
    paymentMethodId: "different-card-uuid"
  }) {
    booking {
      status  # awaiting_payment → confirmed
    }
    authorizationUrl
  }
}
```

**Retry Window**: 24 hours from approval  
**Max Attempts**: Unlimited (until timeout)

---

## Cancellation & Refunds

### Cancellation Policies

#### 1. Flexible (60% of bookings)

**Rules**:

- Cancel before check-in: Full refund
- Cancel after check-in: No refund

**Use Case**: Short stays, budget properties

#### 2. Moderate (30% of bookings)

**Rules**:

- Cancel 7+ days before: Full refund
- Cancel 3-6 days before: 50% refund
- Cancel < 3 days: No refund

**Use Case**: Standard apartments, most listings

#### 3. Strict (10% of bookings)

**Rules**:

- Cancel 30+ days before: 90% refund (10% fee)
- Cancel 14-29 days: 50% refund
- Cancel < 14 days: No refund

**Use Case**: Luxury properties, high-demand periods

### Cancellation Flow

```graphql
mutation {
  cancelBooking(input: {
    bookingId: "booking-uuid"
    reason: "Family emergency - cannot travel"
  }) {
    id
    status  # cancelled
    cancelledAt
    refundAmount  # Calculated based on policy
  }
}
```

**System Actions**:

1. Calculate refund (policy + timing)
2. Update booking status → cancelled
3. Release calendar dates
4. Queue refund job (Cloud Tasks)
5. Finance.RecordRefund()
6. Paystack.RefundPayment()
7. Email guest confirmation + refund details
8. Email host cancellation notice

### Refund Calculation Example

**Scenario**: Guest cancels 10 days before check-in  
**Policy**: Moderate  
**Original Payment**: ₦220,125

```txt
Cancellation: 10 days before (qualifies for full refund)

Refundable Amount: ₦220,125
├─ Base total: ₦166,000 (100% refund)
├─ Cleaning fee: ₦10,000 (100% refund)
├─ Service fee: ₦19,125 (50% refund = ₦9,563)
└─ Caution fee: ₦25,000 (100% refund)

Total Refund: ₦210,563
Platform keeps: ₦9,562 (service fee)

Processing Time: 5-10 business days
```

### Who Can Cancel?

| Actor | Conditions | Refund Policy |
|-------|-----------|---------------|
| **Guest** | Before check-out | Per listing policy |
| **Host** | Before check-in | Guest gets full refund |
| **Admin** | Any time | Custom refund amount |
| **System** | Payment failed or timeout | Automatic full refund |

---

## Check-In & Check-Out

### Manual Recording (Host Initiated)

#### Check-In

```graphql
mutation {
  checkInBooking(bookingId: "booking-uuid") {
    id
    status  # confirmed → active
    checkIn  # Actual timestamp recorded
    activeAt
  }
}
```

**When**: Guest arrives at property  
**Effect**:

- Status: confirmed → active
- Records actual check-in time
- Sends welcome message to guest
- Notifies host "Guest checked in"

#### Check-Out

```graphql
mutation {
  checkOutBooking(bookingId: "booking-uuid") {
    id
    status  # active → completed
    checkOut  # Actual timestamp recorded
    completedAt
  }
}
```

**When**: Guest leaves property  
**Effect**:

- Status: active → completed
- Records actual check-out time
- Starts 48h payout window
- Finance queues payout for processing
- Sends check-out confirmation

### Automatic Population (Cron Fallback)

**Job**: `AutoPopulateCheckInOut()` (runs daily at 3:00 AM)

**Logic**:

```txt
For all bookings in confirmed status:
  IF current_time >= scheduled_check_in_time:
    - Set checkIn = scheduled_check_in_time
    - Status: confirmed → active

For all bookings in active status:
  IF current_time >= scheduled_check_out_time:
    - Set checkOut = scheduled_check_out_time
    - Status: active → completed
```

**Purpose**: Ensure bookings progress even if hosts don't manually record

---

## GraphQL API Reference

### Queries

#### 1. Quote Booking (Get Price & Availability)

```graphql
query QuoteBooking(
  $listingId: UUID!
  $checkIn: Time!
  $checkOut: Time!
  $guestCount: Int!
) {
  quoteBooking(
    listingId: $listingId
    checkIn: $checkIn
    checkOut: $checkOut
    guestCount: $guestCount
  ) {
    listingId
    checkIn
    checkOut
    guestCount
    available
    totalPrice
    currency
    priceBreakdown {
      baseTotal
      cleaningFee
      serviceFee
      cautionFee
      subtotal
      total
      nightlyRates {
        date
        baseRate
        finalRate
      }
      discounts {
        name
        amount
        type
      }
    }
    instantBooking
    minNights
    maxNights
    maxGuests
    checkInTime
    checkOutTime
    responseWindowHours
    unavailabilityReason
  }
}
```

**Input**:

```json
{
  "listingId": "550e8400-e29b-41d4-a716-446655440000",
  "checkIn": "2026-02-15T14:00:00Z",
  "checkOut": "2026-02-22T10:00:00Z",
  "guestCount": 3
}
```

**Response**:

```json
{
  "data": {
    "quoteBooking": {
      "available": true,
      "totalPrice": 245000.0,
      "currency": "NGN",
      "priceBreakdown": {
        "baseTotal": 210000.0,
        "cleaningFee": 12000.0,
        "serviceFee": 23000.0,
        "subtotal": 222000.0,
        "total": 245000.0
      },
      "instantBooking": true,
      "minNights": 2,
      "maxGuests": 4
    }
  }
}
```

#### 2. Get Booking

```graphql
query GetBooking($id: UUID!) {
  booking(id: $id) {
    id
    listingId
    guestId
    guestName
    guestEmail
    guestCount
    status
    bookingType
    checkIn
    checkOut
    checkInTime
    checkOutTime
    totalPrice
    currency
    specialRequests
    priceBreakdown {
      baseTotal
      cleaningFee
      serviceFee
      total
    }
    confirmedAt
    createdAt
  }
}
```

#### 3. My Bookings (Guest View)

```graphql
query MyBookings($limit: Int, $offset: Int) {
  myBookings(limit: $limit, offset: $offset) {
    id
    listingId
    status
    checkInTime
    checkOutTime
    totalPrice
    currency
    createdAt
  }
}
```

#### 4. Listing Bookings (Host View)

```graphql
query ListingBookings(
  $listingId: UUID!
  $status: BookingStatus
  $limit: Int
  $offset: Int
) {
  listingBookings(
    listingId: $listingId
    status: $status
    limit: $limit
    offset: $offset
  ) {
    id
    guestName
    guestEmail
    guestCount
    status
    checkInTime
    checkOutTime
    totalPrice
    createdAt
  }
}
```

### Mutations

#### 1. Reserve Booking (Instant)

```graphql
mutation ReserveBooking($input: ReserveBookingInput!) {
  reserveBooking(input: $input) {
    booking {
      id
      status
      totalPrice
      currency
    }
    paymentId
    paymentStatus
    paymentReference
    authorizationUrl
    requiresAuthorization
  }
}
```

**Input**:

```json
{
  "input": {
    "listingId": "listing-uuid",
    "checkIn": "2026-02-15T14:00:00Z",
    "checkOut": "2026-02-22T10:00:00Z",
    "guestCount": 3,
    "paymentMethodId": "card-uuid",
    "specialRequests": "Late check-in (9pm arrival)"
  }
}
```

#### 2. Request Booking (Manual Approval)

```graphql
mutation RequestBooking($input: RequestBookingInput!) {
  requestBooking(input: $input) {
    id
    status
    holdExpiresAt
    totalPrice
    currency
  }
}
```

#### 3. Confirm Booking (Host Approval)

```graphql
mutation ConfirmBooking($bookingId: UUID!) {
  confirmBooking(bookingId: $bookingId) {
    id
    status
    paymentDueAt
  }
}
```

#### 4. Pay for Booking

```graphql
mutation PayForBooking($input: PayForBookingInput!) {
  payForBooking(input: $input) {
    booking {
      id
      status
      confirmedAt
    }
    paymentId
    paymentStatus
    authorizationUrl
  }
}
```

#### 5. Cancel Booking

```graphql
mutation CancelBooking($input: CancelBookingInput!) {
  cancelBooking(input: $input) {
    id
    status
    cancelledAt
    refundAmount
  }
}
```

#### 6. Check-In Booking

```graphql
mutation CheckInBooking($bookingId: UUID!) {
  checkInBooking(bookingId: $bookingId) {
    id
    status
    checkIn
    activeAt
  }
}
```

#### 7. Check-Out Booking

```graphql
mutation CheckOutBooking($bookingId: UUID!) {
  checkOutBooking(bookingId: $bookingId) {
    id
    status
    checkOut
    completedAt
  }
}
```

---

## Usage Examples

### Example 1: Guest Books Instant Listing (Success)

```graphql
# Step 1: Get quote to check price
query {
  quoteBooking(
    listingId: "abc-123"
    checkIn: "2026-03-01T14:00:00Z"
    checkOut: "2026-03-04T10:00:00Z"
    guestCount: 2
  ) {
    available  # true
    totalPrice  # 85000
    instantBooking  # true
    priceBreakdown {
      baseTotal  # 60000 (₦20k × 3 nights)
      cleaningFee  # 10000
      serviceFee  # 7700
      total  # 77700
    }
  }
}

# Step 2: Reserve and pay
mutation {
  reserveBooking(input: {
    listingId: "abc-123"
    checkIn: "2026-03-01T14:00:00Z"
    checkOut: "2026-03-04T10:00:00Z"
    guestCount: 2
    paymentMethodId: "card-xyz"
  }) {
    booking {
      id  # "booking-456"
      status  # "draft" (waiting payment)
    }
    paymentId
    authorizationUrl  # Guest redirected here
    requiresAuthorization  # true
  }
}

# Step 3: Payment webhook (automatic)
# Paystack calls: POST /webhooks/paystack
# System: HandlePaymentSuccess("booking-456")

# Result: Booking status changes to "confirmed"
# Emails sent to guest and host
```

### Example 2: Guest Books Request Listing (Multi-Step)

```graphql
# Day 1 - 10:00 AM: Guest requests booking
mutation {
  requestBooking(input: {
    listingId: "xyz-789"
    checkIn: "2026-04-10T15:00:00Z"
    checkOut: "2026-04-17T11:00:00Z"
    guestCount: 4
    specialRequests: "Birthday celebration, need extra decorations"
  }) {
    id  # "booking-def"
    status  # "pending_host_approval"
    holdExpiresAt  # "2026-01-04T10:00:00Z" (24h later)
  }
}

# Day 1 - 2:00 PM: Host approves
mutation {
  confirmBooking(bookingId: "booking-def") {
    id
    status  # "awaiting_payment"
    paymentDueAt  # "2026-01-05T14:00:00Z" (24h from approval)
  }
}

# Day 1 - 3:00 PM: Guest pays
mutation {
  payForBooking(input: {
    bookingId: "booking-def"
    paymentMethodId: "card-abc"
  }) {
    booking {
      id
      status  # "confirmed" (after webhook)
      confirmedAt
    }
    authorizationUrl
  }
}

# Result: Booking confirmed, calendar blocked, emails sent
```

### Example 3: Guest Cancels with Refund

```graphql
# Booking details:
# - Created: Jan 3, 2026
# - Check-in: Feb 15, 2026
# - Total paid: ₦220,125
# - Policy: Moderate (full refund if > 7 days)

# Jan 20, 2026: Guest cancels (26 days before check-in)
mutation {
  cancelBooking(input: {
    bookingId: "booking-ghi"
    reason: "Travel plans changed due to work commitment"
  }) {
    id
    status  # "cancelled"
    cancelledAt
    refundAmount  # 210563 (95% refund)
  }
}

# System actions:
# 1. Status: confirmed → cancelled
# 2. Calendar dates released (available again)
# 3. Finance.RecordRefund(₦210,563)
# 4. Cloud Tasks queues refund job
# 5. Paystack.RefundPayment()
# 6. Email guest: "Refund of ₦210,563 processing (5-10 days)"
# 7. Email host: "Booking cancelled - dates now available"
```

### Example 4: Host Manually Records Check-In/Out

```graphql
# March 1, 2026 - 3:15 PM: Guest arrives
mutation {
  checkInBooking(bookingId: "booking-jkl") {
    id
    status  # confirmed → active
    checkIn  # "2026-03-01T15:15:00Z" (actual time)
    checkInTime  # "2026-03-01T14:00:00Z" (scheduled)
    activeAt
  }
}

# March 4, 2026 - 9:45 AM: Guest departs
mutation {
  checkOutBooking(bookingId: "booking-jkl") {
    id
    status  # active → completed
    checkOut  # "2026-03-04T09:45:00Z" (actual time)
    checkOutTime  # "2026-03-04T10:00:00Z" (scheduled)
    completedAt
  }
}

# March 6, 2026 - Automatic (48h later):
# Cron: CompleteBookings()
# - Finance.RecordPayout(hostID, ₦69,930)
# - Status: completed → settled
# - Email host: "Payout of ₦69,930 sent to your account"
# - Email both: "Leave a review for each other"
```

### Example 5: Query My Upcoming Bookings

```graphql
query {
  myBookings(limit: 10, offset: 0) {
    id
    listingId
    status
    checkInTime
    checkOutTime
    totalPrice
    currency
    guestCount
    specialRequests
    createdAt
  }
}
```

**Response**:

```json
{
  "data": {
    "myBookings": [
      {
        "id": "booking-1",
        "status": "confirmed",
        "checkInTime": "2026-02-15T14:00:00Z",
        "checkOutTime": "2026-02-22T10:00:00Z",
        "totalPrice": 245000,
        "currency": "NGN",
        "guestCount": 3
      },
      {
        "id": "booking-2",
        "status": "pending_host_approval",
        "checkInTime": "2026-03-10T15:00:00Z",
        "checkOutTime": "2026-03-17T11:00:00Z",
        "totalPrice": 320000,
        "currency": "NGN",
        "guestCount": 4
      }
    ]
  }
}
```

---

## Best Practices

### For Guests

#### 1. **Always Get Quote First**

```graphql
# GOOD: Get quote before booking
query { quoteBooking(...) }
# Check: available = true
# Check: totalPrice matches expectation
# Then: reserveBooking(...)

# BAD: Direct booking without quote
mutation { reserveBooking(...) }  # May fail if unavailable
```

#### 2. **Provide Accurate Guest Count**

```
2 guests booked, 4 guests arrive = Host can cancel
Extra guest fees apply (₦3k-₦8k/night/guest)
```

#### 3. **Read Cancellation Policy**

```
Before booking, check listing's refundPolicy:
- flexible: Cancel anytime before check-in
- moderate: Cancel 7+ days for full refund
- strict: Cancel 30+ days for 90% refund
```

#### 4. **Communicate Special Requests**

```graphql
mutation {
  reserveBooking(input: {
    ...
    specialRequests: "Arriving late (11pm), please leave key with security"
  })
}
```

### For Hosts

#### 1. **Choose Appropriate Booking Type**

**Use Instant Booking If**:

- ✅ Calendar always updated
- ✅ Property consistently available
- ✅ Standardized space (apartment)
- ✅ Want higher conversion rates

**Use Request Booking If**:

- ✅ Need to vet guests
- ✅ Shared/unique property
- ✅ Flexible availability
- ✅ Want control over bookings

#### 2. **Respond to Requests Quickly**

```
Request approval window: 24 hours
Best practice: Respond within 2-4 hours
High response rate = better search ranking
```

#### 3. **Record Actual Check-In/Out**

```graphql
# When guest arrives (don't rely on cron)
mutation { checkInBooking(bookingId: "...") }

# When guest leaves
mutation { checkOutBooking(bookingId: "...") }
```

**Benefits**:

- Accurate payout timing (starts 48h from actual check-out)
- Better dispute resolution (timestamps matter)
- Shows professionalism

#### 4. **Set Realistic Pricing**

```
Use dynamic pricing factors:
- Weekend: +20% (Friday-Saturday)
- Weekday: -10% (Monday-Thursday)
- Monthly stays: -15% discount
- Cleaning fee: One-time (₦8k-₦15k)
```

### For Developers

#### 1. **Always Check Status Before Actions**

```go
// GOOD
if booking.CanBeConfirmed() {
    booking.MarkConfirmed(time.Now())
    repo.Update(booking)
}

// BAD
booking.MarkConfirmed(time.Now())  // May violate state machine
```

#### 2. **Handle Payment Webhooks Idempotently**

```go
func HandlePaymentSuccess(bookingID, paymentID uuid.UUID) error {
    booking := repo.GetByID(bookingID)
    
    // Already processed?
    if booking.IsConfirmed() {
        log.Info("duplicate webhook, already confirmed")
        return nil
    }
    
    // Process once
    booking.MarkConfirmed(time.Now())
    repo.Update(booking)
    
    // Finance + notifications
    financeHooks.OnPaymentSucceeded(...)
    notifier.SendConfirmation(...)
    
    return nil
}
```

#### 3. **Validate Guest Count**

```go
constraints := listingHooks.GetListingConstraints(listingID)

if guestCount > constraints.MaxGuests {
    return ErrTooManyGuests
}

if guestCount < 1 {
    return ErrInvalidGuestCount
}
```

#### 4. **Use Transactions for Multi-Step Operations**

```go
func ReserveBooking(...) error {
    return db.Transaction(func(tx *gorm.DB) error {
        // 1. Create booking
        booking := &Booking{...}
        if err := tx.Create(booking).Error; err != nil {
            return err
        }
        
        // 2. Block calendar
        event := &CalendarEvent{...}
        if err := calendar.CreateEvent(event); err != nil {
            return err  // Rollback booking
        }
        
        // 3. Initiate payment
        result, err := payment.InitiatePayment(...)
        if err != nil {
            return err  // Rollback all
        }
        
        return nil
    })
}
```

---

## Troubleshooting

### Issue: Quote Shows Unavailable

**Symptom**: `quoteBooking` returns `available: false`

**Possible Causes**:

1. Dates already blocked by another booking
2. Check-in/out violates min/max nights rules
3. Guest count exceeds max guests
4. Listing is paused/delisted

**Diagnosis**:

```graphql
query {
  quoteBooking(...) {
    available
    unavailabilityReason  # Check this field
  }
}
```

**Common Reasons**:

- "Dates are already booked"
- "Stay must be at least 2 nights"
- "Maximum 4 guests allowed"
- "Check-in on Saturday is not allowed"

### Issue: Payment Stuck in Draft

**Symptom**: Booking remains in `draft` status > 10 minutes

**Possible Causes**:

1. Payment webhook not received (Paystack issue)
2. Guest abandoned payment page
3. Card declined (insufficient funds)

**Solution**:

```graphql
# Check booking status
query {
  booking(id: "booking-uuid") {
    status
    paymentDueAt
    lastPaymentId
  }
}

# If expired, allow retry
mutation {
  payForBooking(input: {
    bookingId: "booking-uuid"
    paymentMethodId: "different-card"
  }) {
    authorizationUrl
  }
}
```

### Issue: Payout Not Received

**Symptom**: Booking marked `settled` but host didn't receive money

**Diagnosis**:

```graphql
query {
  booking(id: "booking-uuid") {
    status  # Should be "settled"
    completedAt
    # Check if 48h passed
  }
}

# Check finance disbursement
query {
  disbursement(id: "disb-uuid") {
    status
    attempts
    failureReason
  }
}
```

**Common Issues**:

- Invalid bank account details (fix in profile)
- Provider outage (auto-retries scheduled)
- Insufficient provider balance (contact support)

### Issue: Refund Processing Slowly

**Symptom**: Cancellation confirmed but refund not in bank account

**Expected Timeline**:

- Paystack: 5-10 business days
- Weekends/holidays: Add 2-3 days
- International cards: 7-14 days

**Verification**:

```graphql
query {
  booking(id: "booking-uuid") {
    status  # "cancelled"
    refundAmount
    refundInitiatedAt
    refundProcessedAt
    refundReference
  }
}
```

If `refundProcessedAt` is null > 14 days: Contact support

---

## Cron Jobs & Automation

### 1. CompleteBookings (Hourly)

**Purpose**: Auto-complete bookings after check-out + 48h

```go
func CompleteBookings(ctx context.Context) error {
    // Find bookings eligible for completion
    cutoff := time.Now().Add(-48 * time.Hour)
    
    bookings := repo.FindCompleted(
        status: "completed",
        completedAt: < cutoff
    )
    
    for _, booking := range bookings {
        // Trigger finance payout
        finance.OnBookingCompleted(booking.ID, booking.HostID)
        
        // Update status
        booking.Status = "settled"
        repo.Update(booking)
        
        // Send notifications
        notifier.SendPayoutConfirmation(booking.HostID, amount)
        reviewHooks.SendReviewInvites(booking.ID)
    }
}
```

**Schedule**: Every hour on the hour

### 2. ArchiveExpiredBookings (Daily)

**Purpose**: Archive draft/pending bookings that expired

```go
func ArchiveExpiredBookings(ctx context.Context) error {
    now := time.Now()
    
    // Archive expired drafts (> 1 hour old)
    drafts := repo.FindExpired(
        status: "draft",
        createdAt: < now.Add(-1 * time.Hour)
    )
    
    // Archive expired requests (> holdExpiresAt)
    requests := repo.FindExpired(
        status: "pending_host_approval",
        holdExpiresAt: < now
    )
    
    // Archive unpaid approvals (> paymentDueAt)
    unpaid := repo.FindExpired(
        status: "awaiting_payment",
        paymentDueAt: < now
    )
    
    for _, booking := range append(drafts, requests, unpaid...) {
        booking.MarkArchived(now)
        calendar.ReleaseEvent(booking.CalendarEventID)
        repo.Update(booking)
    }
}
```

**Schedule**: Daily at 2:00 AM

### 3. AutoPopulateCheckInOut (Daily)

**Purpose**: Auto-mark check-in/out if host didn't record

```go
func AutoPopulateCheckInOut(ctx context.Context) error {
    now := time.Now()
    
    // Auto check-in
    confirmed := repo.FindPastCheckIn(
        status: "confirmed",
        checkInTime: < now
    )
    
    for _, booking := range confirmed {
        booking.CheckIn = booking.CheckInTime
        booking.MarkActive(now)
        repo.Update(booking)
    }
    
    // Auto check-out
    active := repo.FindPastCheckOut(
        status: "active",
        checkOutTime: < now
    )
    
    for _, booking := range active {
        booking.CheckOut = booking.CheckOutTime
        booking.Status = "completed"
        booking.CompletedAt = &now
        repo.Update(booking)
    }
}
```

**Schedule**: Daily at 3:00 AM

---

## Appendix: Database Schema

```sql
CREATE TABLE bookings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id UUID NOT NULL REFERENCES listings(id),
    calendar_event_id UUID NOT NULL,
    cleaning_event_id UUID REFERENCES calendar_events(id),
    
    guest_id UUID NOT NULL REFERENCES users(id),
    guest_name VARCHAR(255) NOT NULL,
    guest_email VARCHAR(255) NOT NULL,
    guest_phone VARCHAR(50),
    guest_count INT NOT NULL CHECK (guest_count > 0),
    
    status VARCHAR(50) NOT NULL,
    booking_type VARCHAR(50) NOT NULL,
    
    -- Actual timestamps (nullable)
    check_in TIMESTAMPTZ,
    check_out TIMESTAMPTZ,
    
    -- Scheduled timestamps
    check_in_time TIMESTAMPTZ,
    check_out_time TIMESTAMPTZ,
    
    hold_expires_at TIMESTAMPTZ,
    payment_due_at TIMESTAMPTZ,
    active_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    archived_at TIMESTAMPTZ,
    
    payment_reference VARCHAR(255),
    last_payment_id UUID,
    
    -- Refund tracking
    refund_amount BIGINT DEFAULT 0,
    refund_initiated_at TIMESTAMPTZ,
    refund_processed_at TIMESTAMPTZ,
    refund_reason TEXT,
    refund_reference VARCHAR(255),
    cancelled_by VARCHAR(20),
    
    special_requests TEXT,
    price_breakdown JSONB NOT NULL,
    total_price DECIMAL(12,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN',
    
    confirmed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    CONSTRAINT check_dates CHECK (check_out_time > check_in_time)
);

-- Indexes
CREATE INDEX idx_bookings_guest ON bookings(guest_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_bookings_listing ON bookings(listing_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_bookings_status ON bookings(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_bookings_check_in ON bookings(check_in_time) WHERE deleted_at IS NULL;
CREATE INDEX idx_bookings_check_out ON bookings(check_out_time) WHERE deleted_at IS NULL;
CREATE INDEX idx_bookings_completed ON bookings(completed_at) WHERE status = 'completed';
CREATE INDEX idx_bookings_payment ON bookings(last_payment_id) WHERE deleted_at IS NULL;
```

---

## Support & Contact

For questions about the Booking System:

- **Technical Issues**: <backend-team@hauslet.com>
- **Payment Problems**: <payments@hauslet.com>
- **Cancellation Support**: <support@hauslet.com>
- **Host Assistance**: <hosts@hauslet.com>

---

**Last Updated**: January 3, 2026  
**Document Version**: 1.0  
**Module Version**: Hauslet Services v1.0  
**Status**: Production Ready ✅
