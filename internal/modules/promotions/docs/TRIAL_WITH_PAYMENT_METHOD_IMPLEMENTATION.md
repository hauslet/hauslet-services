# Trial Subscriptions with Payment Method - Implementation Guide

**Date:** January 5, 2026
**Author:** Claude Sonnet 4.5
**Status:** Partial Implementation - Requires Completion

## Overview

This document guides the implementation of trial subscriptions with saved payment methods (credit card required upfront). This enables automatic billing when trials end, significantly improving conversion rates.

---

## ✅ What's Already Implemented

### 1. Payment Module (100% Complete)

The payment module already has everything needed:

- ✅ `AuthorizePaymentMethod()` - Creates payment authorization without charging
- ✅ `SavePaymentMethod()` - Stores authorization code as payment method
- ✅ `GetDefaultPaymentMethod()` - Retrieves user's default payment method
- ✅ `CreatePayment()` with `PaymentMethodID` - Charges saved payment methods
- ✅ `PaymentMethod` domain with authorization codes and card metadata

**Location:** `internal/modules/payments/`

### 2. Subscription Domain (100% Complete)

- ✅ Added `PaymentMethodID *uuid.UUID` field
- ✅ Added helper methods: `HasPaymentMethod()`, `LinkPaymentMethod()`, `UnlinkPaymentMethod()`

**Location:** `internal/modules/promotions/domain/agent_subscription.go`

### 3. Repository Schema (100% Complete)

- ✅ Database schema updated with `payment_method_id` column
- ✅ Mapper functions updated (both directions)
- ✅ Migration created: `db/migrations/003_add_subscription_payment_method.sql`

**Location:** `internal/modules/promotions/repository/schema/`

### 4. Service Interface (100% Complete)

- ✅ `CreateSubscriptionInput` now has `PaymentMethodID *uuid.UUID` field

**Location:** `internal/modules/promotions/service/interface.go:163`

---

## 🔄 What Needs to Be Implemented

### Task 1: Update CreateSubscription Service Logic

**File:** `internal/modules/promotions/service/subscription_service.go:50`

**What to do:**
Add logic to handle payment methods in the `CreateSubscription` method.

**Implementation Steps:**

1. **After line 108 (in the trial subscription block):**

   ```go
   } else {
       // Trial subscription
       status = domain.SubscriptionStatusTrial
       trialEnd := calculateTrialEndDate(s.config, now)
       trialEndsAt = &trialEnd
       nextBillingDate = &trialEnd

       // NEW: If payment method provided, validate and link it
       if input.PaymentMethodID != nil {
           // Validate payment method exists and belongs to user
           paymentMethod, err := s.paymentService.GetPaymentMethod(ctx, *input.PaymentMethodID, input.UserID)
           if err != nil {
               return nil, fmt.Errorf("failed to get payment method: %w", err)
           }

           if paymentMethod == nil {
               return nil, fmt.Errorf("payment method not found")
           }

           if !paymentMethod.CanCharge() {
               return nil, fmt.Errorf("payment method cannot be charged (inactive or expired)")
           }

           s.log.Info("trial subscription with payment method",
               "subscription_id", subscriptionID,
               "payment_method_id", *input.PaymentMethodID,
               "card_last4", paymentMethod.Last4Digits,
           )
       }
   }
   ```

2. **After line 133 (after creating subscription entity):**

   ```go
   subscription := &domain.AgentSubscription{
       // ... all existing fields ...
       Features:                        planConfig.Features,

       // NEW: Link payment method if provided
       PaymentMethodID:                 input.PaymentMethodID,

       CreatedAt:                       now,
       UpdatedAt:                       now,
   }
   ```

**Why:** This validates and links the payment method to the subscription for future automatic billing.

---

### Task 2: Update ProcessBilling to Charge Saved Payment Methods

**File:** `internal/modules/promotions/service/subscription_service.go:621` (ProcessBilling method)

**What to do:**
Modify the payment creation logic to use saved payment methods when available.

**Implementation Steps:**

1. **Find the section around line 547-600 where payments are created**

