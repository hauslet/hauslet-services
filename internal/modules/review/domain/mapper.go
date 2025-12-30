package domain

import (
	"encoding/json"
	"time"

	"hauslet/internal/modules/review/repository/schema"
)

// MapReviewFromSchema converts schema.Review to domain.Review
func MapReviewFromSchema(s *schema.Review) *Review {
	if s == nil {
		return nil
	}

	var subRatings *SubRatings
	if len(s.SubRatings) > 0 {
		subRatings = &SubRatings{}
		json.Unmarshal(s.SubRatings, subRatings)
	}

	var moderationReason *ModerationReason
	if s.ModerationReason != nil {
		reason := ModerationReason(*s.ModerationReason)
		moderationReason = &reason
	}

	var deletedAt *time.Time
	if s.DeletedAt.Valid {
		deletedAt = &s.DeletedAt.Time
	}

	return &Review{
		ID:                  s.ID,
		BookingID:           s.BookingID,
		TargetType:          ReviewTargetType(s.TargetType),
		TargetID:            s.TargetID,
		ReviewerID:          s.ReviewerID,
		ReviewerCountryCode: s.ReviewerCountryCode,
		Rating:              s.Rating,
		Title:               s.Title,
		Body:                s.Body,
		Language:            s.Language,
		SubRatings:          subRatings,
		Status:              ReviewStatus(s.Status),
		IsReported:          s.IsReported,
		ModerationReason:    moderationReason,
		ResponseID:          s.ResponseID,
		PublishedAt:         s.PublishedAt,
		CreatedAt:           s.CreatedAt,
		UpdatedAt:           s.UpdatedAt,
		DeletedAt:           deletedAt,
	}
}

// MapReviewToSchema converts domain.Review to schema.Review
func MapReviewToSchema(d *Review) *schema.Review {
	if d == nil {
		return nil
	}

	var subRatingsJSON []byte
	if d.SubRatings != nil {
		subRatingsJSON, _ = json.Marshal(d.SubRatings)
	}

	var moderationReason *string
	if d.ModerationReason != nil {
		reason := d.ModerationReason.String()
		moderationReason = &reason
	}

	return &schema.Review{
		ID:                  d.ID,
		BookingID:           d.BookingID,
		TargetType:          schema.ReviewTargetType(d.TargetType),
		TargetID:            d.TargetID,
		ReviewerID:          d.ReviewerID,
		ReviewerCountryCode: d.ReviewerCountryCode,
		Rating:              d.Rating,
		Title:               d.Title,
		Body:                d.Body,
		Language:            d.Language,
		SubRatings:          subRatingsJSON,
		Status:              schema.ReviewStatus(d.Status),
		IsReported:          d.IsReported,
		ModerationReason:    moderationReason,
		ResponseID:          d.ResponseID,
		PublishedAt:         d.PublishedAt,
		CreatedAt:           d.CreatedAt,
		UpdatedAt:           d.UpdatedAt,
	}
}

// MapReviewResponseFromSchema converts schema.ReviewResponse to domain.ReviewResponse
func MapReviewResponseFromSchema(s *schema.ReviewResponse) *ReviewResponse {
	if s == nil {
		return nil
	}

	return &ReviewResponse{
		ID:        s.ID,
		ReviewID:  s.ReviewID,
		AuthorID:  s.AuthorID,
		Body:      s.Body,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

// MapReviewResponseToSchema converts domain.ReviewResponse to schema.ReviewResponse
func MapReviewResponseToSchema(d *ReviewResponse) *schema.ReviewResponse {
	if d == nil {
		return nil
	}

	return &schema.ReviewResponse{
		ID:        d.ID,
		ReviewID:  d.ReviewID,
		AuthorID:  d.AuthorID,
		Body:      d.Body,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

// MapListingStatsFromSchema converts schema.ListingStats to domain.ListingStats
func MapListingStatsFromSchema(s *schema.ListingStats) *ListingStats {
	if s == nil {
		return nil
	}

	var subRatingAverages *SubRatings
	if len(s.SubRatingAverages) > 0 {
		subRatingAverages = &SubRatings{}
		json.Unmarshal(s.SubRatingAverages, subRatingAverages)
	}

	var ratingDistribution *RatingDistribution
	if len(s.RatingDistribution) > 0 {
		// Parse the JSON structure: {"5": 100, "4": 12, "3": 1, "2": 0, "1": 2}
		var distMap map[string]int
		if err := json.Unmarshal(s.RatingDistribution, &distMap); err == nil {
			ratingDistribution = &RatingDistribution{
				FiveStarCount:  distMap["5"],
				FourStarCount:  distMap["4"],
				ThreeStarCount: distMap["3"],
				TwoStarCount:   distMap["2"],
				OneStarCount:   distMap["1"],
			}
		}
	}

	return &ListingStats{
		ListingID:          s.ListingID,
		AverageRating:      s.AverageRating,
		ReviewCount:        s.ReviewCount,
		SubRatingAverages:  subRatingAverages,
		RatingDistribution: ratingDistribution,
		LastUpdatedAt:      s.LastUpdatedAt,
	}
}

// MapListingStatsToSchema converts domain.ListingStats to schema.ListingStats
func MapListingStatsToSchema(d *ListingStats) *schema.ListingStats {
	if d == nil {
		return nil
	}

	var subRatingAveragesJSON []byte
	if d.SubRatingAverages != nil {
		subRatingAveragesJSON, _ = json.Marshal(d.SubRatingAverages)
	}

	var ratingDistributionJSON []byte
	if d.RatingDistribution != nil {
		// Convert to the JSON structure: {"5": 100, "4": 12, "3": 1, "2": 0, "1": 2}
		distMap := map[string]int{
			"5": d.RatingDistribution.FiveStarCount,
			"4": d.RatingDistribution.FourStarCount,
			"3": d.RatingDistribution.ThreeStarCount,
			"2": d.RatingDistribution.TwoStarCount,
			"1": d.RatingDistribution.OneStarCount,
		}
		ratingDistributionJSON, _ = json.Marshal(distMap)
	}

	return &schema.ListingStats{
		ListingID:          d.ListingID,
		AverageRating:      d.AverageRating,
		ReviewCount:        d.ReviewCount,
		SubRatingAverages:  subRatingAveragesJSON,
		RatingDistribution: ratingDistributionJSON,
		LastUpdatedAt:      d.LastUpdatedAt,
	}
}

// MapHostStatsFromSchema converts schema.HostStats to domain.HostStats
func MapHostStatsFromSchema(s *schema.HostStats) *HostStats {
	if s == nil {
		return nil
	}

	return &HostStats{
		HostID:                 s.HostID,
		GlobalAverageRating:    s.GlobalAverageRating,
		TotalReviewCount:       s.TotalReviewCount,
		ReviewCountLast365Days: s.ReviewCountLast365Days,
		LastUpdatedAt:          s.LastUpdatedAt,
	}
}

// MapHostStatsToSchema converts domain.HostStats to schema.HostStats
func MapHostStatsToSchema(d *HostStats) *schema.HostStats {
	if d == nil {
		return nil
	}

	return &schema.HostStats{
		HostID:                 d.HostID,
		GlobalAverageRating:    d.GlobalAverageRating,
		TotalReviewCount:       d.TotalReviewCount,
		ReviewCountLast365Days: d.ReviewCountLast365Days,
		LastUpdatedAt:          d.LastUpdatedAt,
	}
}
