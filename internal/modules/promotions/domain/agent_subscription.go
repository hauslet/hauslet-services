package domain

import (
	"time"

	"github.com/google/uuid"
)

// AgentSubscription represents a user's subscription plan
type AgentSubscription struct {
	ID        uuid.UUID          `json:"id"`
	UserID    uuid.UUID          `json:"user_id"`
	UserEmail string             `json:"user_email"`
	UserName  string             `json:"user_name"`
	PlanType  PlanType           `json:"plan_type"`
	Status    SubscriptionStatus `json:"status"`

	// Billing
	BillingCycle    BillingCycle `json:"billing_cycle"`
	Amount          int64        `json:"amount"`   // In minor units (kobo)
	Currency        string       `json:"currency"` // NGN
	NextBillingDate *time.Time   `json:"next_billing_date,omitempty"`

	// Trial
	TrialEndsAt *time.Time `json:"trial_ends_at,omitempty"`

	// Lifecycle
	StartedAt   time.Time  `json:"started_at"`
	CancelledAt *time.Time `json:"cancelled_at,omitempty"`
	ExpiredAt   *time.Time `json:"expired_at,omitempty"`

	// Limits (cached from config for this subscription)
	MaxListings         int `json:"max_listings"`          // -1 = unlimited
	MaxPhotosPerListing int `json:"max_photos_per_listing"`
	MaxVirtualTours     int `json:"max_virtual_tours"` // -1 = unlimited

	// Included Promotions Per Month
	IncludedFeaturedPerMonth int `json:"included_featured_per_month"`
	IncludedPremiumPerMonth  int `json:"included_premium_per_month"` // -1 = unlimited

	// Viewing Events Quotas (NEW)
	IncludedOpenHousesPerMonth      int `json:"included_open_houses_per_month"`       // -1 = unlimited
	IncludedPrivateShowingsPerMonth int `json:"included_private_showings_per_month"` // -1 = unlimited

	// Features (cached from config)
	Features map[string]bool `json:"features"`

	// Pending Plan Changes (Option A: Schedule changes for next billing cycle)
	PendingPlanType       *PlanType  `json:"pending_plan_type,omitempty"`
	PendingPlanScheduledAt *time.Time `json:"pending_plan_scheduled_at,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// IsActive returns true if the subscription is active
func (s *AgentSubscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive || s.Status == SubscriptionStatusTrial
}

// IsInTrial returns true if the subscription is in trial period
func (s *AgentSubscription) IsInTrial() bool {
	if s.Status != SubscriptionStatusTrial {
		return false
	}
	if s.TrialEndsAt == nil {
		return false
	}
	return time.Now().Before(*s.TrialEndsAt)
}

// CanUpgrade returns true if the subscription can be upgraded
func (s *AgentSubscription) CanUpgrade() bool {
	return s.IsActive() && s.PlanType != PlanTypeEnterprise
}

// CanDowngrade returns true if the subscription can be downgraded
func (s *AgentSubscription) CanDowngrade() bool {
	return s.IsActive() && s.PlanType != PlanTypeFree
}

// Cancel marks the subscription as cancelled
func (s *AgentSubscription) Cancel() error {
	if !s.IsActive() {
		return ErrSubscriptionNotActive
	}

	now := time.Now()
	s.Status = SubscriptionStatusCancelled
	s.CancelledAt = &now
	s.UpdatedAt = now

	return nil
}

// Renew updates the subscription for the next billing cycle
func (s *AgentSubscription) Renew(nextBillingDate time.Time) error {
	if s.Status == SubscriptionStatusTrial {
		// Trial ending, convert to active
		s.Status = SubscriptionStatusActive
	} else if s.Status != SubscriptionStatusActive && s.Status != SubscriptionStatusPastDue {
		return ErrSubscriptionNotActive
	}

	s.NextBillingDate = &nextBillingDate
	s.UpdatedAt = time.Now()

	return nil
}

// GetFeature returns whether a feature is enabled for this subscription
func (s *AgentSubscription) GetFeature(name string) bool {
	if s.Features == nil {
		return false
	}
	enabled, exists := s.Features[name]
	return exists && enabled
}

// HasUnlimitedListings returns true if the subscription has unlimited listings
func (s *AgentSubscription) HasUnlimitedListings() bool {
	return s.MaxListings == -1
}

// HasUnlimitedPremium returns true if the subscription has unlimited premium promotions
func (s *AgentSubscription) HasUnlimitedPremium() bool {
	return s.IncludedPremiumPerMonth == -1
}

// HasUnlimitedOpenHouses returns true if the subscription has unlimited open houses
func (s *AgentSubscription) HasUnlimitedOpenHouses() bool {
	return s.IncludedOpenHousesPerMonth == -1
}

// HasUnlimitedPrivateShowings returns true if the subscription has unlimited private showings
func (s *AgentSubscription) HasUnlimitedPrivateShowings() bool {
	return s.IncludedPrivateShowingsPerMonth == -1
}

// MarkPastDue marks the subscription as past due after payment failure
func (s *AgentSubscription) MarkPastDue() error {
	if s.Status != SubscriptionStatusActive {
		return ErrSubscriptionNotActive
	}

	s.Status = SubscriptionStatusPastDue
	s.UpdatedAt = time.Now()

	return nil
}

// Expire marks the subscription as expired after grace period
func (s *AgentSubscription) Expire() error {
	now := time.Now()
	s.Status = SubscriptionStatusExpired
	s.ExpiredAt = &now
	s.UpdatedAt = now

	return nil
}

// SchedulePlanChange schedules a plan change for the next billing cycle
func (s *AgentSubscription) SchedulePlanChange(newPlan PlanType) error {
	if !s.IsActive() {
		return ErrSubscriptionNotActive
	}

	if newPlan == s.PlanType {
		return ErrInvalidPlanType
	}

	// Validate upgrade/downgrade is possible
	if newPlan > s.PlanType && !s.CanUpgrade() {
		return ErrCannotUpgrade
	}

	if newPlan < s.PlanType && !s.CanDowngrade() {
		return ErrCannotDowngrade
	}

	now := time.Now()
	s.PendingPlanType = &newPlan
	s.PendingPlanScheduledAt = &now
	s.UpdatedAt = now

	return nil
}

// CancelPendingPlanChange cancels a scheduled plan change
func (s *AgentSubscription) CancelPendingPlanChange() {
	s.PendingPlanType = nil
	s.PendingPlanScheduledAt = nil
	s.UpdatedAt = time.Now()
}

// HasPendingPlanChange returns true if there's a pending plan change
func (s *AgentSubscription) HasPendingPlanChange() bool {
	return s.PendingPlanType != nil
}

// ApplyPendingPlanChange applies the pending plan change and updates limits
func (s *AgentSubscription) ApplyPendingPlanChange(newLimits PlanLimits, newAmount int64, newCurrency string) error {
	if !s.HasPendingPlanChange() {
		return nil // Nothing to apply
	}

	// Update plan type
	s.PlanType = *s.PendingPlanType

	// Update billing amount
	s.Amount = newAmount
	s.Currency = newCurrency

	// Update limits
	s.MaxListings = newLimits.MaxListings
	s.MaxPhotosPerListing = newLimits.MaxPhotosPerListing
	s.MaxVirtualTours = newLimits.MaxVirtualTours
	s.IncludedFeaturedPerMonth = newLimits.IncludedFeaturedPerMonth
	s.IncludedPremiumPerMonth = newLimits.IncludedPremiumPerMonth
	s.IncludedOpenHousesPerMonth = newLimits.IncludedOpenHousesPerMonth
	s.IncludedPrivateShowingsPerMonth = newLimits.IncludedPrivateShowingsPerMonth
	s.Features = newLimits.Features

	// Clear pending change
	s.PendingPlanType = nil
	s.PendingPlanScheduledAt = nil
	s.UpdatedAt = time.Now()

	return nil
}

// ApplyImmediateUpgrade applies an upgrade immediately (without pending state)
// Used for instant upgrades with proration
func (s *AgentSubscription) ApplyImmediateUpgrade(newPlan PlanType, newLimits PlanLimits, newAmount int64, newCurrency string) error {
	if !s.IsActive() {
		return ErrSubscriptionNotActive
	}

	// Validate it's actually an upgrade
	if newPlan <= s.PlanType {
		return ErrCannotUpgrade
	}

	if !s.CanUpgrade() {
		return ErrCannotUpgrade
	}

	// Clear any pending downgrades (upgrade takes precedence)
	if s.HasPendingPlanChange() && *s.PendingPlanType < s.PlanType {
		s.CancelPendingPlanChange()
	}

	// Update plan type immediately
	s.PlanType = newPlan

	// Update billing amount
	s.Amount = newAmount
	s.Currency = newCurrency

	// Update limits immediately
	s.MaxListings = newLimits.MaxListings
	s.MaxPhotosPerListing = newLimits.MaxPhotosPerListing
	s.MaxVirtualTours = newLimits.MaxVirtualTours
	s.IncludedFeaturedPerMonth = newLimits.IncludedFeaturedPerMonth
	s.IncludedPremiumPerMonth = newLimits.IncludedPremiumPerMonth
	s.IncludedOpenHousesPerMonth = newLimits.IncludedOpenHousesPerMonth
	s.IncludedPrivateShowingsPerMonth = newLimits.IncludedPrivateShowingsPerMonth
	s.Features = newLimits.Features

	s.UpdatedAt = time.Now()

	return nil
}

// PlanLimits holds the configuration for a subscription plan
type PlanLimits struct {
	MaxListings                     int
	MaxPhotosPerListing             int
	MaxVirtualTours                 int
	IncludedFeaturedPerMonth        int
	IncludedPremiumPerMonth         int
	IncludedOpenHousesPerMonth      int
	IncludedPrivateShowingsPerMonth int
	Features                        map[string]bool
}
