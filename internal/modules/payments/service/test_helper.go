package service

import (
	"hauslet/internal/modules/payments/notification"
	"hauslet/internal/modules/payments/repository"
	"hauslet/internal/platform/payment"

	"github.com/go-pkgz/lgr"
)

// newTestService creates a service with test dependencies
// This allows us to inject mocks while maintaining the concrete type requirements
func newTestService(
	repo repository.Repository,
	paymentClient *payment.Client,
	notificationSvc *notification.NotificationService,
) *PaymentServiceImpl {
	logger := lgr.New(lgr.Msec)

	return &PaymentServiceImpl{
		repo:            repo,
		paymentClient:   paymentClient,
		notificationSvc: notificationSvc,
		log:             logger,
	}
}
