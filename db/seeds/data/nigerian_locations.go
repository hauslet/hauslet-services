package data

import "hauslet/db/seeds/utils"

// NigerianLocation represents a specific location in Nigeria
type NigerianLocation struct {
	City      string
	State     string
	Area      string
	Address   string
	PostCode  string
	Latitude  float64
	Longitude float64
	Tier      string // "premium", "mid", "budget"
}

// NigerianLocations contains realistic Nigerian property locations
var NigerianLocations = []NigerianLocation{
	// Lagos - Premium Areas
	{"Lagos", "Lagos", "Victoria Island", "12 Ahmadu Bello Way", "101241", 6.4281, 3.4219, "premium"},
	{"Lagos", "Lagos", "Ikoyi", "45 Bourdillon Road", "106104", 6.4550, 3.4315, "premium"},
	{"Lagos", "Lagos", "Lekki Phase 1", "23 Admiralty Way", "105102", 6.4389, 3.5271, "premium"},

	// Lagos - Mid-tier Areas
	{"Lagos", "Lagos", "Ikeja GRA", "18 Obafemi Awolowo Way", "100271", 6.5833, 3.3522, "mid"},
	{"Lagos", "Lagos", "Yaba", "32 Herbert Macaulay Street", "101245", 6.5095, 3.3711, "mid"},
	{"Lagos", "Lagos", "Surulere", "56 Adeniran Ogunsanya Street", "101283", 6.4969, 3.3539, "mid"},

	// Lagos - Budget Areas
	{"Lagos", "Lagos", "Gbagada", "25 Gbagada Expressway", "100234", 6.5432, 3.3870, "budget"},
	{"Lagos", "Lagos", "Ojota", "89 Ikorodu Road", "100242", 6.5892, 3.3783, "budget"},

	// Abuja - Premium Areas
	{"Abuja", "FCT", "Maitama", "10 Aguiyi Ironsi Street", "900271", 9.0820, 7.4950, "premium"},
	{"Abuja", "FCT", "Asokoro", "5 Yakubu Gowon Crescent", "900103", 9.0330, 7.5270, "premium"},

	// Abuja - Mid-tier Areas
	{"Abuja", "FCT", "Gwarinpa", "45 6th Avenue", "900108", 9.1108, 7.4165, "mid"},
	{"Abuja", "FCT", "Jabi", "12 Ladi Kwali Street", "900108", 9.0704, 7.4329, "mid"},

	// Port Harcourt
	{"Port Harcourt", "Rivers", "GRA Phase 2", "15 Stadium Road", "500102", 4.8156, 7.0498, "premium"},
	{"Port Harcourt", "Rivers", "D-Line", "42 Aba Road", "500102", 4.8357, 7.0129, "mid"},

	// Ibadan
	{"Ibadan", "Oyo", "Bodija", "22 Awolowo Avenue", "200284", 7.4340, 3.9087, "mid"},
}

// GetLocationsByTier returns locations filtered by tier
func GetLocationsByTier(tier string) []NigerianLocation {
	var filtered []NigerianLocation
	for _, loc := range NigerianLocations {
		if loc.Tier == tier {
			filtered = append(filtered, loc)
		}
	}
	return filtered
}

// GetLocationsByCity returns locations in a specific city
func GetLocationsByCity(city string) []NigerianLocation {
	var filtered []NigerianLocation
	for _, loc := range NigerianLocations {
		if loc.City == city {
			filtered = append(filtered, loc)
		}
	}
	return filtered
}

// GetRandomLocation returns a random location
func GetRandomLocation() NigerianLocation {
	return utils.RandomChoice(NigerianLocations)
}
