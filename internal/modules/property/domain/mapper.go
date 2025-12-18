package domain

import (
	moderationservice "hauslet/internal/modules/moderation/service"
	"hauslet/internal/modules/property/repository/schema"
)

// --- PROPERTY MAPPERS ---

// MapPropertyFromSchema converts repository schema.Property to domain Property
func MapPropertyFromSchema(schemaProperty *schema.Property) *Property {
	if schemaProperty == nil {
		return nil
	}

	property := &Property{
		ID:                 schemaProperty.ID,
		PublicID:           schemaProperty.PublicID,
		UnitNumber:         schemaProperty.UnitNumber,
		Address:            schemaProperty.Address,
		City:               schemaProperty.City,
		State:              schemaProperty.State,
		PostalCode:         schemaProperty.PostalCode,
		Country:            CountryCode(schemaProperty.Country),
		PropertyClass:      PropertyClass(schemaProperty.PropertyClass),
		PropertyType:       PropertyType(schemaProperty.PropertyType),
		FurnishingType:     FurnishingType(schemaProperty.FurnishingType),
		PropertyCondition:  PropertyCondition(schemaProperty.PropertyCondition),
		Bedrooms:           schemaProperty.Bedrooms,
		Bathrooms:          schemaProperty.Bathrooms,
		Toilets:            schemaProperty.Toilets,
		HalfBathrooms:      schemaProperty.HalfBathrooms,
		Floors:             schemaProperty.Floors,
		Units:              schemaProperty.Units,
		OwnerID:            schemaProperty.OwnerID,
		SquareMeters:       schemaProperty.SquareMeters,
		FloorArea:          schemaProperty.FloorArea,
		Amenities:          MapAmenityGroupsFromSchema(schemaProperty.Amenities),
		FeaturesCommercial: MapAmenityGroupsFromSchema(schemaProperty.FeaturesCommercial),
		CreatedAt:          schemaProperty.CreatedAt,
		UpdatedAt:          schemaProperty.UpdatedAt,
	}

	if schemaProperty.Location != nil {
		property.Location = &Location{
			Lat:  schemaProperty.Location.Lat,
			Lng:  schemaProperty.Location.Lng,
			SRID: schemaProperty.Location.SRID,
		}
	}

	return property
}

// MapPropertyToSchema converts domain Property to repository schema.Property
func MapPropertyToSchema(domainProperty *Property) *schema.Property {
	if domainProperty == nil {
		return nil
	}

	schemaProperty := &schema.Property{
		ID:                 domainProperty.ID,
		PublicID:           domainProperty.PublicID,
		UnitNumber:         domainProperty.UnitNumber,
		Address:            domainProperty.Address,
		City:               domainProperty.City,
		State:              domainProperty.State,
		PostalCode:         domainProperty.PostalCode,
		Country:            schema.CountryCode(domainProperty.Country),
		PropertyClass:      schema.PropertyClass(domainProperty.PropertyClass),
		PropertyType:       schema.PropertyType(domainProperty.PropertyType),
		FurnishingType:     schema.FurnishingType(domainProperty.FurnishingType),
		PropertyCondition:  schema.PropertyCondition(domainProperty.PropertyCondition),
		Bedrooms:           domainProperty.Bedrooms,
		Bathrooms:          domainProperty.Bathrooms,
		Toilets:            domainProperty.Toilets,
		HalfBathrooms:      domainProperty.HalfBathrooms,
		Floors:             domainProperty.Floors,
		Units:              domainProperty.Units,
		OwnerID:            domainProperty.OwnerID,
		SquareMeters:       domainProperty.SquareMeters,
		FloorArea:          domainProperty.FloorArea,
		Amenities:          MapAmenityGroupsToSchema(domainProperty.Amenities),
		FeaturesCommercial: MapAmenityGroupsToSchema(domainProperty.FeaturesCommercial),
		CreatedAt:          domainProperty.CreatedAt,
		UpdatedAt:          domainProperty.UpdatedAt,
	}

	if domainProperty.Location != nil && domainProperty.Location.Valid() {
		srid := domainProperty.Location.SRID
		if srid == 0 {
			srid = 4326
		}
		schemaProperty.Location = &schema.GeographyPoint{
			Lat:  domainProperty.Location.Lat,
			Lng:  domainProperty.Location.Lng,
			SRID: srid,
		}
	}

	return schemaProperty
}

