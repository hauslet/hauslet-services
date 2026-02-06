package schema

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// --- LISTING (The Commercial Offer) ---
type Listing struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`

	// Removed uniqueIndex from PropertyID to allow multiple listings per property
	PropertyID uuid.UUID `gorm:"type:uuid;not null;index;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	OwnerID    uuid.UUID `gorm:"type:uuid;not null;index"`
	OwnerType  OwnerType `gorm:"size:50;not null"`

	Slug             string       `gorm:"uniqueIndex;not null"`
	Title            string       `gorm:"not null"`
	Description      string       `gorm:"type:text"`
	ExtraDescription string       `gorm:"type:text"`
	Currency         CurrencyCode `gorm:"size:3;default:'NGN'"`

	// Classification
	ListingType ListingType   `gorm:"size:50;index;not null"` // "sale", "rent", "shortlet"
	Status      ListingStatus `gorm:"size:50;index;default:'draft'"`

	// Publishing State
	Published   bool `gorm:"default:false;index"`
	PublishedAt *time.Time

	// Moderation
	LatestReviewStatus ReviewStatus `gorm:"default:'pending'"`

	// Suspension
	SuspendedUntil             *time.Time `gorm:"index"`
	SuspensionReason           string     `gorm:"type:text"`
	SuspensionEndingNotifiedAt *time.Time // Tracks when suspension ending notification was sent

	// Verification
	IsVerified        bool              `gorm:"default:false;index"`
	VerificationLevel VerificationLevel `gorm:"size:50;default:'none'"`
	VerifiedAt        *time.Time

	// Audit
	CreatedBy       *uuid.UUID `gorm:"type:uuid"`
	UpdatedBy       *uuid.UUID `gorm:"type:uuid"`
	StatusChangedAt *time.Time
	ChangeReason    string

	HasCalendar bool `gorm:"default:false"`

	// Polymorphic JSON Details (Refactored to use generic Fees)
	ShortletDetails *ShortletDetail `gorm:"type:jsonb;serializer:json"`
	RentalDetails   *RentalDetail   `gorm:"type:jsonb;serializer:json"`
	SaleDetails     *SaleDetail     `gorm:"type:jsonb;serializer:json"`

	// Vector embeddings
	TextEmbedding         *VectorEmbedding `gorm:"type:vector(768)" json:"text_embedding,omitempty"`
	EmbeddingModel        *string          `gorm:"type:varchar(100)" json:"embedding_model,omitempty"`
	EmbeddingVersion      *string          `gorm:"type:varchar(50)" json:"embedding_version,omitempty"`
	EmbeddingGeneratedAt  *time.Time       `json:"embedding_generated_at,omitempty"`
	EmbeddingDocumentHash *string          `gorm:"type:char(64)" json:"embedding_document_hash,omitempty"`

	// Associations
	Media    []ListingMedia `gorm:"foreignKey:ListingID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Property *Property      `gorm:"foreignKey:PropertyID;references:ID"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// --- POLYMORPHIC DETAILS ---

type ShortletDetail struct {
	NightlyRate float64     `json:"nightly_rate"`
	Fees        []CustomFee `json:"fees,omitempty"`
	Discounts   []Discount  `json:"discounts,omitempty"`

	BookingSettings BookingSettings `json:"booking_settings"`

	StayLimits     StayLimits     `json:"stay_limits"`
	AdvanceBooking AdvanceBooking `json:"advance_booking"`

	// Existing fields...
	CheckInTime  *string `json:"check_in_time,omitempty"`
	CheckOutTime *string `json:"check_out_time,omitempty"`

	MaxGuests      int  `json:"max_guests"`
	BaseGuestCount *int `json:"base_guest_count,omitempty"`

	AccommodationType AccommodationType `json:"accommodation_type"`

	AutoGenerateCalendar bool `json:"auto_generate_calendar"`

	Rules               []RuleGroup        `json:"rules,omitempty"`
	AmenitiesHighlights []AmenityHighlight `json:"amenities_highlights,omitempty"`
}

