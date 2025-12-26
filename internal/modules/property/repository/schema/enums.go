package schema

// --- 1. CORE TYPES ---

type CountryCode string

const (
	CountryNG CountryCode = "NG"
	CountryGH CountryCode = "GH"
)

type CurrencyCode string

const (
	CurrencyNGN CurrencyCode = "NGN"
	CurrencyGHS CurrencyCode = "GHS"
	CurrencyUSD CurrencyCode = "USD"
	CurrencyEUR CurrencyCode = "EUR"
	CurrencyGBP CurrencyCode = "GBP"
	CurrencyCAD CurrencyCode = "CAD"
	CurrencyAUD CurrencyCode = "AUD"
)

type OwnerType string

const (
	OwnerLandlord   OwnerType = "landlord"
	OwnerAgent      OwnerType = "agent"
	OwnerBusiness   OwnerType = "business"
	OwnerIndividual OwnerType = "individual"
)

// --- 2. PROPERTY CLASSIFICATION ---

type PropertyClass string

const (
	ClassResidential PropertyClass = "residential"
	ClassCommercial  PropertyClass = "commercial"
)

type PropertyType string

const (
	TypeApartment    PropertyType = "apartment"
	TypeFlat         PropertyType = "flat"
	TypeDuplex       PropertyType = "duplex"
	TypePenthouse    PropertyType = "penthouse"
	TypeStudio       PropertyType = "studio"
	TypeHouse        PropertyType = "house"
	TypeBungalow     PropertyType = "bungalow"
	TypeDetached     PropertyType = "detached_house"
	TypeSemiDetached PropertyType = "semi_detached"
	TypeTerraced     PropertyType = "terraced"
	TypeTownhouse    PropertyType = "townhouse"
	TypeVilla        PropertyType = "villa"
	TypeMansion      PropertyType = "mansion"
	TypeEstate       PropertyType = "estate"
	TypeOfficeSpace  PropertyType = "office_space"
	TypeRetailSpace  PropertyType = "retail_space"
	TypeWarehouse    PropertyType = "warehouse"
	TypeIndustrial   PropertyType = "industrial"
	TypeMixedUse     PropertyType = "mixed_use"
	TypeEventSpace   PropertyType = "event_space"
	TypeCoWorking    PropertyType = "co_working_space"
)

type PropertyCondition string

const (
	ConditionNew               PropertyCondition = "new"
	ConditionUsed              PropertyCondition = "used"
	ConditionUnderRenovation   PropertyCondition = "under_renovation"
	ConditionUnderConstruction PropertyCondition = "under_construction"
	ConditionRenovated         PropertyCondition = "renovated"
)

type FurnishingType string

const (
	Furnished     FurnishingType = "furnished"
	Unfurnished   FurnishingType = "unfurnished"
	SemiFurnished FurnishingType = "semi_furnished"
)

// --- 3. LISTING STATUS & TYPES ---

type ListingType string

const (
	ListingSale     ListingType = "sale"
	ListingRent     ListingType = "rent"
	ListingShortLet ListingType = "shortlet"
)

type ListingStatus string

const (
	StatusActive              ListingStatus = "active"
	StatusInactive            ListingStatus = "inactive"
	StatusPendingVerification ListingStatus = "pending_verification"
	StatusSold                ListingStatus = "sold"
	StatusRented              ListingStatus = "rented"
	StatusArchived            ListingStatus = "archived"
	StatusDraft               ListingStatus = "draft"
	StatusSuspended           ListingStatus = "suspended"
	StatusUnderReview         ListingStatus = "under_review"
	StatusRequiresUpdates     ListingStatus = "requires_updates"
)

type ReviewStatus string

const (
	ReviewPending      ReviewStatus = "pending"
	ReviewApproved     ReviewStatus = "approved"
	ReviewRejected     ReviewStatus = "rejected"
	ReviewInconclusive ReviewStatus = "inconclusive"
)

// --- 4. DETAILS ---

type PaymentPeriod string

