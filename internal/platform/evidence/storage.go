package evidence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"hauslet/internal/platform/storage"

	"github.com/google/uuid"
)

// R2Store implements Store using Cloudflare R2 (S3-compatible)
// It wraps the existing storage.R2Storage and adds evidence-specific functionality
type R2Store struct {
	r2Storage *storage.R2Storage
}

// NewR2Store creates a new R2-backed evidence store
func NewR2Store(r2Storage *storage.R2Storage) *R2Store {
	return &R2Store{
		r2Storage: r2Storage,
	}
}

// Upload stores evidence in R2 and returns metadata with hash
func (s *R2Store) Upload(ctx context.Context, req UploadRequest) (*Evidence, error) {
	// Validate request
	if req.SessionID == uuid.Nil {
		return nil, fmt.Errorf("session_id is required")
	}
	if len(req.Data) == 0 {
		return nil, fmt.Errorf("evidence data is required")
	}

	// Calculate SHA-256 hash
	hash := calculateHash(req.Data)

	// Generate R2 object key
	// Format: evidence/{session_id}/{hash}.{ext}
	objectKey := fmt.Sprintf("evidence/%s/%s.jpg", req.SessionID, hash)

	// Upload to R2 using existing R2Storage
	err := s.r2Storage.UploadObject(ctx, objectKey, bytes.NewReader(req.Data), req.MimeType)
	if err != nil {
		return nil, fmt.Errorf("failed to upload evidence to R2: %w", err)
	}

	// Return evidence metadata
	return &Evidence{
		SessionID:  req.SessionID,
		URL:        objectKey,
		Hash:       hash,
		Size:       int64(len(req.Data)),
		MimeType:   req.MimeType,
		UploadedAt: time.Now(),
	}, nil
}

// Download retrieves evidence from R2 and verifies integrity
func (s *R2Store) Download(ctx context.Context, evidenceURL string) (*DownloadResponse, error) {
	if evidenceURL == "" {
		return nil, fmt.Errorf("evidence_url is required")
	}

	// Download from R2 using existing R2Storage
	data, contentType, err := s.r2Storage.GetObject(ctx, evidenceURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download evidence from R2: %w", err)
	}

	// Calculate hash for verification
	actualHash := calculateHash(data)

	// Extract session ID from path (evidence/{session_id}/{hash}.jpg)
	var sessionID uuid.UUID
	var storedHash string
	if _, err := fmt.Sscanf(evidenceURL, "evidence/%s/%s", &sessionID, &storedHash); err == nil {
		// Remove extension from stored hash
		if len(storedHash) > 4 {
			storedHash = storedHash[:len(storedHash)-4] // Remove .jpg
		}
	}

	evidence := &Evidence{
		URL:        evidenceURL,
		Hash:       actualHash,
		Size:       int64(len(data)),
		MimeType:   contentType,
		SessionID:  sessionID,
		UploadedAt: time.Now(),
	}

	// Verify integrity
	verified := actualHash == storedHash

	return &DownloadResponse{
		Data:     data,
		Evidence: evidence,
		Verified: verified,
	}, nil
}

// VerifyIntegrity checks if evidence hash matches expected value
func (s *R2Store) VerifyIntegrity(ctx context.Context, evidenceURL string, expectedHash string) (*IntegrityCheck, error) {
	// Download evidence
	resp, err := s.Download(ctx, evidenceURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download evidence for verification: %w", err)
	}

	// Compare hashes
	verified := resp.Evidence.Hash == expectedHash

	return &IntegrityCheck{
		EvidenceURL:  evidenceURL,
		ExpectedHash: expectedHash,
		ActualHash:   resp.Evidence.Hash,
		Verified:     verified,
		CheckedAt:    time.Now(),
	}, nil
}

// GenerateSignedURL creates a temporary signed URL for evidence access
func (s *R2Store) GenerateSignedURL(ctx context.Context, evidenceURL string, expiry time.Duration) (*SignedURLResponse, error) {
	if evidenceURL == "" {
		return nil, fmt.Errorf("evidence_url is required")
	}

	// Generate signed URL using existing R2Storage
	signedURL, err := s.r2Storage.GenerateSignedDownloadURL(ctx, evidenceURL, expiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate signed URL: %w", err)
	}

	return &SignedURLResponse{
		URL:       signedURL,
		ExpiresAt: time.Now().Add(expiry),
	}, nil
}

// Delete removes evidence from R2 storage
func (s *R2Store) Delete(ctx context.Context, evidenceURL string) error {
	if evidenceURL == "" {
		return fmt.Errorf("evidence_url is required")
	}

	// Delete using existing R2Storage
	err := s.r2Storage.DeleteObject(ctx, evidenceURL)
	if err != nil {
		return fmt.Errorf("failed to delete evidence from R2: %w", err)
	}

	return nil
}

// DeleteBySession removes all evidence for a session
func (s *R2Store) DeleteBySession(ctx context.Context, sessionID uuid.UUID) error {
	if sessionID == uuid.Nil {
		return fmt.Errorf("session_id is required")
	}

	prefix := fmt.Sprintf("evidence/%s/", sessionID)
	keys, err := s.r2Storage.ListObjects(ctx, prefix)
	if err != nil {
		return fmt.Errorf("failed to list evidence for session %s: %w", sessionID, err)
	}
	if len(keys) == 0 {
		return nil
	}

	for _, key := range keys {
		if err := s.r2Storage.DeleteObject(ctx, key); err != nil {
			return fmt.Errorf("failed to delete evidence object %s: %w", key, err)
		}
	}

	return nil
}

// Exists checks if evidence exists in R2 storage
func (s *R2Store) Exists(ctx context.Context, evidenceURL string) (bool, error) {
	if evidenceURL == "" {
		return false, fmt.Errorf("evidence_url is required")
	}

	// Check existence using existing R2Storage
	exists, err := s.r2Storage.ObjectExists(ctx, evidenceURL)
	if err != nil {
		return false, nil // Return false on error (not found)
	}

	return exists, nil
}

// Helper: calculateHash computes SHA-256 hash of data
func calculateHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
