# Listing Promotion System Implementation Plan

## 📊 Overview
Building a comprehensive monetization system for Sales/Rentals listings through pay-per-listing promotions (Featured, Premium) and agent subscription plans (Basic, Professional, Enterprise). This system enables platform revenue from listing visibility while maintaining free tier for individual landlords.

**Business Model:**
- **Shortlets:** Commission-based revenue (15% on bookings) - NO subscription
- **Sales/Rentals:** Static pricing, no recurring revenue → Monetize via:
  - Featured listings (₦50k-₦180k for 7-30 days)
  - Premium listings (₦25k-₦100k for 30-180 days)
  - Agent subscriptions (₦30k-₦250k/month for agencies)

## 🎯 Current Status
**Phase:** Phase 0 - Planning & Design
**Last Updated:** 2025-12-30
**Started By:** Claude Code Session
**Configuration:** YAML config created at `config/defaults/promotion.yaml`

---

## 📋 Configuration Foundation

### YAML Configuration Structure ✅ COMPLETE
**File:** `config/defaults/promotion.yaml`

**Key Configuration Sections:**
1. **Free Tier Limits**
   - 3 active listings max
   - 20 photos per listing
   - No analytics, virtual tours, or API access
   - Standard support only

2. **Pay-Per-Listing Promotions**
   - **Featured Listing:** Top placement, ₦50k-₦180k (7-30 days)
   - **Premium Listing:** Enhanced features, ₦25k-₦100k (30-180 days)
   - **Sponsored Listing:** CPC model (disabled, future feature)

3. **Agent Subscription Plans**
   - **Basic:** ₦30k/month, 10 listings, 2 featured/month included
   - **Professional:** ₦100k/month, 25 listings, 5 featured + 10 premium/month
   - **Enterprise:** ₦250k/month, unlimited listings, 15 featured + unlimited premium

4. **Add-Ons**
   - Professional photography: ₦15,000
   - Virtual tour creation: ₦20,000
   - Listing video: ₦30,000
   - Photo boost (+20 photos): ₦5,000

5. **Billing Settings**
   - 14-day free trial for subscriptions
   - 3-day grace period after payment failure
   - Auto-renew enabled by default
   - Proration on upgrades/downgrades

6. **Analytics Tracking**
   - Track: impressions, clicks, leads, messages
   - 365-day retention
   - Conversion funnel: impression → click → contact → inquiry → viewing → conversion

7. **Rate Limiting**
   - Free tier: 50 leads/day, 0 promotions
   - Paid tier: unlimited leads, unlimited promotions
   - 1 active promotion per listing
   - 24-hour minimum promotion duration

---

## Phase 1: Foundation - Domain & Repository

**Goal:** Build core data models and persistence layer for promotions and subscriptions
**Estimated Files:** ~12 files, ~800-1000 LOC
**Dependencies:** None

### 1.1 Domain Models

- [ ] Create `internal/modules/listing-promotion/domain/enums.go`
  ```go
  // Promotion Types
  type PromotionType string
  const (
      PromotionTypeFeatured  PromotionType = "featured"
      PromotionTypePremium   PromotionType = "premium"
      PromotionTypeSponsored PromotionType = "sponsored" // Future
  )

  // Promotion Status
  type PromotionStatus string
  const (
      PromotionStatusPending   PromotionStatus = "pending"   // Payment pending
      PromotionStatusActive    PromotionStatus = "active"    // Currently running
      PromotionStatusExpired   PromotionStatus = "expired"   // Time elapsed
      PromotionStatusCancelled PromotionStatus = "cancelled" // User cancelled
      PromotionStatusFailed    PromotionStatus = "failed"    // Payment failed
  )

  // Subscription Plan Type
  type PlanType string
  const (
      PlanTypeBasic        PlanType = "basic"
      PlanTypeProfessional PlanType = "professional"
      PlanTypeEnterprise   PlanType = "enterprise"
  )

  // Subscription Status
  type SubscriptionStatus string
  const (
      SubscriptionStatusTrial    SubscriptionStatus = "trial"     // Free trial
      SubscriptionStatusActive   SubscriptionStatus = "active"    // Paid and active
      SubscriptionStatusPastDue  SubscriptionStatus = "past_due"  // Payment failed, in grace period
      SubscriptionStatusCancelled SubscriptionStatus = "cancelled" // Cancelled
      SubscriptionStatusExpired  SubscriptionStatus = "expired"   // Expired after grace period
  )

  // Billing Cycle
  type BillingCycle string
  const (
      BillingCycleMonthly BillingCycle = "monthly"
      BillingCycleYearly  BillingCycle = "yearly"
  )

  // Usage Type (for tracking included promotions)
  type UsageType string
  const (
      UsageTypeFeatured UsageType = "featured"
      UsageTypePremium  UsageType = "premium"
  )
  ```

- [ ] Create `internal/modules/listing-promotion/domain/listing_promotion.go`
  ```go
  type ListingPromotion struct {
      ID          uuid.UUID       `json:"id"`
      ListingID   uuid.UUID       `json:"listing_id"`
      OwnerID     uuid.UUID       `json:"owner_id"`

      Type        PromotionType   `json:"type"`
      Status      PromotionStatus `json:"status"`

      // Pricing
      Amount      int64           `json:"amount"`       // In minor units
      Currency    string          `json:"currency"`
      Duration    int             `json:"duration"`     // Duration in days

      // Timestamps
      StartedAt   *time.Time      `json:"started_at,omitempty"`
      ExpiresAt   *time.Time      `json:"expires_at,omitempty"`

      // Payment
      PaymentID   *uuid.UUID      `json:"payment_id,omitempty"`

      // Subscription Integration
      SubscriptionID *uuid.UUID   `json:"subscription_id,omitempty"` // If from included quota
      IsIncluded     bool         `json:"is_included"`                // From subscription quota

      // Metadata
      BoostMultiplier float64     `json:"boost_multiplier"`          // Search ranking boost
      Metadata        map[string]interface{} `json:"metadata,omitempty"`

      CreatedAt   time.Time       `json:"created_at"`
      UpdatedAt   time.Time       `json:"updated_at"`
      DeletedAt   *time.Time      `json:"deleted_at,omitempty"`
  }

  // Business logic methods
  func (p *ListingPromotion) IsActive() bool
  func (p *ListingPromotion) CanBeCancelled() bool
  func (p *ListingPromotion) Start() error
  func (p *ListingPromotion) Expire() error
  func (p *ListingPromotion) Cancel() error
  func (p *ListingPromotion) CalculateSearchBoost() float64
  ```

