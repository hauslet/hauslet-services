package seeders

import (
	"fmt"
	"strings"
	"time"

	"hauslet/db/seeds/data"
	"hauslet/db/seeds/utils"
	propertySchema "hauslet/internal/modules/property/repository/schema"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type propertySeed struct {
	ID           uuid.UUID
	OwnerID      uuid.UUID
	Address      string
	City         string
	State        string
	PropertyType string
	Bedrooms     *int
}

// SeedListing seeds property listings linked to existing users and properties.
func SeedListing(ctx *SeedContext) error {
	var properties []propertySeed
	if err := ctx.DB.Table("properties").
		Select("id, owner_id, address, city, state, property_type, bedrooms").
		Scan(&properties).Error; err != nil {
		return fmt.Errorf("failed to fetch properties: %w", err)
	}

	if len(properties) == 0 {
		return fmt.Errorf("no properties found - seed property module first")
	}

	ownerTypes, err := loadOwnerTypes(ctx)
	if err != nil {
		return err
	}

	listingTypes := buildListingTypes(ctx.Config)
	utils.Shuffle(listingTypes)

	activeTracker := make(map[string]bool)
	for _, listingType := range listingTypes {
		property := utils.RandomChoice(properties)
		if err := createListing(ctx, property, listingType, ownerTypes, activeTracker); err != nil {
			return err
		}
	}

	return nil
}

func buildListingTypes(config *SeedConfig) []propertySchema.ListingType {
	shortletCount := int(float64(config.ListingCount) * config.ShortletRatio)
	rentCount := int(float64(config.ListingCount) * config.RentRatio)
	saleCount := config.ListingCount - shortletCount - rentCount

	listingTypes := make([]propertySchema.ListingType, 0, config.ListingCount)
	for i := 0; i < shortletCount; i++ {
		listingTypes = append(listingTypes, propertySchema.ListingShortLet)
	}
	for i := 0; i < rentCount; i++ {
		listingTypes = append(listingTypes, propertySchema.ListingRent)
	}
	for i := 0; i < saleCount; i++ {
		listingTypes = append(listingTypes, propertySchema.ListingSale)
	}

	return listingTypes
}

func loadOwnerTypes(ctx *SeedContext) (map[uuid.UUID]propertySchema.OwnerType, error) {
	ownerTypes := make(map[uuid.UUID]propertySchema.OwnerType)

	if !ctx.DB.Migrator().HasTable("profiles") {
		return ownerTypes, nil
	}

	rows, err := ctx.DB.Table("profiles").Select("user_id, user_types").Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profiles: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID uuid.UUID
		var userTypes pq.StringArray
		if err := rows.Scan(&userID, &userTypes); err != nil {
			return nil, fmt.Errorf("failed to scan profiles: %w", err)
		}
		ownerTypes[userID] = ownerTypeFromUserTypes(userTypes)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate profiles: %w", err)
	}

	return ownerTypes, nil
}

func ownerTypeFromUserTypes(userTypes []string) propertySchema.OwnerType {
	for _, userType := range userTypes {
		switch strings.ToLower(userType) {
		case "agent":
			return propertySchema.OwnerAgent
		case "landlord", "host", "cohost":
			return propertySchema.OwnerLandlord
		case "business":
			return propertySchema.OwnerBusiness
		}
	}
	return propertySchema.OwnerIndividual
}

func createListing(
	ctx *SeedContext,
	property propertySeed,
	listingType propertySchema.ListingType,
	ownerTypes map[uuid.UUID]propertySchema.OwnerType,
	activeTracker map[string]bool,
) error {
	title := buildListingTitle(property)
	description := buildListingDescription(property, listingType)

	status, published := chooseListingStatus(property.ID, listingType, activeTracker)
	createdAt := utils.RandomPastDate(180)

	listing := propertySchema.Listing{
		ID:         uuid.New(),
		PropertyID: property.ID,
		OwnerID:    property.OwnerID,
		OwnerType:  resolveOwnerType(ownerTypes, property.OwnerID),

		Title:       title,
		Description: description,
		Currency:    propertySchema.CurrencyNGN,

		ListingType: listingType,
		Status:      status,
		Published:   published,

		LatestReviewStatus: reviewStatusForListing(status),
		ViewCount:          utils.RandomInt(0, 500),
		BoostLevel:         utils.RandomInt(0, 3),

		HasCalendar: listingType == propertySchema.ListingShortLet,
		CreatedAt:   createdAt,
		UpdatedAt:   time.Now(),
	}

	if published {
		publishedAt := createdAt.Add(time.Duration(utils.RandomInt(1, 7)) * 24 * time.Hour)
		listing.PublishedAt = &publishedAt
	}

	if status != propertySchema.StatusDraft {
		statusChangedAt := createdAt
		listing.StatusChangedAt = &statusChangedAt
	}

	if err := applyListingDetails(&listing, property, listingType); err != nil {
		return err
	}

	if err := ctx.DB.Create(&listing).Error; err != nil {
		return fmt.Errorf("failed to create listing: %w", err)
	}

	return nil
}

