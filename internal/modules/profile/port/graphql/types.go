package graphql

import "time"

type TravelCompanionInput struct {
	Name         string  `json:"name"`
	AgeGroup     string  `json:"ageGroup"`
	Phone        *string `json:"phone,omitempty"`
	Relationship string  `json:"relationship"`
	PhotoURL     *string `json:"photoUrl,omitempty"`
}

type UpdateProfileInput struct {
	FullName                   *string    `json:"fullName,omitempty"`
	BirthDate                  *time.Time `json:"birthDate,omitempty"`
	Gender                     *string    `json:"gender,omitempty"`
	Email                      *string    `json:"email,omitempty"`
	PhoneNumbers               []string   `json:"phoneNumbers,omitempty"`
	ProfilePhotoURL            *string    `json:"profilePhotoURL,omitempty"`
	Address                    *string    `json:"address,omitempty"`
	Street                     *string    `json:"street,omitempty"`
	HouseNumber                *string    `json:"houseNumber,omitempty"`
	Area                       *string    `json:"area,omitempty"`
	Lga                        *string    `json:"lga,omitempty"`
	District                   *string    `json:"district,omitempty"`
	DigitalAddress             *string    `json:"digitalAddress,omitempty"`
	City                       *string    `json:"city,omitempty"`
	State                      *string    `json:"state,omitempty"`
	Country                    *string    `json:"country,omitempty"`
	ZipCode                    *string    `json:"zipCode,omitempty"`
	Occupation                 *string    `json:"occupation,omitempty"`
	Education                  *string    `json:"education,omitempty"`
	Bio                        *string    `json:"bio,omitempty"`
	Skills                     []string   `json:"skills,omitempty"`
	Languages                  []string   `json:"languages,omitempty"`
	Interests                  []string   `json:"interests,omitempty"`
	Hobbies                    []string   `json:"hobbies,omitempty"`
	FunFact                    *string    `json:"funFact,omitempty"`
	ObsessedWith               *string    `json:"obsessedWith,omitempty"`
	CommunityCommitment        *bool      `json:"communityCommitment,omitempty"`
	BioVisible                 *bool      `json:"bioVisible,omitempty"`
	AllowPersonalizedOffers    *bool      `json:"allowPersonalizedOffers,omitempty"`
	EnablePerformanceAnalytics *bool      `json:"enablePerformanceAnalytics,omitempty"`
}
