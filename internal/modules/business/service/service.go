package service

import (
	"hauslet/internal/modules/business/notification"
	"hauslet/internal/modules/business/repository"

	"github.com/go-pkgz/lgr"
)

// BusinessServiceImpl implements BusinessService
type BusinessServiceImpl struct {
	repo        repository.BusinessRepository
	log         *lgr.Logger
	notifier    *notification.NotificationService
}

// NewBusinessService creates a new business service
func NewBusinessService(repo repository.BusinessRepository, log *lgr.Logger) BusinessService {
	return &BusinessServiceImpl{
		repo: repo,
		log:  log,
	}
}

// WithNotificationService adds notification service to the business service
func (s *BusinessServiceImpl) WithNotificationService(notifier *notification.NotificationService) *BusinessServiceImpl {
	s.notifier = notifier
	return s
}
