package domain

import (
	"regexp"
	"testing"
	"time"

	"hauslet/internal/modules/property/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestLocationValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		loc  *Location
		want bool
	}{
		{name: "nil location", loc: nil, want: false},
		{name: "valid coordinates", loc: &Location{Lat: 6.5, Lng: 3.4}, want: true},
		{name: "invalid latitude", loc: &Location{Lat: 91, Lng: 3.4}, want: false},
		{name: "invalid longitude", loc: &Location{Lat: 6.5, Lng: -181}, want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.loc.Valid(); got != tt.want {
				t.Fatalf("Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPropertyLocationAndClassificationHelpers(t *testing.T) {
	t.Parallel()

	withLocation := &Property{Location: &Location{Lat: 6.45, Lng: 3.39}}
	if !withLocation.HasLocation() {
		t.Fatalf("HasLocation() = false, want true")
	}

	noLocation := &Property{}
	if noLocation.HasLocation() {
		t.Fatalf("HasLocation() = true, want false for nil location")
	}

	invalidLocation := &Property{Location: &Location{Lat: 400, Lng: 0}}
	if invalidLocation.HasLocation() {
		t.Fatalf("HasLocation() = true, want false for invalid coordinates")
	}

	commercial := &Property{PropertyClass: ClassCommercial}
	if !commercial.IsCommercial() || commercial.IsResidential() {
		t.Fatalf("classification helpers returned incorrect values for commercial property")
	}

	residential := &Property{PropertyClass: ClassResidential}
	if !residential.IsResidential() || residential.IsCommercial() {
		t.Fatalf("classification helpers returned incorrect values for residential property")
	}
}

func TestGeneratePropertyPublicIDFormat(t *testing.T) {
	t.Parallel()

	id, err := GeneratePropertyPublicID()
	if err != nil {
		t.Fatalf("GeneratePropertyPublicID() unexpected error: %v", err)
	}

	re := regexp.MustCompile(`^H[ABCDEFGHJKLMNPQRSTUVWXYZ23456789]{7}$`)
	if !re.MatchString(id) {
		t.Fatalf("GeneratePropertyPublicID() = %q, does not match expected format", id)
	}
}

func TestListingStateHelpers(t *testing.T) {
	t.Parallel()

	listing := &Listing{
		Status:    StatusActive,
		Published: true,
	}
	if !listing.IsActive() {
		t.Fatalf("IsActive() = false, want true for active + published")
	}

	listing.Published = false
	if listing.IsActive() {
		t.Fatalf("IsActive() = true, want false when not published")
	}

	listing.Status = StatusInactive
	if listing.IsActive() {
		t.Fatalf("IsActive() = true, want false when status is not active")
	}

	listing.Status = StatusActive
	listing.Published = true
	listing.LatestReviewStatus = ReviewApproved
	if !listing.CanPublish() {
		t.Fatalf("CanPublish() = false, want true for active + approved")
	}

	listing.LatestReviewStatus = ReviewPending
	if listing.CanPublish() {
		t.Fatalf("CanPublish() = true, want false when review not approved")
	}
}

func TestListingIsFeatured(t *testing.T) {
	t.Parallel()

	l := &Listing{}
	if l.IsFeatured() {
		t.Fatalf("IsFeatured() = true, want false when FeaturedUntil is nil")
	}

	past := time.Now().Add(-1 * time.Hour)
	l.FeaturedUntil = &past
	if l.IsFeatured() {
		t.Fatalf("IsFeatured() = true, want false when FeaturedUntil is in the past")
	}

	future := time.Now().Add(2 * time.Hour)
	l.FeaturedUntil = &future
	if !l.IsFeatured() {
		t.Fatalf("IsFeatured() = false, want true when FeaturedUntil is in the future")
	}
}

func TestListingGetPrimaryMedia(t *testing.T) {
	t.Parallel()

	primaryID := uuid.New()
	listing := &Listing{
		Media: []ListingMedia{
			{ID: uuid.New(), IsPrimary: false},
			{ID: primaryID, IsPrimary: true, URL: "primary.jpg"},
		},
	}

	primary := listing.GetPrimaryMedia()
	if primary == nil {
		t.Fatalf("GetPrimaryMedia() = nil, want media")
	}
	if primary.ID != primaryID || primary.URL != "primary.jpg" {
		t.Fatalf("GetPrimaryMedia() returned unexpected media: %+v", primary)
	}

	emptyListing := &Listing{}
	if emptyListing.GetPrimaryMedia() != nil {
		t.Fatalf("GetPrimaryMedia() = non-nil, want nil when no primary media exists")
	}
}

func TestMapPropertyFromSchema(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	createdAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	schemaProp := &schema.Property{
		ID:                uuid.New(),
		PublicID:          "H1234ABC",
		UnitNumber:        "12B",
		Address:           "123 Ikoyi Crescent",
		City:              "Lagos",
		State:             "Lagos",
		PostalCode:        "100001",
		Country:           schema.CountryNG,
		Location:          &schema.GeographyPoint{Lat: 6.45, Lng: 3.39, SRID: 4326},
		PropertyClass:     schema.ClassCommercial,
		PropertyType:      schema.TypeOfficeSpace,
		FurnishingType:    schema.Furnished,
		PropertyCondition: schema.ConditionNew,
		Bedrooms:          intPtr(3),
		Bathrooms:         intPtr(2),
		Toilets:           intPtr(3),
		HalfBathrooms:     intPtr(1),
		Floors:            intPtr(10),
		Units:             2,
		OwnerID:           ownerID,
		SquareMeters:      150.5,
		FloorArea:         floatPtr(120.3),

		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	result := MapPropertyFromSchema(schemaProp)
	if result == nil {
		t.Fatalf("MapPropertyFromSchema() returned nil")
	}

	if result.PublicID != schemaProp.PublicID || result.OwnerID != ownerID || result.PropertyClass != ClassCommercial {
		t.Fatalf("MapPropertyFromSchema() returned unexpected core fields: %+v", result)
	}

	if result.Location == nil || result.Location.Lat != 6.45 || result.Location.Lng != 3.39 || result.Location.SRID != 4326 {
		t.Fatalf("MapPropertyFromSchema() returned incorrect location: %+v", result.Location)
	}

	if result.Bedrooms == nil || *result.Bedrooms != 3 || result.FloorArea == nil || *result.FloorArea != 120.3 {
		t.Fatalf("MapPropertyFromSchema() did not map pointer fields correctly: %+v", result)
	}

	if len(result.Amenities) != 2 || len(result.FeaturesCommercial) != 1 {
		t.Fatalf("MapPropertyFromSchema() did not map slices correctly: %+v", result)
	}
}

func TestMapPropertyToSchema(t *testing.T) {
	t.Parallel()

	withLocation := &Property{
		ID:                uuid.New(),
		PublicID:          "HABCDEFG",
		Address:           "15 Lekki Phase 1",
		City:              "Lagos",
		State:             "Lagos",
		PostalCode:        "105102",
		Country:           CountryNG,
		PropertyClass:     ClassResidential,
		PropertyType:      TypeDuplex,
		FurnishingType:    FurnishingType(Furnished),
		PropertyCondition: ConditionRenovated,
		OwnerID:           uuid.New(),
		Units:             1,
		SquareMeters:      90.5,
		Location:          &Location{Lat: 6.43, Lng: 3.42}, // SRID should default to 4326
		CreatedAt:         time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:         time.Date(2024, 3, 2, 0, 0, 0, 0, time.UTC),
	}

	mapped := MapPropertyToSchema(withLocation)
	if mapped == nil {
		t.Fatalf("MapPropertyToSchema() returned nil")
	}
	if mapped.Location == nil || mapped.Location.SRID != 4326 || mapped.Location.Lat != withLocation.Location.Lat {
		t.Fatalf("MapPropertyToSchema() did not map location with default SRID: %+v", mapped.Location)
	}
	if mapped.PropertyType != schema.PropertyType(TypeDuplex) || mapped.PropertyClass != schema.PropertyClass(ClassResidential) {
		t.Fatalf("MapPropertyToSchema() returned incorrect enum conversions: %+v", mapped)
	}

	invalidLocation := &Property{Location: &Location{Lat: 120, Lng: 0}}
	if res := MapPropertyToSchema(invalidLocation); res.Location != nil {
		t.Fatalf("MapPropertyToSchema() should omit invalid location, got %+v", res.Location)
	}
}

func TestMapPropertiesFromSchema(t *testing.T) {
	t.Parallel()

	if res := MapPropertiesFromSchema(nil); res != nil {
		t.Fatalf("MapPropertiesFromSchema(nil) = %#v, want nil", res)
	}

	propID := uuid.New()
	props := []schema.Property{
		{ID: propID, PublicID: "H1"},
	}

	result := MapPropertiesFromSchema(props)
	if len(result) != 1 || result[0].ID != propID || result[0].PublicID != "H1" {
		t.Fatalf("MapPropertiesFromSchema() returned unexpected slice: %#v", result)
	}
}

func TestMapListingFromSchema(t *testing.T) {
	t.Parallel()

	listingID := uuid.New()
	propertyID := uuid.New()
	ownerID := uuid.New()
	createdBy := uuid.New()
	updatedBy := uuid.New()

	publishedAt := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	lastViewed := time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)
	featuredUntil := time.Date(2024, 1, 3, 12, 0, 0, 0, time.UTC)
	statusChanged := time.Date(2024, 1, 4, 12, 0, 0, 0, time.UTC)
	deletedAt := time.Date(2024, 1, 5, 12, 0, 0, 0, time.UTC)

	amenitiesHighlights := []schema.AmenityHighlight{{Title: "Pool", Summary: "Rooftop pool", Icon: "pool"}}
	serviceCharges := []schema.ServiceCharge{{Name: "Service", Period: schema.PayMonthly, Amount: 2500}}
	saleCharges := []schema.ServiceCharge{{Name: "Development", Period: schema.PayOneTime, Amount: 5000}}

	schemaListing := &schema.Listing{
		ID:                 listingID,
		PropertyID:         propertyID,
		OwnerID:            ownerID,
		OwnerType:          schema.OwnerAgent,
		Slug:               "modern-office",
		Title:              "Modern Office",
		Description:        "Downtown workspace",
		Currency:           schema.CurrencyUSD,
		ListingType:        schema.ListingRent,
		Status:             schema.StatusActive,
		Published:          true,
		PublishedAt:        &publishedAt,
		LatestReviewStatus: schema.ReviewApproved,
		ViewCount:          12,
		LastViewedAt:       &lastViewed,
		FeaturedUntil:      &featuredUntil,
		BoostLevel:         2,
		CreatedBy:          &createdBy,
		UpdatedBy:          &updatedBy,
		StatusChangedAt:    &statusChanged,
		ChangeReason:       "manual update",
		HasCalendar:        true,
		ShortletDetails: &schema.ShortletDetail{
			NightlyRate:          50000,
			MinNights:            2,
			MaxGuests:            4,
			AccommodationType:    schema.AccEntirePlace,
			CalendarMonthsAhead:  6,
			AutoGenerateCalendar: true,
			AmenitiesHighlights:  amenitiesHighlights,
		},
		RentalDetails: &schema.RentalDetail{
			RentalPrice:            200000,
			RentalPricePeriod:      schema.PayYearly,
			MinRentalPeriod:        12,
			MaxRentalPeriod:        intPtr(24),
			RentalAvailabilityFrom: &publishedAt,
			ServiceChargeBreakdown: &serviceCharges,
		},
		SaleDetails: &schema.SaleDetail{
			SalePrice:              45000000,
			OwnershipTitle:         "C of O",
			PaymentPlan:            true,
			YearBuilt:              2020,
			YearRenovated:          2023,
			AgencyFee:              floatPtr(1000),
			ServiceChargeBreakdown: &saleCharges,
			SaleTerms:              "Negotiable",
			SaleAvailabilityFrom:   &featuredUntil,
		},
		Media: []schema.ListingMedia{
			{
				ID:        uuid.New(),
				ListingID: listingID,
				Key:       "1.jpg",
				Type:      schema.MediaTypeImage,
				IsPrimary: true,
				Order:     1,
				CreatedAt: publishedAt,
				UpdatedAt: publishedAt,
			},
		},
		CreatedAt: publishedAt,
		UpdatedAt: featuredUntil,
		DeletedAt: gorm.DeletedAt{Time: deletedAt, Valid: true},
	}

	result := MapListingFromSchema(schemaListing)
	if result == nil {
		t.Fatalf("MapListingFromSchema() returned nil")
	}

	if result.ID != listingID || result.PropertyID != propertyID || result.OwnerType != OwnerType(schema.OwnerAgent) {
		t.Fatalf("MapListingFromSchema() returned incorrect identifiers: %+v", result)
	}
	if result.Status != StatusActive || result.ListingType != ListingRent || !result.Published || result.LatestReviewStatus != ReviewApproved {
		t.Fatalf("MapListingFromSchema() returned incorrect status fields: %+v", result)
	}
	if result.DeletedAt == nil || !result.DeletedAt.Equal(deletedAt) {
		t.Fatalf("MapListingFromSchema() did not map DeletedAt correctly: %+v", result.DeletedAt)
	}
	if result.ShortletDetails == nil || len(result.ShortletDetails.Rules) != 1 || len(result.ShortletDetails.AmenitiesHighlights) != 1 {
		t.Fatalf("MapListingFromSchema() did not map shortlet details correctly: %+v", result.ShortletDetails)
	}
	if result.RentalDetails == nil || result.RentalDetails.ServiceChargeBreakdown == nil || len(*result.RentalDetails.ServiceChargeBreakdown) != 1 {
		t.Fatalf("MapListingFromSchema() did not map rental details correctly: %+v", result.RentalDetails)
	}
	if result.SaleDetails == nil || result.SaleDetails.ServiceChargeBreakdown == nil || len(*result.SaleDetails.ServiceChargeBreakdown) != 1 {
		t.Fatalf("MapListingFromSchema() did not map sale details correctly: %+v", result.SaleDetails)
	}
	if len(result.Media) != 1 || result.Media[0].Key != "1.jpg" || result.Media[0].Type != MediaTypeImage {
		t.Fatalf("MapListingFromSchema() did not map media correctly: %+v", result.Media)
	}
}

func TestMapListingToSchema(t *testing.T) {
	t.Parallel()

	listingID := uuid.New()
	propertyID := uuid.New()
	ownerID := uuid.New()
	createdBy := uuid.New()
	updatedBy := uuid.New()
	publishedAt := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	lastViewed := time.Date(2024, 6, 2, 0, 0, 0, 0, time.UTC)
	featuredUntil := time.Date(2024, 6, 3, 0, 0, 0, 0, time.UTC)
	statusChanged := time.Date(2024, 6, 4, 0, 0, 0, 0, time.UTC)
	deletedAt := time.Date(2024, 6, 5, 0, 0, 0, 0, time.UTC)

	mediaID := uuid.New()
	listing := &Listing{
		ID:                 listingID,
		PropertyID:         propertyID,
		OwnerID:            ownerID,
		OwnerType:          OwnerLandlord,
		Slug:               "lagos-shortlet",
		Title:              "Lagos Shortlet",
		Description:        "Cozy stay in Lekki",
		Currency:           CurrencyNGN,
		ListingType:        ListingShortLet,
		Status:             StatusUnderReview,
		Published:          true,
		PublishedAt:        &publishedAt,
		LatestReviewStatus: ReviewPending,
		ViewCount:          5,
		LastViewedAt:       &lastViewed,
		FeaturedUntil:      &featuredUntil,
		BoostLevel:         1,
		CreatedBy:          &createdBy,
		UpdatedBy:          &updatedBy,
		StatusChangedAt:    &statusChanged,
		ChangeReason:       "auto",
		HasCalendar:        true,
		ShortletDetails: &ShortletDetail{
			NightlyRate:          45000,
			MinNights:            1,
			MaxGuests:            3,
			AccommodationType:    AccEntirePlace,
			CalendarMonthsAhead:  3,
			AutoGenerateCalendar: true,
			AmenitiesHighlights: []AmenityHighlight{
				{Title: "Wifi", Summary: "Fast fibre", Icon: "wifi"},
			},
		},
		SaleDetails: &SaleDetail{
			SalePrice: 75000000,
			ServiceChargeBreakdown: &[]ServiceCharge{
				{Name: "Maintenance", Period: PayMonthly, Amount: 15000},
			},
			SaleAvailabilityFrom: &publishedAt,
		},
		Media: []ListingMedia{
			{
				ID:        mediaID,
				ListingID: listingID,
				URL:       "/media/primary.jpg",
				Type:      MediaTypeImage,
				IsPrimary: true,
				Order:     1,
				CreatedAt: publishedAt,
				UpdatedAt: publishedAt,
			},
		},
		CreatedAt: publishedAt,
		UpdatedAt: featuredUntil,
		DeletedAt: &deletedAt,
	}

	result := MapListingToSchema(listing)
	if result == nil {
		t.Fatalf("MapListingToSchema() returned nil")
	}
	if result.OwnerType != schema.OwnerType(OwnerLandlord) || result.Currency != schema.CurrencyCode(CurrencyNGN) {
		t.Fatalf("MapListingToSchema() converted enums incorrectly: %+v", result)
	}
	if !result.DeletedAt.Valid || !result.DeletedAt.Time.Equal(deletedAt) {
		t.Fatalf("MapListingToSchema() did not set DeletedAt correctly: %+v", result.DeletedAt)
	}
	if result.ShortletDetails == nil || len(result.ShortletDetails.Rules) != 1 || len(result.ShortletDetails.AmenitiesHighlights) != 1 {
		t.Fatalf("MapListingToSchema() did not map shortlet details correctly: %+v", result.ShortletDetails)
	}
	if result.SaleDetails == nil || result.SaleDetails.ServiceChargeBreakdown == nil || len(*result.SaleDetails.ServiceChargeBreakdown) != 1 {
		t.Fatalf("MapListingToSchema() did not map sale details correctly: %+v", result.SaleDetails)
	}
	if len(result.Media) != 1 || result.Media[0].ID != mediaID || result.Media[0].Type != schema.MediaType(MediaTypeImage) {
		t.Fatalf("MapListingToSchema() did not map media correctly: %+v", result.Media)
	}
}

func TestMapListingsFromSchema(t *testing.T) {
	t.Parallel()

	if res := MapListingsFromSchema(nil); res != nil {
		t.Fatalf("MapListingsFromSchema(nil) = %#v, want nil", res)
	}

	listingID := uuid.New()
	listings := []schema.Listing{{ID: listingID}}
	result := MapListingsFromSchema(listings)

	if len(result) != 1 || result[0].ID != listingID {
		t.Fatalf("MapListingsFromSchema() returned unexpected slice: %#v", result)
	}
}

func intPtr(v int) *int { return &v }

func floatPtr(v float64) *float64 { return &v }
