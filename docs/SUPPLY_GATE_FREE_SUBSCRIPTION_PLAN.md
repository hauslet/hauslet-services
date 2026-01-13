# Supply Gate Free Subscription Auto-Creation Plan

## Overview

When a user assumes a supply role (Agent or Landlord) via `SelectSupplyRoles`, they should automatically receive a free subscription plan. This relaxes the supply gate requirements while maintaining proper access control.

## Current State Analysis

### Supply Gate Flow

1. **Location**: `internal/modules/auth/authorization/supply_gate.go`
2. **Current Checks** (in order):
   - Bypass roles (admin/root)
   - User has supply type (Host, Agent, Landlord, CoHost)
   - User is ID verified
   - Shortlet host bypass (for shortlet listings only)
   - **Subscription check** - Returns `ErrSupplySubscriptionRequired` if no subscription

### Current Issue

- `SelectSupplyRoles` only updates profile user types
- No automatic subscription creation
- Supply gate fails when user tries to create listings without subscription

### Free Plan Configuration

From `config/defaults/promotion.yaml`:

```yaml
subscription_plans:
  free:
    name: "Free Plan"
    enabled: true
    pricing:
      monthly: 0
      yearly: 0
    limits:
      max_listings: 3
      max_photos_per_listing: 20
      max_virtual_tours: 0
      included_featured_per_month: 0
      included_premium_per_month: 0
```

## Solution: Adapter Pattern

### Architecture Decision

Use an **adapter pattern** to inject subscription service into profile service, maintaining clean architecture boundaries.

### Implementation Plan

#### 1. Create Subscription Adapter Interface

**File**: `internal/modules/profile/service/subscription_adapter.go`

```go
package service

import (
    "context"
    "hauslet/internal/modules/promotions/domain"
    "github.com/google/uuid"
)

// SubscriptionAdapter provides subscription operations for profile service
// Interface defined in consuming package (profile) following dependency inversion
type SubscriptionAdapter interface {
    // GetOrCreateFreeSubscription gets existing free subscription or creates one
    GetOrCreateFreeSubscription(ctx context.Context, userID uuid.UUID) (*domain.AgentSubscription, error)
    
    // HasActiveSubscription checks if user has any active subscription
    HasActiveSubscription(ctx context.Context, userID uuid.UUID) (bool, error)
}
```

#### 2. Implement Adapter in Profile Module (Following Existing Pattern)

**File**: `internal/modules/profile/port/hooks/subscription_adapter.go`

**Note**: Following the existing pattern where adapters live in the consuming module's `port/hooks` directory (see `internal/modules/promotions/port/hooks/profile_adapter.go` for reference).

```go
package hooks

import (
    "context"
    "fmt"
    "hauslet/internal/modules/promotions/domain"
    promotionservice "hauslet/internal/modules/promotions/service"
    profileservice "hauslet/internal/modules/profile/service"
    "github.com/google/uuid"
)

// ProfileSubscriptionAdapter implements SubscriptionAdapter for profile service
// Follows the same pattern as PromotionProfileAdapter (adapters in consuming module)
type ProfileSubscriptionAdapter struct {
    subscriptionSvc promotionservice.SubscriptionService
}

// NewProfileSubscriptionAdapter creates a new subscription adapter
func NewProfileSubscriptionAdapter(subscriptionSvc promotionservice.SubscriptionService) profileservice.SubscriptionAdapter {
    return &ProfileSubscriptionAdapter{
        subscriptionSvc: subscriptionSvc,
    }
}

// GetOrCreateFreeSubscription gets or creates a free subscription
func (a *ProfileSubscriptionAdapter) GetOrCreateFreeSubscription(ctx context.Context, userID uuid.UUID) (*domain.AgentSubscription, error) {
    // Check if user already has an active subscription
    existing, err := a.subscriptionSvc.GetUserSubscription(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to check existing subscription: %w", err)
    }
    
    // If user has active subscription, return it
    if existing != nil && existing.IsActive() {
        return existing, nil
    }
    
    // Create free subscription
    result, err := a.subscriptionSvc.CreateSubscription(ctx, service.CreateSubscriptionInput{
        UserID:       userID,
        PlanType:     domain.PlanTypeFree,
        BillingCycle: domain.BillingCycleMonthly,
        StartTrial:   false, // Free plan doesn't need trial
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create free subscription: %w", err)
    }
    
    return result.Subscription, nil
}

// HasActiveSubscription checks if user has active subscription
func (a *ProfileSubscriptionAdapter) HasActiveSubscription(ctx context.Context, userID uuid.UUID) (bool, error) {
    subscription, err := a.subscriptionSvc.GetUserSubscription(ctx, userID)
    if err != nil {
        return false, err
    }
    return subscription != nil && subscription.IsActive(), nil
}
```

