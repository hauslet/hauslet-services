package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateFinalScore(t *testing.T) {
	// Use the recommended "Best Match" config
	config := RankingConfig{
		SemanticWeight:        0.80,
		PromotionWeight:       0.10,
		RecencyWeight:         0.05,
		LocationWeight:        0.05,
		PersonalizationWeight: 0.00,
	}

	tests := []struct {
		name     string
		ranking  RankingScore
		expected float64
	}{
		{
			name: "Scenario 1: Perfect Semantic Match, Old, Not Promoted",
			ranking: RankingScore{
				SemanticScore:  floatPtr(1.0), // Perfect match
				PromotionBoost: 1.0,           // No boost
				RecencyScore:   0.1,           // Old (>90 days)
				LocationScore:  nil,
			},
			// Calculation:
			// Semantic: 1.0 * 0.8 = 0.8
			// Promotion: (1.0 - 1.0) * 0.1 = 0
			// Recency: 0.1 * 0.05 = 0.005
			// Total: 0.805
			expected: 0.805,
		},
		{
			name: "Scenario 2: Good Semantic Match, New, Not Promoted",
			ranking: RankingScore{
				SemanticScore:  floatPtr(0.8), // Good match
				PromotionBoost: 1.0,
				RecencyScore:   1.0, // Brand new (<7 days)
				LocationScore:  nil,
			},
			// Calculation:
			// Semantic: 0.8 * 0.8 = 0.64
			// Promotion: 0
			// Recency: 1.0 * 0.05 = 0.05
			// Total: 0.69
			expected: 0.69,
		},
		{
			name: "Scenario 3: Poor Semantic Match, New, Promoted (1.5x)",
			ranking: RankingScore{
				SemanticScore:  floatPtr(0.2), // Poor match
				PromotionBoost: 1.5,           // Significant boost
				RecencyScore:   1.0,           // New
				LocationScore:  nil,
			},
			// Calculation:
			// Semantic: 0.2 * 0.8 = 0.16
			// Promotion: (1.5 - 1.0) * 0.1 = 0.05
			// Recency: 1.0 * 0.05 = 0.05
			// Total: 0.26
			// Result: Even with boost and newness, poor relevance keeps it low. Correct.
			expected: 0.26,
		},
		{
			name: "Scenario 4: The 'Razor Thin' Case (Penthouse vs Villa)",
			// Penthouse: Very high relevance (0.9), Old (0.5 recency)
			ranking: RankingScore{
				SemanticScore:  floatPtr(0.9),
				PromotionBoost: 1.0,
				RecencyScore:   0.5,
				LocationScore:  nil,
			},
			// Calculation:
			// Semantic: 0.9 * 0.8 = 0.72
			// Promotion: 0
			// Recency: 0.5 * 0.05 = 0.025
			// Total: 0.745
			expected: 0.745,
		},
		{
			name: "Scenario 5: The 'Razor Thin' Case (Competing Villa)",
			// Villa: Lower relevance (0.8), Newer (0.8 recency)
			ranking: RankingScore{
				SemanticScore:  floatPtr(0.8),
				PromotionBoost: 1.0,
				RecencyScore:   0.8,
				LocationScore:  nil,
			},
			// Calculation:
			// Semantic: 0.8 * 0.8 = 0.64
			// Promotion: 0
			// Recency: 0.8 * 0.05 = 0.04
			// Total: 0.68
			// Result: Penthouse (0.745) beats Villa (0.68) comfortably by 0.065.
			// The new weights created a healthy buffer!
			expected: 0.68,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := tt.ranking.CalculateFinalScore(config)
			assert.InDelta(t, tt.expected, score, 0.0001)
		})
	}
}

func floatPtr(v float64) *float64 {
	return &v
}
