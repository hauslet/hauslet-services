package seeders

import (
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"hauslet/db/seeds/data"
	"hauslet/db/seeds/utils"
	propertySchema "hauslet/internal/modules/property/repository/schema"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type propertySeed struct {
	ID            uuid.UUID
	OwnerID       uuid.UUID
	Address       string
	City          string
	State         string
	PropertyType  string
	PropertyClass string
	Bedrooms      *int
	Bathrooms     *int
}

type listingLifecycle struct {
	Status                     propertySchema.ListingStatus
	Published                  bool
	LatestReviewStatus         propertySchema.ReviewStatus
	ChangeReason               string
	SuspendedUntil             *time.Time
	SuspensionReason           string
	ModerationNotifiedAt       *time.Time
	SuspensionEndingNotifiedAt *time.Time
}

// SeedListing seeds listings linked to existing properties.
func SeedListing(ctx *SeedContext) error {
	properties, err := loadSeedProperties(ctx)
	if err != nil {
		return err
	}

	ownerTypes, err := loadOwnerTypes(ctx)
	if err != nil {
		return err
	}

	listingTypes := buildListingTypes(ctx.Config)
	utils.Shuffle(listingTypes)

	activeTracker := make(map[string]bool)
	typeUsageTracker := make(map[string]int)

	for _, listingType := range listingTypes {
		property, err := pickPropertyForListingType(properties, listingType, typeUsageTracker)
		if err != nil {
			return err
		}
		if err := createListing(ctx, property, listingType, ownerTypes, activeTracker); err != nil {
			return err
		}
	}

	return nil
}

func loadSeedProperties(ctx *SeedContext) ([]propertySeed, error) {
	var properties []propertySeed
	query := ctx.DB.Table("properties").
		Select("id, owner_id, address, city, state, property_type, property_class, bedrooms, bathrooms")

	targetOwnerIDs, err := parseOwnerIDsCSV(ctx.Config.ListingOwnerIDs)
	if err != nil {
		return nil, fmt.Errorf("invalid SEED_LISTING_OWNER_IDS: %w", err)
	}
	if len(targetOwnerIDs) > 0 {
		query = query.Where("owner_id IN ?", targetOwnerIDs)
	}

	if err := query.Scan(&properties).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch properties: %w", err)
	}

	if len(properties) == 0 {
		if len(targetOwnerIDs) > 0 {
			return nil, fmt.Errorf("no properties found for configured SEED_LISTING_OWNER_IDS - seed property module for those owners first")
		}
		return nil, fmt.Errorf("no properties found - seed property module first")
	}

	return properties, nil
}

func pickPropertyForListingType(
	properties []propertySeed,
	listingType propertySchema.ListingType,
	typeUsageTracker map[string]int,
) (propertySeed, error) {
	grouped := make(map[string][]propertySeed)
	for _, property := range properties {
		if listingType == propertySchema.ListingShortLet &&
			strings.EqualFold(property.PropertyClass, string(propertySchema.ClassCommercial)) {
			continue
		}
		grouped[property.PropertyType] = append(grouped[property.PropertyType], property)
	}

	if len(grouped) == 0 {
		return propertySeed{}, fmt.Errorf("no valid properties found for listing type %s", listingType)
	}

	minUsage := -1
	candidateTypes := make([]string, 0, len(grouped))
	for propertyType := range grouped {
		key := fmt.Sprintf("%s:%s", listingType, propertyType)
		usage := typeUsageTracker[key]
		if minUsage == -1 || usage < minUsage {
			minUsage = usage
			candidateTypes = []string{propertyType}
			continue
		}
		if usage == minUsage {
			candidateTypes = append(candidateTypes, propertyType)
		}
	}

	chosenPropertyType := utils.RandomChoice(candidateTypes)
	typeUsageTracker[fmt.Sprintf("%s:%s", listingType, chosenPropertyType)]++
	return utils.RandomChoice(grouped[chosenPropertyType]), nil
}

