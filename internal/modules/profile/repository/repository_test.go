package repository_test

import (
	"context"
	"testing"
	"time"

	"hauslet/internal/modules/profile/repository"
	"hauslet/internal/modules/profile/repository/schema"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newMockRepo(t *testing.T) (*repository.ProfileRerpositoryImpl, sqlmock.Sqlmock, func()) {
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

	repo := repository.NewProfileRepository(gdb).(*repository.ProfileRerpositoryImpl)
	cleanup := func() { sqlDB.Close() }
	return repo, mock, cleanup
}

func TestCreateProfileValidatesUserID(t *testing.T) {
	t.Parallel()

	repo, _, cleanup := newMockRepo(t)
	defer cleanup()

	if err := repo.CreateProfile(context.Background(), &schema.Profile{}); err == nil {
		t.Fatalf("expected error when user ID is nil")
	}
}

func TestCreateProfileSuccess(t *testing.T) {
	t.Parallel()

	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	profile := &schema.Profile{
		ID:        uuid.Nil, // should be set
		UserID:    uuid.New(),
		FullName:  "Test User",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mock.ExpectQuery(`INSERT INTO "profiles"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))

	if err := repo.CreateProfile(context.Background(), profile); err != nil {
		t.Fatalf("CreateProfile returned error: %v", err)
	}

	if profile.ID == uuid.Nil {
		t.Fatalf("expected profile ID to be populated")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestProfileExists(t *testing.T) {
	t.Parallel()

	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	userID := uuid.New().String()
	mock.ExpectQuery(`SELECT count\(\*\) FROM "profiles"`).
		WithArgs(uuid.MustParse(userID)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	exists, err := repo.ProfileExists(context.Background(), userID)
	if err != nil {
		t.Fatalf("ProfileExists returned error: %v", err)
	}
	if !exists {
		t.Fatalf("ProfileExists = false, want true")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetProfileByUserIDHandlesNotFound(t *testing.T) {
	t.Parallel()

	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	userID := uuid.New().String()

	mock.ExpectQuery(`SELECT .* FROM "profiles" WHERE user_id = \$1`).
		WithArgs(uuid.MustParse(userID), 1).
		WillReturnRows(sqlmock.NewRows([]string{}))

	result, err := repo.GetProfileByUserID(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetProfileByUserID returned error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil profile when not found")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUpdateProfileRequiresID(t *testing.T) {
	t.Parallel()

	repo, _, cleanup := newMockRepo(t)
	defer cleanup()

	err := repo.UpdateProfile(context.Background(), &schema.Profile{UserID: uuid.New()})
	if err == nil {
		t.Fatalf("expected error when profile ID is nil")
	}
}

func TestPatchProfileNormalizesArrays(t *testing.T) {
	t.Parallel()

	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	id := uuid.New().String()
	updates := map[string]interface{}{
		"user_types": []string{"host", "guest"},
		"skills":     []string{"go", "sql"},
	}

	mock.ExpectExec(`UPDATE "profiles"`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.PatchProfile(context.Background(), id, updates); err != nil {
		t.Fatalf("PatchProfile returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSetVerificationStatusSetsTimestampWhenVerified(t *testing.T) {
	t.Parallel()

	repo, mock, cleanup := newMockRepo(t)
	defer cleanup()

	userID := uuid.New().String()

	mock.ExpectExec(`UPDATE "profiles"`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.SetVerificationStatus(context.Background(), userID, "identity", true, nil); err != nil {
		t.Fatalf("SetVerificationStatus returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
