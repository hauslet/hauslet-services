package domain

import "errors"

var (
	// Review-related errors
	ErrReviewNotFound           = errors.New("review not found")
	ErrReviewAlreadyExists      = errors.New("review already exists for this booking")
	ErrReviewNotEditable        = errors.New("review cannot be edited after publication")
	ErrInvalidRating            = errors.New("rating must be between 1 and 5")
	ErrInvalidSubRatings        = errors.New("sub-ratings must be between 1.0 and 5.0")
	ErrReviewBodyRequired       = errors.New("review body is required")
	ErrReviewAlreadyPublished   = errors.New("review is already published")
	ErrReviewAlreadyHidden      = errors.New("review is already hidden")
	ErrCannotPublishHiddenReview = errors.New("cannot publish a hidden review")

	// Response-related errors
	ErrResponseNotFound         = errors.New("review response not found")
	ErrResponseAlreadyExists    = errors.New("response already exists for this review")
	ErrResponseBodyRequired     = errors.New("response body is required")
	ErrUnauthorizedToRespond    = errors.New("only the host can respond to this review")
	ErrCannotRespondToUnpublished = errors.New("cannot respond to an unpublished review")

	// Authorization errors
	ErrUnauthorized            = errors.New("unauthorized to perform this action")
	ErrNotReviewAuthor         = errors.New("only the review author can perform this action")

	// Moderation errors
	ErrModerationReasonRequired = errors.New("moderation reason is required when hiding a review")

	// Stats errors
	ErrStatsNotFound = errors.New("statistics not found")
)
