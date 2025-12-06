package repository

import (
	"context"
	"fmt"
	"hauslet/internal/property/repository/schema"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// =============================================================================
// VECTOR SIMILARITY METHODS (pgvector)
// =============================================================================

// SearchListingsByText performs semantic text search using vector embeddings.
func (r *GormRepository) SearchListingsByText(ctx context.Context, queryVector []float32, filter ListingFilter, similarity SimilarityFilter) ([]ScoredResult[schema.Listing], error) {
	if len(queryVector) == 0 {
		return nil, fmt.Errorf("queryVector is required")
	}
	if similarity.TopK <= 0 {
		similarity.TopK = 10
	}
	threshold := 0.0
	if similarity.MinSimilarity > 0 {
		threshold = 1 - similarity.MinSimilarity
	}

	vectorLit := vectorLiteral(queryVector)

	query := r.db.WithContext(ctx).Table("listings").Where("text_embedding IS NOT NULL")
	query = applyListingFilter(query, filter)

	query = query.Select("listings.*, (1 - (listings.text_embedding <=> ?::vector)) AS similarity", vectorLit)
	if threshold > 0 {
		query = query.Where("(listings.text_embedding <=> ?::vector) <= ?", vectorLit, threshold)
	}
	query = query.Order(clause.Expr{
		SQL:                "listings.text_embedding <=> ?::vector ASC",
		Vars:               []interface{}{vectorLit},
		WithoutParentheses: true,
	}).Limit(similarity.TopK)

	var rows []struct {
		schema.Listing
		Similarity float64 `gorm:"column:similarity"`
	}
	if err := query.Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to search listings by text: %w", err)
	}

	results := make([]ScoredResult[schema.Listing], 0, len(rows))
	for idx, row := range rows {
		results = append(results, ScoredResult[schema.Listing]{
			Item:            row.Listing,
			SimilarityScore: row.Similarity,
			Ranking:         idx + 1,
		})
	}
	return results, nil
}

// FindSimilarListings finds listings similar to a given listing based on text embeddings.
func (r *GormRepository) FindSimilarListings(ctx context.Context, listingID uuid.UUID, limit int, minSimilarity float64) ([]ScoredResult[schema.Listing], error) {
	if limit <= 0 {
		limit = 10
	}
	var embeddingStr string
	if err := r.db.WithContext(ctx).
		Raw("SELECT text_embedding::text FROM listings WHERE id = ? AND text_embedding IS NOT NULL", listingID).
		Scan(&embeddingStr).Error; err != nil {
		return nil, fmt.Errorf("failed to load source embedding: %w", err)
	}
	if embeddingStr == "" {
		return nil, fmt.Errorf("source listing has no embedding")
	}

	threshold := 0.0
	if minSimilarity > 0 {
		threshold = 1 - minSimilarity
	}

	sql := `
		SELECT l.*, (1 - (l.text_embedding <=> ?::vector)) AS similarity
		FROM listings l
		WHERE l.text_embedding IS NOT NULL
		  AND l.id <> ?
	`
	args := []interface{}{embeddingStr, listingID}
	if threshold > 0 {
		sql += " AND (l.text_embedding <=> ?::vector) <= ?"
		args = append(args, embeddingStr, threshold)
	}
	sql += " ORDER BY l.text_embedding <=> ?::vector ASC LIMIT ?"
	args = append(args, embeddingStr, limit)

	var rows []struct {
		schema.Listing
		Similarity float64 `gorm:"column:similarity"`
	}
	if err := r.db.WithContext(ctx).Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to find similar listings: %w", err)
	}

	results := make([]ScoredResult[schema.Listing], 0, len(rows))
	for idx, row := range rows {
		results = append(results, ScoredResult[schema.Listing]{
			Item:            row.Listing,
			SimilarityScore: row.Similarity,
			Ranking:         idx + 1,
		})
	}
	return results, nil
}

