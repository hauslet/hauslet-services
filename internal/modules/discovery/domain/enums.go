package domain

// FeedSectionType represents different sections in the home feed
type FeedSectionType string

const (
	FeedSectionFeatured      FeedSectionType = "featured"
	FeedSectionPremium       FeedSectionType = "premium"
	FeedSectionRecent        FeedSectionType = "recent"
	FeedSectionRecommended   FeedSectionType = "recommended"
	FeedSectionNearYou       FeedSectionType = "near_you"
	FeedSectionRentalsArea   FeedSectionType = "rentals_in_area"
	FeedSectionShortletsArea FeedSectionType = "shortlets_in_area"
	FeedSectionForSaleArea   FeedSectionType = "for_sale_in_area"
)

// String returns the string representation of FeedSectionType
func (f FeedSectionType) String() string {
	return string(f)
}

// IsValid checks if the feed section type is valid
func (f FeedSectionType) IsValid() bool {
	switch f {
	case FeedSectionFeatured, FeedSectionPremium, FeedSectionRecent,
		FeedSectionRecommended, FeedSectionNearYou, FeedSectionRentalsArea,
		FeedSectionShortletsArea, FeedSectionForSaleArea:
		return true
	default:
		return false
	}
}
