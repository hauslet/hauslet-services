package graphql

import (
	"context"
	"encoding/base64"
	"strconv"

	businessmiddleware "hauslet/internal/modules/business/middleware"
	"hauslet/internal/modules/property/domain"
	"hauslet/internal/modules/property/service"
	"hauslet/internal/transport/graph/model"

	"github.com/google/uuid"
)

// ===========================
// HELPER FUNCTIONS
// ===========================

func sanitizeListingForViewer(ctx context.Context, l *domain.Listing, userID uuid.UUID) *domain.Listing {
	if l == nil {
		return nil
	}

	// Unpublished listings (drafts, under review, etc.) satisfy this check
	if !l.Published {
		if userID == l.OwnerID {
			return l
		}

		// If listing is owned by a business, allow members/owners from the business context (set via X-Tenant-Slug).
		if l.OwnerType == domain.OwnerBusiness {
			if bc, ok := businessmiddleware.GetBusinessContext(ctx); ok && bc.BusinessID == l.OwnerID && bc.Membership != nil {
				return l
			}
		}

		// Otherwise unpublished listings stay hidden
		return nil
	}

	// Owners see everything
	if userID == l.OwnerID {
		return l
	}

	// Public view - hide internal fields
	clone := *l
	clone.CreatedBy = nil
	clone.UpdatedBy = nil
	clone.ChangeReason = ""
	return &clone
}

func sanitizePropertyForViewer(ctx context.Context, p *domain.Property, userID uuid.UUID) *domain.Property {
	if p == nil {
		return nil
	}
	// Hide address fields for non-owners while letting owners or their business members see full details.
	if userID == p.OwnerID {
		return p
	}
	if bc, ok := businessmiddleware.GetBusinessContext(ctx); ok && bc.BusinessID == p.OwnerID && bc.Membership != nil {
		return p
	}

	clone := *p
	clone.UnitNumber = ""
	clone.Address = ""
	clone.City = ""
	clone.State = ""
	clone.PostalCode = ""
	return &clone
}

func buildListingConnection(listings []domain.Listing, total int64, offset int, ctx context.Context, userID uuid.UUID) *model.ListingConnection {
	edges := make([]*model.ListingEdge, 0, len(listings))
	for i, listing := range listings {
		l := listing
		sanitized := sanitizeListingForViewer(ctx, &l, userID)
		if sanitized != nil {
			edges = append(edges, &model.ListingEdge{
				Node:   sanitized,
				Cursor: encodeCursor(offset + i),
			})
		}
	}

	var startCursor, endCursor *string
	if len(edges) > 0 {
		startCursor = &edges[0].Cursor
		endCursor = &edges[len(edges)-1].Cursor
	}

	hasNext := int64(offset+len(listings)) < total
	hasPrev := offset > 0

	return &model.ListingConnection{
		Edges: edges,
		PageInfo: &model.PageInfo{
			HasNextPage:     hasNext,
			HasPreviousPage: hasPrev,
			StartCursor:     startCursor,
			EndCursor:       endCursor,
		},
		TotalCount: int(total),
	}
}

func encodeCursor(offset int) string {
	return base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

func decodeCursor(cursor string) (int, error) {
	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(decoded))
}