func buildListingTypes(config *SeedConfig) []propertySchema.ListingType {
	shortletCount := int(float64(config.ListingCount) * config.ShortletRatio)
	rentCount := int(float64(config.ListingCount) * config.RentRatio)
	saleCount := config.ListingCount - shortletCount - rentCount

	listingTypes := make([]propertySchema.ListingType, 0, config.ListingCount)
	for range shortletCount {
		listingTypes = append(listingTypes, propertySchema.ListingShortLet)
	}
	for i := 0; i < rentCount; i++ {
		listingTypes = append(listingTypes, propertySchema.ListingRent)
	}
	for i := 0; i < saleCount; i++ {
		listingTypes = append(listingTypes, propertySchema.ListingSale)
	}

	if len(listingTypes) >= 3 {
		required := []propertySchema.ListingType{
			propertySchema.ListingShortLet,
			propertySchema.ListingRent,
			propertySchema.ListingSale,
		}
		for _, req := range required {
			if !containsListingType(listingTypes, req) {
				listingTypes[utils.RandomInt(0, len(listingTypes)-1)] = req
			}
		}
	}

	return listingTypes
}

func containsListingType(values []propertySchema.ListingType, candidate propertySchema.ListingType) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
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
	title := buildListingTitle(property, listingType)
	description := buildListingDescription(property, listingType)
	extraDescription := buildListingExtraDescription(property, listingType)

	lifecycle := chooseListingLifecycle(property.ID, listingType, activeTracker)
	createdAt := utils.RandomPastDate(240)
	createdBy := property.OwnerID
	updatedBy := property.OwnerID

	listing := propertySchema.Listing{
		ID:         uuid.New(),
		PropertyID: property.ID,
		OwnerID:    property.OwnerID,
		OwnerType:  resolveOwnerType(ownerTypes, property.OwnerID),

		Title:            title,
		Description:      description,
		ExtraDescription: extraDescription,
		Currency:         propertySchema.CurrencyNGN,

		ListingType: listingType,
		Status:      lifecycle.Status,
		Published:   lifecycle.Published,

		LatestReviewStatus:         lifecycle.LatestReviewStatus,
		ModerationNotifiedAt:       lifecycle.ModerationNotifiedAt,
		SuspendedUntil:             lifecycle.SuspendedUntil,
		SuspensionReason:           lifecycle.SuspensionReason,
		SuspensionEndingNotifiedAt: lifecycle.SuspensionEndingNotifiedAt,

		CreatedBy:    &createdBy,
		UpdatedBy:    &updatedBy,
		HasCalendar:  listingType == propertySchema.ListingShortLet,
		ChangeReason: lifecycle.ChangeReason,

		CreatedAt: createdAt,
		UpdatedAt: time.Now(),
	}

	if lifecycle.Published {
		publishedAt := createdAt.Add(time.Duration(utils.RandomInt(1, 21)) * 24 * time.Hour)
		if publishedAt.After(time.Now()) {
			publishedAt = time.Now().Add(-time.Duration(utils.RandomInt(1, 12)) * time.Hour)
		}
		listing.PublishedAt = &publishedAt
	}

	if lifecycle.Status != propertySchema.StatusDraft {
		statusChangedAt := createdAt.Add(time.Duration(utils.RandomInt(1, 35)) * 24 * time.Hour)
		if statusChangedAt.After(time.Now()) {
			statusChangedAt = time.Now().Add(-time.Duration(utils.RandomInt(1, 12)) * time.Hour)
		}
		listing.StatusChangedAt = &statusChangedAt
	}

	applyVerification(&listing, property)

	if err := applyListingDetails(&listing, property, listingType); err != nil {
		return err
	}

	if err := ctx.DB.Create(&listing).Error; err != nil {
		return fmt.Errorf("failed to create listing: %w", err)
	}

	if ctx.Config.SeedMedia {
		if err := seedListingMedia(ctx, listing, property); err != nil {
			return err
		}
	}

	return nil
}

func resolveOwnerType(ownerTypes map[uuid.UUID]propertySchema.OwnerType, ownerID uuid.UUID) propertySchema.OwnerType {
	if ownerType, ok := ownerTypes[ownerID]; ok {
		return ownerType
	}
	return propertySchema.OwnerIndividual
}

