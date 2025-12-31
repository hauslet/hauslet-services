package service

import (
	"hauslet/internal/modules/payments/notification"
	"hauslet/internal/modules/payments/repository"
	"hauslet/internal/platform/payment"
)

// newTestService creates a service with test dependencies
// This allows us to inject mocks while maintaining the concrete type requirements
func newTestService(
	repo repository.Repository,
	paymentClient *payment.Client,
	notificationSvc *notification.NotificationService,
) *PaymentServiceImpl {

	return &PaymentServiceImpl{
		repo:            repo,
		paymentClient:   paymentClient,
		notificationSvc: notificationSvc,
		log:             nil,
	}
}
