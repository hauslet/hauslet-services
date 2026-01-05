# Hauslet Promotions System - Complete Guide

## Table of Contents

1. [Overview](#overview)
2. [Promotion Types](#promotion-types)
3. [Purchase Models](#purchase-models)
4. [Pricing Structure](#pricing-structure)
5. [GraphQL API Reference](#graphql-api-reference)
6. [Usage Examples](#usage-examples)
7. [Best Practices](#best-practices)
8. [Subscription Management](#subscription-management)

---

## Overview

The Hauslet Promotions System helps property owners increase visibility and attract more leads by promoting their sale and rental listings. The system offers two complementary models:

- **Pay-Per-Promotion**: One-time purchases for occasional promotional needs
- **Subscription Plans**: Monthly/annual plans with included promotion quotas for regular sellers

All promotions integrate seamlessly with search algorithms to boost listing visibility and positioning.

---

## Promotion Types

### 1. FEATURED Promotions

**Maximum Visibility Package**

- **Search Boost**: 10x multiplier (highest priority)
- **Placement**: Top of search results and category pages
- **Badge**: Prominent "Featured" badge on listing cards
- **Duration**: 7-30 days
- **Pricing**: ₦50,000 - ₦180,000
- **Best For**: High-value properties, competitive markets, urgent sales

#### Pricing Breakdown

| Duration | Price | Daily Cost |
|----------|-------|------------|
| 7 days   | ₦50,000 | ₦7,143 |
| 14 days  | ₦90,000 | ₦6,429 |
| 30 days  | ₦180,000 | ₦6,000 |

### 2. PREMIUM Promotions

**Enhanced Features Package**

- **Search Boost**: 3x multiplier (high priority)
- **Enhanced Media**: Up to 50 photos (vs. 20 standard)
- **Virtual Tours**: Enabled for immersive property viewing
- **Analytics**: Detailed view/lead tracking
- **Badge**: "Premium" badge on listing cards
- **Duration**: 30-180 days
- **Pricing**: ₦25,000 - ₦100,000
- **Best For**: Long-term rentals, properties needing detailed showcase

#### Pricing Breakdown

| Duration | Price | Daily Cost |
|----------|-------|------------|
| 30 days  | ₦25,000 | ₦833 |
| 60 days  | ₦45,000 | ₦750 |
| 90 days  | ₦65,000 | ₦722 |
| 180 days | ₦100,000 | ₦556 |

---

## Purchase Models

### A. Pay-Per-Promotion (One-Time)

Purchase individual promotions as needed. Ideal for:

- First-time sellers
- Occasional property listings
- Testing promotion effectiveness
- Specific high-value properties

**Process:**

1. Select listing to promote
2. Choose promotion type (FEATURED/PREMIUM)
3. Select duration
4. Complete payment via Paystack
5. Promotion activates immediately upon payment confirmation

### B. Subscription Plans

Subscribe to a plan and receive monthly promotion quotas. Ideal for:

- Real estate agents managing multiple properties
- Property management companies
- Developers with ongoing inventory
- Regular sellers needing consistent visibility

**Process:**

1. Subscribe to a plan (BASIC/PROFESSIONAL/ENTERPRISE)
2. Receive monthly promotion quota
3. Use included promotions without additional payment
4. Quotas reset on billing cycle renewal

---

## Pricing Structure

### Subscription Plans

#### 1. FREE Plan

- **Price**: ₦0/month
- **Included Promotions**: None
- **Features**: Basic listing capabilities
- **Best For**: Casual sellers, testing the platform

#### 2. BASIC Plan

- **Price**: ₦30,000/month or ₦324,000/year (10% discount)
- **Included Promotions**:
  - 2 Featured promotions (14 days each) - Worth ₦180,000
  - 4 Premium promotions (30 days each) - Worth ₦100,000
- **Total Value**: ₦280,000/month
- **Savings**: ₦250,000/month (893% ROI)
- **Best For**: Independent agents with 5-10 active listings

#### 3. PROFESSIONAL Plan

- **Price**: ₦100,000/month or ₦1,080,000/year (10% discount)
- **Included Promotions**:
  - 5 Featured promotions (14 days each) - Worth ₦450,000
  - 10 Premium promotions (30 days each) - Worth ₦250,000
- **Total Value**: ₦700,000/month
- **Savings**: ₦600,000/month (700% ROI)
- **Best For**: Growing agencies with 20-30 active listings

#### 4. ENTERPRISE Plan

- **Price**: ₦250,000/month or ₦2,700,000/year (10% discount)
- **Included Promotions**:
  - 15 Featured promotions (14 days each) - Worth ₦1,350,000
  - 25 Premium promotions (30 days each) - Worth ₦625,000
- **Total Value**: ₦1,975,000/month
- **Savings**: ₦1,725,000/month (790% ROI)
- **Best For**: Large agencies, developers, property management firms

### Value Analysis

**Annual Subscription Savings**:

- BASIC: ₦360,000 saved (1 month free)
- PROFESSIONAL: ₦1,200,000 saved (1 month free)
- ENTERPRISE: ₦3,000,000 saved (1 month free)

---

## GraphQL API Reference

### Mutations

#### 1. Create Pay-Per-Promotion

```graphql
mutation CreatePromotion($input: CreatePromotionInput!) {
  createPromotion(input: $input) {
    promotion {
      id
      type
      startDate
      endDate
      status
      listing {
        id
        title
      }
    }
    paymentURL
  }
}
```

**Input Variables:**

```json
{
  "input": {
    "listingID": "listing-uuid",
    "type": "FEATURED",
    "durationDays": 14
  }
}
```

**Response:**

```json
{
  "data": {
    "createPromotion": {
      "promotion": {
        "id": "promo-uuid",
        "type": "FEATURED",
        "startDate": "2026-01-03T00:00:00Z",
        "endDate": "2026-01-17T00:00:00Z",
        "status": "PENDING_PAYMENT",
        "listing": {
          "id": "listing-uuid",
          "title": "4 Bedroom Duplex in Lekki"
        }
      },
      "paymentURL": "https://checkout.paystack.com/abcd1234"
    }
  }
}
```

#### 2. Create Included Promotion (From Subscription Quota)

```graphql
mutation CreateIncludedPromotion($input: CreateIncludedPromotionInput!) {
  createIncludedPromotion(input: $input) {
    id
    type
    startDate
    endDate
    status
    listing {
      id
      title
    }
  }
}
```

**Input Variables:**

```json
{
  "input": {
    "listingID": "listing-uuid",
    "type": "PREMIUM",
    "durationDays": 30
  }
}
```

**Response:**

```json
{
  "data": {
    "createIncludedPromotion": {
      "id": "promo-uuid",
      "type": "PREMIUM",
      "startDate": "2026-01-03T00:00:00Z",
      "endDate": "2026-02-02T00:00:00Z",
      "status": "ACTIVE",
      "listing": {
        "id": "listing-uuid",
        "title": "3 Bedroom Flat in Ikeja"
      }
    }
  }
}
```

#### 3. Subscribe to Plan

```graphql
mutation Subscribe($input: SubscribeInput!) {
  subscribe(input: $input) {
    subscription {
      id
      plan
      status
      currentPeriodEnd
      featuredQuotaRemaining
      premiumQuotaRemaining
    }
    paymentURL
  }
}
```

**Input Variables:**

```json
{
  "input": {
    "plan": "PROFESSIONAL",
    "billingCycle": "MONTHLY"
  }
}
```

#### 4. Cancel Promotion

```graphql
mutation CancelPromotion($promotionID: ID!) {
  cancelPromotion(promotionID: $promotionID) {
    id
    status
    cancelledAt
  }
}
```

### Queries

#### 1. Get My Active Promotions

```graphql
query MyPromotions {
  myPromotions(status: ACTIVE) {
    id
    type
    startDate
    endDate
    status
    daysRemaining
    searchBoost
    listing {
      id
      title
      address
    }
  }
}
```

#### 2. Get Subscription Status

```graphql
query MySubscription {
  mySubscription {
    id
    plan
    status
    billingCycle
    currentPeriodEnd
    cancelAtPeriodEnd
    featuredQuotaTotal
    featuredQuotaUsed
    featuredQuotaRemaining
    premiumQuotaTotal
    premiumQuotaUsed
    premiumQuotaRemaining
  }
}
```

#### 3. Get Promotion Stats for Listing

```graphql
query ListingPromotionStats($listingID: ID!) {
  listing(id: $listingID) {
    id
    title
    activePromotion {
      id
      type
      daysRemaining
      searchBoost
    }
    promotionHistory {
      id
      type
      startDate
      endDate
      status
    }
  }
}
```

---

## Usage Examples

### Scenario 1: Selling a High-Value Property (₦50M+ House)

**Recommended Approach**: Pay-per-promotion with FEATURED

```graphql
mutation {
  createPromotion(input: {
    listingID: "listing-123",
    type: FEATURED,
    durationDays: 14
  }) {
    promotion {
      id
      endDate
    }
    paymentURL
  }
}
```

**Rationale**:

- Maximum visibility for competitive price point
- 10x search boost ensures top positioning
- 14 days sufficient for serious buyers to find listing
- One-time cost (₦90,000) justified by property value

### Scenario 2: Real Estate Agent with 15 Active Listings

**Recommended Approach**: PROFESSIONAL Plan Subscription

```graphql
mutation {
  subscribe(input: {
    plan: PROFESSIONAL,
    billingCycle: MONTHLY
  }) {
    subscription {
      id
      plan
      featuredQuotaRemaining
      premiumQuotaRemaining
    }
    paymentURL
  }
}
```

**Strategy**:

- Use 5 Featured promotions on top-tier properties
- Use 10 Premium promotions on mid-range rentals
- Rotate promotions as listings sell/rent
- Monthly cost (₦100,000) vs. pay-per-promotion (₦700,000) = 86% savings

**Quarterly Promotion Plan**:

```
Month 1:
- Feature 3 high-value sales (14 days each)
- Promote 6 rental properties (30 days each)

Month 2:
- Feature 2 new sales + extend 1 from Month 1
- Promote 4 new rentals

Month 3:
- Feature 5 new sales (aggressive push)
- Promote remaining inventory
```

### Scenario 3: Long-Term Rental (1-Year Lease)

**Recommended Approach**: Pay-per-promotion with PREMIUM (90 days)

```graphql
mutation {
  createPromotion(input: {
    listingID: "rental-456",
    type: PREMIUM,
    durationDays: 90
  }) {
    promotion {
      id
      endDate
    }
    paymentURL
  }
}
```

**Rationale**:

- PREMIUM provides enhanced media for showcasing amenities
- 90-day duration covers typical search window for tenants
- 3x boost sufficient for rental market competition
- Cost (₦65,000) amortizes to ₦722/day - excellent value

### Scenario 4: Property Developer with 50-Unit Estate

**Recommended Approach**: ENTERPRISE Plan + Strategic Pay-Per-Promotions

```graphql
# Step 1: Subscribe to ENTERPRISE
mutation {
  subscribe(input: {
    plan: ENTERPRISE,
    billingCycle: ANNUAL  # Save ₦3M/year
  }) {
    subscription { id plan }
    paymentURL
  }
}

# Step 2: Use included promotions for ongoing inventory
mutation {
  createIncludedPromotion(input: {
    listingID: "unit-001",
    type: FEATURED,
    durationDays: 14
  }) {
    id
    status
  }
}

# Step 3: Purchase additional promotions for VIP units during launch
mutation {
  createPromotion(input: {
    listingID: "penthouse-001",
    type: FEATURED,
    durationDays: 30
  }) {
    paymentURL
  }
}
```

**Strategy**:

- Annual subscription (₦2,700,000) provides ₦23,700,000 worth of promotions
- Rotate 15 Featured promotions across best units monthly
- Use 25 Premium promotions for standard units
- Purchase additional Featured promotions for penthouse/VIP units during launch events

---

## Best Practices

### For Sale Listings

#### High-Value Properties (₦30M+)

- **Use**: FEATURED promotions (10x boost)
- **Duration**: 14-30 days
- **Timing**: Launch immediately after listing creation
- **Media**: Professional photography, drone shots, video tours
- **Refresh**: If unsold after 30 days, consider repromotion with updated pricing

#### Mid-Range Properties (₦10M-₦30M)

- **Use**: FEATURED (14 days) or PREMIUM (60 days)
- **Decision**: FEATURED if competitive market, PREMIUM if differentiated property
- **Strategy**: Combine with Open House events for maximum impact

#### Budget Properties (Below ₦10M)

- **Use**: PREMIUM promotions (30-60 days)
- **Focus**: Detailed photos showing value-for-money features
- **Market**: First-time buyers who research extensively

### For Rental Listings

#### Short-Term Rentals (Airbnb Style)

- **Use**: PREMIUM promotions (60-90 days)
- **Features**: Leverage 50-photo limit, enable virtual tours
- **Refresh**: Rotate promotions seasonally (peak travel periods)

#### Long-Term Rentals (Annual Leases)

- **Use**: PREMIUM promotions (90-180 days)
- **Duration**: Match to typical tenant search cycles
- **Content**: Focus on neighborhood amenities, commute access

#### Luxury Rentals (₦5M+/year)

- **Use**: FEATURED promotions (14-30 days)
- **Market**: Expatriates, executives relocating
- **Strategy**: Time promotions to align with corporate relocation seasons

### Subscription Strategy

#### When to Subscribe vs. Pay-Per-Promotion

**Subscribe if:**

- Managing 10+ active listings consistently
- Selling/renting 3+ properties per month
- Operating as professional agency or developer
- Need predictable monthly marketing costs

**Pay-Per-Promotion if:**

- Selling personal property (one-time)
- Testing platform effectiveness
- Have sporadic inventory (1-2 properties/year)
- Prefer variable costs tied to specific assets

#### Quota Management Tips

1. **Prioritize High-Value First**: Use Featured quotas on most expensive listings
2. **Stagger Activations**: Don't activate all promotions on day 1 of billing cycle
3. **Monitor Performance**: Track which promotion types drive most leads per property type
4. **Reserve Buffer**: Keep 1-2 Featured promotions unused for unexpected hot properties
5. **Combine with Events**: Activate promotions during Open Houses for compounding effect

### Timing Strategies

#### Best Times to Promote

**For Sales:**

- **January-March**: Post-holiday budget availability, New Year planning
- **August-September**: Pre-end-of-year purchases, back-to-school relocations
- **Avoid**: December (holiday distractions)

**For Rentals:**

- **July-September**: University/school session starts
- **January-February**: New Year relocations, corporate transfers
- **Quarterly**: Corporate relocation cycles (March, June, September, December)

#### Promotion Duration Selection

- **7 Days**: Flash sales, urgent liquidations, time-sensitive offers
- **14 Days**: Standard duration for active markets, most balanced cost/exposure
- **30 Days**: Competitive markets requiring sustained visibility
- **60-90 Days**: Rentals, niche properties, buyer's markets
- **180 Days**: Long-term rentals, commercial properties, patient sellers

### Multi-Listing Optimization

If promoting multiple properties simultaneously:

1. **Stagger Activations**: Launch promotions 3-5 days apart to maintain continuous visibility
2. **Diversify Types**: Mix Featured and Premium based on property tiers
3. **Geographic Spread**: Promote properties across different locations to capture diverse audiences
4. **Price Range Diversity**: Promote mix of high/mid/budget properties to maximize lead quality

Example for 5-property portfolio:

```
Day 1: Featured - Luxury 5BR Ikoyi (₦80M)
Day 4: Premium - 3BR Lekki Rental
Day 7: Featured - 4BR Ajah (₦45M)
Day 10: Premium - 2BR Ikeja Rental
Day 13: Premium - Land in Ibeju-Lekki
```

---

## Subscription Management

### Upgrading Plans

**Upgrades are applied INSTANTLY with proration.**

```graphql
mutation {
  upgradeSubscription(input: {
    newPlan: PROFESSIONAL
  }) {
    subscription {
      id
      plan
      status
    }
  }
}
```

**How Instant Upgrades Work:**

1. **Proration Calculation**: System calculates the prorated difference between your current plan and new plan for the remaining days in your billing cycle.

   ```
   Formula: (newPlanAmount - oldPlanAmount) × (daysRemaining / totalDays)
   ```

2. **Payment Processing**:
   - If proration amount > ₦0, you'll be charged immediately
   - Payment must be **SUCCEEDED** status (not pending) for upgrade to apply
   - 3D Secure payments require webhook confirmation (see [Payment Flows](#payment-status-and-3d-secure) below)

3. **Immediate Benefits**:
   - Plan changes instantly upon successful payment
   - **Usage quotas reset to 0** - you get fresh quotas immediately
   - New limits (max listings, features) apply right away
   - Next billing date remains unchanged

**Example:**

You're on BASIC (₦30,000/month) with 15 days left in your cycle and upgrade to PROFESSIONAL (₦100,000/month):

```
Proration = (₦100,000 - ₦30,000) × (15 / 30) = ₦35,000
```

You pay ₦35,000 now, and your plan upgrades immediately. Your next billing will be ₦100,000 on your original billing date.

**Important Notes:**

- **Card payments** complete instantly if successful
- **3D Secure payments** may require authentication (see Payment Flows section)
- **Usage reset**: All quota counters reset to 0 (you get full new plan quotas immediately)
- Any pending downgrades are automatically cancelled when you upgrade

### Downgrading Plans

**Downgrades are DEFERRED to your next billing cycle.**

```graphql
mutation {
  downgradeSubscription(input: {
    newPlan: BASIC
  }) {
    subscription {
      id
      plan
      pendingPlanType
      pendingPlanScheduledAt
    }
  }
}
```

**How Deferred Downgrades Work:**

1. **Scheduled Change**: Downgrade is scheduled but doesn't apply until your current billing period ends
2. **Keep Current Benefits**: You continue using your current plan's features and quotas until period end
3. **No Immediate Payment**: No charges or refunds at downgrade time
4. **Automatic Application**: On your next billing date, the system:
   - Charges you the new (lower) plan amount
   - Applies the new plan limits and quotas
   - Resets usage counters for the new plan

**Rationale:**

Downgrades are deferred (not instant) because you've already paid for your current plan period. You should get the full value of what you paid for. This prevents "buyer's remorse" scenarios where users accidentally downgrade and lose access to features they've already funded.

**Important Notes:**

- **Cancel pending downgrade**: You can cancel the scheduled downgrade before it takes effect
- **Upgrades override downgrades**: If you upgrade before the downgrade takes effect, the pending downgrade is automatically cancelled
- **No partial refunds**: You're billed for the full current plan period regardless of when you schedule the downgrade
- **Usage quotas**: Continue using current plan quotas - they do NOT prorate down early

### Payment Status and 3D Secure

**Understanding Payment Flows**

The subscription system handles two types of payment flows:

#### 1. Instant Card Payments (Succeeded Immediately)

```
User initiates upgrade → Payment processed → Status: SUCCEEDED → Upgrade applied instantly
```

**Characteristics:**
- Standard card payments without additional authentication
- Completes in seconds
- Upgrade applies immediately upon mutation return
- Most common flow for Nigerian cards

#### 2. 3D Secure / Async Payments (Pending → Succeeded)

```
User initiates upgrade → Payment created → Status: PENDING →
User completes 3DS authentication → Webhook received → Status: SUCCEEDED →
Upgrade applied via webhook handler
```

**Characteristics:**
- Requires additional authentication (OTP, biometric, etc.)
- Payment status is initially `PENDING`
- User redirected to bank's authentication page
- Upgrade completes when webhook confirms payment success
- May take minutes to hours depending on user action

**Error Handling:**

When you call `upgradeSubscription`, you may receive:

**Success Response (Instant):**
```json
{
  "data": {
    "upgradeSubscription": {
      "subscription": {
        "id": "sub-123",
        "plan": "PROFESSIONAL",
        "status": "ACTIVE"
      }
    }
  }
}
```

**Error Response (3DS Required):**
```json
{
  "errors": [{
    "message": "payment requires confirmation - status: PENDING",
    "extensions": {
      "code": "PAYMENT_PENDING",
      "paymentURL": "https://checkout.paystack.com/xyz"
    }
  }]
}
```

**What to do:**
- Redirect user to `paymentURL` to complete authentication
- Listen for payment webhook confirmation
- Upgrade will auto-apply when payment succeeds
- User can check subscription status to confirm upgrade completion

**Webhook Processing:**

Backend listens for Paystack `charge.success` webhooks. When received for an upgrade payment:

1. Verifies payment status is `SUCCEEDED`
2. Extracts subscription and plan info from payment metadata
3. Applies upgrade in database transaction
4. Resets usage quotas
5. Logs completion

**Important:**
- Upgrade mutations ONLY succeed for `SUCCEEDED` payments
- `PENDING` payments must complete via webhook
- This prevents users from accessing upgraded features without confirmed payment

### Cancelling Subscription

```graphql
mutation {
  cancelSubscription(cancelAtPeriodEnd: true) {
    subscription {
      id
      status
      cancelAtPeriodEnd
      currentPeriodEnd
    }
  }
}
```

**Options:**

- `cancelAtPeriodEnd: true` - Continue until billing period ends, then cancel
- `cancelAtPeriodEnd: false` - Cancel immediately (no refund)

**After Cancellation:**

- Active promotions continue until their expiration dates
- Unused quotas are forfeited
- Account reverts to FREE plan
- Can resubscribe anytime (quotas reset to full)

### Checking Quota Usage

```graphql
query {
  mySubscription {
    featuredQuotaTotal
    featuredQuotaUsed
    featuredQuotaRemaining
    premiumQuotaTotal
    premiumQuotaUsed
    premiumQuotaRemaining
    currentPeriodEnd
  }
}
```

**Quota Reset:**

- Quotas reset on billing cycle renewal (monthly/annual anniversary)
- Unused quotas do NOT roll over
- Use-it-or-lose-it policy encourages consistent promotion activity

---

## Frequently Asked Questions

### Q: Can I promote the same listing multiple times?

**A:** Yes, but not simultaneously. You must wait for the current promotion to expire or cancel it before creating a new one for the same listing.

### Q: What happens if I cancel a paid promotion early?

**A:** The promotion ends immediately and the listing returns to standard visibility. Refunds are handled case-by-case (contact support).

### Q: Do subscription quotas roll over?

**A:** No. Unused Featured or Premium quotas expire at the end of each billing period and reset to the plan's allocation.

### Q: Can I use Featured quotas for Premium promotions (or vice versa)?

**A:** No. Featured and Premium quotas are separate and cannot be converted or substituted.

### Q: What's the difference between FEATURED and PREMIUM for search ranking?

**A:** FEATURED provides a 10x search multiplier (highest priority), while PREMIUM provides a 3x multiplier. Featured listings appear above Premium in search results.

### Q: Can I schedule a promotion to start in the future?

**A:** No. Promotions activate immediately upon payment confirmation (pay-per-promotion) or immediately upon creation (included promotions).

### Q: How do I know if my subscription quota is sufficient?

**A:** Monitor your monthly usage in the first 2-3 months. If you consistently max out quotas, consider upgrading. If you consistently have 30%+ unused, consider downgrading.

### Q: Are there discounts for annual subscriptions?

**A:** Yes. Annual subscriptions save 10% (equivalent to 1 month free).

### Q: What payment methods are supported?

**A:** All promotions and subscriptions are processed through Paystack, which accepts: Cards (Visa, Mastercard, Verve), Bank Transfers, USSD, and Mobile Money.

---

## Technical Implementation Notes

### Promotion Lifecycle

```
PENDING_PAYMENT → (payment webhook) → ACTIVE → (duration expires) → EXPIRED
                                    ↓
                              (manual cancel) → CANCELLED
```

### Search Boost Calculation

The `searchBoost` value is applied as a multiplier to the listing's base search score:

```go
finalScore = baseScore * promotionBoost

// Where promotionBoost is:
// - 10.0 for FEATURED promotions
// - 3.0 for PREMIUM promotions
// - 1.0 for non-promoted listings
```

### Webhook Integration

All payment confirmations (for both pay-per-promotions and subscriptions) are processed via Paystack webhooks at:

```
POST /api/webhooks/paystack
```

**Webhook Event Handlers:**

1. **`charge.success`** - Activates pending promotions after payment confirmation
2. **`charge.success` (with upgrade metadata)** - Completes pending subscription upgrades for 3D Secure payments

**Subscription Upgrade Webhook Flow:**

When a payment with `upgrade_type: "instant_proration"` metadata succeeds:

```go
// Webhook handler extracts from payment metadata:
- subscription_id: UUID of subscription being upgraded
- new_plan: Target plan type (BASIC/PROFESSIONAL/ENTERPRISE)
- old_plan: Previous plan type
- proration_amount: Amount charged
- days_remaining: Days left in billing cycle

// Handler then:
1. Validates payment status is SUCCEEDED
2. Applies upgrade to subscription (changes plan, updates limits)
3. Resets usage quotas to 0 (user gets fresh quotas)
4. Saves changes in database transaction (atomic - all or nothing)
5. Logs completion for audit trail
```

**Idempotency:**

Webhook handlers are designed to be idempotent - processing the same webhook multiple times produces the same result. This prevents double-upgrades if Paystack retries webhook delivery.

**Transaction Safety:**

All subscription modifications (upgrade, downgrade, cancellation) are wrapped in database transactions with automatic rollback on errors. This ensures:
- Payment recorded ⇔ Subscription updated (atomic)
- No partial state (either fully upgraded or not upgraded at all)
- Usage counters remain consistent with plan limits

### Database Considerations

- Active promotions are checked on every search query (indexed by `listing_id` and `status`)
- Expired promotions are archived but retained for analytics
- Subscription quotas are tracked in real-time with database transactions to prevent race conditions

---

## Support & Contact

For questions about the Promotions System:

- **Technical Issues**: <backend-team@hauslet.com>
- **Billing Questions**: <billing@hauslet.com>
- **Strategic Consulting**: <sales@hauslet.com>

---

**Last Updated**: January 5, 2026
**Document Version**: 1.1
**Module Version**: Hauslet Services v1.0

**Changelog v1.1 (January 5, 2026):**
- Added instant upgrade implementation with proration details
- Documented deferred downgrade behavior and rationale
- Added comprehensive payment flow documentation (instant vs 3D Secure)
- Documented usage quota reset behavior on upgrades
- Added webhook integration details for async payment completion
- Added transaction safety guarantees
- Clarified payment status requirements for upgrades