func chooseListingLifecycle(
	propertyID uuid.UUID,
	listingType propertySchema.ListingType,
	activeTracker map[string]bool,
) listingLifecycle {
	roll := utils.RandomInt(1, 100)
	status := propertySchema.StatusActive

	switch {
	case roll <= 45:
		status = propertySchema.StatusActive
	case roll <= 58:
		status = propertySchema.StatusDraft
	case roll <= 68:
		status = propertySchema.StatusUnderReview
	case roll <= 76:
		status = propertySchema.StatusPendingVerification
	case roll <= 84:
		status = propertySchema.StatusInactive
	case roll <= 90:
		status = propertySchema.StatusRequiresUpdates
	case roll <= 94:
		status = propertySchema.StatusSuspended
	case roll <= 97:
		status = propertySchema.StatusArchived
	default:
		if listingType == propertySchema.ListingSale {
			status = propertySchema.StatusSold
		} else if listingType == propertySchema.ListingRent {
			status = propertySchema.StatusRented
		} else {
			status = propertySchema.StatusInactive
		}
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

	lifecycle := listingLifecycle{
		Status:             status,
		Published:          published,
		LatestReviewStatus: reviewStatusForListing(status),
		ChangeReason:       statusChangeReason(status),
	}

	if status == propertySchema.StatusSuspended {
		suspendedUntil := time.Now().AddDate(0, 0, utils.RandomInt(7, 45))
		lifecycle.SuspendedUntil = &suspendedUntil
		lifecycle.SuspensionReason = utils.RandomChoice([]string{
			"Pending manual compliance review for listing content.",
			"Temporarily suspended due to policy checks on property media.",
			"Listing paused while ownership verification documents are revalidated.",
		})
	}

	if lifecycle.LatestReviewStatus != propertySchema.ReviewPending && status != propertySchema.StatusDraft {
		notifiedAt := utils.RandomPastDate(45)
		lifecycle.ModerationNotifiedAt = &notifiedAt
	}

	if status == propertySchema.StatusSuspended && lifecycle.SuspendedUntil != nil &&
		lifecycle.SuspendedUntil.Before(time.Now().AddDate(0, 0, 5)) {
		notifiedAt := time.Now().Add(-24 * time.Hour)
		lifecycle.SuspensionEndingNotifiedAt = &notifiedAt
	}

	return lifecycle
}

func reviewStatusForListing(status propertySchema.ListingStatus) propertySchema.ReviewStatus {
	switch status {
	case propertySchema.StatusActive, propertySchema.StatusSold, propertySchema.StatusRented:
		return propertySchema.ReviewApproved
	case propertySchema.StatusUnderReview, propertySchema.StatusPendingVerification:
		return propertySchema.ReviewPending
	case propertySchema.StatusRequiresUpdates, propertySchema.StatusSuspended:
		return propertySchema.ReviewRejected
	default:
		return propertySchema.ReviewInconclusive
	}
}

func statusChangeReason(status propertySchema.ListingStatus) string {
	switch status {
	case propertySchema.StatusActive:
		return "Listing is live and available to guests."
	case propertySchema.StatusDraft:
		return "Listing is still being prepared by the owner."
	case propertySchema.StatusUnderReview:
		return "Listing is pending moderation review."
	case propertySchema.StatusPendingVerification:
		return "Listing requires additional verification checks."
	case propertySchema.StatusInactive:
		return "Listing is temporarily paused by owner or system."
	case propertySchema.StatusRequiresUpdates:
		return "Listing requires updates before approval."
	case propertySchema.StatusSuspended:
		return "Listing was suspended pending compliance review."
	case propertySchema.StatusArchived:
		return "Listing is archived and not currently active."
	case propertySchema.StatusSold:
		return "Property has been marked as sold."
	case propertySchema.StatusRented:
		return "Property has been marked as rented."
	default:
		return "Listing status updated."
	}
}

func applyVerification(listing *propertySchema.Listing, property propertySeed) {
	if listing.Status != propertySchema.StatusActive &&
		listing.Status != propertySchema.StatusSold &&
		listing.Status != propertySchema.StatusRented {
		listing.IsVerified = false
		listing.VerificationLevel = propertySchema.VerificationLevelNone
		return
	}

	verificationChance := 0.35
	if strings.EqualFold(property.PropertyClass, string(propertySchema.ClassCommercial)) {
		verificationChance = 0.55
	}

	if !utils.RandomBoolWithProbability(verificationChance) {
		listing.IsVerified = false
		listing.VerificationLevel = propertySchema.VerificationLevelNone
		return
	}

	listing.IsVerified = true
	if strings.EqualFold(property.PropertyClass, string(propertySchema.ClassCommercial)) {
		listing.VerificationLevel = utils.RandomChoice([]propertySchema.VerificationLevel{
			propertySchema.VerificationLevelPlus,
			propertySchema.VerificationLevelPremium,
		})
	} else {
		listing.VerificationLevel = utils.RandomChoice([]propertySchema.VerificationLevel{
			propertySchema.VerificationLevelBasic,
			propertySchema.VerificationLevelPlus,
			propertySchema.VerificationLevelPremium,
		})
	}

	verifiedAt := listing.CreatedAt.Add(time.Duration(utils.RandomInt(2, 120)) * 24 * time.Hour)
	if verifiedAt.After(time.Now()) {
		verifiedAt = time.Now().Add(-time.Duration(utils.RandomInt(1, 8)) * time.Hour)
	}
	listing.VerifiedAt = &verifiedAt
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
		cleaningFee := float64(utils.RandomInt64(3000, 20000))
		serviceFee := nightlyRate * 0.08
		extraGuestFee := nightlyRate * 0.15

		minNights := utils.RandomInt(1, 3)
		maxNights := utils.RandomInt(14, 120)
		maxGuests := resolveMaxGuests(property.Bedrooms, property.Bathrooms)
		baseGuestCount := maxGuests
		if maxGuests > 2 {
			baseGuestCount = maxGuests - 1
		}

		checkIn := utils.RandomChoice([]string{"13:00", "14:00", "15:00"})
		checkOut := utils.RandomChoice([]string{"10:00", "11:00", "12:00"})

		listing.ShortletDetails = &propertySchema.ShortletDetail{
			NightlyRate: nightlyRate,
			Fees: []propertySchema.CustomFee{
				{
					Name:         "Caution Fee",
					Amount:       cautionFee,
					Frequency:    propertySchema.FeeFreqOneTime,
					IsRefundable: true,
					Category:     propertySchema.FeeCatCaution,
				},
				{
					Name:      "Cleaning Fee",
					Amount:    cleaningFee,
					Frequency: propertySchema.FeeFreqPerStay,
					Category:  propertySchema.FeeCatService,
				},
				{
					Name:      "Service Fee",
					Amount:    serviceFee,
					Frequency: propertySchema.FeeFreqPerNight,
					Category:  propertySchema.FeeCatService,
				},
				{
					Name:      "Extra Guest Fee",
					Amount:    extraGuestFee,
					Frequency: propertySchema.FeeFreqPerExtraGuest,
					Category:  propertySchema.FeeCatService,
				},
			},
			Discounts:       randomShortletDiscounts(),
			BookingSettings: randomBookingSettings(),
			StayLimits: propertySchema.StayLimits{
				MinNights: minNights,
				MaxNights: &maxNights,
			},
			AdvanceBooking: randomAdvanceBooking(),
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
			AutoGenerateCalendar: utils.RandomBoolWithProbability(0.7),
			Rules:                buildShortletRules(),
			AmenitiesHighlights:  buildShortletAmenityHighlights(property),
		}

	case propertySchema.ListingRent:
		rentalPeriod := utils.RandomChoice([]propertySchema.PaymentPeriod{
			propertySchema.PayMonthly,
			propertySchema.PayQuarterly,
			propertySchema.PayYearly,
		})
		minPeriod := 6
		switch rentalPeriod {
		case propertySchema.PayQuarterly:
			minPeriod = 9
		case propertySchema.PayYearly:
			minPeriod = 12
		}

		maxPeriod := minPeriod + utils.RandomChoice([]int{6, 12, 24})
		cautionFee := float64(price) * 0.12
		serviceCharge := float64(price) * 0.05
		legalFee := float64(price) * 0.02
		availabilityFrom := utils.RandomDateInRange(7, 120)
		showingAvailability := buildShowingAvailability()

		listing.RentalDetails = &propertySchema.RentalDetail{
			RentalPrice:       float64(price),
			RentalPricePeriod: rentalPeriod,
			Discounts:         randomRentalDiscounts(),
			Fees: []propertySchema.CustomFee{
				{
					Name:         "Caution Fee",
					Amount:       cautionFee,
					Frequency:    propertySchema.FeeFreqOneTime,
					Category:     propertySchema.FeeCatCaution,
					IsRefundable: true,
				},
				{
					Name:      "Service Charge",
					Amount:    serviceCharge,
					Frequency: propertySchema.FeeFreqPerMonth,
					Category:  propertySchema.FeeCatService,
				},
				{
					Name:      "Legal Documentation Fee",
					Amount:    legalFee,
					Frequency: propertySchema.FeeFreqOneTime,
					Category:  propertySchema.FeeCatLegal,
				},
			},
			MinRentalPeriod:        minPeriod,
			MaxRentalPeriod:        &maxPeriod,
			RentalAvailabilityFrom: &availabilityFrom,
			RentalTerms:            "Payment is made upfront. Caution deposit is refundable subject to property inspection at move-out.",
			RentalRules:            buildRentalRules(),
			ShowingAvailability:    &showingAvailability,
		}

	case propertySchema.ListingSale:
		yearBuilt := utils.RandomInt(1990, 2025)
		yearRenovated := 0
		if utils.RandomBoolWithProbability(0.45) {
			yearRenovated = utils.RandomInt(yearBuilt, 2025)
		}

		agencyFee := float64(price) * 0.025
		legalFee := float64(price) * 0.015
		surveyFee := float64(price) * 0.01
		availabilityFrom := utils.RandomDateInRange(1, 75)
		showingAvailability := buildShowingAvailability()

		listing.SaleDetails = &propertySchema.SaleDetail{
			SalePrice:            float64(price),
			OwnershipTitle:       utils.RandomChoice(data.OwnershipTitles),
			PaymentPlan:          utils.RandomBoolWithProbability(0.42),
			Discounts:            randomSaleDiscounts(),
			YearBuilt:            yearBuilt,
			YearRenovated:        yearRenovated,
			SaleAvailabilityFrom: &availabilityFrom,
			Fees: []propertySchema.CustomFee{
				{
					Name:      "Agency Fee",
					Amount:    agencyFee,
					Frequency: propertySchema.FeeFreqOneTime,
					Category:  propertySchema.FeeCatAgency,
				},
				{
					Name:      "Legal Fee",
					Amount:    legalFee,
					Frequency: propertySchema.FeeFreqOneTime,
					Category:  propertySchema.FeeCatLegal,
				},
				{
					Name:      "Survey Fee",
					Amount:    surveyFee,
					Frequency: propertySchema.FeeFreqOneTime,
					Category:  propertySchema.FeeCatOther,
				},
			},
			SaleTerms:           "Flexible payment options are available after inspection. Legal due diligence required before closing.",
			ShowingAvailability: &showingAvailability,
		}

	default:
		return fmt.Errorf("unknown listing type: %s", listingType)
	}

	return nil
}

