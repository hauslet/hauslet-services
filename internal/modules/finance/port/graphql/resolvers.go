package graphql

import (
	"context"
	"fmt"
	businessmiddleware "hauslet/internal/modules/business/middleware"
	businessservice "hauslet/internal/modules/business/service"
	"hauslet/internal/modules/finance/domain"
	"hauslet/internal/modules/finance/service"
	"log/slog"

	"github.com/google/uuid"
)

// Resolver handles GraphQL queries for finance module
type Resolver struct {
	financeService service.FinanceService
	payoutService  service.PayoutService
	businessAuth   *businessmiddleware.GraphQLAuthHelper
	log            *slog.Logger
}

// NewResolver creates a new GraphQL resolver for finance
func NewResolver(
	financeService service.FinanceService,
	payoutService service.PayoutService,
	businessService businessservice.BusinessService,
	log *slog.Logger,
) *Resolver {
	return &Resolver{
		financeService: financeService,
		payoutService:  payoutService,
		businessAuth:   businessmiddleware.NewGraphQLAuthHelper(businessService),
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
		r.log.Error("invalid wallet ID", "wallet_id", id, "error", err)
		return nil, fmt.Errorf("invalid wallet ID")
	}

	wallet, err := r.financeService.GetWallet(ctx, walletID)
	if err != nil {
		r.log.Error("failed to get wallet", "wallet_id", id, "error", err)
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
		r.log.Error("invalid user ID", "user_id", userID, "error", err)
		return nil, fmt.Errorf("invalid user ID")
	}

	wallets, err := r.financeService.ListUserWallets(ctx, uid)
	if err != nil {
		r.log.Error("failed to list wallets for user", "user_id", userID, "error", err)
		return nil, err
	}

	return wallets, nil
}

// MyWallets lists wallets for the current user
func (r *Resolver) MyWallets(ctx context.Context) ([]*domain.Wallet, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	wallets, err := r.financeService.ListOwnerWallets(ctx, domain.OwnerTypeUser, userID)
	if err != nil {
		r.log.Error("failed to list wallets for user", "user_id", userID, "error", err)
		return nil, err
	}

	return wallets, nil
}

// BusinessWallets lists wallets for a business (member or admin only)
func (r *Resolver) BusinessWallets(ctx context.Context, businessID string) ([]*domain.Wallet, error) {
	bID, err := uuid.Parse(businessID)
	if err != nil {
		return nil, fmt.Errorf("invalid business ID")
	}

	if err := r.requireBusinessAccess(ctx, bID); err != nil {
		return nil, err
	}

	wallets, err := r.financeService.ListOwnerWallets(ctx, domain.OwnerTypeBusiness, bID)
	if err != nil {
		r.log.Error("failed to list wallets for business", "business_id", bID, "error", err)
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
		r.log.Error("invalid resource ID", "resource_id", resourceID, "error", err)
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
		r.log.Error("invalid wallet ID", "wallet_id", walletID, "error", err)
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
		r.log.Error("failed to get wallet ledger", "error", err)
		return nil, err
	}

	return entries, nil
}

// MyWalletLedger retrieves ledger entries for the current user's wallet
func (r *Resolver) MyWalletLedger(ctx context.Context, walletID string, limit *int, offset *int) ([]*domain.LedgerEntry, error) {
	// To return early if unauthorized, userID would be used later for ownership check
	_, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	wid, err := uuid.Parse(walletID)
	if err != nil {
		return nil, fmt.Errorf("invalid wallet ID")
	}

	wallet, err := r.financeService.GetWallet(ctx, wid)
	if err != nil {
		return nil, err
	}

	if wallet.OwnerType != domain.OwnerTypeUser {
		return nil, ErrOwnershipRequired
	}

	if err := requireOwnershipOrAdmin(ctx, wallet.OwnerID); err != nil {
		return nil, err
	}

	l := 20
	if limit != nil && *limit > 0 {
		l = *limit
	}

	o := 0
	if offset != nil && *offset > 0 {
		o = *offset
	}

	return r.financeService.GetWalletHistory(ctx, wallet.ID, l, o)
}

