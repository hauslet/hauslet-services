package service

import (
	"testing"
	"time"

	"hauslet/internal/modules/discovery/domain"
	propertydomain "hauslet/internal/modules/property/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCalculateLocationScore(t *testing.T) {
	tests := []struct {
		name      string
		centerLat float64
		centerLng float64
		radiusKm  float64
		targetLat float64
		targetLng float64
		expected  float64
		delta     float64
	}{
		{
			name:      "Exact Match",
			centerLat: 6.5244,
			centerLng: 3.3792,
			radiusKm:  10.0,
			targetLat: 6.5244,
			targetLng: 3.3792,
			expected:  1.0,
			delta:     0.001,
		},
		{
			name:      "At Radius Boundary",
			centerLat: 0.0,
			centerLng: 0.0,
			radiusKm:  111.0, // Approx 1 degree
			targetLat: 1.0,   // Approx 111km away
			targetLng: 0.0,
			expected:  0.0,
			delta:     0.05, // High delta because 1 degree != exactly 111km everywhere
		},
		{
			name:      "Outside Radius",
			centerLat: 0.0,
			centerLng: 0.0,
			radiusKm:  10.0,
			targetLat: 1.0,
			targetLng: 0.0,
			expected:  0.0,
			delta:     0.001,
		},
		{
			name:      "Half Radius",
			centerLat: 0.0,
			centerLng: 0.0,
			radiusKm:  222.0,
			targetLat: 1.0, // Approx 111km
			targetLng: 0.0,
			expected:  0.5,
			delta:     0.05,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateLocationScore(tt.centerLat, tt.centerLng, tt.radiusKm, tt.targetLat, tt.targetLng)
			assert.InDelta(t, tt.expected, score, tt.delta)
		})
	}
}

func TestRankListings_LocationScore(t *testing.T) {
	// Setup headers
	svc := &ServiceImpl{}
	config := domain.DefaultRankingConfig()
	config.LocationWeight = 1.0 // Emphasize location for test

	listingID := uuid.New()

	// Create a listing with location
	listingWithLoc := propertydomain.Listing{
		ID:        listingID,
		CreatedAt: time.Now(),
	}

	scoredListing := propertydomain.ScoredListing{
		Listing: listingWithLoc,
		Score:   nil, // No semantic score
		Location: &propertydomain.Location{
			Lat: 6.5244, // Lagos
			Lng: 3.3792,
		},
	}

	promotions := make(map[uuid.UUID]*PromotionInfo)

	t.Run("With Location Filter", func(t *testing.T) {
		filter := &LocationFilter{
			Latitude:  6.5244,
			Longitude: 3.3792,
			RadiusKm:  10.0,
		}

		results := svc.rankListings([]propertydomain.ScoredListing{scoredListing}, promotions, config, filter)

		assert.NotEmpty(t, results)
		assert.NotNil(t, results[0].Score.LocationScore)
		assert.InDelta(t, 1.0, *results[0].Score.LocationScore, 0.001)
	})

	t.Run("Without Location Filter", func(t *testing.T) {
		results := svc.rankListings([]propertydomain.ScoredListing{scoredListing}, promotions, config, nil)

		assert.NotEmpty(t, results)
		assert.Nil(t, results[0].Score.LocationScore)
	})

	t.Run("Listing Without Location", func(t *testing.T) {
		listingNoLoc := propertydomain.ScoredListing{
			Listing:  propertydomain.Listing{ID: uuid.New(), CreatedAt: time.Now()},
			Location: nil,
		}

		filter := &LocationFilter{
			Latitude:  6.5244,
			Longitude: 3.3792,
			RadiusKm:  10.0,
		}

		results := svc.rankListings([]propertydomain.ScoredListing{listingNoLoc}, promotions, config, filter)

		assert.NotEmpty(t, results)
		assert.Nil(t, results[0].Score.LocationScore)
	})
}