- [ ] Create `internal/modules/listing-promotion/domain/agent_subscription.go`
  ```go
  type AgentSubscription struct {
      ID             uuid.UUID          `json:"id"`
      UserID         uuid.UUID          `json:"user_id"`
      PlanType       PlanType           `json:"plan_type"`
      Status         SubscriptionStatus `json:"status"`

      // Billing
      BillingCycle   BillingCycle       `json:"billing_cycle"`
      Amount         int64              `json:"amount"`       // In minor units
      Currency       string             `json:"currency"`
      NextBillingDate *time.Time        `json:"next_billing_date,omitempty"`

      // Trial
      TrialEndsAt    *time.Time         `json:"trial_ends_at,omitempty"`

      // Lifecycle
      StartedAt      time.Time          `json:"started_at"`
      CancelledAt    *time.Time         `json:"cancelled_at,omitempty"`
      ExpiredAt      *time.Time         `json:"expired_at,omitempty"`

      // Limits (cached from config for this subscription)
      MaxListings            int   `json:"max_listings"`
      MaxPhotosPerListing    int   `json:"max_photos_per_listing"`
      MaxVirtualTours        int   `json:"max_virtual_tours"` // -1 = unlimited

      // Included Promotions Per Month
      IncludedFeaturedPerMonth int `json:"included_featured_per_month"`
      IncludedPremiumPerMonth  int `json:"included_premium_per_month"` // -1 = unlimited

      // Features (cached from config)
      Features       map[string]bool    `json:"features"`

      CreatedAt      time.Time          `json:"created_at"`
      UpdatedAt      time.Time          `json:"updated_at"`
      DeletedAt      *time.Time         `json:"deleted_at,omitempty"`
  }

  // Business logic methods
  func (s *AgentSubscription) IsActive() bool
  func (s *AgentSubscription) IsInTrial() bool
  func (s *AgentSubscription) CanUpgrade() bool
  func (s *AgentSubscription) CanDowngrade() bool
  func (s *AgentSubscription) Cancel() error
  func (s *AgentSubscription) Renew(nextBillingDate time.Time) error
  func (s *AgentSubscription) GetFeature(name string) bool
  func (s *AgentSubscription) HasUnlimitedListings() bool
  func (s *AgentSubscription) HasUnlimitedPremium() bool
  ```

- [ ] Create `internal/modules/listing-promotion/domain/usage_tracking.go`
  ```go
  // Tracks monthly usage of included promotions
  type UsageTracking struct {
      ID             uuid.UUID  `json:"id"`
      SubscriptionID uuid.UUID  `json:"subscription_id"`
      UserID         uuid.UUID  `json:"user_id"`

      // Period tracking
      PeriodStart    time.Time  `json:"period_start"`
      PeriodEnd      time.Time  `json:"period_end"`

      // Usage counts
      FeaturedUsed   int        `json:"featured_used"`
      PremiumUsed    int        `json:"premium_used"`

      CreatedAt      time.Time  `json:"created_at"`
      UpdatedAt      time.Time  `json:"updated_at"`
  }

  // Business logic methods
  func (u *UsageTracking) CanUseFeatured(limit int) bool
  func (u *UsageTracking) CanUsePremium(limit int) bool
  func (u *UsageTracking) IncrementFeatured() error
  func (u *UsageTracking) IncrementPremium() error
  func (u *UsageTracking) IsCurrentPeriod() bool
  ```

- [ ] Create `internal/modules/listing-promotion/domain/errors.go`
  ```go
  var (
      ErrPromotionNotFound         = errors.New("promotion not found")
      ErrSubscriptionNotFound      = errors.New("subscription not found")
      ErrDuplicateActivePromotion  = errors.New("listing already has active promotion")
      ErrInvalidPromotionType      = errors.New("invalid promotion type")
      ErrPromotionExpired          = errors.New("promotion has expired")
      ErrCannotCancelPromotion     = errors.New("cannot cancel promotion")
      ErrQuotaExceeded             = errors.New("subscription quota exceeded")
      ErrListingLimitReached       = errors.New("listing limit reached")
      ErrPhotoLimitReached         = errors.New("photo limit reached")
      ErrFeatureNotAvailable       = errors.New("feature not available in your plan")
      ErrInvalidBillingCycle       = errors.New("invalid billing cycle")
      ErrSubscriptionNotActive     = errors.New("subscription is not active")
  )
  ```

- [ ] Create `internal/modules/listing-promotion/domain/mapper.go`
  - MapListingPromotionFromSchema / MapListingPromotionToSchema
  - MapAgentSubscriptionFromSchema / MapAgentSubscriptionToSchema
  - MapUsageTrackingFromSchema / MapUsageTrackingToSchema
  - JSON marshaling for metadata and features

### 1.2 GORM Schemas

- [ ] Create `internal/modules/listing-promotion/repository/schema/gorm.go`
  ```go
  type ListingPromotion struct {
      ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
      ListingID   uuid.UUID  `gorm:"type:uuid;not null;index:idx_listing_promo"`
      OwnerID     uuid.UUID  `gorm:"type:uuid;not null;index"`

      Type        string     `gorm:"type:varchar(50);not null;index"`
      Status      string     `gorm:"type:varchar(20);not null;index:idx_promo_status"`

      Amount      int64      `gorm:"not null"`
      Currency    string     `gorm:"type:varchar(3);not null"`
      Duration    int        `gorm:"not null"`

      StartedAt   *time.Time `gorm:"index:idx_promo_active"`
      ExpiresAt   *time.Time `gorm:"index:idx_promo_active"`

      PaymentID      *uuid.UUID `gorm:"type:uuid;index"`
      SubscriptionID *uuid.UUID `gorm:"type:uuid;index"`
      IsIncluded     bool       `gorm:"not null;default:false"`

      BoostMultiplier float64 `gorm:"not null;default:1.0"`
      Metadata        string  `gorm:"type:jsonb"`

      CreatedAt   time.Time
      UpdatedAt   time.Time
      DeletedAt   *time.Time `gorm:"index"`
  }

  type AgentSubscription struct {
      ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
      UserID        uuid.UUID  `gorm:"type:uuid;not null;index:idx_user_subscription"`
      PlanType      string     `gorm:"type:varchar(50);not null"`
      Status        string     `gorm:"type:varchar(20);not null;index:idx_subscription_status"`

      BillingCycle  string     `gorm:"type:varchar(20);not null"`
      Amount        int64      `gorm:"not null"`
      Currency      string     `gorm:"type:varchar(3);not null"`
      NextBillingDate *time.Time `gorm:"index:idx_billing_due"`

      TrialEndsAt   *time.Time
      StartedAt     time.Time  `gorm:"not null"`
      CancelledAt   *time.Time
      ExpiredAt     *time.Time

      MaxListings            int    `gorm:"not null"`
      MaxPhotosPerListing    int    `gorm:"not null"`
      MaxVirtualTours        int    `gorm:"not null"`
      IncludedFeaturedPerMonth int  `gorm:"not null"`
      IncludedPremiumPerMonth  int  `gorm:"not null"`

      Features      string     `gorm:"type:jsonb"`

      CreatedAt     time.Time
      UpdatedAt     time.Time
      DeletedAt     *time.Time `gorm:"index"`
  }

  type UsageTracking struct {
      ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
      SubscriptionID uuid.UUID  `gorm:"type:uuid;not null;index:idx_usage_subscription"`
      UserID         uuid.UUID  `gorm:"type:uuid;not null;index"`

      PeriodStart    time.Time  `gorm:"not null;index:idx_usage_period"`
      PeriodEnd      time.Time  `gorm:"not null;index:idx_usage_period"`

      FeaturedUsed   int        `gorm:"not null;default:0"`
      PremiumUsed    int        `gorm:"not null;default:0"`

      CreatedAt      time.Time
      UpdatedAt      time.Time
  }
  ```

### 1.3 Repositories