// MapPropertiesFromSchema converts a slice of schema properties to domain properties
func MapPropertiesFromSchema(schemaProperties []schema.Property) []Property {
	if schemaProperties == nil {
		return nil
	}

	properties := make([]Property, len(schemaProperties))
	for i, schemaProperty := range schemaProperties {
		if mapped := MapPropertyFromSchema(&schemaProperty); mapped != nil {
			properties[i] = *mapped
		}
	}

	return properties
}

// MapAmenityGroupsFromSchema converts schema amenity groups to domain amenity groups.
func MapAmenityGroupsFromSchema(src []schema.AmenitiesType) []AmenityGroup {
	if src == nil {
		return nil
	}
	out := make([]AmenityGroup, len(src))
	for i, g := range src {
		out[i] = AmenityGroup{
			Group: g.Group,
			Items: g.Items,
		}
	}
	return out
}

// MapAmenityGroupsToSchema converts domain amenity groups to schema amenity groups.
func MapAmenityGroupsToSchema(src []AmenityGroup) []schema.AmenitiesType {
	if src == nil {
		return nil
	}
	out := make([]schema.AmenitiesType, len(src))
	for i, g := range src {
		out[i] = schema.AmenitiesType{
			Group: g.Group,
			Items: g.Items,
		}
	}
	return out
}

// --- LISTING MAPPERS ---

// MapListingFromSchema converts repository schema.Listing to domain Listing
func MapListingFromSchema(schemaListing *schema.Listing) *Listing {
	if schemaListing == nil {
		return nil
	}

	listing := &Listing{
		ID:                 schemaListing.ID,
		PropertyID:         schemaListing.PropertyID,
		OwnerID:            schemaListing.OwnerID,
		OwnerType:          OwnerType(schemaListing.OwnerType),
		Slug:               schemaListing.Slug,
		Title:              schemaListing.Title,
		Description:        schemaListing.Description,
		ExtraDescription:   schemaListing.ExtraDescription,
		Currency:           CurrencyCode(schemaListing.Currency),
		ListingType:        ListingType(schemaListing.ListingType),
		Status:             ListingStatus(schemaListing.Status),
		Published:          schemaListing.Published,
		PublishedAt:        schemaListing.PublishedAt,
		LatestReviewStatus: ReviewStatus(schemaListing.LatestReviewStatus),
		ViewCount:          schemaListing.ViewCount,
		LastViewedAt:       schemaListing.LastViewedAt,
		FeaturedUntil:      schemaListing.FeaturedUntil,
		BoostLevel:         schemaListing.BoostLevel,
		CreatedBy:          schemaListing.CreatedBy,
		UpdatedBy:          schemaListing.UpdatedBy,
		StatusChangedAt:    schemaListing.StatusChangedAt,
		ChangeReason:       schemaListing.ChangeReason,
		HasCalendar:        schemaListing.HasCalendar,
		ShortletDetails:    MapShortletDetailFromSchema(schemaListing.ShortletDetails),
		RentalDetails:      MapRentalDetailFromSchema(schemaListing.RentalDetails),
		SaleDetails:        MapSaleDetailFromSchema(schemaListing.SaleDetails),
		CreatedAt:          schemaListing.CreatedAt,
		UpdatedAt:          schemaListing.UpdatedAt,
	}

	// Handle soft delete
	if schemaListing.DeletedAt.Valid {
		listing.DeletedAt = &schemaListing.DeletedAt.Time
	}

	// Map media
	if len(schemaListing.Media) > 0 {
		listing.Media = make([]ListingMedia, len(schemaListing.Media))
		for i, media := range schemaListing.Media {
			listing.Media[i] = *MapListingMediaFromSchema(&media)
		}
	}

	return listing
}

