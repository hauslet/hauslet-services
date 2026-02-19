package seeders

import (
	"fmt"
	"strings"
	"time"

	"hauslet/db/seeds/data"
	"hauslet/db/seeds/utils"
	propertySchema "hauslet/internal/modules/property/repository/schema"

	"github.com/google/uuid"
)

var propertyTypesWithUnitNumbers = map[string]bool{
	"apartment":        true,
	"flat":             true,
	"studio":           true,
	"penthouse":        true,
	"office_space":     true,
	"co_working_space": true,
	"retail_space":     true,
	"mixed_use":        true,
}

var commercialPropertyTypeSet = buildTypeSet(data.CommercialPropertyTypes)

// SeedProperty seeds physical properties with complete field coverage.
func SeedProperty(ctx *SeedContext) error {
	var users []uuid.UUID
	if err := ctx.DB.Table("users").Select("id").Where("role = ?", "user").Scan(&users).Error; err != nil {
		return fmt.Errorf("failed to fetch users: %w", err)
	}

	targetOwnerIDs, err := resolveTargetOwnerIDs(ctx.Config.ListingOwnerIDs, users)
	if err != nil {
		return fmt.Errorf("failed to resolve target owners: %w", err)
	}

	if len(targetOwnerIDs) == 0 {
		return fmt.Errorf("no users found - seed auth module first")
	}

	fmt.Printf("   Generating %d properties in memory...\n", ctx.Config.PropertyCount)
	propertyTypes := buildPropertyTypePlan(ctx.Config.PropertyCount)
	properties := make([]*propertySchema.Property, 0, ctx.Config.PropertyCount)
	for i := 0; i < ctx.Config.PropertyCount; i++ {
		ownerID := utils.RandomChoice(targetOwnerIDs)
		properties = append(properties, generateRandomProperty(ownerID, propertyTypes[i]))
	}

	fmt.Printf("   Inserting %d properties in batches...\n", len(properties))
	if err := ctx.DB.CreateInBatches(properties, 100).Error; err != nil {
		return fmt.Errorf("failed to batch insert properties: %w", err)
	}

	return nil
}

func buildPropertyTypePlan(count int) []propertySchema.PropertyType {
	if count <= 0 {
		return []propertySchema.PropertyType{}
	}

	all := make([]propertySchema.PropertyType, 0, len(data.PropertyTypes))
	for _, value := range data.PropertyTypes {
		all = append(all, propertySchema.PropertyType(value))
	}
	utils.Shuffle(all)

	plan := make([]propertySchema.PropertyType, 0, count)
	limit := count
	if limit > len(all) {
		limit = len(all)
	}
	plan = append(plan, all[:limit]...)

	for len(plan) < count {
		if utils.RandomBoolWithProbability(0.7) {
			plan = append(plan, propertySchema.PropertyType(utils.RandomChoice(data.ResidentialPropertyTypes)))
			continue
		}
		plan = append(plan, propertySchema.PropertyType(utils.RandomChoice(data.CommercialPropertyTypes)))
	}

	utils.Shuffle(plan)
	return plan
}

