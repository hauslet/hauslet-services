package service

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"hauslet/config"
	"hauslet/internal/modules/finance/domain"
	"hauslet/internal/modules/finance/repository"
	financeSchema "hauslet/internal/modules/finance/repository/schema"
	paymentsSchema "hauslet/internal/modules/payments/repository/schema"
	"hauslet/internal/platform/payment"

	"github.com/google/uuid"
	"github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log/slog"
)

func TestProcessSinglePayoutImmediateDisbursement(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newFinanceTestDB(t)
	walletRepo := repository.NewWalletRepository(db)
	ledgerRepo := repository.NewLedgerRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	disbursementRepo := repository.NewDisbursementRepository(db)

	now := time.Now()
	bookingID := uuid.New()
	hostID := uuid.New()
	escrowWallet := &financeSchema.Wallet{
		ID:         uuid.New(),
		OwnerType:  string(domain.OwnerTypeUser),
		OwnerID:    bookingID,
		WalletType: string(domain.WalletTypeEscrow),
		Balance:    200000,
		Currency:   "NGN",
		Status:     string(domain.WalletStatusActive),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, walletRepo.Create(ctx, escrowWallet))

	platformID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("hauslet"))
	platformWallet := &financeSchema.Wallet{
		ID:         platformID,
		OwnerType:  string(domain.OwnerTypePlatform),
		OwnerID:    platformID,
		WalletType: string(domain.WalletTypePlatformFee),
		Balance:    0,
		Currency:   "NGN",
		Status:     string(domain.WalletStatusActive),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, walletRepo.Create(ctx, platformWallet))

	testLogger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))

	platformConfig := config.PlatformYAMLConfig{
		Fees: config.PlatformFeesConfig{
			HostCommissionPercent: 10,
		},
		Payouts: config.PlatformPayoutConfig{
			EscrowReleaseHours:   48,
			EscrowReleaseEvent:   config.EscrowReleaseCheckoutConfirmed,
			DisbursementProvider: "paystack",
			MaxRetryAttempts:     5,
			RetryBackoffMinutes:  15,
		},
	}

	const payoutAmount = int64(100000)
	commission := calculateCommission(payoutAmount, platformConfig.Fees.HostCommissionPercent)
	hostPayout := payoutAmount - commission
	const currency = "NGN"

	payoutClient := payment.New(&testPaymentFactory{
		payout: &stubPayoutClient{
			response: &payment.PayoutResponse{
				Success:    true,
				TransferID: "test-transfer-id",
				Status:     payment.TransferSuccess,
				Amount:     hostPayout,
				Currency:   payment.NGN,
				Reference:  "",
				Message:    "stub transfer successful",
			},
		},
	})

	payoutDetail := &paymentsSchema.PayoutDetail{
		ID:            uuid.New(),
		UserID:        &hostID,
		BankCode:      "001",
		BankName:      "Test Bank",
		AccountNumber: "1234567890",
		AccountName:   "Test Host",
		Currency:      "NGN",
		Market:        paymentsSchema.MarketNigeria,
		Provider:      "paystack",
		RecipientCode: "rcpt-test",
		IsVerified:    true,
		IsDefault:     true,
		IsActive:      true,
	}

	payoutSvc := NewPayoutService(
		walletRepo,
		ledgerRepo,
		transactionRepo,
		disbursementRepo,
		&stubPayoutDetailRepo{detail: payoutDetail},
		nil,
		nil,
		nil,
		payoutClient,
		nil,
		platformConfig,
		db,
		testLogger,
	)
	svcImpl := payoutSvc.(*PayoutServiceImpl)

	err := svcImpl.processSinglePayout(ctx, bookingID, hostID, escrowWallet.ID, payoutAmount, currency)
	require.NoError(t, err)

	hostWallet, err := walletRepo.GetByOwner(ctx, string(domain.OwnerTypeUser), hostID, string(domain.WalletTypeHostAvailable))
	require.NoError(t, err)
	require.NotNil(t, hostWallet)
	require.Equal(t, hostPayout, hostWallet.Balance)

	var recorded financeSchema.Disbursement
	require.NoError(t, db.WithContext(ctx).First(&recorded).Error)
	require.Equal(t, string(domain.DisbursementStatusCompleted), recorded.Status)
	require.Equal(t, hostPayout, recorded.Amount)
	require.Equal(t, currency, recorded.Currency)
	require.Equal(t, hostWallet.ID, recorded.WalletID)
	require.NotNil(t, recorded.CompletedAt)
}

func newFinanceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	registerSQLiteWithNow()
	rawDB, err := sql.Open("sqlite3_with_now", "file::memory:?cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() {
		rawDB.Close()
	})

	gormDialector := sqlite.New(sqlite.Config{
		Conn:       rawDB,
		DriverName: "sqlite3_with_now",
	})

	db, err := gorm.Open(gormDialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	schemaStatements := []string{
		`CREATE TABLE wallets (
			id TEXT PRIMARY KEY,
			owner_type TEXT NOT NULL,
			owner_id TEXT NOT NULL,
			wallet_type TEXT NOT NULL,
			balance INTEGER NOT NULL,
			currency TEXT NOT NULL,
			status TEXT NOT NULL,
			metadata TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE ledger_entries (
			id TEXT PRIMARY KEY,
			transaction_id TEXT NOT NULL,
			reference TEXT NOT NULL,
			debit_wallet_id TEXT,
			credit_wallet_id TEXT,
			amount INTEGER NOT NULL,
			currency TEXT NOT NULL,
			resource_type TEXT NOT NULL,
			resource_id TEXT NOT NULL,
			memo TEXT,
			created_at DATETIME NOT NULL
		)`,
		`CREATE TABLE finance_transactions (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			status TEXT NOT NULL,
			resource_type TEXT NOT NULL,
			resource_id TEXT NOT NULL,
			amount INTEGER NOT NULL,
			currency TEXT NOT NULL,
			payment_id TEXT,
			error_message TEXT,
			metadata TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE disbursements (
			id TEXT PRIMARY KEY,
			wallet_id TEXT NOT NULL,
			transaction_id TEXT NOT NULL,
			amount INTEGER NOT NULL,
			currency TEXT NOT NULL,
			provider TEXT NOT NULL,
			transfer_code TEXT,
			provider_response TEXT,
			status TEXT NOT NULL,
			attempts INTEGER NOT NULL,
			next_retry_at DATETIME,
			completed_at DATETIME,
			failure_reason TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
	}
	for _, stmt := range schemaStatements {
		require.NoError(t, db.Exec(stmt).Error)
	}

	return db
}

var (
	registerSQLiteWithNowOnce sync.Once
)

func registerSQLiteWithNow() {
	registerSQLiteWithNowOnce.Do(func() {
		sql.Register("sqlite3_with_now", &sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				return conn.RegisterFunc("NOW", func() string {
					return time.Now().UTC().Format("2006-01-02 15:04:05")
				}, true)
			},
		})
	})
}

type stubPayoutDetailRepo struct {
	detail *paymentsSchema.PayoutDetail
}

func (s *stubPayoutDetailRepo) CreatePayoutDetail(ctx context.Context, pd *paymentsSchema.PayoutDetail) error {
	return nil
}

func (s *stubPayoutDetailRepo) GetPayoutDetailByID(ctx context.Context, id uuid.UUID) (*paymentsSchema.PayoutDetail, error) {
	return nil, nil
}

func (s *stubPayoutDetailRepo) GetPayoutDetailByRecipientCode(ctx context.Context, recipientCode string) (*paymentsSchema.PayoutDetail, error) {
	return nil, nil
}

func (s *stubPayoutDetailRepo) UpdatePayoutDetail(ctx context.Context, pd *paymentsSchema.PayoutDetail) error {
	return nil
}

func (s *stubPayoutDetailRepo) DeletePayoutDetail(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (s *stubPayoutDetailRepo) ListPayoutDetailsByUserID(ctx context.Context, userID uuid.UUID) ([]*paymentsSchema.PayoutDetail, error) {
	return nil, nil
}

func (s *stubPayoutDetailRepo) ListPayoutDetailsByBusinessID(ctx context.Context, businessID uuid.UUID) ([]*paymentsSchema.PayoutDetail, error) {
	return nil, nil
}

func (s *stubPayoutDetailRepo) GetDefaultPayoutDetail(ctx context.Context, userID *uuid.UUID, businessID *uuid.UUID) (*paymentsSchema.PayoutDetail, error) {
	if s.detail == nil {
		return nil, nil
	}
	if userID != nil && s.detail.UserID != nil && *userID == *s.detail.UserID {
		return s.detail, nil
	}
	return nil, nil
}

func (s *stubPayoutDetailRepo) SetDefaultPayoutDetail(ctx context.Context, id uuid.UUID, userID *uuid.UUID, businessID *uuid.UUID) error {
	return nil
}

type stubPayoutClient struct {
	response *payment.PayoutResponse
}

func (s *stubPayoutClient) ValidateAccount(ctx context.Context, bankCode, accountNumber string) (string, error) {
	return "Test Host", nil
}

func (s *stubPayoutClient) CreateRecipient(ctx context.Context, bankCode, accountNumber, accountName string) (string, error) {
	return "rcpt-test", nil
}

func (s *stubPayoutClient) Transfer(ctx context.Context, req payment.PayoutRequest) (*payment.PayoutResponse, error) {
	if s.response == nil {
		return nil, fmt.Errorf("no stub response configured")
	}
	s.response.Reference = req.Reference
	return s.response, nil
}

func (s *stubPayoutClient) VerifyTransfer(ctx context.Context, reference string) (*payment.PayoutResponse, error) {
	return s.response, nil
}

func (s *stubPayoutClient) ListBanks(ctx context.Context, currency payment.Currency, country string) ([]payment.Bank, error) {
	return nil, nil
}

type testPaymentFactory struct {
	payout payment.PayoutClient
}

func (f *testPaymentFactory) GetTransactionClient(currency payment.Currency) (payment.TransactionClient, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *testPaymentFactory) GetPayoutClient(currency payment.Currency) (payment.PayoutClient, error) {
	if f.payout == nil {
		return nil, fmt.Errorf("no payout client configured")
	}
	return f.payout, nil
}

func (f *testPaymentFactory) GetWebhookHandler(provider string) (payment.WebhookHandler, error) {
	return nil, fmt.Errorf("not implemented")
}
