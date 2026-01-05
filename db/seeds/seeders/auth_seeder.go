package seeders

import (
	"errors"
	"fmt"
	"time"

	"hauslet/db/seeds/data"
	"hauslet/db/seeds/utils"
	authSchema "hauslet/internal/modules/auth/repository/schema"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SeedAuth seeds users and user identities
func SeedAuth(ctx *SeedContext) error {
	// Always create admin accounts
	for _, email := range ctx.Config.AdminEmails {
		if err := createAdminUser(ctx, email); err != nil {
			return fmt.Errorf("failed to create admin %s: %w", email, err)
		}
	}

	// Create test accounts with known passwords
	testAccounts := []struct {
		email    string
		name     string
		role     authSchema.UserRole
		password string
	}{
		{"theblckpartyfuto@gmail.com", "Test Host", authSchema.RoleUser, "test123"},
		{"ekz.ikwunna@gmail.com", "Test Guest", authSchema.RoleUser, "test123"},
		{"icekkid14@hauslet.com", "Test Agent", authSchema.RoleUser, "test123"},
	}

	for _, account := range testAccounts {
		if err := createTestUser(ctx, account.email, account.name, account.role, account.password); err != nil {
			return fmt.Errorf("failed to create test account %s: %w", account.email, err)
		}
	}

	// Create regular users
	regularUsersToCreate := ctx.Config.UserCount - len(ctx.Config.AdminEmails) - len(testAccounts)
	if regularUsersToCreate < 0 {
		regularUsersToCreate = 0
	}

	for i := 0; i < regularUsersToCreate; i++ {
		if err := createRandomUser(ctx); err != nil {
			return fmt.Errorf("failed to create random user: %w", err)
		}
	}

	return nil
}

// createAdminUser creates an admin user with password authentication
func createAdminUser(ctx *SeedContext, email string) error {
	// Check if user already exists
	var user authSchema.User
	if err := ctx.DB.Where("primary_email = ?", email).First(&user).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		user = authSchema.User{
			ID:           uuid.New(),
			Name:         "Admin User",
			PrimaryEmail: email,
			Role:         authSchema.RoleAdmin,
			IsActive:     true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := ctx.DB.Create(&user).Error; err != nil {
			return err
		}
	}

	return ensurePasswordIdentity(ctx, user, email, "admin123")
}

// createTestUser creates a test user with known credentials
func createTestUser(ctx *SeedContext, email, name string, role authSchema.UserRole, password string) error {
	// Check if user already exists
	var user authSchema.User
	if err := ctx.DB.Where("primary_email = ?", email).First(&user).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		lastLogin := utils.RandomPastDate(30)
		user = authSchema.User{
			ID:           uuid.New(),
			Name:         name,
			PrimaryEmail: email,
			Role:         role,
			IsActive:     true,
			LastLoginAt:  &lastLogin,
			CreatedAt:    utils.RandomPastDate(180),
			UpdatedAt:    time.Now(),
		}

		if err := ctx.DB.Create(&user).Error; err != nil {
			return err
		}
	}

	return ensurePasswordIdentity(ctx, user, email, password)
}

// createRandomUser creates a random user with OAuth identity
func createRandomUser(ctx *SeedContext) error {
	name := data.GenerateFullName()
	email := utils.RandomEmail(name)

	// Random role (mostly regular users)
	role := authSchema.RoleUser
	if utils.RandomBoolWithProbability(0.05) { // 5% moderators
		role = authSchema.RoleModerator
	} else if utils.RandomBoolWithProbability(0.02) { // 2% staff
		role = authSchema.RoleStaff
	}

	var user authSchema.User
	if err := ctx.DB.Where("primary_email = ?", email).First(&user).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		lastLogin := utils.RandomPastDate(60)
		user = authSchema.User{
			ID:           uuid.New(),
			Name:         name,
			PrimaryEmail: email,
			Role:         role,
			IsActive:     true,
			LastLoginAt:  &lastLogin,
			CreatedAt:    utils.RandomPastDate(365),
			UpdatedAt:    time.Now(),
		}

		if err := ctx.DB.Create(&user).Error; err != nil {
			return err
		}
	}

	return ensureGoogleIdentity(ctx, user, email)
}

func ensurePasswordIdentity(ctx *SeedContext, user authSchema.User, email, password string) error {
	var existing authSchema.UserIdentity
	if err := ctx.DB.Where("user_id = ? AND provider = ?", user.ID, "password").First(&existing).Error; err == nil {
		if existing.PasswordHash != nil {
			return nil
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		passwordHash := string(hashedPassword)
		updates := map[string]any{
			"password_hash":  &passwordHash,
			"email":          email,
			"provider_id":    email,
			"email_verified": true,
		}

		return ctx.DB.Model(&existing).Updates(updates).Error
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	passwordHash := string(hashedPassword)

	identity := authSchema.UserIdentity{
		ID:            uuid.New(),
		UserID:        user.ID,
		Provider:      "password",
		ProviderID:    email,
		Email:         email,
		EmailVerified: true,
		PasswordHash:  &passwordHash,
		LastUsedAt:    user.LastLoginAt,
		CreatedAt:     user.CreatedAt,
		UpdatedAt:     time.Now(),
	}

	return ctx.DB.Create(&identity).Error
}

func ensureGoogleIdentity(ctx *SeedContext, user authSchema.User, email string) error {
	var existing authSchema.UserIdentity
	if err := ctx.DB.Where("user_id = ? AND provider = ?", user.ID, "google").First(&existing).Error; err == nil {
		if existing.Email != "" {
			return nil
		}
		return ctx.DB.Model(&existing).Update("email", email).Error
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	providerID := fmt.Sprintf("google_%s", user.ID.String())
	identity := authSchema.UserIdentity{
		ID:            uuid.New(),
		UserID:        user.ID,
		Provider:      "google",
		ProviderID:    providerID,
		Email:         email,
		EmailVerified: true,
		LastUsedAt:    user.LastLoginAt,
		CreatedAt:     user.CreatedAt,
		UpdatedAt:     time.Now(),
	}

	return ctx.DB.Create(&identity).Error
}
