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
				Host:   must("HOST"),
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
					DBHost:     must("DB_HOST"),
					DBUser:     must("DB_USER"),
					DbName:     must("DB_NAME"),
					DBPassword: must("DB_PASSWORD"),
					DBPort:     must("DB_PORT"),
					DBSslmode:  def("DB_SSLMODE", "disable"),
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
				Elastic: ElasticsearchConfig{
					URL:      must("ELASTICSEARCH_URL"),
					Index:    must("ELASTICSEARCH_INDEX"),
					Username: must("ELASTICSEARCH_USERNAME"),
					Password: must("ELASTICSEARCH_PASSWORD"),
					APIKey:   def("ELASTICSEARCH_API_KEY", ""),
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
					APIURL:   must("ANTHROPIC_API_URL"),
					APIModel: must("ANTHROPIC_API_MODEL"),
				},
				Gemini: GeminiConfig{
					APIKey:     must("GEMINI_API_KEY"),
					APIModel:   def("GEMINI_API_MODEL", "gemini-2.5-flash"),
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
					BaseURL: def("FX_URL", "https://api.currencyapi.com/"),
				},
			},
			Infra: InfraConfig{
				RabbitMQ: RabbitMQConfig{
					Addr: must("RABBITMQ_URL"),
				},
				NATS: NATSConfig{
					URL: def("NATS_URL", "nats://localhost:4222"),
					// StreamName, Subjects, and Consumers moved to YAML (cfg.YAML.Queue)
				},
			},
		}
	})

	return cfg
}
