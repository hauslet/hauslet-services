# Discovery Module Implementation Plan

## Overview
Create a new Discovery module that handles all listing search/discovery functionality, integrating Property module (search, semantic similarity) and Promotions module (boost multipliers). Property module will retain CRUD operations only.

## Architecture Principles
- **Clean Architecture**: domain → repository → service → port
- **Service Layer Authority**: All business logic and authorization in service layer
- **Hook/Adapter Pattern**: Decouple dependencies via minimal interfaces
- **One-way Dependencies**: Discovery → Property & Promotions (no reverse dependencies)

---

## Implementation Steps

### 1. Module Structure Setup

Create directory structure:
```
internal/modules/discovery/
├── domain/
│   ├── discovery.go       # RankedListing, RankingScore, HomeFeedSection
│   ├── ranking.go         # RankingConfig, ranking algorithms
│   ├── enums.go          # FeedSectionType enum
│   └── errors.go         # Domain errors
├── repository/
│   ├── interface.go      # Repository contracts
│   ├── discovery_repository.go
│   └── schema/
│       ├── search_history.go
│       └── user_preferences.go
├── service/
│   ├── interface.go      # Service + Hook interfaces
│   ├── service.go        # Main service implementation
│   ├── adapters.go       # Hook adapters for Property/Promotions
│   ├── ranking.go        # Ranking algorithm
│   └── home_feed.go      # Home feed composition
└── port/
    └── graphql/
        ├── schema.graphqls
        └── resolver.go
```

### 2. Domain Layer

**File: `internal/modules/discovery/domain/discovery.go`**
```go
// Core types
- RankedListing: Listing with score and promotion info
- RankingScore: FinalScore + breakdown (semantic, promotion, recency)
- PromotionBoostInfo: Promotion details affecting ranking
- HomeFeedSection: Section with type, title, listings
- SearchHistory: User search tracking
- UserPreferences: Personalization data (future)
```

**File: `internal/modules/discovery/domain/ranking.go`**
```go
- RankingConfig: Weights for ranking factors
- DefaultRankingConfig(): Returns default weights
- CalculateFinalScore(): Combines scores using weights
```

**File: `internal/modules/discovery/domain/enums.go`**
```go
- FeedSectionType: featured | premium | recent | recommended | near_you
```

### 3. Repository Layer

**File: `internal/modules/discovery/repository/interface.go`**
```go
type DiscoveryRepository interface {
    SaveSearchHistory(ctx, *schema.SearchHistory) error
    GetRecentSearches(ctx, userID, limit) ([]*schema.SearchHistory, error)
    SaveUserPreferences(ctx, *schema.UserPreferences) error
    GetUserPreferences(ctx, userID) (*schema.UserPreferences, error)
}
```

**Database Schema (future migration):**
```sql
CREATE TABLE discovery_search_history (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    query TEXT,
    filters JSONB,
    result_count INT NOT NULL,
    clicked_listings UUID[],
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE INDEX idx_search_history_user_created ON discovery_search_history(user_id, created_at DESC);

CREATE TABLE discovery_user_preferences (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL UNIQUE,
    preferred_locations TEXT[],
    preferred_types TEXT[],
    price_range JSONB,
    bedroom_range JSONB,
    saved_filters JSONB,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
```

### 4. Service Layer - Hook Interfaces

**File: `internal/modules/discovery/service/interface.go`**

Define main service interface:
```go
type DiscoveryService interface {
    SearchListings(ctx, filter, options) (*SearchResult, error)
    GetHomeFeed(ctx, userID, options) ([]HomeFeedSection, error)
    GetFeaturedListings(ctx, limit) ([]RankedListing, error)
    FindSimilarListings(ctx, listingID, limit) ([]RankedListing, error)
    // Future: GetRecommendations, SaveSearch, etc.
}
```

Define hook interfaces (what Discovery needs from other modules):
```go
// PropertyDiscoveryHooks - minimal interface from Property module
type PropertyDiscoveryHooks interface {
    GetListingsByIDs(ctx, ids) ([]Listing, error)
    SearchListingsWithEmbedding(ctx, filter, limit) ([]ScoredListing, error)
    GetRecentListings(ctx, limit) ([]Listing, error)
    FindSimilarListings(ctx, listingID, limit) ([]ScoredListing, error)
}

// PromotionDiscoveryHooks - minimal interface from Promotions module
type PromotionDiscoveryHooks interface {
    GetActivePromotionForListings(ctx, listingIDs) (map[uuid.UUID]*PromotionInfo, error)
    GetFeaturedListings(ctx, limit) ([]uuid.UUID, error)
    GetPremiumListings(ctx, limit) ([]uuid.UUID, error)
}

// PromotionInfo - lightweight promotion data for ranking
type PromotionInfo struct {
    PromotionID     uuid.UUID
    PromotionType   string
    BoostMultiplier float64
    ExpiresAt       time.Time
}
```

