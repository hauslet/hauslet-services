# Listing Promotion System Implementation Plan

## 📊 Overview
Comprehensive monetization system for Sales/Rentals listings through pay-per-listing promotions (Featured, Premium) and agent subscription plans (Basic, Professional, Enterprise).

**Business Model:**
- **Shortlets:** Commission-based (15% on bookings) - NO subscription
- **Sales/Rentals:** Monetize via:
  - Featured listings (₦50k-₦180k for 7-30 days)
  - Premium listings (₦25k-₦100k for 30-180 days)
  - Agent subscriptions (₦30k-₦250k/month)

**Subscription Tiers:**
- **Free Tier:** 3 listings, 5 photos/listing, standard support
- **Basic (₦30k/mo):** 5 listings, 15 photos/listing, 2 featured/month
- **Professional (₦100k/mo):** 50 listings, 30 photos/listing, 10 featured + unlimited premium/month
- **Enterprise (₦250k/mo):** Unlimited listings & photos, 15 featured + unlimited premium/month

---

## 🎯 Implementation Status

**Phase:** ✅ **PHASES 1-5 COMPLETE** - System is operational and enforcing limits
**Last Updated:** 2026-01-01
**Current State:** Backend complete, GraphQL API live, service layer enforcing subscription limits

### ✅ **Completed Phases**

#### **Phase 1: Foundation** ✅ COMPLETE
- ✅ Domain models (ListingPromotion, AgentSubscription, UsageTracking)
- ✅ GORM schemas with proper indexing (includes pending plan change fields)
- ✅ Repository implementations
- ✅ Mappers for domain ↔ schema conversion
- ✅ Database migrations created (including 011_add_pending_plan_change_fields.sql)

#### **Phase 2: Core Services** ✅ COMPLETE
- ✅ PromotionService (create, cancel, start, expire promotions)
- ✅ SubscriptionService (create, upgrade, downgrade, billing)
- ✅ UsageService (quota tracking, monthly resets)
- ✅ Payment orchestration (service layer creates payments, returns URLs)
- ✅ Upgrade/downgrade flow (Option A: schedules for next billing cycle)

#### **Phase 3: Property Module Integration** ✅ COMPLETE
- ✅ Subscription service injected into property service
- ✅ Listing creation enforcement (`CanAddListing`)
- ✅ Photo upload enforcement (`CanAddPhotos`)
- ✅ Service layer validates all limits before operations

#### **Phase 4: GraphQL API** ✅ COMPLETE
- ✅ Full schema (17 queries, 12 mutations)
- ✅ Resolvers for promotions and subscriptions
- ✅ Payment integration via webhooks
- ✅ Type mappings in gqlgen.yml
- ✅ Wired into main GraphQL server

#### **Phase 5: Worker Jobs** ✅ COMPLETE
- ✅ Promotion expiry handler (cron job)
- ✅ Subscription billing processor (daily cron)
- ✅ Payment retry queue handlers
- ✅ Webhook handlers for payment events
- ✅ Usage reset automation

---

## 🏗️ **Architecture: Service Layer Enforcement Pattern**

### **Core Principle: "Service Layer is King"**

All subscription limit checks happen in the **service layer**, not in GraphQL resolvers or HTTP handlers. This ensures:
- ✅ Consistent enforcement across all entry points (GraphQL, REST, internal calls)
- ✅ Business logic centralized and testable
- ✅ GraphQL/HTTP layers remain thin adapters

### **Integration Pattern**

```
User Request (GraphQL/HTTP)
    ↓
Resolver/Handler (validates input, extracts userID)
    ↓
Service Method (CHECKS LIMITS HERE ✅)
    ↓
Repository (persists data)
```

### **Key Services for Integration**

**1. SubscriptionService Interface** (`internal/modules/promotions/service/interface.go`)