func randomShortletDiscounts() []propertySchema.Discount {
	weeklyMinNights := 7
	return []propertySchema.Discount{
		{
			Name:       "Weekly Stay Offer",
			Type:       propertySchema.DiscountTypeLengthOfStay,
			Percentage: float64(utils.RandomInt(5, 12)),
			MinNights:  &weeklyMinNights,
			Active:     utils.RandomBoolWithProbability(0.8),
		},
		{
			Name:       "Direct Booking Promo",
			Type:       propertySchema.DiscountTypeFlat,
			Percentage: float64(utils.RandomInt(3, 8)),
			Active:     utils.RandomBoolWithProbability(0.7),
		},
	}
}

func randomRentalDiscounts() []propertySchema.Discount {
	return []propertySchema.Discount{
		{
			Name:       "Long-term Lease Discount",
			Type:       propertySchema.DiscountTypeFlat,
			Percentage: float64(utils.RandomInt(3, 10)),
			Active:     utils.RandomBoolWithProbability(0.75),
		},
	}
}

func randomSaleDiscounts() []propertySchema.Discount {
	return []propertySchema.Discount{
		{
			Name:       "Direct Purchase Discount",
			Type:       propertySchema.DiscountTypeFlat,
			Percentage: float64(utils.RandomInt(1, 7)),
			Active:     utils.RandomBoolWithProbability(0.6),
		},
	}
}