// UpdateListingEmbedding updates or sets the text embedding for a listing.
func (r *GormRepository) UpdateListingEmbedding(ctx context.Context, listingID uuid.UUID, embedding []float32, model, version string) error {
	if len(embedding) == 0 {
		return fmt.Errorf("embedding is required")
	}
	updates := map[string]any{
		"text_embedding":         gorm.Expr("?::vector", vectorLiteral(embedding)),
		"embedding_model":        model,
		"embedding_version":      version,
		"embedding_generated_at": time.Now(),
	}
	result := r.db.WithContext(ctx).Model(&schema.Listing{}).
		Where("id = ?", listingID).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update listing embedding: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("listing not found: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

// SearchImagesByVector performs visual similarity search on listing images.
func (r *GormRepository) SearchImagesByVector(ctx context.Context, queryVector []float32, imgFilter ImageSimilarityFilter) ([]ScoredResult[schema.ListingMedia], error) {
	if len(queryVector) == 0 {
		return nil, fmt.Errorf("queryVector is required")
	}
	topK := imgFilter.TopK
	if topK <= 0 {
		topK = 10
	}
	threshold := 0.0
	if imgFilter.MinSimilarity > 0 {
		threshold = 1 - imgFilter.MinSimilarity
	}

	vectorLit := vectorLiteral(queryVector)
	var rows []struct {
		schema.ListingMedia
		Similarity float64 `gorm:"column:similarity"`
	}

	db := r.db.WithContext(ctx).Table("listing_media").Where("image_embedding IS NOT NULL")
	if imgFilter.ListingID != nil {
		db = db.Where("listing_id = ?", *imgFilter.ListingID)
	}
	if len(imgFilter.MediaTypes) > 0 {
		db = db.Where("type IN ?", imgFilter.MediaTypes)
	}
	if imgFilter.IsPrimaryOnly {
		db = db.Where("is_primary = true")
	}

	sql := `
		SELECT lm.*, (1 - (lm.image_embedding <=> ?::vector)) AS similarity
		FROM listing_media lm
		WHERE lm.image_embedding IS NOT NULL
	`
	args := []interface{}{vectorLit}

	if imgFilter.ListingID != nil {
		sql += " AND lm.listing_id = ?"
		args = append(args, *imgFilter.ListingID)
	}
	if len(imgFilter.MediaTypes) > 0 {
		sql += " AND lm.type IN ?"
		args = append(args, imgFilter.MediaTypes)
	}
	if imgFilter.IsPrimaryOnly {
		sql += " AND lm.is_primary = true"
	}
	if threshold > 0 {
		sql += " AND (lm.image_embedding <=> ?::vector) <= ?"
		args = append(args, vectorLit, threshold)
	}
	sql += " ORDER BY lm.image_embedding <=> ?::vector ASC LIMIT ?"
	args = append(args, vectorLit, topK)

	if err := db.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to search images by vector: %w", err)
	}

	results := make([]ScoredResult[schema.ListingMedia], 0, len(rows))
	for idx, row := range rows {
		results = append(results, ScoredResult[schema.ListingMedia]{
			Item:            row.ListingMedia,
			SimilarityScore: row.Similarity,
			Ranking:         idx + 1,
		})
	}
	return results, nil
}

// FindSimilarImages finds images similar to a given image based on visual embeddings.
func (r *GormRepository) FindSimilarImages(ctx context.Context, mediaID uuid.UUID, limit int, minSimilarity float64) ([]ScoredResult[schema.ListingMedia], error) {
	if limit <= 0 {
		limit = 10
	}
	var embeddingStr string
	if err := r.db.WithContext(ctx).
		Raw("SELECT image_embedding::text FROM listing_media WHERE id = ? AND image_embedding IS NOT NULL", mediaID).
		Scan(&embeddingStr).Error; err != nil {
		return nil, fmt.Errorf("failed to load media embedding: %w", err)
	}
	if embeddingStr == "" {
		return nil, fmt.Errorf("source media has no embedding")
	}

	threshold := 0.0
	if minSimilarity > 0 {
		threshold = 1 - minSimilarity
	}

	sql := `
		SELECT lm.*, (1 - (lm.image_embedding <=> ?::vector)) AS similarity
		FROM listing_media lm
		WHERE lm.image_embedding IS NOT NULL
		  AND lm.id <> ?
	`
	args := []interface{}{embeddingStr, mediaID}
	if threshold > 0 {
		sql += " AND (lm.image_embedding <=> ?::vector) <= ?"
		args = append(args, embeddingStr, threshold)
	}
	sql += " ORDER BY lm.image_embedding <=> ?::vector ASC LIMIT ?"
	args = append(args, embeddingStr, limit)

	var rows []struct {
		schema.ListingMedia
		Similarity float64 `gorm:"column:similarity"`
	}
	if err := r.db.WithContext(ctx).Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to find similar images: %w", err)
	}

	results := make([]ScoredResult[schema.ListingMedia], 0, len(rows))
	for idx, row := range rows {
		results = append(results, ScoredResult[schema.ListingMedia]{
			Item:            row.ListingMedia,
			SimilarityScore: row.Similarity,
			Ranking:         idx + 1,
		})
	}
	return results, nil
}

// UpdateMediaEmbedding updates or sets the image embedding for a media item.
func (r *GormRepository) UpdateMediaEmbedding(ctx context.Context, mediaID uuid.UUID, embedding []float32, model, version string) error {
	if len(embedding) == 0 {
		return fmt.Errorf("embedding is required")
	}
	updates := map[string]any{
		"image_embedding":        gorm.Expr("?::vector", vectorLiteral(embedding)),
		"embedding_model":        model,
		"embedding_version":      version,
		"embedding_generated_at": time.Now(),
	}
	result := r.db.WithContext(ctx).Model(&schema.ListingMedia{}).
		Where("id = ?", mediaID).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update media embedding: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("media not found: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

// vectorLiteral formats a float32 slice as a pgvector literal string.
func vectorLiteral(vec []float32) string {
	if len(vec) == 0 {
		return "[]"
	}
	var sb strings.Builder
	sb.WriteString("[")
	for i, v := range vec {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("%g", v))
	}
	sb.WriteString("]")
	return sb.String()
}
