package domain

import (
	"time"

	"github.com/google/uuid"
)

type ContentType string
type ModerationStatus string

const (
	ContentTypeListingText  ContentType = "listing_text"
	ContentTypeListingImage ContentType = "listing_image"
	ContentTypeListingVideo ContentType = "listing_video"
)

const (
	ModerationStatusPending   ModerationStatus = "pending"
	ModerationStatusAccepted  ModerationStatus = "accepted"
	ModerationStatusEscalated ModerationStatus = "escalated"
	ModerationStatusRejected  ModerationStatus = "rejected"
)

// --- 1. PROPERTY (The Physical Asset) ---

// Location represents a geographic coordinate in WGS84 (default SRID 4326).
type Location struct {
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
	SRID int     `json:"srid,omitempty"`
}

// Valid reports whether the coordinates are within valid ranges.
func (l *Location) Valid() bool {
	if l == nil {
		return false
	}
	return l.Lat >= -90 && l.Lat <= 90 && l.Lng >= -180 && l.Lng <= 180
}

// Property represents the physical asset in the domain model
type Property struct {
	ID       uuid.UUID `json:"id"`
	PublicID string    `json:"public_id"` // User-friendly ID (e.g., H18U3AD8)

	// Basic info
	UnitNumber string      `json:"unit_number"`
	Address    string      `json:"address"`
	City       string      `json:"city"`
	State      string      `json:"state"`
	PostalCode string      `json:"postal_code"`
	Country    CountryCode `json:"country"`

	// Location (PostGIS geography)
	Location *Location `json:"location,omitempty"`

	// Classification
	PropertyClass     PropertyClass     `json:"property_class"`
	PropertyType      PropertyType      `json:"property_type"`
	FurnishingType    FurnishingType    `json:"furnishing_type"`
	PropertyCondition PropertyCondition `json:"property_condition"`

	// Details
	Bedrooms      *int `json:"bedrooms,omitempty"`
	Bathrooms     *int `json:"bathrooms,omitempty"`
	Toilets       *int `json:"toilets,omitempty"`
	HalfBathrooms *int `json:"half_bathrooms,omitempty"`
	Floors        *int `json:"floors,omitempty"`
	Units         int  `json:"units"`

	OwnerID uuid.UUID `json:"owner_id"`

	SquareMeters float64  `json:"square_meters"`
	FloorArea    *float64 `json:"floor_area,omitempty"`

	// Features (grouped)
	Amenities          []AmenityGroup `json:"amenities"`
	FeaturesCommercial []AmenityGroup `json:"features_commercial"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// --- 2. LISTING (The Commercial Offer) ---

// Listing represents the commercial offer in the domain model
type Listing struct {
	ID uuid.UUID `json:"id"`

	PropertyID uuid.UUID `json:"property_id"`
	OwnerID    uuid.UUID `json:"owner_id"`
	OwnerType  OwnerType `json:"owner_type"`

	Slug             string       `json:"slug"`
	Title            string       `json:"title"`
	Description      string       `json:"description"`
	ExtraDescription string       `json:"extra_description,omitempty"`
	Currency         CurrencyCode `json:"currency"`

	// Classification
	ListingType ListingType   `json:"listing_type"`
	Status      ListingStatus `json:"status"`

	// Publishing State
	Published   bool       `json:"published"`
	PublishedAt *time.Time `json:"published_at,omitempty"`

	// Moderation
	LatestReviewStatus ReviewStatus `json:"latest_review_status"`

	// Suspension
	SuspendedUntil   *time.Time `json:"suspended_until,omitempty"`
	SuspensionReason string     `json:"suspension_reason,omitempty"`

	// Verification
	IsVerified        bool              `json:"is_verified"`
	VerificationLevel VerificationLevel `json:"verification_level"`
	VerifiedAt        *time.Time        `json:"verified_at,omitempty"`

	// Embedding metadata
	EmbeddingModel        *string    `json:"embedding_model,omitempty"`
	EmbeddingVersion      *string    `json:"embedding_version,omitempty"`
	EmbeddingGeneratedAt  *time.Time `json:"embedding_generated_at,omitempty"`
	EmbeddingDocumentHash *string    `json:"embedding_document_hash,omitempty"`

	// Audit
	CreatedBy       *uuid.UUID `json:"created_by,omitempty"`
	UpdatedBy       *uuid.UUID `json:"updated_by,omitempty"`
	StatusChangedAt *time.Time `json:"status_changed_at,omitempty"`
	ChangeReason    string     `json:"change_reason,omitempty"`

	HasCalendar bool `json:"has_calendar"`

	// Polymorphic Details
	ShortletDetails *ShortletDetail `json:"shortlet_details,omitempty"`
	RentalDetails   *RentalDetail   `json:"rental_details,omitempty"`
	SaleDetails     *SaleDetail     `json:"sale_details,omitempty"`

	// Media
	Media []ListingMedia `json:"media,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// ScoredListing pairs a listing with a similarity/relevance score.
type ScoredListing struct {
	Listing        Listing
	Score          *float64
	TextMatchScore *float64 // Fuzzy text similarity score
	Ranking        int
	Location       *Location // Enriched from property (if available)
}

// --- 3. HELPER STRUCTS ---

// AmenityHighlight represents a highlighted amenity with description
type AmenityHighlight struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Icon    string `json:"icon"`
}

