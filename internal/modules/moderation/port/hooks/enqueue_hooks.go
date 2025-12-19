package hooks

import (
	"context"

	"hauslet/internal/modules/moderation/domain"
	moderationservice "hauslet/internal/modules/moderation/service"
	propertyservice "hauslet/internal/modules/property/service"

	"github.com/google/uuid"
)

// Ensure compile-time conformance with the property module hooks.
var _ propertyservice.ModerationHooks = (*ModerationAdapter)(nil)

// ModerationAdapter bridges the property module's moderation hooks to the moderation service.
type ModerationAdapter struct {
	svc moderationservice.ModerationService
}

// NewModerationAdapter constructs the adapter.
func NewModerationAdapter(svc moderationservice.ModerationService) *ModerationAdapter {
	return &ModerationAdapter{svc: svc}
}

// EnqueueAIModeration triggers AI moderation for a listing payload.
func (a *ModerationAdapter) EnqueueAIModeration(ctx context.Context, targetID uuid.UUID, contentType, payload string) error {
	_, err := a.svc.EnqueueAIModeration(ctx, moderationservice.CreateModerationRequest{
		ContentType: domain.ContentType(contentType),
		TargetID:    targetID,
		Payload:     payload,
	})
	return err
}
