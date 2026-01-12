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

// TestWalletBalanceUpdateUsesAbsoluteValues verifies that wallet balance updates
// use absolute new balance values, not delta amounts. This test prevents the bug
// where UpdateBalance was called with -amount or +amount instead of oldBalance +/- amount
func TestWalletBalanceUpdateUsesAbsoluteValues(t *testing.T) {

	ctx := context.Background()
	db := newFinanceTestDB(t)
	walletRepo := repository.NewWalletRepository(db)
	ledgerRepo := repository.NewLedgerRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)

	now := time.Now()
	bookingID := uuid.New()
	platformID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("hauslet"))

	// Create escrow wallet with initial balance
	const initialEscrowBalance = int64(500000) // 5000 NGN
	escrowWallet := &financeSchema.Wallet{
		ID:         uuid.New(),
		OwnerType:  string(domain.OwnerTypeUser),
		OwnerID:    bookingID,
		WalletType: string(domain.WalletTypeEscrow),
		Balance:    initialEscrowBalance,
		Currency:   "NGN",
		Status:     string(domain.WalletStatusActive),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, walletRepo.Create(ctx, escrowWallet))

	// Create platform wallet with initial balance
	const initialPlatformBalance = int64(1000000) // 10000 NGN
	platformWallet := &financeSchema.Wallet{
		ID:         platformID,
		OwnerType:  string(domain.OwnerTypePlatform),
		OwnerID:    platformID,
		WalletType: string(domain.WalletTypePlatformFee),
		Balance:    initialPlatformBalance,
		Currency:   "NGN",
		Status:     string(domain.WalletStatusActive),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, walletRepo.Create(ctx, platformWallet))

	testLogger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))

	payoutSvc := &PayoutServiceImpl{
		walletRepo:      walletRepo,
		ledgerRepo:      ledgerRepo,
		transactionRepo: transactionRepo,
		db:              db,
		log:             testLogger,
	}

	// Process commission of 20000 (200 NGN)
	const commissionAmount = int64(20000)
	err := db.Transaction(func(tx *gorm.DB) error {
		_, err := payoutSvc.recordCommissionInternal(
			ctx,
			tx,
			bookingID,
			escrowWallet.ID,
			platformWallet.ID,
			commissionAmount,
			"NGN",
		)
		return err
	})
	require.NoError(t, err)

	// Verify escrow wallet balance decreased by commission amount
	updatedEscrow, err := walletRepo.GetByID(ctx, escrowWallet.ID)
	require.NoError(t, err)
	require.NotNil(t, updatedEscrow)
	expectedEscrowBalance := initialEscrowBalance - commissionAmount
	require.Equal(t, expectedEscrowBalance, updatedEscrow.Balance,
		"Escrow balance should be %d (initial %d - commission %d), got %d",
		expectedEscrowBalance, initialEscrowBalance, commissionAmount, updatedEscrow.Balance)

	// Verify platform wallet balance increased by commission amount
	updatedPlatform, err := walletRepo.GetByID(ctx, platformWallet.ID)
	require.NoError(t, err)
	require.NotNil(t, updatedPlatform)
	expectedPlatformBalance := initialPlatformBalance + commissionAmount
	require.Equal(t, expectedPlatformBalance, updatedPlatform.Balance,
		"Platform balance should be %d (initial %d + commission %d), got %d",
		expectedPlatformBalance, initialPlatformBalance, commissionAmount, updatedPlatform.Balance)

	// Verify ledger entries were created
	var ledgerCount int64
	require.NoError(t, db.Model(&financeSchema.LedgerEntry{}).Count(&ledgerCount).Error)
	require.Equal(t, int64(2), ledgerCount, "Should have 2 ledger entries (debit + credit)")
}

