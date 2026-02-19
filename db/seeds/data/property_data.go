package data

// PropertyData contains property-related seeding data
var (
	ResidentialPropertyTypes = []string{
		"apartment", "flat", "duplex", "penthouse", "studio", "house",
		"bungalow", "detached_house", "semi_detached", "terraced",
		"townhouse", "villa", "mansion", "estate",
	}

	CommercialPropertyTypes = []string{
		"office_space", "retail_space", "warehouse", "industrial",
		"mixed_use", "event_space", "co_working_space",
	}

	PropertyTypes = []string{
		"apartment", "flat", "duplex", "penthouse", "studio", "house",
		"bungalow", "detached_house", "semi_detached", "terraced",
		"townhouse", "villa", "mansion", "estate",
		"office_space", "retail_space", "warehouse", "industrial",
		"mixed_use", "event_space", "co_working_space",
	}

	PropertyConditions = []string{
		"new", "renovated", "used", "under_renovation", "under_construction",
	}

	FurnishingTypes = []string{
		"furnished", "semi_furnished", "unfurnished",
	}

	CommonAmenities = []string{
		"electricity", "security", "parking_space", "water_supply",
		"generator", "wifi", "air_conditioning",
	}

	OptionalAmenities = []string{
		"swimming_pool", "gym", "balcony", "garden",
		"elevator", "cctv", "security_guard", "staff_quarters",
		"borehole", "fenced", "gated_estate", "kitchen_cabinet",
		"wardrobe", "rooftop", "hot_tub",
	}

	CommercialAmenities = []string{
		"electricity", "security", "parking_space", "water_supply",
		"generator", "cctv", "security_guard", "internet",
		"elevator", "fenced", "gated_estate", "storage_space",
		"workspace", "fire_extinguisher", "smoke_detector",
	}

	OwnershipTitles = []string{
		"C of O", "Governor's Consent", "Deed of Assignment", "Registered Survey",
	}
)

// BedroomDistribution represents typical bedroom counts by property type
var BedroomDistribution = map[string][]int{
	"studio":         {0, 1},
	"apartment":      {1, 2, 3, 4},
	"flat":           {1, 2, 3},
	"duplex":         {3, 4, 5},
	"penthouse":      {2, 3, 4, 5},
	"house":          {2, 3, 4, 5},
	"bungalow":       {2, 3, 4},
	"detached_house": {4, 5, 6},
	"semi_detached":  {3, 4, 5},
	"terraced":       {2, 3, 4},
	"townhouse":      {2, 3, 4},
	"villa":          {4, 5, 6},
	"mansion":        {5, 6, 7},
	"estate":         {3, 4, 5},
}

// CommercialUnitDistribution represents typical unit capacities for commercial property types.
var CommercialUnitDistribution = map[string][]int{
	"office_space":     {1, 2, 4, 8},
	"retail_space":     {1, 2, 3},
	"warehouse":        {1, 2, 4},
	"industrial":       {1, 2, 3},
	"mixed_use":        {1, 2, 4, 6},
	"event_space":      {1, 2},
	"co_working_space": {2, 4, 8, 12},
}

// PriceRanges defines realistic price ranges by tier and listing type (in NGN)
var PriceRanges = map[string]map[string][2]int64{
	"premium": {
		"sale":     {50_000_000, 500_000_000},
		"rent":     {1_500_000, 10_000_000},
		"shortlet": {50_000, 300_000},
	},
	"mid": {
		"sale":     {15_000_000, 50_000_000},
		"rent":     {500_000, 3_000_000},
		"shortlet": {20_000, 80_000},
	},
	"budget": {
		"sale":     {5_000_000, 15_000_000},
		"rent":     {200_000, 800_000},
		"shortlet": {8_000, 25_000},
	},
}
