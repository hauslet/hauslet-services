package service

import (
	"log/slog"

	"hauslet/internal/modules/leads/repository"
	"hauslet/internal/platform/events"
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
	}
}