// MapListingToSchema converts domain Listing to repository schema.Listing
func MapListingToSchema(domainListing *Listing) *schema.Listing {
	if domainListing == nil {
		return nil
	}

	schemaListing := &schema.Listing{
		ID:                 domainListing.ID,
		PropertyID:         domainListing.PropertyID,
		OwnerID:            domainListing.OwnerID,
		OwnerType:          schema.OwnerType(domainListing.OwnerType),
		Slug:               domainListing.Slug,
		Title:              domainListing.Title,
		Description:        domainListing.Description,
		ExtraDescription:   domainListing.ExtraDescription,
		Currency:           schema.CurrencyCode(domainListing.Currency),
		ListingType:        schema.ListingType(domainListing.ListingType),
		Status:             schema.ListingStatus(domainListing.Status),
		Published:          domainListing.Published,
		PublishedAt:        domainListing.PublishedAt,
		LatestReviewStatus: schema.ReviewStatus(domainListing.LatestReviewStatus),
		ViewCount:          domainListing.ViewCount,
		LastViewedAt:       domainListing.LastViewedAt,
		FeaturedUntil:      domainListing.FeaturedUntil,
		BoostLevel:         domainListing.BoostLevel,
		CreatedBy:          domainListing.CreatedBy,
		UpdatedBy:          domainListing.UpdatedBy,
		StatusChangedAt:    domainListing.StatusChangedAt,
		ChangeReason:       domainListing.ChangeReason,
		HasCalendar:        domainListing.HasCalendar,
		ShortletDetails:    MapShortletDetailToSchema(domainListing.ShortletDetails),
		RentalDetails:      MapRentalDetailToSchema(domainListing.RentalDetails),
		SaleDetails:        MapSaleDetailToSchema(domainListing.SaleDetails),
		CreatedAt:          domainListing.CreatedAt,
		UpdatedAt:          domainListing.UpdatedAt,
	}

	// Handle soft delete
	if domainListing.DeletedAt != nil {
		schemaListing.DeletedAt.Time = *domainListing.DeletedAt
		schemaListing.DeletedAt.Valid = true
	}

	// Map media
	if len(domainListing.Media) > 0 {
		schemaListing.Media = make([]schema.ListingMedia, len(domainListing.Media))
		for i, media := range domainListing.Media {
			schemaListing.Media[i] = *MapListingMediaToSchema(&media)
		}
	}

	return schemaListing
}

// MapListingsFromSchema converts a slice of schema listings to domain listings
func MapListingsFromSchema(schemaListings []schema.Listing) []Listing {
	if schemaListings == nil {
		return nil
	}

	listings := make([]Listing, 0, len(schemaListings))
	for _, schemaListing := range schemaListings {
		if mapped := MapListingFromSchema(&schemaListing); mapped != nil {
			listings = append(listings, *mapped)
		}
	}

	return listings
}

// MapRuleGroupsFromSchema converts schema rule groups to domain rule groups.
func MapRuleGroupsFromSchema(src []schema.RuleGroup) []RuleGroup {
	if len(src) == 0 {
		return nil
	}
	out := make([]RuleGroup, len(src))
	for i, g := range src {
		out[i] = RuleGroup{
			Category: RuleCategory(g.Category),
			Rules:    MapRuleItemsFromSchema(g.Rules),
		}
	}
	return out
}

// MapRuleGroupsToSchema converts domain rule groups to schema rule groups.
func MapRuleGroupsToSchema(src []RuleGroup) []schema.RuleGroup {
	if len(src) == 0 {
		return nil
	}
	out := make([]schema.RuleGroup, len(src))
	for i, g := range src {
		out[i] = schema.RuleGroup{
			Category: schema.RuleCategory(g.Category),
			Rules:    MapRuleItemsToSchema(g.Rules),
		}
	}
	return out
}