func mapListingFilterToService(filter *model.ListingFilterInput) service.ListingFilter {
	if filter == nil {
		return service.ListingFilter{}
	}

	serviceFilter := service.ListingFilter{
		Query:        filter.Query,
		OwnerID:      filter.OwnerID,
		PropertyID:   filter.PropertyID,
		HasCalendar:  filter.HasCalendar,
		City:         filter.City,
		State:        filter.State,
		Country:      filter.Country,
		Latitude:     filter.Latitude,
		Longitude:    filter.Longitude,
		RadiusMeters: filter.RadiusMeters,
		MinPrice:     filter.MinPrice,
		MaxPrice:     filter.MaxPrice,
		Currency:     filter.Currency,
	}

	if len(filter.OwnerTypes) > 0 {
		serviceFilter.OwnerTypes = filter.OwnerTypes
	}

	if len(filter.ListingTypes) > 0 {
		serviceFilter.ListingTypes = filter.ListingTypes
	}

	if len(filter.Statuses) > 0 {
		serviceFilter.Statuses = filter.Statuses
	}

	if len(filter.ReviewStatuses) > 0 {
		serviceFilter.ReviewStatuses = filter.ReviewStatuses
	}

	if len(filter.PropertyTypes) > 0 {
		serviceFilter.PropertyTypes = filter.PropertyTypes
	}
	if len(filter.FurnishingTypes) > 0 {
		serviceFilter.Furnishings = filter.FurnishingTypes
	}
	serviceFilter.MinBedrooms = filter.MinBedrooms
	serviceFilter.MaxBedrooms = filter.MaxBedrooms
	serviceFilter.MinBathrooms = filter.MinBathrooms
	serviceFilter.MaxBathrooms = filter.MaxBathrooms

	// Map type-specific filters
	if filter.ShortletFilter != nil {
		serviceFilter.ShortletFilter = mapGraphQLShortletFilter(filter.ShortletFilter)
	}
	if filter.RentalFilter != nil {
		serviceFilter.RentalFilter = mapGraphQLRentalFilter(filter.RentalFilter)
	}
	if filter.SaleFilter != nil {
		serviceFilter.SaleFilter = mapGraphQLSaleFilter(filter.SaleFilter)
	}
	if filter.PropertyExtension != nil {
		serviceFilter.PropertyExtension = mapGraphQLPropertyExtension(filter.PropertyExtension)
	}

	return serviceFilter
}

// mapGraphQLShortletFilter maps GraphQL shortlet filter to service shortlet filter
func mapGraphQLShortletFilter(filter *model.ShortletFilterInput) *service.ShortletFilter {
	if filter == nil {
		return nil
	}

	serviceFilter := &service.ShortletFilter{
		MinExtraGuestFee:   filter.MinExtraGuestFee,
		MaxExtraGuestFee:   filter.MaxExtraGuestFee,
		MinNightsMin:       filter.MinNightsMin,
		MinNightsMax:       filter.MinNightsMax,
		MaxNightsMin:       filter.MaxNightsMin,
		MaxNightsMax:       filter.MaxNightsMax,
		MinMaxGuests:       filter.MinMaxGuests,
		BaseGuestCount:     filter.BaseGuestCount,
		CheckInTimeAfter:   filter.CheckInTimeAfter,
		CheckInTimeBefore:  filter.CheckInTimeBefore,
		CheckOutTimeAfter:  filter.CheckOutTimeAfter,
		CheckOutTimeBefore: filter.CheckOutTimeBefore,
	}

	if len(filter.AccommodationTypes) > 0 {
		serviceFilter.AccommodationTypes = filter.AccommodationTypes
	}

	return serviceFilter
}

// mapGraphQLRentalFilter maps GraphQL rental filter to service rental filter
func mapGraphQLRentalFilter(filter *model.RentalFilterInput) *service.RentalFilter {
	if filter == nil {
		return nil
	}

	serviceFilter := &service.RentalFilter{
		MinRentalPeriodMin: filter.MinRentalPeriodMin,
		MinRentalPeriodMax: filter.MinRentalPeriodMax,
		MaxRentalPeriodMin: filter.MaxRentalPeriodMin,
		MaxRentalPeriodMax: filter.MaxRentalPeriodMax,
		AvailableFrom:      filter.AvailableFrom,
		AvailableTo:        filter.AvailableTo,
	}

	if len(filter.RentalPricePeriods) > 0 {
		serviceFilter.RentalPricePeriods = filter.RentalPricePeriods
	}

	return serviceFilter
}

// mapGraphQLSaleFilter maps GraphQL sale filter to service sale filter
func mapGraphQLSaleFilter(filter *model.SaleFilterInput) *service.SaleFilter {
	if filter == nil {
		return nil
	}

	return &service.SaleFilter{
		OwnershipTitles: filter.OwnershipTitles,
		PaymentPlan:     filter.PaymentPlan,
	}
}

