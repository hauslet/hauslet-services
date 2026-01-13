# Promotions vs Subscriptions: Understanding the Difference

**Quick Reference:** This document clarifies the distinction between two core concepts in Hauslet's monetization system.

---

## TL;DR

- **Subscriptions** = Your account's access level (like a gym membership)
- **Promotions** = Boosting specific listings (like paying for an ad)

---

## Subscriptions (Agent Subscriptions)

### What They Are

Monthly or yearly **account-level plans** that grant you access to supply-side features and set limits on what you can do.

### When They Apply

- **Created**: Automatically when you select a supply role (Agent, Landlord, etc.) OR manually subscribe
- **Checked**: Every time you perform a supply-side action (create listing, publish, add photos, etc.)
- **Billed**: Monthly or yearly based on billing cycle
- **Status**: TRIAL, ACTIVE, PAST_DUE, CANCELLED, EXPIRED

### What They Provide

#### 1. Access & Limits

- **Max Listings**: How many active listings you can have (3 for FREE, up to 100 for ENTERPRISE)
- **Photos per Listing**: How many photos each listing can have (20-unlimited)
- **Virtual Tours**: Whether you can add virtual tours
- **Max Virtual Tours**: How many virtual tours per account

#### 2. Monthly Quotas (Consumable)

- **Featured Promotions**: How many included featured promotions per month (0-15)
- **Premium Promotions**: How many included premium promotions per month (0-25)
- **Open Houses**: How many open house events per month (0-unlimited)
- **Private Showings**: How many private showings per month (0-unlimited)

#### 3. Feature Flags (Boolean)

- `analytics_enabled`: Access to advanced analytics
- `lead_management_enabled`: Lead tracking and management
- `priority_support_enabled`: Priority customer support
- `api_access_enabled`: API access for integrations
- `white_label_enabled`: White-label branding (Enterprise)
- `open_house_events_enabled`: Can create open houses
- `private_showings_enabled`: Can create private showings

### Plans & Pricing

| Plan | Price (Monthly) | Max Listings | Featured/Month | Premium/Month |
|------|----------------|--------------|----------------|---------------|
| FREE | ₦0 | 3 | 0 | 0 |
| BASIC | ₦30,000 | 10 | 2 | 4 |
| PROFESSIONAL | ₦100,000 | 30 | 5 | 10 |
| ENTERPRISE | ₦250,000 | 100 | 15 | 25 |

### Configuration

Defined in `config/defaults/promotion.yaml`:

```yaml
subscription_plans:
  free:
    name: "Free Plan"
    pricing:
      monthly: 0
      yearly: 0
    limits:
      max_listings: 3
      max_photos_per_listing: 20
      included_featured_per_month: 0
      included_premium_per_month: 0
```

### Code Location

- **Domain**: `internal/modules/promotions/domain/agent_subscription.go`
- **Service**: `internal/modules/promotions/service/subscription_service.go`
- **Repository**: `internal/modules/promotions/repository/agent_subscription_repo.go`
- **GraphQL**: `internal/modules/promotions/port/graphql/` (queries: `getMySubscription`, mutations: `createSubscription`, `upgradeSubscription`)

---

## Promotions (Listing Promotions)

### What They Are

**Per-listing boosts** that enhance visibility and positioning of specific property listings in search results.

### When They Apply

- **Created**: When you pay for a promotion OR use your subscription's included quota
- **Active**: During the promotion period (7-180 days depending on type)
- **Expires**: Automatically after duration ends
- **Status**: PENDING, ACTIVE, EXPIRED, CANCELLED

### What They Provide

#### 1. Search Visibility Boost

- **FEATURED**: 10x search multiplier (highest priority, top of results)
- **PREMIUM**: 3x search multiplier (high priority, enhanced features)
- **Badges**: Visual badges on listing cards ("Featured" or "Premium")

#### 2. Enhanced Features (Premium Only)

- **Extra Photos**: Up to 50 photos (vs 20 standard)
- **Virtual Tours**: Enabled for immersive viewing
- **Analytics**: Detailed view/lead tracking

### Types & Pricing

#### Featured Promotions

| Duration | Price | Daily Cost | Boost |
|----------|-------|-----------|--------|
| 7 days | ₦50,000 | ₦7,143 | 10x |
| 14 days | ₦90,000 | ₦6,429 | 10x |
| 30 days | ₦180,000 | ₦6,000 | 10x |

#### Premium Promotions

| Duration | Price | Daily Cost | Boost |
|----------|-------|-----------|--------|
| 30 days | ₦25,000 | ₦833 | 3x |
| 60 days | ₦45,000 | ₦750 | 3x |
| 90 days | ₦65,000 | ₦722 | 3x |
| 180 days | ₦100,000 | ₦556 | 3x |

### Configuration

Defined in `config/defaults/promotion.yaml`:

```yaml
listing_promotions:
  featured:
    name: "Featured Listing"
    search_boost: 10.0
    pricing:
      7_days: 50000
      14_days: 90000
      30_days: 180000
```

### Code Location