// BusinessWalletLedger retrieves ledger entries for a business wallet
func (r *Resolver) BusinessWalletLedger(ctx context.Context, walletID string, limit *int, offset *int) ([]*domain.LedgerEntry, error) {
	wid, err := uuid.Parse(walletID)
	if err != nil {
		return nil, fmt.Errorf("invalid wallet ID")
	}

	wallet, err := r.financeService.GetWallet(ctx, wid)
	if err != nil {
		return nil, err
	}

	if wallet.OwnerType != domain.OwnerTypeBusiness {
		return nil, ErrOwnershipRequired
	}

	if err := r.requireBusinessAccess(ctx, wallet.OwnerID); err != nil {
		return nil, err
	}

	l := 20
	if limit != nil && *limit > 0 {
		l = *limit
	}

	o := 0
	if offset != nil && *offset > 0 {
		o = *offset
	}

	return r.financeService.GetWalletHistory(ctx, wallet.ID, l, o)
}

// Disbursement retrieves a disbursement by ID
func (r *Resolver) Disbursement(ctx context.Context, id string) (*domain.Disbursement, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	did, err := uuid.Parse(id)
	if err != nil {
		r.log.Error("invalid disbursement ID", "disbursement_id", id, "error", err)
		return nil, fmt.Errorf("invalid disbursement ID")
	}

	disbursement, err := r.payoutService.GetDisbursement(ctx, did)
	if err != nil {
		r.log.Error("failed to get disbursement", "disbursement_id", id, "error", err)
		return nil, err
	}

	return disbursement, nil
}

// MyDisbursements lists payout disbursements for the current user
func (r *Resolver) MyDisbursements(ctx context.Context, status *domain.DisbursementStatus, limit *int, offset *int) ([]*domain.Disbursement, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	l := 20
	if limit != nil && *limit > 0 {
		l = *limit
	}

	o := 0
	if offset != nil && *offset > 0 {
		o = *offset
	}

	return r.payoutService.ListDisbursementsByOwner(ctx, domain.OwnerTypeUser, userID, status, l, o)
}

// BusinessDisbursements lists payout disbursements for a business
func (r *Resolver) BusinessDisbursements(ctx context.Context, businessID string, status *domain.DisbursementStatus, limit *int, offset *int) ([]*domain.Disbursement, error) {
	bID, err := uuid.Parse(businessID)
	if err != nil {
		return nil, fmt.Errorf("invalid business ID")
	}

	if err := r.requireBusinessAccess(ctx, bID); err != nil {
		return nil, err
	}

	l := 20
	if limit != nil && *limit > 0 {
		l = *limit
	}

	o := 0
	if offset != nil && *offset > 0 {
		o = *offset
	}

	return r.payoutService.ListDisbursementsByOwner(ctx, domain.OwnerTypeBusiness, bID, status, l, o)
}

// MyFinanceTransactions lists finance transactions for the current user
func (r *Resolver) MyFinanceTransactions(ctx context.Context, txType *domain.TransactionType, status *domain.TransactionStatus, limit *int, offset *int) ([]*domain.Transaction, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	l := 20
	if limit != nil && *limit > 0 {
		l = *limit
	}

	o := 0
	if offset != nil && *offset > 0 {
		o = *offset
	}

	return r.financeService.ListTransactionsByOwner(ctx, domain.OwnerTypeUser, userID, txType, status, l, o)
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
		r.log.Error("failed to get wallets for user", "user_id", userID, "error", err)
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

func (r *Resolver) requireBusinessAccess(ctx context.Context, businessID uuid.UUID) error {
	if err := requireAdmin(ctx); err == nil {
		return nil
	} else if err != ErrAdminRequired {
		return err
	}

	if r.businessAuth == nil {
		return ErrUnauthorized
	}

	_, err := r.businessAuth.RequireMembership(ctx, businessID)
	return err
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