// MapRuleItemsFromSchema converts schema rule items to domain rule items.
func MapRuleItemsFromSchema(src []schema.RuleItem) []RuleItem {
	if len(src) == 0 {
		return nil
	}
	out := make([]RuleItem, len(src))
	for i, r := range src {
		out[i] = RuleItem{
			Name:        RuleSubCategory(r.Name),
			Description: r.Description,
		}
	}
	return out
}

// MapRuleItemsToSchema converts domain rule items to schema rule items.
func MapRuleItemsToSchema(src []RuleItem) []schema.RuleItem {
	if len(src) == 0 {
		return nil
	}
	out := make([]schema.RuleItem, len(src))
	for i, r := range src {
		out[i] = schema.RuleItem{
			Name:        schema.RuleSubCategory(r.Name),
			Description: r.Description,
		}
	}
	return out
}

// --- DETAIL MAPPERS ---

// MapShortletDetailFromSchema converts schema.ShortletDetail to domain ShortletDetail
func MapShortletDetailFromSchema(schemaDetail *schema.ShortletDetail) *ShortletDetail {
	if schemaDetail == nil {
		return nil
	}

	detail := &ShortletDetail{
		NightlyRate:          schemaDetail.NightlyRate,
		CautionFee:           schemaDetail.CautionFee,
		CleaningFee:          schemaDetail.CleaningFee,
		ServiceFee:           schemaDetail.ServiceFee,
		ExtraGuestFee:        schemaDetail.ExtraGuestFee,
		MinNights:            schemaDetail.MinNights,
		MaxNights:            schemaDetail.MaxNights,
		MaxGuests:            schemaDetail.MaxGuests,
		BaseGuestCount:       schemaDetail.BaseGuestCount,
		CheckInTime:          schemaDetail.CheckInTime,
		CheckOutTime:         schemaDetail.CheckOutTime,
		AccommodationType:    AccommodationType(schemaDetail.AccommodationType),
		CalendarMonthsAhead:  schemaDetail.CalendarMonthsAhead,
		AutoGenerateCalendar: schemaDetail.AutoGenerateCalendar,
	}

	// Map rules
	detail.Rules = MapRuleGroupsFromSchema(schemaDetail.Rules)

	// Map amenities highlights
	if len(schemaDetail.AmenitiesHighlights) > 0 {
		detail.AmenitiesHighlights = make([]AmenityHighlight, len(schemaDetail.AmenitiesHighlights))
		for i, amenity := range schemaDetail.AmenitiesHighlights {
			detail.AmenitiesHighlights[i] = AmenityHighlight{
				Title:   amenity.Title,
				Summary: amenity.Summary,
				Icon:    amenity.Icon,
			}
		}
	}

	return detail
}

// MapShortletDetailToSchema converts domain ShortletDetail to schema.ShortletDetail
func MapShortletDetailToSchema(domainDetail *ShortletDetail) *schema.ShortletDetail {
	if domainDetail == nil {
		return nil
	}

	detail := &schema.ShortletDetail{
		NightlyRate:          domainDetail.NightlyRate,
		CautionFee:           domainDetail.CautionFee,
		CleaningFee:          domainDetail.CleaningFee,
		ServiceFee:           domainDetail.ServiceFee,
		ExtraGuestFee:        domainDetail.ExtraGuestFee,
		MinNights:            domainDetail.MinNights,
		MaxNights:            domainDetail.MaxNights,
		MaxGuests:            domainDetail.MaxGuests,
		BaseGuestCount:       domainDetail.BaseGuestCount,
		CheckInTime:          domainDetail.CheckInTime,
		CheckOutTime:         domainDetail.CheckOutTime,
		AccommodationType:    schema.AccommodationType(domainDetail.AccommodationType),
		CalendarMonthsAhead:  domainDetail.CalendarMonthsAhead,
		AutoGenerateCalendar: domainDetail.AutoGenerateCalendar,
	}

	// Map rules
	detail.Rules = MapRuleGroupsToSchema(domainDetail.Rules)

	// Map amenities highlights
	if len(domainDetail.AmenitiesHighlights) > 0 {
		detail.AmenitiesHighlights = make([]schema.AmenityHighlight, len(domainDetail.AmenitiesHighlights))
		for i, amenity := range domainDetail.AmenitiesHighlights {
			detail.AmenitiesHighlights[i] = schema.AmenityHighlight{
				Title:   amenity.Title,
				Summary: amenity.Summary,
				Icon:    amenity.Icon,
			}
		}
	}

	return detail
}