// mapGraphQLPropertyExtension maps GraphQL property extension to service property extension
func mapGraphQLPropertyExtension(filter *model.PropertyFilterExtension) *service.PropertyFilterExtension {
	if filter == nil {
		return nil
	}

	serviceFilter := &service.PropertyFilterExtension{
		Amenities: filter.Amenities,
	}

	if len(filter.PropertyClasses) > 0 {
		serviceFilter.PropertyClasses = filter.PropertyClasses
	}

	if len(filter.PropertyConditions) > 0 {
		serviceFilter.PropertyConditions = filter.PropertyConditions
	}

	return serviceFilter
}

func stringOrDefault(val *string, def string) string {
	if val == nil {
		return def
	}
	return *val
}

func intOrDefault(val *int, def int) int {
	if val == nil {
		return def
	}
	return *val
}

func floatOrDefault(val *float64, def float64) float64 {
	if val == nil {
		return def
	}
	return *val
}

func furnishingTypeOrDefault(val *domain.FurnishingType) domain.FurnishingType {
	if val == nil {
		return domain.Furnished
	}
	return *val
}

func propertyConditionOrDefault(val *domain.PropertyCondition) domain.PropertyCondition {
	if val == nil {
		return domain.ConditionUsed
	}
	return *val
}

func defaultCurrency(val *domain.CurrencyCode) domain.CurrencyCode {
	if val == nil {
		return domain.CurrencyNGN
	}
	return *val
}

func boolOrDefault(val *bool, def bool) bool {
	if val == nil {
		return def
	}
	return *val
}

func mapShortletInputToDomain(input *model.ShortletDetailInput) *domain.ShortletDetail {
	if input == nil {
		return nil
	}

	return &domain.ShortletDetail{
		NightlyRate: input.NightlyRate,
		Fees:        mapCustomFeeInputs(input.Fees),
		Discounts:   mapDiscountInputs(input.Discounts),
		BookingSettings: domain.BookingSettings{
			ApprovalMethod:    input.BookingSettings.ApprovalMethod,
			GuestRequirements: mapGuestRequirementsInput(input.BookingSettings.GuestRequirements),
			PreBookingMessage: stringOrDefault(input.BookingSettings.PreBookingMessage, ""),
		},
		StayLimits: domain.StayLimits{
			MinNights: input.StayLimits.MinNights,
			MaxNights: input.StayLimits.MaxNights,
		},
		AdvanceBooking:       mapAdvanceBookingInput(input.AdvanceBooking),
		MaxGuests:            input.MaxGuests,
		BaseGuestCount:       input.BaseGuestCount,
		CheckInTime:          input.CheckInTime,
		CheckOutTime:         input.CheckOutTime,
		AccommodationType:    domain.AccommodationType(input.AccommodationType),
		AutoGenerateCalendar: boolOrDefault(input.AutoGenerateCalendar, false),
		Rules:                mapRuleGroupInputs(input.Rules),
		AmenitiesHighlights:  mapAmenityHighlightInputs(input.AmenitiesHighlights),
	}
}

func mapRentalInputToDomain(input *model.RentalDetailInput) *domain.RentalDetail {
	if input == nil {
		return nil
	}

	return &domain.RentalDetail{
		RentalPrice:            input.RentalPrice,
		RentalPricePeriod:      domain.PaymentPeriod(input.RentalPricePeriod),
		Discounts:              mapDiscountInputs(input.Discounts),
		Fees:                   mapCustomFeeInputs(input.Fees),
		MinRentalPeriod:        input.MinRentalPeriod,
		MaxRentalPeriod:        input.MaxRentalPeriod,
		RentalAvailabilityFrom: input.RentalAvailabilityFrom,
		RentalTerms:            stringOrDefault(input.RentalTerms, ""),
		RentalRules:            mapRuleGroupInputs(input.RentalRules),
		ShowingAvailability:    mapShowingAvailabilityInputs(input.ShowingAvailability),
	}
}

