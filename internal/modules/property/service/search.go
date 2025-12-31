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

	"github.com/google/uuid"
)

const (
	defaultSearchLimit = 20
	maxSearchLimit     = 50
	searchCacheTTL     = 24 * time.Hour

	searchCacheVersion      = "v1"
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
			s.log.Warn("embedding client not configured, falling back to filtered search")
		}
		return s.searchWithFiltersOnly(ctx, repoFilter, limit)
	}

	embedding, err := s.getOrCreateQueryEmbedding(ctx, normalized)
	if err != nil {
		if s.log != nil {
			s.log.Warn("semantic embedding unavailable, falling back to filtered search: %v", err)
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
		s.log.Warn("search embedding cache read failed: %v", err)
	}

	value, err, _ := s.embeddingGroup.Do(cacheKey, func() (any, error) {
		var innerCached []float32
		if ok, err := s.getCachedValue(ctx, cacheKey, &innerCached); err == nil && ok {
			return innerCached, nil
		} else if err != nil && s.log != nil {
			s.log.Warn("search embedding cache read failed: %v", err)
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

// FindSimilarListings finds listings similar to the given listing using vector similarity.
func (s *ServiceImpl) FindSimilarListings(ctx context.Context, listingID uuid.UUID, limit int, minSimilarity float64) ([]domain.ScoredListing, error) {
	// Validate and set defaults
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}

	// Get the source listing with its embedding
	sourceListing, err := s.GetListingByID(ctx, listingID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get source listing: %w", err)
	}

	// Get or generate the embedding for the source listing
	var embedding []float32
	listingSchema := domain.MapListingToSchema(sourceListing)

	if listingSchema.TextEmbedding != nil && len(listingSchema.TextEmbedding.Vector) > 0 {
		// Use existing embedding
		embedding = listingSchema.TextEmbedding.Vector
	} else {
		// Generate embedding on-the-fly
		if s.embedding == nil {
			return nil, fmt.Errorf("embedding client not configured")
		}

		// Get property for building complete embedding document
		property, err := s.GetPropertyByID(ctx, sourceListing.PropertyID)
		if err != nil {
			return nil, fmt.Errorf("failed to get property for embedding generation: %w", err)
		}

		// Build embedding document
		doc := domain.NewEmbeddingDocumentBuilder().
			WithListing(sourceListing).
			WithProperty(property).
			Build()

		// Generate embedding
		embedCtx, cancel := context.WithTimeout(ctx, embeddingRequestTimeout)
		defer cancel()

		vec, err := s.embedding.Embed(embedCtx, doc.Text)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding for source listing: %w", err)
		}
		if len(vec) == 0 {
			return nil, fmt.Errorf("empty embedding generated for source listing")
		}
		embedding = vec
	}

	// Create filter for published, active listings only (SECURITY)
	pub := true
	serviceFilter := ListingFilter{
		Published:      &pub,
		Statuses:       []domain.ListingStatus{domain.StatusActive},
		IncludeDeleted: false,
	}
	repoFilter := mapListingFilterToRepo(serviceFilter)

	// Search for similar listings
	results, err := s.repo.SearchListings(ctx, schema.NewVectorEmbedding(embedding), repoFilter, limit+1)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar listings: %w", err)
	}

	// Filter results
	scored := make([]domain.ScoredListing, 0, len(results))
	rank := 1
	for _, item := range results {
		// Exclude the original listing
		if item.Listing.ID == listingID {
			continue
		}

		// Apply minSimilarity threshold
		// pgvector returns cosine distance (0-2 range, lower is better)
		// Convert to similarity: similarity = 1 - (distance / 2)
		if item.Score > 0 {
			similarity := 1.0 - (item.Score / 2.0)
			if similarity < minSimilarity {
				continue
			}
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
