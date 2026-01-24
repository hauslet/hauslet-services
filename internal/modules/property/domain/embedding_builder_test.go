package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestEmbeddingDocumentBuilder_RichContent(t *testing.T) {
	// Setup test data
	bedrooms := 3
	bathrooms := 4
	toilets := 5
	floors := 2

	prop := &Property{
		ID:             uuid.New(),
		PropertyType:   TypeHouse,
		PropertyClass:  ClassResidential,
		FurnishingType: Furnished,
		City:           "Lekki",
		State:          "Lagos",
		Country:        CountryNG,
		SquareMeters:   250.5,
		Bedrooms:       &bedrooms,
		Bathrooms:      &bathrooms,
		Toilets:        &toilets,
		Floors:         &floors,
		Amenities: []AmenityGroup{
			{Group: "General", Items: []string{"Pool", "Wifi"}},
		},
	}

	listing := &Listing{
		ID:          uuid.New(),
		Title:       "Luxury Villa",
		Description: "A beautiful place",
		ListingType: ListingShortLet,
		ShortletDetails: &ShortletDetail{
			MaxGuests:         6,
			AccommodationType: AccEntirePlace,
			AmenitiesHighlights: []AmenityHighlight{
				{Title: "Ocean View"},
			},
			Rules: []RuleGroup{
				{Rules: []RuleItem{{Name: RuleSmoking}}},
			},
		},
	}

	// Build document
	builder := NewEmbeddingDocumentBuilder()
	doc := builder.WithProperty(prop).WithListing(listing).Build()

	// Verify content
	text := doc.Text

	// Check physical attributes
	assert.Contains(t, text, "Bedrooms: 3")
	assert.Contains(t, text, "Bathrooms: 4")
	assert.Contains(t, text, "Toilets: 5")
	assert.Contains(t, text, "Floors: 2")
	assert.Contains(t, text, "Size: 250 sqm") // Rounded 250.5 -> 250 (half to even)

	// Check polymorphic details
	assert.Contains(t, text, "Accommodation type: entire_place")
	assert.Contains(t, text, "Max guests: 6")
	assert.Contains(t, text, "Highlights: Ocean View")
	assert.Contains(t, text, "Rules: smoking")

	// Check standard fields
	assert.Contains(t, text, "Title: Luxury Villa")
	assert.Contains(t, text, "Location: Lekki, Lagos, NG")
	assert.Contains(t, text, "Property type: house")
}
