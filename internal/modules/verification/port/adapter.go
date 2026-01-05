package port

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ProfileAdapter is an outbound port for notifying the profile module
// when verification status changes for a user.
type ProfileAdapter interface {
	// MarkIdentityVerified updates the user's profile when identity (KYC) verification completes.
	// The verificationLevel can be "basic", "identity", or "trusted" depending on verification tier.
	MarkIdentityVerified(ctx context.Context, userID uuid.UUID, verificationLevel string, verifiedAt time.Time) error

	// MarkPhoneVerified updates the user's profile when phone verification completes.
	MarkPhoneVerified(ctx context.Context, userID uuid.UUID, phoneNumber string, verifiedAt time.Time) error

	// MarkAddressVerified updates the user's profile when address verification completes.
	// This may update the user's verified address and related trust scores.
	MarkAddressVerified(ctx context.Context, userID uuid.UUID, address string, verifiedAt time.Time) error
}

// BusinessAdapter is an outbound port for notifying the business module
// when verification status changes for a business entity.
type BusinessAdapter interface {
	// MarkBusinessVerified updates the business entity when business verification completes.
	MarkBusinessVerified(ctx context.Context, businessID uuid.UUID, verifiedAt time.Time) error
}

// NoopProfileAdapter is a no-op implementation for cases where profile integration is not needed.
type NoopProfileAdapter struct{}

func (n *NoopProfileAdapter) MarkIdentityVerified(ctx context.Context, userID uuid.UUID, verificationLevel string, verifiedAt time.Time) error {
	return nil
}

func (n *NoopProfileAdapter) MarkPhoneVerified(ctx context.Context, userID uuid.UUID, phoneNumber string, verifiedAt time.Time) error {
	return nil
}

func (n *NoopProfileAdapter) MarkAddressVerified(ctx context.Context, userID uuid.UUID, address string, verifiedAt time.Time) error {
	return nil
}

// NoopBusinessAdapter is a no-op implementation for cases where business integration is not needed.
type NoopBusinessAdapter struct{}

func (n *NoopBusinessAdapter) MarkBusinessVerified(ctx context.Context, businessID uuid.UUID, verifiedAt time.Time) error {
	return nil
}
