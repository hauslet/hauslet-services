package seeders

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"hauslet/db/seeds/utils"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// InteractionSeed maps to the "interactions" table schema.
type InteractionSeed struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey"`
	UserID          *uuid.UUID     `gorm:"type:uuid"`
	SessionID       string         `gorm:"type:varchar(64);not null"`
	InteractionType string         `gorm:"type:varchar(32);not null"`
	EntityType      string         `gorm:"type:varchar(32);not null"`
	EntityID        *uuid.UUID     `gorm:"type:uuid"`
	Context         datatypes.JSON `gorm:"type:jsonb;default:'{}'"`
	DeviceType      string         `gorm:"type:varchar(16)"`
	Platform        string         `gorm:"type:varchar(16)"`
	IPHash          string         `gorm:"type:varchar(64)"`
	UserAgent       string         `gorm:"type:text"`
	Referrer        string         `gorm:"type:text"`
	IsBot           bool           `gorm:"default:false"`
	CreatedAt       time.Time      `gorm:"not null"`
}

func (InteractionSeed) TableName() string {
	return "interactions"
}

// InteractionAggregateSeed maps to the "interaction_aggregates" table schema.
type InteractionAggregateSeed struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EntityType  string    `gorm:"type:varchar(32);not null"`
	EntityID    uuid.UUID `gorm:"type:uuid;not null"`
	PeriodType  string    `gorm:"type:varchar(10);not null"`
	PeriodStart time.Time `gorm:"not null"`

	ViewsTotal       int64   `gorm:"default:0"`
	ViewsUnique      int64   `gorm:"default:0"`
	SavesTotal       int64   `gorm:"default:0"`
	UnsavesTotal     int64   `gorm:"default:0"`
	SharesTotal      int64   `gorm:"default:0"`
	ContactsTotal    int64   `gorm:"default:0"`
	BookingRequests  int64   `gorm:"default:0"`
	AvgTimeOnPageSec int     `gorm:"default:0"`
	EngagementScore  float64 `gorm:"default:0"`

	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (InteractionAggregateSeed) TableName() string {
	return "interaction_aggregates"
}

// interactionTypes we'll seed — weighted by realistic frequency.
var seedInteractionTypes = []struct {
	Type   string
	Weight int // Higher = more frequent
}{
	{"view_listing", 50},
	{"view_listing_detail", 30},
	{"view_media", 20},
	{"save_listing", 8},
	{"share_listing", 4},
	{"contact_owner", 3},
	{"booking_request", 2},
	{"scroll_deep", 15},
	{"time_milestone", 10},
}

var deviceTypes = []string{"desktop", "mobile", "tablet"}
var platforms = []string{"web", "ios", "android"}
var referrers = []string{
	"https://www.google.com",
	"https://www.instagram.com",
	"https://x.com",
	"direct",
	"",
}

// engagementTier defines how "hot" a listing is.
type engagementTier int

const (
	tierCold engagementTier = iota // Few interactions
	tierWarm                       // Moderate
	tierHot                        // Lots of activity (trending candidates)
)

// listingCounters tracks interaction counts per listing for building aggregates.
type listingCounters struct {
	views    int64
	saves    int64
	shares   int64
	contacts int64
	bookings int64
	sessions map[string]bool // for unique view counting
}