// AmenityGroup captures grouped amenities with a label and items.
type AmenityGroup struct {
	Group string   `json:"group"`
	Items []string `json:"items"`
}

// RuleGroup represents a group of rules for a listing
type RuleGroup struct {
	Category RuleCategory `json:"category"`
	Rules    []RuleItem   `json:"rules"`
}

// RuleItem represents a rule within a group, with a subcategory and details.
type RuleItem struct {
	Name        RuleSubCategory `json:"name"`
	Description map[string]any  `json:"description"`
}

// ServiceCharge represents a service charge breakdown
type ServiceCharge struct {
	Name   string        `json:"name"`
	Period PaymentPeriod `json:"period"`
	Amount float64       `json:"amount"`
}

// --- FEE & PRICING TYPES ---

// FeeFrequency defines when the fee is applied.
type FeeFrequency string

const (
	FeeFreqOneTime       FeeFrequency = "one_time"
	FeeFreqPerStay       FeeFrequency = "per_stay"
	FeeFreqPerGuest      FeeFrequency = "per_guest"
	FeeFreqPerExtraGuest FeeFrequency = "per_extra_guest"
	FeeFreqPerNight      FeeFrequency = "per_night" // Common for Shortlets
	FeeFreqPerMonth      FeeFrequency = "per_month" // Common for Rentals/Service Charge
	FeeFreqPerYear       FeeFrequency = "per_year"  // Common for Rentals/Service Charge
)

// FeeCategory helps frontend group fees (e.g., hiding legal fees in standard filters)
type FeeCategory string

const (
	FeeCatLegal   FeeCategory = "legal"
	FeeCatAgency  FeeCategory = "agency"
	FeeCatService FeeCategory = "service"
	FeeCatCaution FeeCategory = "caution" // Refundable deposits
	FeeCatOther   FeeCategory = "other"
)

// CustomFee is the generic structure for any cost associated with a listing
type CustomFee struct {
	Name         string       `json:"name"`                    // e.g., "Generator Fuel", "Legal Fee"
	Amount       float64      `json:"amount"`                  // The cost value
	Frequency    FeeFrequency `json:"frequency"`               // How often it is paid
	Category     FeeCategory  `json:"category"`                // For grouping logic
	IsRefundable bool         `json:"is_refundable,omitempty"` // Useful for Caution Fees
	IsOptional   bool         `json:"is_optional,omitempty"`   // Useful for things like "Extra Cleaning"
}

// DiscountType defines the behavior of the discount
type DiscountType string

const (
	DiscountTypeFlat         DiscountType = "flat"           // Standard % off (General Promotion)
	DiscountTypeLengthOfStay DiscountType = "length_of_stay" // Triggered by duration (Weekly/Monthly)
)

// Discount represents a price reduction
type Discount struct {
	Name       string       `json:"name"`                 // e.g., "Flash Sale", "Weekly Discount"
	Type       DiscountType `json:"type"`                 // "flat" or "length_of_stay"
	Percentage float64      `json:"percentage"`           // e.g., 10.0 for 10%
	MinNights  *int         `json:"min_nights,omitempty"` // Required if Type is "length_of_stay"
	Active     bool         `json:"active"`               // Corresponds to the toggle switch
}

// --- BOOKING SETTINGS ---

// ApprovalMethod defines how bookings are approved
type ApprovalMethod string

const (
	ApprovalMethodInstant ApprovalMethod = "instant" // Matches "Instant Book"
	ApprovalMethodRequest ApprovalMethod = "request" // Matches "Request to Book"
)

// GuestRequirements defines requirements for guests
type GuestRequirements struct {
	VerifiedID           bool `json:"verified_id"`            // Matches "Verified ID"
	PositiveReviewsOnly  bool `json:"positive_reviews_only"`  // Matches "Positive Reviews Only"
	ProfilePhotoRequired bool `json:"profile_photo_required"` // Matches "Profile Photo Required"
}

