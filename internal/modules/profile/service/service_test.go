package service_test

import (
	"context"
	"testing"
	"time"

	"hauslet/internal/modules/profile/domain"
	"hauslet/internal/modules/profile/repository"
	"hauslet/internal/modules/profile/service"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newMockService(t *testing.T) (service.ProfileService, sqlmock.Sqlmock, func()) {
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

	repo := repository.NewProfileRepository(gdb)
	svc := service.NewProfileService(repo, nil)

	cleanup := func() { sqlDB.Close() }
	return svc, mock, cleanup
}

func TestServiceCreateProfileValidations(t *testing.T) {
	t.Parallel()

	svc, _, cleanup := newMockService(t)
	defer cleanup()

	if _, err := svc.CreateProfile(context.Background(), domain.Profile{}); err != domain.ErrInvalidUserID {
		t.Fatalf("expected ErrInvalidUserID, got %v", err)
	}

	if _, err := svc.CreateProfile(context.Background(), domain.Profile{
		UserID:       uuid.New().String(),
		PhoneNumbers: []string{"1", "2", "3"},
	}); err != domain.ErrMaxPhoneNumbersReached {
		t.Fatalf("expected ErrMaxPhoneNumbersReached, got %v", err)
	}
}

func TestServiceCreateProfileDefaultsAndPersists(t *testing.T) {
	t.Parallel()

	svc, mock, cleanup := newMockService(t)
	defer cleanup()

	userID := uuid.New().String()

	// ProfileExists check -> false
	mock.ExpectQuery(`SELECT count\(\*\) FROM "profiles"`).
		WithArgs(uuid.MustParse(userID)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// Insert
	mock.ExpectQuery(`INSERT INTO "profiles"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))

	profile := domain.Profile{
		UserID: userID,
		Rating: 5,
		Badges: []domain.Badge{domain.BadgeSuperHost},
	}

	created, err := svc.CreateProfile(context.Background(), profile)
	if err != nil {
		t.Fatalf("CreateProfile returned error: %v", err)
	}
	if len(created.UserTypes) == 0 || created.UserTypes[0] != domain.Guest {
		t.Fatalf("expected default user type guest, got %v", created.UserTypes)
	}
	if created.TrustScore == 0 {
		t.Fatalf("expected trust score to be calculated")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestServiceUpdateProfilePreservesIDAndRecalculatesTrust(t *testing.T) {
	t.Parallel()

	svc, mock, cleanup := newMockService(t)
	defer cleanup()

	userID := uuid.New().String()
	profileID := uuid.New()
	now := time.Now()

	// ensureProfile lookup
	mock.ExpectQuery(`SELECT .* FROM "profiles" WHERE user_id = \$1`).
		WithArgs(uuid.MustParse(userID), 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "user_types", "full_name", "phone_numbers", "rating", "reviews_count", "badges", "created_at", "updated_at",
		}).AddRow(profileID, uuid.MustParse(userID), "{guest}", "Name", pqStringArray(`{}`), 4.5, 10, pqStringArray(`{}`), now, now))
	mock.ExpectQuery(`SELECT \* FROM "travel_companions" WHERE "travel_companions"\."profile_id" = \$1`).
		WithArgs(profileID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "profile_id", "name", "age_group", "gender", "phone", "relationship", "photo_url", "created_at", "updated_at",
		}))

	// update call
	mock.ExpectExec(`UPDATE "profiles"`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	updated, err := svc.UpdateProfile(context.Background(), domain.Profile{
		UserID:       userID,
		Rating:       4.0,
		Badges:       []domain.Badge{domain.BadgeSuperHost},
		Skills:       []string{"go"},
		ReviewsCount: 5,
	})
	if err != nil {
		t.Fatalf("UpdateProfile returned error: %v", err)
	}
	if updated.ID != profileID {
		t.Fatalf("expected profile ID preserved, got %v", updated.ID)
	}
	if updated.TrustScore == 0 {
		t.Fatalf("expected trust score recalculated")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestServicePatchProfileUpdatesTrustScore(t *testing.T) {
	t.Parallel()

	svc, mock, cleanup := newMockService(t)
	defer cleanup()

	id := uuid.New().String()
	userID := uuid.New()
	now := time.Now()

	mock.ExpectExec(`UPDATE "profiles"`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery(`SELECT .* FROM "profiles" WHERE id = \$1`).
		WithArgs(uuid.MustParse(id), 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "user_types", "full_name", "phone_numbers", "rating", "reviews_count", "badges", "created_at", "updated_at",
		}).AddRow(uuid.MustParse(id), userID, "{guest}", "Name", pqStringArray(`{}`), 5.0, 1, pqStringArray(`{}`), now, now))
	mock.ExpectQuery(`SELECT \* FROM "travel_companions" WHERE "travel_companions"\."profile_id" = \$1`).
		WithArgs(uuid.MustParse(id)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "profile_id", "name", "age_group", "gender", "phone", "relationship", "photo_url", "created_at", "updated_at",
		}))

	mock.ExpectExec(`UPDATE "profiles"`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	updated, err := svc.PatchProfile(context.Background(), id, map[string]interface{}{"bio": "hi"})
	if err != nil {
		t.Fatalf("PatchProfile returned error: %v", err)
	}
	if updated.TrustScore == 0 {
		t.Fatalf("expected trust score recalculated")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// pqStringArray helps keep sqlmock inputs concise.
func pqStringArray(val string) interface{} { return val }
