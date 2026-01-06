package domain

// InteractionType represents the type of user interaction being tracked
type InteractionType string

const (
	// View Interactions
	InteractionViewListing       InteractionType = "view_listing"
	InteractionViewListingDetail InteractionType = "view_listing_detail"
	InteractionViewMedia         InteractionType = "view_media" // Photos/Videos
	InteractionViewMap           InteractionType = "view_map"

	// Explicit Actions (High Value)
	InteractionSaveListing    InteractionType = "save_listing"
	InteractionUnsaveListing  InteractionType = "unsave_listing"
	InteractionShareListing   InteractionType = "share_listing"
	InteractionContactOwner   InteractionType = "contact_owner"
	InteractionRequestViewing InteractionType = "request_viewing"
	InteractionBookingRequest InteractionType = "booking_request"

	// Search Behavior
	InteractionSearch      InteractionType = "search"
	InteractionFilterApply InteractionType = "filter_apply"

	// Implicit Signals (Calculated client-side)
	InteractionScrollDeep    InteractionType = "scroll_deep"     // > 75% depth
	InteractionTimeMilestone InteractionType = "time_milestone" // 30s, 60s, 120s on page
)

// String returns the string representation of InteractionType
func (t InteractionType) String() string {
	return string(t)
}

// IsValid checks if the interaction type is valid
func (t InteractionType) IsValid() bool {
	switch t {
	case InteractionViewListing,
		InteractionViewListingDetail,
		InteractionViewMedia,
		InteractionViewMap,
		InteractionSaveListing,
		InteractionUnsaveListing,
		InteractionShareListing,
		InteractionContactOwner,
		InteractionRequestViewing,
		InteractionBookingRequest,
		InteractionSearch,
		InteractionFilterApply,
		InteractionScrollDeep,
		InteractionTimeMilestone:
		return true
	default:
		return false
	}
}

// IsHighValue returns true for interactions that indicate strong user intent
func (t InteractionType) IsHighValue() bool {
	switch t {
	case InteractionSaveListing,
		InteractionShareListing,
		InteractionContactOwner,
		InteractionRequestViewing,
		InteractionBookingRequest:
		return true
	default:
		return false
	}
}

// ParseInteractionType converts a string to InteractionType
func ParseInteractionType(s string) InteractionType {
	return InteractionType(s)
}

// EntityType represents the type of entity being interacted with
type EntityType string

const (
	EntityTypeListing EntityType = "listing"
	EntityTypeSearch  EntityType = "search"
	EntityTypeProfile EntityType = "profile"
)

// String returns the string representation of EntityType
func (t EntityType) String() string {
	return string(t)
}

// IsValid checks if the entity type is valid
func (t EntityType) IsValid() bool {
	switch t {
	case EntityTypeListing, EntityTypeSearch, EntityTypeProfile:
		return true
	default:
		return false
	}
}

// ParseEntityType converts a string to EntityType
func ParseEntityType(s string) EntityType {
	return EntityType(s)
}

// PeriodType represents the aggregation period for analytics
type PeriodType string

const (
	PeriodHour PeriodType = "hour"
	PeriodDay  PeriodType = "day"
	PeriodWeek PeriodType = "week"
)

// String returns the string representation of PeriodType
func (t PeriodType) String() string {
	return string(t)
}

// DeviceType represents the device category
type DeviceType string

const (
	DeviceDesktop DeviceType = "desktop"
	DeviceMobile  DeviceType = "mobile"
	DeviceTablet  DeviceType = "tablet"
	DeviceUnknown DeviceType = "unknown"
)

// String returns the string representation of DeviceType
func (t DeviceType) String() string {
	return string(t)
}

// Platform represents the application platform
type Platform string

const (
	PlatformWeb     Platform = "web"
	PlatformIOS     Platform = "ios"
	PlatformAndroid Platform = "android"
	PlatformUnknown Platform = "unknown"
)

// String returns the string representation of Platform
func (t Platform) String() string {
	return string(t)
}
