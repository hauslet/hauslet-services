package graphql

import (
	"encoding/base64"
	"strconv"

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

func sanitizeListingForViewer(l *domain.Listing, v *viewer.Viewer) *domain.Listing {
	if l == nil {
		return nil
	}

	// Draft listings only visible to owner/admin
	if l.Status == domain.StatusDraft {
		if v == nil || (v.UserID != l.OwnerID.String() && !isAdminRole(v.Role)) {
			return nil
		}
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

func buildListingConnection(listings []domain.Listing, total int64, offset int, limit int, v *viewer.Viewer) *model.ListingConnection {
	edges := make([]*model.ListingEdge, 0, len(listings))
	for i, listing := range listings {
		l := listing
		sanitized := sanitizeListingForViewer(&l, v)
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
		OwnerID:     filter.OwnerID,
		PropertyID:  filter.PropertyID,
		Published:   filter.Published,
		HasCalendar: filter.HasCalendar,
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
			Rules:    input.Rules,
		}
	}
	return rules
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

// mapCreateListingInput converts the CreateListingInput into a domain listing with sane defaults.
func mapCreateListingInput(input model.CreateListingInput, ownerID uuid.UUID) domain.Listing {
	listing := domain.Listing{
		OwnerID:            ownerID,
		OwnerType:          input.OwnerType,
		Title:              input.Title,
		Description:        input.Description,
		Currency:           defaultCurrency(input.Currency),
		ListingType:        input.ListingType,
		Status:             domain.StatusDraft,
		Published:          false,
		LatestReviewStatus: domain.ReviewPending,
		HasCalendar:        boolOrDefault(input.HasCalendar, false),
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
		updates["shortlet_details"] = mapShortletInputToDomain(input.ShortletDetails)
	}
	if input.RentalDetails != nil {
		updates["rental_details"] = mapRentalInputToDomain(input.RentalDetails)
	}
	if input.SaleDetails != nil {
		updates["sale_details"] = mapSaleInputToDomain(input.SaleDetails)
	}
	return updates
}

// mapCreateListingPropertyInput builds a Property domain model from the listing property input.
func mapCreateListingPropertyInput(input *model.CreateListingPropertyInput, ownerID uuid.UUID) *domain.Property {
	if input == nil {
		return nil
	}
	prop := &domain.Property{
		UnitNumber:         stringOrDefault(input.UnitNumber, ""),
		Address:            input.Address,
		City:               input.City,
		State:              input.State,
		PostalCode:         stringOrDefault(input.PostalCode, ""),
		Country:            input.Country,
		PropertyClass:      input.PropertyClass,
		PropertyType:       input.PropertyType,
		FurnishingType:     furnishingTypeOrDefault(input.FurnishingType),
		PropertyCondition:  propertyConditionOrDefault(input.PropertyCondition),
		OwnerID:            ownerID,
		Bedrooms:           input.Bedrooms,
		Bathrooms:          input.Bathrooms,
		Toilets:            input.Toilets,
		HalfBathrooms:      input.HalfBathrooms,
		Floors:             input.Floors,
		Units:              intOrDefault(input.Units, 1),
		SquareMeters:       floatOrDefault(input.SquareMeters, 0),
		FloorArea:          input.FloorArea,
		Amenities:          input.Amenities,
		FeaturesCommercial: input.FeaturesCommercial,
	}
	if input.Location != nil {
		prop.Location = &domain.Location{
			Lat:  input.Location.Lat,
			Lng:  input.Location.Lng,
			SRID: 4326,
		}
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
		updates["amenities"] = input.Amenities
	}
	if input.FeaturesCommercial != nil {
		updates["features_commercial"] = input.FeaturesCommercial
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