// SeedInteractions seeds interaction events and pre-computed aggregates
// for all active published listings.
func SeedInteractions(ctx *SeedContext) error {
	type listingRow struct {
		ID          uuid.UUID
		ListingType string
	}

	var listings []listingRow
	if err := ctx.DB.Table("listings").
		Select("id, listing_type").
		Where("status = ? AND published = ? AND deleted_at IS NULL", "active", true).
		Scan(&listings).Error; err != nil {
		return fmt.Errorf("failed to fetch active listings: %w", err)
	}

	var userIDs []uuid.UUID
	if err := ctx.DB.Table("users").
		Select("id").
		Where("deleted_at IS NULL").
		Scan(&userIDs).Error; err != nil {
		return fmt.Errorf("failed to fetch users: %w", err)
	}

	if len(listings) == 0 {
		fmt.Println("    No active listings found — skipping interactions seeding")
		return nil
	}

	fmt.Printf("    Generating interactions for %d listings...\n", len(listings))

	now := time.Now()
	interactions := make([]InteractionSeed, 0, len(listings)*30)
	aggregates := make([]InteractionAggregateSeed, 0, len(listings)*10)

	for _, listing := range listings {
		tier := assignEngagementTier()

		// How many raw interaction events for this listing?
		eventCount := tierEventCount(tier)

		counters := listingCounters{
			sessions: make(map[string]bool),
		}

		for i := 0; i < eventCount; i++ {
			// Pick a random interaction type (weighted).
			iType := weightedInteractionType()

			// Random time in last 30 days.
			daysAgo := utils.RandomInt(0, 30)
			hoursAgo := utils.RandomInt(0, 23)
			minutesAgo := utils.RandomInt(0, 59)
			createdAt := now.Add(
				-time.Duration(daysAgo)*24*time.Hour -
					time.Duration(hoursAgo)*time.Hour -
					time.Duration(minutesAgo)*time.Minute,
			)

			// 60% of interactions are from logged-in users, 40% anonymous.
			var userID *uuid.UUID
			sessionID := fmt.Sprintf("sess_%s", utils.RandomString(12))
			if len(userIDs) > 0 && utils.RandomBoolWithProbability(0.6) {
				uid := userIDs[utils.RandomInt(0, len(userIDs)-1)]
				userID = &uid
				sessionID = fmt.Sprintf("user_%s", uid.String()[:8])
			}

			device := utils.RandomChoice(deviceTypes)
			platform := utils.RandomChoice(platforms)
			referrer := utils.RandomChoice(referrers)
			entityID := listing.ID

			// Build context JSON.
			ctxMap := map[string]any{
				"source": "home_feed",
			}
			if iType == "view_listing" || iType == "view_listing_detail" {
				ctxMap["position"] = utils.RandomInt(1, 20)
			}
			if iType == "time_milestone" {
				ctxMap["seconds"] = utils.RandomChoice([]int{30, 60, 120})
			}
			ctxJSON, _ := json.Marshal(ctxMap)

			interactions = append(interactions, InteractionSeed{
				ID:              uuid.New(),
				UserID:          userID,
				SessionID:       sessionID,
				InteractionType: iType,
				EntityType:      "listing",
				EntityID:        &entityID,
				Context:         datatypes.JSON(ctxJSON),
				DeviceType:      device,
				Platform:        platform,
				IPHash:          utils.RandomString(16),
				UserAgent:       randomUserAgent(device),
				Referrer:        referrer,
				IsBot:           false,
				CreatedAt:       createdAt,
			})

			// Count for aggregates.
			counters.sessions[sessionID] = true
			switch iType {
			case "view_listing", "view_listing_detail":
				counters.views++
			case "save_listing":
				counters.saves++
			case "share_listing":
				counters.shares++
			case "contact_owner":
				counters.contacts++
			case "booking_request":
				counters.bookings++
			}
		}

		// Build aggregates for this listing: daily for last 7 days, weekly for last 4 weeks.
		aggregates = append(aggregates, buildDailyAggregates(listing.ID, counters, now)...)
		aggregates = append(aggregates, buildWeeklyAggregate(listing.ID, counters, now))
	}

	fmt.Printf("    Inserting %d interaction events...\n", len(interactions))
	if err := batchInsertInteractions(ctx.DB, interactions, 500); err != nil {
		return fmt.Errorf("failed to create interactions: %w", err)
	}

	fmt.Printf("    Inserting %d interaction aggregates...\n", len(aggregates))
	if err := ctx.DB.CreateInBatches(aggregates, 100).Error; err != nil {
		return fmt.Errorf("failed to create interaction aggregates: %w", err)
	}

	return nil
}

// batchInsertInteractions inserts interactions in batches.
// The interactions table may be partitioned, so we use raw SQL to avoid GORM
// issues with partitioned tables.
func batchInsertInteractions(db *gorm.DB, interactions []InteractionSeed, batchSize int) error {
	for i := 0; i < len(interactions); i += batchSize {
		end := i + batchSize
		if end > len(interactions) {
			end = len(interactions)
		}
		batch := interactions[i:end]
		if err := db.CreateInBatches(batch, batchSize).Error; err != nil {
			return err
		}
	}
	return nil
}

// ===== TIER ASSIGNMENT =====

// assignEngagementTier gives ~20% of listings "hot" status for trending.
func assignEngagementTier() engagementTier {
	roll := utils.RandomInt(1, 100)
	switch {
	case roll <= 20:
		return tierHot // 20% — trending candidates
	case roll <= 50:
		return tierWarm // 30% — moderate engagement
	default:
		return tierCold // 50% — low engagement
	}
}