#### 3. Update Profile Service Interface

**File**: `internal/modules/profile/service/interface.go`

```go
// Add SubscriptionAdapter interface (similar to ModerationHooks pattern)
type SubscriptionAdapter interface {
    GetOrCreateFreeSubscription(ctx context.Context, userID uuid.UUID) (*promotionsdomain.AgentSubscription, error)
    HasActiveSubscription(ctx context.Context, userID uuid.UUID) (bool, error)
}

// Update ProfileServiceImpl struct
type ProfileServiceImpl struct {
    repo                repository.ProfileRepository
    storage             *storage.R2Storage
    moderationHooks     ModerationHooks
    subscriptionAdapter SubscriptionAdapter // NEW (optional, can be nil)
    notificationService *notification.NotificationService
    log                 *slog.Logger
}
```

#### 4. Update Profile Service Constructor

**File**: `internal/modules/profile/service/service.go`

```go
func NewProfileService(
    repo repository.ProfileRepository,
    storage *storage.R2Storage,
    moderationHooks ModerationHooks,
    subscriptionAdapter SubscriptionAdapter, // NEW (optional parameter)
    notificationService *notification.NotificationService,
    log *slog.Logger,
) ProfileService {
    return &ProfileServiceImpl{
        repo:                repo,
        storage:             storage,
        moderationHooks:     moderationHooks,
        subscriptionAdapter: subscriptionAdapter, // NEW
        notificationService: notificationService,
        log:                 log,
    }
}
```

#### 5. Update SelectSupplyRoles Method

**File**: `internal/modules/profile/service/profile_roles.go`

// Update SelectSupplyRoles method
func (s *ProfileServiceImpl) SelectSupplyRoles(ctx context.Context, userID string, userTypes []domain.UserType) (*domain.Profile, error) {
    // ... existing validation ...

    // Update profile with supply roles
    profile.UserTypes = merged
    
    // Save profile first
    schemaProfile, err := domain.MapProfileToSchema(profile)
    if err != nil {
        return nil, err
    }
    
    if err := s.repo.UpdateProfile(ctx, schemaProfile); err != nil {
        s.log.Error("failed to update supply roles", "user_id", userID, "error", err)
        return nil, err
    }
    
    // NEW: Auto-create free subscription if adapter is available
    if s.subscriptionAdapter != nil {
        userUUID, err := uuid.Parse(userID)
        if err == nil {
            subscription, err := s.subscriptionAdapter.GetOrCreateFreeSubscription(ctx, userUUID)
            if err != nil {
                // Log error but don't fail - subscription creation is best-effort
                s.log.Warn("failed to auto-create free subscription",
                    "user_id", userID,
                    "error", err,
                )
            } else {
                s.log.Info("auto-created free subscription for supply role",
                    "user_id", userID,
                    "subscription_id", subscription.ID,
                    "plan_type", subscription.PlanType,
                )
            }
        }
    }
    
    s.log.Info("updated supply roles", "user_id", userID, "roles", normalized)
    return domain.MapProfileFromSchema(schemaProfile), nil
}