```go
// Limit Checks
CanAddListing(ctx, userID) (bool, error)
CanAddPhotos(ctx, userID, listingID, photoCount) (bool, error)
CanUseFeature(ctx, userID, feature) (bool, error)

// Viewing Events (for booking module)
CanCreateOpenHouse(ctx, userID) (bool, remaining int, error)
CanCreatePrivateShowing(ctx, userID) (bool, remaining int, error)

// Usage Tracking
UseOpenHouse(ctx, userID) error
UsePrivateShowing(ctx, userID) error

// Subscription Management
GetUserSubscription(ctx, userID) (*AgentSubscription, error)
UpgradeSubscription(ctx, subscriptionID, newPlan) error
DowngradeSubscription(ctx, subscriptionID, newPlan) error
```

---

## 📍 **Integration Points - Where to Add Checks**

### **✅ 1. Listing Creation** - IMPLEMENTED
**File:** `internal/modules/property/service/listing_crud.go:24-33`

```go
func (s *ServiceImpl) CreateListing(ctx context.Context, l domain.Listing) (*domain.Listing, error) {
    // ✅ Check subscription limit
    canCreate, err := s.subscriptionService.CanAddListing(ctx, l.OwnerID)
    if err != nil {
        return nil, err
    }
    if !canCreate {
        return nil, fmt.Errorf("listing limit reached - upgrade your subscription")
    }

    // Continue with listing creation...
}
```

**What it does:**
- Checks if user has reached listing limit (Free: 3, Basic: 5, Pro: 50, Enterprise: unlimited)
- Blocks creation if limit exceeded
- Returns clear upgrade message

---

### **✅ 2. Photo Upload** - IMPLEMENTED
**File:** `internal/modules/property/service/listing_media.go:34-74`

```go
func (s *ServiceImpl) UploadListingMedia(ctx context.Context, listingID uuid.UUID, media []domain.ListingMediaInput) ([]domain.ListingMediaResult, error) {
    listing, err := s.ensureListing(ctx, listingID, true)

    // ✅ Count existing + new photos
    existingMedia, _ := s.repo.ListListingMedia(ctx, listingID)
    existingPhotoCount := countPhotos(existingMedia)
    newPhotoCount := countPhotos(media)
    totalPhotoCount := existingPhotoCount + newPhotoCount

    // ✅ Check subscription limit
    canAdd, err := s.subscriptionService.CanAddPhotos(ctx, listing.OwnerID, listingID, totalPhotoCount)
    if !canAdd {
        return nil, fmt.Errorf("photo limit exceeded - your plan allows fewer photos")
    }

    // Continue with upload...
}
```

**What it does:**
- Counts existing photos in listing
- Adds new photos being uploaded
- Checks total against subscription limit (Free: 5, Basic: 15, Pro: 30, Enterprise: unlimited)
- Blocks upload if limit exceeded

---

### **❌ 3. Open House Creation** - NOT YET IMPLEMENTED
**Target File:** `internal/modules/booking/service/viewing_events.go` (when created)

```go
func (s *BookingServiceImpl) CreateOpenHouse(ctx context.Context, input CreateOpenHouseInput) (*OpenHouse, error) {
    // ✅ ADD THIS CHECK
    canCreate, remaining, err := s.subscriptionService.CanCreateOpenHouse(ctx, input.UserID)
    if err != nil {
        return nil, err
    }
    if !canCreate {
        return nil, fmt.Errorf("open house quota exceeded - %d remaining this month", remaining)
    }

    // Create open house...

    // ✅ Track usage
    if err := s.subscriptionService.UseOpenHouse(ctx, input.UserID); err != nil {
        // Log but don't fail
    }

    return openHouse, nil
}
```

**What it does:**
- Checks monthly open house quota (Free: 0, Basic: 2, Pro: 10, Enterprise: unlimited)
- Returns remaining count for UI display
- Increments usage counter after successful creation

---

### **❌ 4. Private Showing Creation** - NOT YET IMPLEMENTED
**Target File:** `internal/modules/booking/service/viewing_events.go` (when created)

```go
func (s *BookingServiceImpl) CreatePrivateShowing(ctx context.Context, input CreatePrivateShowingInput) (*PrivateShowing, error) {
    // ✅ ADD THIS CHECK
    canCreate, remaining, err := s.subscriptionService.CanCreatePrivateShowing(ctx, input.UserID)
    if err != nil {
        return nil, err
    }
    if !canCreate {
        return nil, fmt.Errorf("private showing quota exceeded - %d remaining this month", remaining)
    }

    // Create private showing...

    // ✅ Track usage
    if err := s.subscriptionService.UsePrivateShowing(ctx, input.UserID); err != nil {
        // Log but don't fail
    }

    return showing, nil
}
```

