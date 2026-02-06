# Implementation Plan: Progressive Host Cancellation Penalty

**Status**: 📋 PLANNED  
**Priority**: HIGH  
**Estimated Effort**: 3-5 days  
**Author**: Engineering Team  
**Created**: 2026-02-06  

---

## 1. Problem Statement

### Current Behavior
When a host cancels a booking, the guest receives a **100% refund** (including service fees), and the **platform absorbs the ~2% processing fee** (~₦2,000 on a ₦100k booking). The host faces **zero financial consequences**.

### Code Evidence
```go
// internal/modules/pricing/service/refund_calculator.go:143-155
func (s *PricingServiceImpl) calculateHostCancellationRefund(...) {
    breakdown.ProcessingFee = 0                    // ← No fee calculated
    breakdown.ProcessingFeePayer = "host"          // ← Label but no action
    breakdown.NetRefund = breakdown.OriginalAmount // ← Guest gets 100%
}
```

### Business Risk
- Hosts can accept bookings as "placeholders" and cancel when better offers arrive
- Platform loses money on every host cancellation
- Guest experience degradation (last-minute scrambles)
- No incentive for hosts to maintain commitment

---

## 2. Proposed Solution: Progressive Penalty System

### Penalty Tiers

| Cancellation # (in 30 days) | Penalty | Additional Action |
|-----------------------------|---------|-------------------|
| 1st | ₦0 (Warning only) | Email warning sent |
| 2nd | ₦5,000 | Deducted from host wallet |
| 3rd | ₦10,000 | + 7-day listing suspension |
| 4th+ | ₦15,000 | + 14-day suspension + manual review |

### Key Principles
1. **First cancellation is forgiven** (emergencies happen)
2. **Penalties escalate** to deter repeat offenders
3. **Suspensions** remove listings from search temporarily
4. **All penalties are transparent** (shown before host confirms cancellation)

---

## 3. Technical Implementation

### Phase 1: Data Model Changes

#### 3.1.1 Add Host Cancellation Tracking Table

**File**: `internal/modules/profile/repository/schema/gorm.go`

```go
// HostCancellationRecord tracks host cancellations for penalty calculation
type HostCancellationRecord struct {
    ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    HostID        uuid.UUID  `gorm:"type:uuid;not null;index:idx_host_cancellations_host_id"`
    BookingID     uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"`
    CancelledAt   time.Time  `gorm:"not null"`
    PenaltyAmount int64      `gorm:"not null;default:0"` // Minor units
    PenaltyPaid   bool       `gorm:"not null;default:false"`
    WarningSent   bool       `gorm:"not null;default:false"`
    Reason        string     `gorm:"type:text"`
    CreatedAt     time.Time  `gorm:"not null;default:now()"`
}
```

**Migration SQL**:
```sql
CREATE TABLE host_cancellation_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host_id UUID NOT NULL REFERENCES users(id),
    booking_id UUID NOT NULL UNIQUE REFERENCES bookings(id),
    cancelled_at TIMESTAMPTZ NOT NULL,
    penalty_amount BIGINT NOT NULL DEFAULT 0,
    penalty_paid BOOLEAN NOT NULL DEFAULT false,
    warning_sent BOOLEAN NOT NULL DEFAULT false,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_host_cancellations_host_id ON host_cancellation_records(host_id);
CREATE INDEX idx_host_cancellations_cancelled_at ON host_cancellation_records(cancelled_at);
```

#### 3.1.2 Add Suspension Fields to Listing

**File**: `internal/modules/property/repository/schema/listing.go`

```go
// Add to Listing struct:
SuspendedUntil *time.Time `gorm:"index"`
SuspensionReason string   `gorm:"type:text"`
```

---

### Phase 2: Configuration

#### 3.2.1 Add Penalty Config to Platform YAML

**File**: `config/defaults/platform.yaml`

```yaml
platform:
  host_cancellation:
    # Penalty tiers (in minor units - kobo)
    penalties:
      - count: 1
        amount: 0           # First cancellation: warning only
        suspension_days: 0
      - count: 2
        amount: 500000      # ₦5,000
        suspension_days: 0
      - count: 3
        amount: 1000000     # ₦10,000
        suspension_days: 7
      - count: 4
        amount: 1500000     # ₦15,000
        suspension_days: 14
        requires_review: true  # Flag for manual review
    
    # Rolling window for counting cancellations
    window_days: 30
    
    # Grace period for new hosts (first 90 days, no penalties)
    new_host_grace_days: 90
    
    # Notify host X hours before suspension ends
    suspension_ending_notice_hours: 24