func mapSaleInputToDomain(input *model.SaleDetailInput) *domain.SaleDetail {
	if input == nil {
		return nil
	}

	return &domain.SaleDetail{
		SalePrice:            input.SalePrice,
		OwnershipTitle:       input.OwnershipTitle,
		PaymentPlan:          boolOrDefault(input.PaymentPlan, false),
		Discounts:            mapDiscountInputs(input.Discounts),
		YearBuilt:            intOrDefault(input.YearBuilt, 0),
		YearRenovated:        intOrDefault(input.YearRenovated, 0),
		Fees:                 mapCustomFeeInputs(input.Fees),
		SaleTerms:            stringOrDefault(input.SaleTerms, ""),
		SaleAvailabilityFrom: input.SaleAvailabilityFrom,
		ShowingAvailability:  mapShowingAvailabilityInputs(input.ShowingAvailability),
	}
}

// Update input mappers for partial updates
func mapUpdateShortletInputToDomain(input *model.UpdateShortletDetailInput) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	updates := map[string]any{}
	if input.NightlyRate != nil {
		updates["nightly_rate"] = *input.NightlyRate
	}
	if input.Fees != nil {
		updates["fees"] = mapCustomFeeInputs(input.Fees)
	}
	if input.Discounts != nil {
		updates["discounts"] = mapDiscountInputs(input.Discounts)
	}

	// Nest StayLimits
	if input.StayLimits != nil {
		updates["stay_limits"] = domain.StayLimits{
			MinNights: input.StayLimits.MinNights,
			MaxNights: input.StayLimits.MaxNights,
		}
	}

	// Nest BookingSettings
	if input.BookingSettings != nil {
		updates["booking_settings"] = domain.BookingSettings{
			ApprovalMethod:    input.BookingSettings.ApprovalMethod,
			GuestRequirements: mapGuestRequirementsInput(input.BookingSettings.GuestRequirements),
			PreBookingMessage: stringOrDefault(input.BookingSettings.PreBookingMessage, ""),
		}
	}

	// Nest AdvanceBooking
	if input.AdvanceBooking != nil {
		updates["advance_booking"] = mapAdvanceBookingInput(input.AdvanceBooking)
	}

	if input.MaxGuests != nil {
		updates["max_guests"] = *input.MaxGuests
	}
	if input.BaseGuestCount != nil {
		updates["base_guest_count"] = *input.BaseGuestCount
	}
	if input.CheckInTime != nil {
		updates["check_in_time"] = *input.CheckInTime
	}
	if input.CheckOutTime != nil {
		updates["check_out_time"] = *input.CheckOutTime
	}
	if input.AccommodationType != nil {
		updates["accommodation_type"] = *input.AccommodationType
	}
	if input.AutoGenerateCalendar != nil {
		updates["auto_generate_calendar"] = *input.AutoGenerateCalendar
	}
	if input.Rules != nil {
		updates["rules"] = mapRuleGroupInputs(input.Rules)
	}
	if input.AmenitiesHighlights != nil {
		updates["amenities_highlights"] = mapAmenityHighlightInputs(input.AmenitiesHighlights)
	}
	return updates
}

func mapUpdateRentalInputToDomain(input *model.UpdateRentalDetailInput) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	updates := map[string]any{}
	if input.RentalPrice != nil {
		updates["rental_price"] = *input.RentalPrice
	}
	if input.RentalPricePeriod != nil {
		updates["rental_price_period"] = *input.RentalPricePeriod
	}
	if input.Fees != nil {
		updates["fees"] = mapCustomFeeInputs(input.Fees)
	}
	if input.MinRentalPeriod != nil {
		updates["min_rental_period"] = *input.MinRentalPeriod
	}
	if input.MaxRentalPeriod != nil {
		updates["max_rental_period"] = *input.MaxRentalPeriod
	}
	if input.RentalAvailabilityFrom != nil {
		updates["rental_availability_from"] = *input.RentalAvailabilityFrom
	}
	if input.RentalTerms != nil {
		updates["rental_terms"] = *input.RentalTerms
	}
	if input.RentalRules != nil {
		updates["rental_rules"] = mapRuleGroupInputs(input.RentalRules)
	}
	if input.Discounts != nil {
		updates["discounts"] = mapDiscountInputs(input.Discounts)
	}
	if input.ShowingAvailability != nil {
		updates["showing_availability"] = mapShowingAvailabilityInputs(input.ShowingAvailability)
	}
	return updates
}

