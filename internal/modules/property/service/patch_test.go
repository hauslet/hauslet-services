package service_test

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestServicePatchShortletDetails(t *testing.T) {
	t.Parallel()

	svc, mock, cleanup := newMockService(t)
	defer cleanup()

	ctx := context.Background()
	listingID := uuid.New()
	ownerID := uuid.New()
	propertyID := uuid.New()

	updates := map[string]any{
		"nightly_rate": 15000.0,
	}

	// Expect PatchShortletDetails repo call
	// Note: Repo implementation uses Updates, so we expect UPDATE query
	// The service calls repo.PatchShortletDetails
	// The repo implementation:
	// 1. First(&listing)
	// 2. Unmarshal updates
	// 3. Save(&listing)

	mock.ExpectBegin()

	// Mock fetching the listing
	mock.ExpectQuery(`SELECT .* FROM "listings" WHERE id = \$1.*`).
		WithArgs(listingID, 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "property_id", "owner_id", "owner_type", "shortlet_details", "created_at", "updated_at",
		}).AddRow(
			listingID, propertyID, ownerID, "landlord", []byte(`{"nightly_rate": 10000}`), time.Now(), time.Now(),
		))

	// Mock the update
	// The exact query depends on how GORM updates.
	// Since repo uses 'tx.Save(&listing)', it updates ALL columns.
	// We matched this behavior in previous sessions.
	// But wait, the repo implementation was CHANGED to use `Updates` or similar?
	// Let's check listing_repo.go implementation of PatchShortletDetails again.
	// "Implemented... using 'Read-Patch-Write'... tx.Save(&listing)... Optimized... to use tx.Model(...).Select(...).Updates(&listing)"

	// So it selects specific column "ShortletDetails".
	mock.ExpectExec(`UPDATE "listings" SET "shortlet_details"=\$1,"updated_at"=\$2 WHERE "listings"."deleted_at" IS NULL AND "id" = \$3`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), listingID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := svc.PatchShortletDetails(ctx, listingID, updates)
	if err != nil {
		t.Fatalf("PatchShortletDetails returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestServicePatchListingSafe(t *testing.T) {
	// Test that PatchListing logic in Composite Service (UpdateListingWithProperty) uses PatchListing
	// This is indirectly tested by UpdateListingWithProperty tests if they existed.
	// Here we just test the direct PatchListing service method.
	t.Parallel()

	svc, mock, cleanup := newMockService(t)
	defer cleanup()

	ctx := context.Background()
	listingID := uuid.New()

	updates := map[string]any{
		"title": "Patched Title",
	}

	// Mock ensureListing (used for invalidation? No, PatchListing calls ensureListing internally for invalidation?)
	// Let's check service logic.
	// service.PatchListing calls ensureListing (read), then repo.PatchListing (update), then invalidate.

	// 1. EnsureListing (Read)
	mock.ExpectQuery(`SELECT .*`).
		WithArgs(listingID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "slug", "property_id"}).AddRow(listingID, "slug", uuid.New()))

	// 2. Repo Patch
	// GORM generates: UPDATE ... WHERE id = $3 AND deleted_at IS NULL AND "id" = $4
	mock.ExpectExec(`UPDATE "listings" SET "title"=\$1,"updated_at"=\$2 WHERE id = \$3 AND "listings"."deleted_at" IS NULL AND "id" = \$4`).
		WithArgs("Patched Title", sqlmock.AnyArg(), listingID, listingID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 3. Refetch updated listing
	mock.ExpectQuery(`SELECT .*`).
		WithArgs(listingID, 1). // GetListingByID also uses First which adds LIMIT 1
		WillReturnRows(sqlmock.NewRows([]string{"id", "slug", "property_id"}).AddRow(listingID, "slug", uuid.New()))

	// 3. EnsureListing (Read for cache invalidation - actually ensureListing is called at start, invalidation uses existing data?
	// Check code: existing, err := s.ensureListing... s.invalidateListingCache
	// It relies on 'existing' for slug and propertyID. It doesn't fetch again?
	// Ah, PatchListing implementation:
	// existing, err := s.ensureListing...
	// repo.PatchListing...
	// invalidateListingCache(..., existing.Slug, ...)
	// So only 1 read.

	_, err := svc.PatchListing(ctx, listingID, updates)
	if err != nil {
		t.Fatalf("PatchListing returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