2. **Replace the payment creation block with:**

   ```go
   // Determine amount to charge (existing logic)
   amountToCharge := subscription.Amount
   // ... existing pending plan change logic ...

   var payment *paymentDomain.Payment
   var err error

   // NEW: Check if subscription has saved payment method
   if subscription.HasPaymentMethod() {
       // Charge using saved payment method
       s.log.Info("charging saved payment method for recurring billing",
           "subscription_id", subscription.ID,
           "payment_method_id", *subscription.PaymentMethodID,
           "amount", amountToCharge,
       )

       paymentInput := paymentDomain.CreatePaymentInput{
           Amount:          amountToCharge,
           Currency:        payment.Currency(currencyToCharge),
           Market:          paymentDomain.MarketNigeria,
           PayerID:         subscription.UserID,
           PayerEmail:      subscription.UserEmail,
           PayerName:       subscription.UserName,
           ResourceType:    paymentDomain.ResourceTypeSubscription,
           ResourceID:      &subscription.ID,
           PaymentMethodID: subscription.PaymentMethodID, // Use saved method
           Description:     fmt.Sprintf("Subscription renewal - %s plan", planForDescription),
           Metadata: map[string]string{
               "subscription_id": subscription.ID.String(),
               "plan":            planForMetadata,
               "billing_cycle":   subscription.BillingCycle.String(),
               "billing_type":    "automatic_renewal",
           },
       }

       payment, err = s.paymentService.CreatePayment(ctx, paymentInput)
       if err != nil {
           s.log.Error("failed to charge saved payment method",
               "subscription_id", subscription.ID,
               "payment_method_id", *subscription.PaymentMethodID,
               "error", err,
           )

           // Mark subscription as past due
           if err := subscription.MarkPastDue(); err != nil {
               s.log.Error("failed to mark subscription as past due", "error", err)
           }

           // Save updated status
           subscriptionSchema, _ := schema.MapAgentSubscriptionToSchema(subscription)
           s.subscriptionRepo.Update(ctx, subscriptionSchema)

           // TODO: Send email to user to update payment method
           failureCount++
           continue
       }
   } else {
       // NO SAVED PAYMENT METHOD: Create payment link (existing flow)
       s.log.Info("no saved payment method, creating payment link",
           "subscription_id", subscription.ID,
       )

       // Use existing CreatePayment logic without PaymentMethodID
       paymentInput := paymentDomain.CreatePaymentInput{
           Amount:       amountToCharge,
           Currency:     payment.Currency(currencyToCharge),
           Market:       paymentDomain.MarketNigeria,
           PayerID:      subscription.UserID,
           PayerEmail:   subscription.UserEmail,
           PayerName:    subscription.UserName,
           ResourceType: paymentDomain.ResourceTypeSubscription,
           ResourceID:   &subscription.ID,
           Description:  fmt.Sprintf("Subscription renewal - %s plan", planForDescription),
           Metadata: map[string]string{
               "subscription_id": subscription.ID.String(),
               "plan":            planForMetadata,
               "billing_cycle":   subscription.BillingCycle.String(),
           },
       }

       payment, err = s.paymentService.CreatePayment(ctx, paymentInput)
       if err != nil {
           s.log.Error("failed to create payment for billing",
               "subscription_id", subscription.ID,
               "error", err,
           )
           failureCount++
           continue
       }

       // TODO: Send email with payment link
   }

   // Continue with existing payment processing logic...
   if payment.Status == paymentDomain.PaymentStatusSucceeded {
       // Renew subscription (existing logic)
   } else {
       // Mark as past due (existing logic)
   }
   ```

**Why:** This enables automatic charging for subscriptions with saved payment methods, while maintaining the manual payment link flow for those without.

---

### Task 3: Update GraphQL Schema

**File:** `internal/modules/promotions/port/graphql/schema.graphqls`

**What to do:**
Add payment method fields to subscription types and inputs.

**Implementation Steps:**

1. **Update `CreateSubscriptionInput` (around line 122):**

   ```graphql
   input CreateSubscriptionInput {
     planType: PlanType!
     billingCycle: BillingCycle!
     startTrial: Boolean!
     paymentMethodID: UUID  # NEW: Optional payment method to link
   }
   ```