- [ ] Create `internal/modules/listing-promotion/repository/interface.go`
  ```go
  type ListingPromotionRepository interface {
      Create(ctx context.Context, promo *schema.ListingPromotion) error
      GetByID(ctx context.Context, id uuid.UUID) (*schema.ListingPromotion, error)
      GetActiveByListing(ctx context.Context, listingID uuid.UUID) (*schema.ListingPromotion, error)
      ListByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*schema.ListingPromotion, error)
      ListExpiring(ctx context.Context, withinHours int) ([]*schema.ListingPromotion, error)
      UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
      Update(ctx context.Context, promo *schema.ListingPromotion) error
      ListActivePromotions(ctx context.Context, promoType string, limit int) ([]*schema.ListingPromotion, error)
  }

  type AgentSubscriptionRepository interface {
      Create(ctx context.Context, sub *schema.AgentSubscription) error
      GetByID(ctx context.Context, id uuid.UUID) (*schema.AgentSubscription, error)
      GetActiveByUser(ctx context.Context, userID uuid.UUID) (*schema.AgentSubscription, error)
      Update(ctx context.Context, sub *schema.AgentSubscription) error
      UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
      ListDueForBilling(ctx context.Context) ([]*schema.AgentSubscription, error)
      ListByUser(ctx context.Context, userID uuid.UUID) ([]*schema.AgentSubscription, error)
  }

  type UsageTrackingRepository interface {
      Create(ctx context.Context, usage *schema.UsageTracking) error
      GetCurrentPeriod(ctx context.Context, subscriptionID uuid.UUID) (*schema.UsageTracking, error)
      Update(ctx context.Context, usage *schema.UsageTracking) error
      CreateNewPeriod(ctx context.Context, subscriptionID, userID uuid.UUID, start, end time.Time) (*schema.UsageTracking, error)
  }
  ```

- [ ] Create `internal/modules/listing-promotion/repository/listing_promotion_repo.go`
  - Implement ListingPromotionRepository with GORM
  - Use transactions for status updates
  - Index on (listing_id, status) for fast active promotion lookup

- [ ] Create `internal/modules/listing-promotion/repository/agent_subscription_repo.go`
  - Implement AgentSubscriptionRepository with GORM
  - Unique constraint on (user_id, status) where status = 'active' (one active subscription per user)

- [ ] Create `internal/modules/listing-promotion/repository/usage_tracking_repo.go`
  - Implement UsageTrackingRepository with GORM
  - Atomic increment operations for usage counts

### 1.4 Database Migration

- [ ] Add to `cmd/api/main.go` AutoMigrate:
  ```go
  // Listing Promotion schemas
  &promotionSchema.ListingPromotion{},
  &promotionSchema.AgentSubscription{},
  &promotionSchema.UsageTracking{},
  ```

- [ ] SQL Migration (for reference):
  ```sql
  -- Listing promotions table
  CREATE TABLE listing_promotions (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      listing_id UUID NOT NULL,
      owner_id UUID NOT NULL,
      type VARCHAR(50) NOT NULL,
      status VARCHAR(20) NOT NULL,
      amount BIGINT NOT NULL,
      currency VARCHAR(3) NOT NULL,
      duration INT NOT NULL,
      started_at TIMESTAMP,
      expires_at TIMESTAMP,
      payment_id UUID,
      subscription_id UUID,
      is_included BOOLEAN NOT NULL DEFAULT FALSE,
      boost_multiplier DECIMAL(5,2) NOT NULL DEFAULT 1.0,
      metadata JSONB,
      created_at TIMESTAMP NOT NULL DEFAULT NOW(),
      updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
      deleted_at TIMESTAMP,
      CONSTRAINT unique_active_promo UNIQUE (listing_id, status) WHERE status = 'active'
  );

  CREATE INDEX idx_listing_promo ON listing_promotions(listing_id, deleted_at);
  CREATE INDEX idx_promo_owner ON listing_promotions(owner_id);
  CREATE INDEX idx_promo_status ON listing_promotions(status);
  CREATE INDEX idx_promo_active ON listing_promotions(started_at, expires_at) WHERE status = 'active';

  -- Agent subscriptions table
  CREATE TABLE agent_subscriptions (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      user_id UUID NOT NULL,
      plan_type VARCHAR(50) NOT NULL,
      status VARCHAR(20) NOT NULL,
      billing_cycle VARCHAR(20) NOT NULL,
      amount BIGINT NOT NULL,
      currency VARCHAR(3) NOT NULL,
      next_billing_date TIMESTAMP,
      trial_ends_at TIMESTAMP,
      started_at TIMESTAMP NOT NULL,
      cancelled_at TIMESTAMP,
      expired_at TIMESTAMP,
      max_listings INT NOT NULL,
      max_photos_per_listing INT NOT NULL,
      max_virtual_tours INT NOT NULL,
      included_featured_per_month INT NOT NULL,
      included_premium_per_month INT NOT NULL,
      features JSONB,
      created_at TIMESTAMP NOT NULL DEFAULT NOW(),
      updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
      deleted_at TIMESTAMP,
      CONSTRAINT unique_active_subscription UNIQUE (user_id, status) WHERE status = 'active'
  );

  CREATE INDEX idx_user_subscription ON agent_subscriptions(user_id, status);
  CREATE INDEX idx_subscription_status ON agent_subscriptions(status);
  CREATE INDEX idx_billing_due ON agent_subscriptions(next_billing_date) WHERE status = 'active';

  -- Usage tracking table
  CREATE TABLE usage_trackings (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      subscription_id UUID NOT NULL,
      user_id UUID NOT NULL,
      period_start TIMESTAMP NOT NULL,
      period_end TIMESTAMP NOT NULL,
      featured_used INT NOT NULL DEFAULT 0,
      premium_used INT NOT NULL DEFAULT 0,
      created_at TIMESTAMP NOT NULL DEFAULT NOW(),
      updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
      CONSTRAINT unique_usage_period UNIQUE (subscription_id, period_start)
  );

  CREATE INDEX idx_usage_subscription ON usage_trackings(subscription_id, period_start);
  CREATE INDEX idx_usage_period ON usage_trackings(period_start, period_end);
  ```

### Phase 1 Verification
- [ ] AutoMigrate wired to cmd/api/main.go
- [ ] All repositories implemented with GORM
- [ ] Domain models with business logic
- [ ] Mapper functions for domain<->schema conversion
- [ ] Code compiles successfully

**Completion Criteria:** Foundation layer ready for service implementation

---

## Phase 2: Core Services - Promotion & Subscription Management

**Goal:** Implement business logic for promotions, subscriptions, and usage tracking
**Estimated Files:** ~6 files, ~1200-1500 LOC
**Dependencies:** Phase 1 complete

### 2.1 Service Interfaces

