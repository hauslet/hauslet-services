package evidence

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Store defines the interface for evidence storage operations
type Store interface {
	// Upload stores evidence and returns metadata with hash
	Upload(ctx context.Context, req UploadRequest) (*Evidence, error)

	// Download retrieves evidence and verifies integrity
	Download(ctx context.Context, evidenceURL string) (*DownloadResponse, error)

	// VerifyIntegrity checks if evidence hash matches expected value
	VerifyIntegrity(ctx context.Context, evidenceURL string, expectedHash string) (*IntegrityCheck, error)

	// GenerateSignedURL creates a temporary signed URL for evidence access
	GenerateSignedURL(ctx context.Context, evidenceURL string, expiry time.Duration) (*SignedURLResponse, error)

	// Delete removes evidence from storage (for GDPR compliance)
	Delete(ctx context.Context, evidenceURL string) error

	// DeleteBySession removes all evidence for a session
	DeleteBySession(ctx context.Context, sessionID uuid.UUID) error

	// Exists checks if evidence exists in storage
	Exists(ctx context.Context, evidenceURL string) (bool, error)
}