2. **Update `AgentSubscription` type (around line 25-41):**

   ```graphql
   type AgentSubscription {
     id: UUID!
     userID: UUID!
     planType: PlanType!
     billingCycle: BillingCycle!
     status: SubscriptionStatus!
     # ... existing fields ...

     # NEW: Payment method info
     hasPaymentMethod: Boolean!
     paymentMethodID: UUID

     createdAt: Time!
     updatedAt: Time!
   }
   ```

3. **Add new query (around line 165):**

   ```graphql
   extend type Query {
     # ... existing queries ...

     # NEW: Get payment methods for subscription management
     getMyPaymentMethods: [PaymentMethod!]!
   }
   ```

4. **Add PaymentMethod type (if not already present):**

   ```graphql
   type PaymentMethod {
     id: UUID!
     type: String!
     last4Digits: String
     cardType: String
     brand: String
     expiryMonth: Int
     expiryYear: Int
     isDefault: Boolean!
     isActive: Boolean!
     isExpired: Boolean!
     displayName: String!
   }
   ```

---

### Task 4: Update GraphQL Resolvers

**File:** `internal/modules/promotions/port/graphql/resolver.go`

**What to do:**
Wire up the new GraphQL fields to service calls.

**Implementation Steps:**

1. **Update `createSubscription` mutation resolver:**

   ```go
   func (r *mutationResolver) CreateSubscription(ctx context.Context, input CreateSubscriptionInput) (*CreateSubscriptionPayload, error) {
       userID := auth.GetUserIDFromContext(ctx)
       user := auth.GetUserFromContext(ctx)

       serviceInput := service.CreateSubscriptionInput{
           UserID:          userID,
           UserEmail:       user.Email,
           UserName:        user.Name,
           PlanType:        input.PlanType,
           BillingCycle:    input.BillingCycle,
           StartTrial:      input.StartTrial,
           PaymentMethodID: input.PaymentMethodID, // NEW: Pass payment method ID
       }

       result, err := r.subscriptionService.CreateSubscription(ctx, serviceInput)
       // ... rest of resolver
   }
   ```

2. **Add resolver for `hasPaymentMethod` field:**

   ```go
   func (r *agentSubscriptionResolver) HasPaymentMethod(ctx context.Context, obj *domain.AgentSubscription) (bool, error) {
       return obj.HasPaymentMethod(), nil
   }
   ```

3. **Add resolver for `getMyPaymentMethods` query:**

   ```go
   func (r *queryResolver) GetMyPaymentMethods(ctx context.Context) ([]*paymentDomain.PaymentMethod, error) {
       userID := auth.GetUserIDFromContext(ctx)

       methods, err := r.paymentService.ListPaymentMethods(ctx, userID)
       if err != nil {
           return nil, fmt.Errorf("failed to list payment methods: %w", err)
       }

       // Convert to pointers
       result := make([]*paymentDomain.PaymentMethod, len(methods))
       for i := range methods {
           result[i] = &methods[i]
       }

       return result, nil
   }
   ```

---

## 📋 Client-Side Integration Guide

### Trial Signup Flow with Payment Method

**Step 1: User initiates trial signup**

```graphql
mutation StartTrial($paymentMethodID: UUID) {
  createSubscription(
    input: {
      planType: BASIC
      billingCycle: MONTHLY
      startTrial: true
      paymentMethodID: $paymentMethodID  # Obtained from Paystack authorization
    }
  ) {
    subscription {
      id
      status
      trialEndsAt
      hasPaymentMethod
    }
    paymentURL
    paymentID
  }
}
```

**Step 2: Client handles two scenarios**

**Scenario A: User has saved payment method**

- Pass existing `paymentMethodID`
- Trial starts immediately with card on file
- Auto-billing when trial ends

**Scenario B: User needs to add payment method**

1. Client calls Paystack authorization API first
2. User completes card authorization (no charge)
3. Client receives authorization code
4. Client calls payment service to save method:

   ```graphql
   mutation SaveCard($authCode: String!) {
     savePaymentMethod(authorizationCode: $authCode) {
       id
       last4Digits
       cardType
     }
   }
   ```

5. Client calls `createSubscription` with new payment method ID

---

## 🧪 Testing Checklist

