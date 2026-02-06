package service

import (
	"context"
	"testing"
	"time"

	"hauslet/config"
	"hauslet/internal/modules/booking/domain"
	"hauslet/internal/modules/booking/repository"
	"hauslet/internal/modules/booking/repository/schema"
	calendardomain "hauslet/internal/modules/calendar/domain"
	pricingdomain "hauslet/internal/modules/pricing/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// -- Mocks --

type MockBookingRepo struct {
	mock.Mock
}

func (m *MockBookingRepo) GetBookingByID(ctx context.Context, id uuid.UUID) (*schema.Booking, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.Booking), args.Error(1)
}

func (m *MockBookingRepo) UpdateBooking(ctx context.Context, booking *schema.Booking) error {
	args := m.Called(ctx, booking)
	return args.Error(0)
}

// Stubs for other repo methods to satisfy interface
func (m *MockBookingRepo) CreateBooking(_ context.Context, _ *schema.Booking) error { return nil }
func (m *MockBookingRepo) GetBookingsByIDs(_ context.Context, _ []uuid.UUID) ([]*schema.Booking, error) {
	return nil, nil
}
func (m *MockBookingRepo) GetBookingByReference(_ context.Context, _ string) (*schema.Booking, error) {
	return nil, nil
}
func (m *MockBookingRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ schema.BookingStatus, _, _ *time.Time) error {
	return nil
}
func (m *MockBookingRepo) ListBookingsForGuest(_ context.Context, _ uuid.UUID, _, _ int) ([]*schema.Booking, error) {
	return nil, nil
}
func (m *MockBookingRepo) ListBookingsForListing(_ context.Context, _ uuid.UUID, _, _ int) ([]*schema.Booking, error) {
	return nil, nil
}
func (m *MockBookingRepo) ListBookingsForHost(_ context.Context, _ uuid.UUID, _ *schema.BookingStatus, _, _ int) ([]*schema.Booking, error) {
	return nil, nil
}
func (m *MockBookingRepo) FindExpiredHolds(_ context.Context, _ time.Time) ([]*schema.Booking, error) {
	return nil, nil
}
func (m *MockBookingRepo) FindBookingsReadyForPayout(_ context.Context, _ string, _ int, _ int) ([]*repository.BookingPayoutInfo, error) {
	return nil, nil
}
func (m *MockBookingRepo) FindBookingsReadyForCompletion(_ context.Context, _ string, _ int, _ int) ([]*schema.Booking, error) {
	return nil, nil
}
func (m *MockBookingRepo) FindCompletedBookingsInRange(_ context.Context, _, _ time.Time, _ int) ([]*schema.Booking, error) {
	return nil, nil
}
func (m *MockBookingRepo) FindBookingsPendingCheckIn(_ context.Context, _ time.Time, _ int) ([]*schema.Booking, error) {
	return nil, nil
}
func (m *MockBookingRepo) FindBookingsPendingCheckOut(_ context.Context, _ time.Time, _ int) ([]*schema.Booking, error) {
	return nil, nil
}

type MockPricingService struct {
	mock.Mock
}

func (m *MockPricingService) CalculateRefund(ctx context.Context, input pricingdomain.RefundCalculationInput) (*pricingdomain.RefundBreakdown, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pricingdomain.RefundBreakdown), args.Error(1)
}

// Stubs
func (m *MockPricingService) CalculatePrice(_ context.Context, _ uuid.UUID, _, _ time.Time, _ int) (*pricingdomain.PriceBreakdown, error) {
	return nil, nil
}
func (m *MockPricingService) GetBasePrice(_ context.Context, _ uuid.UUID) (float64, string, error) {
	return 0, "", nil
}

type MockFinanceHooks struct {
	mock.Mock
}

func (m *MockFinanceHooks) OnBookingCancelledWithFunds(ctx context.Context, bookingID, hostID uuid.UUID, hostAmount, platformAmount int64, currency string) error {
	args := m.Called(ctx, bookingID, hostID, hostAmount, platformAmount, currency)
	return args.Error(0)
}