```

#### 3.2.2 Add Config Struct

**File**: `config/yaml.go`

```go
type HostCancellationConfig struct {
    Penalties               []PenaltyTier `yaml:"penalties"`
    WindowDays              int           `yaml:"window_days"`
    NewHostGraceDays        int           `yaml:"new_host_grace_days"`
    SuspensionEndingNotice  int           `yaml:"suspension_ending_notice_hours"`
}

type PenaltyTier struct {
    Count          int   `yaml:"count"`
    Amount         int64 `yaml:"amount"`          // Minor units
    SuspensionDays int   `yaml:"suspension_days"`
    RequiresReview bool  `yaml:"requires_review"`
}
```

---

### Phase 3: Service Layer

#### 3.3.1 Create Host Penalty Service

**File**: `internal/modules/profile/service/host_penalty.go`

```go
package service

type HostPenaltyService interface {
    // GetCancellationCountInWindow returns # of cancellations in the rolling window
    GetCancellationCountInWindow(ctx context.Context, hostID uuid.UUID) (int, error)
    
    // CalculatePenalty determines the penalty for the next cancellation
    CalculatePenalty(ctx context.Context, hostID uuid.UUID) (*PenaltyResult, error)
    
    // RecordCancellation logs the cancellation and applies penalty
    RecordCancellation(ctx context.Context, input RecordCancellationInput) error
    
    // ApplySuspension suspends all host listings for N days
    ApplySuspension(ctx context.Context, hostID uuid.UUID, days int, reason string) error
    
    // CheckAndLiftSuspensions (cron job) lifts expired suspensions
    CheckAndLiftSuspensions(ctx context.Context) error
}

type PenaltyResult struct {
    CancellationCount   int
    PenaltyAmount       int64  // Minor units
    SuspensionDays      int
    RequiresReview      bool
    IsNewHostGracePeriod bool
    WarningMessage      string
}

