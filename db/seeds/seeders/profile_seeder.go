package seeders

import (
	"fmt"
	"strings"
	"time"

	"hauslet/db/seeds/data"
	"hauslet/db/seeds/utils"
	authSchema "hauslet/internal/modules/auth/repository/schema"
	profileSchema "hauslet/internal/modules/profile/repository/schema"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// SeedProfile seeds user profiles linked to existing users.
func SeedProfile(ctx *SeedContext) error {
	var users []authSchema.User
	if err := ctx.DB.Find(&users).Error; err != nil {
		return fmt.Errorf("failed to fetch users: %w", err)
	}

	if len(users) == 0 {
		return fmt.Errorf("no users found - seed auth module first")
	}

	existingProfiles := make(map[uuid.UUID]bool)
	if ctx.DB.Migrator().HasTable("profiles") {
		var rows []struct {
			UserID uuid.UUID
		}
		if err := ctx.DB.Table("profiles").Select("user_id").Scan(&rows).Error; err != nil {
			return fmt.Errorf("failed to fetch profiles: %w", err)
		}
		for _, row := range rows {
			existingProfiles[row.UserID] = true
		}
	}

	for _, user := range users {
		if existingProfiles[user.ID] {
			continue
		}
		if err := createProfileForUser(ctx, user); err != nil {
			return err
		}
	}

	return nil
}

func createProfileForUser(ctx *SeedContext, user authSchema.User) error {
	location := data.GetRandomLocation()
	userTypes := resolveUserTypes(user.PrimaryEmail)
	phones := buildPhoneNumbers()
	gender := utils.RandomChoice([]profileSchema.Gender{
		profileSchema.GenderMale,
		profileSchema.GenderFemale,
		profileSchema.GenderOther,
		profileSchema.GenderUndisclosed,
	})

	birthDate := maybeBirthDate()
	email := user.PrimaryEmail

	houseNumber, street := splitAddress(location.Address)
	area := location.Area
	city := location.City
	state := location.State
	country := "Nigeria"
	postalCode := location.PostCode

	bio := utils.RandomChoice([]string{
		"Focused on quality homes and transparent listings.",
		"Enjoys helping guests find the right place to stay.",
		"Real estate enthusiast with local market knowledge.",
	})

	profile := profileSchema.Profile{
		ID:        uuid.New(),
		FullName:  user.Name,
		BirthDate: birthDate,
		Gender:    gender,
		Email:     &email,
		UserID:    user.ID,
		UserTypes: pq.StringArray(userTypes),

		IsModerated:  utils.RandomBoolWithProbability(0.7),
		PhoneNumbers: phones,
		Address: profileSchema.Address{
			HouseNumber: houseNumber,
			Street:      street,
			Area:        &area,
			City:        &city,
			State:       &state,
			Country:     &country,
			PostalCode:  &postalCode,
		},
		Bio: &bio,

		CommunityCommitment:        utils.RandomBoolWithProbability(0.3),
		BioVisible:                 utils.RandomBoolWithProbability(0.4),
		AllowPersonalizedOffers:    utils.RandomBoolWithProbability(0.3),
		EnablePerformanceAnalytics: utils.RandomBoolWithProbability(0.2),

		PhoneVerified:     true,
		IDVerified:        true,
		VerificationLevel: "identity",

		CreatedAt: user.CreatedAt,
		UpdatedAt: time.Now(),
	}

	// All seed users are verified
	verificationDate := utils.RandomPastDate(365)
	profile.VerificationDate = &verificationDate

	if err := ctx.DB.Create(&profile).Error; err != nil {
		return fmt.Errorf("failed to create profile for user %s: %w", user.ID, err)
	}

	return nil
}

func resolveUserTypes(email string) []string {
	lower := strings.ToLower(email)
	switch {
	case strings.HasPrefix(lower, "host@"):
		return []string{"host", "guest"}
	case strings.HasPrefix(lower, "guest@"):
		return []string{"guest"}
	case strings.HasPrefix(lower, "agent@"):
		return []string{"agent", "guest"}
	}

	userTypes := []string{"guest"}
	if utils.RandomBoolWithProbability(0.35) {
		userTypes = append(userTypes, "host")
	}
	if utils.RandomBoolWithProbability(0.15) {
		userTypes = append(userTypes, "landlord")
	}
	if utils.RandomBoolWithProbability(0.10) {
		userTypes = append(userTypes, "agent")
	}
	if utils.RandomBoolWithProbability(0.05) {
		userTypes = append(userTypes, "cohost")
	}

	return uniqueStrings(userTypes)
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			unique = append(unique, value)
		}
	}
	return unique
}

func maybeBirthDate() *time.Time {
	if !utils.RandomBoolWithProbability(0.7) {
		return nil
	}

	age := utils.RandomInt(18, 65)
	offsetDays := utils.RandomInt(0, 364)
	date := time.Now().AddDate(-age, 0, -offsetDays)
	return &date
}

func buildPhoneNumbers() pq.StringArray {
	phones := pq.StringArray{utils.RandomPhoneNG()}
	if utils.RandomBoolWithProbability(0.25) {
		phones = append(phones, utils.RandomPhoneNG())
	}
	return phones
}

func splitAddress(address string) (*string, *string) {
	parts := strings.Fields(address)
	if len(parts) < 2 {
		return nil, strPtr(address)
	}
	house := parts[0]
	street := strings.Join(parts[1:], " ")
	return &house, &street
}

func strPtr(value string) *string {
	return &value
}
