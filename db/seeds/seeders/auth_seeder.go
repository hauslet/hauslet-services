package seeders

import (
	"fmt"
	"time"

	"hauslet/db/seeds/data"
	"hauslet/db/seeds/utils"
	authSchema "hauslet/internal/modules/auth/repository/schema"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
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
	var existing authSchema.User
	if err := ctx.DB.Where("primary_email = ?", email).First(&existing).Error; err == nil {
		return nil // User already exists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := authSchema.User{
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

	// Create password identity
	passwordHash := string(hashedPassword)
	identity := authSchema.UserIdentity{
		ID:            uuid.New(),
		UserID:        user.ID,
		Provider:      "password",
		ProviderID:    email,
		Email:         email,
		EmailVerified: true,
		PasswordHash:  &passwordHash,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	return ctx.DB.Create(&identity).Error
}

// createTestUser creates a test user with known credentials
func createTestUser(ctx *SeedContext, email, name string, role authSchema.UserRole, password string) error {
	// Check if user already exists
	var existing authSchema.User
	if err := ctx.DB.Where("primary_email = ?", email).First(&existing).Error; err == nil {
		return nil // User already exists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	lastLogin := utils.RandomPastDate(30)
	user := authSchema.User{
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

	// Create password identity
	passwordHashStr := string(hashedPassword)
	identity := authSchema.UserIdentity{
		ID:            uuid.New(),
		UserID:        user.ID,
		Provider:      "password",
		ProviderID:    email,
		Email:         email,
		EmailVerified: true,
		PasswordHash:  &passwordHashStr,
		LastUsedAt:    user.LastLoginAt,
		CreatedAt:     user.CreatedAt,
		UpdatedAt:     time.Now(),
	}

	return ctx.DB.Create(&identity).Error
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

	lastLogin := utils.RandomPastDate(60)
	user := authSchema.User{
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

	// Create Google OAuth identity
	googleIdentity := authSchema.UserIdentity{
		ID:            uuid.New(),
		UserID:        user.ID,
		Provider:      "google",
		ProviderID:    fmt.Sprintf("google_%s", utils.RandomString(20)),
		Email:         email,
		EmailVerified: true,
		LastUsedAt:    user.LastLoginAt,
		CreatedAt:     user.CreatedAt,
		UpdatedAt:     time.Now(),
	}

	return ctx.DB.Create(&googleIdentity).Error
}
