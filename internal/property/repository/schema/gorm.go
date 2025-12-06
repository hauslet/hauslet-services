package schema

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// --- 1. PROPERTY (The Physical Asset) ---
type Property struct {
	ID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	PublicID string    `gorm:"type:varchar(8);uniqueIndex;not null"`

	// Basic info
	UnitNumber string `gorm:"type:varchar(50)"`
	Address    string `gorm:"type:varchar(500);not null"`
	City       string `gorm:"type:varchar(100);index"`
	State      string `gorm:"type:varchar(100);index"`
	PostalCode string `gorm:"type:varchar(20)"`

	// Enums
	Country CountryCode `gorm:"type:varchar(50);default:'NG'"`

	// PostGIS geospatial location (preferred for queries)
	Location *GeographyPoint `gorm:"type:geography(Point,4326);index:idx_properties_location_gist,type:gist" json:"location,omitempty"`

	// Classification (Using Enums)
	PropertyClass     PropertyClass     `gorm:"type:varchar(50);default:'residential';index"`
	PropertyType      PropertyType      `gorm:"type:varchar(50);default:'apartment';index"`
	FurnishingType    FurnishingType    `gorm:"type:varchar(50);default:'furnished'"`
	PropertyCondition PropertyCondition `gorm:"type:varchar(50);default:'used'"`

	// Details
	Bedrooms      *int `gorm:"type:int"`
	Bathrooms     *int `gorm:"type:int"`
	Toilets       *int `gorm:"type:int"`
	HalfBathrooms *int `gorm:"type:int"`
	Floors        *int `gorm:"type:int"`
	Units         int  `gorm:"type:int;default:1"`

	OwnerID uuid.UUID `gorm:"type:uuid;not null;index;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`

	SquareMeters float64  `gorm:"type:float;default:0"`
	FloorArea    *float64 `gorm:"type:float"`

	// JSON fields (Validated via Hook)
	Amenities          []string `gorm:"serializer:json"`
	FeaturesCommercial []string `gorm:"serializer:json"`

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// Hook to validate Property Enums before writing to DB
func (p *Property) BeforeSave(tx *gorm.DB) error {
	// Validate Amenities
	for _, a := range p.Amenities {
		if !IsValidAmenity(a) {
			return fmt.Errorf("invalid amenity: %s", a)
		}
	}
	return nil
}

// BeforeCreate hook to generate PublicID if not set
func (p *Property) BeforeCreate(tx *gorm.DB) error {
	if p.PublicID == "" {
		// Try to generate unique public ID with retries
		for attempts := 0; attempts < 10; attempts++ {
			publicID, err := generatePropertyPublicID()
			if err != nil {
				return fmt.Errorf("failed to generate public ID: %w", err)
			}

			// Check if this ID already exists
			var count int64
			if err := tx.Model(&Property{}).Where("public_id = ?", publicID).Count(&count).Error; err != nil {
				return fmt.Errorf("failed to check public ID uniqueness: %w", err)
			}

			if count == 0 {
				p.PublicID = publicID
				return nil
			}
		}
		return fmt.Errorf("failed to generate unique public ID after 10 attempts")
	}
	return nil
}

// generatePropertyPublicID generates a random public ID in format H + 7 alphanumeric chars
func generatePropertyPublicID() (string, error) {
	const (
		prefix = "H"
		length = 8
		chars  = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // Exclude 0, O, 1, I
	)

	b := make([]byte, length-1)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	result := make([]byte, length)
	result[0] = prefix[0]

	for i := 1; i < length; i++ {
		result[i] = chars[int(b[i-1])%len(chars)]
	}

	return string(result), nil
}

// --- 2. LISTING (The Commercial Offer) ---
type Listing struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`

	PropertyID uuid.UUID `gorm:"type:uuid;not null;index;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	OwnerID    uuid.UUID `gorm:"type:uuid;not null;index"`
	OwnerType  OwnerType `gorm:"size:50;not null"` // "landlord", "agent", etc.

	Slug        string       `gorm:"uniqueIndex;not null"`
	Title       string       `gorm:"not null"`
	Description string       `gorm:"type:text"`
	Currency    CurrencyCode `gorm:"size:3;default:'NGN'"`

	// Classification
	ListingType ListingType   `gorm:"size:50;index;not null"` // "sale", "rent", "shortlet"
	Status      ListingStatus `gorm:"size:50;index;default:'draft'"`

	// Publishing State
	Published   bool `gorm:"default:false;index"`
	PublishedAt *time.Time

	// Moderation
	LatestReviewStatus ReviewStatus `gorm:"default:'pending'"`

	// Metrics
	ViewCount     int `gorm:"default:0"`
	LastViewedAt  *time.Time
	FeaturedUntil *time.Time `gorm:"index"`
	BoostLevel    int        `gorm:"default:0"`

	// Audit
	CreatedBy       *uuid.UUID `gorm:"type:uuid"`
	UpdatedBy       *uuid.UUID `gorm:"type:uuid"`
	StatusChangedAt *time.Time
	ChangeReason    string

	HasCalendar bool `gorm:"default:false"`

	// Polymorphic JSON Details
	ShortletDetails *ShortletDetail `gorm:"type:jsonb;serializer:json"`
	RentalDetails   *RentalDetail   `gorm:"type:jsonb;serializer:json"`
	SaleDetails     *SaleDetail     `gorm:"type:jsonb;serializer:json"`

	// Vector embeddings for similarity search
	TextEmbedding *VectorEmbedding `gorm:"type:vector(768)" json:"text_embedding,omitempty"`

	// Embedding metadata
	EmbeddingModel       *string    `gorm:"type:varchar(100)" json:"embedding_model,omitempty"`
	EmbeddingVersion     *string    `gorm:"type:varchar(50)" json:"embedding_version,omitempty"`
	EmbeddingGeneratedAt *time.Time `json:"embedding_generated_at,omitempty"`

	// Media
	Media []ListingMedia `gorm:"foreignKey:ListingID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// Hook to validate Listing Enums
func (l *Listing) BeforeSave(tx *gorm.DB) error {
	// Validate OwnerType
	switch l.OwnerType {
	case OwnerLandlord, OwnerAgent, OwnerBusiness, OwnerIndividual:
		// Valid
	default:
		return fmt.Errorf("invalid owner type: %s", l.OwnerType)
	}

	// Validate Currency
	switch l.Currency {
	case CurrencyNGN, CurrencyGHS, CurrencyUSD, CurrencyEUR:
		// Valid
	default:
		return fmt.Errorf("invalid currency: %s", l.Currency)
	}

	// Validate Status
	switch l.Status {
	case StatusActive, StatusInactive, StatusDraft, StatusSold, StatusRented,
		StatusPendingVerification, StatusArchived, StatusSuspended,
		StatusUnderReview, StatusRequiresUpdates:
		// Valid
	default:
		return fmt.Errorf("invalid listing status: %s", l.Status)
	}

	return nil
}

// --- 3. HELPER STRUCTS (Embedded in JSONB) ---

type AmenityHighlight struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Icon    string `json:"icon"`
}

type RuleGroup struct {
	Category RuleCategory `json:"category" validate:"required"`
	Rules    []string     `json:"rules" validate:"required,min=1"`
}

type ServiceCharge struct {
	Name   string        `json:"name"`
	Period PaymentPeriod `json:"period"` // e.g., "yearly", "monthly"
	Amount float64       `json:"amount"`
}

type ShortletDetail struct {
	// Pricing
	NightlyRate   float64  `json:"nightly_rate"`
	CautionFee    *float64 `json:"caution_fee,omitempty"`
	CleaningFee   *float64 `json:"cleaning_fee,omitempty"`
	ServiceFee    *float64 `json:"service_fee,omitempty"`
	ExtraGuestFee *float64 `json:"extra_guest_fee,omitempty"`

	// Capacity & Rules
	MinNights      int  `json:"min_nights"`
	MaxNights      *int `json:"max_nights,omitempty"`
	MaxGuests      int  `json:"max_guests"`
	BaseGuestCount *int `json:"base_guest_count,omitempty"`

	// Logistics
	CheckInTime  *string `json:"check_in_time,omitempty"`
	CheckOutTime *string `json:"check_out_time,omitempty"`

	AccommodationType AccommodationType `json:"accommodation_type"`

	// Automation
	CalendarMonthsAhead  int  `json:"calendar_months_ahead"`
	AutoGenerateCalendar bool `json:"auto_generate_calendar"`

	// Metadata
	Rules               []RuleGroup        `json:"rules,omitempty"`
	AmenitiesHighlights []AmenityHighlight `json:"amenities_highlights,omitempty"`
}

type RentalDetail struct {
	RentalPrice       float64       `json:"rental_price"`
	RentalPricePeriod PaymentPeriod `json:"rental_price_period"`

	// Fees
	AgencyFee       *float64 `json:"agency_fee,omitempty"`
	LegalFee        *float64 `json:"legal_fee,omitempty"`
	RegistrationFee *float64 `json:"registration_fee,omitempty"`
	CautionFee      *float64 `json:"caution_fee,omitempty"`
	ServiceCharge   *float64 `json:"service_charge,omitempty"`

	ServiceChargeBreakdown *[]ServiceCharge `json:"service_charges,omitempty"`

	MinRentalPeriod        int        `json:"min_rental_period"`
	MaxRentalPeriod        *int       `json:"max_rental_period,omitempty"`
	RentalAvailabilityFrom *time.Time `json:"rental_availability_from,omitempty"`

	RentalTerms string      `json:"rental_terms,omitempty"`
	RentalRules []RuleGroup `json:"rental_rules,omitempty"`
}

type SaleDetail struct {
	SalePrice      float64 `json:"sale_price"`
	OwnershipTitle string  `json:"ownership_title"`
	PaymentPlan    bool    `json:"payment_plan"`

	YearBuilt     int `json:"year_built,omitempty"`
	YearRenovated int `json:"year_renovated,omitempty"`

	AgencyFee          *float64 `json:"agency_fee,omitempty"`
	LegalFee           *float64 `json:"legal_fee,omitempty"`
	SurveyFee          *float64 `json:"survey_fee,omitempty"`
	TitleProcessingFee *float64 `json:"title_processing_fee,omitempty"`
	DevelopmentFee     *float64 `json:"development_fee,omitempty"`
	OtherFees          *float64 `json:"other_fees,omitempty"`

	ServiceCharge          *float64         `json:"service_charge,omitempty"`
	ServiceChargeBreakdown *[]ServiceCharge `json:"service_charges,omitempty"`

	SaleTerms            string     `json:"sale_terms,omitempty"`
	SaleAvailabilityFrom *time.Time `json:"sale_availability_from,omitempty"`
}

type MediaType string

const (
	MediaTypeImage    MediaType = "image"
	MediaTypeVideo    MediaType = "video"
	MediaTypeTour360  MediaType = "360_tour"
	MediaTypeDocument MediaType = "document"
)

type ListingMedia struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ListingID uuid.UUID `gorm:"type:uuid;not null;index"` // Foreign Key

	// REQUIRED FIELDS
	URL  string    `gorm:"not null"`
	Type MediaType `gorm:"size:20;default:'image'"`

	// OPTIONAL Metadata
	Group    *string `gorm:"type:varchar(100);index"` // e.g., "kitchen", "exterior"
	Caption  *string `gorm:"size:100"`                // e.g., "Marble countertops"
	MimeType *string `gorm:"size:50"`                 // e.g., "image/jpeg"

	SizeBytes int64 `gorm:"default:0"`

	// FLAGS
	// IsPrimary: The main photo for the listing card.
	IsPrimary bool `gorm:"default:false;index"`

	// IsGroupCover: The main photo if viewing the "Kitchen" gallery specifically.
	IsGroupCover bool `gorm:"default:false"`

	Order int `gorm:"default:0;index"`

	// Vector embeddings for image similarity search
	ImageEmbedding *VectorEmbedding `gorm:"type:vector(512)" json:"image_embedding,omitempty"`

	// Embedding metadata
	EmbeddingModel       *string    `gorm:"type:varchar(100)" json:"embedding_model,omitempty"`
	EmbeddingVersion     *string    `gorm:"type:varchar(50)" json:"embedding_version,omitempty"`
	EmbeddingGeneratedAt *time.Time `json:"embedding_generated_at,omitempty"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// Validation to ensure Type is correct
func (m *ListingMedia) BeforeSave(tx *gorm.DB) error {
	switch m.Type {
	case MediaTypeImage, MediaTypeVideo, MediaTypeTour360, MediaTypeDocument:
		return nil
	default:
		return fmt.Errorf("invalid media type: %s", m.Type)
	}
}
