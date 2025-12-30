package data

// PropertyData contains property-related seeding data
var (
	PropertyTypes = []string{
		"apartment", "duplex", "villa", "studio",
		"townhouse", "penthouse", "bungalow",
	}

	PropertyConditions = []string{
		"newly_built", "renovated", "used",
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
)

// BedroomDistribution represents typical bedroom counts by property type
var BedroomDistribution = map[string][]int{
	"studio":    {0, 1},
	"apartment": {1, 2, 3, 4},
	"duplex":    {3, 4, 5},
	"villa":     {4, 5, 6},
	"townhouse": {2, 3, 4},
	"penthouse": {3, 4, 5},
	"bungalow":  {2, 3, 4},
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
