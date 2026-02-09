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
	YAML     *ServiceConfig // YAML-based service configuration
}

type AppConfig struct {
	Env    string
	Port   string
	Client string
	Server string
}

type AuthConfig struct {
	Env                string
	JWTSecret          string
	EncryptAuthCodeKey []byte
	GoogleClientID     string
	GoogleCLSecret     string
	ClientRedirectURL  string
	ServerURL          string
	SessionDuration    time.Duration

	// Security settings (configurable via env vars)
	TokenDuration   time.Duration // JWT token validity
	CookieDuration  time.Duration // Cookie validity
	AvatarStorePath string        // Avatar storage location
	DisableXSRF     bool          // XSRF protection toggle
	CookieDomain    string        // Domain to scope auth cookies (e.g., .yourdomain.com for subdomains)
}

type StorageConfig struct {
	DB    DBConfig
	Redis RedisConfig
	R2    R2Config
}

type DBConfig struct {
	DatabaseURL string
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

type ServicesConfig struct {
	Email     EmailConfig
	Anthropic AnthropicConfig
	Gemini    GeminiConfig
	Fallback  FallbackConfig
	Calendar  CalendarConfig
	Payment   PaymentConfig
	FX        FXConfig
	KYC       KYCConfig
	SMS       SMSConfig
	VertexAI  VertexAIConfig
	Messaging MessagingConfig
}

type EmailConfig struct {
	From   string
	SMTP   SMTPConfig
	Resend ResendConfig
}

type SMTPConfig struct {
	Host string
	Port int
	User string
	Pass string
}

type ResendConfig struct {
	APIKey string
}

type AnthropicConfig struct {
	APIKey   string
	APIURL   string
	APIModel string
}

type GeminiConfig struct {
	APIKey     string
	APIModel   string
	Project    string
	EmbedModel string
}

type FallbackConfig struct {
	Enable           bool
	AttemptThreshold int
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
	PaystackSecretKey        string
	PaystackPublicKey        string
	FlutterwaveWebhookSecret string
	FlutterwaveOAuthClientID string
	FlutterwaveOAuthSecret   string
	FlutterwaveBaseURL       string // optional: override sandbox/live base URL
}

type FXConfig struct {
	APIKey  string
	BaseURL string
}

type KYCConfig struct {
	DojahAPIKey         string
	DojahSecretKey      string
	DojahWebhookSecret  string
	VeriffAPIKey        string
	VeriffSecretKey     string
	VeriffWebhookSecret string
}

type SMSConfig struct {
	TermiiAPIKey     string
	TermiiSenderID   string
	TermiiBaseURL    string
	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioFromNumber string
	TwilioBaseURL    string
}

type VertexAIConfig struct {
	ProjectID       string
	AgentID         string
	Location        string
	CredentialsPath string
}

type MessagingConfig struct {
	AIContextMessages      int
	AIConfidenceThreshold  float64
	AITimeoutSeconds       int
	MaxConversationAgeDays int
	AIServingConfig        string
}

type InfraConfig struct {
	CloudTasks CloudTasksConfig
}

type CloudTasksConfig struct {
	ProjectID           string
	Location            string
	WorkerBaseURL       string
	ServiceAccountEmail string
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

func getFloat(k string, d float64) float64 {
	v, ok := os.LookupEnv(k)
	if !ok || v == "" {
		return d
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return d
	}
	return f
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

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Fatalf("Invalid duration: %s (error: %v)", s, err)
	}
	return d
}

func getDuration(k, defaultVal string) time.Duration {
	v := def(k, defaultVal)
	return parseDuration(v)
}