func buildShortletRules() []propertySchema.RuleGroup {
	return []propertySchema.RuleGroup{
		{
			Category: propertySchema.RuleHouseRules,
			Rules: []propertySchema.RuleItem{
				{
					Name:        propertySchema.RuleSmoking,
					Description: map[string]any{"en": "No indoor smoking is allowed."},
				},
				{
					Name:        propertySchema.RulePets,
					Description: map[string]any{"en": "Pets are allowed only with prior host approval."},
				},
				{
					Name:        propertySchema.RuleEvents,
					Description: map[string]any{"en": "Parties and loud events are not allowed."},
				},
			},
		},
		{
			Category: propertySchema.RuleCheckInOut,
			Rules: []propertySchema.RuleItem{
				{
					Name:        propertySchema.RuleCheckInWindow,
					Description: map[string]any{"en": "Standard check-in window is between 1:00pm and 9:00pm."},
				},
				{
					Name:        propertySchema.RuleCheckOutTime,
					Description: map[string]any{"en": "Check-out is required by 11:00am unless approved otherwise."},
				},
			},
		},
	}
}

func buildRentalRules() []propertySchema.RuleGroup {
	return []propertySchema.RuleGroup{
		{
			Category: propertySchema.RuleGeneral,
			Rules: []propertySchema.RuleItem{
				{
					Name:        propertySchema.RuleGuests,
					Description: map[string]any{"en": "Subletting and unauthorized occupants are not allowed."},
				},
				{
					Name:        propertySchema.RuleSecurity,
					Description: map[string]any{"en": "Valid identification is required during inspection and move-in."},
				},
			},
		},
		{
			Category: propertySchema.RuleCancellationPolicy,
			Rules: []propertySchema.RuleItem{
				{
					Name:        propertySchema.RuleRefundPolicy,
					Description: map[string]any{"en": "Refund terms follow the signed lease and legal policy."},
				},
			},
		},
	}
}

