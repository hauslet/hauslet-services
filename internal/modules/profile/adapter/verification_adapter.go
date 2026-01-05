package adapter

import (
	"context"
	"log/slog"
	"time"

	"hauslet/internal/modules/profile/service"
	"hauslet/internal/modules/verification/port"

	"github.com/google/uuid"
)

// verificationAdapter implements port.ProfileAdapter
// This adapter allows the verification module to update profile verification status.
type verificationAdapter struct {
	profileService service.ProfileService
	logger         *slog.Logger
}

// NewVerificationAdapter creates a new profile adapter for verification callbacks.
func NewVerificationAdapter(profileService service.ProfileService, logger *slog.Logger) port.ProfileAdapter {
	return &verificationAdapter{
		profileService: profileService,
		logger:         logger,
	}
}

// MarkIdentityVerified implements port.ProfileAdapter
func (a *verificationAdapter) MarkIdentityVerified(ctx context.Context, userID uuid.UUID, verificationLevel string, verifiedAt time.Time) error {
	a.logger.Info("marking identity verified in profile",
		"user_id", userID.String(),
		"level", verificationLevel,
		"verified_at", verifiedAt,
	)

	err := a.profileService.SetVerificationStatus(ctx, userID.String(), verificationLevel, true, &verifiedAt)
	if err != nil {
		a.logger.Error("failed to mark identity verified in profile",
			"user_id", userID.String(),
			"error", err,
		)
		return err
	}

	a.logger.Info("identity verification updated in profile",
		"user_id", userID.String(),
		"level", verificationLevel,
	)
	return nil
}

// MarkPhoneVerified implements port.ProfileAdapter
func (a *verificationAdapter) MarkPhoneVerified(ctx context.Context, userID uuid.UUID, phoneNumber string, verifiedAt time.Time) error {
	a.logger.Info("marking phone verified in profile",
		"user_id", userID.String(),
		"phone_number", phoneNumber,
		"verified_at", verifiedAt,
	)

	// Update phone verification status on the profile
	// The profile service will automatically calculate the verification level
	updates := map[string]any{
		"phone_verified": true,
	}

	_, err := a.profileService.PatchProfile(ctx, userID.String(), updates)
	if err != nil {
		a.logger.Error("failed to mark phone verified in profile",
			"user_id", userID.String(),
			"error", err,
		)
		return err
	}

	// Recalculate verification level based on both phone and ID verification
	// Let the service figure out the new level based on current state
	err = a.profileService.SetVerificationStatus(ctx, userID.String(), "", false, nil)
	if err != nil {
		// Log but don't fail - the phone_verified was already set
		a.logger.Warn("failed to recalculate verification level after phone verification",
			"user_id", userID.String(),
			"error", err,
		)
	}

	a.logger.Info("phone verification updated in profile",
		"user_id", userID.String(),
	)
	return nil
}

// MarkAddressVerified implements port.ProfileAdapter
func (a *verificationAdapter) MarkAddressVerified(ctx context.Context, userID uuid.UUID, address string, verifiedAt time.Time) error {
	a.logger.Info("marking address verified in profile",
		"user_id", userID.String(),
		"address", address,
		"verified_at", verifiedAt,
	)

	// Update address in profile if different from current
	// Address verification might also update the address field itself
	updates := map[string]any{
		"address": address,
	}

	_, err := a.profileService.PatchProfile(ctx, userID.String(), updates)
	if err != nil {
		a.logger.Error("failed to update verified address in profile",
			"user_id", userID.String(),
			"error", err,
		)
		return err
	}

	a.logger.Info("address verification updated in profile",
		"user_id", userID.String(),
	)
	return nil
}

// Ensure verificationAdapter implements port.ProfileAdapter
var _ port.ProfileAdapter = (*verificationAdapter)(nil)
