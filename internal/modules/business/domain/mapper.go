package domain

import (
	"errors"

	"hauslet/internal/modules/business/repository/schema"
	propertySchema "hauslet/internal/modules/property/repository/schema"

	"github.com/lib/pq"
)

// MapBusinessFromSchema converts a repository business to a domain business
func MapBusinessFromSchema(schemaBusiness *schema.Business) *Business {
	if schemaBusiness == nil {
		return nil
	}

	business := &Business{
		ID:                 schemaBusiness.ID,
		Slug:               schemaBusiness.Slug,
		Name:               schemaBusiness.Name,
		DisplayName:        schemaBusiness.DisplayName,
		Description:        schemaBusiness.Description,
		BusinessType:       BusinessType(schemaBusiness.BusinessType),
		RegistrationNumber: schemaBusiness.RegistrationNumber,
		TaxID:              schemaBusiness.TaxID,
		LegalEntityType:    schemaBusiness.LegalEntityType,
		Email:              schemaBusiness.Email,
		PhoneNumbers:       make([]string, len(schemaBusiness.PhoneNumbers)),
		Website:            schemaBusiness.Website,
		Address: Address{
			HouseNumber:    schemaBusiness.Address.HouseNumber,
			Street:         schemaBusiness.Address.Street,
			Area:           schemaBusiness.Address.Area,
			LGA:            schemaBusiness.Address.LGA,
			City:           schemaBusiness.Address.City,
			State:          schemaBusiness.Address.State,
			Country:        schemaBusiness.Address.Country,
			PostalCode:     schemaBusiness.Address.PostalCode,
			District:       schemaBusiness.Address.District,
			DigitalAddress: schemaBusiness.Address.DigitalAddress,
		},
		LogoURL:       schemaBusiness.LogoURL,
		CoverImageURL: schemaBusiness.CoverImageURL,
		BrandColor:    schemaBusiness.BrandColor,
		IsVerified:    schemaBusiness.IsVerified,
		VerifiedAt:    schemaBusiness.VerifiedAt,
		IsActive:      schemaBusiness.IsActive,
		BillingEmail:  schemaBusiness.BillingEmail,
		CreatedBy:     schemaBusiness.CreatedBy,
		CreatedAt:     schemaBusiness.CreatedAt,
		UpdatedAt:     schemaBusiness.UpdatedAt,
	}

	// Copy phone numbers
	copy(business.PhoneNumbers, schemaBusiness.PhoneNumbers)

	// Map location if present
	if schemaBusiness.Location != nil {
		business.Location = &Location{
			Lat:  schemaBusiness.Location.Lat,
			Lng:  schemaBusiness.Location.Lng,
			SRID: schemaBusiness.Location.SRID,
		}
	}

	// Map DeletedAt
	if schemaBusiness.DeletedAt.Valid {
		business.DeletedAt = &schemaBusiness.DeletedAt.Time
	}

	return business
}

// MapBusinessToSchema converts a domain business to a repository business
func MapBusinessToSchema(domainBusiness *Business) (*schema.Business, error) {
	if domainBusiness == nil {
		return nil, errors.New("business cannot be nil")
	}

	schemaBusiness := &schema.Business{
		ID:                 domainBusiness.ID,
		Slug:               domainBusiness.Slug,
		Name:               domainBusiness.Name,
		DisplayName:        domainBusiness.DisplayName,
		Description:        domainBusiness.Description,
		BusinessType:       schema.BusinessType(domainBusiness.BusinessType),
		RegistrationNumber: domainBusiness.RegistrationNumber,
		TaxID:              domainBusiness.TaxID,
		LegalEntityType:    domainBusiness.LegalEntityType,
		Email:              domainBusiness.Email,
		PhoneNumbers:       make(pq.StringArray, len(domainBusiness.PhoneNumbers)),
		Website:            domainBusiness.Website,
		Address: schema.Address{
			HouseNumber:    domainBusiness.Address.HouseNumber,
			Street:         domainBusiness.Address.Street,
			Area:           domainBusiness.Address.Area,
			LGA:            domainBusiness.Address.LGA,
			City:           domainBusiness.Address.City,
			State:          domainBusiness.Address.State,
			Country:        domainBusiness.Address.Country,
			PostalCode:     domainBusiness.Address.PostalCode,
			District:       domainBusiness.Address.District,
			DigitalAddress: domainBusiness.Address.DigitalAddress,
		},
		LogoURL:       domainBusiness.LogoURL,
		CoverImageURL: domainBusiness.CoverImageURL,
		BrandColor:    domainBusiness.BrandColor,
		IsVerified:    domainBusiness.IsVerified,
		VerifiedAt:    domainBusiness.VerifiedAt,
		IsActive:      domainBusiness.IsActive,
		BillingEmail:  domainBusiness.BillingEmail,
		CreatedBy:     domainBusiness.CreatedBy,
		CreatedAt:     domainBusiness.CreatedAt,
		UpdatedAt:     domainBusiness.UpdatedAt,
	}

	// Copy phone numbers
	copy(schemaBusiness.PhoneNumbers, domainBusiness.PhoneNumbers)

	// Map location if present
	if domainBusiness.Location != nil {
		schemaBusiness.Location = &propertySchema.GeographyPoint{
			Lat:  domainBusiness.Location.Lat,
			Lng:  domainBusiness.Location.Lng,
			SRID: domainBusiness.Location.SRID,
		}
	}

	// Map DeletedAt
	if domainBusiness.DeletedAt != nil {
		schemaBusiness.DeletedAt.Time = *domainBusiness.DeletedAt
		schemaBusiness.DeletedAt.Valid = true
	}

	return schemaBusiness, nil
}

