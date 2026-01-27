package notification

import (
	"context"
	"fmt"
	authtemplates "hauslet/internal/modules/auth/templates"
	emailJob "hauslet/internal/queue/jobs/emails"
	"time"
)

func (s *NotificationService) SendWelcomeEmail(ctx context.Context, emailAddr, name string, otpCode string) error {

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

func (s *NotificationService) SendIdentityLinkedEmail(ctx context.Context, emailAddr, name, provider string) error {
	subject := "Security Alert: New Login Method Added"
	preview := "A new login method has been added to your Hauslet account"

	emailData := map[string]any{
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

func (s *NotificationService) SendPasswordResetEmail(ctx context.Context, emailAddr, name, token string, ttlMinutes int) error {
	subject := "Reset your Hauslet password"
	preview := "Use this code to reset your Hauslet password"

	emailData := map[string]any{
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

func (s *NotificationService) SendPasswordChangedEmail(ctx context.Context, emailAddr, name string) error {
	subject := "Your Hauslet password was changed"
	preview := "We updated your Hauslet password"

	emailData := map[string]any{
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

func (s *NotificationService) SendPasswordlessLoginEmail(ctx context.Context, emailAddr, otpCode, magicLink string, ttlMinutes int) error {
	subject := "Your Hauslet login code"
	preview := "Use this code to sign in to Hauslet"

	emailData := map[string]any{
		"Code":      otpCode,
		"MagicLink": magicLink,
		"TTL":       ttlMinutes,
		"Subject":   subject,
		"Preview":   preview,
		"Year":      time.Now().Year(),
	}

	htmlBody, err := s.mailClient.RenderTemplate(authtemplates.FS, "passwordless_login.html", emailData)
	if err != nil {
		return err
	}

	// Send synchronously since the verified provider expects immediate delivery
	job := emailJob.EmailJob{To: emailAddr, Subject: subject, HTML: htmlBody}
	if err := s.publishEmailJob(job); err == nil {
		return nil
	} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
		return err
	}
	return s.mailClient.SendHTML(ctx, emailAddr, subject, htmlBody)
}

func (s *NotificationService) Send2FASetupEmail(ctx context.Context, emailAddr, name, code string) error {
	subject := "Hauslet: Your 2FA Setup Code"
	preview := "Your verification code for 2FA setup is " + code

	emailData := map[string]any{
		"Name":    name,
		"Code":    code,
		"TTL":     10, // 10 minutes
		"Subject": subject,
		"Preview": preview,
		"Year":    time.Now().Year(),
	}

	htmlBody, err := s.mailClient.RenderTemplate(authtemplates.FS, "2fa_setup.html", emailData)
	if err != nil {
		return err
	}

	s.sendEmailAsync("send 2FA setup email", func() error {
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

func (s *NotificationService) Send2FAVerificationEmail(ctx context.Context, emailAddr, name, code string) error {
	subject := "Hauslet: Your Login Verification Code"
	preview := "Your verification code is " + code

	emailData := map[string]any{
		"Name":    name,
		"Code":    code,
		"TTL":     5, // 5 minutes
		"Subject": subject,
		"Preview": preview,
		"Year":    time.Now().Year(),
	}

	htmlBody, err := s.mailClient.RenderTemplate(authtemplates.FS, "2fa_verify.html", emailData)
	if err != nil {
		return err
	}

	s.sendEmailAsync("send 2FA verification email", func() error {
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

// sendEmailAsync runs the provided send function in a goroutine and logs errors.
func (s *NotificationService) sendEmailAsync(label string, fn func() error) {
	go func() {
		if err := fn(); err != nil && s.log != nil {
			s.log.Warn("send email async error", "label", label, "error", err)
		}
	}()
}

// publishEmailJob tries to enqueue the email job and returns an error on failure.
func (s *NotificationService) publishEmailJob(job emailJob.EmailJob) error {
	if s.queueClient == nil || s.emailSubject == "" {
		return fmt.Errorf("queue not configured")
	}

	pubCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.queueClient.Publish(pubCtx, s.emailSubject, job); err != nil {
		if s.log != nil {
			s.log.Warn("failed to publish auth email job", "queue_subject", s.emailSubject, "error", err)
		}
		return err
	}

	return nil
}
