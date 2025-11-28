package service

import (
	"sync"
	"time"
)

// RequestMetadata holds IP and User-Agent for a request
type RequestMetadata struct {
	IP        string
	UserAgent string
	Timestamp time.Time
}

// RequestMetadataStore is a thread-safe store for request metadata
// Used to pass IP/User-Agent from request handlers to ClaimsUpdater
type RequestMetadataStore struct {
	data sync.Map
}

// NewRequestMetadataStore creates a new request metadata store
func NewRequestMetadataStore() *RequestMetadataStore {
	store := &RequestMetadataStore{}

	// Start cleanup goroutine to remove stale entries
	go store.cleanup()

	return store
}

// Set stores request metadata for a given key (usually email)
func (s *RequestMetadataStore) Set(key string, ip, userAgent string) {
	s.data.Store(key, &RequestMetadata{
		IP:        ip,
		UserAgent: userAgent,
		Timestamp: time.Now(),
	})
}

// Get retrieves and removes request metadata for a given key
func (s *RequestMetadataStore) Get(key string) *RequestMetadata {
	if val, ok := s.data.LoadAndDelete(key); ok {
		if metadata, ok := val.(*RequestMetadata); ok {
			return metadata
		}
	}
	return nil
}

// cleanup removes stale entries (older than 5 minutes) every minute
func (s *RequestMetadataStore) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cutoff := time.Now().Add(-5 * time.Minute)

		s.data.Range(func(key, value interface{}) bool {
			if metadata, ok := value.(*RequestMetadata); ok {
				if metadata.Timestamp.Before(cutoff) {
					s.data.Delete(key)
				}
			}
			return true
		})
	}
}