type RecordCancellationInput struct {
    HostID       uuid.UUID
    BookingID    uuid.UUID
    CancelledAt  time.Time
    Reason       string
}
```

#### 3.3.2 Implementation Logic

```go
func (s *HostPenaltyServiceImpl) CalculatePenalty(ctx context.Context, hostID uuid.UUID) (*PenaltyResult, error) {
    // 1. Check if host is in new-host grace period
    host, err := s.profileRepo.GetProfile(ctx, hostID)
    if err != nil {
        return nil, err
    }
    
    graceDays := s.config.HostCancellation.NewHostGraceDays
    if time.Since(host.CreatedAt).Hours() < float64(graceDays*24) {
        return &PenaltyResult{
            IsNewHostGracePeriod: true,
            WarningMessage: fmt.Sprintf(
                "You're within your %d-day new host grace period. This cancellation won't incur a penalty, but please avoid frequent cancellations.",
                graceDays,
            ),
        }, nil
    }
    
    // 2. Count cancellations in rolling window
    windowDays := s.config.HostCancellation.WindowDays
    cutoff := time.Now().AddDate(0, 0, -windowDays)
    count, err := s.repo.CountCancellationsSince(ctx, hostID, cutoff)
    if err != nil {
        return nil, err
    }
    
    // 3. Find applicable penalty tier
    nextCount := count + 1
    var tier PenaltyTier
    for _, t := range s.config.HostCancellation.Penalties {
        if t.Count == nextCount {
            tier = t
            break
        }
        if t.Count > nextCount {
            break
        }
        tier = t // Use highest applicable tier
    }
    
    // 4. Build result
    result := &PenaltyResult{
        CancellationCount: nextCount,
        PenaltyAmount:     tier.Amount,
        SuspensionDays:    tier.SuspensionDays,
        RequiresReview:    tier.RequiresReview,
    }
    
    // 5. Generate warning message
    if tier.Amount == 0 {
        result.WarningMessage = "This is your first cancellation in 30 days. No penalty will apply, but repeated cancellations will incur fees."
    } else {
        result.WarningMessage = fmt.Sprintf(
            "This is cancellation #%d in the last 30 days. A penalty of ₦%.2f will be deducted from your wallet.",
            nextCount, float64(tier.Amount)/100,
        )
        if tier.SuspensionDays > 0 {
            result.WarningMessage += fmt.Sprintf(" Your listings will be suspended for %d days.", tier.SuspensionDays)
        }
    }
    
    return result, nil
}
```

---

### Phase 4: Integration Points

#### 3.4.1 Modify Booking Cancellation Flow

**File**: `internal/modules/booking/service/service.go`

In `CancelBooking()`, add penalty logic BEFORE processing refund:

```go
func (s *BookingServiceImpl) CancelBooking(ctx context.Context, input CancelBookingInput) (*domain.Booking, error) {
    // ... existing validation ...
    
    // NEW: Check if host-initiated cancellation
    if input.CancelledBy == domain.CancelledByHost {
        // 1. Calculate penalty
        penalty, err := s.hostPenaltyService.CalculatePenalty(ctx, booking.HostID)
        if err != nil {
            return nil, fmt.Errorf("failed to calculate penalty: %w", err)
        }
        
        // 2. If penalty exists, deduct from wallet
        if penalty.PenaltyAmount > 0 {
            err := s.financeHooks.DeductPenalty(ctx, booking.HostID, penalty.PenaltyAmount, booking.ID)
            if err != nil {
                // If wallet insufficient, block cancellation or flag for review
                if errors.Is(err, finance.ErrInsufficientBalance) {
                    return nil, fmt.Errorf("insufficient wallet balance for cancellation penalty (₦%.2f required)", 
                        float64(penalty.PenaltyAmount)/100)
                }
                return nil, err
            }
        }
        
        // 3. Record the cancellation
        err = s.hostPenaltyService.RecordCancellation(ctx, RecordCancellationInput{
            HostID:      booking.HostID,
            BookingID:   booking.ID,
            CancelledAt: time.Now(),
            Reason:      input.Reason,
        })
        if err != nil {
            s.log.Error("failed to record host cancellation", "error", err)
            // Non-blocking - continue with cancellation
        }
        
        // 4. Apply suspension if required
        if penalty.SuspensionDays > 0 {
            err = s.hostPenaltyService.ApplySuspension(ctx, booking.HostID, penalty.SuspensionDays, 
                fmt.Sprintf("Automatic suspension after %d cancellations in 30 days", penalty.CancellationCount))
            if err != nil {
                s.log.Error("failed to apply suspension", "error", err)
            }
        }
    }
    
    // ... continue with existing refund logic ...
}
```

#### 3.4.2 Add Finance Hook for Penalty Deduction

**File**: `internal/modules/finance/service/ledger.go`

```go
// DeductPenalty withdraws penalty amount from host wallet
func (s *LedgerServiceImpl) DeductPenalty(
    ctx context.Context, 
    hostID uuid.UUID, 
    amount int64, 
    bookingID uuid.UUID,
) error {
    // 1. Get host wallet
    wallet, err := s.GetWallet(ctx, hostID, domain.WalletTypeHost)
    if err != nil {
        return err
    }
    
    // 2. Check balance
    if wallet.Balance < amount {
        return ErrInsufficientBalance
    }
    
    // 3. Create ledger entry
    entry := &domain.LedgerEntry{
        WalletID:    wallet.ID,
        Type:        domain.EntryTypePenalty,
        Amount:      -amount, // Negative = deduction
        Currency:    wallet.Currency,
        Description: fmt.Sprintf("Host cancellation penalty for booking %s", bookingID),
        ReferenceID: &bookingID,
        CreatedAt:   time.Now(),
    }
    
    // 4. Update wallet balance
    wallet.Balance -= amount
    
    // 5. Persist
    if err := s.repo.CreateLedgerEntry(ctx, entry); err != nil {
        return err
    }
    if err := s.repo.UpdateWallet(ctx, wallet); err != nil {
        return err
    }
    
    s.log.Info("penalty deducted", "host_id", hostID, "amount", amount, "booking_id", bookingID)
    return nil
}
```

#### 3.4.3 Modify Listing Discovery to Exclude Suspended

**File**: `internal/modules/discovery/service/service.go`

```go
// In buildListingQuery(), add:
query = query.Where("suspended_until IS NULL OR suspended_until < ?", time.Now())
```

---

### Phase 5: API Changes

#### 3.5.1 Add GraphQL Query for Penalty Preview

**File**: `internal/transport/graph/schema/booking.graphqls`

```graphql
type CancellationPenaltyPreview {
    cancellationCount: Int!
    penaltyAmount: Float!           # In major units (Naira)
    suspensionDays: Int!
    isNewHostGracePeriod: Boolean!
    warningMessage: String!
}