```

#### 6. Wire Adapter in Container
**File**: `cmd/api/server/container.go`

```go
func (c *Container) initProfile() error {
    emailSubject := c.Config.YAML.Queue.Subjects["email"]
    profileRepo := profilerepository.NewProfileRepository(c.DB)
    profileNotificationService := profilenotification.NewNotificationService(
        c.EmailClient,
        c.Queue,
        emailSubject,
        c.Config.App.Client,
        c.Logger,
    )

    moderationAdapter := moderationhooks.NewModerationAdapter(c.ModerationSvc)
    
    // NEW: Create subscription adapter (must be initialized after promotions)
    // Note: initProfile() runs before initPromotions(), so we need to ensure
    // SubscriptionSvc is available. If not, adapter will be nil (optional).
    var subscriptionAdapter profileservice.SubscriptionAdapter
    if c.SubscriptionSvc != nil {
        subscriptionAdapter = profilehooks.NewProfileSubscriptionAdapter(c.SubscriptionSvc)
    }

    c.ProfileSvc = profileservice.NewProfileService(
        profileRepo,
        c.R2,
        moderationAdapter,
        subscriptionAdapter, // NEW: Pass adapter (can be nil)
        profileNotificationService,
        c.Logger,
    )

    return nil
}
```

**Important**: Since `initProfile()` runs before `initPromotions()`, we have two options:

1. **Option A**: Make subscription adapter optional (nil check in SelectSupplyRoles)
2. **Option B**: Reorder initialization (initPromotions before initProfile)

**Recommendation**: Use Option A (optional adapter) for flexibility.

#### 6. Update Supply Gate Logic (Optional Enhancement)

**File**: `internal/modules/auth/authorization/supply_gate.go`

The supply gate already handles `nil` subscriptions correctly (returns error), but we can add a helpful log:

```go
subscription, err := g.subscriptions.GetUserSubscription(ctx, userID)
if err != nil {
    return fmt.Errorf("fetch subscription: %w", err)
}
if subscription == nil {
    // User should have free subscription - this shouldn't happen if auto-creation works
    g.log.Warn("user with supply role has no subscription",
        "user_id", userID,
        "action", action,
    )
    return ErrSupplySubscriptionRequired
}
```

## Usage Tracking Integration Investigation

### Current Integration Points

#### ✅ Well Integrated

1. **Open Houses**: `CanCreateOpenHouse()` checks usage via `usageService.GetOrCreateCurrentUsage()`
2. **Private Showings**: `CanCreatePrivateShowing()` checks usage
3. **Promotions**: `CreateIncludedPromotion()` increments usage via `usageService.IncrementUsage()`
4. **Billing Cycle**: Usage resets automatically on subscription renewal/billing

#### ⚠️ Potential Gaps

1. **Listing Creation**: No usage tracking for listing count
   - **Current**: `CanAddListing()` checks subscription limit but doesn't track usage
   - **Recommendation**: Usage tracking is implicit (count from DB), but consider explicit tracking for analytics

2. **Photo Uploads**: No usage tracking
   - **Current**: `CanAddPhotos()` checks limit but doesn't track
   - **Recommendation**: Track photo count per listing for analytics

3. **Virtual Tours**: No usage tracking
   - **Current**: Limit checked but not tracked
   - **Recommendation**: Add usage tracking for virtual tour creation

### Usage Tracking Flow

```
User Action → Service Method → SubscriptionService Check → UsageService
                                                              ↓
                                                         GetOrCreateCurrentUsage()
                                                              ↓
                                                         Check Quota (limit - used)
                                                              ↓
                                                         IncrementUsage() (if allowed)
