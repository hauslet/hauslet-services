package setup

import (
	"hauslet/config"
	"hauslet/internal/platform/email"
	"log/slog"
)

// InitializeEmailClient creates the appropriate email client based on environment.
func InitializeEmailClient(cfg *config.GlobalConfig, log *slog.Logger) *email.Client {
	var emailSender email.Sender

	switch cfg.App.Env {
	case "development", "testing":
		emailSender = email.NewSMTPAdapter(
			cfg.Services.Email.SMTP.Host,
			cfg.Services.Email.SMTP.Port,
			cfg.Services.Email.SMTP.User,
			cfg.Services.Email.SMTP.Pass,
			cfg.Services.Email.From,
		)
		log.Info("✅ SMTP email adapter initialized")
	default:
		emailSender = email.NewResendAdapter(
			cfg.Services.Email.Resend.APIKey,
			cfg.Services.Email.From,
		)
		log.Info("✅ Resend email adapter initialized")
	}

	return email.New(emailSender)
}
