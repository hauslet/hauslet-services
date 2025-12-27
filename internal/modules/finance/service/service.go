package service

import (
	"context"
	"hauslet/config"
	"hauslet/internal/modules/finance/notification"
	"hauslet/internal/modules/finance/repository"
	paymentsRepository "hauslet/internal/modules/payments/repository"
	"hauslet/internal/platform/payment"

	"github.com/go-pkgz/lgr"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FinanceServiceImpl implements the FinanceService interface
type FinanceServiceImpl struct {
	walletRepo       repository.WalletRepository
	ledgerRepo       repository.LedgerRepository
	transactionRepo  repository.TransactionRepository
	disbursementRepo repository.DisbursementRepository
	db               *gorm.DB
	log              *lgr.Logger
}

// NewFinanceService creates a new finance service
func NewFinanceService(
	walletRepo repository.WalletRepository,
	ledgerRepo repository.LedgerRepository,
	transactionRepo repository.TransactionRepository,
	disbursementRepo repository.DisbursementRepository,
	db *gorm.DB,
	log *lgr.Logger,
) FinanceService {
	return &FinanceServiceImpl{
		walletRepo:       walletRepo,
		ledgerRepo:       ledgerRepo,
		transactionRepo:  transactionRepo,
		disbursementRepo: disbursementRepo,
		db:               db,
		log:              log,
	}
}

// ProfileAdapter defines the interface for getting user profile data
type ProfileAdapter interface {
	GetProfileData(ctx context.Context, userID uuid.UUID) (string, string, error)
}

// BookingForPayout represents minimal booking data needed for payout processing
type BookingForPayout struct {
	ID            uuid.UUID
	HostID        uuid.UUID
	TotalAmount   int64
	Currency      string
	LastPaymentID uuid.UUID
}

// BookingQuerier defines the interface for querying booking data for payouts
type BookingQuerier interface {
	FindBookingsReadyForPayout(ctx context.Context, payoutWindowHours int, limit int) ([]*BookingForPayout, error)
}

// PayoutServiceImpl implements PayoutService
type PayoutServiceImpl struct {
	walletRepo       repository.WalletRepository
	ledgerRepo       repository.LedgerRepository
	transactionRepo  repository.TransactionRepository
	disbursementRepo repository.DisbursementRepository
	payoutDetailRepo paymentsRepository.PayoutDetailRepository
	bookingQuerier   BookingQuerier
	notificationSvc  *notification.NotificationService
	bookingHooks     BookingHooks
	paymentClient    *payment.Client
	profileAdapter   ProfileAdapter
	platformConfig   config.PlatformYAMLConfig
	db               *gorm.DB
	log              *lgr.Logger
}

// NewPayoutService creates a new payout service
func NewPayoutService(
	walletRepo repository.WalletRepository,
	ledgerRepo repository.LedgerRepository,
	transactionRepo repository.TransactionRepository,
	disbursementRepo repository.DisbursementRepository,
	payoutDetailRepo paymentsRepository.PayoutDetailRepository,
	bookingQuerier BookingQuerier,
	notificationSvc *notification.NotificationService,
	bookingHooks BookingHooks,
	paymentClient *payment.Client,
	profileAdapter ProfileAdapter,
	platformConfig config.PlatformYAMLConfig,
	db *gorm.DB,
	log *lgr.Logger,
) PayoutService {
	return &PayoutServiceImpl{
		walletRepo:       walletRepo,
		ledgerRepo:       ledgerRepo,
		transactionRepo:  transactionRepo,
		disbursementRepo: disbursementRepo,
		payoutDetailRepo: payoutDetailRepo,
		bookingQuerier:   bookingQuerier,
		notificationSvc:  notificationSvc,
		bookingHooks:     bookingHooks,
		paymentClient:    paymentClient,
		profileAdapter:   profileAdapter,
		platformConfig:   platformConfig,
		db:               db,
		log:              log,
	}
}
