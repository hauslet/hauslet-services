package service

import (
	"context"
	"fmt"
	"time"

	"hauslet/internal/modules/verification/domain"
)

// =======================
// Phone Verification (OTP)
// =======================

const (
	otpTTL       = 10 * time.Minute
	otpKeyPrefix = "verification:otp:"
)

func (s *verificationService) GeneratePhoneOTP(ctx context.Context, req GeneratePhoneOTPRequest) (*GeneratePhoneOTPResponse, error) {
	// Get session and authorize
	session, err := s.GetSession(ctx, req.SessionID, req.UserID)
	if err != nil {
		return nil, err
	}

	// Ensure session is phone verification type
	if session.Type != domain.VerificationPhone {
		return nil, fmt.Errorf("session is not phone verification type")
	}

	// Get phone data
	phoneData, err := session.GetPhoneData()
	if err != nil {
		return nil, err
	}

	// Check if OTP can be retried
	if !phoneData.CanRetryOTP() {
		return nil, fmt.Errorf("maximum OTP attempts exceeded")
	}

	// Generate secure 6-digit OTP
	otpCode, err := s.generateSecureOTP()
	if err != nil {
		return nil, fmt.Errorf("failed to generate OTP code: %w", err)
	}

	now := time.Now()
	expiresAt := now.Add(otpTTL)

	// Store OTP in Redis with TTL (NOT in database)
	otpKey := s.buildOTPKey(session.ID.String())
	if err := s.redisClient.Set(ctx, otpKey, otpCode, otpTTL).Err(); err != nil {
		s.logger.Error("failed to store OTP in Redis", "error", err, "session_id", session.ID)
		return nil, fmt.Errorf("failed to store OTP: %w", err)
	}

	// Update phone data with metadata ONLY (not the OTP code)
	phoneData.OTPGeneratedAt = &now
	phoneData.OTPExpiresAt = &expiresAt

	// Update session data
	session.Data.Phone = phoneData
	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	// Enqueue SMS job for async sending or send directly (abstracted by notifier)
	smsResp, err := s.notifier.SendVerificationSMS(ctx, phoneData.PhoneNumber, otpCode)
	if err != nil {
		_ = s.redisClient.Del(ctx, otpKey)
		s.logger.Error("failed to send verification SMS", "error", err, "phone", phoneData.PhoneNumber)
		return nil, fmt.Errorf("failed to send OTP: %w", err)
	}

	provider := smsResp.Provider
	phoneData.SMSProvider = &provider
	session.Data.Phone = phoneData
	_ = s.repo.UpdateSession(ctx, session)

	s.logger.Info("OTP generated and queued/sent",
		"session_id", session.ID,
		"phone", phoneData.PhoneNumber,
		"expires_at", expiresAt.Format(time.RFC3339),
	)

	expiresAtStr := expiresAt.Format(time.RFC3339)
	return &GeneratePhoneOTPResponse{
		Session:     session,
		OTPSent:     true,
		ExpiresAt:   &expiresAtStr,
		SMSProvider: provider,
		Message:     "OTP sent successfully",
	}, nil
}

func (s *verificationService) VerifyPhoneOTP(ctx context.Context, req VerifyPhoneOTPRequest) (*VerifyPhoneOTPResponse, error) {
	// Get session and authorize
	session, err := s.GetSession(ctx, req.SessionID, req.UserID)
	if err != nil {
		return nil, err
	}

	// Ensure session is phone verification type
	if session.Type != domain.VerificationPhone {
		return nil, fmt.Errorf("session is not phone verification type")
	}

	// Get phone data
	phoneData, err := session.GetPhoneData()
	if err != nil {
		return nil, err
	}

	// Check if OTP is expired (based on DB timestamp)
	if phoneData.IsOTPExpired() {
		return &VerifyPhoneOTPResponse{
			Session:   session,
			Verified:  false,
			Remaining: session.AttemptsRemaining(),
			Message:   "OTP has expired. Please request a new one.",
		}, nil
	}

	// Retrieve OTP from Redis
	otpKey := s.buildOTPKey(session.ID.String())
	storedOTP, err := s.redisClient.Get(ctx, otpKey).Result()
	if err != nil {
		s.logger.Error("failed to retrieve OTP from Redis", "error", err, "session_id", session.ID)
		return &VerifyPhoneOTPResponse{
			Session:   session,
			Verified:  false,
			Remaining: session.AttemptsRemaining(),
			Message:   "OTP has expired or not found. Please request a new one.",
		}, nil
	}

	// Verify OTP code
	if storedOTP != req.OTPCode {
		// Increment failed attempts
		phoneData.OTPAttempts++
		session.Data.Phone = phoneData
		_ = s.repo.UpdateSession(ctx, session)

		s.logger.Warn("invalid OTP attempt",
			"session_id", session.ID,
			"attempts", phoneData.OTPAttempts,
			"max_attempts", phoneData.MaxOTPAttempts,
		)

		if !phoneData.CanRetryOTP() {
			// Delete OTP from Redis
			_ = s.redisClient.Del(ctx, otpKey)

			// Mark session as rejected if max attempts exceeded
			_ = session.Reject(domain.RejectionOther, stringPtr("Maximum OTP attempts exceeded"))
			_ = s.repo.UpdateSession(ctx, session)

			return &VerifyPhoneOTPResponse{
				Session:   session,
				Verified:  false,
				Remaining: 0,
				Message:   "Maximum verification attempts exceeded",
			}, nil
		}

		return &VerifyPhoneOTPResponse{
			Session:   session,
			Verified:  false,
			Remaining: phoneData.MaxOTPAttempts - phoneData.OTPAttempts,
			Message:   "Invalid OTP code",
		}, nil
	}

	// OTP verified successfully - delete from Redis
	if err := s.redisClient.Del(ctx, otpKey).Err(); err != nil {
		s.logger.Warn("failed to delete OTP from Redis after verification", "error", err)
	}

	// Update phone data
	now := time.Now()
	phoneData.VerifiedAt = &now
	session.Data.Phone = phoneData

	// Mark session as approved
	if err := session.Approve(); err != nil {
		return nil, err
	}

	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	// Notify profile module of phone verification
	if err := s.profileAdapter.MarkPhoneVerified(ctx, session.UserID, phoneData.PhoneNumber, now); err != nil {
		// Log but don't fail - verification was successful
		s.logger.Error("failed to notify profile of phone verification",
			"session_id", session.ID,
			"user_id", session.UserID,
			"error", err,
		)
	}

	s.logger.Info("phone verification successful",
		"session_id", session.ID,
		"phone", phoneData.PhoneNumber,
	)

	return &VerifyPhoneOTPResponse{
		Session:   session,
		Verified:  true,
		Remaining: session.AttemptsRemaining(),
		Message:   "Phone verified successfully",
	}, nil
}