- [ ] Create `internal/modules/listing-promotion/service/interface.go`
  ```go
  // PromotionService handles listing promotions
  type PromotionService interface {
      // Create promotion (validates payment, checks quota)
      CreatePromotion(ctx context.Context, listingID, ownerID uuid.UUID, promoType PromotionType, duration int, paymentID *uuid.UUID) (*domain.ListingPromotion, error)

      // Create promotion from subscription quota (no payment)
      CreateIncludedPromotion(ctx context.Context, listingID, ownerID uuid.UUID, promoType PromotionType, duration int) (*domain.ListingPromotion, error)

      // Start promotion (called after payment confirmation)
      StartPromotion(ctx context.Context, promoID uuid.UUID) error

      // Cancel promotion (before start or refund)
      CancelPromotion(ctx context.Context, promoID uuid.UUID) error

      // Get promotion details
      GetPromotion(ctx context.Context, promoID uuid.UUID) (*domain.ListingPromotion, error)

      // List user promotions
      ListUserPromotions(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*domain.ListingPromotion, error)

      // Check if listing has active promotion
      GetActivePromotion(ctx context.Context, listingID uuid.UUID) (*domain.ListingPromotion, error)

      // Expire promotions (called by cron)
      ExpirePromotions(ctx context.Context) error

      // Get featured/premium listings for display
      GetFeaturedListings(ctx context.Context, limit int) ([]*domain.ListingPromotion, error)
      GetPremiumListings(ctx context.Context, limit int) ([]*domain.ListingPromotion, error)
  }

  // SubscriptionService handles agent subscriptions
  type SubscriptionService interface {
      // Create subscription (starts trial or paid)
      CreateSubscription(ctx context.Context, userID uuid.UUID, planType PlanType, billingCycle BillingCycle, paymentID *uuid.UUID) (*domain.AgentSubscription, error)

      // Upgrade/downgrade subscription
      UpgradeSubscription(ctx context.Context, subscriptionID uuid.UUID, newPlan PlanType) error
      DowngradeSubscription(ctx context.Context, subscriptionID uuid.UUID, newPlan PlanType) error

      // Cancel subscription
      CancelSubscription(ctx context.Context, subscriptionID uuid.UUID) error

      // Renew subscription (called after payment)
      RenewSubscription(ctx context.Context, subscriptionID uuid.UUID, paymentID uuid.UUID) error

      // Get subscription details
      GetSubscription(ctx context.Context, subscriptionID uuid.UUID) (*domain.AgentSubscription, error)
      GetUserSubscription(ctx context.Context, userID uuid.UUID) (*domain.AgentSubscription, error)

      // Process billing (called by cron)
      ProcessBilling(ctx context.Context) error

      // Check limits
      CanAddListing(ctx context.Context, userID uuid.UUID) (bool, error)
      CanAddPhotos(ctx context.Context, listingID uuid.UUID, count int) (bool, error)
      CanUseFeature(ctx context.Context, userID uuid.UUID, feature string) (bool, error)

      // Check included promotion quota
      CanUseIncludedPromotion(ctx context.Context, userID uuid.UUID, promoType PromotionType) (bool, error)
      UseIncludedPromotion(ctx context.Context, userID uuid.UUID, promoType PromotionType) error
  }

  // UsageService tracks monthly quota usage
  type UsageService interface {
      GetCurrentUsage(ctx context.Context, subscriptionID uuid.UUID) (*domain.UsageTracking, error)
      IncrementUsage(ctx context.Context, subscriptionID uuid.UUID, usageType UsageType) error
      ResetUsage(ctx context.Context, subscriptionID uuid.UUID) error
  }
  ```

### 2.2 Promotion Service Implementation

- [ ] Create `internal/modules/listing-promotion/service/promotion_service.go`
  ```go
  type PromotionServiceImpl struct {
      promoRepo        repository.ListingPromotionRepository
      subscriptionRepo repository.AgentSubscriptionRepository
      usageService     UsageService
      config           *config.PromotionConfig // YAML config
      db               *gorm.DB
      log              *slog.Logger 
  }

  func NewPromotionService(...) *PromotionServiceImpl

  // Implementation highlights:
  // - CreatePromotion: Validates no active promotion exists, creates pending promotion
  // - CreateIncludedPromotion: Checks subscription quota via UsageService
  // - StartPromotion: Activates promotion, sets started_at and expires_at
  // - ExpirePromotions: Queries promotions where expires_at < now and status = active
  // - GetFeaturedListings: Returns active featured listings sorted by started_at
  ```

### 2.3 Subscription Service Implementation

- [ ] Create `internal/modules/listing-promotion/service/subscription_service.go`
  ```go
  type SubscriptionServiceImpl struct {
      subscriptionRepo repository.AgentSubscriptionRepository
      usageRepo        repository.UsageTrackingRepository
      config           *config.PromotionConfig
      db               *gorm.DB
      log              *slog.Logger 
  }

  func NewSubscriptionService(...) *SubscriptionServiceImpl

  // Implementation highlights:
  // - CreateSubscription: Loads plan config from YAML, creates subscription with limits cached
  // - UpgradeSubscription: Prorate current cycle, charge difference, update limits
  // - DowngradeSubscription: Schedule downgrade for next billing cycle
  // - CancelSubscription: Set cancelled_at, continue until next_billing_date
  // - ProcessBilling: Query subscriptions with next_billing_date <= now, create payment requests
  // - CanAddListing: Check current listing count vs subscription.MaxListings
  // - CanUseIncludedPromotion: Check usage_tracking.featured_used vs subscription.IncludedFeaturedPerMonth
  ```

### 2.4 Usage Service Implementation

- [ ] Create `internal/modules/listing-promotion/service/usage_service.go`
  ```go
  type UsageServiceImpl struct {
      usageRepo repository.UsageTrackingRepository
      log       *slog.Logger 
  }

  func NewUsageService(...) *UsageServiceImpl

  // Implementation highlights:
  // - GetCurrentUsage: Find usage record for current month, create if not exists
  // - IncrementUsage: Atomic increment of featured_used or premium_used
  // - ResetUsage: Called on 1st of month (from config), creates new period record
  ```

### 2.5 Helper Functions

- [ ] Create `internal/modules/listing-promotion/service/helpers.go`
  - `calculatePromotionPrice(promoType, duration)` - Lookup from YAML config
  - `calculateBoostMultiplier(promoType)` - Featured=10x, Premium=3x
  - `calculateProrationAmount(oldPlan, newPlan, daysRemaining)` - For upgrades
  - `loadPlanConfig(planType, billingCycle)` - Load from YAML
  - `validatePromotionDuration(promoType, duration)` - Check allowed durations
  - `getCurrentBillingPeriod()` - Calculate period_start and period_end

### Phase 2 Verification
- [ ] Create promotion with payment
- [ ] Verify promotion status = pending
- [ ] Start promotion, verify status = active and expires_at set
- [ ] Create subscription, verify limits cached from config
- [ ] Use included promotion, verify quota decremented
- [ ] Test listing limit enforcement
- [ ] Test photo limit enforcement

**Completion Criteria:** Can create promotions and subscriptions with proper business logic

---

## Phase 3: Integration Patterns - Property Module Interface

**Goal:** Wire promotion system into property module for limit enforcement
**Estimated Files:** ~4 files modified, ~200 LOC
**Dependencies:** Phase 2 complete

### 3.1 PromotionChecker Interface

- [ ] Create `internal/modules/property/port/hooks/promotion_hooks.go`
  ```go
  package hooks

  import (
      "context"
      "github.com/google/uuid"
  )

  // PromotionChecker provides subscription limit checking for property module
  type PromotionChecker interface {
      // Check if user can create a new listing
      CanCreateListing(ctx context.Context, userID uuid.UUID) (bool, error)

      // Check if user can add photos to listing
      CanAddPhotos(ctx context.Context, userID, listingID uuid.UUID, photoCount int) (bool, error)

      // Check if user has feature access
      HasFeature(ctx context.Context, userID uuid.UUID, feature string) (bool, error)

      // Get user's listing count
      GetListingCount(ctx context.Context, userID uuid.UUID) (int, error)

      // Get max allowed listings for user
      GetMaxListings(ctx context.Context, userID uuid.UUID) (int, error)
  }
  ```