### 5. Service Layer - Adapters

**File: `internal/modules/discovery/service/adapters.go`**

Following the pattern from `internal/modules/leads/service/adapters.go`:

```go
// PropertyDiscoveryAdapter wraps PropertyService
type PropertyDiscoveryAdapter struct {
    propertySvc propertyservice.PropertyService
}

func NewPropertyDiscoveryAdapter(svc propertyservice.PropertyService) PropertyDiscoveryHooks {
    return &PropertyDiscoveryAdapter{propertySvc: svc}
}

func (a *PropertyDiscoveryAdapter) GetListingsByIDs(ctx, ids) ([]Listing, error) {
    return a.propertySvc.GetListingsByIDs(ctx, ids, false)
}

func (a *PropertyDiscoveryAdapter) SearchListingsWithEmbedding(ctx, filter, limit) {
    // Convert discovery filter to property filter
    propertyFilter := mapToPropertyFilter(filter)
    return a.propertySvc.SearchListings(ctx, propertyFilter, limit)
}
// ... implement other methods

// PromotionDiscoveryAdapter wraps PromotionService
type PromotionDiscoveryAdapter struct {
    promotionSvc promotionservice.PromotionService
}

func NewPromotionDiscoveryAdapter(svc promotionservice.PromotionService) PromotionDiscoveryHooks {
    return &PromotionDiscoveryAdapter{promotionSvc: svc}
}

func (a *PromotionDiscoveryAdapter) GetActivePromotionForListings(ctx, listingIDs) {
    // Loop through listings and fetch promotions
    // Note: Can optimize later with batch method in PromotionService
}
// ... implement other methods
```

### 6. Service Implementation - Ranking

**File: `internal/modules/discovery/service/service.go`**
```go
type ServiceImpl struct {
    repo            repository.DiscoveryRepository
    propertyHooks   PropertyDiscoveryHooks
    promotionHooks  PromotionDiscoveryHooks
    rankingConfig   domain.RankingConfig
    log             *slog.Logger
}

func NewDiscoveryService(
    repo repository.DiscoveryRepository,
    propertyHooks PropertyDiscoveryHooks,
    promotionHooks PromotionDiscoveryHooks,
    log *slog.Logger,
) *ServiceImpl {
    return &ServiceImpl{
        repo:           repo,
        propertyHooks:  propertyHooks,
        promotionHooks: promotionHooks,
        rankingConfig:  domain.DefaultRankingConfig(),
        log:            log,
    }
}
```

**File: `internal/modules/discovery/service/ranking.go`**
```go
func (s *ServiceImpl) SearchListings(ctx, filter, options) (*SearchResult, error) {
    // 1. Get base results from property semantic search
    propertyResults, err := s.propertyHooks.SearchListingsWithEmbedding(ctx, filter, options.Limit*2)

    // 2. Get promotions for listings
    listingIDs := extractListingIDs(propertyResults)
    promotions, _ := s.promotionHooks.GetActivePromotionForListings(ctx, listingIDs)

    // 3. Calculate rankings (semantic + promotion + recency)
    rankedListings := s.rankListings(propertyResults, promotions, options.RankingConfig)

    // 4. Apply limit after ranking
    if len(rankedListings) > options.Limit {
        rankedListings = rankedListings[:options.Limit]
    }

    return &SearchResult{Listings: rankedListings, ...}, nil
}

func (s *ServiceImpl) rankListings(propertyResults, promotions, config) []RankedListing {
    // For each listing:
    // - Start with semantic score from property search
    // - Apply promotion boost multiplier (10x featured, 3x premium, 1x none)
    // - Apply recency score (decay over time)
    // - Calculate final score using weighted formula
    // - Sort by final score descending
    // - Assign ranking positions
}

func calculateRecencyScore(createdAt time.Time) float64 {
    daysSinceCreation := time.Since(createdAt).Hours() / 24
    // Decay: 1.0 (new) → 0.8 (7 days) → 0.5 (30 days) → 0.1 (90+ days)
}
```