extend type Query {
    """
    Preview the penalty that would apply if the host cancels this booking.
    Only available to the host of the booking.
    """
    previewHostCancellationPenalty(bookingId: UUID!): CancellationPenaltyPreview!
}
```

#### 3.5.2 Add GraphQL Mutation Response Field

**File**: Modify `CancelBookingPayload`

```graphql
type CancelBookingPayload {
    booking: Booking!
    refundAmount: Float!
    refundBreakdown: RefundBreakdown
    # NEW FIELDS:
    penaltyApplied: Float           # Amount deducted from host wallet (null if guest cancelled)
    suspensionApplied: Int          # Days of suspension (null if none)
}
```

---

### Phase 6: Notifications

#### 3.6.1 Email Templates

**File**: `internal/modules/booking/templates/host_cancellation_warning.html`

Subject: "Your first booking cancellation - Important information"

```html
<h1>Booking Cancellation Processed</h1>
<p>Hi {{.HostName}},</p>
<p>We've processed your cancellation for booking #{{.BookingRef}}.</p>

<div class="warning-box">
    <h3>⚠️ Important Notice</h3>
    <p>This is your <strong>first cancellation</strong> in the last 30 days. 
    No penalty has been applied this time.</p>
    <p>However, please be aware of our cancellation policy:</p>
    <ul>
        <li>2nd cancellation: ₦5,000 penalty</li>
        <li>3rd cancellation: ₦10,000 + 7-day listing suspension</li>
        <li>4th+ cancellations: ₦15,000 + 14-day suspension</li>
    </ul>
</div>
```

**File**: `internal/modules/booking/templates/host_cancellation_penalty.html`

Subject: "Cancellation penalty applied - ₦{{.PenaltyAmount}}"

```html
<h1>Cancellation Penalty Applied</h1>
<p>Hi {{.HostName}},</p>

<div class="penalty-box">
    <p>A penalty of <strong>₦{{.PenaltyAmount}}</strong> has been deducted 
    from your wallet for cancelling booking #{{.BookingRef}}.</p>
    
    {{if .SuspensionDays}}
    <p>Additionally, your listings have been suspended for 
    <strong>{{.SuspensionDays}} days</strong> until {{.SuspensionEndDate}}.</p>
    {{end}}
</div>

<p>This was cancellation #{{.CancellationCount}} in the last 30 days.</p>
```

---

### Phase 7: Cron Jobs

#### 3.7.1 Suspension Lifter Job

**File**: `internal/jobs/host_suspension_lifter.go`

```go
// RunHostSuspensionLifter checks for expired suspensions and lifts them
func RunHostSuspensionLifter(ctx context.Context, listingRepo ListingRepository, notifSvc NotificationService) error {
    // Find listings where suspended_until < now
    listings, err := listingRepo.FindExpiredSuspensions(ctx, time.Now())
    if err != nil {
        return err
    }
    
    for _, listing := range listings {
        listing.SuspendedUntil = nil
        listing.SuspensionReason = ""
        
        if err := listingRepo.Update(ctx, listing); err != nil {
            log.Error("failed to lift suspension", "listing_id", listing.ID, "error", err)
            continue
        }
        
        // Notify host
        notifSvc.SendSuspensionLiftedNotification(listing.OwnerID, listing.ID)
    }
    
    return nil
}
```

**Schedule**: Run every hour via Cloud Scheduler

---

## 4. Testing Strategy

### 4.1 Unit Tests

**File**: `internal/modules/profile/service/host_penalty_test.go`

| Test Case | Input | Expected Output |
|-----------|-------|-----------------|
| First cancellation (new host) | Host created 30 days ago | Warning only, ₦0 penalty |
| First cancellation (established host) | Host created 120 days ago | Warning only, ₦0 penalty |
| Second cancellation | 1 prior in last 30 days | ₦5,000 penalty |
| Third cancellation | 2 prior in last 30 days | ₦10,000 + 7-day suspension |
| Old cancellations don't count | 1 prior 45 days ago | Warning only, ₦0 penalty |
| Insufficient wallet balance | ₦2,000 balance, ₦5,000 penalty | ErrInsufficientBalance |

### 4.2 Integration Tests

**File**: `internal/modules/booking/service/cancel_booking_host_penalty_test.go`

```go
func TestCancelBooking_HostPenalty_SecondCancellation(t *testing.T) {
    // Setup: Create host with 1 prior cancellation in window
    // Action: Host cancels booking
    // Assert:
    //   - ₦5,000 deducted from wallet
    //   - Cancellation record created
    //   - Guest received 100% refund
    //   - Host received penalty notification email
}