func mapUpdateSaleInputToDomain(input *model.UpdateSaleDetailInput) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	updates := map[string]any{}
	if input.SalePrice != nil {
		updates["sale_price"] = *input.SalePrice
	}
	if input.OwnershipTitle != nil {
		updates["ownership_title"] = *input.OwnershipTitle
	}
	if input.PaymentPlan != nil {
		updates["payment_plan"] = *input.PaymentPlan
	}
	if input.YearBuilt != nil {
		updates["year_built"] = *input.YearBuilt
	}
	if input.YearRenovated != nil {
		updates["year_renovated"] = *input.YearRenovated
	}
	if input.Fees != nil {
		updates["fees"] = mapCustomFeeInputs(input.Fees)
	}
	if input.SaleTerms != nil {
		updates["sale_terms"] = *input.SaleTerms
	}
	if input.SaleAvailabilityFrom != nil {
		updates["sale_availability_from"] = *input.SaleAvailabilityFrom
	}
	if input.Discounts != nil {
		updates["discounts"] = mapDiscountInputs(input.Discounts)
	}
	if input.ShowingAvailability != nil {
		updates["showing_availability"] = mapShowingAvailabilityInputs(input.ShowingAvailability)
	}
	return updates
}

func mapRuleGroupInputs(inputs []*domain.RuleGroup) []domain.RuleGroup {
	if len(inputs) == 0 {
		return []domain.RuleGroup{}
	}
	rules := make([]domain.RuleGroup, len(inputs))
	for i, input := range inputs {
		if input == nil {
			continue
		}
		group := *input
		group.Rules = mapRuleItems(group.Rules)
		rules[i] = group
	}
	return rules
}

func mapRuleItems(inputs []domain.RuleItem) []domain.RuleItem {
	if len(inputs) == 0 {
		return []domain.RuleItem{}
	}
	items := make([]domain.RuleItem, len(inputs))
	for i, input := range inputs {
		if input.Description == nil {
			input.Description = map[string]any{}
		}
		items[i] = input
	}
	return items
}

func mapAmenityHighlightInputs(inputs []*model.AmenityHighlightInput) []domain.AmenityHighlight {
	if len(inputs) == 0 {
		return []domain.AmenityHighlight{}
	}
	highlights := make([]domain.AmenityHighlight, len(inputs))
	for i, input := range inputs {
		if input == nil {
			continue
		}
		highlights[i] = domain.AmenityHighlight{
			Title:   input.Title,
			Summary: input.Summary,
			Icon:    input.Icon,
		}
	}
	return highlights
}

// mapAmenityGroupInputsToDomain converts AmenityGroupInput slice to domain AmenityGroup slice
func mapAmenityGroupInputsToDomain(inputs []*domain.AmenityGroup) []domain.AmenityGroup {
	if len(inputs) == 0 {
		return []domain.AmenityGroup{}
	}
	groups := make([]domain.AmenityGroup, len(inputs))
	for i, input := range inputs {
		if input == nil {
			continue
		}
		groups[i] = *input
	}
	return groups
}

// Helper for Fees
func mapCustomFeeInputs(inputs []*model.CustomFeeInput) []domain.CustomFee {
	if len(inputs) == 0 {
		return []domain.CustomFee{}
	}
	fees := make([]domain.CustomFee, len(inputs))
	for i, input := range inputs {
		if input == nil {
			continue
		}
		fees[i] = domain.CustomFee{
			Name:         input.Name,
			Amount:       input.Amount,
			Frequency:    input.Frequency,
			Category:     input.Category,
			IsRefundable: boolOrDefault(input.IsRefundable, false),
			IsOptional:   boolOrDefault(input.IsOptional, false),
		}
	}
	return fees
}

// Helper for Discounts
func mapDiscountInputs(inputs []*model.DiscountInput) []domain.Discount {
	if len(inputs) == 0 {
		return []domain.Discount{}
	}
	discounts := make([]domain.Discount, len(inputs))
	for i, input := range inputs {
		if input == nil {
			continue
		}
		discounts[i] = domain.Discount{
			Name:       input.Name,
			Type:       domain.DiscountType(input.Type),
			Percentage: input.Percentage,
			MinNights:  input.MinNights,
			Active:     input.Active,
		}
	}
	return discounts
}