// MapRentalDetailFromSchema converts schema.RentalDetail to domain RentalDetail
func MapRentalDetailFromSchema(schemaDetail *schema.RentalDetail) *RentalDetail {
	if schemaDetail == nil {
		return nil
	}

	detail := &RentalDetail{
		RentalPrice:            schemaDetail.RentalPrice,
		RentalPricePeriod:      PaymentPeriod(schemaDetail.RentalPricePeriod),
		AgencyFee:              schemaDetail.AgencyFee,
		LegalFee:               schemaDetail.LegalFee,
		RegistrationFee:        schemaDetail.RegistrationFee,
		CautionFee:             schemaDetail.CautionFee,
		ServiceCharge:          schemaDetail.ServiceCharge,
		MinRentalPeriod:        schemaDetail.MinRentalPeriod,
		MaxRentalPeriod:        schemaDetail.MaxRentalPeriod,
		RentalAvailabilityFrom: schemaDetail.RentalAvailabilityFrom,
		RentalTerms:            schemaDetail.RentalTerms,
	}

	// Map service charge breakdown
	if schemaDetail.ServiceChargeBreakdown != nil && len(*schemaDetail.ServiceChargeBreakdown) > 0 {
		charges := make([]ServiceCharge, len(*schemaDetail.ServiceChargeBreakdown))
		for i, charge := range *schemaDetail.ServiceChargeBreakdown {
			charges[i] = ServiceCharge{
				Name:   charge.Name,
				Period: PaymentPeriod(charge.Period),
				Amount: charge.Amount,
			}
		}
		detail.ServiceChargeBreakdown = &charges
	}

	// Map rental rules
	detail.RentalRules = MapRuleGroupsFromSchema(schemaDetail.RentalRules)

	return detail
}

// MapRentalDetailToSchema converts domain RentalDetail to schema.RentalDetail
func MapRentalDetailToSchema(domainDetail *RentalDetail) *schema.RentalDetail {
	if domainDetail == nil {
		return nil
	}

	detail := &schema.RentalDetail{
		RentalPrice:            domainDetail.RentalPrice,
		RentalPricePeriod:      schema.PaymentPeriod(domainDetail.RentalPricePeriod),
		AgencyFee:              domainDetail.AgencyFee,
		LegalFee:               domainDetail.LegalFee,
		RegistrationFee:        domainDetail.RegistrationFee,
		CautionFee:             domainDetail.CautionFee,
		ServiceCharge:          domainDetail.ServiceCharge,
		MinRentalPeriod:        domainDetail.MinRentalPeriod,
		MaxRentalPeriod:        domainDetail.MaxRentalPeriod,
		RentalAvailabilityFrom: domainDetail.RentalAvailabilityFrom,
		RentalTerms:            domainDetail.RentalTerms,
	}

	// Map service charge breakdown
	if domainDetail.ServiceChargeBreakdown != nil && len(*domainDetail.ServiceChargeBreakdown) > 0 {
		charges := make([]schema.ServiceCharge, len(*domainDetail.ServiceChargeBreakdown))
		for i, charge := range *domainDetail.ServiceChargeBreakdown {
			charges[i] = schema.ServiceCharge{
				Name:   charge.Name,
				Period: schema.PaymentPeriod(charge.Period),
				Amount: charge.Amount,
			}
		}
		detail.ServiceChargeBreakdown = &charges
	}

	// Map rental rules
	detail.RentalRules = MapRuleGroupsToSchema(domainDetail.RentalRules)

	return detail
}

