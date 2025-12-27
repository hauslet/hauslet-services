package graphql

import (
	"context"
	"encoding/base64"
	"strconv"

	businessmiddleware "hauslet/internal/modules/business/middleware"
	"hauslet/internal/modules/property/domain"
	"hauslet/internal/modules/property/service"
	"hauslet/internal/transport/graph/model"
	"hauslet/internal/transport/graph/viewer"

	"github.com/google/uuid"
)

// ===========================
// HELPER FUNCTIONS
// ===========================

func isAdminRole(role string) bool {
	return role == "admin" || role == "root"
}

func sanitizePropertyForViewer(p *domain.Property, v *viewer.Viewer) *domain.Property {
	if p == nil {
		return nil
	}
	// Admins and owners see full property
	if v != nil && (v.UserID == p.OwnerID.String() || isAdminRole(v.Role)) {
		return p
	}
	// Public view - no sensitive data to hide for properties
	return p
}

func sanitizeListingForViewer(ctx context.Context, l *domain.Listing, v *viewer.Viewer) *domain.Listing {
	if l == nil {
		return nil
	}

	// Draft listings only visible to owner/admin/business members
	if l.Status == domain.StatusDraft {
		if v != nil && (v.UserID == l.OwnerID.String() || isAdminRole(v.Role)) {
			return l
		}

		// If listing is owned by a business, allow members/owners from the business context (set via X-Tenant-Slug).
		if l.OwnerType == domain.OwnerBusiness {
			if bc, ok := businessmiddleware.GetBusinessContext(ctx); ok && bc.BusinessID == l.OwnerID && bc.Membership != nil {
				return l
			}
		}

		// Otherwise drafts stay hidden
		return nil
	}

	// Admins and owners see everything
	if v != nil && (v.UserID == l.OwnerID.String() || isAdminRole(v.Role)) {
		return l
	}

	// Public view - hide internal fields
	clone := *l
	clone.CreatedBy = nil
	clone.UpdatedBy = nil
	clone.ChangeReason = ""
	return &clone
}

func buildListingConnection(listings []domain.Listing, total int64, offset int, limit int, ctx context.Context, v *viewer.Viewer) *model.ListingConnection {
	edges := make([]*model.ListingEdge, 0, len(listings))
	for i, listing := range listings {
		l := listing
		sanitized := sanitizeListingForViewer(ctx, &l, v)
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
		Query:      filter.Query,
		OwnerID:    filter.OwnerID,
		PropertyID: filter.PropertyID,
		// NOTE: Published field removed from GraphQL for security - service layer will enforce published=true
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
		MinViewCount: filter.MinViewCount,
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
		NightlyRate:          input.NightlyRate,
		CautionFee:           input.CautionFee,
		CleaningFee:          input.CleaningFee,
		ServiceFee:           input.ServiceFee,
		ExtraGuestFee:        input.ExtraGuestFee,
		MinNights:            input.MinNights,
		MaxNights:            input.MaxNights,
		MaxGuests:            input.MaxGuests,
		BaseGuestCount:       input.BaseGuestCount,
		CheckInTime:          input.CheckInTime,
		CheckOutTime:         input.CheckOutTime,
		AccommodationType:    domain.AccommodationType(input.AccommodationType),
		AutoAcceptBookings:   boolOrDefault(input.AutoAcceptBookings, true),
		CalendarMonthsAhead:  intOrDefault(input.CalendarMonthsAhead, 0),
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
		AgencyFee:              input.AgencyFee,
		LegalFee:               input.LegalFee,
		RegistrationFee:        input.RegistrationFee,
		CautionFee:             input.CautionFee,
		ServiceCharge:          input.ServiceCharge,
		ServiceChargeBreakdown: mapServiceChargeInputs(input.ServiceCharges),
		MinRentalPeriod:        input.MinRentalPeriod,
		MaxRentalPeriod:        input.MaxRentalPeriod,
		RentalAvailabilityFrom: input.RentalAvailabilityFrom,
		RentalTerms:            stringOrDefault(input.RentalTerms, ""),
		RentalRules:            mapRuleGroupInputs(input.RentalRules),
	}
}

