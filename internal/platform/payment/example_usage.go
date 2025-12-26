package payment

/*
EXAMPLE USAGE - DO NOT IMPORT THIS FILE IN PRODUCTION

This file demonstrates how to use the payment platform abstraction.
Your future `internal/modules/payment` (transaction) module will use this platform layer.

=============================================================================
INITIALIZATION
=============================================================================
*/

// Initialize the payment client (typically done once in your app setup)
func ExampleInitialization() {
	// Load config
	// cfg := config.Load()

	// Create provider factory with your payment credentials
	// factory := NewProviderFactory(cfg.Services.Payment)

	// Create payment client
	// paymentClient := New(factory)

	// Now use paymentClient in your payment/transaction module
}

/*
=============================================================================
GUEST PAYMENT FLOWS (Collecting Money)
=============================================================================
*/

// Example 1: Initialize a new payment (First-time payment)
func ExampleInitializePayment() {
	/*
		// In your Payments Module
		ctx := context.Background()

		resp, err := paymentClient.Initialize(ctx, payment.PaymentRequest{
			Amount:      100050, // NGN 1000.50 in kobo (use ToMinorUnits helper)
			Currency:    payment.NGN,
			Reference:   "BKG-12345",
			Email:       "guest@example.com",
			CallbackURL: "https://app.hauslet.com/payment/callback",
			Metadata: map[string]string{
				"booking_id": "12345",
				"user_id":    "67890",
			},
		})

		if err != nil {
			// Handle error
			return
		}

		// For NGN (Paystack): User gets redirected to checkout
		if resp.RedirectURL != "" {
			// Return redirect URL to frontend
			// Frontend redirects user to: resp.RedirectURL
		}

		// For USD/GHS (Stripe): Frontend uses client secret
		if resp.RequiresAction {
			clientSecret := resp.ActionPayload["client_secret"]
			// Return client secret to frontend
			// Frontend completes payment with Stripe.js
		}
	*/
}

// Example 2: Charge a saved card (One-click payment)
func ExampleChargeAuthorization() {
	/*
		// In your Payments Module
		ctx := context.Background()

		// For Paystack (NGN): Use authorization_code from previous transaction
		authCode := "AUTH_xyz123"

		// For Stripe (USD/GHS): Use payment_method_id
		// paymentMethodID := "pm_abc456"

		resp, err := paymentClient.ChargeAuthorization(ctx, payment.PaymentRequest{
			Amount:    50000, // NGN 500.00
			Currency:  payment.NGN,
			Reference: "BKG-67890",
			Email:     "guest@example.com",
			AuthToken: &authCode,
			Metadata: map[string]string{
				"booking_id": "67890",
			},
		})

		if err != nil {
			// Handle authorization failure
			return
		}

		if resp.Status == payment.StatusSuccess {
			// Payment successful - update booking status
		} else if resp.RequiresAction {
			// Additional verification needed (3DS/OTP)
			// Prompt user for OTP or 3DS verification
		}
	*/
}

// Example 3: Verify payment status (After redirect callback)
func ExampleVerifyPayment() {
	/*
		// In your callback handler
		ctx := context.Background()
		reference := "BKG-12345"

		resp, err := paymentClient.Verify(ctx, payment.NGN, reference)
		if err != nil {
			// Handle error
			return
		}

		if resp.Status == payment.StatusSuccess {
			// Payment confirmed - activate booking
			// Save transaction details to database
		} else {
			// Payment failed or pending
		}
	*/
}

// Example 4: Refund a payment
func ExampleRefundPayment() {
	/*
		// In your Payments Module
		ctx := context.Background()

		resp, err := paymentClient.Refund(
			ctx,
			payment.NGN,
			"123456789", // Original transaction ID from provider
			50000,       // Partial refund: NGN 500 (or 0 for full refund)
			"Booking cancelled by host",
		)

		if err != nil {
			// Handle refund failure
			return
		}

		if resp.Success {
			// Refund initiated - update booking and notify guest
		}
	*/
}

/*
=============================================================================
HOST PAYOUT FLOWS (Sending Money)
=============================================================================
*/

// Example 5: Validate bank account before creating recipient
func ExampleValidateBankAccount() {
	/*
		// In your Payout Module
		ctx := context.Background()

		// Paystack bank codes: https://paystack.com/docs/api/miscellaneous/#bank
		accountName, err := paymentClient.ValidateAccount(
			ctx,
			payment.NGN,
			"058", // GTBank
			"0123456789",
		)

		if err != nil {
			// Invalid account
			return
		}

		// Show account name to host for confirmation
		// "John Doe" - is this correct?
	*/
}