// MapSaleDetailFromSchema converts schema.SaleDetail to domain SaleDetail
func MapSaleDetailFromSchema(schemaDetail *schema.SaleDetail) *SaleDetail {
	if schemaDetail == nil {
		return nil
	}

	detail := &SaleDetail{
		SalePrice:            schemaDetail.SalePrice,
		OwnershipTitle:       schemaDetail.OwnershipTitle,
		PaymentPlan:          schemaDetail.PaymentPlan,
		YearBuilt:            schemaDetail.YearBuilt,
		YearRenovated:        schemaDetail.YearRenovated,
		AgencyFee:            schemaDetail.AgencyFee,
		LegalFee:             schemaDetail.LegalFee,
		SurveyFee:            schemaDetail.SurveyFee,
		TitleProcessingFee:   schemaDetail.TitleProcessingFee,
		DevelopmentFee:       schemaDetail.DevelopmentFee,
		OtherFees:            schemaDetail.OtherFees,
		ServiceCharge:        schemaDetail.ServiceCharge,
		SaleTerms:            schemaDetail.SaleTerms,
		SaleAvailabilityFrom: schemaDetail.SaleAvailabilityFrom,
	}

	// Map service charge breakdown
	if schemaDetail.ServiceChargeBreakdown != nil && len(*schemaDetail.ServiceChargeBreakdown) > 0 {
		charges := make([]ServiceCharge, len(*schemaDetail.ServiceChargeBreakdown))
		for i, charge := range *schemaDetail.ServiceChargeBreakdown {
			charges[i] = ServiceCharge{
				Name:   charge.Name,
				Period: PaymentPeriod(charge.Period),
				Amount: charge.Amount,
			}
		}
		detail.ServiceChargeBreakdown = &charges
	}

	return detail
}

// MapSaleDetailToSchema converts domain SaleDetail to schema.SaleDetail
func MapSaleDetailToSchema(domainDetail *SaleDetail) *schema.SaleDetail {
	if domainDetail == nil {
		return nil
	}

	detail := &schema.SaleDetail{
		SalePrice:            domainDetail.SalePrice,
		OwnershipTitle:       domainDetail.OwnershipTitle,
		PaymentPlan:          domainDetail.PaymentPlan,
		YearBuilt:            domainDetail.YearBuilt,
		YearRenovated:        domainDetail.YearRenovated,
		AgencyFee:            domainDetail.AgencyFee,
		LegalFee:             domainDetail.LegalFee,
		SurveyFee:            domainDetail.SurveyFee,
		TitleProcessingFee:   domainDetail.TitleProcessingFee,
		DevelopmentFee:       domainDetail.DevelopmentFee,
		OtherFees:            domainDetail.OtherFees,
		ServiceCharge:        domainDetail.ServiceCharge,
		SaleTerms:            domainDetail.SaleTerms,
		SaleAvailabilityFrom: domainDetail.SaleAvailabilityFrom,
	}

	// Map service charge breakdown
	if domainDetail.ServiceChargeBreakdown != nil && len(*domainDetail.ServiceChargeBreakdown) > 0 {
		charges := make([]schema.ServiceCharge, len(*domainDetail.ServiceChargeBreakdown))
		for i, charge := range *domainDetail.ServiceChargeBreakdown {
			charges[i] = schema.ServiceCharge{
				Name:   charge.Name,
				Period: schema.PaymentPeriod(charge.Period),
				Amount: charge.Amount,
			}
		}
		detail.ServiceChargeBreakdown = &charges
	}

	return detail
}

// --- MEDIA MAPPERS ---

