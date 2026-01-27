package notification

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"hauslet/internal/platform/queue"
	"hauslet/internal/platform/sms"
	smsJob "hauslet/internal/queue/jobs/sms"
)

// NotificationService handles verification-module notifications.
type NotificationService struct {
	smsClient       *sms.Client
	queueClient     *queue.Client
	smsQueueSubject string
	log             *slog.Logger
}

// NewNotificationService creates a new notification service for verification.
func NewNotificationService(
	smsClient *sms.Client,
	queueClient *queue.Client,
	smsQueueSubject string,
	log *slog.Logger,
) *NotificationService {
	return &NotificationService{
		smsClient:       smsClient,
		queueClient:     queueClient,
		smsQueueSubject: smsQueueSubject,
		log:             log,
	}
}

// SendVerificationSMS sends a verification OTP via SMS (async with fallback).
func (s *NotificationService) SendVerificationSMS(ctx context.Context, phoneNumber, otpCode string) (*sms.SMSResponse, error) {
	if s.smsClient == nil {
		return nil, fmt.Errorf("SMS client not configured")
	}

	message := fmt.Sprintf("Your Hauslet verification code is: %s. Valid for 10 minutes.", otpCode)

	// Async sending via queue
	if s.queueClient != nil && s.smsQueueSubject != "" {
		job := smsJob.SMSJob{
			PhoneNumber: phoneNumber,
			Message:     message,
		}

		if err := s.publishSMSJob(job); err == nil {
			return &sms.SMSResponse{Provider: "queued", Status: "queued"}, nil
		} else if !s.queueClient.AllowFallback() {
			return nil, err
		}
		// Continue to fallback
	}

	// Fallback to direct send
	return s.sendSMSDirect(ctx, phoneNumber, message)
}

// sendSMSDirect sends an SMS directly.
func (s *NotificationService) sendSMSDirect(ctx context.Context, phoneNumber, message string) (*sms.SMSResponse, error) {
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
		return nil, fmt.Errorf("failed to send SMS: %w", err)
	}

	return resp, nil
}

// publishSMSJob tries to enqueue the SMS job and returns an error on failure.
func (s *NotificationService) publishSMSJob(job smsJob.SMSJob) error {
	pubCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.queueClient.Publish(pubCtx, s.smsQueueSubject, job); err != nil {
		if s.log != nil {
			s.log.Warn("failed to publish verification SMS job", "queue_subject", s.smsQueueSubject, "error", err)
		}
		return err
	}

	return nil
}