### 3.2 Promotion Hooks Adapter

- [ ] Create `internal/modules/listing-promotion/port/hooks/property_hooks_adapter.go`
  ```go
  package hooks

  import (
      "context"
      "hauslet/internal/modules/listing-promotion/service"
      propertyHooks "hauslet/internal/modules/property/port/hooks"
      "github.com/google/uuid"
  )

  // PropertyPromotionAdapter implements property.PromotionChecker using promotion service
  type PropertyPromotionAdapter struct {
      subscriptionService service.SubscriptionService
      propertyService     PropertyListingCounter // Interface to count user's listings
  }

  func NewPropertyPromotionAdapter(
      subscriptionService service.SubscriptionService,
      propertyService PropertyListingCounter,
  ) propertyHooks.PromotionChecker {
      return &PropertyPromotionAdapter{
          subscriptionService: subscriptionService,
          propertyService:     propertyService,
      }
  }

  // PropertyListingCounter interface (implemented by property service)
  type PropertyListingCounter interface {
      CountUserListings(ctx context.Context, userID uuid.UUID) (int, error)
  }

  // Implementation
  func (a *PropertyPromotionAdapter) CanCreateListing(ctx context.Context, userID uuid.UUID) (bool, error) {
      // Get current listing count
      currentCount, err := a.propertyService.CountUserListings(ctx, userID)
      if err != nil {
          return false, err
      }

      // Get max allowed from subscription (or free tier)
      maxAllowed, err := a.GetMaxListings(ctx, userID)
      if err != nil {
          return false, err
      }

      return currentCount < maxAllowed, nil
  }

  func (a *PropertyPromotionAdapter) GetMaxListings(ctx context.Context, userID uuid.UUID) (int, error) {
      // Try to get active subscription
      sub, err := a.subscriptionService.GetUserSubscription(ctx, userID)
      if err != nil {
          // No subscription = free tier
          return 3, nil // From YAML config free_tier.max_listings
      }

      return sub.MaxListings, nil
  }

  func (a *PropertyPromotionAdapter) CanAddPhotos(ctx context.Context, userID, listingID uuid.UUID, photoCount int) (bool, error) {
      sub, err := a.subscriptionService.GetUserSubscription(ctx, userID)
      if err != nil {
          // Free tier: 20 photos max
          return photoCount <= 20, nil
      }

      return photoCount <= sub.MaxPhotosPerListing, nil
  }

  func (a *PropertyPromotionAdapter) HasFeature(ctx context.Context, userID uuid.UUID, feature string) (bool, error) {
      return a.subscriptionService.CanUseFeature(ctx, userID, feature)
  }
  ```

### 3.3 Update Property Service

- [ ] Modify `internal/modules/property/service/service.go`
  - Add `promotionChecker PromotionChecker` field (can be nil for backward compatibility)
  - Update `NewPropertyService` to accept promotion checker
  - **In CreateListing method:**
    ```go
    // Check listing limit
    if s.promotionChecker != nil {
        canCreate, err := s.promotionChecker.CanCreateListing(ctx, ownerID)
        if err != nil {
            return nil, err
        }
        if !canCreate {
            return nil, errors.New("listing limit reached - upgrade subscription")
        }
    }
    ```
  - **In AddPhotos method:**
    ```go
    // Check photo limit
    if s.promotionChecker != nil {
        canAdd, err := s.promotionChecker.CanAddPhotos(ctx, ownerID, listingID, len(photos))
        if err != nil {
            return err
        }
        if !canAdd {
            return errors.New("photo limit reached - upgrade subscription or purchase photo boost")
        }
    }
    ```
  - **In EnableVirtualTour method:**
    ```go
    // Check feature access
    if s.promotionChecker != nil {
        hasFeature, err := s.promotionChecker.HasFeature(ctx, ownerID, "virtual_tour_support")
        if err != nil {
            return err
        }
        if !hasFeature {
            return errors.New("virtual tours not available - upgrade subscription")
        }
    }
    ```

- [ ] Add method to property service:
  ```go
  func (s *PropertyServiceImpl) CountUserListings(ctx context.Context, userID uuid.UUID) (int, error) {
      return s.propertyRepo.CountByOwner(ctx, userID)
  }
  ```

### 3.4 Wire Services in Setup

- [ ] Modify `cmd/api/server/setup.go`
  ```go
  // Create promotion services
  promotionService := promotionService.NewPromotionService(...)
  subscriptionService := subscriptionService.NewSubscriptionService(...)

  // Create property service FIRST (without promotion checker)
  propertyService := propertyService.NewPropertyService(
      propertyRepo,
      nil, // promotion checker = nil initially
      log,
  )

  // Create adapter with property service
  promotionChecker := promotionHooks.NewPropertyPromotionAdapter(
      subscriptionService,
      propertyService, // Implements PropertyListingCounter
  )

  // Update property service with promotion checker
  propertyService.SetPromotionChecker(promotionChecker)
  ```

### Phase 3 Verification
- [ ] Create listing as free tier user
- [ ] Verify can create up to 3 listings
- [ ] Try to create 4th listing, verify error
- [ ] Create subscription (Basic plan)
- [ ] Verify can now create up to 10 listings
- [ ] Test photo limit enforcement
- [ ] Test feature access (virtual tours, analytics)

**Completion Criteria:** Property module enforces subscription limits via promotion system

---

## Phase 4: GraphQL API & Payment Integration

**Goal:** Expose promotion and subscription management via GraphQL
**Estimated Files:** ~5 files, ~800-1000 LOC
**Dependencies:** Phase 3 complete

### 4.1 GraphQL Schema

