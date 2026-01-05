# Hauslet Finance System - Complete Guide

## Table of Contents
1. [Overview](#overview)
2. [Core Concepts](#core-concepts)
3. [Double-Entry Bookkeeping](#double-entry-bookkeeping)
4. [Wallet System](#wallet-system)
5. [Transaction Lifecycle](#transaction-lifecycle)
6. [Escrow & Payouts](#escrow--payouts)
7. [Dispute Management](#dispute-management)
8. [Reconciliation](#reconciliation)
9. [GraphQL API Reference](#graphql-api-reference)
10. [Usage Examples](#usage-examples)
11. [Best Practices](#best-practices)

---

## Overview

The **Hauslet Finance System** is a comprehensive double-entry ledger system that manages all monetary flows on the platform. It provides secure escrow services, automated payouts, dispute resolution, and complete financial auditing capabilities for Nigerian real estate transactions.

### Key Features

- **Double-Entry Bookkeeping**: Every transaction creates balanced debit/credit entries (financial accuracy)
- **Escrow Management**: Guest payments held securely until booking completion + 48 hours
- **Automated Payouts**: Cron jobs automatically release funds to hosts with retry logic
- **Dispute Resolution**: Freeze escrow, investigate, resolve with refund or release
- **Multi-Currency Support**: NGN primary, extensible to USD/other currencies
- **Complete Audit Trail**: Every financial action immutably recorded
- **Provider Integration**: Paystack/Flutterwave for bank transfers
- **Reconciliation**: Automated checks ensure ledger integrity

### System Architecture

```
┌─────────────┐
│   Booking   │ ─────┐
└─────────────┘      │
                     ├──→ ┌──────────────┐      ┌─────────────┐
┌─────────────┐      │    │   Finance    │◄────►│  Wallets    │
│  Payments   │ ─────┤    │   Service    │      └─────────────┘
└─────────────┘      │    └──────────────┘              │
                     │           │                      │
┌─────────────┐      │           │                      ▼
│  Disputes   │ ─────┘           │              ┌──────────────┐
└─────────────┘                  ▼              │    Ledger    │
                          ┌─────────────┐       │   Entries    │
                          │ Disbursement│       └──────────────┘
                          └─────────────┘
                                 │
                                 ▼
                          ┌─────────────┐
                          │  Paystack   │
                          │ Flutterwave │
                          └─────────────┘
```

---

## Core Concepts

### 1. Wallet

A **Wallet** is a logical balance bucket that holds funds. Every financial transaction involves moving money between wallets.

**Wallet Types**:

| Type | Owner | Purpose | Balance Flow |
|------|-------|---------|--------------|
| **escrow** | Per Booking | Holds guest payment until release | Guest → Escrow → Host/Refund |
| **host_available** | Per Host | Available for withdrawal | Escrow → Host Wallet → Bank |
| **platform_fee** | Platform | Commission revenue | Escrow → Platform (5-15%) |
| **refund_pool** | Platform | Aggregates refunds | Escrow → Refund Pool → Guest Bank |

**Wallet States**:
- **active**: Normal operations (credit/debit allowed)
- **frozen**: Locked during disputes (only credits allowed)
- **closed**: Permanently closed (balance must be zero)

### 2. Ledger Entry

A **Ledger Entry** is an immutable record of money movement. Every transaction creates **paired entries** (double-entry bookkeeping).

**Entry Structure**:
```
Debit:  Take money FROM a wallet
Credit: Put money INTO a wallet
```

**Golden Rule**: For every transaction, `Total Debits = Total Credits`

### 3. Transaction

A **Transaction** is a high-level financial operation that owns one or more ledger entry pairs.

**Transaction Types**:
- **charge**: Guest payment received (Payment → Escrow)
- **commission**: Platform fee deduction (Escrow → Platform Fee)
- **payout**: Host payment (Escrow → Host Available)
- **refund**: Cancel refund (Escrow → Refund Pool)
- **reversal**: Dispute reversal (reverse previous entries)

**Transaction Status**:
- **pending**: Awaiting processing
- **completed**: Successfully executed
- **failed**: Failed to process
- **reversed**: Reversed due to dispute

### 4. Disbursement

A **Disbursement** is an actual bank transfer to a host's account via payment provider (Paystack/Flutterwave).

**Retry Strategy** (Exponential Backoff):
```
Attempt 1: Immediate
Attempt 2: +1 minute
Attempt 3: +5 minutes
Attempt 4: +15 minutes
Attempt 5: +1 hour
Attempt 6: +6 hours
Max Retries: 6 attempts
```

### 5. Dispute

A **Dispute** is a formal challenge to a booking transaction (guest or host initiated). Disputes freeze escrow and pause payouts until resolution.

---

## Double-Entry Bookkeeping

The finance system implements strict double-entry accounting. Every financial transaction creates **balanced pairs** of debits and credits.

### Fundamental Equation

```
Assets = Liabilities + Equity

For Hauslet:
(Escrow + Host Wallets + Platform Fee) = Guest Liabilities + Platform Equity
```

### Accounting Entries

#### Example 1: Guest Pays ₦100,000 for Booking

```
Transaction: CHARGE
Amount: ₦100,000

Ledger Entries:
┌──────────────────────────────────────────────┐
│ Debit:  External (Guest)      ₦100,000      │  ← Money leaves guest
│ Credit: Booking Escrow        ₦100,000      │  ← Money enters escrow
└──────────────────────────────────────────────┘
Total Debit = Total Credit = ₦100,000 ✓
```

#### Example 2: Platform Takes 10% Commission

```
Transaction: COMMISSION
Amount: ₦10,000

Ledger Entries:
┌──────────────────────────────────────────────┐
│ Debit:  Booking Escrow        ₦10,000       │  ← Money leaves escrow
│ Credit: Platform Fee Wallet   ₦10,000       │  ← Money enters platform
└──────────────────────────────────────────────┘
Escrow Balance: ₦100,000 - ₦10,000 = ₦90,000
```

#### Example 3: Host Receives Payout

```
Transaction: PAYOUT
Amount: ₦90,000

Ledger Entries:
┌──────────────────────────────────────────────┐
│ Debit:  Booking Escrow        ₦90,000       │  ← Money leaves escrow
│ Credit: Host Available Wallet ₦90,000       │  ← Money enters host wallet
└──────────────────────────────────────────────┘
Escrow Balance: ₦90,000 - ₦90,000 = ₦0 ✓
```

#### Example 4: Full Refund (Before Payout)

```
Transaction: REFUND
Amount: ₦100,000

Ledger Entries:
┌──────────────────────────────────────────────┐
│ Debit:  Booking Escrow        ₦100,000      │  ← Money leaves escrow
│ Credit: Refund Pool           ₦100,000      │  ← Money enters refund pool
└──────────────────────────────────────────────┘

Then Paystack processes refund to guest bank account
```

### Idempotency

Every ledger entry has a unique **reference hash** to prevent duplicates:

```go
reference = SHA256(
    transactionType + 
    resourceID + 
    amount + 
    timestamp
)
```

If the same charge is recorded twice (webhook retry), the second attempt is rejected due to duplicate reference.

---

## Wallet System

### Wallet Ownership Model

```
┌──────────────────────────────────────────────────┐
│              Platform Wallets (2)                │
│  - platform_fee (Commission Revenue)             │
│  - refund_pool (Aggregated Refunds)              │
└──────────────────────────────────────────────────┘
                      ▲
                      │
        ┌─────────────┴─────────────┐
        │                           │
┌───────┴───────┐          ┌────────┴────────┐
│ Host Wallets  │          │ Escrow Wallets  │
│ (Per User)    │          │ (Per Booking)   │
│               │          │                 │
│ - Available   │          │ - Unique ID     │
│   Balance     │          │ - Lifecycle:    │
│ - Can Withdraw│          │   Created →     │
│               │          │   Funded →      │
│               │          │   Released →    │
│               │          │   Closed        │
└───────────────┘          └─────────────────┘
```

### Wallet Lifecycle

#### 1. Escrow Wallet (Per Booking)

```
State: NEW (Created when booking confirmed)
Balance: ₦0

↓ Guest pays ₦100,000

State: FUNDED
Balance: ₦100,000

↓ Checkout + 48 hours (no dispute)

State: RELEASING
Balance: ₦100,000 → ₦90,000 (commission taken)
          ₦90,000 → ₦0 (payout to host)

State: CLOSED
Balance: ₦0
```

#### 2. Host Available Wallet (Per User)

```
Balance: ₦0

↓ Booking 1 payout (₦90,000)

Balance: ₦90,000

↓ Booking 2 payout (₦150,000)

Balance: ₦240,000

↓ Host requests withdrawal (₦200,000)

Balance: ₦40,000
Status: Disbursement pending → processing → completed
```

### Wallet Operations

#### Credit (Add Funds)

```go
wallet.Credit(amount)

Rules:
- Amount must be > 0
- Wallet status must not be CLOSED
- Frozen wallets CAN receive credits (but not debits)
```

#### Debit (Remove Funds)

```go
wallet.Debit(amount)

Rules:
- Amount must be > 0
- Wallet status must be ACTIVE (not frozen/closed)
- Balance must be >= amount (no overdrafts)
```

#### Freeze (Lock Wallet)

```go
wallet.Freeze()

Effect:
- Status changes to FROZEN
- Debits blocked (payouts stopped)
- Credits allowed (refunds can still enter)

Use Cases:
- Dispute filed
- Fraud investigation
- Manual admin hold
```

---

## Transaction Lifecycle

### Flow: Guest Payment → Host Payout

```
Timeline: Day 0 (Booking) → Day 7 (Checkout) → Day 9 (Payout)

┌─────────────────────────────────────────────────────────┐
│ Day 0: Booking Confirmed + Payment Success              │
├─────────────────────────────────────────────────────────┤
│ 1. Payments module receives webhook from Paystack       │
│ 2. Finance.RecordCharge(bookingID, paymentID, ₦100k)    │
│    ├─ Create Transaction (type=charge, status=pending)  │
│    ├─ Create Escrow Wallet for booking                  │
│    ├─ Create Ledger Entries:                            │
│    │  • Debit: External (₦100k)                         │
│    │  • Credit: Escrow (₦100k)                          │
│    ├─ Update Escrow Balance: ₦0 → ₦100k                 │
│    └─ Mark Transaction: completed                       │
│ 3. Send receipt email to guest                          │
└─────────────────────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────────────────────┐
│ Day 7: Guest Checks Out                                 │
├─────────────────────────────────────────────────────────┤
│ 1. Booking status: active → checked_out                 │
│ 2. Start 48-hour dispute window                         │
│ 3. No payout yet (waiting period)                       │
└─────────────────────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────────────────────┐
│ Day 9: Payout Eligible (48h passed, no disputes)        │
├─────────────────────────────────────────────────────────┤
│ Cron Job: ProcessDuePayouts() runs hourly               │
│                                                          │
│ 1. Query bookings: checked_out + 48h + no disputes     │
│ 2. For each eligible booking:                           │
│    a. Calculate commission (10% = ₦10k)                 │
│    b. Finance.RecordCommission(bookingID, ₦10k)         │
│       ├─ Debit: Escrow (₦10k)                           │
│       └─ Credit: Platform Fee (₦10k)                    │
│    c. Finance.RecordPayout(bookingID, hostID, ₦90k)     │
│       ├─ Debit: Escrow (₦90k)                           │
│       └─ Credit: Host Available (₦90k)                  │
│    d. Create Disbursement record                        │
│    e. Call Paystack Transfer API                        │
│       ├─ Success: Mark disbursement=completed           │
│       └─ Failure: Schedule retry (+1 min)              │
│ 3. Update Booking: status=settled                       │
│ 4. Send payout email to host                            │
└─────────────────────────────────────────────────────────┘
```

### Transaction States

```
PENDING
   │
   ├──→ Processing ledger entries
   │
   ├──→ Success: COMPLETED ✓
   │
   ├──→ Error: FAILED ✗
   │      │
   │      └──→ Admin review or auto-retry
   │
   └──→ Dispute: REVERSED 🔄
          │
          └──→ Creates reversal transaction
```

---

## Escrow & Payouts

### Escrow Protection Model

**Purpose**: Hold guest funds securely until host fulfills booking obligations.

**Protection Window**: Checkout + 48 hours (no dispute)

**Release Conditions**:
1. ✅ Booking status = `checked_out`
2. ✅ Checkout time + 48 hours elapsed
3. ✅ No active dispute filed
4. ✅ Escrow wallet not frozen

### Payout Automation (Cron Jobs)

#### Job 1: ProcessDuePayouts (Hourly)

```go
func ProcessDuePayouts(ctx context.Context) error {
    // 1. Find eligible bookings
    eligibleBookings := FindBookings(
        status: "checked_out",
        checkoutTime: < (now - 48h),
        dispute: nil,
        settled: false
    )
    
    // 2. Process each booking
    for _, booking := range eligibleBookings {
        // Calculate splits
        totalAmount := booking.TotalPaid
        commission := totalAmount * commissionRate // 10%
        hostPayout := totalAmount - commission
        
        // Record financial transactions
        finance.RecordCommission(booking.ID, commission)
        finance.RecordPayout(booking.ID, booking.HostID, hostPayout)
        
        // Create disbursement
        disbursement := CreateDisbursement(
            walletID: host.AvailableWallet.ID,
            amount: hostPayout,
            provider: "paystack"
        )
        
        // Initiate bank transfer
        result := paystack.InitiateTransfer(
            amount: hostPayout,
            recipient: host.BankAccount,
            reference: disbursement.ID
        )
        
        if result.Success {
            disbursement.MarkCompleted(result.TransferCode)
            booking.MarkSettled()
            SendPayoutSuccessEmail(host, hostPayout)
        } else {
            disbursement.MarkFailed(result.Error)
            // Will retry in next job cycle
        }
    }
}
```

#### Job 2: RetryFailedDisbursements (Every 15 min)

```go
func RetryFailedDisbursements(ctx context.Context) error {
    // Find failed disbursements eligible for retry
    retryable := FindDisbursements(
        status: "failed",
        nextRetryAt: < now,
        attempts: < 6
    )
    
    for _, disb := range retryable {
        // Attempt transfer again
        result := paystack.InitiateTransfer(...)
        
        if result.Success {
            disb.MarkCompleted(result.TransferCode)
        } else {
            disb.MarkFailed(result.Error) // Increments attempt, schedules next retry
            
            if disb.Attempts >= 6 {
                // Max retries exceeded
                AlertAdmins("Disbursement failed after 6 attempts", disb.ID)
                SendPayoutFailedEmail(host, disb.Amount)
            }
        }
    }
}
```

### Commission Calculation

**Platform Commission Tiers**:

| Booking Value | Commission Rate | Example |
|---------------|-----------------|---------|
| ₦0 - ₦50,000 | 5% | ₦50k × 5% = ₦2,500 |
| ₦50,001 - ₦200,000 | 10% | ₦100k × 10% = ₦10,000 |
| ₦200,001+ | 15% | ₦500k × 15% = ₦75,000 |

**Host Receives**: `Total Payment - Commission`

---

## Dispute Management

### Dispute Lifecycle

```
1. OPEN
   ↓ Guest/Host files dispute
   ├─ Escrow wallet FROZEN (no payouts)
   ├─ Booking marked "disputed"
   └─ Email notifications sent

2. INVESTIGATING
   ↓ Admin reviews evidence
   ├─ Evidence uploaded (photos, messages)
   ├─ Admin notes added
   └─ Investigation period (3-14 days)

3. RESOLUTION
   ├─ RESOLVED_REFUND (Guest wins)
   │  ├─ Escrow → Refund Pool
   │  ├─ Paystack refund to guest
   │  └─ Unfreeze wallet
   │
   └─ RESOLVED_RELEASE (Host wins)
      ├─ Unfreeze escrow
      ├─ Process normal payout
      └─ Close dispute

4. CANCELLED
   └─ Disputing party withdraws claim
```

### Filing a Dispute

**Eligible Parties**:
- **Guest**: Property issues, safety, cleanliness, fraud
- **Host**: Guest damage, unauthorized occupancy, payment dispute

**Timing Window**: Checkout → Checkout + 14 days

**Dispute Reasons**:

| Reason | Description | Typical Resolution |
|--------|-------------|-------------------|
| property_mismatch | Listing doesn't match reality | Partial/full refund |
| uninhabitable | Property unsafe/unlivable | Full refund |
| safety_issue | Security/safety concerns | Full refund |
| cleanliness | Poor hygiene standards | Partial refund |
| amenity_missing | Advertised feature absent | Partial refund |
| no_show | Host/guest didn't show up | Context-dependent |
| unauthorized_charges | Unexpected fees | Refund extra charges |
| other | Other issues | Case-by-case |

### Dispute Resolution Process

#### Scenario: Guest Files Dispute

```graphql
# 1. Guest files dispute
mutation {
  fileDispute(input: {
    bookingId: "booking-uuid"
    reason: CLEANLINESS
    description: "Property had cockroaches, moldy bathroom, dirty bedding"
    amount: 50000  # Requesting ₦50k refund (partial)
    currency: "NGN"
  }) {
    id
    status  # OPEN
    walletId  # Escrow wallet ID
  }
}

# System automatically:
# - Freezes escrow wallet
# - Pauses payout processing
# - Sends notifications to host + admin
# - Creates dispute event record
```

#### Scenario: Admin Investigates

```graphql
# 2. Admin marks as investigating
mutation {
  investigateDispute(disputeId: "dispute-uuid") {
    status  # INVESTIGATING
  }
}

# 3. Guest uploads evidence
mutation {
  addDisputeEvidence(input: {
    disputeId: "dispute-uuid"
    type: "photo"
    url: "https://s3.../cockroach-photo.jpg"
    description: "Photo of cockroaches in kitchen"
  }) {
    id
    evidence {
      type
      url
      uploadedAt
    }
  }
}

# 4. Host uploads counter-evidence
mutation {
  addDisputeEvidence(input: {
    disputeId: "dispute-uuid"
    type: "photo"
    url: "https://s3.../clean-inspection.jpg"
    description: "Professional cleaning report from 2 days before check-in"
  }) {
    id
  }
}
```

#### Scenario: Admin Resolves (Guest Favor)

```graphql
# 5. Admin resolves with partial refund
mutation {
  resolveDispute(input: {
    disputeId: "dispute-uuid"
    outcome: RESOLVED_REFUND
    refundAmount: 30000  # ₦30k partial refund
    reason: "Evidence supports cleanliness issues, but property was usable"
    notes: "Refunding 30% of booking cost. Host warned to improve cleaning standards."
  }) {
    id
    status  # RESOLVED_REFUND
    resolution {
      refundAmount
      reason
      resolvedAt
    }
  }
}

# System automatically:
# 1. Records refund transaction:
#    - Debit: Escrow (₦30k)
#    - Credit: Refund Pool (₦30k)
# 2. Initiates Paystack refund to guest
# 3. Unfreezes escrow
# 4. Processes remaining balance to host (₦100k - ₦30k - ₦10k commission = ₦60k)
# 5. Sends resolution emails to both parties
```

---

## Reconciliation

### Purpose

Ensure financial integrity by detecting:
- ❌ Ledger imbalances (debit ≠ credit)
- ❌ Wallet mismatches (balance ≠ sum of entries)
- ❌ Orphaned transactions (no ledger entries)
- ❌ Provider mismatches (Paystack balance ≠ platform fee wallet)

### Reconciliation Checks

#### 1. Ledger Balance Validation

```sql
-- For each transaction, verify debit = credit
SELECT 
  transaction_id,
  SUM(CASE WHEN debit_wallet_id IS NOT NULL THEN amount ELSE 0 END) as total_debits,
  SUM(CASE WHEN credit_wallet_id IS NOT NULL THEN amount ELSE 0 END) as total_credits
FROM ledger_entries
GROUP BY transaction_id
HAVING total_debits != total_credits;

-- Result: Empty = No issues ✓
--         Rows = Imbalance detected ✗
```

#### 2. Wallet Balance Validation

```sql
-- For each wallet, verify balance = sum of entries
SELECT 
  w.id,
  w.balance as wallet_balance,
  (
    COALESCE(SUM(CASE WHEN le.credit_wallet_id = w.id THEN le.amount ELSE 0 END), 0) -
    COALESCE(SUM(CASE WHEN le.debit_wallet_id = w.id THEN le.amount ELSE 0 END), 0)
  ) as calculated_balance
FROM wallets w
LEFT JOIN ledger_entries le ON (le.credit_wallet_id = w.id OR le.debit_wallet_id = w.id)
GROUP BY w.id, w.balance
HAVING wallet_balance != calculated_balance;
```

#### 3. Orphaned Transaction Check

```sql
-- Find transactions without ledger entries
SELECT t.id, t.type, t.amount, t.created_at
FROM transactions t
LEFT JOIN ledger_entries le ON le.transaction_id = t.id
WHERE le.id IS NULL AND t.status = 'completed';
```

### Running Reconciliation

```graphql
# Manual reconciliation (admin only)
mutation {
  runReconciliation {
    id
    status
    totalWalletsChecked
    totalTransactionsChecked
    discrepanciesFound
    summary
  }
}

# Query latest report
query {
  latestReconciliation {
    id
    status
    completedAt
    discrepancies {
      type
      severity
      description
      walletId
      transactionId
    }
  }
}
```

**Automated Schedule**: Daily at 2:00 AM

**Discrepancy Severity**:
- **CRITICAL**: Ledger imbalance, missing funds
- **HIGH**: Wallet mismatch, orphaned transaction
- **MEDIUM**: Provider mismatch
- **LOW**: Stale pending transactions

---

## GraphQL API Reference

### Queries

#### 1. Get Wallet

```graphql
query GetWallet($id: UUID!) {
  wallet(id: $id) {
    id
    ownerType
    ownerId
    walletType
    balance
    currency
    status
    createdAt
    updatedAt
  }
}
```

#### 2. Get User Wallets

```graphql
query GetUserWallets($userId: UUID!) {
  userWallets(userId: $userId) {
    id
    walletType
    balance
    currency
    status
  }
}
```

#### 3. Get Transaction History

```graphql
query GetTransactionHistory(
  $resourceType: String!
  $resourceId: UUID!
) {
  financeTransactionHistory(
    resourceType: $resourceType
    resourceId: $resourceId
  ) {
    id
    type
    status
    amount
    currency
    paymentId
    ledgerEntries {
      id
      reference
      debitWalletId
      creditWalletId
      amount
      memo
      createdAt
    }
    createdAt
  }
}
```

**Example**:
```json
{
  "resourceType": "booking",
  "resourceId": "550e8400-e29b-41d4-a716-446655440000"
}
```

#### 4. Get Wallet Ledger

```graphql
query GetWalletLedger(
  $walletId: UUID!
  $limit: Int
  $offset: Int
) {
  walletLedger(
    walletId: $walletId
    limit: $limit
    offset: $offset
  ) {
    id
    transactionId
    reference
    debitWalletId
    creditWalletId
    amount
    currency
    resourceType
    resourceId
    memo
    createdAt
  }
}
```

#### 5. Get My Earnings (Host)

```graphql
query MyEarnings {
  myEarnings {
    totalEarned
    availableBalance
    pendingPayouts
    currency
  }
}
```

**Response**:
```json
{
  "data": {
    "myEarnings": {
      "totalEarned": 5400000,  // ₦5.4M total lifetime
      "availableBalance": 850000,  // ₦850k ready to withdraw
      "pendingPayouts": 300000,  // ₦300k in escrow (waiting 48h)
      "currency": "NGN"
    }
  }
}
```

### Mutations (Disputes)

#### 1. File Dispute

```graphql
mutation FileDispute($input: FileDisputeInput!) {
  fileDispute(input: $input) {
    id
    bookingId
    walletId
    filedBy
    reason
    status
    amount
    currency
    description
    createdAt
  }
}
```

**Input**:
```json
{
  "input": {
    "bookingId": "booking-uuid",
    "reason": "SAFETY_ISSUE",
    "description": "No smoke detectors, broken lock on front door, exposed wiring",
    "amount": 100000,
    "currency": "NGN"
  }
}
```

#### 2. Add Dispute Evidence

```graphql
mutation AddEvidence($input: AddDisputeEvidenceInput!) {
  addDisputeEvidence(input: $input) {
    id
    evidence {
      type
      url
      description
      uploadedBy
      uploadedAt
    }
  }
}
```

#### 3. Resolve Dispute (Admin)

```graphql
mutation ResolveDispute($input: ResolveDisputeInput!) {
  resolveDispute(input: $input) {
    id
    status
    resolution {
      outcome
      refundAmount
      reason
      notes
      resolvedAt
    }
  }
}
```

#### 4. Cancel Dispute

```graphql
mutation CancelDispute($disputeId: UUID!) {
  cancelDispute(disputeId: $disputeId) {
    id
    status
    resolvedAt
  }
}
```

---

## Usage Examples

### Scenario 1: Complete Booking Payment Flow

```graphql
# Step 1: Guest pays ₦150,000 via Paystack
# (Handled automatically by payment webhook)

# Result: Finance system records charge
{
  "transaction": {
    "type": "charge",
    "amount": 150000,
    "currency": "NGN",
    "status": "completed"
  },
  "escrowWallet": {
    "balance": 150000
  },
  "ledgerEntries": [
    {
      "debitWalletId": null,  # External
      "creditWalletId": "escrow-uuid",
      "amount": 150000,
      "memo": "Guest payment for booking"
    }
  ]
}

# Step 2: Guest checks out (Day 7)
# (No finance action yet - waiting 48h)

# Step 3: Day 9 - Payout eligible
# Cron job processes:

# 3a. Deduct commission (10% = ₦15,000)
{
  "transaction": {
    "type": "commission",
    "amount": 15000
  },
  "escrowWallet": {
    "balance": 135000  # 150k - 15k
  },
  "platformFeeWallet": {
    "balance": 15000  # Increased
  }
}

# 3b. Payout to host (₦135,000)
{
  "transaction": {
    "type": "payout",
    "amount": 135000
  },
  "escrowWallet": {
    "balance": 0  # Empty
  },
  "hostAvailableWallet": {
    "balance": 135000  # Host can withdraw
  }
}

# 3c. Initiate bank transfer
{
  "disbursement": {
    "amount": 135000,
    "provider": "paystack",
    "status": "completed",
    "transferCode": "TRF_abc123xyz"
  }
}

# Step 4: Query host earnings
query {
  myEarnings {
    totalEarned: 135000
    availableBalance: 135000
    pendingPayouts: 0
  }
}
```

### Scenario 2: Cancellation with Full Refund

```graphql
# Booking: ₦200,000 paid, guest cancels before check-in

# Step 1: Process refund
# (Called by booking cancellation handler)

# Result: Finance records refund
{
  "transaction": {
    "type": "refund",
    "amount": 200000,
    "status": "completed"
  },
  "escrowWallet": {
    "balance": 0  # Emptied
  },
  "refundPool": {
    "balance": 200000  # Aggregated
  }
}

# Step 2: Paystack refunds to guest bank
# (Webhook confirms refund processed)

# Step 3: View transaction history
query {
  financeTransactionHistory(
    resourceType: "booking"
    resourceId: "booking-uuid"
  ) {
    # Results:
    # 1. charge (₦200k, completed)
    # 2. refund (₦200k, completed)
  }
}
```

### Scenario 3: Dispute Resolution (Partial Refund)

```graphql
# Booking: ₦300,000, guest complains about cleanliness

# Day 1: Guest files dispute
mutation {
  fileDispute(input: {
    bookingId: "booking-uuid"
    reason: CLEANLINESS
    description: "Bathroom had mold, kitchen dirty"
    amount: 100000  # Requesting ₦100k partial refund
    currency: "NGN"
  }) {
    id
    status  # OPEN
  }
}

# Effect: Escrow frozen (₦300k locked)

# Day 3: Admin investigates
mutation {
  investigateDispute(disputeId: "dispute-uuid") {
    status  # INVESTIGATING
  }
}

# Day 5: Guest uploads photos
mutation {
  addDisputeEvidence(input: {
    disputeId: "dispute-uuid"
    type: "photo"
    url: "https://s3.../mold.jpg"
    description: "Mold in bathroom"
  }) {
    id
  }
}

# Day 7: Admin resolves
mutation {
  resolveDispute(input: {
    disputeId: "dispute-uuid"
    outcome: RESOLVED_REFUND
    refundAmount: 60000  # ₦60k partial refund
    reason: "Minor cleanliness issues confirmed"
    notes: "Host required to deep clean before next booking"
  }) {
    id
    status  # RESOLVED_REFUND
  }
}

# Automatic actions:
# 1. Refund ₦60k to guest
# 2. Deduct commission (10% × ₦240k = ₦24k)
# 3. Payout ₦216k to host (₦300k - ₦60k - ₦24k)
# 4. Unfreeze escrow
# 5. Close wallet (balance: ₦0)

# Final balances:
# - Guest received: ₦60k refund
# - Host received: ₦216k payout
# - Platform fee: ₦24k commission
# Total: ₦60k + ₦216k + ₦24k = ₦300k ✓
```

### Scenario 4: Failed Disbursement with Retry

```graphql
# Payout: ₦500,000 to host, bank transfer fails

# Attempt 1: Immediate (Failed - Bank API down)
{
  "disbursement": {
    "status": "failed",
    "attempts": 1,
    "nextRetryAt": "2026-01-03T10:31:00Z",  # +1 minute
    "failureReason": "Provider timeout"
  }
}

# Attempt 2: +1 minute (Failed - Invalid account)
{
  "disbursement": {
    "status": "failed",
    "attempts": 2,
    "nextRetryAt": "2026-01-03T10:36:00Z",  # +5 minutes
    "failureReason": "Invalid account number"
  }
}

# Admin notified, updates host bank details

# Attempt 3: +5 minutes (Success)
{
  "disbursement": {
    "status": "completed",
    "attempts": 3,
    "transferCode": "TRF_xyz789",
    "completedAt": "2026-01-03T10:36:00Z"
  }
}

# Host receives email: "Payout successful - ₦500,000 sent to account ***1234"
```

---

## Best Practices

### For Platform Administrators

#### 1. **Monitor Escrow Balances Daily**

```sql
-- Total escrowed funds (guest money held)
SELECT 
  SUM(balance) as total_escrow,
  COUNT(*) as num_escrow_wallets
FROM wallets
WHERE wallet_type = 'escrow' AND status = 'active';

-- Should match: Sum of all active booking payments
```

#### 2. **Run Reconciliation Weekly**

- Automated: Every Sunday at 2:00 AM
- Manual: After any data migration or bulk updates
- Alert threshold: Any CRITICAL or HIGH severity discrepancies

#### 3. **Review Failed Disbursements**

```graphql
query {
  disbursements(status: FAILED) {
    id
    amount
    attempts
    failureReason
    nextRetryAt
  }
}
```

**Action Items**:
- < 3 attempts: Monitor (will auto-retry)
- 3-5 attempts: Investigate provider issues
- 6 attempts: Manual intervention required

#### 4. **Dispute SLA Tracking**

| Status | Target Resolution Time |
|--------|----------------------|
| OPEN | Acknowledge within 24h |
| INVESTIGATING | Resolve within 7 days |
| RESOLVED | Process refund/payout within 24h |

### For Hosts

#### 1. **Understand Payout Timeline**

```
Check-in       → Checkout       → Payout Eligible → Payout Processed
Day 0            Day 7            Day 9 (48h later)  Day 9-10
                 
                 └──────────────┬──────────────┘
                         48-hour dispute window
```

**Best Practice**: Communicate checkout checklist to guests to minimize disputes.

#### 2. **Optimize Bank Details**

- Verify account number accuracy (prevents failed disbursements)
- Use business account for tax purposes
- Enable SMS alerts from bank to confirm receipt

#### 3. **Track Earnings**

```graphql
query {
  myEarnings {
    totalEarned      # Lifetime earnings
    availableBalance # Ready to withdraw
    pendingPayouts   # In escrow (waiting 48h)
  }
}
```

**Formula**:
```
Expected Monthly = (Bookings × Average Price × (1 - Commission Rate))

Example:
10 bookings × ₦100k × 0.90 = ₦900k/month
```

#### 4. **Dispute Prevention**

- Upload accurate photos (avoid "property_mismatch" disputes)
- Respond to guest messages quickly
- Provide welcome guide with property instructions
- Schedule professional cleaning between guests

### For Developers

#### 1. **Always Use Transactions**

```go
func (s *FinanceService) RecordCharge(ctx context.Context, ...) error {
    // WRONG: Multiple operations without transaction
    wallet.Credit(amount)
    s.walletRepo.Update(wallet)
    s.ledgerRepo.CreateEntry(entry)
    
    // CORRECT: Atomic transaction
    return s.db.Transaction(func(tx *gorm.DB) error {
        wallet.Credit(amount)
        if err := s.walletRepo.UpdateWithTx(tx, wallet); err != nil {
            return err
        }
        return s.ledgerRepo.CreateEntryWithTx(tx, entry)
    })
}
```

#### 2. **Validate Double-Entry Balance**

```go
func validateEntries(entries []LedgerEntry) error {
    var totalDebit, totalCredit int64
    
    for _, entry := range entries {
        if entry.DebitWalletID != nil {
            totalDebit += entry.Amount
        }
        if entry.CreditWalletID != nil {
            totalCredit += entry.Amount
        }
    }
    
    if totalDebit != totalCredit {
        return ErrLedgerImbalance
    }
    
    return nil
}
```

#### 3. **Implement Idempotency**

```go
func generateReference(txType, resourceID string, amount int64) string {
    data := fmt.Sprintf("%s:%s:%d:%d", txType, resourceID, amount, time.Now().Unix())
    hash := sha256.Sum256([]byte(data))
    return hex.EncodeToString(hash[:])
}

// Check before creating entry
existingEntry, _ := ledgerRepo.GetByReference(reference)
if existingEntry != nil {
    return nil, ErrDuplicateTransaction
}
```

#### 4. **Log All Financial Operations**

```go
log.Info("finance.charge",
    "booking_id", bookingID,
    "payment_id", paymentID,
    "amount", amount,
    "currency", currency,
    "escrow_wallet_id", escrowWallet.ID,
    "transaction_id", transaction.ID,
)
```

---

## Security Considerations

### 1. **Authorization**

All finance queries require strict authorization:

```go
// Only allow access if:
// 1. User owns the wallet (owner_id = user_id)
// 2. User is admin
// 3. User is authorized for related resource (booking host/guest)

func (s *FinanceService) GetWallet(ctx context.Context, walletID, requesterID uuid.UUID) (*Wallet, error) {
    wallet := s.repo.GetByID(walletID)
    
    if wallet.OwnerID != requesterID && !IsAdmin(requesterID) {
        return nil, ErrUnauthorized
    }
    
    return wallet, nil
}
```

### 2. **Immutability**

Ledger entries are **immutable** - never update, only create:

```go
// WRONG: Modifying existing entry
entry.Amount = newAmount
repo.Update(entry)

// CORRECT: Create reversal entry
reversal := CreateReversalEntry(entry)
repo.Create(reversal)
```

### 3. **Webhook Verification**

Always verify webhook signatures:

```go
func verifyPaystackSignature(body []byte, signature string, secret string) bool {
    mac := hmac.New(sha512.New, []byte(secret))
    mac.Write(body)
    expected := hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(signature))
}
```

### 4. **PII Protection**

Sensitive data encrypted at rest:

- Bank account numbers
- Transfer codes
- Provider API responses

---

## Troubleshooting

### Issue: Wallet Balance Mismatch

**Symptom**: Wallet balance ≠ sum of ledger entries

**Diagnosis**:
```graphql
query {
  reconciliationReport(id: "latest") {
    discrepancies {
      type  # WALLET_MISMATCH
      walletId
      expectedValue
      actualValue
    }
  }
}
```

**Solution**:
1. Run reconciliation to identify affected wallet
2. Recalculate balance from ledger entries
3. Create adjustment entry if necessary
4. Investigate root cause (concurrent update? failed transaction?)

### Issue: Failed Disbursement Loop

**Symptom**: Same disbursement failing repeatedly

**Diagnosis**:
```graphql
query {
  disbursement(id: "disb-uuid") {
    attempts
    failureReason
    providerResponse
  }
}
```

**Common Causes**:
- Invalid bank account number
- Insufficient provider balance
- Provider API outage

**Solution**:
1. If invalid account: Update host bank details, retry manually
2. If provider issue: Wait for provider fix, contact support
3. If max retries exceeded: Cancel and create new disbursement

### Issue: Duplicate Charge

**Symptom**: Same payment charged twice

**Prevention**: Idempotency reference prevents this

**If it occurs**:
```graphql
query {
  financeTransactionHistory(
    resourceType: "booking"
    resourceId: "booking-uuid"
  ) {
    # Check for duplicate transactions
    id
    type
    amount
    createdAt
  }
}
```

**Solution**:
1. Identify duplicate transaction
2. Create reversal transaction
3. Refund guest for duplicate charge

---

## Appendix: Database Schema

```sql
-- Wallets
CREATE TABLE wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_type VARCHAR(50) NOT NULL,
    owner_id UUID NOT NULL,
    wallet_type VARCHAR(50) NOT NULL,
    balance BIGINT NOT NULL DEFAULT 0 CHECK (balance >= 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (owner_type, owner_id, wallet_type)
);

-- Ledger Entries
CREATE TABLE ledger_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL,
    reference VARCHAR(255) UNIQUE NOT NULL,
    debit_wallet_id UUID REFERENCES wallets(id),
    credit_wallet_id UUID REFERENCES wallets(id),
    amount BIGINT NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN',
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID NOT NULL,
    memo TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CHECK (debit_wallet_id IS NOT NULL OR credit_wallet_id IS NOT NULL)
);

-- Transactions
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID NOT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN',
    payment_id UUID,
    error_message TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Disbursements
CREATE TABLE disbursements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL REFERENCES wallets(id),
    transaction_id UUID NOT NULL REFERENCES transactions(id),
    amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN',
    provider VARCHAR(50) NOT NULL,
    transfer_code VARCHAR(255),
    provider_response TEXT,
    status VARCHAR(20) NOT NULL,
    attempts INT DEFAULT 0,
    next_retry_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    failure_reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Disputes
CREATE TABLE disputes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID NOT NULL,
    wallet_id UUID NOT NULL REFERENCES wallets(id),
    payment_id UUID NOT NULL,
    filed_by VARCHAR(10) NOT NULL,
    filed_by_id UUID NOT NULL,
    reason VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    description TEXT NOT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN',
    evidence JSONB,
    admin_notes TEXT,
    resolution JSONB,
    resolved_by_id UUID,
    resolved_at TIMESTAMPTZ,
    refund_amount BIGINT,
    transaction_id UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

## Support & Contact

For questions about the Finance System:
- **Technical Issues**: finance-team@hauslet.com
- **Payout Problems**: payouts@hauslet.com
- **Dispute Support**: disputes@hauslet.com
- **Reconciliation Alerts**: ops@hauslet.com

---

**Last Updated**: January 3, 2026  
**Document Version**: 1.0  
**Module Version**: Hauslet Services v1.0  
**Status**: Phase 3 Complete ✅ (Wallet, Ledger, Disputes operational)