func generateRandomProperty(ownerID uuid.UUID, propertyType propertySchema.PropertyType) *propertySchema.Property {
	location := data.GetRandomLocation()
	propertyClass := resolvePropertyClass(propertyType)

	unitNumber := randomUnitNumber(propertyType)
	furnishing := randomFurnishing(propertyClass)
	condition := propertySchema.PropertyCondition(utils.RandomChoice(data.PropertyConditions))

	var (
		bedrooms      *int
		bathrooms     *int
		toilets       *int
		halfBathrooms *int
		floors        *int
		units         int
		squareMeters  float64
		floorArea     *float64
	)

	if propertyClass == propertySchema.ClassCommercial {
		bathroomCount := utils.RandomInt(1, 4)
		toiletCount := utils.RandomInt(bathroomCount, bathroomCount+2)
		halfBathCount := 0
		if utils.RandomBoolWithProbability(0.35) {
			halfBathCount = utils.RandomInt(1, 2)
		}
		floorCount := utils.RandomInt(1, 8)

		unitOptions := data.CommercialUnitDistribution[string(propertyType)]
		if len(unitOptions) == 0 {
			unitOptions = []int{1, 2, 4}
		}
		units = utils.RandomChoice(unitOptions)
		squareMeters = float64(utils.RandomInt(90, 2200))
		floorAreaValue := squareMeters * 0.85

		bathrooms = ptrInt(bathroomCount)
		toilets = ptrInt(toiletCount)
		halfBathrooms = ptrInt(halfBathCount)
		floors = ptrInt(floorCount)
		floorArea = ptrFloat(floorAreaValue)
	} else {
		bedroomOptions := data.BedroomDistribution[string(propertyType)]
		if len(bedroomOptions) == 0 {
			bedroomOptions = data.BedroomDistribution["apartment"]
		}
		bedroomCount := utils.RandomChoice(bedroomOptions)
		bathroomCount := 1
		if bedroomCount > 0 {
			bathroomCount = utils.RandomInt(maxInt(1, bedroomCount), bedroomCount+1)
		}
		toiletCount := utils.RandomInt(bathroomCount, bathroomCount+1)
		halfBathCount := 0
		if utils.RandomBoolWithProbability(0.3) {
			halfBathCount = utils.RandomInt(1, 2)
		}
		floorCount := utils.RandomInt(1, 4)
		units = 1
		if propertyType == propertySchema.TypeEstate {
			units = utils.RandomInt(2, 8)
		}

		minArea := 35
		maxArea := 600
		if bedroomCount > 0 {
			minArea = bedroomCount * 35
			maxArea = bedroomCount * 130
		}
		squareMeters = float64(utils.RandomInt(minArea, maxArea))

		bedrooms = ptrInt(bedroomCount)
		bathrooms = ptrInt(bathroomCount)
		toilets = ptrInt(toiletCount)
		halfBathrooms = ptrInt(halfBathCount)
		floors = ptrInt(floorCount)
		if utils.RandomBoolWithProbability(0.65) {
			floorArea = ptrFloat(squareMeters * 0.8)
		}
	}

	return &propertySchema.Property{
		ID:       uuid.New(),
		PublicID: "", // Auto-generated by GORM hook

		UnitNumber: unitNumber,
		Address:    location.Address,
		City:       location.City,
		State:      location.State,
		PostalCode: location.PostCode,
		Country:    propertySchema.CountryNG,
		Location: &propertySchema.GeographyPoint{
			Lat: location.Latitude,
			Lng: location.Longitude,
		},

		PropertyClass:     propertyClass,
		PropertyType:      propertyType,
		FurnishingType:    furnishing,
		PropertyCondition: condition,

		Bedrooms:      bedrooms,
		Bathrooms:     bathrooms,
		Toilets:       toilets,
		HalfBathrooms: halfBathrooms,
		Floors:        floors,
		Units:         units,
		OwnerID:       ownerID,
		SquareMeters:  squareMeters,
		FloorArea:     floorArea,

		Amenities:          generateAmenities(propertyClass),
		FeaturesCommercial: generateCommercialFeatures(propertyClass),

		CreatedAt: utils.RandomPastDate(365),
		UpdatedAt: time.Now(),
	}
}

func resolvePropertyClass(propertyType propertySchema.PropertyType) propertySchema.PropertyClass {
	if commercialPropertyTypeSet[string(propertyType)] {
		return propertySchema.ClassCommercial
	}
	return propertySchema.ClassResidential
}

func randomUnitNumber(propertyType propertySchema.PropertyType) string {
	if !propertyTypesWithUnitNumbers[string(propertyType)] && !utils.RandomBoolWithProbability(0.25) {
		return ""
	}

	letter := utils.RandomChoice([]string{"A", "B", "C", "D", "E"})
	return fmt.Sprintf("%s-%02d", letter, utils.RandomInt(1, 40))
}

func randomFurnishing(propertyClass propertySchema.PropertyClass) propertySchema.FurnishingType {
	if propertyClass == propertySchema.ClassCommercial {
		return propertySchema.FurnishingType(utils.RandomChoice([]string{"unfurnished", "semi_furnished"}))
	}
	return propertySchema.FurnishingType(utils.RandomChoice(data.FurnishingTypes))
}

func generateAmenities(propertyClass propertySchema.PropertyClass) []propertySchema.AmenitiesType {
	if propertyClass == propertySchema.ClassCommercial {
		base := utils.RandomUniqueChoices(data.CommercialAmenities, utils.RandomInt(4, 8))
		return []propertySchema.AmenitiesType{
			{
				Group: "commercial",
				Items: uniqueStrings(base),
			},
		}
	}

	amenityCount := utils.RandomInt(3, 7)
	selected := utils.RandomUniqueChoices(data.OptionalAmenities, amenityCount)
	all := uniqueStrings(append(append([]string{}, data.CommonAmenities...), selected...))

	return []propertySchema.AmenitiesType{
		{
			Group: "general",
			Items: all,
		},
	}
}

func generateCommercialFeatures(propertyClass propertySchema.PropertyClass) []propertySchema.AmenitiesType {
	if propertyClass != propertySchema.ClassCommercial {
		return nil
	}

	featurePool := []string{
		"workspace", "parking_space", "security", "generator", "internet",
		"elevator", "fire_extinguisher", "smoke_detector", "storage_space",
	}
	selected := utils.RandomUniqueChoices(featurePool, utils.RandomInt(3, 6))

	return []propertySchema.AmenitiesType{
		{
			Group: "commercial_features",
			Items: selected,
		},
	}
}

func buildTypeSet(values []string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[strings.TrimSpace(value)] = true
	}
	return result
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func ptrInt(v int) *int {
	return &v
}

func ptrFloat(v float64) *float64 {
	return &v
}
