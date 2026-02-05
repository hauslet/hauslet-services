package loaders

import (
	"context"
	"net/http"

	bookingservice "hauslet/internal/modules/booking/service"
	profileservice "hauslet/internal/modules/profile/service"
	propertyservice "hauslet/internal/modules/property/service"
	reviewservice "hauslet/internal/modules/review/service"
)

type loadersKey struct{}

// Loaders bundles request-scoped loaders.
type Loaders struct {
	Profile            *ProfileLoader
	Property           *PropertyLoader
	Listing            *ListingLoader
	Booking            *BookingLoader
	ListingsByProperty *ListingsByPropertyLoader
	ListingStats       *ListingStatsLoader
}

// Middleware attaches loaders to the request context for GraphQL handlers.
func Middleware(
	profileSvc profileservice.ProfileService,
	propertySvc propertyservice.PropertyService,
	bookingSvc bookingservice.BookingService,
	reviewSvc reviewservice.ReviewService,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), loadersKey{}, &Loaders{
				Profile:            NewProfileLoader(profileSvc),
				Property:           NewPropertyLoader(propertySvc),
				Listing:            NewListingLoader(propertySvc),
				Booking:            NewBookingLoader(bookingSvc),
				ListingsByProperty: NewListingsByPropertyLoader(propertySvc),
				ListingStats:       NewListingStatsLoader(reviewSvc),
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// For extracts loaders from context.
func For(ctx context.Context) *Loaders {
	if l, ok := ctx.Value(loadersKey{}).(*Loaders); ok {
		return l
	}
	return nil
}
