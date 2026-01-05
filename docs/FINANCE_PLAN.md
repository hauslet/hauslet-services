# Finance Module Master Plan

## 1. Vision & Objectives
- **Comprehensive ledger** covering all monetary moves (guest charges, refunds, payouts, commissions).
- **Escrow enforcement** so guest funds sit in `WalletEscrow` until release (48h post-checkout & dispute-free).
- **Automated host payouts** via hourly jobs that move cleared balances and trigger transfers.
- **Dispute/chargeback handling**: freeze or reverse ledger entries when disputes occur.
- **Auditable & extensible** to future revenue streams (subscriptions, ID verification) through polymorphic resource references.

## 2. Core Concepts
| Concept          | Description                                                           |
|------------------|-----------------------------------------------------------------------|
| **Wallet**       | Logical balance bucket (Escrow, Host, Fee, Refund).                   |
| **Ledger Entry** | Immutable debit/credit record; every transaction creates a pair.      |
| **Transaction**  | High-level action (charge/refund/payout) that owns ledger entries.    |
| **Disbursement** | External transfer to host bank account with retry & status tracking. |
| **Freeze**       | Lock on a wallet during disputes or manual reviews.                   |

## 3. Domain Model
- **Wallet**: `ID`, `OwnerType`, `OwnerID`, `WalletType`, `Balance`, `Currency`, `Status`, `Metadata`. Methods `Credit`, `Debit`, `Freeze`, `Unfreeze`.
- **LedgerEntry**: `TransactionID`, `Reference`, `DebitWalletID`, `CreditWalletID`, `Amount`, `Currency`, `ResourceType/ID`, `Memo`, timestamps.
- **Transaction**: `Type`, `Status`, `ResourceType/ID`, `Amount`, `LedgerEntries`, `ErrorInfo`.
- **Disbursement**: `WalletID`, `Amount`, `Currency`, `Provider`, `TransferCode`, `Attempts`, `NextRetryAt`, `Status`.
- **Dispute**: `BookingID`, `Reason`, `Evidence`, `Status`, `FrozenWalletIDs`, `Resolution`.

## 4. Repository Layer
- `WalletRepository`: CRUD, `GetOrCreate`, `AdjustBalance`, `ListByOwner`.
- `LedgerRepository`: write/read ledger entries (must run inside SQL tx).
- `TransactionRepository`: store parent transactions, query by resource.
- `DisbursementRepository` and `DisputeRepository` for payouts & disputes.

## 5. Services
### 5.1 WalletService
- `GetOrCreateWallet`
- `FreezeWallet` / `UnfreezeWallet`
- `GetBalance`

### 5.2 LedgerService
- `RecordCharge(bookingID, paymentID, amount)`
- `RecordCommission`
- `RecordRefund`
- `RecordPayout`
- Always enforce double-entry and idempotency (reference hash).

### 5.3 PayoutService
- `QueuePayout(bookingID)` once stay completes & no disputes.
- `ProcessDuePayouts(ctx)` cron: move Escrow → HostAvailable, create disbursement, call payment provider, mark booking `settled`.
- `RetryFailedDisbursements` cron.

### 5.4 Refund/DisputeService
- `InitiateRefund`: debit escrow/host wallet, credit refund pool, notify guest.
- `OpenDispute`: freeze wallets, mark booking/disbursement statuses.
- `ResolveDispute`: apply resolution (refund guest or release host) and unfreeze wallets.

### 5.5 Integration Hooks
- Booking calls:
  - `finance.RecordCharge(bookingID, paymentID, amount)` upon payment success.
  - `finance.ReleaseEscrow(bookingID)` post stay to trigger payouts.
  - `finance.RecordRefund(bookingID, paymentID, amount)` on refund.
- Payments uses same hooks for non-booking resource types via ResourceType/ID.

## 6. Workflow Summary
1. **Booking Paid** → finance debits guest payment → credits booking escrow wallet. Ledger records `charge` transaction.
2. **Stay Completed + 48h** (no dispute) → cron moves escrow → host wallet, queues disbursement, updates booking `settled`.
3. **Payout Execution** → PayoutService calls payment provider; success triggers host notification, failure schedules retry + alert.
4. **Refund** → finance debits escrow/host wallet, credits refund pool, informs payments to send refund, sends guest email.
5. **Dispute** → freeze wallets, mark booking `disputed`, hold payouts until resolution.

## 7. Notifications & Templates
- Package `internal/modules/finance/notification` with templates:
  - `payment_receipt.html`
  - `payout_initiated.html`
  - `payout_success.html`
  - `payout_failed.html`
  - `refund_processed.html`
  - `dispute_alert.html`
- Trigger notifications on charge, payout (initiate/success/fail), refund, dispute open/resolution.

## 8. Cron Jobs
1. `ProcessDuePayouts` (hourly)
2. `RetryFailedDisbursements` (15 min)
3. `ArchiveOldLedgerEntries` (daily maintenance)
4. Optional `ReconcileProviderStatements` (daily).

## 9. Migrations & Seeding
- Tables: `wallets`, `ledger_entries`, `transactions`, `disbursements`, `disputes`.
- Seed platform wallets (commission/refund pool).
- Add indexes on `(owner_type, owner_id, wallet_type)` and `(resource_type, resource_id)`.

## 10. Observability & Security
- Structured logs/metrics for charges, payouts, failed disbursements.
- Alerts on >N failed payouts or stuck disputes.
- All finance APIs internal-authenticated; audit log manual adjustments.
- Encrypt sensitive payout data.

## 11. Integration Steps
1. Implement finance domain/repo/service skeleton + migrations.
2. Wire finance hooks into booking’s payment lifecycle (`HandlePaymentSuccess`, etc.).
3. Update payments webhook flow to call finance before booking updates.
4. Add cron jobs to worker.
5. Build admin APIs for wallet/ledger inspection.
6. Backfill existing confirmed bookings into finance ledger.
7. Roll out behind feature flag, monitor, then enable payouts.