type RentalDetail struct {
	RentalPrice       float64       `json:"rental_price"`
	RentalPricePeriod PaymentPeriod `json:"rental_price_period"`
	Discounts         []Discount    `json:"discounts,omitempty"`
	Fees              []CustomFee   `json:"fees,omitempty"` // Handles agency, legal, caution, service charges

	MinRentalPeriod        int        `json:"min_rental_period"`
	MaxRentalPeriod        *int       `json:"max_rental_period,omitempty"`
	RentalAvailabilityFrom *time.Time `json:"rental_availability_from,omitempty"`

	RentalTerms         string                 `json:"rental_terms,omitempty"`
	RentalRules         []RuleGroup            `json:"rental_rules,omitempty"`
	ShowingAvailability *[]ShowingAvailability `json:"showing_availability,omitempty"`
}

type SaleDetail struct {
	SalePrice      float64    `json:"sale_price"`
	OwnershipTitle string     `json:"ownership_title"`
	PaymentPlan    bool       `json:"payment_plan"`
	Discounts      []Discount `json:"discounts,omitempty"`

	YearBuilt     int `json:"year_built,omitempty"`
	YearRenovated int `json:"year_renovated,omitempty"`

	Fees []CustomFee `json:"fees,omitempty"` // Handles agency, legal, survey, development fees

	SaleTerms            string                 `json:"sale_terms,omitempty"`
	SaleAvailabilityFrom *time.Time             `json:"sale_availability_from,omitempty"`
	ShowingAvailability  *[]ShowingAvailability `json:"showing_availability,omitempty"`
}

// --- HOOKS ---

// BeforeCreate hook for Listing to ensure unique Slug
func (l *Listing) BeforeCreate(tx *gorm.DB) error {
	if l.Slug == "" {
		// Fallback slug generation if not provided
		baseSlug := strings.ReplaceAll(strings.ToLower(l.Title), " ", "-")
		if len(baseSlug) > 50 {
			baseSlug = baseSlug[:50]
		}

		// Try to generate unique slug
		for i := range 10 {
			candidate := baseSlug
			if i > 0 {
				rnd, _ := generateRandomString("", 4)
				candidate = fmt.Sprintf("%s-%s", baseSlug, rnd)
			}

			var count int64
			if err := tx.Model(&Listing{}).Where("slug = ?", candidate).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				l.Slug = candidate
				return nil
			}
		}
		return fmt.Errorf("failed to generate unique slug")
	}
	return nil
}

// BeforeSave Hook: Validates Enums AND Enforces "One Active Listing" Rule
func (l *Listing) BeforeSave(tx *gorm.DB) error {
	changed := func(field string) bool {
		return tx != nil && tx.Statement != nil && tx.Statement.Changed(field)
	}

	// 1. Enum Validation
	if changed("owner_type") {
		switch l.OwnerType {
		case OwnerLandlord, OwnerAgent, OwnerBusiness, OwnerIndividual:
		default:
			return fmt.Errorf("invalid owner type: %s", l.OwnerType)
		}
	}
	if changed("currency") {
		switch l.Currency {
		case CurrencyNGN, CurrencyGHS, CurrencyUSD, CurrencyEUR:
		default:
			return fmt.Errorf("invalid currency: %s", l.Currency)
		}
	}
	if changed("status") {
		switch l.Status {
		case StatusActive, StatusInactive, StatusDraft, StatusSold, StatusRented,
			StatusPendingVerification, StatusArchived, StatusSuspended,
			StatusUnderReview, StatusRequiresUpdates:
		default:
			return fmt.Errorf("invalid listing status: %s", l.Status)
		}
	}

	// 2. Logic Check: Enforce One "Active" Listing per Type per Property
	// We only care if the status is currently set to ACTIVE
	if l.Status == StatusActive {
		var count int64
		query := tx.Model(&Listing{}).
			Where("property_id = ?", l.PropertyID).
			Where("listing_type = ?", l.ListingType).
			Where("status = ?", StatusActive)

		// If this is an update (ID exists), exclude the current record from the check
		if l.ID != uuid.Nil {
			query = query.Where("id != ?", l.ID)
		}

		if err := query.Count(&count).Error; err != nil {
			return fmt.Errorf("failed to validate active listings: %w", err)
		}

		if count > 0 {
			return fmt.Errorf("property already has an active '%s' listing. archive it before activating this one", l.ListingType)
		}
	}

	return nil
}
