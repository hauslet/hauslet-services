package hooks

import (
	"context"

	leadsservice "hauslet/internal/modules/leads/service"
	messagingservice "hauslet/internal/modules/messaging/service"

	"github.com/google/uuid"
)

type messagingHooksAdapter struct {
	messagingSvc messagingservice.MessagingService
}

func NewMessagingHooksAdapter(messagingSvc messagingservice.MessagingService) leadsservice.MessagingHooks {
	return &messagingHooksAdapter{
		messagingSvc: messagingSvc,
	}
}

func (a *messagingHooksAdapter) EnsureInquiryConversation(ctx context.Context, leadID, requesterID uuid.UUID) error {
	_, err := a.messagingSvc.GetOrCreateInquiryConversation(ctx, leadID, requesterID)
	return err
}