// Stubs
func (m *MockFinanceHooks) OnPaymentSucceeded(_ context.Context, _, _ uuid.UUID, _ int64, _ string) error {
	return nil
}
func (m *MockFinanceHooks) OnRefundProcessed(_ context.Context, _, _ uuid.UUID, _ int64, _ string) error {
	return nil
}
func (m *MockFinanceHooks) OnBookingCompleted(_ context.Context, _, _ uuid.UUID) error {
	return nil
}
func (m *MockFinanceHooks) DeductPenalty(_ context.Context, _ uuid.UUID, _ int64, _ uuid.UUID, _ string) error {
	return nil
}

type MockPaymentGateway struct {
	mock.Mock
}

func (m *MockPaymentGateway) RefundPayment(ctx context.Context, input RefundPaymentInput) (*RefundResult, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RefundResult), args.Error(1)
}

// Stubs
func (m *MockPaymentGateway) InitiatePayment(_ context.Context, _ PaymentInput) (*PaymentResult, error) {
	return nil, nil
}
func (m *MockPaymentGateway) VerifyPayment(_ context.Context, _ uuid.UUID) (*PaymentStatus, error) {
	return nil, nil
}
func (m *MockPaymentGateway) GetDefaultPaymentMethodID(_ context.Context, _ uuid.UUID) (*uuid.UUID, error) {
	return nil, nil
}

type MockCalendarGateway struct {
	mock.Mock
}

func (m *MockCalendarGateway) CancelEvent(ctx context.Context, eventID uuid.UUID, requestorID uuid.UUID) error {
	args := m.Called(ctx, eventID, requestorID)
	return args.Error(0)
}