// Example 6: Create transfer recipient (One-time setup per host)
func ExampleCreateRecipient() {
	/*
		// In your Payout Module (when host adds bank details)
		ctx := context.Background()

		recipientCode, err := paymentClient.CreateRecipient(
			ctx,
			payment.NGN,
			"058",         // GTBank
			"0123456789",  // Account number
			"John Doe",    // Account name (from validation)
		)

		if err != nil {
			// Handle error
			return
		}

		// Store recipientCode in database for this host
		// host.PaystackRecipientCode = recipientCode
		// Save(host)
	*/
}

// Example 7: Transfer money to host (Escrow release)
func ExampleTransferPayout() {
	/*
		// In your Payout Module (when releasing escrow)
		ctx := context.Background()

		resp, err := paymentClient.Transfer(ctx, payment.PayoutRequest{
			Amount:        450000, // NGN 4500.00 (booking payout)
			Currency:      payment.NGN,
			RecipientCode: host.PaystackRecipientCode, // From database
			Reference:     fmt.Sprintf("PAYOUT-BKG-%s", booking.ID),
			Narration:     fmt.Sprintf("Payout for booking #%s", booking.ID),
			Metadata: map[string]string{
				"booking_id": booking.ID.String(),
				"host_id":    host.ID.String(),
			},
		})

		if err != nil {
			// Handle transfer failure
			return
		}

		if resp.Success {
			// Transfer initiated - save transaction record
			// Note: Transfer may still be pending - use webhooks or polling
		}
	*/
}

// Example 8: Verify transfer status
func ExampleVerifyTransfer() {
	/*
		// In your Payout Module
		ctx := context.Background()

		resp, err := paymentClient.VerifyTransfer(
			ctx,
			payment.NGN,
			"PAYOUT-BKG-12345", // Your internal reference or provider's transfer ID
		)

		if err != nil {
			// Handle error
			return
		}

		if resp.Status == payment.TransferSuccess {
			// Transfer completed - notify host
		} else if resp.Status == payment.TransferFailed {
			// Transfer failed - investigate and retry
		}
	*/
}

/*
=============================================================================
WEBHOOK HANDLING
=============================================================================
*/

// Example 9: Handle Paystack webhook
func ExamplePaystackWebhook() {
	/*
		// In your webhook handler
		func HandlePaystackWebhook(w http.ResponseWriter, r *http.Request) {
			payload, _ := io.ReadAll(r.Body)
			signature := r.Header.Get("X-Paystack-Signature")

			// Verify webhook signature
			valid, err := paymentClient.VerifyWebhookSignature("paystack", signature, payload)
			if err != nil || !valid {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// Parse webhook event
			event, err := paymentClient.ParseWebhookEvent("paystack", payload)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Handle different event types
			switch event.Type {
			case "charge.success":
				// Payment successful - update booking
				// reference := event.Reference
				// UpdateBookingStatus(reference, "paid")

			case "transfer.success":
				// Payout successful - notify host
				// reference := event.Reference
				// NotifyHostPayoutComplete(reference)

			case "transfer.failed":
				// Payout failed - alert admin
				// reference := event.Reference
				// AlertAdminPayoutFailed(reference)
			}

			w.WriteHeader(http.StatusOK)
		}
	*/
}

// Example 10: Handle Stripe webhook
func ExampleStripeWebhook() {
	/*
		// In your webhook handler
		func HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
			payload, _ := io.ReadAll(r.Body)
			signature := r.Header.Get("Stripe-Signature")

			// Verify webhook signature
			valid, err := paymentClient.VerifyWebhookSignature("stripe", signature, payload)
			if err != nil || !valid {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// Parse webhook event
			event, err := paymentClient.ParseWebhookEvent("stripe", payload)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Handle different event types
			switch event.Type {
			case "payment_intent.succeeded":
				// Payment successful
				// reference := event.Reference
				// UpdateBookingStatus(reference, "paid")

			case "payment_intent.payment_failed":
				// Payment failed
				// reference := event.Reference
				// NotifyGuestPaymentFailed(reference)
			}

			w.WriteHeader(http.StatusOK)
		}
	*/
}

/*
=============================================================================
HELPER UTILITIES
=============================================================================
*/

// Example 11: Using helper functions
func ExampleHelpers() {
	/*
		// Convert amount to minor units
		amount := payment.ToMinorUnits(1000.50, payment.NGN) // 100050 kobo

		// Convert back to major units
		naira := payment.FromMinorUnits(100050, payment.NGN) // 1000.50

		// Format for display
		formatted := payment.FormatAmount(100050, payment.NGN) // "NGN 1000.50"

		// Get currency symbol
		symbol := payment.GetCurrencySymbol(payment.NGN) // "₦"

		// Validate reference
		err := payment.ValidateReference("BKG-12345") // nil if valid
	*/
}

/*
=============================================================================
CURRENCY ROUTING (Automatic)
=============================================================================

The platform automatically routes requests to the correct provider:

NGN (Nigerian Naira) → Paystack
USD (US Dollar) → Stripe
GHS (Ghanaian Cedi) → Stripe

You don't need to worry about which provider to use - just specify the currency!
*/