// MapListingMediaFromSchema converts schema.ListingMedia to domain ListingMedia
func MapListingMediaFromSchema(schemaMedia *schema.ListingMedia) *ListingMedia {
	if schemaMedia == nil {
		return nil
	}

	var thumbs ThumbnailMap
	if schemaMedia.Thumbnails != nil {
		thumbs = make(ThumbnailMap, len(schemaMedia.Thumbnails))
		for k, v := range schemaMedia.Thumbnails {
			thumbs[k] = Thumbnail{
				Key:       v.Key,
				URL:       "",
				Width:     v.Width,
				Height:    v.Height,
				SizeBytes: v.Size,
				MimeType:  v.MimeType,
			}
		}
	}

	return &ListingMedia{
		ID:           schemaMedia.ID,
		ListingID:    schemaMedia.ListingID,
		URL:          "",
		Key:          schemaMedia.Key,
		Type:         MediaType(schemaMedia.Type),
		Thumbnails:   thumbs,
		Group:        schemaMedia.Group,
		Caption:      schemaMedia.Caption,
		MimeType:     schemaMedia.MimeType,
		SizeBytes:    schemaMedia.SizeBytes,
		IsPrimary:    schemaMedia.IsPrimary,
		IsGroupCover: schemaMedia.IsGroupCover,
		Order:        schemaMedia.Order,
		CreatedAt:    schemaMedia.CreatedAt,
		UpdatedAt:    schemaMedia.UpdatedAt,
	}
}

// MapListingMediaToSchema converts domain ListingMedia to schema.ListingMedia
func MapListingMediaToSchema(domainMedia *ListingMedia) *schema.ListingMedia {
	if domainMedia == nil {
		return nil
	}

	var thumbs schema.ThumbnailMap
	if domainMedia.Thumbnails != nil {
		thumbs = make(schema.ThumbnailMap, len(domainMedia.Thumbnails))
		for k, v := range domainMedia.Thumbnails {
			thumbs[k] = schema.Thumbnail{
				Key:      v.Key,
				Width:    v.Width,
				Height:   v.Height,
				Size:     v.SizeBytes,
				MimeType: v.MimeType,
			}
		}
	}

	return &schema.ListingMedia{
		ID:           domainMedia.ID,
		ListingID:    domainMedia.ListingID,
		Key:          domainMedia.Key,
		Type:         schema.MediaType(domainMedia.Type),
		Thumbnails:   thumbs,
		Group:        domainMedia.Group,
		Caption:      domainMedia.Caption,
		MimeType:     domainMedia.MimeType,
		SizeBytes:    domainMedia.SizeBytes,
		IsPrimary:    domainMedia.IsPrimary,
		IsGroupCover: domainMedia.IsGroupCover,
		Order:        domainMedia.Order,
		CreatedAt:    domainMedia.CreatedAt,
		UpdatedAt:    domainMedia.UpdatedAt,
	}
}

func MapModerationAggToDomain(agg moderationservice.AggregatedModeration) *AggregatedModeration {
	contentTypes := make([]ContentType, len(agg.ContentTypes))
	for i, ct := range agg.ContentTypes {
		contentTypes[i] = ContentType(ct)
	}

	reasons := make([]string, len(agg.Reasons))
	copy(reasons, agg.Reasons)

	return &AggregatedModeration{
		TargetID:     agg.TargetID,
		Pending:      agg.Pending,
		Accepted:     agg.Accepted,
		Rejected:     agg.Rejected,
		Escalated:    agg.Escalated,
		ContentTypes: contentTypes,
		Reasons:      reasons,
	}
}

// FinalStatus returns the terminal moderation status for the aggregate.
func (a AggregatedModeration) FinalStatus() ModerationStatus {
	if a.Pending > 0 {
		return ModerationStatusPending
	}
	if a.Rejected > 0 {
		return ModerationStatusRejected
	}
	if a.Escalated > 0 {
		return ModerationStatusEscalated
	}
	if a.Accepted > 0 {
		return ModerationStatusAccepted
	}
	return ModerationStatusPending
}
