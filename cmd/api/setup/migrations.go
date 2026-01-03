package setup

import (
	authSchema "hauslet/internal/modules/auth/repository/schema"
	bookingSchema "hauslet/internal/modules/booking/repository/schema"
	businessSchema "hauslet/internal/modules/business/repository/schema"
	calendarSchema "hauslet/internal/modules/calendar/repository/schema"
	financeSchema "hauslet/internal/modules/finance/repository/schema"
	interactionsSchema "hauslet/internal/modules/interactions/repository/schema"
	moderationSchema "hauslet/internal/modules/moderation/repository/schema"
	paymentsSchema "hauslet/internal/modules/payments/repository/schema"
	pricingSchema "hauslet/internal/modules/pricing/repository/schema"
	profileSchema "hauslet/internal/modules/profile/repository/schema"
	promotionsSchema "hauslet/internal/modules/promotions/repository/schema"
	propertySchema "hauslet/internal/modules/property/repository/schema"
	reviewSchema "hauslet/internal/modules/review/repository/schema"
	wishlistSchema "hauslet/internal/modules/wishlist/repository/schema"
	"hauslet/internal/platform/database"
	"log/slog"

	"gorm.io/gorm"
)

// RunMigrations applies the schema migrations for the API.
func RunMigrations(db *gorm.DB, log *slog.Logger) error {
	if err := database.RunMigrations(db, log,
		&authSchema.User{},
		&authSchema.UserIdentity{},
		&profileSchema.Profile{},
		&profileSchema.TravelCompanion{},
		&propertySchema.Property{},
		&propertySchema.Listing{},
		&propertySchema.ListingMedia{},
		&businessSchema.Business{},
		&businessSchema.BusinessMember{},
		&businessSchema.BusinessInvitation{},
		&moderationSchema.Moderation{},
		&paymentsSchema.Payment{},
		&paymentsSchema.Transaction{},
		&paymentsSchema.PaymentMethod{},
		&paymentsSchema.PayoutDetail{},
		&wishlistSchema.Wishlist{},
		&wishlistSchema.WishlistItem{},
		&bookingSchema.Booking{},
		&calendarSchema.CalendarEvent{},
		&calendarSchema.CalendarConfig{},
		&pricingSchema.PricingRule{},
		&pricingSchema.MultiPropertyDiscount{},
		&financeSchema.Wallet{},
		&financeSchema.LedgerEntry{},
		&financeSchema.Transaction{},
		&financeSchema.Disbursement{},
		&financeSchema.Dispute{},
		&financeSchema.ReconciliationReport{},
		&financeSchema.Discrepancy{},
		&reviewSchema.Review{},
		&reviewSchema.ReviewResponse{},
		&reviewSchema.ListingStats{},
		&reviewSchema.HostStats{},
		&promotionsSchema.ListingPromotion{},
		&promotionsSchema.AgentSubscription{},
		&promotionsSchema.UsageTracking{},
		&interactionsSchema.Interaction{},
		&interactionsSchema.InteractionAggregate{},
	); err != nil {
		return err
	}
	log.Info("✅ Database migrations completed")
	return nil
}
