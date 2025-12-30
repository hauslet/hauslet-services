package service

import (
	"hauslet/internal/modules/review/notification"
	"hauslet/internal/modules/review/repository"

	moderationservice "hauslet/internal/modules/moderation/service"

	"github.com/go-pkgz/lgr"
)

// ReviewServiceImpl implements ReviewService interface
type ReviewServiceImpl struct {
	// Repositories
	reviewRepo   repository.ReviewRepository
	responseRepo repository.ResponseRepository
	statsRepo    repository.StatsRepository

	// Internal services (same module)
	notificationSvc *notification.NotificationService

	// External dependencies (cross-module)
	bookingQuerier BookingQuerier
	bookingHooks   BookingHooks
	userQuerier    UserQuerier
	moderationSvc  moderationservice.ModerationService

	// Logger
	log *lgr.Logger
}

// NewReviewService creates a new review service instance
func NewReviewService(
	reviewRepo repository.ReviewRepository,
	responseRepo repository.ResponseRepository,
	statsRepo repository.StatsRepository,
	notificationSvc *notification.NotificationService,
	bookingQuerier BookingQuerier,
	bookingHooks BookingHooks,
	userQuerier UserQuerier,
	moderationSvc moderationservice.ModerationService,
	log *lgr.Logger,
) ReviewService {
	return &ReviewServiceImpl{
		reviewRepo:      reviewRepo,
		responseRepo:    responseRepo,
		statsRepo:       statsRepo,
		notificationSvc: notificationSvc,
		bookingQuerier:  bookingQuerier,
		bookingHooks:    bookingHooks,
		userQuerier:     userQuerier,
		moderationSvc:   moderationSvc,
		log:             log,
	}
}
