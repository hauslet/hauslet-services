package service

import (
	"log/slog"

	"hauslet/internal/modules/leads/repository"
	aiassist "hauslet/internal/platform/ai/assist"
	"hauslet/internal/platform/events"
	"hauslet/internal/platform/queue"
	"hauslet/internal/platform/ratelimit"
)

// ServiceImpl implements the LeadService interface
type ServiceImpl struct {
	leadRepo       repository.LeadRepository
	eventRepo      repository.LeadEventRepository
	assignmentRepo repository.LeadAssignmentRepository

	validator    *LeadValidator
	spamDetector *SpamDetector
	rateLimiter  *RateLimiter

	propertyHooks  PropertyHooks
	businessHooks  BusinessHooks
	profileHooks   ProfileHooks // For hybrid authentication (optional)
	analyticsHooks AnalyticsHooks

	log            *slog.Logger
	eventPublisher *events.Publisher // For domain events
	queue          *queue.Client     // For background jobs (AI qualification)
	aiAssist       aiassist.AssistClient
}

// NewLeadService creates a new instance of LeadService
func NewLeadService(
	leadRepo repository.LeadRepository,
	eventRepo repository.LeadEventRepository,
	assignmentRepo repository.LeadAssignmentRepository,
	propertyHooks PropertyHooks,
	businessHooks BusinessHooks,
	profileHooks ProfileHooks, // Optional - for hybrid authentication
	limiter ratelimit.Limiter,
	rateLimitConfig RateLimitConfig,
	eventPublisher *events.Publisher, // For domain events
	queue *queue.Client,
	aiAssist aiassist.AssistClient,
	log *slog.Logger,
) LeadService {
	return &ServiceImpl{
		leadRepo:       leadRepo,
		eventRepo:      eventRepo,
		assignmentRepo: assignmentRepo,
		validator:      NewLeadValidator(),
		spamDetector:   NewSpamDetector(),
		rateLimiter:    NewRateLimiter(limiter, rateLimitConfig),
		propertyHooks:  propertyHooks,
		businessHooks:  businessHooks,
		profileHooks:   profileHooks,
		analyticsHooks: &NullAnalyticsHooks{}, // No-op for Phase 1
		log:            log,
		eventPublisher: eventPublisher,
		queue:          queue,
		aiAssist:       aiAssist,
	}
}