- [ ] Create `internal/modules/listing-promotion/port/graphql/schema.graphqls`
  ```graphql
  # ============================================================================
  # Enums
  # ============================================================================

  enum PromotionType {
      featured
      premium
      sponsored
  }

  enum PromotionStatus {
      pending
      active
      expired
      cancelled
      failed
  }

  enum PlanType {
      basic
      professional
      enterprise
  }

  enum SubscriptionStatus {
      trial
      active
      past_due
      cancelled
      expired
  }

  enum BillingCycle {
      monthly
      yearly
  }

  # ============================================================================
  # Types
  # ============================================================================

  type ListingPromotion {
      id: UUID!
      listingId: UUID!
      ownerId: UUID!
      type: PromotionType!
      status: PromotionStatus!
      amount: Int!
      currency: String!
      duration: Int!
      startedAt: Time
      expiresAt: Time
      paymentId: UUID
      subscriptionId: UUID
      isIncluded: Boolean!
      boostMultiplier: Float!
      createdAt: Time!
      updatedAt: Time!
  }

  type AgentSubscription {
      id: UUID!
      userId: UUID!
      planType: PlanType!
      status: SubscriptionStatus!
      billingCycle: BillingCycle!
      amount: Int!
      currency: String!
      nextBillingDate: Time
      trialEndsAt: Time
      startedAt: Time!
      cancelledAt: Time
      maxListings: Int!
      maxPhotosPerListing: Int!
      maxVirtualTours: Int!
      includedFeaturedPerMonth: Int!
      includedPremiumPerMonth: Int!
      features: [String!]!
      currentUsage: UsageTracking
      createdAt: Time!
      updatedAt: Time!
  }

  type UsageTracking {
      id: UUID!
      subscriptionId: UUID!
      periodStart: Time!
      periodEnd: Time!
      featuredUsed: Int!
      premiumUsed: Int!
      featuredRemaining: Int!
      premiumRemaining: Int!
  }

  type SubscriptionPlan {
      planType: PlanType!
      name: String!
      description: String!
      monthlyPrice: Int!
      yearlyPrice: Int!
      maxListings: Int!
      maxPhotosPerListing: Int!
      includedFeaturedPerMonth: Int!
      includedPremiumPerMonth: Int!
      features: [String!]!
  }

  type PromotionPricing {
      type: PromotionType!
      name: String!
      description: String!
      durations: [PromotionDuration!]!
      benefits: [String!]!
  }

  type PromotionDuration {
      days: Int!
      price: Int!
      pricePerDay: Int!
  }

  # ============================================================================
  # Inputs
  # ============================================================================

  input CreatePromotionInput {
      listingId: UUID!
      type: PromotionType!
      duration: Int! # in days
      paymentMethodId: String # Payment method to charge
      useIncludedQuota: Boolean! # Use subscription included quota
  }

  input CreateSubscriptionInput {
      planType: PlanType!
      billingCycle: BillingCycle!
      paymentMethodId: String! # Payment method for billing
  }

  input UpgradeSubscriptionInput {
      subscriptionId: UUID!
      newPlan: PlanType!
  }

  # ============================================================================
  # Queries
  # ============================================================================

  extend type Query {
      # Promotions
      listingPromotion(id: UUID!): ListingPromotion
      myPromotions(limit: Int, offset: Int): [ListingPromotion!]!
      activePromotionForListing(listingId: UUID!): ListingPromotion

      # Subscriptions
      mySubscription: AgentSubscription
      subscription(id: UUID!): AgentSubscription

      # Admin queries
      allSubscriptions(status: SubscriptionStatus, limit: Int, offset: Int): [AgentSubscription!]! @requireAdmin

      # Configuration/Pricing (public)
      availablePlans: [SubscriptionPlan!]!
      promotionPricing: [PromotionPricing!]!

      # Usage
      myCurrentUsage: UsageTracking
  }

  # ============================================================================
  # Mutations
  # ============================================================================

  extend type Mutation {
      # Promotions
      createPromotion(input: CreatePromotionInput!): ListingPromotion!
      startPromotion(promotionId: UUID!): ListingPromotion!
      cancelPromotion(promotionId: UUID!): ListingPromotion!

      # Subscriptions
      createSubscription(input: CreateSubscriptionInput!): AgentSubscription!
      upgradeSubscription(input: UpgradeSubscriptionInput!): AgentSubscription!
      downgradeSubscription(subscriptionId: UUID!, newPlan: PlanType!): AgentSubscription!
      cancelSubscription(subscriptionId: UUID!): AgentSubscription!

      # Admin
      renewSubscription(subscriptionId: UUID!, paymentId: UUID!): AgentSubscription! @requireAdmin
  }
  ```

### 4.2 GraphQL Resolvers

- [ ] Create `internal/modules/listing-promotion/port/graphql/resolvers.go`
  ```go
  type Resolver struct {
      promotionService    service.PromotionService
      subscriptionService service.SubscriptionService
      usageService        service.UsageService
      config              *config.PromotionConfig
      log                 *slog.Logger 
  }

  func NewResolver(
      promotionService service.PromotionService,
      subscriptionService service.SubscriptionService,
      usageService service.UsageService,
      config *config.PromotionConfig,
      log *slog.Logger ,
  ) *Resolver

  // Query resolvers
  func (r *Resolver) ListingPromotion(ctx context.Context, id uuid.UUID) (*domain.ListingPromotion, error)
  func (r *Resolver) MyPromotions(ctx context.Context, limit, offset *int) ([]*domain.ListingPromotion, error)
  func (r *Resolver) MySubscription(ctx context.Context) (*domain.AgentSubscription, error)
  func (r *Resolver) AvailablePlans(ctx context.Context) ([]*SubscriptionPlan, error) // Load from YAML
  func (r *Resolver) PromotionPricing(ctx context.Context) ([]*PromotionPricing, error) // Load from YAML

  // Mutation resolvers
  func (r *Resolver) CreatePromotion(ctx context.Context, input CreatePromotionInput) (*domain.ListingPromotion, error)
  func (r *Resolver) CreateSubscription(ctx context.Context, input CreateSubscriptionInput) (*domain.AgentSubscription, error)
  func (r *Resolver) UpgradeSubscription(ctx context.Context, input UpgradeSubscriptionInput) (*domain.AgentSubscription, error)
  func (r *Resolver) CancelSubscription(ctx context.Context, subscriptionID uuid.UUID) (*domain.AgentSubscription, error)

  // Field resolvers
  func (r *agentSubscriptionResolver) CurrentUsage(ctx context.Context, obj *domain.AgentSubscription) (*domain.UsageTracking, error)
  func (r *usageTrackingResolver) FeaturedRemaining(ctx context.Context, obj *domain.UsageTracking) (int, error)
  func (r *usageTrackingResolver) PremiumRemaining(ctx context.Context, obj *domain.UsageTracking) (int, error)
  ```

### 4.3 Payment Integration

- [ ] Update `internal/modules/payments/port/http/webhook_handler.go`
  - Add `promotionHooks PromotionHooks` field
  - **In handleChargeSuccess:**
    ```go
    // Check if payment is for a promotion
    if pmt.Metadata != nil {
        if promoID, ok := pmt.Metadata["promotion_id"]; ok {
            promotionID, _ := uuid.Parse(promoID.(string))
            if err := h.promotionHooks.OnPromotionPaymentSucceeded(ctx, promotionID, pmt.ID); err != nil {
                h.log.Error("failed to start promotion: %v", err)
            }
        }

        if subID, ok := pmt.Metadata["subscription_id"]; ok {
            subscriptionID, _ := uuid.Parse(subID.(string))
            if err := h.promotionHooks.OnSubscriptionPaymentSucceeded(ctx, subscriptionID, pmt.ID); err != nil {
                h.log.Error("failed to activate subscription: %v", err)
            }
        }
    }
    ```

- [ ] Create `internal/modules/listing-promotion/port/hooks/payment_hooks_adapter.go`
  ```go
  package hooks

  import (
      "context"
      "hauslet/internal/modules/listing-promotion/service"
      "github.com/google/uuid"
  )

  type PromotionPaymentHooks interface {
      OnPromotionPaymentSucceeded(ctx context.Context, promotionID, paymentID uuid.UUID) error
      OnPromotionPaymentFailed(ctx context.Context, promotionID uuid.UUID) error
      OnSubscriptionPaymentSucceeded(ctx context.Context, subscriptionID, paymentID uuid.UUID) error
      OnSubscriptionPaymentFailed(ctx context.Context, subscriptionID uuid.UUID) error
  }

  type PaymentHooksAdapter struct {
      promotionService    service.PromotionService
      subscriptionService service.SubscriptionService
      log                 *slog.Logger 
  }

  func (a *PaymentHooksAdapter) OnPromotionPaymentSucceeded(ctx context.Context, promotionID, paymentID uuid.UUID) error {
      // Start the promotion
      return a.promotionService.StartPromotion(ctx, promotionID)
  }

  func (a *PaymentHooksAdapter) OnSubscriptionPaymentSucceeded(ctx context.Context, subscriptionID, paymentID uuid.UUID) error {
      // Renew subscription
      return a.subscriptionService.RenewSubscription(ctx, subscriptionID, paymentID)
  }

  func (a *PaymentHooksAdapter) OnSubscriptionPaymentFailed(ctx context.Context, subscriptionID uuid.UUID) error {
      // TODO: Mark subscription as past_due, send notification
      return nil
  }
  ```