// Helper for Showing Availability
func mapShowingAvailabilityInputs(inputs []*model.ShowingAvailabilityInput) []domain.ShowingAvailability {
	if len(inputs) == 0 {
		return []domain.ShowingAvailability{}
	}
	availability := make([]domain.ShowingAvailability, len(inputs))
	for i, input := range inputs {
		if input == nil {
			continue
		}
		availability[i] = domain.ShowingAvailability{
			DayOfWeek: input.DayOfWeek,
			StartTime: input.StartTime,
			EndTime:   input.EndTime,
			Timezone:  input.Timezone,
		}
	}
	return availability
}

func mapGuestRequirementsInput(input *model.GuestRequirementsInput) domain.GuestRequirements {
	if input == nil {
		return domain.GuestRequirements{}
	}
	return domain.GuestRequirements{
		VerifiedID:           input.VerifiedID,
		PositiveReviewsOnly:  input.PositiveReviewsOnly,
		ProfilePhotoRequired: input.ProfilePhotoRequired,
	}
}

func mapAdvanceBookingInput(input *model.AdvanceBookingInput) domain.AdvanceBooking {
	if input == nil {
		// Provide reasonable defaults if mandatory input missing (though schema should enforce)
		return domain.AdvanceBooking{MonthsAhead: 6, MinNoticeHours: 24}
	}
	return domain.AdvanceBooking{
		MonthsAhead:    input.MonthsAhead,
		MinNoticeHours: input.MinNoticeHours,
	}
}

// mapCreateListingInput converts the CreateListingInput into a domain listing with sane defaults.
func mapCreateListingInput(input model.CreateListingInput, ownerID uuid.UUID) domain.Listing {
	hasCalendar := false
	if input.HasCalendar != nil {
		hasCalendar = *input.HasCalendar
	} else if input.ListingType == domain.ListingShortLet || input.ShortletDetails != nil {
		hasCalendar = true
	}

	listing := domain.Listing{
		OwnerID:            ownerID,
		OwnerType:          input.OwnerType,
		Title:              input.Title,
		Description:        input.Description,
		ExtraDescription:   stringOrDefault(input.ExtraDescription, ""),
		Currency:           defaultCurrency(input.Currency),
		ListingType:        input.ListingType,
		Status:             domain.StatusDraft,
		Published:          false,
		LatestReviewStatus: domain.ReviewPending,
		HasCalendar:        hasCalendar,
	}

	if input.ShortletDetails != nil {
		listing.ShortletDetails = mapShortletInputToDomain(input.ShortletDetails)
	}
	if input.RentalDetails != nil {
		listing.RentalDetails = mapRentalInputToDomain(input.RentalDetails)
	}
	if input.SaleDetails != nil {
		listing.SaleDetails = mapSaleInputToDomain(input.SaleDetails)
	}

	return listing
}

// mapListingUpdateInput converts UpdateListingInput into repo-friendly update map.
func mapListingUpdateInput(input *model.UpdateListingInput) map[string]any {
	if input == nil {
		return map[string]any{}
	}

	updates := map[string]any{}
	if input.Title != nil {
		updates["title"] = *input.Title
	}
	if input.Description != nil {
		updates["description"] = *input.Description
	}
	if input.ExtraDescription != nil {
		updates["extra_description"] = *input.ExtraDescription
	}
	if input.Currency != nil {
		updates["currency"] = *input.Currency
	}
	if input.OwnerType != nil {
		updates["owner_type"] = *input.OwnerType
	}
	if input.HasCalendar != nil {
		updates["has_calendar"] = *input.HasCalendar
	}
	if input.ShortletDetails != nil {
		updates["shortlet_details"] = mapUpdateShortletInputToDomain(input.ShortletDetails)
	}
	if input.RentalDetails != nil {
		updates["rental_details"] = mapUpdateRentalInputToDomain(input.RentalDetails)
	}
	if input.SaleDetails != nil {
		updates["sale_details"] = mapUpdateSaleInputToDomain(input.SaleDetails)
	}
	if _, ok := updates["has_calendar"]; !ok && input.ShortletDetails != nil {
		updates["has_calendar"] = true
	}
	return updates
}