- **Domain**: `internal/modules/promotions/domain/listing_promotion.go`
- **Service**: `internal/modules/promotions/service/promotion_service.go`
- **Repository**: `internal/modules/promotions/repository/listing_promotion_repo.go`
- **GraphQL**: `internal/modules/promotions/port/graphql/` (queries: `listMyPromotions`, mutations: `createPromotion`, `createIncludedPromotion`)

---

## Key Differences

| Aspect | Subscriptions | Promotions |
|--------|--------------|------------|
| **Scope** | Account-level | Listing-level |
| **Duration** | Ongoing (monthly/yearly) | Fixed (7-180 days) |
| **Payment** | Recurring | One-time or included |
| **Purpose** | Access to features & limits | Boost listing visibility |
| **Granularity** | Applies to entire account | Applies to one listing |
| **Auto-renew** | Yes (if enabled) | No (expires) |
| **Usage Tracking** | Monthly quotas | Per promotion |
| **Can be cancelled** | Yes (affects account) | Yes (affects one listing) |

---

## How They Work Together

### Example Flow

```
1. User selects "Agent" role
   → Auto-creates FREE subscription (3 listings, 0 promotions/month)

2. User creates Listing A, B, C
   → Subscription check: Can add listing? ✅ (3/3 used)
   → Subscription check: Can add 20 photos? ✅

3. User cannot create Listing D
   → Subscription check: Can add listing? ❌ (3/3 limit reached)
   → Error: "Upgrade to BASIC plan to create more listings"

4. User upgrades to BASIC plan
   → Gets 10 listings, 2 featured/month, 4 premium/month
   → Usage quotas reset

5. User wants to promote Listing A
   → Option A: Use included quota (if available)
      → Calls createIncludedPromotion()
      → Uses 1 of 2 featured quota
      → No payment required
   → Option B: Pay separately
      → Calls createPromotion()
      → Pays ₦50,000 for featured
      → Doesn't use quota

6. User creates 2 featured promotions (uses included quota)
   → Quota: 2/2 used
   → Next promotion requires payment OR wait for next month

7. Billing cycle renews (1 month later)
   → Subscription charged ₦30,000
   → Usage quotas reset to 0
   → User gets fresh 2 featured + 4 premium
```

---

## Usage Tracking

### What Gets Tracked Monthly (Resets on Billing Cycle)

The `usage_tracking` table tracks **consumable monthly quotas**:

```go
type UsageTracking struct {
    SubscriptionID      uuid.UUID
    UserID             uuid.UUID
    PeriodStart        time.Time
    PeriodEnd          time.Time

    // Monthly consumables
    FeaturedUsed       int  // Featured promotions used this month
    PremiumUsed        int  // Premium promotions used this month
    OpenHousesUsed     int  // Open houses created this month
    PrivateShowingsUsed int // Private showings created this month
}
```

**When it resets:**
- Monthly subscriptions: Every 30 days from start date
- Annual subscriptions: Every 365 days from start date

**What happens on reset:**
- All counters go back to 0
- You get fresh quotas based on your plan
- Unused quotas DO NOT roll over

### What Gets Tracked Persistently (No Reset)

These are **counted from the database**, not tracked in `usage_tracking`:

- **Listing count**: `SELECT COUNT(*) FROM listings WHERE user_id = ? AND deleted_at IS NULL`
- **Photo count per listing**: `SELECT COUNT(*) FROM listing_photos WHERE listing_id = ?`
- **Virtual tours**: Counted from `virtual_tours` table

**Why separate?**

These are **persistent limits** (max total), not **monthly quotas** (consumption over time).

- If you delete a listing, you get that slot back
- If you remove photos, you can add more
- Doesn't reset monthly - it's always "current count"

---

## Common Scenarios

### Scenario 1: New Agent Starting Out

**User Journey:**

```
1. User registers as Guest
2. User selects "Agent" role via selectSupplyRoles mutation
   → System auto-creates FREE subscription
3. User can now create 3 listings immediately
4. No promotions available (FREE plan = 0 included promotions)
5. User can purchase pay-per-promotion if needed
```

**GraphQL:**

```graphql
mutation {
  selectSupplyRoles(userTypes: [AGENT]) {
    id
    userTypes
  }
}

# System automatically creates FREE subscription behind the scenes

query {
  getMySubscription {
    planType  # Returns: FREE
    maxListings  # Returns: 3
    includedFeaturedPerMonth  # Returns: 0
  }
}
```

### Scenario 2: Active Agent Needs Promotions

**User Journey:**

```
1. User has FREE subscription (3 listings)
2. User wants to promote a high-value property
3. Options:
   a) Pay ₦90,000 for 14-day featured promotion (one-time)
   b) Upgrade to BASIC (₦30,000/month) to get 2 featured/month included
4. User chooses BASIC (more economical if promoting regularly)
5. Uses 1 of 2 included featured promotions
6. Still has 1 featured + 4 premium quotas left this month
```

**GraphQL:**

