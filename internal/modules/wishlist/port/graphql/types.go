package graphql

type CreateWishlistInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	IsPrivate   *bool   `json:"isPrivate,omitempty"`
}

type UpdateWishlistInput struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	IsPrivate   *bool   `json:"isPrivate,omitempty"`
}
