package notification

import (
	"context"
	"fmt"
	"time"

	"hauslet/internal/platform/sms"
	smsJob "hauslet/internal/queue/jobs/sms"
)

// Send2FASetupSMS sends a 2FA setup OTP code via SMS (async with fallback).
func (s *NotificationService) Send2FASetupSMS(ctx context.Context, phoneNumber, otpCode string) error {
	if s.smsClient == nil {
		return fmt.Errorf("SMS client not configured")
	}

	message := fmt.Sprintf("Your Hauslet 2FA setup code is: %s. This code expires in 10 minutes. Do not share this code with anyone.", otpCode)

	s.sendSMSAsync("send 2FA setup SMS", func() error {
		job := smsJob.SMSJob{
			PhoneNumber: phoneNumber,
			Message:     message,
		}
		if err := s.publishSMSJob(job); err == nil {
			return nil
		} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
			return err
		}
		// Fallback to direct send
		return s.sendSMSDirect(ctx, phoneNumber, message)
	})

	return nil
}

// Send2FAVerificationSMS sends a 2FA verification OTP code via SMS during login (async with fallback).
func (s *NotificationService) Send2FAVerificationSMS(ctx context.Context, phoneNumber, otpCode string) error {
	if s.smsClient == nil {
		return fmt.Errorf("SMS client not configured")
	}

	message := fmt.Sprintf("Your Hauslet login verification code is: %s. This code expires in 5 minutes. Do not share this code with anyone.", otpCode)

	s.sendSMSAsync("send 2FA verification SMS", func() error {
		job := smsJob.SMSJob{
			PhoneNumber: phoneNumber,
			Message:     message,
		}
		if err := s.publishSMSJob(job); err == nil {
			return nil
		} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
			return err
		}
		// Fallback to direct send
		return s.sendSMSDirect(ctx, phoneNumber, message)
	})

	return nil
}

// sendSMSDirect sends an SMS directly (fallback when queue fails).
func (s *NotificationService) sendSMSDirect(ctx context.Context, phoneNumber, message string) error {
	req := sms.SMSRequest{
		To:      phoneNumber,
		Message: message,
	}

	resp, err := s.smsClient.Send(ctx, req)
	if err != nil {
		s.log.Error("failed to send SMS directly",
			"phone", phoneNumber,
			"error", err,
		)
		return fmt.Errorf("failed to send SMS: %w", err)
	}

	s.log.Info("SMS sent successfully (direct)",
		"phone", phoneNumber,
		"message_id", resp.MessageID,
		"provider", resp.Provider,
	)
	return nil
}

// sendSMSAsync runs fn in a goroutine, logging errors on failure.
func (s *NotificationService) sendSMSAsync(label string, fn func() error) {
	go func() {
		if err := fn(); err != nil && s.log != nil {
			s.log.Warn("send SMS async error", "label", label, "error", err)
		}
	}()
}

// publishSMSJob tries to enqueue the SMS job and returns an error on failure.
func (s *NotificationService) publishSMSJob(job smsJob.SMSJob) error {
	if s.queueClient == nil || s.smsQueueSubject == "" {
		return fmt.Errorf("SMS queue not configured")
	}

	pubCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.queueClient.Publish(pubCtx, s.smsQueueSubject, job); err != nil {
		if s.log != nil {
			s.log.Warn("failed to publish auth SMS job", "queue_subject", s.smsQueueSubject, "error", err)
		}
		return err
	}

	return nil
}
