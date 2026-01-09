# Events - Redis Pub/Sub Event System

This package provides a Redis-based pub/sub event system for GraphQL subscriptions and real-time updates.

## Architecture

```
Service Layer (publishes events)
    ↓
Redis Pub/Sub (broker)
    ↓
GraphQL Resolvers (subscribe & filter)
    ↓
WebSocket Clients
```

## Components

### 1. Broker (`broker.go`)
Low-level Redis pub/sub implementation. Handles publishing and subscribing to Redis channels.

### 2. Publisher (`publisher.go`)
Service-friendly wrapper for publishing events. Used by service layer to emit domain events.

### 3. Subscriber (`subscriber.go`)
Resolver-friendly wrapper for subscribing with filters. Used by GraphQL subscription resolvers.

### 4. Types (`types.go`)
Event types, channels, and domain event definitions.

## Usage

### Initialization (in `cmd/api/main.go`)

```go
// Create Redis client (already exists)
redisClient := redis.NewClient(&redis.Options{
    Addr: cfg.Storage.Redis.Addr,
})

// Create event broker
eventBroker := events.NewRedisBroker(redisClient, log)
defer eventBroker.Close()

// Create publisher for services
eventPublisher := events.NewPublisher(eventBroker, log)

// Create subscriber for GraphQL resolvers
eventSubscriber := events.NewSubscriber(eventBroker, log)

// Pass to services (for publishing)
bookingService := bookingservice.New(
    bookingRepo,
    eventPublisher,  // Add this parameter
    // ... other deps
)

// Pass to GraphQL (for subscribing)
graph.SetupGraphQL(
    r,
    // ... all services
    eventSubscriber,  // Add this parameter
    cfg,
    log,
)
```

### Publishing Events (Service Layer)

In `internal/modules/booking/service/service.go`:

```go
type service struct {
    repo           BookingRepository
    eventPublisher *events.Publisher  // Add this field
    // ... other fields
}

func New(repo BookingRepository, eventPublisher *events.Publisher, ...) BookingService {
    return &service{
        repo:           repo,
        eventPublisher: eventPublisher,
        // ...
    }
}

func (s *service) CreateBooking(ctx context.Context, input CreateBookingInput) (*domain.Booking, error) {
    // 1. Authorization
    if err := s.CheckPermission(ctx, ...); err != nil {
        return nil, err
    }

    // 2. Business logic
    booking, err := s.repo.Create(ctx, ...)
    if err != nil {
        return nil, err
    }

    // 3. Publish event (fire-and-forget, logged if fails)
    s.eventPublisher.PublishBookingEvent(
        ctx,
        events.EventBookingCreated,
        booking.ID.String(),
        booking,
        &booking.BusinessID,  // tenantID
    )

    return booking, nil
}
```

### Subscribing to Events (GraphQL Resolvers)

In `internal/transport/graph/schema.resolvers.go`:

```graphql
# schema.graphqls
type Subscription {
    bookingUpdated(propertyID: ID!): Booking!
}
```

```go
// Resolver struct
type Resolver struct {
    // ... existing services
    eventSubscriber *events.Subscriber  // Add this field
}

func NewResolver(..., eventSubscriber *events.Subscriber) *Resolver {
    return &Resolver{
        // ...
        eventSubscriber: eventSubscriber,
    }
}

// Subscription resolver
func (r *subscriptionResolver) BookingUpdated(ctx context.Context, propertyID string) (<-chan *model.Booking, error) {
    // 1. Get viewer for authorization
    viewer := viewer.FromContext(ctx)
    if viewer == nil {
        return nil, errors.New("unauthorized")
    }

    // 2. Subscribe with filters
    eventCh, err := r.eventSubscriber.SubscribeWithFilter(
        ctx,
        events.ChannelBookings,
        events.FilterByType(events.EventBookingUpdated),  // Only updates
    )
    if err != nil {
        return nil, err
    }

    // 3. Create output channel
    outCh := make(chan *model.Booking)

    // 4. Process events
    go func() {
        defer close(outCh)

        for {
            select {
            case <-ctx.Done():
                return

            case event, ok := <-eventCh:
                if !ok {
                    return
                }

                // Parse payload
                var booking domain.Booking
                json.Unmarshal(event.Payload, &booking)

                // Filter by propertyID
                if booking.PropertyID.String() != propertyID {
                    continue
                }

                // CRITICAL: Re-authorize for EVERY event
                // Check if viewer can see this booking
                if !r.bookingService.CanViewBooking(ctx, viewer.ID, booking.ID) {
                    continue
                }

                // Convert and send
                outCh <- toGraphQLModel(&booking)
            }
        }
    }()

    return outCh, nil
}
```

## Event Types

See `types.go` for all available event types. Events are organized by domain:

- **Bookings**: `EventBookingCreated`, `EventBookingUpdated`, `EventBookingCanceled`
- **Payments**: `EventPaymentProcessed`, `EventPaymentFailed`, `EventPayoutCompleted`
- **Properties**: `EventPropertyCreated`, `EventPropertyUpdated`
- **Reviews**: `EventReviewCreated`
- **Leads**: `EventLeadCreated`, `EventLeadUpdated`
- **Verifications**: `EventVerificationStarted`, `EventVerificationCompleted`, `EventVerificationFailed`

## Channels

Events are published to topic-based channels:

- `ChannelBookings`: All booking events
- `ChannelPayments`: All payment events
- `ChannelProperties`: All property events
- `ChannelReviews`: All review events
- `ChannelLeads`: All lead events
- `ChannelVerifications`: All verification events

## Filtering

Use built-in filters for common cases:

```go
// Filter by event type
events.FilterByType(events.EventBookingCreated, events.EventBookingUpdated)

// Filter by entity ID
events.FilterByEntityID(bookingID)

// Filter by tenant
events.FilterByTenant(businessID)

// Filter by actor (user who triggered event)
events.FilterByActor(userID)

// Custom filter
events.FilterCustom(func(e *events.Event) bool {
    // Custom logic
    return true
})
```

## Security

**CRITICAL**: Always re-authorize events in subscription resolvers. Never trust that a client should receive an event just because they subscribed to the channel.

```go
// BAD - No authorization
for event := range eventCh {
    outCh <- event  // Sends everything!
}

// GOOD - Authorize every event
for event := range eventCh {
    if !canViewEvent(ctx, viewer, event) {
        continue  // Skip unauthorized events
    }
    outCh <- event
}
```

## Error Handling

- **Publishing**: Failures are logged but don't fail the operation (subscriptions are optional)
- **Subscribing**: Connection failures return errors immediately
- **Redis down**: Subscribers will disconnect, publishers will log errors

## Testing

Mock the `Broker` interface for unit tests:

```go
type mockBroker struct {
    events []*events.Event
}

func (m *mockBroker) Publish(ctx context.Context, ch events.Channel, e *events.Event) error {
    m.events = append(m.events, e)
    return nil
}
```

## Performance

- Events are buffered (10 capacity) to prevent blocking
- Slow subscribers are automatically dropped after 5s
- Multiple API instances automatically share events via Redis