func buildShortletAmenityHighlights(property propertySeed) []propertySchema.AmenityHighlight {
	propertyLabel := titleCase(property.PropertyType)
	return []propertySchema.AmenityHighlight{
		{
			Title:   "Reliable Utilities",
			Summary: "Stable electricity and water setup suitable for daily comfort.",
			Icon:    "bolt",
		},
		{
			Title:   "Great Location",
			Summary: fmt.Sprintf("Located in %s with easy access to key routes and services.", property.City),
			Icon:    "map-pin",
		},
		{
			Title:   "Practical Layout",
			Summary: fmt.Sprintf("The %s layout is optimized for short stays and comfort.", propertyLabel),
			Icon:    "layout",
		},
	}
}

func buildShowingAvailability() []propertySchema.ShowingAvailability {
	return []propertySchema.ShowingAvailability{
		{
			DayOfWeek: "monday",
			StartTime: "09:00",
			EndTime:   "16:00",
			Timezone:  "Africa/Lagos",
		},
		{
			DayOfWeek: "wednesday",
			StartTime: "10:00",
			EndTime:   "17:00",
			Timezone:  "Africa/Lagos",
		},
		{
			DayOfWeek: "saturday",
			StartTime: "11:00",
			EndTime:   "15:00",
			Timezone:  "Africa/Lagos",
		},
	}
}

func buildListingTitle(property propertySeed, listingType propertySchema.ListingType) string {
	bedroomLabel := ""
	if property.Bedrooms != nil && *property.Bedrooms > 0 {
		bedroomLabel = fmt.Sprintf("%d Bedroom ", *property.Bedrooms)
	}

	propertyType := titleCase(property.PropertyType)
	switch listingType {
	case propertySchema.ListingSale:
		return strings.TrimSpace(fmt.Sprintf("%s%s for Sale in %s", bedroomLabel, propertyType, property.City))
	case propertySchema.ListingRent:
		return strings.TrimSpace(fmt.Sprintf("%s%s for Rent in %s", bedroomLabel, propertyType, property.City))
	default:
		return strings.TrimSpace(fmt.Sprintf("%s%s Shortlet in %s", bedroomLabel, propertyType, property.City))
	}
}

