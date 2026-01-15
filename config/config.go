// Package config loads application configuration from environment variables.
// SECURITY: Never commit . env files or hardcode secrets in this file.
// Use . env.example for documentation of required variables.
package config

import (
	"sync"

	"github.com/joho/godotenv"
)

var (
	cfg  *GlobalConfig
	once sync.Once
)

func Load() *GlobalConfig {
	once.Do(func() {
		_ = godotenv.Load()

		// Load YAML service configurations
		yamlCfg, err := LoadYAMLConfig()
		if err != nil {
			must := func() { panic("Failed to load YAML config: " + err.Error()) }
			must()
		}

		cfg = &GlobalConfig{
			YAML: yamlCfg,
			App: AppConfig{
				Env:    must("APP_ENV"),
				Port:   def("PORT", "8080"),
				Client: def("CLIENT", "http://localhost:3000"),
				Server: def("SERVER", "http://localhost:3000/v1"),
			},
			Auth: AuthConfig{
				JWTSecret:          must("JWT_SECRET"),
				EncryptAuthCodeKey: mustKey("ENCRYPTION_KEY"),
				GoogleClientID:     must("GOOGLE_CLIENT_ID"),
				GoogleCLSecret:     must("GOOGLE_CLIENT_SECRET"),
				RedirectURL:        must("REDIRECT_URL"),
				SessionDuration:    getDuration("SESSION_DURATION", "24h"),

				// Security settings (configurable)
				TokenDuration:   getDuration("TOKEN_DURATION", "5m"),
				CookieDuration:  getDuration("COOKIE_DURATION", "24h"),
				AvatarStorePath: def("AVATAR_STORE_PATH", "/tmp/avatars"),
				DisableXSRF:     getBool("DISABLE_XSRF", false), // Default: XSRF enabled
				CookieDomain:    def("COOKIE_DOMAIN", ""),
			},
			Storage: StorageConfig{
				DB: DBConfig{
					DatabaseURL: must("DATABASE_URL"),
				},
				Redis: RedisConfig{
					Addr: must("REDIS_ADDR"),
				},
				R2: R2Config{
					AccountID:       must("R2_ACCOUNT_ID"),
					AccessKeyID:     must("R2_ACCESS_KEY_ID"),
					AccessKeySecret: must("R2_ACCESS_KEY_SECRET"),
					BucketName:      must("R2_BUCKET_NAME"),
					Endpoint:        must("R2_ENDPOINT"),
					CDNHost:         must("CDN_HOST"),
				},
			},
			Services: ServicesConfig{
				Email: EmailConfig{
					From: must("EMAIL_FROM"),
					SMTP: SMTPConfig{
						Host: def("SMTP_HOST", "localhost"),
						Port: getInt("SMTP_PORT", 1025),
						User: def("SMTP_USER", ""),
						Pass: def("SMTP_PASS", ""),
					},
					Resend: ResendConfig{
						APIKey: def("RESEND_API_KEY", ""),
					},
				},
				Anthropic: AnthropicConfig{
					APIKey:   must("ANTHROPIC_API_KEY"),
					APIURL:   def("ANTHROPIC_API_URL", "https://api.anthropic.com/v1"),
					APIModel: def("ANTHROPIC_API_MODEL", "claude-sonnet-4-20250514"),
				},
				Gemini: GeminiConfig{
					APIKey:     must("GEMINI_API_KEY"),
					APIModel:   def("GEMINI_API_MODEL", "gemini-3-flash-preview"),
					Project:    def("GEMINI_PROJECT", ""),
					EmbedModel: def("GEMINI_EMBED_MODEL", "gemini-embedding-001"),
				},
				Fallback: FallbackConfig{
					Enable:           getBool("AI_FALLBACK_ENABLE", true),
					AttemptThreshold: getInt("AI_FALLBACK_ATTEMPT_THRESHOLD", 3),
				},
				// Calendar config moved to YAML (cfg.YAML.Calendar)
				Payment: PaymentConfig{
					PaystackSecretKey:    must("PAYSTACK_SECRET_KEY"),
					PaystackPublicKey:    must("PAYSTACK_PUBLIC_KEY"),
					StripeSecretKey:      must("STRIPE_SECRET_KEY"),
					StripeWebhookSecret:  must("STRIPE_WEBHOOK_SECRET"),
					StripePublishableKey: must("STRIPE_PUBLISHABLE_KEY"),
				},
				FX: FXConfig{
					APIKey:  must("FX_API_KEY"),
					BaseURL: def("FX_URL", "https://v6.exchangerate-api.com/v6"),
				},
				KYC: KYCConfig{
					DojahAPIKey:         def("DOJAH_API_KEY", ""),
					DojahSecretKey:      def("DOJAH_SECRET_KEY", ""),
					DojahWebhookSecret:  def("DOJAH_WEBHOOK_SECRET", ""),
					VeriffAPIKey:        def("VERIFF_API_KEY", ""),
					VeriffSecretKey:     def("VERIFF_SECRET_KEY", ""),
					VeriffWebhookSecret: def("VERIFF_WEBHOOK_SECRET", ""),
				},
				SMS: SMSConfig{
					TermiiAPIKey:     def("TERMII_API_KEY", ""),
					TermiiSenderID:   def("TERMII_SENDER_ID", "Hauslet"),
					TwilioAccountSID: def("TWILIO_ACCOUNT_SID", ""),
					TwilioAuthToken:  def("TWILIO_AUTH_TOKEN", ""),
					TwilioFromNumber: def("TWILIO_FROM_NUMBER", ""),
				},
				VertexAI: VertexAIConfig{
					ProjectID:       def("VERTEX_AI_PROJECT_ID", ""),
					AgentID:         def("VERTEX_AI_AGENT_ID", ""),
					Location:        def("VERTEX_AI_LOCATION", ""),
					CredentialsPath: def("VERTEX_AI_CREDENTIALS_PATH", ""),
				},
				Messaging: MessagingConfig{
					AIContextMessages:      getInt("MESSAGING_AI_CONTEXT_MESSAGES", 10),
					AIConfidenceThreshold:  getFloat("MESSAGING_AI_CONFIDENCE_THRESHOLD", 0.6),
					AITimeoutSeconds:       getInt("MESSAGING_AI_TIMEOUT_SECONDS", 15),
					MaxConversationAgeDays: getInt("MESSAGING_MAX_CONVERSATION_AGE_DAYS", 30),
					AIServingConfig:        def("MESSAGING_AI_SERVING_CONFIG", ""),
				},
			},
			Infra: InfraConfig{
				CloudTasks: CloudTasksConfig{
					ProjectID:           def("CLOUD_TASKS_PROJECT_ID", ""),
					Location:            def("CLOUD_TASKS_LOCATION", ""),
					WorkerBaseURL:       def("CLOUD_TASKS_WORKER_URL", ""),
					ServiceAccountEmail: def("CLOUD_TASKS_SERVICE_ACCOUNT", ""),
				},
			},
		}
	})

	return cfg
}
