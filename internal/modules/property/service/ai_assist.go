package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/property/domain"
	aiassist "hauslet/internal/platform/ai/assist"
)

// GenerateListingDescription generates a listing description using AI.
func (s *ServiceImpl) GenerateListingDescription(ctx context.Context, input domain.GenerateListingDescriptionInput) (string, error) {
	if s.aiAssist == nil {
		return "", fmt.Errorf("AI assist client is not configured")
	}

	// Transform domain input to AI platform input
	aiInput := aiassist.DescriptionGenerationInput{
		PropertyType: input.PropertyType,
		City:         input.City,
		State:        input.State,
		Bedrooms:     input.Bedrooms,
		Bathrooms:    input.Bathrooms,
		Amenities:    input.Amenities,
		Highlights:   input.Highlights,
		Tone:         input.Tone,
	}

	description, err := s.aiAssist.GenerateDescription(ctx, aiInput)
	if err != nil {
		s.log.Error("failed to generate description", "error", err)
		return "", fmt.Errorf("failed to generate description: %w", err)
	}

	return description, nil
}