### Unit Tests to Add

1. **Domain Tests:**
   - [ ] `HasPaymentMethod()` returns true when ID present
   - [ ] `LinkPaymentMethod()` sets ID and updates timestamp
   - [ ] `UnlinkPaymentMethod()` clears ID

2. **Service Tests:**
   - [ ] CreateSubscription validates payment method ownership
   - [ ] CreateSubscription rejects inactive/expired payment methods
   - [ ] CreateSubscription links payment method to subscription
   - [ ] ProcessBilling charges saved payment method
   - [ ] ProcessBilling falls back to payment link when no method saved

3. **Integration Tests:**
   - [ ] Trial with payment method converts to active on trial end
   - [ ] Failed payment marks subscription as past_due
   - [ ] User can update payment method on subscription

### Manual Testing Scenarios

1. **Happy Path:**
   - Create trial with valid payment method
   - Wait for trial to end (or manually trigger billing)
   - Verify automatic charge succeeds
   - Verify subscription becomes ACTIVE

2. **Payment Failure:**
   - Create trial with payment method
   - Expire/deactivate the card
   - Trigger billing
   - Verify subscription goes PAST_DUE
   - Verify user receives notification

3. **No Payment Method:**
   - Create trial without payment method
   - Trigger billing
   - Verify payment link is generated
   - Verify email sent to user

---

## 🚨 Important Notes

### Security Considerations

1. **Always validate payment method ownership:**
   - Never allow user A to use user B's payment method
   - Check `paymentMethod.UserID == subscription.UserID`

2. **Validate payment method status:**
   - Check `paymentMethod.CanCharge()` before linking
   - Handle expired cards gracefully

3. **PCI Compliance:**
   - Never store raw card numbers
   - Only store Paystack authorization codes
   - Card details (last4, brand) are display-only

### Error Handling

1. **Payment method not found:**
   - Return clear error to user
   - Don't create subscription

2. **Card expired:**
   - Allow trial to start
   - Send warning email
   - Remind user to update card before trial ends

3. **Automatic charge fails:**
   - Mark subscription PAST_DUE (not EXPIRED immediately)
   - Give 7-day grace period
   - Send multiple reminder emails

### Monitoring & Alerts

Set up alerts for:

- High trial-to-paid conversion failures
- Spike in past_due subscriptions
- Payment method authorization failures

---

## 📝 Migration Steps

### Running the Migration

```bash
# The GORM auto-migration will handle this automatically on app start
# But you can also run the SQL migration explicitly:

# Development
make migrate-up

# Production
goose -dir db/migrations postgres "postgresql://..." up
```

### Rollback Plan

If issues arise:

```bash
# Rollback migration
goose -dir db/migrations postgres "postgresql://..." down

# Or manually:
DROP INDEX IF EXISTS idx_subscription_payment_method;
ALTER TABLE agent_subscriptions DROP COLUMN IF EXISTS payment_method_id;
```

---

## 🎯 Success Metrics

Track these metrics after implementation:

1. **Trial Conversion Rate:**
   - Before: ~20-30% (no card required)
   - Target: ~60-70% (card required upfront)

2. **Payment Failure Rate:**
   - Target: <5% for first charge after trial
   - Alert if >10%

3. **User Friction:**
   - Monitor trial signup abandonment at payment step
   - A/B test card-required vs no-card trials

---

## 🔗 Related Documentation

- [PROMOTIONS_GUIDE.md](./PROMOTIONS_GUIDE.md) - User-facing documentation
- Payment Module Documentation: `internal/modules/payments/README.md`
- Paystack Integration: `docs/PAYSTACK_INTEGRATION.md`

---

## ✅ Implementation Checklist

Before marking complete, ensure:

- [ ] All code changes implemented
- [ ] Migration run successfully
- [ ] Unit tests added and passing
- [ ] Integration tests passing
- [ ] Manual testing completed
- [ ] Error handling tested
- [ ] Monitoring/alerts configured
- [ ] Documentation updated
- [ ] Code reviewed
- [ ] Deployed to staging
- [ ] Smoke tested in production

---

**Questions?** Contact the team or refer to the payment module documentation for Paystack integration details.
