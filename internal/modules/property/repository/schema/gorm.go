package schema

import (
	"crypto/rand"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
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

	// PostGIS geospatial location
	Location *GeographyPoint `gorm:"type:geography(Point,4326);index:idx_properties_location_gist,type:gist" json:"location,omitempty"`

	// Classification
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

	// JSON fields
	Amenities          []AmenitiesType `gorm:"type:jsonb;serializer:json"`
	FeaturesCommercial []AmenitiesType `gorm:"type:jsonb;serializer:json"`

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// Hook to validate Property Enums
func (p *Property) BeforeSave(tx *gorm.DB) error {
	for _, a := range p.Amenities {
		for _, item := range a.Items {
			if !IsValidAmenity(item) {
				return fmt.Errorf("invalid amenity: %s", item)
			}
		}
	}
	return nil
}

// BeforeCreate hook to generate PublicID
func (p *Property) BeforeCreate(tx *gorm.DB) error {
	if p.PublicID == "" {
		for range 10 {
			publicID, err := generateRandomString("H", 8)
			if err != nil {
				return fmt.Errorf("failed to generate public ID: %w", err)
			}

			var count int64
			if err := tx.Model(&Property{}).Where("public_id = ?", publicID).Count(&count).Error; err != nil {
				return fmt.Errorf("check public ID: %w", err)
			}

			if count == 0 {
				p.PublicID = publicID
				return nil
			}
		}
		return fmt.Errorf("failed to generate unique public ID")
	}
	return nil
}

// --- 2. LISTING (The Commercial Offer) ---
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

type AmenityHighlight struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Icon    string `json:"icon"`
}

type RuleItem struct {
	Name        RuleSubCategory `json:"name" validate:"required"`
	Description map[string]any  `json:"description" validate:"required,min=1"`
}

type RuleGroup struct {
	Category RuleCategory `json:"category" validate:"required"`
	Rules    []RuleItem   `json:"rules" validate:"required,min=1"`
}

type ServiceCharge struct {
	Name   string        `json:"name"`
	Period PaymentPeriod `json:"period"`
	Amount float64       `json:"amount"`
}

type AmenitiesType struct {
	Group string   `json:"group"`
	Items []string `json:"items"`
}

type ShortletDetail struct {
	NightlyRate   float64  `json:"nightly_rate"`
	CautionFee    *float64 `json:"caution_fee,omitempty"`
	CleaningFee   *float64 `json:"cleaning_fee,omitempty"`
	ServiceFee    *float64 `json:"service_fee,omitempty"`
	ExtraGuestFee *float64 `json:"extra_guest_fee,omitempty"`

	MinNights      int  `json:"min_nights"`
	MaxNights      *int `json:"max_nights,omitempty"`
	MaxGuests      int  `json:"max_guests"`
	BaseGuestCount *int `json:"base_guest_count,omitempty"`

	CheckInTime  *string `json:"check_in_time,omitempty"`
	CheckOutTime *string `json:"check_out_time,omitempty"`

	AccommodationType  AccommodationType `json:"accommodation_type"`
	AutoAcceptBookings bool              `json:"auto_accept_bookings" gorm:"default:true"`

	CalendarMonthsAhead  int  `json:"calendar_months_ahead"`
	AutoGenerateCalendar bool `json:"auto_generate_calendar"`

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

type RentalDetail struct {
	RentalPrice       float64       `json:"rental_price"`
	RentalPricePeriod PaymentPeriod `json:"rental_price_period"`

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

	// Showing availability windows for viewings
	ShowingAvailability *[]ShowingAvailability `json:"showing_availability,omitempty"`
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

	// Showing availability windows for viewings
	ShowingAvailability *[]ShowingAvailability `json:"showing_availability,omitempty"`
}

type MediaType string

const (
	MediaTypeImage    MediaType = "image"
	MediaTypeVideo    MediaType = "video"
	MediaTypeTour360  MediaType = "360_tour"
	MediaTypeDocument MediaType = "document"
)

type Thumbnail struct {
	Key      string `json:"key"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Size     int64  `json:"size_bytes"`
	MimeType string `json:"mime_type"`
}

type ThumbnailMap map[string]Thumbnail

func (tm ThumbnailMap) Value() (driver.Value, error) {
	return json.Marshal(tm)
}

func (tm *ThumbnailMap) Scan(value interface{}) error {
	return json.Unmarshal(value.([]byte), tm)
}

type ListingMedia struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ListingID uuid.UUID `gorm:"type:uuid;not null;index"`

	Key  string    `gorm:"not null"`
	Type MediaType `gorm:"size:20;default:'image'"`

	Thumbnails ThumbnailMap `gorm:"type:jsonb;serializer:json"`

	Group    *string `gorm:"type:varchar(100);index"`
	Caption  *string `gorm:"size:100"`
	MimeType *string `gorm:"size:50"`

	SizeBytes int64 `gorm:"default:0"`
	Duration  *int  `gorm:"default:null"`

	IsPrimary    bool `gorm:"default:false;index"`
	IsGroupCover bool `gorm:"default:false"`
	Order        int  `gorm:"default:0;index"`

	ImageEmbedding       *VectorEmbedding `gorm:"type:vector(512)"`
	EmbeddingModel       *string          `gorm:"type:varchar(100)"`
	EmbeddingVersion     *string          `gorm:"type:varchar(50)"`
	EmbeddingGeneratedAt *time.Time

	UrlGeneratedAt time.Time `gorm:"index"`
	Uploaded       bool      `gorm:"default:false;index"`
	UploadedAt     time.Time `gorm:"index"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (m *ListingMedia) BeforeSave(tx *gorm.DB) error {
	if tx != nil && tx.Statement != nil {
		rv := tx.Statement.ReflectValue
		if rv.IsValid() {
			for rv.Kind() == reflect.Pointer && !rv.IsNil() {
				rv = rv.Elem()
			}
			if rv.IsValid() && rv.Kind() == reflect.Struct {
				if !tx.Statement.Changed("Type") {
					return nil
				}
			}
		}
	}

	switch m.Type {
	case MediaTypeImage, MediaTypeVideo, MediaTypeTour360, MediaTypeDocument:
		return nil
	default:
		return fmt.Errorf("invalid media type: %s", m.Type)
	}
}

// generateRandomString creates a random string of specified total length,
func generateRandomString(prefix string, totalLength int) (string, error) {
	// If prefix is "H" and totalLength is 8, we need 7 random chars
	randomLen := totalLength - len(prefix)
	if randomLen < 1 {
		return "", fmt.Errorf("prefix is longer than total length")
	}

	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, randomLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	var result strings.Builder
	result.WriteString(prefix)

	for i := 0; i < randomLen; i++ {
		result.WriteByte(chars[int(b[i])%len(chars)])
	}

	return result.String(), nil
}
