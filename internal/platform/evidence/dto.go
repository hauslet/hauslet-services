package evidence

import (
	"time"

	"github.com/google/uuid"
)

// Evidence represents verification evidence metadata
type Evidence struct {
	SessionID  uuid.UUID // Associated verification session
	URL        string    // R2 object key
	Hash       string    // SHA-256 hash of image bytes
	Size       int64     // File size in bytes
	MimeType   string    // Content type (image/jpeg, image/png)
	UploadedAt time.Time // When evidence was uploaded
}

// UploadRequest contains data for uploading evidence
type UploadRequest struct {
	SessionID uuid.UUID         // Associated verification session
	Data      []byte            // Raw image bytes
	MimeType  string            // Content type
	Metadata  map[string]string // Additional metadata (user_id, type, etc.)
}

// DownloadResponse contains downloaded evidence and verification status
type DownloadResponse struct {
	Data     []byte    // Raw image bytes
	Evidence *Evidence // Evidence metadata
	Verified bool      // Whether hash verification passed
}

// SignedURLRequest contains parameters for generating signed URLs
type SignedURLRequest struct {
	EvidenceURL string        // R2 object key
	Expiry      time.Duration // URL expiration duration
}

// SignedURLResponse contains the generated signed URL
type SignedURLResponse struct {
	URL       string    // Pre-signed URL
	ExpiresAt time.Time // When URL expires
}

// IntegrityCheck represents the result of evidence integrity verification
type IntegrityCheck struct {
	EvidenceURL  string    // R2 object key
	ExpectedHash string    // Expected SHA-256 hash
	ActualHash   string    // Calculated SHA-256 hash
	Verified     bool      // Whether hashes match
	CheckedAt    time.Time // When verification was performed
}