```

### Recommendations

1. **Add Usage Tracking for Listings** (Optional):

   ```go
   // In property service after listing creation
   if err := s.subscriptionSvc.UseListing(ctx, userID); err != nil {
       // Log but don't fail - usage tracking is best-effort
   }
   ```

2. **Ensure Usage Resets**: Already implemented in `ProcessBilling()` and `RenewSubscription()`

3. **Add Usage Queries**: Consider adding GraphQL queries to show users their current usage:

   ```graphql
   query {
     mySubscription {
       currentUsage {
         featuredUsed
         premiumUsed
         openHousesUsed
         privateShowingsUsed
       }
     }
   }
   ```

## Promotions vs Subscriptions: Distinctions

### Subscriptions (Agent Subscriptions)

**What**: Monthly/yearly recurring plans that grant access to supply-side features

**When Applied**:

- **Created**: When user selects supply roles OR manually subscribes
- **Checked**: On every supply-side action (create listing, publish, etc.)
- **Billed**: Monthly/yearly based on billing cycle
- **Status**: Active, Trial, Pending, Cancelled

**What They Provide**:

- **Limits**: Max listings, photos, virtual tours
- **Features**: Boolean flags (analytics, lead management, etc.)
- **Included Promotions**: Monthly quotas for featured/premium promotions
- **Viewing Events**: Quotas for open houses and private showings

**Configuration**: `config/defaults/promotion.yaml` → `subscription_plans`

**Code Location**:

- Domain: `internal/modules/promotions/domain/agent_subscription.go`
- Service: `internal/modules/promotions/service/subscription_service.go`
- Repository: `internal/modules/promotions/repository/agent_subscription_repo.go`

**Key Methods**:

- `GetUserSubscription()` - Gets active subscription (returns nil for free tier)
- `CanAddListing()` - Checks listing limit
- `CanUseFeature()` - Checks feature access
- `CanCreateOpenHouse()` - Checks quota + usage

### Promotions (Listing Promotions)

**What**: Pay-per-listing boosts that enhance visibility of specific listings

**When Applied**:

- **Created**: When user pays for promotion OR uses included quota
- **Active**: During promotion period (7, 14, or 30 days)
- **Expires**: Automatically after duration
- **Status**: Pending, Active, Expired, Cancelled

**What They Provide**:

- **Featured**: Top placement in search (₦50k-₦180k)
- **Premium**: Enhanced listing features (₦25k-₦100k)
- **Sponsored**: CPC ads (not implemented yet)

**Configuration**: `config/defaults/promotion.yaml` → `listing_promotions`

**Code Location**:

- Domain: `internal/modules/promotions/domain/listing_promotion.go`
- Service: `internal/modules/promotions/service/promotion_service.go`
- Repository: `internal/modules/promotions/repository/listing_promotion_repo.go`

**Key Methods**:

- `CreatePromotion()` - Creates paid promotion
- `CreateIncludedPromotion()` - Uses subscription quota (no payment)
- `GetActivePromotion()` - Gets active promotion for listing

### Key Differences

| Aspect | Subscriptions | Promotions |
|--------|--------------|------------|
| **Scope** | User/Account level | Listing level |
| **Duration** | Ongoing (monthly/yearly) | Fixed (7-30 days) |
| **Payment** | Recurring | One-time or included |
| **Purpose** | Access to features/limits | Boost listing visibility |
| **Granularity** | Account-wide | Per listing |
| **Auto-renew** | Yes (if enabled) | No (expires) |
| **Usage Tracking** | Monthly quotas | Per promotion |

### How They Work Together

1. **User subscribes** → Gets subscription with limits and included promotions
2. **User creates listing** → Subscription limit checked (`CanAddListing()`)
3. **User promotes listing** → Can use included quota OR pay separately
   - If included: `CreateIncludedPromotion()` → Uses subscription quota
   - If paid: `CreatePromotion()` → Separate payment, doesn't use quota
4. **Usage tracked** → Monthly quotas decremented for included promotions
5. **Reset on billing** → Quotas reset when subscription renews

### Example Flow

```
1. User selects "Agent" role
   → Auto-creates Free subscription (3 listings, 0 featured/month)