// Stubs
func (m *MockCalendarGateway) CheckAvailability(_ context.Context, _ uuid.UUID, _, _ time.Time) (*calendardomain.AvailabilityResult, error) {
	return nil, nil
}
func (m *MockCalendarGateway) CreateEvent(_ context.Context, _ *calendardomain.CalendarEvent) (*calendardomain.CalendarEvent, error) {
	return nil, nil
}
func (m *MockCalendarGateway) GetCalendarConfig(_ context.Context, _ uuid.UUID) (*calendardomain.CalendarConfig, error) {
	return nil, nil
}
func (m *MockCalendarGateway) GetEvent(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*calendardomain.CalendarEvent, error) {
	return nil, nil
}
func (m *MockCalendarGateway) UpdateEvent(_ context.Context, _ *calendardomain.CalendarEvent, _ uuid.UUID) (*calendardomain.CalendarEvent, error) {
	return nil, nil
}
func (m *MockCalendarGateway) DeleteEvent(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

type MockListingHooks struct{ mock.Mock }

func (m *MockListingHooks) GetListingOwner(ctx context.Context, listingID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, listingID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}
func (m *MockListingHooks) GetListingConstraints(_ context.Context, _ uuid.UUID) (*ListingConstraints, error) {
	return nil, nil
}

type MockProfileProvider struct{}

func (m *MockProfileProvider) GetUserContact(_ context.Context, _ uuid.UUID) (*ContactInfo, error) {
	return nil, nil
}

// -- Test --

func TestCancelBooking_PartialRefund(t *testing.T) {
	// Setup
	mockRepo := new(MockBookingRepo)
	mockPricing := new(MockPricingService)
	mockFinanceHook := new(MockFinanceHooks)
	mockPayment := new(MockPaymentGateway)
	mockCalendar := new(MockCalendarGateway)
	ctx := context.Background()
	bookingID := uuid.New()
	guestID := uuid.New()
	hostID := uuid.New()
	paymentID := uuid.New()
	listingID := uuid.New()

	mockListing := new(MockListingHooks)
	// Setup listing hooks expectation
	mockListing.On("GetListingConstraints", ctx, mock.Anything).Return(&ListingConstraints{
		RefundPolicy: "moderate",
	}, nil)
	mockListing.On("GetListingOwner", ctx, listingID).Return(hostID, nil)

	// Since notifyBookingCancellation uses notifier/profile provider which are optional or pointers,
	// if notifier is nil, it skips. I'll pass default/custom provider if needed.
	// But `CancelBooking` calls `GetListingOwner` from hook or uses internal?
	// `CancelBooking` gets booking from repo. `Booking` has fields.
	// It doesn't call ListingHooks in `CancelBooking`.
	// It calls `calendar.CancelEvent`.

	svc := &BookingServiceImpl{
		repo:           mockRepo,
		calendar:       mockCalendar,
		pricing:        mockPricing,
		payment:        mockPayment,
		financeHooks:   mockFinanceHook,
		listingHooks:   mockListing,
		platformConfig: config.PlatformYAMLConfig{},
		// other deps nil/ignored
	}

	checkInTime := time.Now().Add(10 * 24 * time.Hour)

	// Initial Booking State
	booking := &domain.Booking{
		ID:              bookingID,
		GuestID:         guestID,
		ListingID:       listingID,
		Status:          domain.BookingStatusConfirmed,
		TotalPrice:      1000.0,
		Currency:        "USD",
		LastPaymentID:   &paymentID,                      // Payment made using this ID
		CreatedAt:       time.Now().Add(-48 * time.Hour), // 2 days ago
		CheckIn:         &checkInTime,
		CalendarEventID: uuid.New(),
	}

	// Schema booking mapping for repo return
	schemaBooking := domain.MapBookingFromDomain(booking)
	// OwnerID is resolved via GetListingOwner in service, not stored in booking directly usually

	// Mock Expectations
	mockRepo.On("GetBookingByID", ctx, bookingID).Return(schemaBooking, nil)

	// Pricing logic: Partial refund
	// refundInput variable is unused in matching below, removing declaration to fix lint
	// but logic uses similar fields

	refundBreakdown := &pricingdomain.RefundBreakdown{
		BookingID:          bookingID,
		OriginalAmount:     1000.0,
		NetRefund:          800.0,
		RefundPercentage:   80.0,
		NonRefundedAmount:  200.0,
		PlatformRetained:   50.0,
		HostRetainedAmount: 150.0,
	}

	mockPricing.On("CalculateRefund", ctx, mock.MatchedBy(func(input pricingdomain.RefundCalculationInput) bool {
		return input.BookingID == bookingID && input.TotalPaid == 1000.0 && input.RefundPolicy == "moderate"
	})).Return(refundBreakdown, nil)

	// Payment Refund
	mockPayment.On("RefundPayment", ctx, mock.MatchedBy(func(input RefundPaymentInput) bool {
		return input.PaymentID == paymentID && *input.Amount == 80000 // 800.0 * 100 minor units
	})).Return(&RefundResult{}, nil)

	// Finance Hook (Split of Non-refunded)
	// Host Amount: 150.0 -> 15000 cents
	// Platform Amount: 50.0 -> 5000 cents
	mockFinanceHook.On("OnBookingCancelledWithFunds", ctx, bookingID, hostID, int64(15000), int64(5000), "USD").Return(nil)

	// Calendar cancellation
	mockCalendar.On("CancelEvent", ctx, booking.CalendarEventID, hostID).Return(nil)

	// Repo Update (expect status cancelled and refund breakdown)
	mockRepo.On("UpdateBooking", ctx, mock.MatchedBy(func(b *schema.Booking) bool {
		return b.Status == schema.BookingStatusCancelled
	})).Return(nil)

	// Run
	reason := "changed plans"
	resBooking, err := svc.CancelBooking(ctx, bookingID, guestID, &reason)

	assert.NoError(t, err)
	assert.Equal(t, domain.BookingStatusCancelled, resBooking.Status)

	if assert.NotNil(t, resBooking.RefundBreakdown) {
		assert.Equal(t, refundBreakdown.NetRefund, resBooking.RefundBreakdown.NetRefund)
		assert.Equal(t, refundBreakdown.HostRetainedAmount, resBooking.RefundBreakdown.HostRetainedAmount)
		assert.Equal(t, refundBreakdown.PlatformRetained, resBooking.RefundBreakdown.PlatformRetained)
		assert.Equal(t, refundBreakdown.RefundPercentage, resBooking.RefundBreakdown.RefundPercentage)
	}

	mockRepo.AssertExpectations(t)
	mockPricing.AssertExpectations(t)
	mockFinanceHook.AssertExpectations(t)
	mockPayment.AssertExpectations(t)
	mockCalendar.AssertExpectations(t)
}
