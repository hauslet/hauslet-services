package domain

// Badge is a type alias to prevent using raw strings
type Badge string

const (
	// --- TRUST & SAFETY (System Assigned) ---
	// User has uploaded a valid Govt ID and Selfie
	BadgeIdentityVerified Badge = "identity_verified"
	// User has verified a phone number
	BadgePhoneVerified Badge = "phone_verified"
	// User has a verified payment method on file
	BadgePaymentVerified Badge = "payment_verified"

	// --- HOSTING MERIT (Earned by Algorithms) ---
	// "Superhost" equivalent: High rating (4.8+), 0 cancellations, high response rate
	BadgeSuperHost Badge = "super_host"
	// Responds to messages within 1 hour on average
	BadgeFastResponder Badge = "fast_responder"
	// Has hosted 50+ stays
	BadgeExperiencedHost Badge = "experienced_host"

	// --- GUEST MERIT (Earned by Behavior) ---
	// "Great Guest": consistently gets 5-star reviews from hosts
	BadgeTopGuest Badge = "top_guest"
	// Has completed 10+ bookings
	BadgeFrequentTraveler Badge = "frequent_traveler"

	// --- PROFESSIONAL ROLES (Assigned by Admin/Verification) ---
	// Verified Real Estate Agent (uploaded license)
	BadgeLicensedAgent Badge = "licensed_agent"
	// Verified Property Owner (uploaded deed/proof of ownership)
	BadgeVerifiedLandlord Badge = "verified_landlord"
	// Official Hauslet Photographer
	BadgeProPhotographer Badge = "pro_photographer"

	// --- COMMUNITY ---
	// Joined in the first 6 months
	BadgeEarlyAdopter Badge = "early_adopter"
)

// AllBadges returns a list of all valid badges (useful for validation)
func AllBadges() []Badge {
	return []Badge{
		BadgeIdentityVerified, BadgePhoneVerified, BadgePaymentVerified,
		BadgeSuperHost, BadgeFastResponder, BadgeExperiencedHost,
		BadgeTopGuest, BadgeFrequentTraveler,
		BadgeLicensedAgent, BadgeVerifiedLandlord, BadgeProPhotographer,
		BadgeEarlyAdopter,
	}
}

// IsValidBadge checks if the badge is recognized.
func IsValidBadge(b Badge) bool {
	for _, allowed := range AllBadges() {
		if allowed == b {
			return true
		}
	}
	return false
}

type BadgeDetails struct {
	ID          Badge  `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Category    string `json:"category"` // "trust", "merit", "role"
}

// GetBadgeDetails returns the UI-ready data for a specific badge
func GetBadgeDetails(b Badge) BadgeDetails {
	switch b {
	case BadgeIdentityVerified:
		return BadgeDetails{ID: b, Label: "Identity Verified", Category: "trust", Description: "This user has verified their government ID."}
	case BadgeSuperHost:
		return BadgeDetails{ID: b, Label: "Super Host", Category: "merit", Description: "Recognized for outstanding hospitality and reliability."}
	case BadgeFastResponder:
		return BadgeDetails{ID: b, Label: "Fast Responder", Category: "merit", Description: "Usually responds within an hour."}
	case BadgeLicensedAgent:
		return BadgeDetails{ID: b, Label: "Licensed Agent", Category: "role", Description: "A verified real estate professional."}
	case BadgeVerifiedLandlord:
		return BadgeDetails{ID: b, Label: "Verified Landlord", Category: "role", Description: "Ownership of property has been verified."}
	case BadgeTopGuest:
		return BadgeDetails{ID: b, Label: "Top Guest", Category: "merit", Description: "Highly rated by hosts."}
	default:
		return BadgeDetails{ID: b, Label: string(b), Category: "general", Description: "Hauslet Community Badge"}
	}
}