func buildListingDescription(property propertySeed, listingType propertySchema.ListingType) string {
	bedroomLabel := "well-planned"
	if property.Bedrooms != nil && *property.Bedrooms > 0 {
		bedroomLabel = fmt.Sprintf("%d-bedroom", *property.Bedrooms)
	}

	typeLabel := titleCase(property.PropertyType)
	base := fmt.Sprintf(
		"This %s %s is located in %s, %s at %s. It offers practical space planning, reliable utility access, and strong neighborhood connectivity for daily living and business convenience.",
		bedroomLabel,
		typeLabel,
		property.City,
		property.State,
		property.Address,
	)

	switch listingType {
	case propertySchema.ListingSale:
		return base + " The sale listing includes complete ownership documentation details, transparent fees, and flexible buyer inspection scheduling."
	case propertySchema.ListingRent:
		return base + " The rental offer supports predictable lease terms, clear recurring charges, and structured viewing availability for prospective tenants."
	default:
		return base + " The shortlet setup includes guest-focused booking controls, stay limits, and curated house rules for a smooth host and guest experience."
	}
}

func buildListingExtraDescription(property propertySeed, listingType propertySchema.ListingType) string {
	switch listingType {
	case propertySchema.ListingSale:
		return fmt.Sprintf(
			"Inspection windows are open weekly in %s. Ownership paperwork and transaction support can be shared on request after buyer pre-qualification.",
			property.City,
		)
	case propertySchema.ListingRent:
		return fmt.Sprintf(
			"The property is positioned close to key transit routes in %s, with easy access to schools, business districts, and everyday retail services.",
			property.City,
		)
	default:
		return fmt.Sprintf(
			"Guests can expect responsive host communication, clear check-in instructions, and nearby access to convenience services around %s.",
			property.City,
		)
	}
}