const (
	PayOneTime    PaymentPeriod = "one_time"
	PayDaily      PaymentPeriod = "daily"
	PayWeekly     PaymentPeriod = "weekly"
	PayMonthly    PaymentPeriod = "monthly"
	PayQuarterly  PaymentPeriod = "quarterly"
	PayBiAnnually PaymentPeriod = "bi_annually"
	PayYearly     PaymentPeriod = "yearly"
)

type AccommodationType string

const (
	AccSingleRoom  AccommodationType = "single_room"
	AccDoubleRoom  AccommodationType = "double_room"
	AccSharedRoom  AccommodationType = "shared_room"
	AccEntirePlace AccommodationType = "entire_place"
)

type RuleCategory string

const (
	RuleHouseRules          RuleCategory = "house_rules"
	RuleGeneral             RuleCategory = "general"
	RuleCheckInOut          RuleCategory = "check_in_check_out"
	RuleCancellationPolicy  RuleCategory = "cancellation_policy"
	RuleSafetyAndDisclosure RuleCategory = "safety_and_disclosure"
	RuleCustom              RuleCategory = "custom"
)

type RuleSubCategory string

const (
	RuleSecurity      RuleSubCategory = "security"
	RuleProhibited    RuleSubCategory = "prohibited_activities"
	RuleCheckInWindow RuleSubCategory = "check_in_window"
	RuleCheckOutTime  RuleSubCategory = "check_out_time"
	RuleCheckinMethod RuleSubCategory = "check_in_method"
	RuleSmoking       RuleSubCategory = "smoking"
	RulePets          RuleSubCategory = "pets"
	RuleEvents        RuleSubCategory = "events"
	RuleGuests        RuleSubCategory = "guests"
	RuleRefundPolicy  RuleSubCategory = "refund_policy"
	RuleMustKnow      RuleSubCategory = "must_know"
	RuleUserCustom    RuleSubCategory = "custom"
)

// --- 5. VALIDATION MAPS (Lookup Tables) ---
var validAmenities = map[string]bool{
	"borehole": true, "fenced": true, "gated_estate": true, "security": true,
	"cctv": true, "interlocked_floor": true, "balcony": true, "air_conditioning": true,
	"parking_space": true, "pop_ceiling": true, "water_heater": true, "kitchen_cabinet": true,
	"wardrobe": true, "prepaid_meter": true, "swimming_pool": true, "gym": true,
	"children_play_area": true, "elevator": true, "generator": true, "solar_power": true,
	"internet": true, "laundry_room": true, "guest_toilet": true, "inverter": true,
	"security_house": true, "staff_quarters": true, "wifi": true, "electricity": true,
	"water_supply": true, "kitchen": true, "private_bathroom": true, "microwave": true,
	"refrigerator": true, "television": true, "iron": true, "kettle": true,
	"stove": true, "oven": true, "kitchen_utensils": true, "closet": true,
	"bedding_towels": true, "dining_area": true, "workspace": true, "sofa": true,
	"curtains": true, "fan": true, "bedside_lamp": true, "mirror": true,
	"security_guard": true, "gated_compound": true, "access_control": true,
	"smoke_detector": true, "fire_extinguisher": true, "pet_friendly": true,
	"shared_living": true, "outdoor_furniture": true, "garden": true,
	"private_entrance": true, "storage_space": true, "hot_tub": true, "rooftop": true,
}

// IsValidAmenity checks if a string is in the allowed list
func IsValidAmenity(a string) bool {
	return validAmenities[a]
}

// IsValidRuleCategory checks if a RuleCategory is valid
func (r RuleCategory) IsValid() bool {
	switch r {
	case RuleHouseRules, RuleGeneral, RuleCheckInOut, RuleCancellationPolicy, RuleSafetyAndDisclosure, RuleCustom:
		return true
	}
	return false
}

// IsValidRuleSubCategory checks if a RuleSubCategory is valid
func (r RuleSubCategory) IsValid() bool {
	switch r {
	case RuleSecurity, RuleProhibited, RuleCheckInWindow, RuleCheckOutTime, RuleCheckinMethod, RuleSmoking, RulePets, RuleEvents, RuleGuests, RuleRefundPolicy, RuleMustKnow, RuleUserCustom:
		return true
	}
	return false
}
