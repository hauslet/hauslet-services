package domain

import (
	"strings"
	"time"
)

// EmbeddingDocumentVersion represents the semantic document structure version.
const EmbeddingDocumentVersion = "listing/v2"

// EmbeddingDocument contains the composed text and its version for embeddings.
type EmbeddingDocument struct {
	Text    string
	Version string
}

// EmbeddingDocumentBuilder composes semantic text for embeddings while avoiding noise.
type EmbeddingDocumentBuilder struct {
	version string
	parts   []string
	seen    map[string]struct{}
	nowFunc func() time.Time
}

// NewEmbeddingDocumentBuilder initializes the builder with sensible defaults.
func NewEmbeddingDocumentBuilder() *EmbeddingDocumentBuilder {
	return &EmbeddingDocumentBuilder{
		version: EmbeddingDocumentVersion,
		seen:    make(map[string]struct{}),
		nowFunc: time.Now,
	}
}

// WithListing adds listing-specific context to the embedding document.
func (b *EmbeddingDocumentBuilder) WithListing(listing *Listing) *EmbeddingDocumentBuilder {
	if listing == nil {
		return b
	}

	b.addWithLabel("Title", listing.Title)
	b.addWithLabel("Description", listing.Description)
	b.addWithLabel("Extra details", listing.ExtraDescription)

	if listing.ListingType != "" {
		b.addWithLabel("Listing type", string(listing.ListingType))
	}

	return b
}

// WithProperty adds property-specific context to the embedding document.
func (b *EmbeddingDocumentBuilder) WithProperty(property *Property) *EmbeddingDocumentBuilder {
	if property == nil {
		return b
	}

	if property.PropertyType != "" {
		b.addWithLabel("Property type", string(property.PropertyType))
	}
	if property.PropertyClass != "" {
		b.addWithLabel("Property class", string(property.PropertyClass))
	}
	if property.PropertyCondition != "" {
		b.addWithLabel("Condition", string(property.PropertyCondition))
	}
	if property.FurnishingType != "" {
		b.addWithLabel("Furnishing", string(property.FurnishingType))
	}

	var locationParts []string
	if property.City != "" {
		locationParts = append(locationParts, property.City)
	}
	if property.State != "" {
		locationParts = append(locationParts, property.State)
	}
	if property.Country != "" {
		locationParts = append(locationParts, string(property.Country))
	}
	if len(locationParts) > 0 {
		b.addWithLabel("Location", strings.Join(locationParts, ", "))
	}

	var amenities []string
	for _, group := range property.Amenities {
		for _, item := range group.Items {
			trimmed := strings.TrimSpace(item)
			if trimmed != "" {
				amenities = append(amenities, strings.ToLower(trimmed))
			}
		}
	}
	if len(amenities) > 0 {
		b.addWithLabel("Amenities", strings.Join(amenities, ", "))
	}

	var commercialFeatures []string
	for _, group := range property.FeaturesCommercial {
		for _, item := range group.Items {
			trimmed := strings.TrimSpace(item)
			if trimmed != "" {
				commercialFeatures = append(commercialFeatures, strings.ToLower(trimmed))
			}
		}
	}
	if len(commercialFeatures) > 0 {
		b.addWithLabel("Commercial features", strings.Join(commercialFeatures, ", "))
	}

	if property.Bedrooms != nil && *property.Bedrooms >= 3 {
		b.add("Spacious property suitable for families")
	}
	if property.FurnishingType == Furnished {
		b.add("Ideal for immediate move-in or short stays")
	}

	return b
}

// WithVersion overrides the default document version.
func (b *EmbeddingDocumentBuilder) WithVersion(version string) *EmbeddingDocumentBuilder {
	if v := strings.TrimSpace(version); v != "" {
		b.version = v
	}
	return b
}

// Build composes the final embedding document.
func (b *EmbeddingDocumentBuilder) Build() EmbeddingDocument {
	text := strings.TrimSpace(strings.Join(b.parts, "\n"))
	return EmbeddingDocument{
		Text:    text,
		Version: b.version,
	}
}

func (b *EmbeddingDocumentBuilder) addWithLabel(label, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	b.add(label + ": " + value)
}

func (b *EmbeddingDocumentBuilder) add(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	if _, exists := b.seen[line]; exists {
		return
	}
	b.seen[line] = struct{}{}
	b.parts = append(b.parts, line)
}
