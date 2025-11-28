package config

import (
	"sync"
	"time"

	"github.com/joho/godotenv"
)

var (
	cfg  *GlobalConfig
	once sync.Once
)

func Load() *GlobalConfig {
	once.Do(func() {
		_ = godotenv.Load()

		cfg = &GlobalConfig{
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
				SessionDuration:    time.Hour * 24, // 24 hours default
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
					SendgridAPIKey: must("SENDGRID_API_KEY"),
					SendgridSender: must("SENDGRID_SENDER"),
				},
				Claude: ClaudeConfig{
					APIKey:   must("CLAUDE_API_KEY"),
					APIURL:   must("CLAUDE_API_URL"),
					APIModel: must("CLAUDE_API_MODEL"),
				},
				Calendar: CalendarConfig{
					MaintenanceHour:     getInt("CALENDAR_MAINTENANCE_HOUR", 2),
					WeeklyCheckDay:      getInt("CALENDAR_WEEKLY_CHECK_DAY", 0),
					CleanupOldDays:      getInt("CALENDAR_CLEANUP_OLD_DAYS", 30),
					SchedulerEnabled:    getBool("CALENDAR_SCHEDULER_ENABLED", true),
					ImmediateGeneration: getBool("CALENDAR_IMMEDIATE_GENERATION", true),
					RetryDelayMinutes:   getInt("CALENDAR_RETRY_DELAY_MINUTES", 10),
				},
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
			},
		}
	})

	return cfg
}
