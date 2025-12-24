package service

import (
	"context"
	"hauslet/internal/modules/wishlist/domain"
	"hauslet/internal/modules/wishlist/repository"
	"hauslet/internal/platform/redis"

	"github.com/go-pkgz/lgr"
	"github.com/google/uuid"
)

type WishlistService interface {
	// --- Lifecycle & Management ---
	CreateWishlist(ctx context.Context, userID uuid.UUID, name string, description *string, isPrivate bool) (*domain.Wishlist, error)
	GetWishlist(ctx context.Context, wishlistID uuid.UUID, viewerID uuid.UUID) (*domain.Wishlist, error)
	ListUserWishlists(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Wishlist, error)
	UpdateWishlist(ctx context.Context, wishlistID uuid.UUID, userID uuid.UUID, name *string, description *string, isPrivate *bool) (*domain.Wishlist, error)
	DeleteWishlist(ctx context.Context, wishlistID uuid.UUID, userID uuid.UUID) error

	// --- Item Operations ---
	AddItem(ctx context.Context, wishlistID uuid.UUID, userID uuid.UUID, listingID uuid.UUID, source domain.WishlistItemSource) (*domain.WishlistItem, error)
	RemoveItem(ctx context.Context, wishlistID uuid.UUID, userID uuid.UUID, listingID uuid.UUID) error
	GetItems(ctx context.Context, wishlistID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]*domain.WishlistItem, error)
	// IsListed checks if a listing is in a specific wishlist (for UI state).
	IsListed(ctx context.Context, wishlistID uuid.UUID, listingID uuid.UUID) (bool, error)
	CountItems(ctx context.Context, wishlistID uuid.UUID) (int64, error)
	ImportItems(ctx context.Context, targetWishlistID uuid.UUID, userID uuid.UUID, sourceWishlistID uuid.UUID, listingIDs []uuid.UUID) (imported int, skipped int, err error)
	CloneWishlist(ctx context.Context, sourceWishlistID uuid.UUID, targetUserID uuid.UUID, newName string) (*domain.Wishlist, error)

	// MergeWishlist copies items from 'sourceWishlistID' into an *existing* 'targetWishlistID'.
	// Useful if a user wants to combine a shared list into their main list.
	// Returns the number of items merged.
	MergeWishlist(ctx context.Context, sourceWishlistID uuid.UUID, targetWishlistID uuid.UUID, userID uuid.UUID) (int, error)
}

type ListingHooks interface {
	//	CanAddListingToWishlist checks if a listing is eligible to be added to wishlists.
	CanAddListingToWishlist(ctx context.Context, listingID uuid.UUID) (bool, error)
}

type WishlistServiceImpl struct {
	repo         repository.WishlistRepository
	cache        redis.RedisClient
	listingHooks ListingHooks
	log          *lgr.Logger
}

func NewWishlistService(repo repository.WishlistRepository, cache redis.RedisClient, listingHooks ListingHooks, log *lgr.Logger) WishlistService {
	return &WishlistServiceImpl{
		repo:         repo,
		cache:        cache,
		listingHooks: listingHooks,
		log:          log,
	}
}