**File: `internal/modules/discovery/service/home_feed.go`**
```go
func (s *ServiceImpl) GetHomeFeed(ctx, userID, options) ([]HomeFeedSection, error) {
    sections := []HomeFeedSection{}

    // 1. Featured Section - get featured promotion IDs, fetch listings, rank
    featuredIDs := s.promotionHooks.GetFeaturedListings(ctx, 10)
    featuredListings := s.propertyHooks.GetListingsByIDs(ctx, featuredIDs)
    promotions := s.promotionHooks.GetActivePromotionForListings(ctx, featuredIDs)
    sections = append(sections, HomeFeedSection{
        SectionType: FeedSectionFeatured,
        Title: "Featured Properties",
        Listings: s.rankListings(featuredListings, promotions, nil),
    })

    // 2. Premium Section - same pattern
    // 3. Recent Section - get recent listings
    // 4. Recommended (if authenticated) - future implementation

    return sections, nil
}
```

### 7. GraphQL Layer

**File: `internal/modules/discovery/port/graphql/schema.graphqls`**
```graphql
extend type Query {
  discover(
    filter: DiscoverySearchFilterInput!
    options: SearchOptionsInput
  ): SearchResult!

  homeFeed(options: FeedOptionsInput): [HomeFeedSection!]!
  featuredListings(limit: Int): [RankedListing!]!
  similarListings(listingId: UUID!, limit: Int): [RankedListing!]!
}

type RankedListing {
  listing: Listing!
  score: RankingScore!
  ranking: Int!
  promotionBoost: PromotionBoostInfo
}

type RankingScore {
  finalScore: Float!
  semanticScore: Float
  promotionBoost: Float!
  recencyScore: Float!
}

type HomeFeedSection {
  sectionType: FeedSectionType!
  title: String!
  listings: [RankedListing!]!
  totalCount: Int!
}

enum FeedSectionType {
  featured
  premium
  recent
  recommended
}
```

**File: `internal/modules/discovery/port/graphql/resolver.go`**
```go
type Resolver struct {
    discoverySvc service.DiscoveryService
    log          *slog.Logger
}

func NewResolver(discoverySvc service.DiscoveryService, log *slog.Logger) *Resolver {
    return &Resolver{discoverySvc: discoverySvc, log: log}
}

func (r *Resolver) Discover(ctx, filter, options) (*SearchResult, error) {
    serviceFilter := mapToServiceFilter(filter)
    serviceOptions := mapToServiceOptions(options)

    result, err := r.discoverySvc.SearchListings(ctx, serviceFilter, serviceOptions)
    if err != nil {
        r.log.Error("discover query failed", "error", err)
        return nil, err
    }

    return mapToGraphQLSearchResult(result), nil
}
// ... other resolver methods
```

### 8. Container Wiring

**File: `cmd/api/server/container.go`**

Add to Container struct (after PromotionSvc):
```go
DiscoverySvc discoveryservice.DiscoveryService
```

Add initialization method (call AFTER initPromotions):
```go
func (c *Container) initDiscovery() error {
    discoveryRepo := discoveryrepository.NewDiscoveryRepository(c.DB)

    // Create hook adapters
    propertyHooks := discoveryservice.NewPropertyDiscoveryAdapter(c.PropertySvc)
    promotionHooks := discoveryservice.NewPromotionDiscoveryAdapter(c.PromotionSvc)

    c.DiscoverySvc = discoveryservice.NewDiscoveryService(
        discoveryRepo,
        propertyHooks,
        promotionHooks,
        c.Logger,
    )

    return nil
}
```

Add to NewContainer function:
```go
if err := c.initDiscovery(); err != nil {
    return nil, fmt.Errorf("failed to initialize discovery: %w", err)
}
```

Add imports:
```go
discoveryrepository "hauslet/internal/modules/discovery/repository"
discoveryservice "hauslet/internal/modules/discovery/service"
```

### 9. GraphQL Resolver Wiring

**File: `internal/transport/graph/resolver.go`**

Add to Resolver struct:
```go
DiscoveryResolver *discoverygraphql.Resolver
```

Add to NewResolver function:
```go
DiscoveryResolver: discoverygraphql.NewResolver(discoverySvc, log),
```

Add import:
```go
discoverygraphql "hauslet/internal/modules/discovery/port/graphql"
```

**File: `gqlgen.yml`**

Add to schema section:
```yaml
- internal/modules/discovery/port/graphql/*.graphqls
```