// TestPayoutInternalWalletBalances verifies that payout internal function
// correctly updates wallet balances using absolute values
func TestPayoutInternalWalletBalances(t *testing.T) {

	ctx := context.Background()
	db := newFinanceTestDB(t)
	walletRepo := repository.NewWalletRepository(db)
	ledgerRepo := repository.NewLedgerRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)

	now := time.Now()
	bookingID := uuid.New()
	hostID := uuid.New()

	// Create escrow wallet with initial balance
	const initialEscrowBalance = int64(300000) // 3000 NGN
	escrowWallet := &financeSchema.Wallet{
		ID:         uuid.New(),
		OwnerType:  string(domain.OwnerTypeUser),
		OwnerID:    bookingID,
		WalletType: string(domain.WalletTypeEscrow),
		Balance:    initialEscrowBalance,
		Currency:   "NGN",
		Status:     string(domain.WalletStatusActive),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, walletRepo.Create(ctx, escrowWallet))

	// Create host wallet with initial balance
	const initialHostBalance = int64(750000) // 7500 NGN
	hostWallet := &financeSchema.Wallet{
		ID:         uuid.New(),
		OwnerType:  string(domain.OwnerTypeUser),
		OwnerID:    hostID,
		WalletType: string(domain.WalletTypeHostAvailable),
		Balance:    initialHostBalance,
		Currency:   "NGN",
		Status:     string(domain.WalletStatusActive),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, walletRepo.Create(ctx, hostWallet))

	testLogger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))

	payoutSvc := &PayoutServiceImpl{
		walletRepo:      walletRepo,
		ledgerRepo:      ledgerRepo,
		transactionRepo: transactionRepo,
		db:              db,
		log:             testLogger,
	}

	// Process payout of 150000 (1500 NGN)
	const payoutAmount = int64(150000)
	err := db.Transaction(func(tx *gorm.DB) error {
		_, err := payoutSvc.recordPayoutInternal(
			ctx,
			tx,
			bookingID,
			escrowWallet.ID,
			hostWallet.ID,
			payoutAmount,
			"NGN",
		)
		return err
	})
	require.NoError(t, err)

	// Verify escrow wallet balance decreased by payout amount
	updatedEscrow, err := walletRepo.GetByID(ctx, escrowWallet.ID)
	require.NoError(t, err)
	require.NotNil(t, updatedEscrow)
	expectedEscrowBalance := initialEscrowBalance - payoutAmount
	require.Equal(t, expectedEscrowBalance, updatedEscrow.Balance,
		"Escrow balance should be %d (initial %d - payout %d), got %d",
		expectedEscrowBalance, initialEscrowBalance, payoutAmount, updatedEscrow.Balance)

	// Verify host wallet balance increased by payout amount
	updatedHost, err := walletRepo.GetByID(ctx, hostWallet.ID)
	require.NoError(t, err)
	require.NotNil(t, updatedHost)
	expectedHostBalance := initialHostBalance + payoutAmount
	require.Equal(t, expectedHostBalance, updatedHost.Balance,
		"Host balance should be %d (initial %d + payout %d), got %d",
		expectedHostBalance, initialHostBalance, payoutAmount, updatedHost.Balance)

	// Verify ledger entries were created
	var ledgerCount int64
	require.NoError(t, db.Model(&financeSchema.LedgerEntry{}).Count(&ledgerCount).Error)
	require.Equal(t, int64(2), ledgerCount, "Should have 2 ledger entries (debit + credit)")
}

