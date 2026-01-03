package graphql

import (
	"hauslet/internal/modules/business/domain"
	"hauslet/internal/transport/graph/model"
)

type BusinessAddressInput struct {
	HouseNumber    *string `json:"houseNumber,omitempty"`
	Street         *string `json:"street,omitempty"`
	Area           *string `json:"area,omitempty"`
	Lga            *string `json:"lga,omitempty"`
	City           *string `json:"city,omitempty"`
	State          *string `json:"state,omitempty"`
	Country        *string `json:"country,omitempty"`
	PostalCode     *string `json:"postalCode,omitempty"`
	District       *string `json:"district,omitempty"`
	DigitalAddress *string `json:"digitalAddress,omitempty"`
}

type CreateBusinessInput struct {
	Name               string                `json:"name"`
	DisplayName        string                `json:"displayName"`
	Description        *string               `json:"description,omitempty"`
	BusinessType       domain.BusinessType   `json:"businessType"`
	RegistrationNumber *string               `json:"registrationNumber,omitempty"`
	TaxID              *string               `json:"taxID,omitempty"`
	LegalEntityType    *string               `json:"legalEntityType,omitempty"`
	Email              string                `json:"email"`
	PhoneNumbers       []string              `json:"phoneNumbers,omitempty"`
	Website            *string               `json:"website,omitempty"`
	Address            *BusinessAddressInput `json:"address"`
	Location           *model.LocationInput  `json:"location,omitempty"`
	LogoURL            *string               `json:"logoURL,omitempty"`
	CoverImageURL      *string               `json:"coverImageURL,omitempty"`
	BrandColor         *string               `json:"brandColor,omitempty"`
	BillingEmail       *string               `json:"billingEmail,omitempty"`
}

type UpdateBusinessInput struct {
	DisplayName   *string               `json:"displayName,omitempty"`
	Description   *string               `json:"description,omitempty"`
	Email         *string               `json:"email,omitempty"`
	PhoneNumbers  []string              `json:"phoneNumbers,omitempty"`
	Website       *string               `json:"website,omitempty"`
	Address       *BusinessAddressInput `json:"address,omitempty"`
	Location      *model.LocationInput  `json:"location,omitempty"`
	LogoURL       *string               `json:"logoURL,omitempty"`
	CoverImageURL *string               `json:"coverImageURL,omitempty"`
	BrandColor    *string               `json:"brandColor,omitempty"`
	BillingEmail  *string               `json:"billingEmail,omitempty"`
}

type InviteMemberInput struct {
	Email             string                  `json:"email"`
	Role              domain.MemberRole       `json:"role"`
	CustomPermissions *MemberPermissionsInput `json:"customPermissions,omitempty"`
}

type MemberPermissionsInput struct {
	CanCreateListings  *bool `json:"canCreateListings,omitempty"`
	CanEditListings    *bool `json:"canEditListings,omitempty"`
	CanDeleteListings  *bool `json:"canDeleteListings,omitempty"`
	CanPublishListings *bool `json:"canPublishListings,omitempty"`
	CanManageMedia     *bool `json:"canManageMedia,omitempty"`
	CanViewAnalytics   *bool `json:"canViewAnalytics,omitempty"`
	CanManageMembers   *bool `json:"canManageMembers,omitempty"`
	CanEditBusiness    *bool `json:"canEditBusiness,omitempty"`
	CanViewFinancials  *bool `json:"canViewFinancials,omitempty"`
}
