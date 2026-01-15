package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"hauslet/internal/modules/messaging/domain"
	"hauslet/internal/modules/messaging/repository/schema"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

const conversationStatusActive = "active"

func (s *messagingServiceImpl) GetOrCreateInquiryConversation(ctx context.Context, leadID, requesterID uuid.UUID) (*domain.Conversation, error) {
    return s.getOrCreateInquiryConversation(ctx, leadID, requesterID, false)
}

func (s *messagingServiceImpl) getOrCreateInquiryConversation(ctx context.Context, leadID, requesterID uuid.UUID, skipLeadAccess bool) (*domain.Conversation, error) {
    if leadID == uuid.Nil || requesterID == uuid.Nil {
        return nil, fmt.Errorf("lead_id and requester_id are required")
    }

    if s.leadHooks == nil {
        return nil, fmt.Errorf("lead hooks not configured")
    }

    if !skipLeadAccess {
        allowed, err := s.leadHooks.ValidateLeadAccess(ctx, leadID, requesterID)
        if err != nil {
            return nil, fmt.Errorf("failed to validate lead access: %w", err)
        }
        if !allowed {
            return nil, domain.ErrUnauthorizedAccess
        }
    }

    existing, err := s.convRepo.GetByContext(ctx, string(domain.ContextTypeLead), leadID)
    if err != nil {
        return nil, fmt.Errorf("failed to query conversation for lead %s: %w", leadID, err)
    }
    if existing != nil {
        return s.ensureConversationAccess(existing, requesterID)
    }

    prospectID, ownerID, err := s.leadHooks.GetLeadParticipants(ctx, leadID)
    if err != nil {
        return nil, fmt.Errorf("failed to resolve lead participants: %w", err)
    }
    if prospectID == uuid.Nil || ownerID == uuid.Nil {
        return nil, fmt.Errorf("invalid participants for lead %s", leadID)
    }

    convID := uuid.New()
    participants := buildParticipants(convID, []participantSeed{
        {userID: prospectID, typ: domain.ParticipantTypeGuest},
        {userID: ownerID, typ: domain.ParticipantTypeHost},
    })
    if len(participants) == 0 {
        return nil, fmt.Errorf("no participants could be created for lead %s", leadID)
    }

    conversation := &schema.Conversation{
        ID:           convID,
        Type:         string(domain.ConversationTypeInquiry),
        Status:       conversationStatusActive,
        ContextType:  string(domain.ContextTypeLead),
        ContextID:    leadID,
        Participants: participants,
    }

    if err := s.convRepo.Create(ctx, conversation); err != nil {
        return nil, fmt.Errorf("failed to create inquiry conversation: %w", err)
    }

    return s.ensureConversationAccess(conversation, requesterID)
}

func (s *messagingServiceImpl) GetOrCreateTransactionConversation(ctx context.Context, contextType domain.ConversationContextType, contextID, requesterID uuid.UUID) (*domain.Conversation, error) {
	if contextID == uuid.Nil || requesterID == uuid.Nil {
		return nil, fmt.Errorf("context_id and requester_id are required")
	}
	if contextType == "" {
		return nil, fmt.Errorf("context_type is required")
	}

	existing, err := s.convRepo.GetByContext(ctx, string(contextType), contextID)
	if err != nil {
		return nil, fmt.Errorf("failed to query conversation: %w", err)
	}
	if existing != nil {
		return s.ensureConversationAccess(existing, requesterID)
	}

	seeds, err := s.transactionParticipants(ctx, contextType, contextID)
	if err != nil {
		return nil, err
	}
	if len(seeds) == 0 {
		return nil, fmt.Errorf("no participants resolved for context %s/%s", contextType, contextID)
	}

	isParticipant := false
	for _, seed := range seeds {
		if seed.userID == requesterID {
			isParticipant = true
			break
		}
	}
	if !isParticipant {
		return nil, domain.ErrUnauthorizedAccess
	}

	convID := uuid.New()
	participants := buildParticipants(convID, seeds)
	if len(participants) == 0 {
		return nil, fmt.Errorf("no participants available for conversation")
	}

	conversation := &schema.Conversation{
		ID:           convID,
		Type:         string(domain.ConversationTypeTransaction),
		Status:       conversationStatusActive,
		ContextType:  string(contextType),
		ContextID:    contextID,
		Participants: participants,
	}

	if err := s.convRepo.Create(ctx, conversation); err != nil {
		return nil, fmt.Errorf("failed to create transaction conversation: %w", err)
	}

	return s.ensureConversationAccess(conversation, requesterID)
}

