package setup

import (
	"context"
	"log/slog"

	"hauslet/config"
	"hauslet/internal/platform/breaker"
	"hauslet/internal/platform/database"
	"hauslet/internal/platform/email"
	"hauslet/internal/platform/evidence"
	"hauslet/internal/platform/kyc"
	"hauslet/internal/platform/queue"
	"hauslet/internal/platform/ratelimit"
	"hauslet/internal/platform/redis"
	"hauslet/internal/platform/sms"
	"hauslet/internal/platform/storage"

	"gorm.io/gorm"
)

// Infrastructure holds shared platform dependencies for the API.
type Infrastructure struct {
	DB      *gorm.DB
	Storage *storage.R2Storage
	Email   *email.Client
	Queue   *queue.Client
	Cache   redis.RedisClient
	// Verification services
	KYC            *kyc.Client
	SMS            *sms.Client
	Evidence       evidence.Store
	RateLimiter    ratelimit.Limiter
	CircuitBreaker breaker.CircuitBreaker
}

// CloseDB closes the DB connection.
func (i *Infrastructure) CloseDB() {
	if i != nil && i.DB != nil {
		database.Close(i.DB)
	}
}

// CloseQueue closes the queue client.
func (i *Infrastructure) CloseQueue() {
	if i != nil && i.Queue != nil {
		i.Queue.Close()
	}
}

// CloseCache closes the Redis cache connection.
func (i *Infrastructure) CloseCache() {
	if i != nil && i.Cache != nil {
		redis.CloseRedis()
	}
}

// InitInfrastructure establishes DB, storage, email, and cache clients.
func InitInfrastructure(ctx context.Context, cfg *config.GlobalConfig, log *slog.Logger) (*Infrastructure, error) {
	db, err := database.NewPostgresWithContext(ctx, &cfg.Storage.DB, cfg.App.Env)
	if err != nil {
		return nil, err
	}
	log.Info(" ✅ Database connected successfully")

	if err := redis.InitRedis(&cfg.Storage.Redis, ctx); err != nil {
		database.Close(db)
		return nil, err
	}
	redisClient, err := redis.GetRedis()
	if err != nil {
		database.Close(db)
		return nil, err
	}
	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		database.Close(db)
		redis.CloseRedis()
		return nil, err
	}
	log.Info(" ✅ Redis connected successfully")

	storageClient, err := storage.NewR2S3Client(cfg.Storage.R2)
	if err != nil {
		database.Close(db)
		redis.CloseRedis()
		return nil, err
	}
	r2Storage := storage.NewR2Storage(storageClient, cfg.Storage.R2.BucketName)
	log.Info(" ✅ Cloudflare R2 storage client initialized")

	emailClient := InitializeEmailClient(cfg, log)
	log.Info(" ✅ Email client initialized")

	// Initialize verification services
	kycClient := InitKYCClient(cfg, log)
	log.Info(" ✅ KYC client initialized (Dojah + Veriff)")

	smsClient := InitSMSClient(cfg, log)
	log.Info(" ✅ SMS client initialized (Termii + Twilio)")

	evidenceStore := InitEvidenceStore(r2Storage)
	log.Info(" ✅ Evidence store initialized")

	rateLimiter := InitRateLimiter(redisClient, cfg)
	log.Info(" ✅ Rate limiter initialized")

	circuitBreaker, err := InitCircuitBreaker(ctx, redisClient, log)
	if err != nil {
		database.Close(db)
		redis.CloseRedis()
		return nil, err
	}
	log.Info(" ✅ Circuit breaker initialized")

	return &Infrastructure{
		DB:             db,
		Storage:        r2Storage,
		Email:          emailClient,
		Cache:          redisClient,
		KYC:            kycClient,
		SMS:            smsClient,
		Evidence:       evidenceStore,
		RateLimiter:    rateLimiter,
		CircuitBreaker: circuitBreaker,
	}, nil
}