// TestReconciliationPassesAfterPayouts verifies that the reconciliation
// validation passes after payouts are processed, ensuring ledger entries
// balance correctly with wallet updates
func TestReconciliationPassesAfterPayouts(t *testing.T) {

	ctx := context.Background()
	db := newFinanceTestDB(t)
	walletRepo := repository.NewWalletRepository(db)
	ledgerRepo := repository.NewLedgerRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)

	now := time.Now()
	bookingID := uuid.New()
	hostID := uuid.New()
	platformID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("hauslet"))

	// Create wallets - escrow starts with balance (simulating previous charge)
	// We don't create ledger entries for this initial balance to keep the test focused
	// on testing the payout internal functions, not the charge flow
	const initialEscrowBalance = int64(300000)
	escrowWallet := &financeSchema.Wallet{
		ID:         uuid.New(),
		OwnerType:  string(domain.OwnerTypeUser),
		OwnerID:    bookingID,
		WalletType: string(domain.WalletTypeEscrow),
		Balance:    initialEscrowBalance,
		Currency:   "NGN",
		Status:     string(domain.WalletStatusActive),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, walletRepo.Create(ctx, escrowWallet))

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

	hostWallet := &financeSchema.Wallet{
		ID:         uuid.New(),
		OwnerType:  string(domain.OwnerTypeUser),
		OwnerID:    hostID,
		WalletType: string(domain.WalletTypeHostAvailable),
		Balance:    0,
		Currency:   "NGN",
		Status:     string(domain.WalletStatusActive),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, walletRepo.Create(ctx, hostWallet))

	testLogger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))

	payoutSvc := &PayoutServiceImpl{
		walletRepo:      walletRepo,
		ledgerRepo:      ledgerRepo,
		transactionRepo: transactionRepo,
		db:              db,
		log:             testLogger,
	}

	// Record commission and payout
	const commissionAmount = int64(50000)
	const payoutAmount = int64(100000)

	err := db.Transaction(func(tx *gorm.DB) error {
		// Record commission
		_, err := payoutSvc.recordCommissionInternal(
			ctx,
			tx,
			bookingID,
			escrowWallet.ID,
			platformWallet.ID,
			commissionAmount,
			"NGN",
		)
		if err != nil {
			return err
		}

		// Record payout
		_, err = payoutSvc.recordPayoutInternal(
			ctx,
			tx,
			bookingID,
			escrowWallet.ID,
			hostWallet.ID,
			payoutAmount,
			"NGN",
		)
		return err
	})
	require.NoError(t, err)

	// Run reconciliation validation queries
	type TransactionBalance struct {
		TransactionID uuid.UUID
		TotalDebit    int64
		TotalCredit   int64
	}

	// Check for imbalanced transactions
	var imbalances []TransactionBalance
	err = db.WithContext(ctx).Raw(`
		SELECT
			transaction_id,
			COALESCE(SUM(CASE WHEN debit_wallet_id IS NOT NULL AND debit_wallet_id != '' THEN amount ELSE 0 END), 0) as total_debit,
			COALESCE(SUM(CASE WHEN credit_wallet_id IS NOT NULL AND credit_wallet_id != '' THEN amount ELSE 0 END), 0) as total_credit
		FROM ledger_entries
		GROUP BY transaction_id
		HAVING SUM(CASE WHEN debit_wallet_id IS NOT NULL AND debit_wallet_id != '' THEN amount ELSE 0 END) !=
		       SUM(CASE WHEN credit_wallet_id IS NOT NULL AND credit_wallet_id != '' THEN amount ELSE 0 END)
	`).Scan(&imbalances).Error
	require.NoError(t, err)
	require.Empty(t, imbalances, "No transactions should have imbalanced ledger entries")

	// Check wallet balance integrity - for wallets touched by our transactions
	// Platform wallet: should match ledger entries (started at 0)
	var platformLedgerSum int64
	err = db.WithContext(ctx).Raw(`
		SELECT COALESCE(
			SUM(CASE WHEN credit_wallet_id = ? THEN amount ELSE 0 END) -
			SUM(CASE WHEN debit_wallet_id = ? THEN amount ELSE 0 END),
			0
		)
		FROM ledger_entries
	`, platformWallet.ID, platformWallet.ID).Scan(&platformLedgerSum).Error
	require.NoError(t, err)

	finalPlatformCheck, err := walletRepo.GetByID(ctx, platformWallet.ID)
	require.NoError(t, err)
	require.Equal(t, platformLedgerSum, finalPlatformCheck.Balance,
		"Platform wallet balance should equal ledger entries sum")

	// Host wallet: should match ledger entries (started at 0)
	var hostLedgerSum int64
	err = db.WithContext(ctx).Raw(`
		SELECT COALESCE(
			SUM(CASE WHEN credit_wallet_id = ? THEN amount ELSE 0 END) -
			SUM(CASE WHEN debit_wallet_id = ? THEN amount ELSE 0 END),
			0
		)
		FROM ledger_entries
	`, hostWallet.ID, hostWallet.ID).Scan(&hostLedgerSum).Error
	require.NoError(t, err)

	finalHostCheck, err := walletRepo.GetByID(ctx, hostWallet.ID)
	require.NoError(t, err)
	require.Equal(t, hostLedgerSum, finalHostCheck.Balance,
		"Host wallet balance should equal ledger entries sum")

	// Escrow wallet: balance should equal initial + ledger entries
	var escrowLedgerSum int64
	err = db.WithContext(ctx).Raw(`
		SELECT COALESCE(
			SUM(CASE WHEN credit_wallet_id = ? THEN amount ELSE 0 END) -
			SUM(CASE WHEN debit_wallet_id = ? THEN amount ELSE 0 END),
			0
		)
		FROM ledger_entries
	`, escrowWallet.ID, escrowWallet.ID).Scan(&escrowLedgerSum).Error
	require.NoError(t, err)

	finalEscrowCheck, err := walletRepo.GetByID(ctx, escrowWallet.ID)
	require.NoError(t, err)
	require.Equal(t, initialEscrowBalance+escrowLedgerSum, finalEscrowCheck.Balance,
		"Escrow balance should equal initial balance + ledger entries sum")

	// Verify final balances
	finalEscrow, err := walletRepo.GetByID(ctx, escrowWallet.ID)
	require.NoError(t, err)
	require.Equal(t, initialEscrowBalance-commissionAmount-payoutAmount, finalEscrow.Balance,
		"Escrow should have %d - %d - %d = %d",
		initialEscrowBalance, commissionAmount, payoutAmount, initialEscrowBalance-commissionAmount-payoutAmount)

	finalPlatform, err := walletRepo.GetByID(ctx, platformWallet.ID)
	require.NoError(t, err)
	require.Equal(t, commissionAmount, finalPlatform.Balance)

	finalHost, err := walletRepo.GetByID(ctx, hostWallet.ID)
	require.NoError(t, err)
	require.Equal(t, payoutAmount, finalHost.Balance)
}