// BookingSettings defines booking configuration
type BookingSettings struct {
	ApprovalMethod    ApprovalMethod    `json:"approval_method"`
	GuestRequirements GuestRequirements `json:"guest_requirements"`
	PreBookingMessage string            `json:"pre_booking_message,omitempty"` // The text area input
}

// --- STAY LIMITS & ADVANCE BOOKING ---

// StayLimits defines minimum and maximum stay duration
type StayLimits struct {
	MinNights int  `json:"min_nights"`           // Matches "Minimum Night Stay"
	MaxNights *int `json:"max_nights,omitempty"` // Matches "Maximum Night Stay"
}

// AdvanceBooking defines advance booking settings
type AdvanceBooking struct {
	MonthsAhead    int `json:"months_ahead"`     // How far in advance can guests book
	MinNoticeHours int `json:"min_notice_hours"` // Minimum hours required before check-in
}

// ShortletDetail represents short-term rental specific details
type ShortletDetail struct {
	// Pricing
	NightlyRate float64     `json:"nightly_rate"`
	Fees        []CustomFee `json:"fees,omitempty"`
	Discounts   []Discount  `json:"discounts,omitempty"`

	// Booking Configuration
	BookingSettings BookingSettings `json:"booking_settings"`
	StayLimits      StayLimits      `json:"stay_limits"`
	AdvanceBooking  AdvanceBooking  `json:"advance_booking"`

	// Capacity
	MaxGuests      int  `json:"max_guests"`
	BaseGuestCount *int `json:"base_guest_count,omitempty"`

	// Logistics
	CheckInTime  *string `json:"check_in_time,omitempty"`
	CheckOutTime *string `json:"check_out_time,omitempty"`

	AccommodationType    AccommodationType `json:"accommodation_type"`
	AutoGenerateCalendar bool              `json:"auto_generate_calendar"`

	// Metadata
	Rules               []RuleGroup        `json:"rules,omitempty"`
	AmenitiesHighlights []AmenityHighlight `json:"amenities_highlights,omitempty"`
}

// ShowingAvailability defines when viewings can be scheduled
type ShowingAvailability struct {
	DayOfWeek string `json:"day_of_week"` // "monday", "tuesday", etc.
	StartTime string `json:"start_time"`  // HH:MM format (24h)
	EndTime   string `json:"end_time"`    // HH:MM format (24h)
	Timezone  string `json:"timezone"`    // IANA timezone (e.g., "Africa/Lagos")
}

// RentalDetail represents long-term rental specific details
type RentalDetail struct {
	RentalPrice       float64       `json:"rental_price"`
	RentalPricePeriod PaymentPeriod `json:"rental_price_period"`
	Discounts         []Discount    `json:"discounts,omitempty"`
	Fees              []CustomFee   `json:"fees,omitempty"` // Handles agency, legal, caution, service charges

	MinRentalPeriod        int        `json:"min_rental_period"`
	MaxRentalPeriod        *int       `json:"max_rental_period,omitempty"`
	RentalAvailabilityFrom *time.Time `json:"rental_availability_from,omitempty"`

	RentalTerms string      `json:"rental_terms,omitempty"`
	RentalRules []RuleGroup `json:"rental_rules,omitempty"`

	// Showing availability windows for viewings
	ShowingAvailability []ShowingAvailability `json:"showing_availability,omitempty"`
}

// SaleDetail represents property sale specific details
type SaleDetail struct {
	SalePrice      float64    `json:"sale_price"`
	OwnershipTitle string     `json:"ownership_title"`
	PaymentPlan    bool       `json:"payment_plan"`
	Discounts      []Discount `json:"discounts,omitempty"`

	YearBuilt     int `json:"year_built,omitempty"`
	YearRenovated int `json:"year_renovated,omitempty"`

	Fees []CustomFee `json:"fees,omitempty"` // Handles agency, legal, survey, development fees

	SaleTerms            string     `json:"sale_terms,omitempty"`
	SaleAvailabilityFrom *time.Time `json:"sale_availability_from,omitempty"`

	// Showing availability windows for viewings
	ShowingAvailability []ShowingAvailability `json:"showing_availability,omitempty"`
}

// ListingMedia represents media attached to a listing
type ListingMedia struct {
	ID        uuid.UUID `json:"id"`
	ListingID uuid.UUID `json:"listing_id"`

	// Required fields
	URL        string       `json:"url"`
	Key        string       `json:"key"`
	Type       MediaType    `json:"type"`
	Thumbnails ThumbnailMap `json:"thumbnails,omitempty"`

	// Optional metadata
	Group    *string `json:"group,omitempty"`
	Caption  *string `json:"caption,omitempty"`
	MimeType *string `json:"mime_type,omitempty"`

	SizeBytes int64 `json:"size_bytes"`

	// Flags
	IsPrimary    bool `json:"is_primary"`
	IsGroupCover bool `json:"is_group_cover"`

	Order int `json:"order"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Thumbnail represents a generated thumbnail variant for media.
type Thumbnail struct {
	Key       string `json:"key"`
	URL       string `json:"url,omitempty"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	SizeBytes int64  `json:"size_bytes"`
	MimeType  string `json:"mime_type"`
}

