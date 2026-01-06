package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hauslet/internal/modules/interactions/domain"
	"hauslet/internal/platform/redis"
	"log/slog"
)

const (
	// RedisQueueKey is the key for the interaction queue in Redis
	RedisQueueKey = "interactions:queue"
)

// TrackerServiceImpl implements TrackerService
type TrackerServiceImpl struct {
	redis       redis.RedisClient
	botDetector BotDetector
	log         *slog.Logger
}

// NewTrackerService creates a new tracker service
func NewTrackerService(redis redis.RedisClient, botDetector BotDetector, log *slog.Logger) TrackerService {
	return &TrackerServiceImpl{
		redis:       redis,
		botDetector: botDetector,
		log:         log,
	}
}

// Track records a new interaction by pushing to Redis queue
func (s *TrackerServiceImpl) Track(ctx context.Context, input TrackInput) error {
	// Validate input
	if err := s.validateInput(input); err != nil {
		return err
	}

	// Create interaction
	interaction := domain.NewInteraction(
		input.UserID,
		input.SessionID,
		input.Type,
		input.EntityType,
		input.EntityID,
		input.Context,
		domain.InteractionMeta{
			DeviceType: input.DeviceType,
			Platform:   input.Platform,
			IPHash:     s.hashIP(input.IPAddress),
			UserAgent:  input.UserAgent,
			Referrer:   input.Referrer,
		},
	)

	// Bot detection
	if s.botDetector != nil {
		interaction.IsBot = s.botDetector.IsBot(input.UserAgent)
	}

	// Serialize to JSON
	payload, err := json.Marshal(interaction)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to marshal interaction", "error", err)
		}
		return fmt.Errorf("failed to marshal interaction: %w", err)
	}

	// Push to Redis queue (fire-and-forget, very fast)
	if err := s.redis.RPush(ctx, RedisQueueKey, string(payload)).Err(); err != nil {
		if s.log != nil {
			s.log.Error("failed to push interaction to Redis", "error", err)
		}
		return fmt.Errorf("failed to queue interaction: %w", err)
	}

	if s.log != nil {
		s.log.Debug("interaction queued",
			"type", input.Type,
			"entity_type", input.EntityType,
			"entity_id", input.EntityID,
			"user_id", input.UserID)
	}

	return nil
}

// TrackBatch records multiple interactions
func (s *TrackerServiceImpl) TrackBatch(ctx context.Context, inputs []TrackInput) error {
	for _, input := range inputs {
		if err := s.Track(ctx, input); err != nil {
			// Log error but continue with other interactions
			if s.log != nil {
				s.log.Warn("failed to track interaction in batch", "error", err)
			}
		}
	}
	return nil
}

// validateInput validates the track input
func (s *TrackerServiceImpl) validateInput(input TrackInput) error {
	if input.SessionID == "" {
		return domain.ErrInvalidSessionID
	}

	if !input.Type.IsValid() {
		return domain.ErrInvalidInteractionType
	}

	if !input.EntityType.IsValid() {
		return domain.ErrInvalidEntityType
	}

	// Entity ID is required for most types except search
	if input.EntityType != domain.EntityTypeSearch && input.EntityID == nil {
		return domain.ErrMissingEntityID
	}

	return nil
}

// hashIP anonymizes IP addresses using SHA-256
func (s *TrackerServiceImpl) hashIP(ip string) string {
	if ip == "" {
		return ""
	}

	hash := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(hash[:])
}
