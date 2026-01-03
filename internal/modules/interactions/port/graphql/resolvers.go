package graphql

import (
	"context"
	"hauslet/internal/modules/interactions/domain"
	"hauslet/internal/modules/interactions/service"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Resolver holds the services for interactions module
type Resolver struct {
	tracker service.TrackerService
	reader  service.ReaderService
	log     *slog.Logger
}

// NewResolver creates a new GraphQL resolver for interactions
func NewResolver(tracker service.TrackerService, reader service.ReaderService, log *slog.Logger) *Resolver {
	return &Resolver{
		tracker: tracker,
		reader:  reader,
		log:     log,
	}
}

// TrackInteraction tracks a user interaction (mutation)
func (r *Resolver) TrackInteraction(ctx context.Context, input TrackInteractionInput) (bool, error) {
	// Get session ID from context (set by middleware)
	sessionID, ok := ctx.Value("session_id").(string)
	if !ok || sessionID == "" {
		sessionID = uuid.New().String() // Fallback to generated session
	}

	// Get user ID if authenticated (optional)
	var userID *uuid.UUID
	if userIDStr, ok := ctx.Value("user_id").(string); ok && userIDStr != "" {
		if uid, err := uuid.Parse(userIDStr); err == nil {
			userID = &uid
		}
	}

	// Get metadata from context (set by middleware)
	userAgent, _ := ctx.Value("user_agent").(string)
	ipAddress, _ := ctx.Value("ip_address").(string)

	trackInput := service.TrackInput{
		UserID:     userID,
		SessionID:  sessionID,
		Type:       convertInteractionType(input.Type),
		EntityType: convertEntityType(input.EntityType),
		EntityID:   input.EntityID,
		Context:    input.Context,
		DeviceType: convertDeviceType(input.DeviceType),
		Platform:   convertPlatform(input.Platform),
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Referrer:   stringValue(input.Referrer),
	}

	if err := r.tracker.Track(ctx, trackInput); err != nil {
		if r.log != nil {
			r.log.Error("failed to track interaction", "error", err)
		}
		return false, err
	}

	return true, nil
}

// ListingAnalytics retrieves analytics for a listing (query)
func (r *Resolver) ListingAnalytics(ctx context.Context, listingID uuid.UUID, days int) (*ListingAnalyticsResponse, error) {
	analytics, err := r.reader.GetListingAnalytics(ctx, listingID, days)
	if err != nil {
		if r.log != nil {
			r.log.Error("failed to get listing analytics", "error", err, "listing_id", listingID)
		}
		return nil, err
	}

	return convertListingAnalytics(analytics), nil
}

// MyInteractionHistory retrieves user's interaction history (query)
func (r *Resolver) MyInteractionHistory(ctx context.Context, limit *int) ([]*InteractionResponse, error) {
	// Get authenticated user ID from context
	userIDStr, ok := ctx.Value("user_id").(string)
	if !ok || userIDStr == "" {
		return nil, domain.ErrUnauthorized
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	historyLimit := 50
	if limit != nil && *limit > 0 {
		historyLimit = *limit
	}

	history, err := r.reader.GetUserHistory(ctx, userID, historyLimit)
	if err != nil {
		if r.log != nil {
			r.log.Error("failed to get user history", "error", err, "user_id", userID)
		}
		return nil, err
	}

	return convertInteractions(history.Interactions), nil
}

// Helper functions for type conversion

func convertInteractionType(t InteractionTypeInput) domain.InteractionType {
	switch t {
	case InteractionTypeViewListing:
		return domain.InteractionViewListing
	case InteractionTypeViewListingDetail:
		return domain.InteractionViewListingDetail
	case InteractionTypeViewMedia:
		return domain.InteractionViewMedia
	case InteractionTypeViewMap:
		return domain.InteractionViewMap
	case InteractionTypeSaveListing:
		return domain.InteractionSaveListing
	case InteractionTypeUnsaveListing:
		return domain.InteractionUnsaveListing
	case InteractionTypeShareListing:
		return domain.InteractionShareListing
	case InteractionTypeContactOwner:
		return domain.InteractionContactOwner
	case InteractionTypeRequestViewing:
		return domain.InteractionRequestViewing
	case InteractionTypeBookingRequest:
		return domain.InteractionBookingRequest
	case InteractionTypeSearch:
		return domain.InteractionSearch
	case InteractionTypeFilterApply:
		return domain.InteractionFilterApply
	case InteractionTypeScrollDeep:
		return domain.InteractionScrollDeep
	case InteractionTypeTimeMilestone:
		return domain.InteractionTimeMilestone
	default:
		return domain.InteractionViewListing
	}
}

func convertEntityType(t EntityTypeInput) domain.EntityType {
	switch t {
	case EntityTypeListing:
		return domain.EntityTypeListing
	case EntityTypeSearch:
		return domain.EntityTypeSearch
	case EntityTypeProfile:
		return domain.EntityTypeProfile
	default:
		return domain.EntityTypeListing
	}
}

func convertDeviceType(t *DeviceTypeInput) domain.DeviceType {
	if t == nil {
		return domain.DeviceUnknown
	}
	switch *t {
	case DeviceTypeDesktop:
		return domain.DeviceDesktop
	case DeviceTypeMobile:
		return domain.DeviceMobile
	case DeviceTypeTablet:
		return domain.DeviceTablet
	default:
		return domain.DeviceUnknown
	}
}

func convertPlatform(p *PlatformInput) domain.Platform {
	if p == nil {
		return domain.PlatformUnknown
	}
	switch *p {
	case PlatformWeb:
		return domain.PlatformWeb
	case PlatformIos:
		return domain.PlatformIOS
	case PlatformAndroid:
		return domain.PlatformAndroid
	default:
		return domain.PlatformUnknown
	}
}

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func convertListingAnalytics(a *domain.ListingAnalytics) *ListingAnalyticsResponse {
	return &ListingAnalyticsResponse{
		ListingID:       a.ListingID,
		Period:          convertPeriod(a.Period),
		TotalViews:      int(a.TotalViews),
		UniqueViews:     int(a.UniqueViews),
		MediaViews:      int(a.MediaViews),
		MapViews:        int(a.MapViews),
		TotalSaves:      int(a.TotalSaves),
		NetSaves:        int(a.NetSaves),
		TotalShares:     int(a.TotalShares),
		TotalContacts:   int(a.TotalContacts),
		BookingRequests: int(a.BookingRequests),
		ConversionRate:  a.ConversionRate,
		EngagementRate:  a.EngagementRate,
		AvgTimeOnPage:   a.AvgTimeOnPage,
		ViewsTrend:      a.ViewsTrend,
		SavesTrend:      a.SavesTrend,
		EngagementTrend: a.EngagementTrend,
	}
}

func convertPeriod(p domain.Period) PeriodResponse {
	return PeriodResponse{
		StartDate: p.StartDate,
		EndDate:   p.EndDate,
		Days:      p.Days,
	}
}

func convertInteractions(interactions []*domain.Interaction) []*InteractionResponse {
	responses := make([]*InteractionResponse, len(interactions))
	for i, interaction := range interactions {
		responses[i] = &InteractionResponse{
			ID:         interaction.ID,
			UserID:     interaction.UserID,
			SessionID:  interaction.SessionID,
			Type:       string(interaction.Type),
			EntityType: string(interaction.EntityType),
			EntityID:   interaction.EntityID,
			Context:    interaction.Context,
			IsBot:      interaction.IsBot,
			CreatedAt:  interaction.CreatedAt,
		}
	}
	return responses
}

// GraphQL types (these will be auto-generated by gqlgen, but defined here for reference)

type TrackInteractionInput struct {
	Type       InteractionTypeInput
	EntityType EntityTypeInput
	EntityID   *uuid.UUID
	Context    map[string]any
	DeviceType *DeviceTypeInput
	Platform   *PlatformInput
	Referrer   *string
}

type InteractionTypeInput string
type EntityTypeInput string
type DeviceTypeInput string
type PlatformInput string

const (
	InteractionTypeViewListing       InteractionTypeInput = "VIEW_LISTING"
	InteractionTypeViewListingDetail InteractionTypeInput = "VIEW_LISTING_DETAIL"
	InteractionTypeViewMedia         InteractionTypeInput = "VIEW_MEDIA"
	InteractionTypeViewMap           InteractionTypeInput = "VIEW_MAP"
	InteractionTypeSaveListing       InteractionTypeInput = "SAVE_LISTING"
	InteractionTypeUnsaveListing     InteractionTypeInput = "UNSAVE_LISTING"
	InteractionTypeShareListing      InteractionTypeInput = "SHARE_LISTING"
	InteractionTypeContactOwner      InteractionTypeInput = "CONTACT_OWNER"
	InteractionTypeRequestViewing    InteractionTypeInput = "REQUEST_VIEWING"
	InteractionTypeBookingRequest    InteractionTypeInput = "BOOKING_REQUEST"
	InteractionTypeSearch            InteractionTypeInput = "SEARCH"
	InteractionTypeFilterApply       InteractionTypeInput = "FILTER_APPLY"
	InteractionTypeScrollDeep        InteractionTypeInput = "SCROLL_DEEP"
	InteractionTypeTimeMilestone     InteractionTypeInput = "TIME_MILESTONE"

	EntityTypeListing EntityTypeInput = "LISTING"
	EntityTypeSearch  EntityTypeInput = "SEARCH"
	EntityTypeProfile EntityTypeInput = "PROFILE"

	DeviceTypeDesktop DeviceTypeInput = "DESKTOP"
	DeviceTypeMobile  DeviceTypeInput = "MOBILE"
	DeviceTypeTablet  DeviceTypeInput = "TABLET"

	PlatformWeb     PlatformInput = "WEB"
	PlatformIos     PlatformInput = "IOS"
	PlatformAndroid PlatformInput = "ANDROID"
)

type ListingAnalyticsResponse struct {
	ListingID       uuid.UUID
	Period          PeriodResponse
	TotalViews      int
	UniqueViews     int
	MediaViews      int
	MapViews        int
	TotalSaves      int
	NetSaves        int
	TotalShares     int
	TotalContacts   int
	BookingRequests int
	ConversionRate  float64
	EngagementRate  float64
	AvgTimeOnPage   int
	ViewsTrend      float64
	SavesTrend      float64
	EngagementTrend float64
}

type PeriodResponse struct {
	StartDate time.Time
	EndDate   time.Time
	Days      int
}

type InteractionResponse struct {
	ID         uuid.UUID
	UserID     *uuid.UUID
	SessionID  string
	Type       string
	EntityType string
	EntityID   *uuid.UUID
	Context    map[string]any
	IsBot      bool
	CreatedAt  time.Time
}
