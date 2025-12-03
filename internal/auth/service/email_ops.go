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

// sendEmailAsync runs the provided send function in a goroutine and logs errors.
func (s *AuthServiceImpl) sendEmailAsync(label string, fn func() error) {
	go func() {
		if err := fn(); err != nil && s.log != nil {
			s.log.Logf("[WARN] %s: %v", label, err)
		}
	}()
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
		if s.queueClient != nil && s.queueSubject != "" {
			job := jobs.EmailJob{
				To:      emailAddr,
				Subject: subject,
				HTML:    htmlBody,
			}
			if err := s.queueClient.Publish(ctx, s.queueSubject, job); err == nil {
				return nil
			}
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
		if s.queueClient != nil && s.queueSubject != "" {
			job := jobs.EmailJob{
				To:      emailAddr,
				Subject: subject,
				HTML:    htmlBody,
			}
			if err := s.queueClient.Publish(ctx, s.queueSubject, job); err == nil {
				return nil
			}
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
		if s.queueClient != nil && s.queueSubject != "" {
			job := jobs.EmailJob{To: emailAddr, Subject: subject, HTML: htmlBody}
			if err := s.queueClient.Publish(ctx, s.queueSubject, job); err == nil {
				return nil
			}
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
		if s.queueClient != nil && s.queueSubject != "" {
			job := jobs.EmailJob{To: emailAddr, Subject: subject, HTML: htmlBody}
			if err := s.queueClient.Publish(ctx, s.queueSubject, job); err == nil {
				return nil
			}
		}
		return s.mailClient.SendHTML(ctx, emailAddr, subject, htmlBody)
	})

	return nil
}
