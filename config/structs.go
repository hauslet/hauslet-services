package config

import (
	"encoding/base64"
	"log"
	"os"
	"strconv"
	"time"
)

type GlobalConfig struct {
	App      AppConfig
	Auth     AuthConfig
	Storage  StorageConfig
	Services ServicesConfig
	Infra    InfraConfig
}

type AppConfig struct {
	Env    string
	Host   string
	Port   string
	Client string
	Server string
}

type AuthConfig struct {
	JWTSecret          string
	EncryptAuthCodeKey []byte
	GoogleClientID     string
	GoogleCLSecret     string
	RedirectURL        string
	SessionDuration    time.Duration
}

type StorageConfig struct {
	DB      DBConfig
	Redis   RedisConfig
	R2      R2Config
	Elastic ElasticsearchConfig
}

type DBConfig struct {
	DBHost     string
	DBUser     string
	DbName     string
	DBPassword string
	DBPort     string
	DBSslmode  string
}

type RedisConfig struct {
	Addr string
}

type R2Config struct {
	AccountID       string
	AccessKeyID     string
	AccessKeySecret string
	BucketName      string
	Endpoint        string
	CDNHost         string
}

type ElasticsearchConfig struct {
	URL      string
	Index    string
	Username string
	Password string
	APIKey   string
}

type ServicesConfig struct {
	Email    EmailConfig
	Claude   ClaudeConfig
	Calendar CalendarConfig
	Payment  PaymentConfig
	FX       FXConfig
}

type EmailConfig struct {
	SendgridAPIKey string
	SendgridSender string
}

type ClaudeConfig struct {
	APIKey   string
	APIURL   string
	APIModel string
}

type CalendarConfig struct {
	MaintenanceHour     int
	WeeklyCheckDay      int
	CleanupOldDays      int
	SchedulerEnabled    bool
	ImmediateGeneration bool
	RetryDelayMinutes   int
}

type PaymentConfig struct {
	PaystackSecretKey    string
	PaystackPublicKey    string
	StripeSecretKey      string
	StripeWebhookSecret  string
	StripePublishableKey string
}

type FXConfig struct {
	APIKey  string
	BaseURL string
}

type InfraConfig struct {
	RabbitMQ RabbitMQConfig
}

type RabbitMQConfig struct {
	Addr string
}

func must(k string) string {
	v, ok := os.LookupEnv(k)
	if !ok || v == "" {
		log.Fatalf("ENV %s is missing", k)
	}
	return v
}

func def(k, d string) string {
	v, ok := os.LookupEnv(k)
	if !ok || v == "" {
		return d
	}
	return v
}

func getInt(k string, d int) int {
	v, ok := os.LookupEnv(k)
	if !ok || v == "" {
		return d
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return d
	}
	return i
}

func getBool(k string, d bool) bool {
	v, ok := os.LookupEnv(k)
	if !ok || v == "" {
		return d
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return d
	}
	return b
}

func mustKey(k string) []byte {
	raw := must(k)
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(key) != 32 {
		log.Fatalf("ENV %s must be a valid 32-byte base64 string", k)
	}
	return key
}
