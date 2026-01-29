package aiassist

import "context"

// AssistClient defines the interface for AI assistance capabilities.
type AssistClient interface {
	// GenerateDescription generates a listing description based on property details.
	GenerateDescription(ctx context.Context, input DescriptionGenerationInput) (string, error)
	QualifyLead(ctx context.Context, input LeadQualificationInput) (*LeadQualificationResult, error)
}

// DescriptionGenerationInput holds data for generating property descriptions.
type DescriptionGenerationInput struct {
	PropertyType string
	City         string
	State        string
	Bedrooms     int
	Bathrooms    int
	Amenities    []string
	Highlights   []string
	Tone         string
}

// LeadQualificationInput holds data for lead qualification.
type LeadQualificationInput struct {
	Name    string
	Email   string
	Message string
	Source  string
}

// LeadQualificationResult holds the output of lead qualification.
type LeadQualificationResult struct {
	Score   float64 // 0.0 - 1.0 (1.0 = highly qualified)
	Reason  string  // Explanation for the score
	Intent  string  // e.g., "inquiry", "booking", "complaint", "spam"
	Urgency string  // e.g., "high", "medium", "low"
}