func resolveMaxGuests(bedrooms *int, bathrooms *int) int {
	if bedrooms != nil && *bedrooms > 0 {
		maxGuests := *bedrooms * 2
		if bathrooms != nil && *bathrooms > *bedrooms {
			maxGuests++
		}
		if maxGuests < 2 {
			maxGuests = 2
		}
		return maxGuests
	}
	return utils.RandomInt(2, 6)
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

func randomBookingSettings() propertySchema.BookingSettings {
	method := propertySchema.ApprovalMethodRequest
	if utils.RandomBoolWithProbability(0.7) {
		method = propertySchema.ApprovalMethodInstant
	}

	return propertySchema.BookingSettings{
		ApprovalMethod:    method,
		GuestRequirements: randomGuestRequirements(),
		PreBookingMessage: utils.RandomChoice([]string{
			"Hosts typically respond quickly to booking requests.",
			"Please complete your profile details before requesting a stay.",
			"Share the purpose of your trip to help the host prepare better.",
		}),
	}
}

func randomGuestRequirements() propertySchema.GuestRequirements {
	return propertySchema.GuestRequirements{
		VerifiedID:           utils.RandomBoolWithProbability(0.6),
		PositiveReviewsOnly:  utils.RandomBoolWithProbability(0.45),
		ProfilePhotoRequired: utils.RandomBoolWithProbability(0.7),
	}
}

func randomAdvanceBooking() propertySchema.AdvanceBooking {
	return propertySchema.AdvanceBooking{
		MonthsAhead:    utils.RandomInt(3, 12),
		MinNoticeHours: utils.RandomChoice([]int{0, 24, 48, 72}),
	}
}

func seedListingMedia(ctx *SeedContext, listing propertySchema.Listing, property propertySeed) error {
	references := mediaReferencesForPropertyType(property.PropertyType)
	mediaCount := utils.RandomInt(4, 8)
	now := time.Now()

	records := make([]propertySchema.ListingMedia, 0, mediaCount)
	for i := 0; i < mediaCount; i++ {
		ref := ""
		if len(references) > 0 {
			ref = references[i%len(references)]
		}

		key := resolveSeedMediaKey(ref, listing.ID, property.PropertyType, i)
		caption := mediaCaptionForOrder(i, property.PropertyType, listing.ListingType)
		group := mediaGroupForOrder(i)
		mimeType := mimeTypeFromMediaReference(ref)
		thumbnails := buildSeedThumbnails(key, mimeType)
		lastModeratedAt := now.Add(-time.Duration(utils.RandomInt(1, 20)) * 24 * time.Hour)

		records = append(records, propertySchema.ListingMedia{
			ID:              uuid.New(),
			ListingID:       listing.ID,
			Key:             key,
			Type:            propertySchema.MediaTypeImage,
			Thumbnails:      thumbnails,
			Group:           ptrString(group),
			Caption:         ptrString(caption),
			MimeType:        ptrString(mimeType),
			SizeBytes:       int64(utils.RandomInt(180_000, 2_400_000)),
			IsPrimary:       i == 0,
			IsGroupCover:    i == 0,
			Order:           i,
			LastModeratedAt: &lastModeratedAt,
			UrlGeneratedAt:  now,
			Uploaded:        true,
			UploadedAt:      now,
			CreatedAt:       now,
			UpdatedAt:       now,
		})
	}

	if err := ctx.DB.CreateInBatches(records, 20).Error; err != nil {
		return fmt.Errorf("failed to create listing media for listing %s: %w", listing.ID, err)
	}

	return nil
}

func mediaReferencesForPropertyType(propertyType string) []string {
	propertyType = strings.TrimSpace(strings.ToLower(propertyType))
	references := data.ListingSeedImageLinksByPropertyType[propertyType]
	if len(references) == 0 {
		references = data.ListingSeedImageLinksByPropertyType["default"]
	}

	clean := make([]string, 0, len(references))
	for _, ref := range references {
		value := strings.TrimSpace(ref)
		if value != "" {
			clean = append(clean, value)
		}
	}
	return clean
}

func resolveSeedMediaKey(reference string, listingID uuid.UUID, propertyType string, index int) string {
	reference = strings.TrimSpace(reference)
	if reference != "" {
		if parsed, err := url.Parse(reference); err == nil && parsed.Scheme != "" && parsed.Host != "" {
			return reference
		}
		return strings.TrimPrefix(reference, "/")
	}

	propertyType = strings.ReplaceAll(strings.ToLower(strings.TrimSpace(propertyType)), " ", "_")
	return fmt.Sprintf("listings/seed/%s/%s/image-%02d.jpg", propertyType, listingID.String(), index+1)
}

func mediaGroupForOrder(order int) string {
	groups := []string{"exterior", "living_area", "bedroom", "kitchen", "bathroom", "amenities"}
	return groups[order%len(groups)]
}

func mediaCaptionForOrder(order int, propertyType string, listingType propertySchema.ListingType) string {
	captions := []string{
		"Front View",
		"Main Living Area",
		"Primary Room",
		"Kitchen Setup",
		"Bathroom Finish",
		"Facility and Amenities",
	}
	base := captions[order%len(captions)]
	return fmt.Sprintf("%s - %s %s", base, titleCase(propertyType), strings.ToUpper(string(listingType)))
}

func mimeTypeFromMediaReference(reference string) string {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return "image/jpeg"
	}

	lookupPath := reference
	if parsed, err := url.Parse(reference); err == nil && parsed.Path != "" {
		lookupPath = parsed.Path
	}

	switch strings.ToLower(path.Ext(lookupPath)) {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return "image/jpeg"
	}
}

func buildSeedThumbnails(key string, mimeType string) propertySchema.ThumbnailMap {
	if strings.HasPrefix(strings.ToLower(key), "http://") || strings.HasPrefix(strings.ToLower(key), "https://") {
		return nil
	}

	ext := path.Ext(key)
	if ext == "" {
		ext = ".jpg"
	}
	base := strings.TrimSuffix(key, ext)

	return propertySchema.ThumbnailMap{
		"small": {
			Key:      base + "_sm" + ext,
			Width:    320,
			Height:   240,
			Size:     int64(utils.RandomInt(40_000, 100_000)),
			MimeType: mimeType,
		},
		"medium": {
			Key:      base + "_md" + ext,
			Width:    720,
			Height:   540,
			Size:     int64(utils.RandomInt(110_000, 280_000)),
			MimeType: mimeType,
		},
	}
}

func ptrString(value string) *string {
	return &value
}