func resolveOwnerType(ownerTypes map[uuid.UUID]propertySchema.OwnerType, ownerID uuid.UUID) propertySchema.OwnerType {
	if ownerType, ok := ownerTypes[ownerID]; ok {
		return ownerType
	}
	return propertySchema.OwnerIndividual
}

func chooseListingStatus(
	propertyID uuid.UUID,
	listingType propertySchema.ListingType,
	activeTracker map[string]bool,
) (propertySchema.ListingStatus, bool) {
	roll := utils.RandomInt(1, 100)
	status := propertySchema.StatusActive
	switch {
	case roll <= 70:
		status = propertySchema.StatusActive
	case roll <= 85:
		status = propertySchema.StatusDraft
	case roll <= 95:
		status = propertySchema.StatusUnderReview
	default:
		status = propertySchema.StatusInactive
	}

	published := status == propertySchema.StatusActive
	if status == propertySchema.StatusActive {
		key := propertyID.String() + ":" + string(listingType)
		if activeTracker[key] {
			status = propertySchema.StatusDraft
			published = false
		} else {
			activeTracker[key] = true
		}
	}

	return status, published
}

func reviewStatusForListing(status propertySchema.ListingStatus) propertySchema.ReviewStatus {
	switch status {
	case propertySchema.StatusActive:
		return propertySchema.ReviewApproved
	case propertySchema.StatusUnderReview, propertySchema.StatusPendingVerification:
		return propertySchema.ReviewPending
	case propertySchema.StatusRequiresUpdates, propertySchema.StatusSuspended:
		return propertySchema.ReviewRejected
	default:
		return propertySchema.ReviewPending
	}
}

func applyListingDetails(
	listing *propertySchema.Listing,
	property propertySeed,
	listingType propertySchema.ListingType,
) error {
	tier := locationTier(property.Address, property.City)
	price := randomPrice(tier, listingType)

	switch listingType {
	case propertySchema.ListingShortLet:
		nightlyRate := float64(price)
		cautionFee := nightlyRate * float64(utils.RandomInt(1, 2))
		cleaningFee := float64(utils.RandomInt64(2000, 15000))
		serviceFee := nightlyRate * 0.08

		minNights := utils.RandomInt(1, 3)
		maxNights := utils.RandomInt(14, 90)
		maxGuests := resolveMaxGuests(property.Bedrooms)
		baseGuestCount := maxGuests
		if maxGuests > 2 {
			baseGuestCount = maxGuests - 1
		}

		checkIn := "14:00"
		checkOut := "11:00"

		listing.ShortletDetails = &propertySchema.ShortletDetail{
			NightlyRate:    nightlyRate,
			CautionFee:     &cautionFee,
			CleaningFee:    &cleaningFee,
			ServiceFee:     &serviceFee,
			MinNights:      minNights,
			MaxNights:      &maxNights,
			MaxGuests:      maxGuests,
			BaseGuestCount: &baseGuestCount,
			CheckInTime:    &checkIn,
			CheckOutTime:   &checkOut,
			AccommodationType: utils.RandomChoice([]propertySchema.AccommodationType{
				propertySchema.AccEntirePlace,
				propertySchema.AccSingleRoom,
				propertySchema.AccDoubleRoom,
				propertySchema.AccSharedRoom,
			}),
			AutoAcceptBookings:   utils.RandomBoolWithProbability(0.7),
			CalendarMonthsAhead:  utils.RandomInt(3, 12),
			AutoGenerateCalendar: utils.RandomBoolWithProbability(0.6),
		}
	case propertySchema.ListingRent:
		rentalPeriod := utils.RandomChoice([]propertySchema.PaymentPeriod{
			propertySchema.PayMonthly,
			propertySchema.PayYearly,
		})
		minPeriod := 6
		if rentalPeriod == propertySchema.PayYearly {
			minPeriod = 12
		}

		cautionFee := float64(price) * 0.10
		serviceCharge := float64(price) * 0.05
		availabilityFrom := utils.RandomDateInRange(7, 90)

		listing.RentalDetails = &propertySchema.RentalDetail{
			RentalPrice:            float64(price),
			RentalPricePeriod:      rentalPeriod,
			CautionFee:             &cautionFee,
			ServiceCharge:          &serviceCharge,
			MinRentalPeriod:        minPeriod,
			RentalAvailabilityFrom: &availabilityFrom,
			RentalTerms:            "Upfront payment required with refundable caution fee.",
		}
	case propertySchema.ListingSale:
		ownershipTitles := []string{"C of O", "Governor's Consent", "Deed of Assignment"}
		yearBuilt := utils.RandomInt(1995, 2023)
		yearRenovated := 0
		if utils.RandomBoolWithProbability(0.35) {
			yearRenovated = utils.RandomInt(yearBuilt, 2024)
		}

		agencyFee := float64(price) * 0.02
		legalFee := float64(price) * 0.015
		surveyFee := float64(price) * 0.01

		listing.SaleDetails = &propertySchema.SaleDetail{
			SalePrice:      float64(price),
			OwnershipTitle: utils.RandomChoice(ownershipTitles),
			PaymentPlan:    utils.RandomBoolWithProbability(0.4),
			YearBuilt:      yearBuilt,
			YearRenovated:  yearRenovated,
			AgencyFee:      &agencyFee,
			LegalFee:       &legalFee,
			SurveyFee:      &surveyFee,
			SaleTerms:      "Price negotiable with flexible payment options.",
		}
	default:
		return fmt.Errorf("unknown listing type: %s", listingType)
	}

	return nil
}