### 4.4 Register GraphQL Types

- [ ] Update `gqlgen.yml`:
  ```yaml
  # Promotion types
  ListingPromotion:
    model: hauslet/internal/modules/listing-promotion/domain.ListingPromotion
  AgentSubscription:
    model: hauslet/internal/modules/listing-promotion/domain.AgentSubscription
  UsageTracking:
    model: hauslet/internal/modules/listing-promotion/domain.UsageTracking

  # Enums
  PromotionType:
    model: hauslet/internal/modules/listing-promotion/domain.PromotionType
  PromotionStatus:
    model: hauslet/internal/modules/listing-promotion/domain.PromotionStatus
  PlanType:
    model: hauslet/internal/modules/listing-promotion/domain.PlanType
  SubscriptionStatus:
    model: hauslet/internal/modules/listing-promotion/domain.SubscriptionStatus
  BillingCycle:
    model: hauslet/internal/modules/listing-promotion/domain.BillingCycle
  ```

- [ ] Regenerate GraphQL code: `make gql-gen`

- [ ] Update `internal/transport/graph/schema.resolvers.go` to delegate to promotion resolver

### Phase 4 Verification
- [ ] Query available plans via GraphQL
- [ ] Create subscription with payment
- [ ] Verify subscription activated after payment webhook
- [ ] Create promotion with payment
- [ ] Verify promotion started after payment
- [ ] Create promotion using included quota
- [ ] Verify usage tracking updated

**Completion Criteria:** Full GraphQL API with payment integration

---

## Phase 5: Worker Jobs - Billing & Expiry Automation

**Goal:** Implement automated billing and promotion expiry
**Estimated Files:** ~4 files, ~400-500 LOC
**Dependencies:** Phase 4 complete

### 5.1 Promotion Expiry Job

- [ ] Create `internal/queue/jobs/promotion/expire_promotions.go`
  ```go
  type ExpirePromotionsJob struct {
      promotionService promotion.PromotionService
      log              *slog.Logger 
  }

  func NewExpirePromotionsJob(
      promotionService promotion.PromotionService,
      log *slog.Logger ,
  ) *ExpirePromotionsJob

  func (j *ExpirePromotionsJob) Run(ctx context.Context) error {
      j.log.Info(" starting promotion expiry check")

      if err := j.promotionService.ExpirePromotions(ctx); err != nil {
          j.log.Error("failed to expire promotions: %v", err)
          return err
      }

      j.log.Info(" promotion expiry check complete")
      return nil
  }
  ```

- [ ] Register in `cmd/worker/setup/handlers.go`:
  ```go
  // Expire promotions job - every hour
  expirePromotionsJob := promotionJobs.NewExpirePromotionsJob(promotionService, log)
  scheduler.AddFunc("0 * * * *", func() {
      expirePromotionsJob.Run(context.Background())
  })
  ```

### 5.2 Subscription Billing Job

- [ ] Create `internal/queue/jobs/promotion/process_billing.go`
  ```go
  type ProcessBillingJob struct {
      subscriptionService promotion.SubscriptionService
      paymentService      payment.PaymentService // To create charges
      log                 *slog.Logger 
  }

  func NewProcessBillingJob(
      subscriptionService promotion.SubscriptionService,
      paymentService payment.PaymentService,
      log *slog.Logger ,
  ) *ProcessBillingJob

  func (j *ProcessBillingJob) Run(ctx context.Context) error {
      j.log.Info(" starting subscription billing")

      // Get subscriptions due for billing
      if err := j.subscriptionService.ProcessBilling(ctx); err != nil {
          j.log.Error("failed to process billing: %v", err)
          return err
      }

      j.log.Info(" subscription billing complete")
      return nil
  }
  ```

- [ ] Update `SubscriptionService.ProcessBilling`:
  ```go
  func (s *SubscriptionServiceImpl) ProcessBilling(ctx context.Context) error {
      // Get subscriptions due for billing
      subs, err := s.subscriptionRepo.ListDueForBilling(ctx)
      if err != nil {
          return err
      }

      for _, sub := range subs {
          // Create payment charge via payment service
          payment, err := s.createBillingCharge(ctx, sub)
          if err != nil {
              s.log.Error("failed to charge subscription %s: %v", sub.ID, err)
              // Mark as past_due
              s.subscriptionRepo.UpdateStatus(ctx, sub.ID, string(SubscriptionStatusPastDue))
              continue
          }

          // Update next billing date
          nextBilling := calculateNextBillingDate(sub.NextBillingDate, sub.BillingCycle)
          sub.NextBillingDate = &nextBilling
          s.subscriptionRepo.Update(ctx, mappers.MapSubscriptionToSchema(sub))

          s.log.Info(" successfully charged subscription %s, payment %s", sub.ID, payment.ID)
      }

      return nil
  }
  ```

- [ ] Register in `cmd/worker/setup/handlers.go`:
  ```go
  // Process subscription billing job - daily at 9 AM
  processBillingJob := promotionJobs.NewProcessBillingJob(subscriptionService, paymentService, log)
  scheduler.AddFunc("0 9 * * *", func() {
      processBillingJob.Run(context.Background())
  })
  ```

### 5.3 Usage Reset Job

- [ ] Create `internal/queue/jobs/promotion/reset_usage.go`
  ```go
  type ResetUsageJob struct {
      usageService usage.UsageService
      config       *config.PromotionConfig
      log          *slog.Logger 
  }

  func NewResetUsageJob(
      usageService usage.UsageService,
      config *config.PromotionConfig,
      log *slog.Logger ,
  ) *ResetUsageJob

  func (j *ResetUsageJob) Run(ctx context.Context) error {
      j.log.Info(" starting monthly usage reset")

      // Reset usage for all active subscriptions
      // This creates new usage tracking records for the new month

      j.log.Info(" monthly usage reset complete")
      return nil
  }
  ```

- [ ] Register in `cmd/worker/setup/handlers.go`:
  ```go
  // Reset monthly usage - 1st of month at midnight
  resetUsageJob := promotionJobs.NewResetUsageJob(usageService, config, log)
  scheduler.AddFunc("0 0 1 * *", func() {
      resetUsageJob.Run(context.Background())
  })
  ```

### 5.4 Cloud Scheduler Configuration

