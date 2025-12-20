package service

import (
	"context"
	"crypto/sha1"
	"fmt"
	"strings"
	"time"

	"hauslet/internal/modules/property/domain"
	"hauslet/internal/modules/property/repository"
	"hauslet/internal/modules/property/repository/schema"
)

const (
	defaultSearchLimit = 20
	maxSearchLimit     = 50
	searchCacheTTL     = 24 * time.Hour
)

// SearchListings performs semantic search if Query is provided; otherwise applies filters only.
func (s *ServiceImpl) SearchListings(ctx context.Context, filter ListingFilter, limit int) ([]domain.ScoredListing, error) {
	normalized := ""
	if filter.Query != nil {
		normalized = strings.TrimSpace(*filter.Query)
	}

	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}

	// Default to active listings unless explicitly overridden.
	if len(filter.Statuses) == 0 {
		filter.Statuses = []domain.ListingStatus{domain.StatusActive}
	}

	// SECURITY: ALWAYS enforce published=true for public search.
	// This cannot be overridden by clients to prevent exposure of unpublished listings.
	// The published field has been removed from the GraphQL API for this reason.
	pub := true
	filter.Published = &pub

	filter.IncludeDeleted = false

	repoFilter := mapListingFilterToRepo(filter)

	// If no free-text query, fall back to filtered listings (no vector sort).
	if normalized == "" {
		page := repository.Pagination{Limit: limit, Offset: 0}
		result, err := s.repo.ListListings(ctx, repoFilter, page)
		if err != nil {
			return nil, err
		}
		domainListings := domain.MapListingsFromSchema(result.Items)
		scored := make([]domain.ScoredListing, 0, len(domainListings))
		for i := range domainListings {
			scored = append(scored, domain.ScoredListing{
				Listing: domainListings[i],
				Score:   0,
				Ranking: i + 1,
			})
		}
		return scored, nil
	}

	if s.embedding == nil {
		return nil, fmt.Errorf("embedding client not configured")
	}

	embedding, err := s.getOrCreateQueryEmbedding(ctx, normalized)
	if err != nil {
		return nil, err
	}

	results, err := s.repo.SearchListings(ctx, schema.NewVectorEmbedding(embedding), repoFilter, limit)
	if err != nil {
		return nil, err
	}

	scored := make([]domain.ScoredListing, 0, len(results))
	for i, item := range results {
		l := domain.MapListingFromSchema(&item.Listing)
		if l == nil {
			continue
		}
		scored = append(scored, domain.ScoredListing{
			Listing: *l,
			Score:   item.Score,
			Ranking: i + 1,
		})
	}

	return scored, nil
}

func (s *ServiceImpl) getOrCreateQueryEmbedding(ctx context.Context, query string) ([]float32, error) {
	cacheKey := fmt.Sprintf("search:q:%x", sha1.Sum([]byte(strings.ToLower(query))))

	var cached []float32
	if ok, err := s.getCachedValue(ctx, cacheKey, &cached); err == nil && ok {
		return cached, nil
	} else if err != nil {
		s.log.Logf("WARN search embedding cache read failed: %v", err)
	}

	vec, err := s.embedding.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}
	if len(vec) == 0 {
		return nil, fmt.Errorf("empty embedding generated")
	}

	s.setCachedValue(ctx, cacheKey, searchCacheTTL, vec)
	return vec, nil
}
