package service_test

import (
	"context"
	"database/sql/driver"
	"io"
	"log/slog"
	"testing"
	"time"

	promotionsDomain "hauslet/internal/modules/promotions/domain"
	promotionsService "hauslet/internal/modules/promotions/service"
	"hauslet/internal/modules/property/domain"
	"hauslet/internal/modules/property/repository"
	"hauslet/internal/modules/property/service"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func anyArgs(n int) []driver.Value {
	args := make([]driver.Value, n)
	for i := range args {
		args[i] = sqlmock.AnyArg()
	}
	return args
}

func newMockService(t *testing.T) (service.PropertyService, sqlmock.Sqlmock, func()) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	dialector := postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	})

	gdb, err := gorm.Open(dialector, &gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("failed to open gorm DB: %v", err)
	}

	repo := repository.NewPropertyRepository(gdb)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := service.NewPropertyService(repo, nil, nil, nil, nil, "", nil, nil, nil, logger, nil, nil, &mockSubscriptionService{}, nil)

	cleanup := func() { sqlDB.Close() }
	return svc, mock, cleanup
}

type mockSubscriptionService struct{}

func (m *mockSubscriptionService) CanAddListing(ctx context.Context, userID uuid.UUID) (bool, error) {
	return true, nil
}
func (m *mockSubscriptionService) CanPromoteListing(ctx context.Context, userID uuid.UUID) (bool, error) {
	return true, nil
}
func (m *mockSubscriptionService) GetSubscription(ctx context.Context, userID uuid.UUID) (*promotionsDomain.AgentSubscription, error) {
	return nil, nil
}
func (m *mockSubscriptionService) CanAddPhotos(ctx context.Context, ownerID uuid.UUID, listingID uuid.UUID, photoCount int) (bool, error) {
	return true, nil
}
func (m *mockSubscriptionService) CanCreateOpenHouse(ctx context.Context, userID uuid.UUID) (bool, int, error) {
	return true, 1, nil
}
func (m *mockSubscriptionService) CanCreatePrivateShowing(ctx context.Context, userID uuid.UUID) (bool, int, error) {
	return true, 1, nil
}
func (m *mockSubscriptionService) CanUseFeature(ctx context.Context, userID uuid.UUID, featureKey string) (bool, error) {
	return true, nil
}
func (m *mockSubscriptionService) CanUseIncludedPromotion(ctx context.Context, userID uuid.UUID, promoType promotionsDomain.PromotionType) (bool, error) {
	return true, nil
}
func (m *mockSubscriptionService) CancelSubscription(ctx context.Context, userID uuid.UUID) error {
	return nil
}
func (m *mockSubscriptionService) CreateSubscription(ctx context.Context, input promotionsService.CreateSubscriptionInput) (*promotionsService.CreateSubscriptionResult, error) {
	return nil, nil
}
func (m *mockSubscriptionService) DowngradeSubscription(ctx context.Context, userID uuid.UUID, planType promotionsDomain.PlanType) error {
	return nil
}
func (m *mockSubscriptionService) GetFeatureLimit(ctx context.Context, userID uuid.UUID, featureKey string) (int, error) {
	return 100, nil
}
func (m *mockSubscriptionService) GetUserSubscription(ctx context.Context, userID uuid.UUID) (*promotionsDomain.AgentSubscription, error) {
	return nil, nil
}
func (m *mockSubscriptionService) HandlePaymentSuccess(ctx context.Context, userID uuid.UUID) error {
	return nil
}
func (m *mockSubscriptionService) ProcessBilling(ctx context.Context) error {
	return nil
}
func (m *mockSubscriptionService) RenewSubscription(ctx context.Context, userID uuid.UUID, subscriptionID uuid.UUID) error {
	return nil
}
func (m *mockSubscriptionService) UpgradeSubscription(ctx context.Context, userID uuid.UUID, planType promotionsDomain.PlanType) error {
	return nil
}
func (m *mockSubscriptionService) UseIncludedPromotion(ctx context.Context, userID uuid.UUID, promoType promotionsDomain.PromotionType) error {
	return nil
}
func (m *mockSubscriptionService) UseOpenHouse(ctx context.Context, userID uuid.UUID) error {
	return nil
}
func (m *mockSubscriptionService) UsePrivateShowing(ctx context.Context, userID uuid.UUID) error {
	return nil
}

