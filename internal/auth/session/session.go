package session

import (
	"context"
	"encoding/json"
	"fmt"
	"hauslet/internal/auth/domain"
	"hauslet/internal/platform/redis"
	"time"
)

const (
	sessionKeyPrefix      = "session:"
	userSessionsKeyPrefix = "user:sessions:"
)

type SessionStore interface {
	Store(sessionID string, session *domain.Session) error
	Load(sessionID string) (*domain.Session, bool)
	Delete(sessionID string) error
	DeleteByUserID(userID string) error
	ListActiveByUserID(userID string) ([]*domain.Session, error)
	UpdateExpiry(sessionID string, expiresAt time.Time) error
	Range(f func(sessionID string, session *domain.Session) bool)
}

type SessionStoreImpl struct {
	store redis.RedisClient
	ctx   context.Context
}

// NewSessionStore creates a new Redis-backed session store
func NewSessionStore(redisClient redis.RedisClient) *SessionStoreImpl {
	return &SessionStoreImpl{
		store: redisClient,
		ctx:   context.Background(),
	}
}

// Store saves a session to Redis
func (s *SessionStoreImpl) Store(sessionID string, session *domain.Session) error {
	if session == nil {
		return fmt.Errorf("session cannot be nil")
	}

	// Marshal session to JSON
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	key := sessionKeyPrefix + sessionID
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		ttl = 24 * time.Hour // Default 24 hours
	}

	// Store session data
	if err := s.store.Set(s.ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store session: %w", err)
	}

	// Add session to user's session set
	userSessionsKey := userSessionsKeyPrefix + session.UserID.String()
	if err := s.store.SAdd(s.ctx, userSessionsKey, sessionID).Err(); err != nil {
		return fmt.Errorf("failed to add session to user set: %w", err)
	}

	return nil
}

// Load retrieves a session from Redis
func (s *SessionStoreImpl) Load(sessionID string) (*domain.Session, bool) {
	key := sessionKeyPrefix + sessionID

	result := s.store.Get(s.ctx, key)
	if result.Err() != nil {
		return nil, false
	}

	data, err := result.Bytes()
	if err != nil {
		return nil, false
	}

	var session domain.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, false
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		_ = s.Delete(sessionID) // Ignore error on cleanup
		return nil, false
	}

	return &session, true
}

// Delete removes a session from Redis
func (s *SessionStoreImpl) Delete(sessionID string) error {
	// Load session to get UserID
	session, exists := s.Load(sessionID)
	if !exists {
		return nil // Session doesn't exist, nothing to delete
	}

	// Delete session data
	key := sessionKeyPrefix + sessionID
	if err := s.store.Del(s.ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Remove from user's session set
	userSessionsKey := userSessionsKeyPrefix + session.UserID.String()
	if err := s.store.SRem(s.ctx, userSessionsKey, sessionID).Err(); err != nil {
		return fmt.Errorf("failed to remove session from user set: %w", err)
	}

	return nil
}

// DeleteByUserID removes all sessions for a user
func (s *SessionStoreImpl) DeleteByUserID(userID string) error {
	userSessionsKey := userSessionsKeyPrefix + userID

	// Get all session IDs for this user
	sessionIDs := s.store.SMembers(s.ctx, userSessionsKey)
	if sessionIDs.Err() != nil {
		return fmt.Errorf("failed to get user sessions: %w", sessionIDs.Err())
	}

	ids, err := sessionIDs.Result()
	if err != nil {
		return fmt.Errorf("failed to parse session IDs: %w", err)
	}

	// Delete each session
	for _, sessionID := range ids {
		key := sessionKeyPrefix + sessionID
		if err := s.store.Del(s.ctx, key).Err(); err != nil {
			return fmt.Errorf("failed to delete session %s: %w", sessionID, err)
		}
	}

	// Clear the user's session set
	if err := s.store.Del(s.ctx, userSessionsKey).Err(); err != nil {
		return fmt.Errorf("failed to delete user session set: %w", err)
	}

	return nil
}

// ListActiveByUserID retrieves all active sessions for a user
func (s *SessionStoreImpl) ListActiveByUserID(userID string) ([]*domain.Session, error) {
	userSessionsKey := userSessionsKeyPrefix + userID

	// Get all session IDs for this user
	sessionIDs := s.store.SMembers(s.ctx, userSessionsKey)
	if sessionIDs.Err() != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", sessionIDs.Err())
	}

	ids, err := sessionIDs.Result()
	if err != nil {
		return nil, fmt.Errorf("failed to parse session IDs: %w", err)
	}

	var sessions []*domain.Session
	now := time.Now()

	for _, sessionID := range ids {
		session, exists := s.Load(sessionID)
		if !exists {
			continue
		}

		// Only include active (non-expired) sessions
		if session.ExpiresAt.After(now) {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// UpdateExpiry updates the expiration time of a session
func (s *SessionStoreImpl) UpdateExpiry(sessionID string, expiresAt time.Time) error {
	// Load existing session
	session, exists := s.Load(sessionID)
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Update expiry
	session.ExpiresAt = expiresAt

	// Re-store with new expiry
	return s.Store(sessionID, session)
}

// Range iterates over all sessions (using pattern matching)
func (s *SessionStoreImpl) Range(f func(sessionID string, session *domain.Session) bool) {
	pattern := sessionKeyPrefix + "*"

	keys := s.store.Keys(s.ctx, pattern)
	if keys.Err() != nil {
		return
	}

	keyList, err := keys.Result()
	if err != nil {
		return
	}

	for _, key := range keyList {
		// Extract session ID from key
		sessionID := key[len(sessionKeyPrefix):]

		session, exists := s.Load(sessionID)
		if !exists {
			continue
		}

		// Call the function, stop if it returns false
		if !f(sessionID, session) {
			break
		}
	}
}