```graphql
# Option A: Pay-per-promotion
mutation {
  createPromotion(input: {
    listingID: "abc-123"
    type: FEATURED
    duration: 14
  }) {
    promotion { id }
    paymentURL  # Pay ₦90,000
  }
}

# Option B: Subscribe + Use Included
mutation {
  createSubscription(input: {
    planType: BASIC
    billingCycle: MONTHLY
    startTrial: false
  }) {
    subscription { id planType }
    paymentURL  # Pay ₦30,000/month
  }
}

# Then use included quota
mutation {
  createIncludedPromotion(input: {
    listingID: "abc-123"
    type: FEATURED
    duration: 14
  }) {
    id status  # No payment required
  }
}

# Check remaining quota
query {
  getCurrentUsage {
    featuredUsed  # Returns: 1
    premiumUsed   # Returns: 0
  }

  getMySubscription {
    includedFeaturedPerMonth  # Returns: 2
    includedPremiumPerMonth   # Returns: 4
  }
}
```

### Scenario 3: Developer with 50-Unit Estate

**User Journey:**

```
1. User has ENTERPRISE subscription
   → 100 listings, 15 featured/month, 25 premium/month
2. Launches estate with 50 units
3. Creates all 50 listings (under 100 limit)
4. Uses monthly quota strategically:
   - 15 featured on penthouse units
   - 25 premium on standard units
5. Needs more promotions beyond quota
6. Purchases additional featured promotions for VIP units (₦180,000 each)
```

**Economics:**

- Subscription: ₦250,000/month
- Included value: ₦1,975,000/month
  - 15 featured @ ₦90,000 = ₦1,350,000
  - 25 premium @ ₦25,000 = ₦625,000
- Savings: ₦1,725,000/month (790% ROI)
- Additional pay-per-promotions for launch events

---

## Decision Guide

### When to Use Subscriptions

✅ **Subscribe if:**
- You manage 10+ active listings consistently
- You sell/rent 3+ properties per month
- You operate as a professional agency or developer
- You need predictable monthly costs
- You want included promotions quota
- You need viewing events (open houses, private showings)

### When to Use Pay-Per-Promotions

✅ **Pay-per-promotion if:**
- Selling personal property (one-time)
- Testing platform effectiveness
- Have sporadic inventory (1-2 properties/year)
- Already maxed out subscription quota this month
- Need promotions for specific high-value listings beyond quota

### Hybrid Approach (Recommended for Agencies)

Most successful agencies use **both**:

1. **Subscribe** to appropriate tier for base needs
2. **Use included quota** for regular inventory
3. **Purchase additional** promotions for:
   - Launch events
   - High-value properties
   - Months where you exceed quota
   - Special marketing campaigns

---

## Technical Implementation Notes

### Auto-Create Free Subscription

**When:** User selects supply role (Agent, Landlord, Host, CoHost)

**How:**

```go
// internal/modules/profile/service/profile_roles.go

func (s *ProfileServiceImpl) SelectSupplyRoles(ctx context.Context, userID string, userTypes []domain.UserType) (*domain.Profile, error) {
    // Update profile with supply roles
    // ...

    // Auto-create free subscription if adapter is available
    if s.subscriptionAdapter != nil {
        userUUID, _ := uuid.Parse(userID)
        if err := s.subscriptionAdapter.GetOrCreateFreeSubscription(ctx, userUUID); err != nil {
            // Log but don't fail - best-effort
            s.log.Warn("failed to auto-create free subscription", "user_id", userID, "error", err)
        } else {
            s.log.Info("auto-created free subscription for supply role", "user_id", userID)
        }
    }

    return updatedProfile, nil
}
```

**Adapter Pattern:**

- Interface defined in profile service: `SubscriptionAdapter`
- Implementation in promotions module: `ProfileSubscriptionAdapter`
- Injected via dependency inversion
- Handles circular dependency between profile and promotions modules

### Supply Gate Enforcement

**Location:** `internal/modules/auth/authorization/supply_gate.go`

**Checks (in order):**

1. Bypass roles (admin, root, support) → Allow
2. User has supply type (Host, Agent, Landlord, CoHost)? → No: Reject
3. User is ID verified? → No: Reject (for certain operations)
4. Shortlet host bypass (for shortlet listings only)
5. **Subscription check:** Has active subscription? → No: Reject with `ErrSupplySubscriptionRequired`
6. **Limit check:** Under subscription limits? → No: Reject with appropriate error

**Example:**

```go
// Check if user can create listing
subscription, err := g.subscriptions.GetUserSubscription(ctx, userID)
if subscription == nil {
    return ErrSupplySubscriptionRequired  // No subscription
}

// Check listing limit
canAdd, err := g.subscriptions.CanAddListing(ctx, userID)
if !canAdd {
    return fmt.Errorf("listing limit reached: %d/%d", current, subscription.MaxListings)
}
```

---

## Summary

**Subscriptions:**
- Account-level access control
- Set limits and quotas
- Recurring billing
- Grant feature access
- Auto-created on role selection (FREE plan)

**Promotions:**
- Listing-level visibility boosts
- Fixed duration
- One-time payment or use subscription quota
- Enhance search positioning
- Expire automatically

**Together:**
- Subscriptions grant access
- Promotions enhance visibility
- Subscription quotas provide included promotions
- Pay-per-promotions available beyond quota
- Usage tracking ensures fair limits
