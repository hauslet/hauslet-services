package graphql

import (
	"context"
	"fmt"
	"hauslet/internal/modules/finance/domain"
	"hauslet/internal/modules/finance/service"
	"log/slog"

	"github.com/google/uuid"
)

// Resolver handles GraphQL queries for finance module
type Resolver struct {
	financeService service.FinanceService
	payoutService  service.PayoutService
	log            *slog.Logger
}

// NewResolver creates a new GraphQL resolver for finance
func NewResolver(
	financeService service.FinanceService,
	payoutService service.PayoutService,
	log *slog.Logger,
) *Resolver {
	return &Resolver{
		financeService: financeService,
		payoutService:  payoutService,
		log:            log,
	}
}

// ============================================================================
// Query Resolvers
// ============================================================================

// Wallet retrieves a wallet by ID (admin only)
func (r *Resolver) Wallet(ctx context.Context, id string) (*domain.Wallet, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	walletID, err := uuid.Parse(id)
	if err != nil {
		r.log.Error("invalid wallet ID %s: %v", id, err)
		return nil, fmt.Errorf("invalid wallet ID")
	}

	wallet, err := r.financeService.GetWallet(ctx, walletID)
	if err != nil {
		r.log.Error("failed to get wallet %s: %v", id, err)
		return nil, err
	}

	return wallet, nil
}

// UserWallets lists all wallets for a user (admin only)
func (r *Resolver) UserWallets(ctx context.Context, userID string) ([]*domain.Wallet, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		r.log.Error("invalid user ID %s: %v", userID, err)
		return nil, fmt.Errorf("invalid user ID")
	}

	wallets, err := r.financeService.ListUserWallets(ctx, uid)
	if err != nil {
		r.log.Error("failed to list wallets for user %s: %v", userID, err)
		return nil, err
	}

	return wallets, nil
}

// FinanceTransactionHistory retrieves transaction history for a resource
func (r *Resolver) FinanceTransactionHistory(
	ctx context.Context,
	resourceType string,
	resourceID string,
) ([]*domain.Transaction, error) {
	rid, err := uuid.Parse(resourceID)
	if err != nil {
		r.log.Error("invalid resource ID %s: %v", resourceID, err)
		return nil, fmt.Errorf("invalid resource ID")
	}

	// Admin can view any transactions; users can view their own bookings
	if err := requireAdmin(ctx); err != nil {
		// If not admin, check if this is the user's own resource
		// For bookings, check ownership through booking service (TODO)
		return nil, err
	}

	transactions, err := r.financeService.GetTransactionHistory(ctx, domain.ResourceType(resourceType), rid)
	if err != nil {
		r.log.Error("failed to get transaction history", "error", err)
		return nil, err
	}

	return transactions, nil
}

// WalletLedger retrieves ledger entries for a wallet
func (r *Resolver) WalletLedger(
	ctx context.Context,
	walletID string,
	limit *int,
	offset *int,
) ([]*domain.LedgerEntry, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	wid, err := uuid.Parse(walletID)
	if err != nil {
		r.log.Error("invalid wallet ID %s: %v", walletID, err)
		return nil, fmt.Errorf("invalid wallet ID")
	}

	l := 20
	if limit != nil && *limit > 0 {
		l = *limit
	}

	o := 0
	if offset != nil && *offset > 0 {
		o = *offset
	}

	entries, err := r.financeService.GetWalletHistory(ctx, wid, l, o)
	if err != nil {
		r.log.Error("failed to get wallet ledger: %v", err)
		return nil, err
	}

	return entries, nil
}

// Disbursement retrieves a disbursement by ID
func (r *Resolver) Disbursement(ctx context.Context, id string) (*domain.Disbursement, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	did, err := uuid.Parse(id)
	if err != nil {
		r.log.Error("invalid disbursement ID %s: %v", id, err)
		return nil, fmt.Errorf("invalid disbursement ID")
	}

	disbursement, err := r.payoutService.GetDisbursement(ctx, did)
	if err != nil {
		r.log.Error("failed to get disbursement %s: %v", id, err)
		return nil, err
	}

	return disbursement, nil
}

// MyEarnings returns earnings summary for the current host
func (r *Resolver) MyEarnings(ctx context.Context) (*EarningsSummary, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Get host's available wallet
	wallets, err := r.financeService.ListUserWallets(ctx, userID)
	if err != nil {
		r.log.Error("failed to get wallets for user %s: %v", userID, err)
		return nil, err
	}

	var availableBalance int64
	var currency string = "NGN"

	for _, wallet := range wallets {
		if wallet.WalletType == domain.WalletTypeHostAvailable {
			availableBalance = wallet.Balance
			currency = wallet.Currency
			break
		}
	}

	// Calculate total earned (would need to query completed payouts)
	// For now, return available balance as totalEarned
	totalEarned := availableBalance

	// Calculate pending payouts (would need to query pending disbursements)
	// For now, return 0
	pendingPayouts := int64(0)

	return &EarningsSummary{
		TotalEarned:      totalEarned,
		AvailableBalance: availableBalance,
		PendingPayouts:   pendingPayouts,
		Currency:         currency,
	}, nil
}

// ============================================================================
// GraphQL Schema Types
// ============================================================================

// EarningsSummary represents a host's earnings summary
type EarningsSummary struct {
	TotalEarned      int64
	AvailableBalance int64
	PendingPayouts   int64
	Currency         string
}
