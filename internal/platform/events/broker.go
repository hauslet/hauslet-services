package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Broker defines the interface for publishing and subscribing to events.
type Broker interface {
	// Publish sends an event to a channel.
	Publish(ctx context.Context, channel Channel, event *Event) error

	// Subscribe creates a subscription to a channel and returns a channel of events.
	// The subscription remains active until the context is cancelled or an error occurs.
	Subscribe(ctx context.Context, channel Channel) (<-chan *Event, error)

	// Close gracefully shuts down the broker and all active subscriptions.
	Close() error

	// Health checks if the broker is healthy and can communicate with Redis.
	Health(ctx context.Context) error
}

// RedisBroker implements the Broker interface using Redis pub/sub.
type RedisBroker struct {
	client *redis.Client
	log    *slog.Logger

	// Track active subscriptions for cleanup
	mu            sync.RWMutex
	subscriptions map[string]context.CancelFunc
	closed        bool
}

// NewRedisBroker creates a new Redis-based event broker.
func NewRedisBroker(client *redis.Client, log *slog.Logger) *RedisBroker {
	return &RedisBroker{
		client:        client,
		log:           log,
		subscriptions: make(map[string]context.CancelFunc),
	}
}

// Publish sends an event to a Redis channel.
func (b *RedisBroker) Publish(ctx context.Context, channel Channel, event *Event) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return fmt.Errorf("broker is closed")
	}
	b.mu.RUnlock()

	// Marshal event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Publish to Redis channel
	if err := b.client.Publish(ctx, channel.String(), data).Err(); err != nil {
		return fmt.Errorf("failed to publish to channel %s: %w", channel, err)
	}

	b.log.Debug("Event published",
		"channel", channel,
		"event_type", event.Type,
		"event_id", event.ID,
		"entity_id", event.EntityID,
	)

	return nil
}

// Subscribe creates a subscription to a Redis channel.
func (b *RedisBroker) Subscribe(ctx context.Context, channel Channel) (<-chan *Event, error) {
	// 1. Initial check (Fast path)
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil, fmt.Errorf("broker is closed")
	}
	b.mu.RUnlock()

	// 2. Create Redis pub/sub subscription (Network call, done outside lock)
	pubsub := b.client.Subscribe(ctx, channel.String())

	// Wait for subscription confirmation
	if _, err := pubsub.Receive(ctx); err != nil {
		pubsub.Close()
		return nil, fmt.Errorf("failed to subscribe to channel %s: %w", channel, err)
	}

	// Create output channel for events
	eventCh := make(chan *Event, 10)

	// Create cancellable context for this subscription
	subCtx, cancel := context.WithCancel(context.Background())

	// Track subscription for cleanup
	subID := fmt.Sprintf("%s-%d", channel, time.Now().UnixNano())

	// 3. Register subscription (Critical Section)
	b.mu.Lock()
	// IMPROVEMENT: Re-check closed state to prevent race condition.
	// If Close() ran while we were doing the network call above, we must abort.
	if b.closed {
		b.mu.Unlock()
		pubsub.Close()
		cancel()
		close(eventCh)
		return nil, fmt.Errorf("broker is closed")
	}
	b.subscriptions[subID] = cancel
	b.mu.Unlock()

	// Start goroutine to receive messages
	go func() {
		// IMPROVEMENT: Use a reusable timer to avoid allocating new timers in the loop
		// which reduces GC pressure.
		timer := time.NewTimer(0)
		if !timer.Stop() {
			<-timer.C
		}
		defer timer.Stop()

		defer func() {
			pubsub.Close()
			close(eventCh)
			b.mu.Lock()
			delete(b.subscriptions, subID)
			b.mu.Unlock()
			b.log.Debug("Subscription closed", "channel", channel, "sub_id", subID)
		}()

		msgCh := pubsub.Channel()

		for {
			select {
			case <-ctx.Done():
				return // Client disconnected
			case <-subCtx.Done():
				return // Broker shutdown
			case msg, ok := <-msgCh:
				if !ok {
					b.log.Warn("Redis channel closed", "channel", channel)
					return
				}

				var event Event
				if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
					b.log.Error("Failed to unmarshal event",
						"channel", channel,
						"error", err,
					)
					continue
				}

				// Reset timer for the write timeout
				timer.Reset(5 * time.Second)

				select {
				case eventCh <- &event:
					// Stop timer and drain channel if it fired
					if !timer.Stop() {
						<-timer.C
					}
					b.log.Debug("Event delivered", "channel", channel, "event_id", event.ID)

				case <-timer.C:
					b.log.Warn("Subscriber too slow, dropping event",
						"channel", channel,
						"event_id", event.ID,
					)

				// IMPROVEMENT: Listen for cancellations during the send attempt.
				// Originally, if the buffer was full, this would block for 5s even if the app was shutting down.
				case <-ctx.Done():
					return
				case <-subCtx.Done():
					return
				}
			}
		}
	}()

	b.log.Info("Subscription created",
		"channel", channel,
		"sub_id", subID,
	)

	return eventCh, nil
}

// Close gracefully shuts down the broker and all subscriptions.
func (b *RedisBroker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	b.closed = true

	// Cancel all active subscriptions
	for subID, cancel := range b.subscriptions {
		cancel()
		b.log.Debug("Cancelled subscription", "sub_id", subID)
	}

	b.log.Info("Event broker closed")
	return nil
}

// Health checks the broker's connection to Redis.
func (b *RedisBroker) Health(ctx context.Context) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return fmt.Errorf("broker is closed")
	}
	b.mu.RUnlock()

	if err := b.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}

	return nil
}