func mapSaleInputToDomain(input *model.SaleDetailInput) *domain.SaleDetail {
	if input == nil {
		return nil
	}
	return &domain.SaleDetail{
		SalePrice:              input.SalePrice,
		OwnershipTitle:         input.OwnershipTitle,
		PaymentPlan:            boolOrDefault(input.PaymentPlan, false),
		YearBuilt:              intOrDefault(input.YearBuilt, 0),
		YearRenovated:          intOrDefault(input.YearRenovated, 0),
		AgencyFee:              input.AgencyFee,
		LegalFee:               input.LegalFee,
		SurveyFee:              input.SurveyFee,
		TitleProcessingFee:     input.TitleProcessingFee,
		DevelopmentFee:         input.DevelopmentFee,
		OtherFees:              input.OtherFees,
		ServiceCharge:          input.ServiceCharge,
		ServiceChargeBreakdown: mapServiceChargeInputs(input.ServiceCharges),
		SaleTerms:              stringOrDefault(input.SaleTerms, ""),
		SaleAvailabilityFrom:   input.SaleAvailabilityFrom,
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
	if input.CautionFee != nil {
		updates["caution_fee"] = *input.CautionFee
	}
	if input.CleaningFee != nil {
		updates["cleaning_fee"] = *input.CleaningFee
	}
	if input.ServiceFee != nil {
		updates["service_fee"] = *input.ServiceFee
	}
	if input.ExtraGuestFee != nil {
		updates["extra_guest_fee"] = *input.ExtraGuestFee
	}
	if input.MinNights != nil {
		updates["min_nights"] = *input.MinNights
	}
	if input.MaxNights != nil {
		updates["max_nights"] = *input.MaxNights
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
	if input.AutoAcceptBookings != nil {
		updates["auto_accept_bookings"] = *input.AutoAcceptBookings
	}
	if input.CalendarMonthsAhead != nil {
		updates["calendar_months_ahead"] = *input.CalendarMonthsAhead
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
	if input.AgencyFee != nil {
		updates["agency_fee"] = *input.AgencyFee
	}
	if input.LegalFee != nil {
		updates["legal_fee"] = *input.LegalFee
	}
	if input.RegistrationFee != nil {
		updates["registration_fee"] = *input.RegistrationFee
	}
	if input.CautionFee != nil {
		updates["caution_fee"] = *input.CautionFee
	}
	if input.ServiceCharge != nil {
		updates["service_charge"] = *input.ServiceCharge
	}
	if input.ServiceCharges != nil {
		updates["service_charge_breakdown"] = mapServiceChargeInputs(input.ServiceCharges)
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
	if input.AgencyFee != nil {
		updates["agency_fee"] = *input.AgencyFee
	}
	if input.LegalFee != nil {
		updates["legal_fee"] = *input.LegalFee
	}
	if input.SurveyFee != nil {
		updates["survey_fee"] = *input.SurveyFee
	}
	if input.TitleProcessingFee != nil {
		updates["title_processing_fee"] = *input.TitleProcessingFee
	}
	if input.DevelopmentFee != nil {
		updates["development_fee"] = *input.DevelopmentFee
	}
	if input.OtherFees != nil {
		updates["other_fees"] = *input.OtherFees
	}
	if input.ServiceCharge != nil {
		updates["service_charge"] = *input.ServiceCharge
	}
	if input.ServiceCharges != nil {
		updates["service_charge_breakdown"] = mapServiceChargeInputs(input.ServiceCharges)
	}
	if input.SaleTerms != nil {
		updates["sale_terms"] = *input.SaleTerms
	}
	if input.SaleAvailabilityFrom != nil {
		updates["sale_availability_from"] = *input.SaleAvailabilityFrom
	}
	return updates
}

func mapRuleGroupInputs(inputs []*model.RuleGroupInput) []domain.RuleGroup {
	if len(inputs) == 0 {
		return []domain.RuleGroup{}
	}
	rules := make([]domain.RuleGroup, len(inputs))
	for i, input := range inputs {
		if input == nil {
			continue
		}
		rules[i] = domain.RuleGroup{
			Category: domain.RuleCategory(input.Category),
			Rules:    mapRuleItems(input.Rules),
		}
	}
	return rules
}

func mapRuleItems(inputs []*model.RuleItemInput) []domain.RuleItem {
	if len(inputs) == 0 {
		return []domain.RuleItem{}
	}
	items := make([]domain.RuleItem, len(inputs))
	for i, input := range inputs {
		if input == nil {
			continue
		}
		// Default to empty description if not provided
		description := map[string]any{}
		if input.Description != nil {
			description = input.Description
		}
		items[i] = domain.RuleItem{
			Name:        domain.RuleSubCategory(input.Name),
			Description: description,
		}
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

func mapServiceChargeInputs(inputs []*model.ServiceChargeInput) *[]domain.ServiceCharge {
	if len(inputs) == 0 {
		return nil
	}
	charges := make([]domain.ServiceCharge, len(inputs))
	for i, input := range inputs {
		if input == nil {
			continue
		}
		charges[i] = domain.ServiceCharge{
			Name:   input.Name,
			Period: domain.PaymentPeriod(input.Period),
			Amount: input.Amount,
		}
	}
	return &charges
}

// mapAmenityGroupInputsToDomain converts AmenityGroupInput slice to domain AmenityGroup slice
func mapAmenityGroupInputsToDomain(inputs []*model.AmenityGroupInput) []domain.AmenityGroup {
	if len(inputs) == 0 {
		return []domain.AmenityGroup{}
	}
	groups := make([]domain.AmenityGroup, len(inputs))
	for i, input := range inputs {
		if input == nil {
			continue
		}
		groups[i] = domain.AmenityGroup{
			Group: input.Group,
			Items: input.Items,
		}
	}
	return groups
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