func (s *messagingServiceImpl) GetOrCreateSupportConversation(ctx context.Context, userID uuid.UUID) (*domain.Conversation, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user_id is required")
	}

	existing, err := s.convRepo.GetSupportConversation(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup support conversation: %w", err)
	}
	if existing != nil {
		return s.ensureConversationAccess(existing, userID)
	}

	statePayload, err := json.Marshal(domain.NewSupportState())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal support state: %w", err)
	}

	convID := uuid.New()
	participants := buildParticipants(convID, []participantSeed{
		{userID: userID, typ: domain.ParticipantTypeUser},
	})
	if len(participants) == 0 {
		return nil, fmt.Errorf("failed to create support participants for user %s", userID)
	}

	conversation := &schema.Conversation{
		ID:           convID,
		Type:         string(domain.ConversationTypeSupport),
		Status:       conversationStatusActive,
		ContextType:  string(domain.ContextTypeNone),
		ContextID:    userID,
		SupportState: datatypes.JSON(statePayload),
		Participants: participants,
	}

	if err := s.convRepo.Create(ctx, conversation); err != nil {
		return nil, fmt.Errorf("failed to create support conversation: %w", err)
	}

	return s.ensureConversationAccess(conversation, userID)
}

func (s *messagingServiceImpl) transactionParticipants(ctx context.Context, contextType domain.ConversationContextType, contextID uuid.UUID) ([]participantSeed, error) {
	if contextType != domain.ContextTypeBooking {
		return nil, fmt.Errorf("unsupported transaction context type: %s", contextType)
	}
	if s.bookingHooks == nil {
		return nil, fmt.Errorf("booking hooks not configured")
	}

	guestID, hostID, err := s.bookingHooks.GetBookingParticipants(ctx, contextID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve booking participants: %w", err)
	}

	seeds := []participantSeed{
		{userID: guestID, typ: domain.ParticipantTypeGuest},
	}
	if hostID != uuid.Nil {
		seeds = append(seeds, participantSeed{userID: hostID, typ: domain.ParticipantTypeHost})
	}

	return seeds, nil
}

func (s *messagingServiceImpl) ensureConversationAccess(conv *schema.Conversation, requesterID uuid.UUID) (*domain.Conversation, error) {
	domainConv, err := domain.MapConversationToDomain(conv)
	if err != nil {
		return nil, fmt.Errorf("failed to map conversation: %w", err)
	}
	if domainConv == nil {
		return nil, domain.ErrConversationNotFound
	}
	if !domainConv.CanUserParticipate(requesterID) {
		return nil, domain.ErrUnauthorizedAccess
	}
	return domainConv, nil
}

type participantSeed struct {
	userID uuid.UUID
	typ    domain.ParticipantType
}

func buildParticipants(convID uuid.UUID, seeds []participantSeed) []schema.Participant {
	now := time.Now()
	seen := make(map[uuid.UUID]struct{}, len(seeds))
	participants := make([]schema.Participant, 0, len(seeds))

	for _, seed := range seeds {
		if seed.userID == uuid.Nil {
			continue
		}
		if _, ok := seen[seed.userID]; ok {
			continue
		}
		seen[seed.userID] = struct{}{}

		participants = append(participants, schema.Participant{
			ID:             uuid.New(),
			ConversationID: convID,
			UserID:         seed.userID,
			Type:           string(seed.typ),
			JoinedAt:       now,
			IsVisible:      true,
		})
	}

	return participants
}