**What it does:**
- Checks monthly private showing quota (Free: 0, Basic: 0, Pro: 20, Enterprise: unlimited)
- Returns remaining count for UI display
- Increments usage counter after successful creation

---

### **❌ 5. Feature-Gated Access** - NOT YET IMPLEMENTED
**Pattern for any premium feature:**

```go
func (r *Resolver) GetAnalyticsDashboard(ctx context.Context) (*Analytics, error) {
    userID, _ := getUserIDFromContext(ctx)

    // ✅ ADD THIS CHECK
    hasAccess, err := r.subscriptionService.CanUseFeature(ctx, userID, "analytics_dashboard")
    if err != nil {
        return nil, err
    }
    if !hasAccess {
        return nil, fmt.Errorf("analytics dashboard not available - upgrade to Professional or higher")
    }

    // Return analytics...
}
```

**Features to gate:**
- `analytics_dashboard` - Professional+
- `priority_support` - Basic+
- `api_access` - Enterprise only
- `custom_branding` - Enterprise only
- `virtual_tour_support` - Professional+

---

## 🔧 **Service Wiring Pattern**

### **Step 1: Add to Service Struct**
```go
type ServiceImpl struct {
    // ... existing fields
    subscriptionService promotionservice.SubscriptionService
}
```

### **Step 2: Update Constructor**
```go
func NewService(
    // ... existing params
    subscriptionService promotionservice.SubscriptionService,
) *ServiceImpl {
    return &ServiceImpl{
        // ... existing fields
        subscriptionService: subscriptionService,
    }
}
```

### **Step 3: Wire in Container**
```go
// cmd/api/server/container.go
service := NewService(
    // ... existing params
    container.SubscriptionSvc,
)
```

---

## 📊 **Subscription Limits Reference**

| Feature | Free Tier | Basic (₦30k/mo) | Professional (₦100k/mo) | Enterprise (₦250k/mo) |
|---------|-----------|-----------------|-------------------------|----------------------|
| **Max Listings** | 3 | 5 | 50 | Unlimited |
| **Photos per Listing** | 5 | 15 | 30 | Unlimited |
| **Virtual Tours** | 0 | 0 | 5 | Unlimited |
| **Featured/Month** | 0 (pay) | 2 included | 10 included | 15 included |
| **Premium/Month** | 0 (pay) | 0 (pay) | Unlimited | Unlimited |
| **Open Houses/Month** | 0 | 2 | 10 | Unlimited |
| **Private Showings/Month** | 0 | 0 | 20 | Unlimited |
| **Analytics Dashboard** | ❌ | ❌ | ✅ | ✅ |
| **Priority Support** | ❌ | ✅ | ✅ | ✅ |
| **API Access** | ❌ | ❌ | ❌ | ✅ |
| **Custom Branding** | ❌ | ❌ | ❌ | ✅ |

---

## 🚀 **Upgrade/Downgrade Flow (Option A)**

### **How Upgrades Work:**
1. User calls `UpgradeSubscription` mutation with new plan
2. Service schedules plan change for next billing cycle:
   - Sets `PendingPlanType` = new plan
   - Sets `PendingPlanScheduledAt` = now
3. User continues on current plan until billing date
4. On billing cycle:
   - System detects pending plan change
   - Charges new plan amount
   - On successful payment, applies new limits
   - Clears pending change fields

### **Example Timeline:**
```
Day 1: User on Basic (₦30k/mo), next billing = Day 30
Day 15: User requests upgrade to Professional (₦100k/mo)
  → System schedules upgrade for Day 30
  → User still has Basic limits (5 listings)
Day 30: Billing runs
  → Charges ₦100k (Professional plan)
  → Applies Professional limits (50 listings)
  → User now has Professional plan
```

### **Benefits:**
- ✅ No prorated billing complexity
- ✅ Simple to understand for users
- ✅ No immediate payment required
- ✅ Graceful transition at billing boundary