func TestServiceCreatePropertyValidation(t *testing.T) {
	t.Parallel()

	svc, mock, cleanup := newMockService(t)
	defer cleanup()

	_, err := svc.CreateProperty(context.Background(), domain.Property{})
	if err != domain.ErrInvalidOwnerID {
		t.Fatalf("CreateProperty expected ErrInvalidOwnerID, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestServiceCreatePropertySuccess(t *testing.T) {
	t.Parallel()

	svc, mock, cleanup := newMockService(t)
	defer cleanup()

	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	prop := domain.Property{
		ID:                uuid.New(),
		PublicID:          "HABCDEFG",
		OwnerID:           uuid.New(),
		Address:           "123 Road",
		City:              "Lagos",
		State:             "Lagos",
		Country:           domain.CountryNG,
		PropertyClass:     domain.ClassResidential,
		PropertyType:      domain.TypeApartment,
		FurnishingType:    domain.Furnished,
		PropertyCondition: domain.ConditionNew,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	mock.ExpectQuery(`INSERT INTO "properties"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(prop.ID))

	created, err := svc.CreateProperty(context.Background(), prop)
	if err != nil {
		t.Fatalf("CreateProperty returned error: %v", err)
	}
	if created == nil {
		t.Fatalf("CreateProperty returned nil property")
	}
	if created.Units != 1 {
		t.Fatalf("expected Units default to 1, got %d", created.Units)
	}
	if created.Amenities == nil {
		t.Fatalf("expected Amenities to be initialized, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestServiceUpdatePropertyPreservesCreatedAt(t *testing.T) {
	t.Parallel()

	svc, mock, cleanup := newMockService(t)
	defer cleanup()

	ctx := context.Background()
	id := uuid.New()
	ownerID := uuid.New()
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`SELECT .* FROM "properties" WHERE id = \$1`).
		WithArgs(id, 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "public_id", "owner_id", "address", "city", "state", "country", "property_class", "property_type", "furnishing_type", "property_condition", "units", "square_meters", "created_at", "updated_at",
		}).AddRow(
			id, "H1234AB", ownerID, "123 Road", "Lagos", "Lagos", "NG", "residential", "apartment", "furnished", "new", 1, 100.0, createdAt, updatedAt,
		))

	mock.ExpectExec(`UPDATE "properties"`).
		WithArgs(anyArgs(26)...).
		WillReturnResult(sqlmock.NewResult(0, 1))

	result, err := svc.UpdateProperty(ctx, domain.Property{
		ID:                id,
		OwnerID:           ownerID,
		Address:           "New Address",
		City:              "Lagos",
		State:             "Lagos",
		Country:           domain.CountryNG,
		PropertyClass:     domain.ClassResidential,
		PropertyType:      domain.TypeApartment,
		FurnishingType:    domain.Furnished,
		PropertyCondition: domain.ConditionNew,
		CreatedAt:         time.Now(), // should be overwritten
	})
	if err != nil {
		t.Fatalf("UpdateProperty returned error: %v", err)
	}

	if !result.CreatedAt.Equal(createdAt) {
		t.Fatalf("expected CreatedAt preserved as %v, got %v", createdAt, result.CreatedAt)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestServiceCreateListingValidationsAndSuccess(t *testing.T) {
	t.Parallel()

	svc, mock, cleanup := newMockService(t)
	defer cleanup()

	ctx := context.Background()
	ownerID := uuid.New()
	propertyID := uuid.New()

	// Invalid property ID
	_, err := svc.CreateListing(ctx, domain.Listing{
		OwnerID:   ownerID,
		OwnerType: domain.OwnerLandlord,
	})
	if err != domain.ErrInvalidPropertyID {
		t.Fatalf("expected ErrInvalidPropertyID, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}

	// Property missing
	mock.ExpectQuery(`SELECT count\(\*\) FROM "properties"`).
		WithArgs(propertyID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	_, err = svc.CreateListing(ctx, domain.Listing{
		PropertyID:  propertyID,
		OwnerID:     ownerID,
		OwnerType:   domain.OwnerLandlord,
		Slug:        "slug",
		Title:       "title",
		Currency:    domain.CurrencyNGN,
		ListingType: domain.ListingRent,
		Status:      domain.StatusDraft,
	})
	if err != domain.ErrPropertyNotFound {
		t.Fatalf("expected ErrPropertyNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}

	// Happy path (note CreateListing calls PropertyExists internally again)
	mock.ExpectQuery(`SELECT count\(\*\) FROM "properties"`).
		WithArgs(propertyID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT count\(\*\) FROM "properties"`).
		WithArgs(propertyID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT count\(\*\) FROM "listings" WHERE property_id`).
		WithArgs(propertyID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`INSERT INTO "listings"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))

	created, err := svc.CreateListing(ctx, domain.Listing{
		PropertyID:  propertyID,
		OwnerID:     ownerID,
		OwnerType:   domain.OwnerLandlord,
		Title:       "title",
		Currency:    domain.CurrencyNGN,
		ListingType: domain.ListingRent,
		Status:      domain.StatusDraft,
	})
	if err != nil {
		t.Fatalf("CreateListing returned error: %v", err)
	}
	if created == nil {
		t.Fatalf("CreateListing returned nil listing")
	}
	if created.ID == uuid.Nil {
		t.Fatalf("expected listing ID to be set")
	}
	if created.Slug == "" {
		t.Fatalf("expected slug to be generated when blank")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestServiceUpdateListingPreservesSlug(t *testing.T) {
	t.Parallel()

	svc, mock, cleanup := newMockService(t)
	defer cleanup()

	ctx := context.Background()
	listingID := uuid.New()
	propertyID := uuid.New()
	ownerID := uuid.New()
	existingSlug := "existing-slug"
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`SELECT .* FROM "listings" WHERE id = \$1`).
		WithArgs(listingID, 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "property_id", "owner_id", "owner_type", "slug", "title", "description", "currency", "listing_type", "status", "published", "latest_review_status", "view_count", "created_at", "updated_at",
		}).AddRow(
			listingID, propertyID, ownerID, "landlord", existingSlug, "Old title", "desc", "NGN", "rent", "draft", false, "pending", 0, now, now,
		))

	mock.ExpectExec(`UPDATE "listings"`).
		WithArgs(anyArgs(30)...).
		WillReturnResult(sqlmock.NewResult(0, 1))

	updated, err := svc.UpdateListing(ctx, domain.Listing{
		ID:          listingID,
		PropertyID:  propertyID,
		OwnerID:     ownerID,
		OwnerType:   domain.OwnerLandlord,
		Title:       "New title",
		Currency:    domain.CurrencyNGN,
		ListingType: domain.ListingRent,
		Status:      domain.StatusDraft,
		Slug:        "", // should preserve existing slug
	})
	if err != nil {
		t.Fatalf("UpdateListing returned error: %v", err)
	}
	if updated.Slug != existingSlug {
		t.Fatalf("expected slug to remain %q, got %q", existingSlug, updated.Slug)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
