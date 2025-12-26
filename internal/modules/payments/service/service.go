package service

import (
	"hauslet/internal/modules/payments/notification"
	"hauslet/internal/modules/payments/repository"
	"hauslet/internal/platform/payment"

	"github.com/go-pkgz/lgr"
)

// PaymentServiceImpl implements the PaymentService interface
type PaymentServiceImpl struct {
	repo            repository.Repository
	paymentClient   *payment.Client
	notificationSvc *notification.NotificationService
	log             *lgr.Logger
}

// NewPaymentService creates a new payment service instance
func NewPaymentService(
	repo repository.Repository,
	paymentClient *payment.Client,
	notificationSvc *notification.NotificationService,
	log *lgr.Logger,
) PaymentService {
	return &PaymentServiceImpl{
		repo:            repo,
		paymentClient:   paymentClient,
		notificationSvc: notificationSvc,
		log:             log,
	}
}
