package domain

import (
	"hauslet/internal/modules/auth/repository/schema"
)

// MapUserFromSchema converts repository schema.User to domain User
// SECURITY: Filters out sensitive fields (DeletedAt, raw Identities with passwords)
func MapUserFromSchema(schemaUser *schema.User) *User {
	if schemaUser == nil {
		return nil
	}

	user := &User{
		ID:                schemaUser.ID,
		Name:              schemaUser.Name,
		PrimaryEmail:      schemaUser.PrimaryEmail,
		Role:              UserRole(schemaUser.Role),
		IsActive:          schemaUser.IsActive,
		DeactivatedAt:     schemaUser.DeactivatedAt,
		DeactivatedReason: schemaUser.DeactivatedReason,
		DeactivatedBy:     schemaUser.DeactivatedBy,
		ReactivateOnLogin: schemaUser.ReactivateOnLogin,
		ReactivatedAt:     schemaUser.ReactivatedAt,
		ReactivatedBy:     schemaUser.ReactivatedBy,
		CreatedAt:         schemaUser.CreatedAt,
		UpdatedAt:         schemaUser.UpdatedAt,
	}

	// Handle optional LastLoginAt
	user.LastLoginAt = schemaUser.LastLoginAt

	// Map identities if present (without sensitive data)
	if len(schemaUser.Identities) > 0 {
		user.Identities = make([]UserIdentity, len(schemaUser.Identities))
		for i, identity := range schemaUser.Identities {
			user.Identities[i] = *MapUserIdentityFromSchema(&identity)
		}
	}

	return user
}

// MapUserToSchema converts domain User to repository schema.User
// SECURITY: Only maps fields that should be written back to DB
func MapUserToSchema(domainUser *User) *schema.User {
	if domainUser == nil {
		return nil
	}

	schemaUser := &schema.User{
		ID:                domainUser.ID,
		Name:              domainUser.Name,
		PrimaryEmail:      domainUser.PrimaryEmail,
		Role:              schema.UserRole(domainUser.Role),
		IsActive:          domainUser.IsActive,
		DeactivatedAt:     domainUser.DeactivatedAt,
		DeactivatedReason: domainUser.DeactivatedReason,
		DeactivatedBy:     domainUser.DeactivatedBy,
		ReactivateOnLogin: domainUser.ReactivateOnLogin,
		ReactivatedAt:     domainUser.ReactivatedAt,
		ReactivatedBy:     domainUser.ReactivatedBy,
		CreatedAt:         domainUser.CreatedAt,
		UpdatedAt:         domainUser.UpdatedAt,
	}

	// Handle optional LastLoginAt
	schemaUser.LastLoginAt = domainUser.LastLoginAt

	return schemaUser
}

// MapUserIdentityFromSchema converts schema.UserIdentity to domain UserIdentity
// SECURITY: Strips PasswordHash, ProviderID, and DeletedAt
func MapUserIdentityFromSchema(schemaIdentity *schema.UserIdentity) *UserIdentity {
	if schemaIdentity == nil {
		return nil
	}

	identity := &UserIdentity{
		ID:            schemaIdentity.ID,
		UserID:        schemaIdentity.UserID,
		Provider:      schemaIdentity.Provider,
		Email:         schemaIdentity.Email,
		EmailVerified: schemaIdentity.EmailVerified,
		CreatedAt:     schemaIdentity.CreatedAt,
	}

	// Handle optional LastUsedAt
	identity.LastUsedAt = schemaIdentity.LastUsedAt

	return identity
}

// MapUserIdentityToSchema converts domain UserIdentity to schema.UserIdentity
// SECURITY: Only maps non-sensitive fields. PasswordHash must be set separately
func MapUserIdentityToSchema(domainIdentity *UserIdentity) *schema.UserIdentity {
	if domainIdentity == nil {
		return nil
	}

	schemaIdentity := &schema.UserIdentity{
		ID:            domainIdentity.ID,
		UserID:        domainIdentity.UserID,
		Provider:      domainIdentity.Provider,
		Email:         domainIdentity.Email,
		EmailVerified: domainIdentity.EmailVerified,
		CreatedAt:     domainIdentity.CreatedAt,
	}

	// Handle optional LastUsedAt
	schemaIdentity.LastUsedAt = domainIdentity.LastUsedAt

	return schemaIdentity
}

// ToPublicUser converts domain User to PublicUser for API responses
// SECURITY: Minimal data exposure - only safe fields
func (u *User) ToPublicUser() *PublicUser {
	if u == nil {
		return nil
	}

	return &PublicUser{
		ID:           u.ID,
		Name:         u.Name,
		PrimaryEmail: u.PrimaryEmail,
		CreatedAt:    u.CreatedAt,
	}
}

// ToAuthenticatedUser adds session context to a User
func (u *User) ToAuthenticatedUser(sessionID string) *AuthenticatedUser {
	if u == nil {
		return nil
	}

	return &AuthenticatedUser{
		User:      *u,
		SessionID: sessionID,
	}
}

// MapUsersFromSchema converts a slice of schema users to domain users
func MapUsersFromSchema(schemaUsers []schema.User) []User {
	if schemaUsers == nil {
		return nil
	}

	users := make([]User, len(schemaUsers))
	for i, schemaUser := range schemaUsers {
		if mapped := MapUserFromSchema(&schemaUser); mapped != nil {
			users[i] = *mapped
		}
	}

	return users
}