Add to models section:
```yaml
RankedListing:
  model: hauslet/internal/modules/discovery/domain.RankedListing
RankingScore:
  model: hauslet/internal/modules/discovery/domain.RankingScore
HomeFeedSection:
  model: hauslet/internal/modules/discovery/domain.HomeFeedSection
FeedSectionType:
  model: hauslet/internal/modules/discovery/domain.FeedSectionType
```

**Regenerate GraphQL code:**
```bash
go run github.com/99designs/gqlgen generate
```

### 10. Testing

Create test files:
- `service/ranking_test.go` - Test ranking algorithm
- `service/home_feed_test.go` - Test feed composition
- `service/adapters_test.go` - Test hook adapters
- `service/discovery_integration_test.go` - Integration tests

### 11. Migration Strategy

**Phase 1: Parallel Operation**
- Deploy Discovery module alongside existing Property search
- Both endpoints available (property.SearchListings + discovery.Discover)

**Phase 2: Gradual Migration**
- Update frontend to use Discovery endpoints
- Monitor performance and ranking quality

**Phase 3: Cleanup (Later)**
- Deprecate Property.SearchListings GraphQL query
- Remove SearchListings method from PropertyService
- Property module becomes pure CRUD

---

## Critical Files to Create/Modify

### New Files (Create)
1. `internal/modules/discovery/domain/discovery.go` - Core domain types
2. `internal/modules/discovery/domain/ranking.go` - Ranking logic
3. `internal/modules/discovery/service/interface.go` - Service + hook interfaces
4. `internal/modules/discovery/service/adapters.go` - Hook adapters
5. `internal/modules/discovery/service/service.go` - Service implementation
6. `internal/modules/discovery/service/ranking.go` - Ranking algorithm
7. `internal/modules/discovery/service/home_feed.go` - Home feed composition
8. `internal/modules/discovery/port/graphql/schema.graphqls` - GraphQL schema
9. `internal/modules/discovery/port/graphql/resolver.go` - GraphQL resolver
10. `internal/modules/discovery/repository/interface.go` - Repository interface
11. `internal/modules/discovery/repository/discovery_repository.go` - Repository impl

### Files to Modify
1. `/Users/ikwunna/Documents/hauslet-services/cmd/api/server/container.go` - Wire discovery service
2. `/Users/ikwunna/Documents/hauslet-services/internal/transport/graph/resolver.go` - Add discovery resolver
3. `/Users/ikwunna/Documents/hauslet-services/gqlgen.yml` - Add schema and model bindings

---

## Implementation Order

1. **Domain Layer** - Define core types (RankedListing, RankingScore, etc.)
2. **Service Interfaces** - Define DiscoveryService and hook interfaces
3. **Adapters** - Implement PropertyDiscoveryAdapter and PromotionDiscoveryAdapter
4. **Service Implementation** - Implement SearchListings with ranking algorithm
5. **Home Feed** - Implement GetHomeFeed composition logic
6. **Repository** - Implement basic repository (can stub for now)
7. **GraphQL Schema** - Define GraphQL types and queries
8. **GraphQL Resolver** - Implement resolver methods
9. **Container Wiring** - Wire everything together
10. **Testing** - Write unit and integration tests
11. **Build & Test** - Verify compilation and run tests

---

## Key Design Decisions

### Ranking Algorithm
- **Weights**: Semantic (0.4) + Promotion (0.3) + Recency (0.2) + Location (0.1)
- **Promotion Boost**: Featured (10x), Premium (3x), None (1x)
- **Recency Decay**: Linear decay from 1.0 (new) to 0.1 (90+ days)
- **Configurable**: RankingConfig can be customized per search

### Home Feed Composition
1. Featured (promoted listings, top 10)
2. Premium (promoted listings, top 10)
3. Recent (recently published, top 20)
4. Recommended (personalized, future)

### Personalization (Future)
- Store search history in discovery_search_history table
- Track clicked listings for CTR analysis
- Use for personalized recommendations (Phase 2)

### Caching Strategy (Future Optimization)
- Cache ranked results for popular searches (5 min TTL)
- Cache promotion boost data (1 min TTL)
- Cache featured/premium listing IDs (30 sec TTL)

---

## Success Criteria

✅ Discovery module compiles and initializes
✅ `discover` GraphQL query returns ranked results
✅ Promotion boosts correctly applied (featured 10x, premium 3x)
✅ `homeFeed` query returns multiple sections
✅ Property module CRUD operations unchanged
✅ No circular dependencies between modules
✅ Tests pass for ranking algorithm
✅ Performance acceptable (<500ms for search)
