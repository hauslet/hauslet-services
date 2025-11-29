package service

import (
	"context"
	"time"

	authtemplates "hauslet/internal/auth/templates"
	"hauslet/internal/queue/jobs"
)

// ============================================================================
// Email Operations
// ============================================================================

func (s *AuthServiceImpl) SendWelcomeEmail(ctx context.Context, emailAddr, name string, otpCode string) error {

	// Determine the logic: If there is an OTP, we send the verification version
	sendOTP := otpCode != ""

	var subject, preview string

	if sendOTP {
		subject = "Verify your email address"
		preview = "Your Hauslet verification code is " + otpCode
	} else {
		subject = "Welcome to Hauslet"
		preview = "Thanks for joining Hauslet!"
	}

	emailData := map[string]interface{}{
		"Name":    name,
		"OTP":     otpCode,
		"SendOTP": sendOTP, // This controls the {{if}} block in the HTML

		// Required for the Layout
		"Subject": subject,
		"Preview": preview,
		"Year":    time.Now().Year(),
	}

	// Render HTML in the producer
	htmlBody, err := s.mailClient.RenderTemplate(
		authtemplates.FS,
		"welcome.html",
		emailData,
	)
	if err != nil {
		return err
	}

	// Prefer queued send if available
	if s.queueClient != nil && s.queueSubject != "" {
		job := jobs.EmailJob{
			To:      emailAddr,
			Subject: subject,
			HTML:    htmlBody,
		}
		if err := s.queueClient.Publish(ctx, s.queueSubject, job); err == nil {
			return nil
		} else if s.log != nil {
			s.log.Logf("[WARN] queue publish failed, falling back to direct send: %v", err)
		}
	}

	// Fallback: direct send
	return s.mailClient.SendHTML(ctx, emailAddr, subject, htmlBody)
}

func (s *AuthServiceImpl) SendIdentityLinkedEmail(ctx context.Context, emailAddr, name, provider string) error {
	subject := "Security Alert: New Login Method Added"
	preview := "A new login method has been added to your Hauslet account"

	emailData := map[string]interface{}{
		"Name":      name,
		"Provider":  provider,
		"Email":     emailAddr,
		"Timestamp": time.Now().Format("Monday, January 2, 2006 at 3:04 PM MST"),

		// Required for the Layout
		"Subject": subject,
		"Preview": preview,
		"Year":    time.Now().Year(),
	}

	// Render HTML in the producer
	htmlBody, err := s.mailClient.RenderTemplate(
		authtemplates.FS,
		"identity_linked.html",
		emailData,
	)
	if err != nil {
		return err
	}

	// Prefer queued send if available
	if s.queueClient != nil && s.queueSubject != "" {
		job := jobs.EmailJob{
			To:      emailAddr,
			Subject: subject,
			HTML:    htmlBody,
		}
		if err := s.queueClient.Publish(ctx, s.queueSubject, job); err == nil {
			return nil
		} else if s.log != nil {
			s.log.Logf("[WARN] queue publish failed, falling back to direct send: %v", err)
		}
	}

	// Fallback: direct send
	return s.mailClient.SendHTML(ctx, emailAddr, subject, htmlBody)
}
