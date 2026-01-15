package service

import (
	"context"

	"github.com/google/uuid"
)

// MessagingServicePort defines the minimal interface leads needs from messaging.
// This avoids circular imports by not importing the full messaging/service package.
type MessagingServicePort interface {
	GetOrCreateInquiryConversation(ctx context.Context, leadID uuid.UUID, requesterID uuid.UUID) (interface{}, error)
}

type messagingHooksAdapter struct {
	messagingSvc MessagingServicePort
}

func NewMessagingHooksAdapter(messagingSvc MessagingServicePort) MessagingHooks {
	return &messagingHooksAdapter{
		messagingSvc: messagingSvc,
	}
}

func (a *messagingHooksAdapter) EnsureInquiryConversation(ctx context.Context, leadID, requesterID uuid.UUID) error {
	_, err := a.messagingSvc.GetOrCreateInquiryConversation(ctx, leadID, requesterID)
	return err
}
