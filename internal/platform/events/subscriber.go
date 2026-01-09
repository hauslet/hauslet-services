package events

import (
	"context"
	"log/slog"
)

// Subscriber provides utilities for subscribing to events with filtering.
// This is designed for use in GraphQL subscription resolvers.
type Subscriber struct {
	broker Broker
	log    *slog.Logger
}

// NewSubscriber creates a new event subscriber.
func NewSubscriber(broker Broker, log *slog.Logger) *Subscriber {
	return &Subscriber{
		broker: broker,
		log:    log,
	}
}

// FilterFunc is a predicate function that determines whether an event should be delivered.
// Return true to deliver the event, false to filter it out.
type FilterFunc func(*Event) bool

// SubscribeWithFilter subscribes to a channel and applies filters to incoming events.
// Only events that pass all filters will be delivered to the output channel.
func (s *Subscriber) SubscribeWithFilter(ctx context.Context, channel Channel, filters ...FilterFunc) (<-chan *Event, error) {
	// Subscribe to the base channel
	eventCh, err := s.broker.Subscribe(ctx, channel)
	if err != nil {
		return nil, err
	}

	// Optimization: If no filters, return the raw channel directly
	// to avoid the overhead of the extra goroutine.
	if len(filters) == 0 {
		return eventCh, nil
	}

	// Create filtered output channel
	filteredCh := make(chan *Event, 10)

	// Start filtering goroutine
	go func() {
		defer close(filteredCh)

		for {
			select {
			case <-ctx.Done():
				return

			case event, ok := <-eventCh:
				if !ok {
					// Source channel closed
					return
				}

				// IMPROVEMENT: Defensive check against nil events
				if event == nil {
					continue
				}

				// Apply all filters
				passedAllFilters := true
				for _, filter := range filters {
					if !filter(event) {
						passedAllFilters = false
						// IMPROVEMENT: Removed "Event filtered out" log.
						// In high-throughput systems, logging dropped events creates
						// massive noise and performance degradation.
						break
					}
				}

				// Deliver if passed all filters
				if passedAllFilters {
					select {
					case filteredCh <- event:
						s.log.Debug("Filtered event delivered",
							"channel", channel,
							"event_type", event.Type,
							"event_id", event.ID,
						)
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return filteredCh, nil
}

// Common filter constructors for convenience.

// FilterByType creates a filter that only allows events of specific types.
func FilterByType(eventTypes ...EventType) FilterFunc {
	// Optimization: Pre-allocate map for O(1) lookup
	typeMap := make(map[EventType]struct{}, len(eventTypes))
	for _, t := range eventTypes {
		typeMap[t] = struct{}{}
	}

	return func(e *Event) bool {
		_, exists := typeMap[e.Type]
		return exists
	}
}

// FilterByEntityID creates a filter that only allows events for a specific entity.
func FilterByEntityID(entityID string) FilterFunc {
	return func(e *Event) bool {
		return e.EntityID == entityID
	}
}

// FilterByTenant creates a filter that only allows events for a specific tenant.
func FilterByTenant(tenantID string) FilterFunc {
	return func(e *Event) bool {
		return e.TenantID != nil && *e.TenantID == tenantID
	}
}

// FilterByActor creates a filter that only allows events triggered by a specific user.
func FilterByActor(actorID string) FilterFunc {
	return func(e *Event) bool {
		return e.ActorID != nil && *e.ActorID == actorID
	}
}

// FilterByMetadata creates a filter that checks for a specific metadata key-value pair.
func FilterByMetadata(key, value string) FilterFunc {
	return func(e *Event) bool {
		if e.Metadata == nil {
			return false
		}
		// Maps in Go return zero value (empty string) if key is missing,
		// so this safely handles missing keys unless value is also "".
		v, ok := e.Metadata[key]
		return ok && v == value
	}
}

// FilterCustom creates a custom filter from a predicate function.
// This is for complex authorization or business logic filtering.
func FilterCustom(predicate func(*Event) bool) FilterFunc {
	return predicate
}

// Health checks if the subscriber's broker is healthy.
func (s *Subscriber) Health(ctx context.Context) error {
	return s.broker.Health(ctx)
}