// MapBusinessesFromSchema converts a slice of schema businesses to domain businesses
func MapBusinessesFromSchema(schemaBusinesses []*schema.Business) []Business {
	if schemaBusinesses == nil {
		return nil
	}

	businesses := make([]Business, 0, len(schemaBusinesses))
	for _, sb := range schemaBusinesses {
		if mapped := MapBusinessFromSchema(sb); mapped != nil {
			businesses = append(businesses, *mapped)
		}
	}

	return businesses
}

// MapBusinessMemberFromSchema converts a repository business member to a domain business member
func MapBusinessMemberFromSchema(schemaMember *schema.BusinessMember) *BusinessMember {
	if schemaMember == nil {
		return nil
	}

	member := &BusinessMember{
		ID:         schemaMember.ID,
		BusinessID: schemaMember.BusinessID,
		UserID:     schemaMember.UserID,
		Role:       MemberRole(schemaMember.Role),
		Permissions: MemberPermissions{
			CanCreateListings:  schemaMember.Permissions["CanCreateListings"],
			CanEditListings:    schemaMember.Permissions["CanEditListings"],
			CanDeleteListings:  schemaMember.Permissions["CanDeleteListings"],
			CanPublishListings: schemaMember.Permissions["CanPublishListings"],
			CanManageMedia:     schemaMember.Permissions["CanManageMedia"],
			CanViewAnalytics:   schemaMember.Permissions["CanViewAnalytics"],
			CanManageMembers:   schemaMember.Permissions["CanManageMembers"],
			CanEditBusiness:    schemaMember.Permissions["CanEditBusiness"],
			CanViewFinancials:  schemaMember.Permissions["CanViewFinancials"],
		},
		IsActive:  schemaMember.IsActive,
		InvitedBy: schemaMember.InvitedBy,
		JoinedAt:  schemaMember.JoinedAt,
		CreatedAt: schemaMember.CreatedAt,
		UpdatedAt: schemaMember.UpdatedAt,
	}

	return member
}

// MapBusinessMemberToSchema converts a domain business member to a repository business member
func MapBusinessMemberToSchema(domainMember *BusinessMember) (*schema.BusinessMember, error) {
	if domainMember == nil {
		return nil, errors.New("business member cannot be nil")
	}

	schemaMember := &schema.BusinessMember{
		ID:         domainMember.ID,
		BusinessID: domainMember.BusinessID,
		UserID:     domainMember.UserID,
		Role:       schema.MemberRole(domainMember.Role),
		Permissions: map[string]bool{
			"CanCreateListings":  domainMember.Permissions.CanCreateListings,
			"CanEditListings":    domainMember.Permissions.CanEditListings,
			"CanDeleteListings":  domainMember.Permissions.CanDeleteListings,
			"CanPublishListings": domainMember.Permissions.CanPublishListings,
			"CanManageMedia":     domainMember.Permissions.CanManageMedia,
			"CanViewAnalytics":   domainMember.Permissions.CanViewAnalytics,
			"CanManageMembers":   domainMember.Permissions.CanManageMembers,
			"CanEditBusiness":    domainMember.Permissions.CanEditBusiness,
			"CanViewFinancials":  domainMember.Permissions.CanViewFinancials,
		},
		IsActive:  domainMember.IsActive,
		InvitedBy: domainMember.InvitedBy,
		JoinedAt:  domainMember.JoinedAt,
		CreatedAt: domainMember.CreatedAt,
		UpdatedAt: domainMember.UpdatedAt,
	}

	return schemaMember, nil
}

// MapBusinessMembersFromSchema converts a slice of schema business members to domain business members
func MapBusinessMembersFromSchema(schemaMembers []*schema.BusinessMember) []BusinessMember {
	if schemaMembers == nil {
		return nil
	}

	members := make([]BusinessMember, 0, len(schemaMembers))
	for _, sm := range schemaMembers {
		if mapped := MapBusinessMemberFromSchema(sm); mapped != nil {
			members = append(members, *mapped)
		}
	}

	return members
}

