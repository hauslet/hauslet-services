package domain

import "errors"

// Domain errors for discovery module
var (
	// ErrInvalidSearchFilter is returned when search filter is invalid
	ErrInvalidSearchFilter = errors.New("invalid search filter")

	// ErrInvalidRankingConfig is returned when ranking configuration is invalid
	ErrInvalidRankingConfig = errors.New("invalid ranking configuration")

	// ErrNoResults is returned when no results are found
	ErrNoResults = errors.New("no results found")

	// ErrInvalidFeedSection is returned when feed section type is invalid
	ErrInvalidFeedSection = errors.New("invalid feed section type")

	// ErrSearchHistoryNotFound is returned when search history is not found
	ErrSearchHistoryNotFound = errors.New("search history not found")

	// ErrPreferencesNotFound is returned when user preferences are not found
	ErrPreferencesNotFound = errors.New("user preferences not found")
)
