package push

import "errors"

var (
	// ErrProviderUnavailable indicates the push notification provider is unreachable
	ErrProviderUnavailable = errors.New("push notification provider is currently unavailable")

	// ErrInvalidToken indicates the device token is invalid or expired
	ErrInvalidToken = errors.New("invalid device token")

	// ErrInvalidPayload indicates the notification payload is invalid
	ErrInvalidPayload = errors.New("invalid notification payload")

	// ErrRateLimited indicates too many requests to provider
	ErrRateLimited = errors.New("push notification rate limit exceeded")

	// ErrAllProvidersFailed indicates all providers failed (for future multi-provider support)
	ErrAllProvidersFailed = errors.New("all push notification providers failed")

	// ErrInvalidTopic indicates the topic name is invalid
	ErrInvalidTopic = errors.New("invalid topic name")

	// ErrNoTokensProvided indicates no device tokens were provided
	ErrNoTokensProvided = errors.New("no device tokens provided")
)
