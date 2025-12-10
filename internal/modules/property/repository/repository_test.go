package repository_test

import (
	"context"
	"database/sql/driver"
	"testing"
	"time"

	"hauslet/internal/modules/property/repository"
	"hauslet/internal/modules/property/repository/schema"

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

func newMockRepo(t *testing.T) (repository.Repository, sqlmock.Sqlmock, func()) {
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
	cleanup := func() { sqlDB.Close() }
	return repo, mock, cleanup
}

func TestPropertyExists(t *testing.T) {
	t.Parallel()

	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	ctx := context.Background()
	id := uuid.New()

	mock.ExpectQuery(`SELECT count\(\*\) FROM "properties"`).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	exists, err := repo.PropertyExists(ctx, id)
	if err != nil {
		t.Fatalf("PropertyExists returned error: %v", err)
	}
	if !exists {
		t.Fatalf("PropertyExists returned false, want true")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}

	mock.ExpectQuery(`SELECT count\(\*\) FROM "properties"`).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	exists, err = repo.PropertyExists(ctx, id)
	if err != nil {
		t.Fatalf("PropertyExists returned error: %v", err)
	}
	if exists {
		t.Fatalf("PropertyExists returned true, want false")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreateProperty(t *testing.T) {
	t.Parallel()

	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	prop := &schema.Property{
		ID:                uuid.New(),
		PublicID:          "H1234AB",
		Address:           "123 Ikoyi Road",
		City:              "Lagos",
		State:             "Lagos",
		Country:           schema.CountryNG,
		PropertyClass:     schema.ClassResidential,
		PropertyType:      schema.TypeApartment,
		FurnishingType:    schema.Furnished,
		PropertyCondition: schema.ConditionNew,
		OwnerID:           uuid.New(),
		Units:             1,
		Amenities:         []string{"wifi"},
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	mock.ExpectQuery(`INSERT INTO "properties"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(prop.ID))

	if err := repo.CreateProperty(ctx, prop); err != nil {
		t.Fatalf("CreateProperty returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSoftDeletePropertyNotFound(t *testing.T) {
	t.Parallel()

	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	ctx := context.Background()
	id := uuid.New()

	mock.ExpectExec(`DELETE FROM "properties"`).
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := repo.SoftDeleteProperty(ctx, id); err == nil {
		t.Fatalf("SoftDeleteProperty expected error for missing record")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreateListingValidationAndSuccess(t *testing.T) {
	t.Parallel()

	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	ctx := context.Background()
	propertyID := uuid.New()

	// Property missing should fail
	mock.ExpectQuery(`SELECT count\(\*\) FROM "properties"`).
		WithArgs(propertyID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	err := repo.CreateListing(ctx, &schema.Listing{
		ID:          uuid.New(),
		PropertyID:  propertyID,
		OwnerID:     uuid.New(),
		OwnerType:   schema.OwnerLandlord,
		Slug:        "slug",
		Title:       "title",
		Currency:    schema.CurrencyNGN,
		ListingType: schema.ListingRent,
		Status:      schema.StatusDraft,
	})
	if err == nil {
		t.Fatalf("CreateListing expected error when property is missing")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}

	// Duplicate listing should fail
	mock.ExpectQuery(`SELECT count\(\*\) FROM "properties"`).
		WithArgs(propertyID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT count\(\*\) FROM "listings" WHERE property_id`).
		WithArgs(propertyID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	err = repo.CreateListing(ctx, &schema.Listing{
		ID:          uuid.New(),
		PropertyID:  propertyID,
		OwnerID:     uuid.New(),
		OwnerType:   schema.OwnerLandlord,
		Slug:        "slug2",
		Title:       "title",
		Currency:    schema.CurrencyNGN,
		ListingType: schema.ListingRent,
		Status:      schema.StatusDraft,
	})
	if err == nil {
		t.Fatalf("CreateListing expected error when listing already exists for property")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}

	// Happy path
	listingID := uuid.New()
	mock.ExpectQuery(`SELECT count\(\*\) FROM "properties"`).
		WithArgs(propertyID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT count\(\*\) FROM "listings" WHERE property_id`).
		WithArgs(propertyID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`INSERT INTO "listings"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(listingID))

	err = repo.CreateListing(ctx, &schema.Listing{
		ID:          listingID,
		PropertyID:  propertyID,
		OwnerID:     uuid.New(),
		OwnerType:   schema.OwnerLandlord,
		Slug:        "slug3",
		Title:       "title",
		Currency:    schema.CurrencyNGN,
		ListingType: schema.ListingRent,
		Status:      schema.StatusDraft,
	})
	if err != nil {
		t.Fatalf("CreateListing returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
