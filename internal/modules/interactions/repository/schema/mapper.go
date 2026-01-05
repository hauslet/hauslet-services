package schema

import (
	"encoding/json"
	"hauslet/internal/modules/interactions/domain"

	"gorm.io/datatypes"
)

// MapInteractionFromDomain converts domain.Interaction to schema.Interaction
func MapInteractionFromDomain(i *domain.Interaction) *Interaction {
	if i == nil {
		return nil
	}

	// Convert context map to JSON
	var contextJSON datatypes.JSON
	if i.Context != nil {
		contextBytes, _ := json.Marshal(i.Context)
		contextJSON = datatypes.JSON(contextBytes)
	} else {
		contextJSON = datatypes.JSON([]byte("{}"))
	}

	return &Interaction{
		ID:              i.ID,
		UserID:          i.UserID,
		SessionID:       i.SessionID,
		InteractionType: i.Type.String(),
		EntityType:      i.EntityType.String(),
		EntityID:        i.EntityID,
		Context:         contextJSON,
		DeviceType:      string(i.Metadata.DeviceType),
		Platform:        string(i.Metadata.Platform),
		IPHash:          i.Metadata.IPHash,
		UserAgent:       i.Metadata.UserAgent,
		Referrer:        i.Metadata.Referrer,
		IsBot:           i.IsBot,
		CreatedAt:       i.CreatedAt,
	}
}

// MapInteractionToDomain converts schema.Interaction to domain.Interaction
func MapInteractionToDomain(s *Interaction) *domain.Interaction {
	if s == nil {
		return nil
	}

	// Parse context JSON to map
	var context map[string]any
	if len(s.Context) > 0 {
		json.Unmarshal(s.Context, &context)
	}

	return &domain.Interaction{
		ID:         s.ID,
		UserID:     s.UserID,
		SessionID:  s.SessionID,
		Type:       domain.InteractionType(s.InteractionType),
		EntityType: domain.EntityType(s.EntityType),
		EntityID:   s.EntityID,
		Context:    context,
		Metadata: domain.InteractionMeta{
			DeviceType: domain.DeviceType(s.DeviceType),
			Platform:   domain.Platform(s.Platform),
			IPHash:     s.IPHash,
			UserAgent:  s.UserAgent,
			Referrer:   s.Referrer,
		},
		IsBot:     s.IsBot,
		CreatedAt: s.CreatedAt,
	}
}

// MapInteractionsToDomain converts multiple schema.Interaction to domain.Interaction
func MapInteractionsToDomain(schemas []*Interaction) []*domain.Interaction {
	if schemas == nil {
		return nil
	}

	interactions := make([]*domain.Interaction, len(schemas))
	for i, s := range schemas {
		interactions[i] = MapInteractionToDomain(s)
	}
	return interactions
}

// MapAggregateFromDomain converts domain.InteractionAggregate to schema.InteractionAggregate
func MapAggregateFromDomain(a *domain.InteractionAggregate) *InteractionAggregate {
	if a == nil {
		return nil
	}

	return &InteractionAggregate{
		ID:                a.ID,
		EntityType:        a.EntityType.String(),
		EntityID:          a.EntityID,
		PeriodType:        a.PeriodType.String(),
		PeriodStart:       a.PeriodStart,
		ViewsTotal:        a.ViewsTotal,
		ViewsUnique:       a.ViewsUnique,
		SavesTotal:        a.SavesTotal,
		UnsavesTotal:      a.UnsavesTotal,
		SharesTotal:       a.SharesTotal,
		ContactsTotal:     a.ContactsTotal,
		BookingRequests:   a.BookingRequests,
		AvgTimeOnPageSec:  a.AvgTimeOnPageSec,
		EngagementScore:   a.EngagementScore,
		CreatedAt:         a.CreatedAt,
		UpdatedAt:         a.UpdatedAt,
	}
}

// MapAggregateToDomain converts schema.InteractionAggregate to domain.InteractionAggregate
func MapAggregateToDomain(s *InteractionAggregate) *domain.InteractionAggregate {
	if s == nil {
		return nil
	}

	return &domain.InteractionAggregate{
		ID:                s.ID,
		EntityType:        domain.EntityType(s.EntityType),
		EntityID:          s.EntityID,
		PeriodType:        domain.PeriodType(s.PeriodType),
		PeriodStart:       s.PeriodStart,
		ViewsTotal:        s.ViewsTotal,
		ViewsUnique:       s.ViewsUnique,
		SavesTotal:        s.SavesTotal,
		UnsavesTotal:      s.UnsavesTotal,
		SharesTotal:       s.SharesTotal,
		ContactsTotal:     s.ContactsTotal,
		BookingRequests:   s.BookingRequests,
		AvgTimeOnPageSec:  s.AvgTimeOnPageSec,
		EngagementScore:   s.EngagementScore,
		CreatedAt:         s.CreatedAt,
		UpdatedAt:         s.UpdatedAt,
	}
}

// MapAggregatesToDomain converts multiple schema.InteractionAggregate to domain.InteractionAggregate
func MapAggregatesToDomain(schemas []*InteractionAggregate) []*domain.InteractionAggregate {
	if schemas == nil {
		return nil
	}

	aggregates := make([]*domain.InteractionAggregate, len(schemas))
	for i, s := range schemas {
		aggregates[i] = MapAggregateToDomain(s)
	}
	return aggregates
}
