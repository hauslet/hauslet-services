package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"hauslet/internal/modules/review/repository/schema"
)

type ResponseRepositoryImpl struct {
	db *gorm.DB
}

func NewResponseRepository(db *gorm.DB) ResponseRepository {
	return &ResponseRepositoryImpl{db: db}
}

func (r *ResponseRepositoryImpl) Create(ctx context.Context, response *schema.ReviewResponse) error {
	// CREATE (The "One Reply" Rule)
	// 1. Transactional Integrity
	// We use a transaction because creating a response usually implies
	// we want to update the parent review's "ResponseID" pointer for faster fetching later.
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		// 2. Create the Response
		// This will fail with a "duplicate key" error if a response already exists
		// because of the unique index on ReviewID in the schema.
		if err := tx.Create(response).Error; err != nil {
			// Optional: check for specific postgres duplicate error to return a cleaner message
			return err
		}

		// 3. Update the Parent Review
		// We deliberately duplicate the ResponseID onto the parent Review table.
		// Why? It allows us to fetch Review + Response in a single simple JOIN
		// without needing a separate query to check IF a response exists.
		if err := tx.Model(&schema.Review{}).
			Where("id = ?", response.ReviewID).
			Update("response_id", response.ID).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *ResponseRepositoryImpl) GetByReviewID(ctx context.Context, reviewID uuid.UUID) (*schema.ReviewResponse, error) {
	var response schema.ReviewResponse

	// Simple lookup by the Foreign Key
	err := r.db.WithContext(ctx).
		Where("review_id = ?", reviewID).
		First(&response).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil // Return nil if no response exists (common case)
	}

	return &response, err
}

func (r *ResponseRepositoryImpl) Update(ctx context.Context, response *schema.ReviewResponse) error {
	// Limit updates to the Body and UpdatedAt timestamp.
	// Never allow changing AuthorID or ReviewID after creation.
	return r.db.WithContext(ctx).Model(response).
		Updates(map[string]interface{}{
			"body":       response.Body,
			"updated_at": time.Now(),
		}).Error
}

func (r *ResponseRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Get the response to find the ReviewID (needed to clean up the parent)
		var resp schema.ReviewResponse
		if err := tx.First(&resp, "id = ?", id).Error; err != nil {
			return err
		}

		// 2. Remove the pointer from the Parent Review
		if err := tx.Model(&schema.Review{}).
			Where("id = ?", resp.ReviewID).
			Update("response_id", nil).Error; err != nil {
			return err
		}

		// 3. Delete the Response record
		if err := tx.Delete(&resp).Error; err != nil {
			return err
		}

		return nil
	})
}