2. User creates listing
   → Supply gate checks: Has subscription? ✅
   → Subscription check: Can add listing? ✅ (0/3 used)

3. User wants to promote listing
   → Option A: Use included quota (if available)
   → Option B: Pay ₦50k for featured promotion

4. User upgrades to Basic plan
   → Gets 10 listings, 2 featured/month
   → Usage quotas reset

5. User uses 2 featured promotions
   → Quota: 2/2 used
   → Next promotion requires payment OR wait for next month
```

## Implementation Checklist

### Phase 1: Auto-Create Free Subscription

- [x] Create `SubscriptionAdapter` interface in profile service
- [x] Implement adapter in promotions module
- [x] Update `SelectSupplyRoles()` to call adapter
- [x] Wire adapter in container
- [x] Add tests for auto-creation
- [x] Test supply gate with auto-created subscription

### Phase 2: Usage Tracking Audit

- [x] Review all usage tracking points
- [x] Add usage tracking for listings (optional) - NOT NEEDED: Uses DB count correctly
- [x] Add usage tracking for photos (optional) - NOT NEEDED: Per-listing limit, not quota
- [x] Add usage tracking for virtual tours (optional) - NOT NEEDED: Account limit, not quota
- [x] Add GraphQL queries for usage display - ALREADY EXISTS: getCurrentUsage query
- [x] Document usage tracking flow - COMPLETED: See USAGE_TRACKING_AUDIT_REPORT.md

### Phase 3: Documentation

- [x] Update architecture docs - COMPLETED: Updated PROMOTIONS_GUIDE.md with supply gate, auto-subscription, usage tracking
- [x] Document promotions vs subscriptions - COMPLETED: Created PROMOTIONS_VS_SUBSCRIPTIONS.md with detailed distinction
- [x] Add examples to README - COMPLETED: Added subscription & promotion flow to main README
- [ ] Create migration guide for existing users - SKIPPED: Not needed for this release

## Testing Strategy

### Unit Tests

- Test `SelectSupplyRoles()` creates subscription
- Test adapter handles existing subscriptions
- Test adapter handles errors gracefully
- Test supply gate with free subscription

### Integration Tests

- Test full flow: SelectSupplyRoles → Create Listing
- Test upgrade from free to paid subscription
- Test usage tracking increments correctly
- Test usage resets on billing cycle

### Edge Cases

- User already has paid subscription → Don't create free
- User selects supply role twice → Don't create duplicate
- Subscription creation fails → Log but don't fail profile update
- User cancels subscription → Can still use free tier limits

## Migration for Existing Users

For users who already selected supply roles but don't have subscriptions:

```go
// One-time migration script
func MigrateExistingSupplyUsers(ctx context.Context) error {
    // Get all users with supply types but no subscription
    users := getUsersWithSupplyTypesButNoSubscription(ctx)
    
    for _, userID := range users {
        subscription, err := subscriptionSvc.GetOrCreateFreeSubscription(ctx, userID)
        if err != nil {
            log.Error("failed to create subscription for user", "user_id", userID)
            continue
        }
        log.Info("created free subscription for existing user",
            "user_id", userID,
            "subscription_id", subscription.ID,
        )
    }
    
    return nil
}
```

## Summary

1. **Auto-Create Free Subscription**: Use adapter pattern to inject subscription service into profile service, auto-create on `SelectSupplyRoles()`

2. **Usage Tracking**: Well integrated for promotions and viewing events. Consider adding explicit tracking for listings/photos for analytics.

3. **Promotions vs Subscriptions**:
   - **Subscriptions** = Account-level access (limits, features, quotas)
   - **Promotions** = Listing-level boosts (visibility, placement)
   - Work together: Subscriptions provide included promotion quotas