---

## 📝 **Next Steps for Full Integration**

### **High Priority (Not Yet Implemented):**
1. **Open House Integration**
   - Add `CanCreateOpenHouse` check to booking service
   - Add `UseOpenHouse` to track usage
   - Target: `internal/modules/booking/service/` (when viewing events module is created)

2. **Private Showing Integration**
   - Add `CanCreatePrivateShowing` check to booking service
   - Add `UsePrivateShowing` to track usage
   - Target: `internal/modules/booking/service/`

### **Medium Priority:**
3. **Feature Gating**
   - Analytics dashboard: Add `CanUseFeature(ctx, userID, "analytics_dashboard")` check
   - Virtual tours: Add `CanUseFeature(ctx, userID, "virtual_tour_support")` check
   - API access: Add `CanUseFeature(ctx, userID, "api_access")` check

### **Low Priority:**
4. **Included Promotions**
   - Implement `UseIncludedPromotion` (currently TODO)
   - Implement `CanUseIncludedPromotion` (currently TODO)
   - Track monthly usage of included Featured/Premium promotions

---

## 🧪 **Testing Checklist**

### **Implemented & Tested:**
- ✅ Free tier user blocked at 4th listing
- ✅ Basic tier user blocked at 16th photo (limit: 15)
- ✅ Professional tier user creates 48 listings (within 50 limit)
- ✅ Subscription upgrade scheduled for next billing cycle
- ✅ Billing applies pending upgrades and charges new amount
- ✅ Monthly usage resets on 1st of month

### **To Be Tested (When Features Implemented):**
- ⏳ Open house quota enforcement
- ⏳ Private showing quota enforcement
- ⏳ Feature access gating (analytics, API, etc.)
- ⏳ Included promotion quota tracking

---

## 🏛️ **System Architecture**

```
┌─────────────────────────────────────────────────────────────┐
│                     GraphQL/HTTP Layer                       │
│  (Thin - validates input, delegates to services)            │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                    Service Layer ⭐                          │
│  ✅ ENFORCES ALL SUBSCRIPTION LIMITS HERE                    │
│  - Property Service (listings, photos)                      │
│  - Booking Service (open houses, showings)                  │
│  - Promotion Service (promotions)                           │
│  - Subscription Service (plans, billing)                    │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                  Repository Layer                            │
│  (Data access - GORM, PostgreSQL)                           │
└─────────────────────────────────────────────────────────────┘
```

**Key Tables:**
- `listing_promotions` - Paid promotions for listings
- `agent_subscriptions` - User subscription plans
- `usage_trackings` - Monthly quota usage (resets monthly)

---

## 📌 **Configuration**

**File:** `config/defaults/promotion.yaml`

Contains all plan limits, pricing, features. Service layer loads this config at startup and uses it for:
- Plan limit enforcement
- Payment amount calculation
- Feature access checks
- Free tier defaults

**Example:**
```yaml
plans:
  basic:
    monthly_price: 3000000  # ₦30,000 in kobo
    max_listings: 5
    max_photos_per_listing: 15
    included_featured_per_month: 2
```

---

## 🎯 **Critical Success Factors**

1. ✅ **Service Layer Enforcement:** All checks in service layer - ACHIEVED
2. ✅ **Payment Integration:** Promotions activate after payment - ACHIEVED
3. ✅ **Billing Automation:** Subscriptions renew automatically - ACHIEVED
4. ✅ **Upgrade Flow:** Option A implemented (scheduled for next cycle) - ACHIEVED
5. ⏳ **Complete Integration:** Need to add open house/showing checks
6. ⏳ **Feature Gating:** Need to add premium feature checks

---

**Last Updated:** 2026-01-01
**Status:** System ready for deployment - run migration 011, then deploy

**Deployment Checklist:**

- [ ] Run database migration: `011_add_pending_plan_change_fields.sql`
- [ ] Deploy API server with GraphQL layer
- [ ] Deploy worker with billing/expiry jobs
- [ ] Verify subscription limits enforcement on staging

**Next Milestone:** Integrate viewing events when booking module is ready
