package domain

import (
	"time"

	"github.com/google/uuid"
)

// InteractionAggregate represents rolled-up analytics data for a specific entity and time period
type InteractionAggregate struct {
	ID          uuid.UUID  `json:"id"`
	EntityType  EntityType `json:"entity_type"`
	EntityID    uuid.UUID  `json:"entity_id"`
	PeriodType  PeriodType `json:"period_type"`
	PeriodStart time.Time  `json:"period_start"`

	// Metrics
	ViewsTotal        int64 `json:"views_total"`
	ViewsUnique       int64 `json:"views_unique"`        // Unique sessions
	SavesTotal        int64 `json:"saves_total"`
	UnsavesTotal      int64 `json:"unsaves_total"`
	SharesTotal       int64 `json:"shares_total"`
	ContactsTotal     int64 `json:"contacts_total"`
	BookingRequests   int64 `json:"booking_requests"`

	// Derived Metrics
	AvgTimeOnPageSec int     `json:"avg_time_on_page_sec"`
	EngagementScore  float64 `json:"engagement_score"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListingAnalytics provides comprehensive analytics for a listing
type ListingAnalytics struct {
	ListingID uuid.UUID `json:"listing_id"`
	Period    Period    `json:"period"`

	// View Metrics
	TotalViews       int64 `json:"total_views"`
	UniqueViews      int64 `json:"unique_views"`
	MediaViews       int64 `json:"media_views"`
	MapViews         int64 `json:"map_views"`

	// Engagement Metrics
	TotalSaves       int64 `json:"total_saves"`
	NetSaves         int64 `json:"net_saves"` // Saves - Unsaves
	TotalShares      int64 `json:"total_shares"`
	TotalContacts    int64 `json:"total_contacts"`
	BookingRequests  int64 `json:"booking_requests"`

	// Calculated Metrics
	ConversionRate   float64 `json:"conversion_rate"`   // (Contacts + Bookings) / Unique Views
	EngagementRate   float64 `json:"engagement_rate"`   // High-value actions / Unique Views
	AvgTimeOnPage    int     `json:"avg_time_on_page"`  // Seconds

	// Trend Data (compared to previous period)
	ViewsTrend       float64 `json:"views_trend"`       // % change
	SavesTrend       float64 `json:"saves_trend"`
	EngagementTrend  float64 `json:"engagement_trend"`
}

// UserActivityHistory represents a user's interaction history
type UserActivityHistory struct {
	UserID       uuid.UUID      `json:"user_id"`
	Interactions []*Interaction `json:"interactions"`
	TotalCount   int            `json:"total_count"`
}

// Period represents a time range for analytics queries
type Period struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Days      int       `json:"days"`
}

// NewPeriod creates a period for the last N days
func NewPeriod(days int) Period {
	end := time.Now()
	start := end.AddDate(0, 0, -days)

	return Period{
		StartDate: start,
		EndDate:   end,
		Days:      days,
	}
}

// CalculateEngagementScore calculates an engagement score based on interaction counts
func (a *InteractionAggregate) CalculateEngagementScore() float64 {
	if a.ViewsUnique == 0 {
		return 0
	}

	// Weighted scoring
	highValueActions := float64(a.SavesTotal*3 + a.SharesTotal*2 + a.ContactsTotal*5 + a.BookingRequests*10)
	uniqueViews := float64(a.ViewsUnique)

	// Normalize to 0-100 scale
	score := (highValueActions / uniqueViews) * 100
	if score > 100 {
		score = 100
	}

	return score
}

// GetNetSaves returns net saves (saves minus unsaves)
func (a *InteractionAggregate) GetNetSaves() int64 {
	return a.SavesTotal - a.UnsavesTotal
}