// MapBusinessInvitationFromSchema converts a repository business invitation to a domain business invitation
func MapBusinessInvitationFromSchema(schemaInvitation *schema.BusinessInvitation) *BusinessInvitation {
	if schemaInvitation == nil {
		return nil
	}

	invitation := &BusinessInvitation{
		ID:         schemaInvitation.ID,
		BusinessID: schemaInvitation.BusinessID,
		Email:      schemaInvitation.Email,
		UserID:     schemaInvitation.UserID,
		Role:       MemberRole(schemaInvitation.Role),
		Permissions: MemberPermissions{
			CanCreateListings:  schemaInvitation.Permissions["CanCreateListings"],
			CanEditListings:    schemaInvitation.Permissions["CanEditListings"],
			CanDeleteListings:  schemaInvitation.Permissions["CanDeleteListings"],
			CanPublishListings: schemaInvitation.Permissions["CanPublishListings"],
			CanManageMedia:     schemaInvitation.Permissions["CanManageMedia"],
			CanViewAnalytics:   schemaInvitation.Permissions["CanViewAnalytics"],
			CanManageMembers:   schemaInvitation.Permissions["CanManageMembers"],
			CanEditBusiness:    schemaInvitation.Permissions["CanEditBusiness"],
			CanViewFinancials:  schemaInvitation.Permissions["CanViewFinancials"],
		},
		InvitedBy:  schemaInvitation.InvitedBy,
		Token:      schemaInvitation.Token,
		Status:     InvitationStatus(schemaInvitation.Status),
		ExpiresAt:  schemaInvitation.ExpiresAt,
		AcceptedAt: schemaInvitation.AcceptedAt,
		CreatedAt:  schemaInvitation.CreatedAt,
		UpdatedAt:  schemaInvitation.UpdatedAt,
	}

	return invitation
}

// MapBusinessInvitationToSchema converts a domain business invitation to a repository business invitation
func MapBusinessInvitationToSchema(domainInvitation *BusinessInvitation) (*schema.BusinessInvitation, error) {
	if domainInvitation == nil {
		return nil, errors.New("business invitation cannot be nil")
	}

	schemaInvitation := &schema.BusinessInvitation{
		ID:         domainInvitation.ID,
		BusinessID: domainInvitation.BusinessID,
		Email:      domainInvitation.Email,
		UserID:     domainInvitation.UserID,
		Role:       schema.MemberRole(domainInvitation.Role),
		Permissions: map[string]bool{
			"CanCreateListings":  domainInvitation.Permissions.CanCreateListings,
			"CanEditListings":    domainInvitation.Permissions.CanEditListings,
			"CanDeleteListings":  domainInvitation.Permissions.CanDeleteListings,
			"CanPublishListings": domainInvitation.Permissions.CanPublishListings,
			"CanManageMedia":     domainInvitation.Permissions.CanManageMedia,
			"CanViewAnalytics":   domainInvitation.Permissions.CanViewAnalytics,
			"CanManageMembers":   domainInvitation.Permissions.CanManageMembers,
			"CanEditBusiness":    domainInvitation.Permissions.CanEditBusiness,
			"CanViewFinancials":  domainInvitation.Permissions.CanViewFinancials,
		},
		InvitedBy:  domainInvitation.InvitedBy,
		Token:      domainInvitation.Token,
		Status:     schema.InvitationStatus(domainInvitation.Status),
		ExpiresAt:  domainInvitation.ExpiresAt,
		AcceptedAt: domainInvitation.AcceptedAt,
		CreatedAt:  domainInvitation.CreatedAt,
		UpdatedAt:  domainInvitation.UpdatedAt,
	}

	return schemaInvitation, nil
}

// MapBusinessInvitationsFromSchema converts a slice of schema business invitations to domain business invitations
func MapBusinessInvitationsFromSchema(schemaInvitations []*schema.BusinessInvitation) []BusinessInvitation {
	if schemaInvitations == nil {
		return nil
	}

	invitations := make([]BusinessInvitation, 0, len(schemaInvitations))
	for _, si := range schemaInvitations {
		if mapped := MapBusinessInvitationFromSchema(si); mapped != nil {
			invitations = append(invitations, *mapped)
		}
	}

	return invitations
}

// MapBusinessMemberToPermissionsMap converts MemberPermissions to map[string]bool
func MapBusinessMemberToPermissionsMap(permissions MemberPermissions) map[string]bool {
	return map[string]bool{
		"CanCreateListings":  permissions.CanCreateListings,
		"CanEditListings":    permissions.CanEditListings,
		"CanDeleteListings":  permissions.CanDeleteListings,
		"CanPublishListings": permissions.CanPublishListings,
		"CanManageMedia":     permissions.CanManageMedia,
		"CanViewAnalytics":   permissions.CanViewAnalytics,
		"CanManageMembers":   permissions.CanManageMembers,
		"CanEditBusiness":    permissions.CanEditBusiness,
		"CanViewFinancials":  permissions.CanViewFinancials,
	}
}
