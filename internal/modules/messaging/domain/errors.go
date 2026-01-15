package domain

import "errors"

var (
	ErrConversationNotFound = errors.New("conversation not found")
	ErrUnauthorizedAccess   = errors.New("user is not authorized to access this conversation")
	ErrMessageEmpty         = errors.New("message content cannot be empty")
	ErrInvalidParticipant   = errors.New("invalid participant")
	ErrConversationClosed   = errors.New("conversation is closed or archived")
	ErrAIServiceUnavailable = errors.New("ai service is currently unavailable")
)
 