// ThumbnailMap maps variant names (e.g., "small", "medium") to thumbnail metadata.
type ThumbnailMap map[string]Thumbnail

// ThumbnailVariant represents a named thumbnail variant for GraphQL responses.
// It combines the variant size (e.g., "small", "medium", "large") with the thumbnail metadata.
type ThumbnailVariant struct {
	Size      string `json:"size"`
	Key       string `json:"key"`
	URL       string `json:"url"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	SizeBytes int64  `json:"size_bytes"`
	MimeType  string `json:"mime_type"`
}

// ListingMediaInput represents input data for listing media
type ListingMediaInput struct {
	Type         MediaType `json:"type"`
	Group        *string   `json:"group,omitempty"`
	Caption      *string   `json:"caption,omitempty"`
	MimeType     *string   `json:"mime_type,omitempty"`
	SizeBytes    int64     `json:"size_bytes"`
	IsPrimary    bool      `json:"is_primary"`
	IsGroupCover bool      `json:"is_group_cover"`
	Order        int       `json:"order"`
	Filename     string    `json:"filename"`
	Duration     *int      `json:"duration,omitempty"` // in seconds, for videos
}

// ListingMediaUpdateInput represents updatable fields for listing media
type ListingMediaUpdateInput struct {
	Caption      *string `json:"caption,omitempty"`
	IsPrimary    *bool   `json:"is_primary,omitempty"`
	IsGroupCover *bool   `json:"is_group_cover,omitempty"`
	Order        *int    `json:"order,omitempty"`
}

// ListingMediaDeleteInput represents input for deleting listing media
type ListingMediaDeleteInput struct {
	MediaID uuid.UUID `json:"media_id"`
	Key     string    `json:"key"`
}

type FinalizedListingMediaItem struct {
	ID  uuid.UUID `json:"id"`
	Key string    `json:"key"`
}

// FinalizedListingMedia represents media that has been finalized for a listing
type FinalizedListingMedia struct {
	ListingID uuid.UUID `json:"listing_id"`
	MediaKeys []string  `json:"media_keys"`
}

// ListingMediaResult represents the result of uploading listing media
type ListingMediaResult struct {
	ID       uuid.UUID `json:"id"`
	Filename string    `json:"filename"`
	URL      string    `json:"url"`
	Key      string    `json:"key"`
}

// ListingCompleteness represents a completeness assessment for a listing.
type ListingCompleteness struct {
	ListingID        uuid.UUID `json:"listing_id"`
	HasBasicInfo     bool      `json:"has_basic_info"`
	HasPropertyInfo  bool      `json:"has_property_info"`
	HasPricingInfo   bool      `json:"has_pricing_info"`
	HasImages        bool      `json:"has_images"`
	HasDescription   bool      `json:"has_description"`
	CompletionScore  int       `json:"completion_score"` // 0-100
	ReadyToPublish   bool      `json:"ready_to_publish"`
	MissingFields    []string  `json:"missing_fields"`
	Recommendations  []string  `json:"recommendations"`
	LastCalculatedAt time.Time `json:"last_calculated_at"`
}

type AggregatedModeration struct {
	TargetID uuid.UUID

	Pending   int64
	Accepted  int64
	Rejected  int64
	Escalated int64

	ContentTypes []ContentType
	Reasons      []string
}

// --- DOMAIN METHODS ---

// IsActive checks if the listing is currently active
func (l *Listing) IsActive() bool {
	return l.Status == StatusActive && l.Published
}

// CanPublish checks if the listing can be published
func (l *Listing) CanPublish() bool {
	return l.Status == StatusActive && l.LatestReviewStatus == ReviewApproved
}

// GetPrimaryMedia returns the primary media for the listing
func (l *Listing) GetPrimaryMedia() *ListingMedia {
	for _, media := range l.Media {
		if media.IsPrimary {
			return &media
		}
	}
	return nil
}

// HasLocation checks if the property has location coordinates
func (p *Property) HasLocation() bool {
	return p.Location != nil && p.Location.Valid()
}

// IsCommercial checks if the property is commercial
func (p *Property) IsCommercial() bool {
	return p.PropertyClass == ClassCommercial
}

// IsResidential checks if the property is residential
func (p *Property) IsResidential() bool {
	return p.PropertyClass == ClassResidential
}