// mapCreateListingPropertyInput builds a Property domain model from the listing property input.
func mapCreateListingPropertyInput(input *model.CreateListingPropertyInput, ownerID uuid.UUID) *domain.Property {
	if input == nil {
		return nil
	}
	prop := &domain.Property{
		UnitNumber:        stringOrDefault(input.UnitNumber, ""),
		Address:           input.Address,
		City:              input.City,
		State:             input.State,
		PostalCode:        stringOrDefault(input.PostalCode, ""),
		Country:           input.Country,
		PropertyClass:     input.PropertyClass,
		PropertyType:      input.PropertyType,
		FurnishingType:    furnishingTypeOrDefault(input.FurnishingType),
		PropertyCondition: propertyConditionOrDefault(input.PropertyCondition),
		OwnerID:           ownerID,
		Bedrooms:          input.Bedrooms,
		Bathrooms:         input.Bathrooms,
		Toilets:           input.Toilets,
		HalfBathrooms:     input.HalfBathrooms,
		Floors:            input.Floors,
		Units:             intOrDefault(input.Units, 1),
		SquareMeters:      floatOrDefault(input.SquareMeters, 0),
		FloorArea:         input.FloorArea,
	}
	if input.Location != nil {
		prop.Location = &domain.Location{
			Lat:  input.Location.Lat,
			Lng:  input.Location.Lng,
			SRID: 4326,
		}
	}

	// Convert AmenityGroupInput to domain AmenityGroup
	if len(input.Amenities) > 0 {
		prop.Amenities = mapAmenityGroupInputsToDomain(input.Amenities)
	}
	if len(input.FeaturesCommercial) > 0 {
		prop.FeaturesCommercial = mapAmenityGroupInputsToDomain(input.FeaturesCommercial)
	}

	return prop
}

// mapUpdateListingPropertyInput builds partial updates for property from the listing update payload.
func mapUpdateListingPropertyInput(input *model.UpdateListingPropertyInput) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	updates := map[string]any{}
	if input.UnitNumber != nil {
		updates["unit_number"] = *input.UnitNumber
	}
	if input.Address != nil {
		updates["address"] = *input.Address
	}
	if input.City != nil {
		updates["city"] = *input.City
	}
	if input.State != nil {
		updates["state"] = *input.State
	}
	if input.PostalCode != nil {
		updates["postal_code"] = *input.PostalCode
	}
	if input.Country != nil {
		updates["country"] = *input.Country
	}
	if input.PropertyClass != nil {
		updates["property_class"] = *input.PropertyClass
	}
	if input.PropertyType != nil {
		updates["property_type"] = *input.PropertyType
	}
	if input.FurnishingType != nil {
		updates["furnishing_type"] = *input.FurnishingType
	}
	if input.PropertyCondition != nil {
		updates["property_condition"] = *input.PropertyCondition
	}
	if input.Bedrooms != nil {
		updates["bedrooms"] = *input.Bedrooms
	}
	if input.Bathrooms != nil {
		updates["bathrooms"] = *input.Bathrooms
	}
	if input.Toilets != nil {
		updates["toilets"] = *input.Toilets
	}
	if input.HalfBathrooms != nil {
		updates["half_bathrooms"] = *input.HalfBathrooms
	}
	if input.Floors != nil {
		updates["floors"] = *input.Floors
	}
	if input.Units != nil {
		updates["units"] = *input.Units
	}
	if input.SquareMeters != nil {
		updates["square_meters"] = *input.SquareMeters
	}
	if input.FloorArea != nil {
		updates["floor_area"] = *input.FloorArea
	}
	if input.Amenities != nil {
		updates["amenities"] = mapAmenityGroupInputsToDomain(input.Amenities)
	}
	if input.FeaturesCommercial != nil {
		updates["features_commercial"] = mapAmenityGroupInputsToDomain(input.FeaturesCommercial)
	}
	if input.Location != nil {
		updates["location"] = &domain.Location{
			Lat:  input.Location.Lat,
			Lng:  input.Location.Lng,
			SRID: 4326,
		}
	}
	return updates
}