func TestCancelBooking_HostPenalty_ThirdCancellation_Suspension(t *testing.T) {
    // Setup: Create host with 2 prior cancellations
    // Action: Host cancels booking
    // Assert:
    //   - ₦10,000 deducted
    //   - All host listings have suspended_until set to +7 days
    //   - Listings no longer appear in discovery search
}
```

---

## 5. Rollout Plan

### Phase 1: Shadow Mode (Week 1-2)
- Deploy code but **do not deduct penalties**
- Log what penalties *would* have been applied
- Monitor volume of host cancellations
- Validate calculation logic

### Phase 2: Soft Launch (Week 3-4)
- Enable for **new hosts only** (signed up after launch date)
- Full penalty enforcement
- Monitor support tickets

### Phase 3: Full Launch (Week 5+)
- Enable for **all hosts**
- Send announcement email explaining new policy
- Update Terms of Service

---

## 6. Metrics to Track

| Metric | Baseline | Target |
|--------|----------|--------|
| Host cancellation rate | TBD | -50% |
| Platform processing fee absorption | ~₦X/month | -80% |
| Repeat host cancellations (2+ in 30 days) | TBD | -70% |
| Guest complaints about host cancellations | TBD | -60% |

---

## 7. Dependencies

### Modules Affected
- `profile` - Host cancellation tracking
- `booking` - Cancellation flow modification
- `finance` - Penalty deduction
- `property` - Listing suspension
- `discovery` - Hide suspended listings
- `notifications` - Penalty emails

### External Dependencies
- None (uses existing wallet system)

---

## 8. Open Questions

1. **Should penalties be waived for documented emergencies?**
   - Option: Allow admin to waive via support ticket
   
2. **What happens if host has no wallet balance?**
   - Option A: Block cancellation (guest stuck)
   - Option B: Allow, mark penalty as "owed" (debt)
   - **Recommendation**: Option B with debt collection from future payouts

3. **Should we notify the guest about host penalty?**
   - Pro: Transparency, guest feels "justice served"
   - Con: May seem petty, TMI
   - **Recommendation**: No, keep internal

---

## 9. File Checklist

When implementing, create/modify these files:

### New Files
- [ ] `internal/modules/profile/repository/schema/host_cancellation_record.go`
- [ ] `internal/modules/profile/service/host_penalty.go`
- [ ] `internal/modules/profile/service/host_penalty_test.go`
- [ ] `internal/modules/booking/templates/host_cancellation_warning.html`
- [ ] `internal/modules/booking/templates/host_cancellation_penalty.html`
- [ ] `internal/jobs/host_suspension_lifter.go`
- [ ] `migrations/XXXXXX_add_host_cancellation_tracking.sql`

### Modified Files
- [ ] `config/defaults/platform.yaml` (add host_cancellation section)
- [ ] `config/yaml.go` (add HostCancellationConfig struct)
- [ ] `internal/modules/property/repository/schema/listing.go` (add suspension fields)
- [ ] `internal/modules/booking/service/service.go` (integrate penalty in CancelBooking)
- [ ] `internal/modules/finance/service/ledger.go` (add DeductPenalty method)
- [ ] `internal/modules/finance/service/interface.go` (add DeductPenalty to interface)
- [ ] `internal/modules/discovery/service/service.go` (exclude suspended listings)
- [ ] `internal/transport/graph/schema/booking.graphqls` (add penalty preview query)

---

**End of Implementation Plan**
Remaining Items for Future Work
The following items were identified but not implemented as they require more planning:

