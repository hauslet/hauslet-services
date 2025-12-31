package service

import (
	"hauslet/internal/modules/payments/notification"
	"hauslet/internal/modules/payments/repository"
	"hauslet/internal/platform/payment"
	"hauslet/internal/platform/redis"
	"log/slog"
)

// PaymentServiceImpl implements the PaymentService interface
type PaymentServiceImpl struct {
	repo            repository.Repository
	paymentClient   *payment.Client
	notificationSvc *notification.NotificationService
	cache           redis.RedisClient
	log             *slog.Logger
}

// NewPaymentService creates a new payment service instance
func NewPaymentService(
	repo repository.Repository,
	paymentClient *payment.Client,
	notificationSvc *notification.NotificationService,
	cache redis.RedisClient,
	log *slog.Logger,
) PaymentService {
	return &PaymentServiceImpl{
		repo:            repo,
		paymentClient:   paymentClient,
		notificationSvc: notificationSvc,
		cache:           cache,
		log:             log,
	}
}
