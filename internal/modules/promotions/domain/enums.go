package domain

// PromotionType represents the type of listing promotion
type PromotionType string

const (
	PromotionTypeFeatured  PromotionType = "featured"
	PromotionTypePremium   PromotionType = "premium"
	PromotionTypeSponsored PromotionType = "sponsored" // Future feature
)

// String returns the string representation of PromotionType
func (p PromotionType) String() string {
	return string(p)
}

// IsValid checks if the promotion type is valid
func (p PromotionType) IsValid() bool {
	switch p {
	case PromotionTypeFeatured, PromotionTypePremium, PromotionTypeSponsored:
		return true
	}
	return false
}

// PromotionStatus represents the current status of a promotion
type PromotionStatus string

const (
	PromotionStatusPending   PromotionStatus = "pending"   // Payment pending
	PromotionStatusActive    PromotionStatus = "active"    // Currently running
	PromotionStatusExpired   PromotionStatus = "expired"   // Time elapsed
	PromotionStatusCancelled PromotionStatus = "cancelled" // User cancelled
	PromotionStatusFailed    PromotionStatus = "failed"    // Payment failed
)

// String returns the string representation of PromotionStatus
func (p PromotionStatus) String() string {
	return string(p)
}

// IsValid checks if the promotion status is valid
func (p PromotionStatus) IsValid() bool {
	switch p {
	case PromotionStatusPending, PromotionStatusActive, PromotionStatusExpired,
		PromotionStatusCancelled, PromotionStatusFailed:
		return true
	}
	return false
}

// PlanType represents the subscription plan tier
type PlanType string

const (
	PlanTypeFree         PlanType = "free"
	PlanTypeBasic        PlanType = "basic"
	PlanTypeProfessional PlanType = "professional"
	PlanTypeEnterprise   PlanType = "enterprise"
)

// String returns the string representation of PlanType
func (p PlanType) String() string {
	return string(p)
}

// IsValid checks if the plan type is valid
func (p PlanType) IsValid() bool {
	switch p {
	case PlanTypeFree, PlanTypeBasic, PlanTypeProfessional, PlanTypeEnterprise:
		return true
	}
	return false
}

// SubscriptionStatus represents the current status of a subscription
type SubscriptionStatus string

const (
	SubscriptionStatusPending   SubscriptionStatus = "pending"   // Awaiting initial payment
	SubscriptionStatusTrial     SubscriptionStatus = "trial"     // Free trial
	SubscriptionStatusActive    SubscriptionStatus = "active"    // Paid and active
	SubscriptionStatusPastDue   SubscriptionStatus = "past_due"  // Payment failed, in grace period
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled" // Cancelled
	SubscriptionStatusExpired   SubscriptionStatus = "expired"   // Expired after grace period
)

// String returns the string representation of SubscriptionStatus
func (s SubscriptionStatus) String() string {
	return string(s)
}

// IsValid checks if the subscription status is valid
func (s SubscriptionStatus) IsValid() bool {
	switch s {
	case SubscriptionStatusPending, SubscriptionStatusTrial, SubscriptionStatusActive,
		SubscriptionStatusPastDue, SubscriptionStatusCancelled, SubscriptionStatusExpired:
		return true
	}
	return false
}

// BillingCycle represents the billing frequency
type BillingCycle string

const (
	BillingCycleMonthly BillingCycle = "monthly"
	BillingCycleYearly  BillingCycle = "yearly"
)

// String returns the string representation of BillingCycle
func (b BillingCycle) String() string {
	return string(b)
}

// IsValid checks if the billing cycle is valid
func (b BillingCycle) IsValid() bool {
	switch b {
	case BillingCycleMonthly, BillingCycleYearly:
		return true
	}
	return false
}

// UsageType represents the type of usage being tracked
type UsageType string

const (
	UsageTypeFeatured       UsageType = "featured"
	UsageTypePremium        UsageType = "premium"
	UsageTypeOpenHouse      UsageType = "open_house"
	UsageTypePrivateShowing UsageType = "private_showing"
)

// String returns the string representation of UsageType
func (u UsageType) String() string {
	return string(u)
}

// IsValid checks if the usage type is valid
func (u UsageType) IsValid() bool {
	switch u {
	case UsageTypeFeatured, UsageTypePremium, UsageTypeOpenHouse, UsageTypePrivateShowing:
		return true
	}
	return false
}
