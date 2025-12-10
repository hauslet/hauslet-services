package service

import (
	"context"

	"hauslet/internal/modules/property/domain"

	"github.com/google/uuid"
)

// SearchListingsByText searches listings using text similarity.
func (s *ServiceImpl) SearchListingsByText(ctx context.Context, sim SimilarityQuery, filter ListingFilter) ([]ScoredResult[domain.Listing], error) {
	// TODO: Implement
	return nil, nil
}

// FindSimilarListings finds listings similar to a given listing.
func (s *ServiceImpl) FindSimilarListings(ctx context.Context, listingID uuid.UUID, limit int, minSimilarity float64) ([]ScoredResult[domain.Listing], error) {
	// TODO: Implement
	return nil, nil
}

// UpdateListingEmbedding updates the text embedding for a listing.
func (s *ServiceImpl) UpdateListingEmbedding(ctx context.Context, listingID uuid.UUID, embedding []float32, model, version string) error {
	// TODO: Implement
	return nil
}
