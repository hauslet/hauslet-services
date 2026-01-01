package service

import (
	"log/slog"

	"hauslet/internal/modules/leads/repository"
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
	analyticsHooks AnalyticsHooks

	log *slog.Logger
}

// NewLeadService creates a new instance of LeadService
func NewLeadService(
	leadRepo repository.LeadRepository,
	eventRepo repository.LeadEventRepository,
	assignmentRepo repository.LeadAssignmentRepository,
	propertyHooks PropertyHooks,
	businessHooks BusinessHooks,
	log *slog.Logger,
) LeadService {
	return &ServiceImpl{
		leadRepo:       leadRepo,
		eventRepo:      eventRepo,
		assignmentRepo: assignmentRepo,
		validator:      NewLeadValidator(),
		spamDetector:   NewSpamDetector(),
		rateLimiter:    NewRateLimiter(leadRepo),
		propertyHooks:  propertyHooks,
		businessHooks:  businessHooks,
		analyticsHooks: &NullAnalyticsHooks{}, // No-op for Phase 1
		log:            log,
	}
}