func tierEventCount(tier engagementTier) int {
	switch tier {
	case tierHot:
		return utils.RandomInt(40, 80) // Lots of activity
	case tierWarm:
		return utils.RandomInt(15, 35)
	default:
		return utils.RandomInt(3, 12) // Low / new listing
	}
}

// ===== WEIGHTED RANDOM =====

func weightedInteractionType() string {
	totalWeight := 0
	for _, t := range seedInteractionTypes {
		totalWeight += t.Weight
	}

	roll := utils.RandomInt(1, totalWeight)
	cumulative := 0
	for _, t := range seedInteractionTypes {
		cumulative += t.Weight
		if roll <= cumulative {
			return t.Type
		}
	}
	return "view_listing"
}

// ===== AGGREGATE BUILDERS =====

func buildDailyAggregates(listingID uuid.UUID, c listingCounters, now time.Time) []InteractionAggregateSeed {
	var aggs []InteractionAggregateSeed

	// Distribute counts across the last 7 days with some randomness.
	for day := 0; day < 7; day++ {
		periodStart := time.Date(now.Year(), now.Month(), now.Day()-day, 0, 0, 0, 0, now.Location())

		// Each day gets a fraction of the total, with variance.
		dayFraction := 1.0 / 7.0
		variance := 0.5 + float64(utils.RandomInt(50, 150))/100.0

		views := int64(math.Round(float64(c.views) * dayFraction * variance))
		unique := int64(float64(len(c.sessions)) * dayFraction * variance)
		if unique > views {
			unique = views
		}
		saves := int64(math.Round(float64(c.saves) * dayFraction * variance))
		shares := int64(math.Round(float64(c.shares) * dayFraction * variance))
		contacts := int64(math.Round(float64(c.contacts) * dayFraction * variance))
		bookings := int64(math.Round(float64(c.bookings) * dayFraction * variance))

		engagement := calculateEngagementScore(views, unique, saves, shares, contacts, bookings)

		aggs = append(aggs, InteractionAggregateSeed{
			ID:               uuid.New(),
			EntityType:       "listing",
			EntityID:         listingID,
			PeriodType:       "day",
			PeriodStart:      periodStart,
			ViewsTotal:       views,
			ViewsUnique:      unique,
			SavesTotal:       saves,
			UnsavesTotal:     0,
			SharesTotal:      shares,
			ContactsTotal:    contacts,
			BookingRequests:  bookings,
			AvgTimeOnPageSec: utils.RandomInt(15, 120),
			EngagementScore:  engagement,
			CreatedAt:        periodStart,
			UpdatedAt:        now,
		})
	}
	return aggs
}

func buildWeeklyAggregate(listingID uuid.UUID, c listingCounters, now time.Time) InteractionAggregateSeed {
	periodStart := time.Date(now.Year(), now.Month(), now.Day()-7, 0, 0, 0, 0, now.Location())

	engagement := calculateEngagementScore(c.views, int64(len(c.sessions)), c.saves, c.shares, c.contacts, c.bookings)

	return InteractionAggregateSeed{
		ID:               uuid.New(),
		EntityType:       "listing",
		EntityID:         listingID,
		PeriodType:       "week",
		PeriodStart:      periodStart,
		ViewsTotal:       c.views,
		ViewsUnique:      int64(len(c.sessions)),
		SavesTotal:       c.saves,
		UnsavesTotal:     0,
		SharesTotal:      c.shares,
		ContactsTotal:    c.contacts,
		BookingRequests:  c.bookings,
		AvgTimeOnPageSec: utils.RandomInt(20, 90),
		EngagementScore:  engagement,
		CreatedAt:        periodStart,
		UpdatedAt:        now,
	}
}

func calculateEngagementScore(views, unique, saves, shares, contacts, bookings int64) float64 {
	if unique == 0 {
		return 0
	}
	highValueActions := float64(saves*3 + shares*2 + contacts*5 + bookings*10)
	score := (highValueActions / float64(unique)) * 100
	if score > 100 {
		score = 100
	}
	return math.Round(score*100) / 100
}

// ===== HELPERS =====

func randomUserAgent(device string) string {
	switch device {
	case "mobile":
		agents := []string{
			"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
			"Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
		}
		return utils.RandomChoice(agents)
	case "tablet":
		return "Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1"
	default:
		agents := []string{
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		}
		return utils.RandomChoice(agents)
	}
}
