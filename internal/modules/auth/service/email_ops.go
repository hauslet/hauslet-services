package service

import (
	"context"
	"fmt"
	"time"

	authtemplates "hauslet/internal/modules/auth/templates"
	emailJob "hauslet/internal/queue/jobs/emails"
)

// ============================================================================
// Email Operations
// ============================================================================

// sendEmailAsync runs the provided send function in a goroutine and logs errors.
func (s *AuthServiceImpl) sendEmailAsync(label string, fn func() error) {
	go func() {
		if err := fn(); err != nil && s.log != nil {
			s.log.Warn("email operation failed", "operation", label, "error", err)
		}
	}()
}

// publishEmailJob tries to enqueue the email job and returns an error on failure.
// It uses a short-lived background context so cancellation of the request
// doesn't prevent publishing.
func (s *AuthServiceImpl) publishEmailJob(job emailJob.EmailJob) error {
	if s.queueClient == nil || s.queueSubject == "" {
		return fmt.Errorf("queue not configured")
	}

	pubCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.queueClient.Publish(pubCtx, s.queueSubject, job); err != nil {
		if s.log != nil {
			s.log.Warn("failed to publish email job", "subject", s.queueSubject, "error", err)
		}
		return err
	}

	return nil
}

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

	emailData := map[string]any{
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

	s.sendEmailAsync("send welcome email", func() error {
		job := emailJob.EmailJob{
			To:      emailAddr,
			Subject: subject,
			HTML:    htmlBody,
		}
		if err := s.publishEmailJob(job); err == nil {
			return nil
		} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
			return err
		}
		return s.mailClient.SendHTML(ctx, emailAddr, subject, htmlBody)
	})

	return nil
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

	s.sendEmailAsync("send identity linked email", func() error {
		job := emailJob.EmailJob{
			To:      emailAddr,
			Subject: subject,
			HTML:    htmlBody,
		}
		if err := s.publishEmailJob(job); err == nil {
			return nil
		} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
			return err
		}
		return s.mailClient.SendHTML(ctx, emailAddr, subject, htmlBody)
	})

	return nil
}

func (s *AuthServiceImpl) SendPasswordResetEmail(ctx context.Context, emailAddr, name, token string, ttlMinutes int) error {
	subject := "Reset your Hauslet password"
	preview := "Use this code to reset your Hauslet password"

	emailData := map[string]interface{}{
		"Name":    name,
		"Token":   token,
		"TTL":     ttlMinutes,
		"Subject": subject,
		"Preview": preview,
		"Year":    time.Now().Year(),
	}

	htmlBody, err := s.mailClient.RenderTemplate(authtemplates.FS, "forgot_password.html", emailData)
	if err != nil {
		return err
	}

	s.sendEmailAsync("send password reset email", func() error {
		job := emailJob.EmailJob{To: emailAddr, Subject: subject, HTML: htmlBody}
		if err := s.publishEmailJob(job); err == nil {
			return nil
		} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
			return err
		}
		return s.mailClient.SendHTML(ctx, emailAddr, subject, htmlBody)
	})

	return nil
}

func (s *AuthServiceImpl) SendPasswordChangedEmail(ctx context.Context, emailAddr, name string) error {
	subject := "Your Hauslet password was changed"
	preview := "We updated your Hauslet password"

	emailData := map[string]interface{}{
		"Name":      name,
		"Timestamp": time.Now().Format("Monday, January 2, 2006 at 3:04 PM MST"),
		"Subject":   subject,
		"Preview":   preview,
		"Year":      time.Now().Year(),
	}

	htmlBody, err := s.mailClient.RenderTemplate(authtemplates.FS, "password_changed.html", emailData)
	if err != nil {
		return err
	}

	s.sendEmailAsync("send password changed email", func() error {
		job := emailJob.EmailJob{To: emailAddr, Subject: subject, HTML: htmlBody}
		if err := s.publishEmailJob(job); err == nil {
			return nil
		} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
			return err
		}
		return s.mailClient.SendHTML(ctx, emailAddr, subject, htmlBody)
	})

	return nil
}