// TestExternalTransactionsNotFlaggedAsImbalanced verifies that charge and refund
// transactions (which involve external parties) are NOT flagged as imbalanced
// by the reconciliation logic, even though they only have debit OR credit entries.
func TestExternalTransactionsNotFlaggedAsImbalanced(t *testing.T) {
	ctx := context.Background()
	db := newFinanceTestDB(t)

	walletRepo := repository.NewWalletRepository(db)
	txnRepo := repository.NewTransactionRepository(db)
	ledgerRepo := repository.NewLedgerRepository(db)
	svc := &FinanceServiceImpl{
		db:              db,
		walletRepo:      walletRepo,
		transactionRepo: txnRepo,
		ledgerRepo:      ledgerRepo,
		log:             slog.Default(),
	}

	// Create a charge transaction (external → internal)
	// This only creates a CREDIT entry in the ledger (no debit)
	chargeBookingID := uuid.New()
	chargePaymentID := uuid.New()
	chargeAmount := int64(100000) // 1000 NGN

	chargeTransaction, err := svc.RecordCharge(ctx, chargeBookingID, chargePaymentID, chargeAmount, "NGN")
	require.NoError(t, err)

	// Create a refund transaction (internal → external)
	// This only creates a DEBIT entry in the ledger (no credit)
	refundBookingID := uuid.New()
	refundPaymentID := uuid.New()
	refundAmount := int64(50000) // 500 NGN

	// First record a charge to have balance for refund
	_, err = svc.RecordCharge(ctx, refundBookingID, uuid.New(), refundAmount, "NGN")
	require.NoError(t, err)

	// Now record the refund
	refundTransaction, err := svc.RecordRefund(ctx, refundBookingID, refundPaymentID, refundAmount, "NGN")
	require.NoError(t, err)

	// Run reconciliation validation
	discrepancies, err := svc.ValidateLedgerBalance(ctx)
	require.NoError(t, err)

	// CRITICAL ASSERTION: External transactions should NOT be flagged as imbalanced
	// The charge transaction (only credit) and refund transaction (only debit)
	// should be excluded from validation
	for _, disc := range discrepancies {
		// Make sure neither the charge nor refund transaction is in the discrepancies
		if disc.TransactionID != nil {
			require.NotEqual(t, chargeTransaction.ID, *disc.TransactionID,
				"Charge transaction should not be flagged as imbalanced")
			require.NotEqual(t, refundTransaction.ID, *disc.TransactionID,
				"Refund transaction should not be flagged as imbalanced")
		}
	}

	// Verify wallet reconciliation also passes
	walletDiscrepancies, err := svc.ValidateWalletBalance(ctx)
	require.NoError(t, err)

	// All wallets should balance correctly despite having external transactions
	require.Empty(t, walletDiscrepancies, "Wallets should not have balance discrepancies after external transactions")
}
