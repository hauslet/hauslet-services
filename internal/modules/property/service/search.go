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

	searchCacheVersion     = "v1"
	embeddingRequestTimeout = 1500 * time.Millisecond
	// semanticScoreCutoff drops low-quality matches when >0; distance scores above this value are ignored.
	semanticScoreCutoff = 0.0
)

// SearchListings performs semantic search if Query is provided; otherwise applies filters only.
func (s *ServiceImpl) SearchListings(ctx context.Context, filter ListingFilter, limit int) ([]domain.ScoredListing, error) {
	normalized := ""
	if filter.Query != nil {
		normalized = strings.TrimSpace(strings.ToLower(*filter.Query))
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
		return s.searchWithFiltersOnly(ctx, repoFilter, limit)
	}

	if s.embedding == nil {
		if s.log != nil {
			s.log.Logf("WARN embedding client not configured, falling back to filtered search")
		}
		return s.searchWithFiltersOnly(ctx, repoFilter, limit)
	}

	embedding, err := s.getOrCreateQueryEmbedding(ctx, normalized)
	if err != nil {
		if s.log != nil {
			s.log.Logf("WARN semantic embedding unavailable, falling back to filtered search: %v", err)
		}
		return s.searchWithFiltersOnly(ctx, repoFilter, limit)
	}

	results, err := s.repo.SearchListings(ctx, schema.NewVectorEmbedding(embedding), repoFilter, limit)
	if err != nil {
		return nil, err
	}

	scored := make([]domain.ScoredListing, 0, len(results))
	rank := 1
	for _, item := range results {
		if semanticScoreCutoff > 0 && item.Score > semanticScoreCutoff {
			continue
		}
		l := domain.MapListingFromSchema(&item.Listing)
		if l == nil {
			continue
		}
		score := item.Score
		scored = append(scored, domain.ScoredListing{
			Listing: *l,
			Score:   &score,
			Ranking: rank,
		})
		rank++
	}

	return scored, nil
}

func (s *ServiceImpl) searchWithFiltersOnly(ctx context.Context, repoFilter repository.ListingFilter, limit int) ([]domain.ScoredListing, error) {
	page := repository.Pagination{Limit: limit, Offset: 0}
	result, err := s.repo.ListListings(ctx, repoFilter, page)
	if err != nil {
		return nil, err
	}
	domainListings := domain.MapListingsFromSchema(result.Items)
	scored := make([]domain.ScoredListing, 0, len(domainListings))
	rank := 1
	for i := range domainListings {
		scored = append(scored, domain.ScoredListing{
			Listing: domainListings[i],
			Score:   nil,
			Ranking: rank,
		})
		rank++
	}
	return scored, nil
}

func (s *ServiceImpl) getOrCreateQueryEmbedding(ctx context.Context, normalizedQuery string) ([]float32, error) {
	cacheKey := s.searchEmbeddingCacheKey(normalizedQuery)

	var cached []float32
	if ok, err := s.getCachedValue(ctx, cacheKey, &cached); err == nil && ok {
		return cached, nil
	} else if err != nil {
		s.log.Logf("WARN search embedding cache read failed: %v", err)
	}

	value, err, _ := s.embeddingGroup.Do(cacheKey, func() (any, error) {
		var innerCached []float32
		if ok, err := s.getCachedValue(ctx, cacheKey, &innerCached); err == nil && ok {
			return innerCached, nil
		} else if err != nil && s.log != nil {
			s.log.Logf("WARN search embedding cache read failed: %v", err)
		}

		embedCtx, cancel := context.WithTimeout(ctx, embeddingRequestTimeout)
		defer cancel()

		vec, err := s.embedding.Embed(embedCtx, normalizedQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding: %w", err)
		}
		if len(vec) == 0 {
			return nil, fmt.Errorf("empty embedding generated")
		}

		s.setCachedValue(ctx, cacheKey, searchCacheTTL, vec)
		return vec, nil
	})
	if err != nil {
		return nil, err
	}

	embedding, ok := value.([]float32)
	if !ok {
		return nil, fmt.Errorf("unexpected embedding type %T", value)
	}
	return embedding, nil
}

func (s *ServiceImpl) searchEmbeddingCacheKey(normalizedQuery string) string {
	model := "unknown"
	if s.embedding != nil && s.embedding.GetModelName() != "" {
		model = s.embedding.GetModelName()
	}
	return fmt.Sprintf("search:v=%s:model=%s:q=%x", searchCacheVersion, model, sha1.Sum([]byte(normalizedQuery)))
}