func buildListingTitle(property propertySeed) string {
	bedroomLabel := ""
	if property.Bedrooms != nil && *property.Bedrooms > 0 {
		bedroomLabel = fmt.Sprintf("%d Bedroom ", *property.Bedrooms)
	}

	return strings.TrimSpace(fmt.Sprintf(
		"%s%s in %s",
		bedroomLabel,
		titleCase(property.PropertyType),
		property.City,
	))
}

func buildListingDescription(property propertySeed, listingType propertySchema.ListingType) string {
	templates := []string{
		"Spacious {bedrooms}{propertyType} located in {city}, {state}. Close to {address}.",
		"Modern {bedrooms}{propertyType} in {city}. Easy access to {address} and nearby amenities.",
		"Well-finished {bedrooms}{propertyType} in {city}, {state}. Ideal for {listingType} listings.",
		"Comfortable {bedrooms}{propertyType} with quick access to {address} in {city}.",
	}

	bedroomLabel := ""
	if property.Bedrooms != nil && *property.Bedrooms > 0 {
		bedroomLabel = fmt.Sprintf("%d-bedroom ", *property.Bedrooms)
	}

	description := utils.RandomChoice(templates)
	description = strings.ReplaceAll(description, "{bedrooms}", bedroomLabel)
	description = strings.ReplaceAll(description, "{propertyType}", titleCase(property.PropertyType))
	description = strings.ReplaceAll(description, "{city}", property.City)
	description = strings.ReplaceAll(description, "{state}", property.State)
	description = strings.ReplaceAll(description, "{address}", property.Address)
	description = strings.ReplaceAll(description, "{listingType}", string(listingType))

	return strings.ReplaceAll(description, "  ", " ")
}

func resolveMaxGuests(bedrooms *int) int {
	if bedrooms != nil && *bedrooms > 0 {
		return utils.RandomInt(*bedrooms+1, *bedrooms*2)
	}
	return utils.RandomInt(1, 4)
}

func locationTier(address, city string) string {
	for _, loc := range data.NigerianLocations {
		if strings.EqualFold(loc.Address, address) && strings.EqualFold(loc.City, city) {
			return loc.Tier
		}
	}
	return "mid"
}

func randomPrice(tier string, listingType propertySchema.ListingType) int64 {
	rangeByType, ok := data.PriceRanges[tier]
	if !ok {
		rangeByType = data.PriceRanges["mid"]
	}

	priceRange, ok := rangeByType[string(listingType)]
	if !ok {
		priceRange = data.PriceRanges["mid"][string(listingType)]
	}

	return utils.RandomInt64(priceRange[0], priceRange[1])
}

func titleCase(value string) string {
	value = strings.ReplaceAll(value, "_", " ")
	parts := strings.Fields(value)
	for i, part := range parts {
		if len(part) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