- [ ] Update `deploy/terraform/cloudscheduler.tf`:
  ```hcl
  # Expire promotions job
  resource "google_cloud_scheduler_job" "expire_promotions" {
    name        = "expire-promotions-hourly"
    description = "Expire listing promotions that have reached their end date"
    schedule    = "0 * * * *"
    time_zone   = "UTC"

    http_target {
      uri         = "${var.cloud_tasks_url}/jobs/promotion/expire"
      http_method = "POST"

      oidc_token {
        service_account_email = var.service_account_email
      }
    }
  }

  # Process subscription billing
  resource "google_cloud_scheduler_job" "process_billing" {
    name        = "process-subscription-billing-daily"
    description = "Process subscription billing for due subscriptions"
    schedule    = "0 9 * * *"
    time_zone   = "UTC"

    http_target {
      uri         = "${var.cloud_tasks_url}/jobs/promotion/billing"
      http_method = "POST"

      oidc_token {
        service_account_email = var.service_account_email
      }
    }
  }

  # Reset monthly usage
  resource "google_cloud_scheduler_job" "reset_usage" {
    name        = "reset-monthly-usage"
    description = "Reset monthly promotion usage tracking"
    schedule    = "0 0 1 * *"
    time_zone   = "UTC"

    http_target {
      uri         = "${var.cloud_tasks_url}/jobs/promotion/reset-usage"
      http_method = "POST"

      oidc_token {
        service_account_email = var.service_account_email
      }
    }
  }
  ```

### Phase 5 Verification
- [ ] Run expiry job manually, verify expired promotions marked
- [ ] Create subscription with billing due tomorrow
- [ ] Run billing job, verify charge created
- [ ] Verify payment webhook activates subscription
- [ ] Run usage reset job, verify new period created
- [ ] Check Cloud Scheduler jobs registered

**Completion Criteria:** Automated billing and promotion lifecycle management

---

## Phase 6: Analytics & Reporting (Optional - Future)

**Goal:** Track promotion performance and subscription metrics
**Estimated Files:** ~6 files, ~500-700 LOC
**Dependencies:** Phase 5 complete

### 6.1 Analytics Domain Models

- [ ] Create `internal/modules/listing-promotion/domain/analytics.go`
  ```go
  type PromotionAnalytics struct {
      PromotionID    uuid.UUID
      Impressions    int64
      Clicks         int64
      Leads          int64
      Conversions    int64
      CTR            float64  // Click-through rate
      ConversionRate float64
      ROI            float64  // Return on investment
      Date           time.Time
  }

  type SubscriptionMetrics struct {
      TotalSubscribers    int
      ActiveSubscribers   int
      TrialSubscribers    int
      ChurnRate           float64
      MRR                 int64 // Monthly recurring revenue
      ARR                 int64 // Annual recurring revenue
      AverageRevenue      int64
      Date                time.Time
  }
  ```

### 6.2 Analytics Service

- [ ] Create `internal/modules/listing-promotion/service/analytics_service.go`
  - Track promotion impressions, clicks, leads
  - Calculate ROI for promotions
  - Calculate subscription metrics (MRR, ARR, churn)
  - Generate performance reports

### 6.3 GraphQL Analytics Queries

- [ ] Extend GraphQL schema:
  ```graphql
  extend type Query {
      promotionAnalytics(promotionId: UUID!): PromotionAnalytics!
      myPromotionPerformance(limit: Int): [PromotionAnalytics!]!

      # Admin only
      subscriptionMetrics(startDate: Time!, endDate: Time!): SubscriptionMetrics! @requireAdmin
      topPerformingPromotions(limit: Int): [PromotionAnalytics!]! @requireAdmin
  }
  ```

**Note:** This phase can be deferred until core system is stable and in production

---

## 🔧 Integration Checklist

- [ ] Promotion services initialized in `cmd/api/main.go`
- [ ] Promotion services initialized in `cmd/worker/main.go`
- [ ] Property service updated with promotion checker
- [ ] Webhook handler updated with promotion hooks
- [ ] GraphQL schema includes promotion types
- [ ] Worker cron jobs registered
- [ ] Cloud Scheduler configured
- [ ] Migration applied to database
- [ ] YAML config loaded in service initialization

---

## 📝 Testing Strategy

### Unit Tests
- [ ] Domain model validation (promotion, subscription)
- [ ] Repository CRUD operations
- [ ] Service business logic
- [ ] Limit enforcement (listings, photos, features)
- [ ] Quota tracking (included promotions)
- [ ] Billing calculations

### Integration Tests
- [ ] End-to-end promotion creation → payment → activation
- [ ] Subscription creation → billing → renewal
- [ ] Limit enforcement in property module
- [ ] Usage tracking across billing cycles
- [ ] Webhook event handling

### Manual Testing
- [ ] Create promotion with payment
- [ ] Verify search boost applied
- [ ] Create subscription
- [ ] Use included promotion quota
- [ ] Hit listing limit, verify error
- [ ] Upgrade subscription, verify new limits
- [ ] Cancel subscription, verify grace period
- [ ] Test expiry automation
- [ ] Test billing automation

---

## 🚨 Critical Success Factors

1. **Configuration Accuracy**: YAML config must match business pricing strategy
2. **Limit Enforcement**: Property module must strictly enforce subscription limits
3. **Quota Tracking**: Monthly usage must reset correctly on billing cycle
4. **Payment Integration**: Promotions/subscriptions must activate only after payment confirmation
5. **Billing Automation**: Subscription renewals must charge correctly with retry logic
6. **Search Ranking**: Promotion boost multipliers must affect listing visibility
7. **User Experience**: Clear error messages when limits reached

---

## 📊 Progress Tracking

**Current Phase:** Phase 0 - Planning & Configuration
**Completion:** 0/6 phases
**Configuration:** ✅ YAML created

**Phase Summary:**
- ✅ Phase 0: Configuration (YAML) - 100%
- ⏸️  Phase 1: Foundation (Domain & Repository) - 0%
- ⏸️  Phase 2: Core Services - 0%
- ⏸️  Phase 3: Integration Patterns - 0%
- ⏸️  Phase 4: GraphQL API - 0%
- ⏸️  Phase 5: Worker Jobs - 0%
- ⏸️  Phase 6: Analytics (Optional) - 0%

**Next Steps:**
1. Create domain models and enums
2. Implement GORM schemas
3. Build repositories
4. Add AutoMigrate
5. Verify code compiles

---

## 📌 Architecture Decisions

- **Free Tier:** 3 listings, 20 photos, basic features (no subscription required)
- **Promotion Types:** Featured (top placement), Premium (enhanced features), Sponsored (future CPC)
- **Subscription Tiers:** Basic (₦30k/month), Professional (₦100k/month), Enterprise (₦250k/month)
- **Billing Cycle:** Monthly or yearly (17% discount on yearly)
- **Trial Period:** 14 days free trial for all subscriptions
- **Grace Period:** 3 days after payment failure before suspension
- **Usage Reset:** Monthly on 1st of month at midnight UTC
- **Promotion Limits:** 1 active promotion per listing
- **Integration Pattern:** Adapter pattern for property module decoupling
- **Search Boost:** Featured = 10x, Premium = 3x ranking multiplier
- **Revenue Model:** Shortlets = commission, Sales/Rentals = subscriptions + promotions

---

**Last Updated:** 2025-12-30
**Estimated Total Effort:** 8-10 days for Phases 1-5
**Risk Level:** Medium (payment integration, limit enforcement, billing automation)
**Business Impact:** HIGH - Primary monetization for Sales/Rentals listings
