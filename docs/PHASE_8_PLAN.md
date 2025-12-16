# Phase 8: Business-Property Integration Plan

## Current State Analysis

### ✅ What Already Exists

1. **Property Domain Model** (`internal/modules/property/domain/property.go`)
   - `Listing.OwnerID` (line 76) - Can be UserID or BusinessID
   - `Listing.OwnerType` (line 77) - Enum with values: "landlord", "agent", "business", "individual"
   - `Listing.CreatedBy` (line 102) - Currently `*uuid.UUID` (optional)

2. **Property Enums** (`internal/modules/property/domain/enums.go`)
   - `OwnerType` enum defined (lines 21-28)
   - Values: `OwnerLandlord`, `OwnerAgent`, `OwnerBusiness`, `OwnerIndividual`

3. **GraphQL Schema** (`internal/modules/property/port/graphql/schema.graphqls`)
   - `Listing.ownerType` field exists (line 181)
   - `Listing.createdBy` field exists (line 201)
   - `CreateListingInput.ownerType` exists (line 401)

4. **Business Service Updates** (User's recent changes)
   - `BusinessServiceImpl` now has `ProfileProvider` interface
   - Helper method `getProfileName()` to fetch user display names
   - `NotificationService` integrated

### ❌ What's Missing

1. **BusinessID field in CreateListingInput**
   - Currently CreateListing only uses viewer's UserID as ownerID
   - No way to specify which business should own the listing

2. **Permission Checks**
   - No validation that user has `CanCreateListings` permission in the business
   - No authorization checks for editing/deleting business-owned listings

3. **CreatedBy Population**
   - `CreatedBy` field exists but is not populated during creation
   - Need to track which team member created the listing

4. **Business Context in Resolvers**
   - Resolvers don't check business membership or permissions
   - No integration with business middleware

5. **Listing Queries by Business**
   - No easy way to filter listings by businessID
   - Need queries like `businessListings(businessId: UUID!)`

---

## Implementation Plan

### **Step 1: Update GraphQL Schema**

#### 1.1 Add BusinessID to CreateListingInput

**File**: `internal/modules/property/port/graphql/schema.graphqls`

```graphql
input CreateListingInput {
  ownerType: OwnerType!
  businessID: UUID  # NEW: Optional business ID for business-owned listings
  property: CreateListingPropertyInput!
  title: String!
  description: String!
  currency: CurrencyCode
  listingType: ListingType!

  hasCalendar: Boolean

  shortletDetails: ShortletDetailInput
  rentalDetails: RentalDetailInput
  saleDetails: SaleDetailInput
}
```

**Validation Rules**:
- If `ownerType == "business"`, then `businessID` is **required**
- If `ownerType == "individual"`, then `businessID` must be **null**

#### 1.2 Add Business-Specific Queries

```graphql
extend type Query {
  # Get listings for a specific business
  businessListings(
    businessId: UUID!
    status: ListingStatus
    limit: Int
    offset: Int
  ): [Listing!]!

  # Get user's individual listings only
  myIndividualListings(
    status: ListingStatus
    limit: Int
    offset: Int
  ): [Listing!]!
}
```

#### 1.3 Add CreatedByUser Field to Listing Type

```graphql
type Listing {
  id: UUID!
  # ... existing fields ...

  createdBy: UUID
  createdByUser: User  # NEW: Resolve to actual user profile

  # ... rest of fields ...
}
```

---

### **Step 2: Update Property Service**

#### 2.1 Add Business Service Dependency

**File**: `internal/modules/property/service/service.go`

```go
type ServiceImpl struct {
    repo          repository.Repository
    storage       *storage.R2Storage
    queueClient   *queue.Client
    queueSubject  string
    log           *lgr.Logger
    businessSvc   BusinessService  // NEW: Add business service
}

// BusinessService interface (subset needed by property module)
type BusinessService interface {
    HasPermission(ctx context.Context, userID, businessID uuid.UUID, permission string) (bool, error)
    GetBusiness(ctx context.Context, businessID uuid.UUID) (*BusinessInfo, error)
    IsMember(ctx context.Context, userID, businessID uuid.UUID) (bool, error)
}

// BusinessInfo minimal business info needed by property module
type BusinessInfo struct {
    ID   uuid.UUID
    Name string
}

// NewService updated constructor
func NewService(
    repo repository.Repository,
    storage *storage.R2Storage,
    queueClient *queue.Client,
    queueSubject string,
    businessSvc BusinessService,  // NEW parameter
    log *lgr.Logger,
) Service {
    return &ServiceImpl{
        repo:         repo,
        storage:      storage,
        queueClient:  queueClient,
        queueSubject: queueSubject,
        businessSvc:  businessSvc,  // NEW
        log:          log,
    }
}
```

#### 2.2 Add Permission Validation Method

**File**: `internal/modules/property/service/business_auth.go` (NEW FILE)

```go
package service

import (
    "context"
    "fmt"

    "hauslet/internal/modules/property/domain"

    "github.com/google/uuid"
)

// validateBusinessListingPermission checks if user can perform action on business listing
func (s *ServiceImpl) validateBusinessListingPermission(
    ctx context.Context,
    userID uuid.UUID,
    businessID uuid.UUID,
    permission string,
) error {
    if s.businessSvc == nil {
        return fmt.Errorf("business service not configured")
    }

    // Check if user is a member
    isMember, err := s.businessSvc.IsMember(ctx, userID, businessID)
    if err != nil {
        return fmt.Errorf("failed to check business membership: %w", err)
    }
    if !isMember {
        return domain.ErrNotBusinessMember
    }

    // Check specific permission
    hasPermission, err := s.businessSvc.HasPermission(ctx, userID, businessID, permission)
    if err != nil {
        return fmt.Errorf("failed to check permission: %w", err)
    }
    if !hasPermission {
        return domain.ErrInsufficientPermissions
    }

    return nil
}

// canCreateBusinessListing checks if user can create a listing for the business
func (s *ServiceImpl) canCreateBusinessListing(
    ctx context.Context,
    userID uuid.UUID,
    businessID uuid.UUID,
) error {
    return s.validateBusinessListingPermission(ctx, userID, businessID, "CanCreateListings")
}

// canEditBusinessListing checks if user can edit a business listing
func (s *ServiceImpl) canEditBusinessListing(
    ctx context.Context,
    userID uuid.UUID,
    businessID uuid.UUID,
) error {
    return s.validateBusinessListingPermission(ctx, userID, businessID, "CanEditListings")
}

// canDeleteBusinessListing checks if user can delete a business listing
func (s *ServiceImpl) canDeleteBusinessListing(
    ctx context.Context,
    userID uuid.UUID,
    businessID uuid.UUID,
) error {
    return s.validateBusinessListingPermission(ctx, userID, businessID, "CanDeleteListings")
}

// canPublishBusinessListing checks if user can publish a business listing
func (s *ServiceImpl) canPublishBusinessListing(
    ctx context.Context,
    userID uuid.UUID,
    businessID uuid.UUID,
) error {
    return s.validateBusinessListingPermission(ctx, userID, businessID, "CanPublishListings")
}
```

#### 2.3 Update Domain Errors

**File**: `internal/modules/property/domain/errors.go`

Add these errors:
```go
var (
    // ... existing errors ...

    // Business-related errors
    ErrNotBusinessMember       = errors.New("user is not a member of this business")
    ErrInsufficientPermissions = errors.New("user lacks required permissions")
    ErrBusinessRequired        = errors.New("businessID required for business-owned listings")
    ErrBusinessNotAllowed      = errors.New("businessID not allowed for individual listings")
)
```

#### 2.4 Update CreateListing Method

**File**: `internal/modules/property/service/listing_crud.go`

```go
// CreateListingWithContext creates a listing with business context awareness
func (s *ServiceImpl) CreateListingWithContext(
    ctx context.Context,
    l domain.Listing,
    creatorUserID uuid.UUID,  // NEW: The actual user creating it
) (*domain.Listing, error) {
    // Validate basic fields
    if l.PropertyID == uuid.Nil {
        return nil, domain.ErrInvalidPropertyID
    }
    if l.OwnerID == uuid.Nil {
        return nil, domain.ErrInvalidOwnerID
    }

    // Business-specific validation
    if l.OwnerType == domain.OwnerBusiness {
        // For business listings, OwnerID = BusinessID
        businessID := l.OwnerID

        // Check permission
        if err := s.canCreateBusinessListing(ctx, creatorUserID, businessID); err != nil {
            s.log.Logf("ERROR User %s cannot create listing for business %s: %v",
                creatorUserID, businessID, err)
            return nil, err
        }

        s.log.Logf("INFO User %s creating business listing for business %s",
            creatorUserID, businessID)
    } else if l.OwnerType == domain.OwnerIndividual {
        // For individual listings, OwnerID must be the user's ID
        if l.OwnerID != creatorUserID {
            return nil, fmt.Errorf("individual listing ownerID must match creator")
        }
    }

    // Set CreatedBy to track who created it
    l.CreatedBy = &creatorUserID

    // Ensure property exists
    exists, err := s.repo.PropertyExists(ctx, l.PropertyID)
    if err != nil {
        return nil, err
    }
    if !exists {
        return nil, domain.ErrPropertyNotFound
    }

    // Normalize media slice
    if l.Media == nil {
        l.Media = []domain.ListingMedia{}
    }

    schemaListing := domain.MapListingToSchema(&l)
    if schemaListing.ID == uuid.Nil {
        schemaListing.ID = uuid.New()
    }
    if schemaListing.Slug == "" {
        schemaListing.Slug = generateSlug(l.Title) + "-" + shortid()
    }

    if err := s.repo.CreateListing(ctx, schemaListing); err != nil {
        return nil, err
    }

    created := domain.MapListingFromSchema(schemaListing)

    s.log.Logf("INFO Listing %s created successfully (owner_type=%s, owner_id=%s, created_by=%s)",
        created.ID, created.OwnerType, created.OwnerID, creatorUserID)

    return created, nil
}
```

#### 2.5 Update UpdateListing Method

Add permission checks:

```go
func (s *ServiceImpl) UpdateListingWithContext(
    ctx context.Context,
    l domain.Listing,
    updaterUserID uuid.UUID,
) (*domain.Listing, error) {
    // ... existing validation ...

    // Get existing listing to check ownership
    existing, err := s.ensureListing(ctx, l.ID, false)
    if err != nil {
        return nil, err
    }

    // Authorization check
    if existing.OwnerType == domain.OwnerBusiness {
        businessID := existing.OwnerID
        if err := s.canEditBusinessListing(ctx, updaterUserID, businessID); err != nil {
            return nil, err
        }
    } else {
        // Individual listing - must be the owner
        if existing.OwnerID != updaterUserID {
            return nil, fmt.Errorf("unauthorized: not the listing owner")
        }
    }

    // Set updatedBy
    l.UpdatedBy = &updaterUserID

    // ... rest of update logic ...
}
```

#### 2.6 Add Business Listing Query Methods

**File**: `internal/modules/property/service/listing_query.go`

```go
// GetBusinessListings returns all listings for a business
func (s *ServiceImpl) GetBusinessListings(
    ctx context.Context,
    businessID uuid.UUID,
    filters map[string]interface{},
    limit, offset int,
) ([]domain.Listing, error) {
    // Verify business exists (optional, for better error messages)
    if s.businessSvc != nil {
        _, err := s.businessSvc.GetBusiness(ctx, businessID)
        if err != nil {
            return nil, fmt.Errorf("business not found: %w", err)
        }
    }

    // Build filters
    if filters == nil {
        filters = make(map[string]interface{})
    }
    filters["owner_type"] = domain.OwnerBusiness
    filters["owner_id"] = businessID

    return s.repo.ListListings(ctx, filters, limit, offset)
}

// GetUserIndividualListings returns only user's individual listings
func (s *ServiceImpl) GetUserIndividualListings(
    ctx context.Context,
    userID uuid.UUID,
    filters map[string]interface{},
    limit, offset int,
) ([]domain.Listing, error) {
    if filters == nil {
        filters = make(map[string]interface{})
    }
    filters["owner_type"] = domain.OwnerIndividual
    filters["owner_id"] = userID

    return s.repo.ListListings(ctx, filters, limit, offset)
}
```

---

### **Step 3: Update GraphQL Resolvers**

#### 3.1 Update CreateListing Resolver

**File**: `internal/modules/property/port/graphql/resolvers.go`

```go
func (r *Resolver) CreateListing(ctx context.Context, input model.CreateListingInput) (*domain.Listing, error) {
    v := viewer.FromContext(ctx)
    if v == nil || v.UserID == "" {
        r.log.Logf("WARN Unauthenticated attempt to create listing")
        return nil, fmt.Errorf("unauthenticated")
    }

    creatorUserID, err := uuid.Parse(v.UserID)
    if err != nil {
        r.log.Logf("ERROR Invalid user ID in createListing: %s", v.UserID)
        return nil, fmt.Errorf("invalid user ID")
    }

    if input.Property == nil {
        r.log.Logf("WARN CreateListing called without property payload by user %s", v.UserID)
        return nil, fmt.Errorf("property payload is required")
    }

    // Determine ownerID based on owner type
    var ownerID uuid.UUID
    var propertyOwnerID uuid.UUID

    switch input.OwnerType {
    case domain.OwnerBusiness:
        // For business listings
        if input.BusinessID == nil {
            return nil, fmt.Errorf("businessID required for business-owned listings")
        }
        businessID, err := uuid.Parse(*input.BusinessID)
        if err != nil {
            return nil, fmt.Errorf("invalid businessID")
        }
        ownerID = businessID
        propertyOwnerID = businessID

    case domain.OwnerIndividual:
        // For individual listings
        if input.BusinessID != nil {
            return nil, fmt.Errorf("businessID not allowed for individual listings")
        }
        ownerID = creatorUserID
        propertyOwnerID = creatorUserID

    default:
        // Legacy: landlord, agent
        ownerID = creatorUserID
        propertyOwnerID = creatorUserID
    }

    // Map inputs to domain models
    property := mapCreateListingPropertyInput(input.Property, propertyOwnerID)
    listing := mapCreateListingInput(input, ownerID)

    // Create property and listing atomically
    createdProperty, createdListing, err := r.propertyService.CreatePropertyWithListingContext(
        ctx,
        *property,
        listing,
        creatorUserID,  // Pass the actual creator
    )
    if err != nil {
        r.log.Logf("ERROR Failed to create property with listing for user %s: %v", v.UserID, err)
        return nil, err
    }

    r.log.Logf("INFO Property %s and listing %s created successfully by user %s (owner_type=%s)",
        createdProperty.ID, createdListing.ID, v.UserID, input.OwnerType)

    return sanitizeListingForViewer(createdListing, v), nil
}
```

#### 3.2 Add New Query Resolvers

```go
// BusinessListings returns listings for a specific business
func (r *Resolver) BusinessListings(
    ctx context.Context,
    businessID uuid.UUID,
    status *domain.ListingStatus,
    limit *int,
    offset *int,
) ([]*domain.Listing, error) {
    filters := make(map[string]interface{})
    if status != nil {
        filters["status"] = *status
    }

    l := intOrDefault(limit, 20)
    o := intOrDefault(offset, 0)

    listings, err := r.propertyService.GetBusinessListings(ctx, businessID, filters, l, o)
    if err != nil {
        r.log.Logf("ERROR Failed to get business listings: %v", err)
        return nil, err
    }

    result := make([]*domain.Listing, len(listings))
    for i := range listings {
        result[i] = &listings[i]
    }
    return result, nil
}

// MyIndividualListings returns only the user's individual listings
func (r *Resolver) MyIndividualListings(
    ctx context.Context,
    status *domain.ListingStatus,
    limit *int,
    offset *int,
) ([]*domain.Listing, error) {
    v := viewer.FromContext(ctx)
    if v == nil || v.UserID == "" {
        return nil, fmt.Errorf("unauthenticated")
    }

    userID, err := uuid.Parse(v.UserID)
    if err != nil {
        return nil, fmt.Errorf("invalid user ID")
    }

    filters := make(map[string]interface{})
    if status != nil {
        filters["status"] = *status
    }

    l := intOrDefault(limit, 20)
    o := intOrDefault(offset, 0)

    listings, err := r.propertyService.GetUserIndividualListings(ctx, userID, filters, l, o)
    if err != nil {
        r.log.Logf("ERROR Failed to get individual listings: %v", err)
        return nil, err
    }

    result := make([]*domain.Listing, len(listings))
    for i := range listings {
        result[i] = &listings[i]
    }
    return result, nil
}
```

#### 3.3 Add CreatedByUser Field Resolver

```go
// CreatedByUser resolves the user who created the listing
func (r *listingResolver) CreatedByUser(ctx context.Context, obj *domain.Listing) (*profiledomain.Profile, error) {
    if obj.CreatedBy == nil {
        return nil, nil
    }

    // Use profile service to get user details
    // This would need profile service integration
    // For now, return placeholder
    return nil, fmt.Errorf("not implemented: need profile service integration")
}
```

---

### **Step 4: Update Service Interface**

**File**: `internal/modules/property/service/interface.go`

Add new methods:
```go
type Service interface {
    // ... existing methods ...

    // Business-aware methods
    CreateListingWithContext(ctx context.Context, l domain.Listing, creatorUserID uuid.UUID) (*domain.Listing, error)
    UpdateListingWithContext(ctx context.Context, l domain.Listing, updaterUserID uuid.UUID) (*domain.Listing, error)
    GetBusinessListings(ctx context.Context, businessID uuid.UUID, filters map[string]interface{}, limit, offset int) ([]domain.Listing, error)
    GetUserIndividualListings(ctx context.Context, userID uuid.UUID, filters map[string]interface{}, limit, offset int) ([]domain.Listing, error)
}
```

---

### **Step 5: Update Initialization in routes.go**

**File**: `cmd/api/server/routes.go`

```go
// Initialize property service with business service dependency
propertyRepo := propertyrepository.NewPropertyRepository(db)
thumbnailSubject := cfg.YAML.Queue.Subjects["media_thumbnail"]
propertyService := propertyservice.NewPropertyService(
    propertyRepo,
    r2,
    q,
    thumbnailSubject,
    businessService,  // NEW: Pass business service
    log,
)
```

---

### **Step 6: Update Database Migration (If Needed)**

If `created_by` field needs to be made NOT NULL or indexed:

```sql
-- Make created_by required for new listings
ALTER TABLE listings
ALTER COLUMN created_by SET NOT NULL;

-- Add index for querying by creator
CREATE INDEX idx_listings_created_by ON listings(created_by);

-- Add compound index for business listings
CREATE INDEX idx_listings_business ON listings(owner_type, owner_id)
WHERE owner_type = 'business';
```

---

## Testing Checklist

### Unit Tests
- [ ] Test `canCreateBusinessListing` with valid permission
- [ ] Test `canCreateBusinessListing` with invalid permission
- [ ] Test `canCreateBusinessListing` with non-member
- [ ] Test individual listing creation (existing flow)
- [ ] Test business listing creation with permission
- [ ] Test business listing creation without permission
- [ ] Test businessID validation (required for business, not allowed for individual)

### Integration Tests
- [ ] Create individual listing (should work as before)
- [ ] Create business listing as owner (should succeed)
- [ ] Create business listing as admin with permission (should succeed)
- [ ] Create business listing as member without permission (should fail)
- [ ] Create business listing as non-member (should fail)
- [ ] Update business listing with permission (should succeed)
- [ ] Update business listing without permission (should fail)
- [ ] Delete business listing with permission (should succeed)
- [ ] Delete business listing without permission (should fail)
- [ ] Query `businessListings` for a business
- [ ] Query `myIndividualListings` for a user
- [ ] Verify `createdBy` is populated correctly

### GraphQL Tests
```graphql
# Test 1: Create individual listing
mutation {
  createListing(input: {
    ownerType: individual
    property: { ... }
    title: "My Personal Apartment"
    description: "..."
    listingType: rent
  }) {
    id
    ownerType
    ownerId
    createdBy
  }
}

# Test 2: Create business listing
mutation {
  createListing(input: {
    ownerType: business
    businessID: "uuid-of-business"
    property: { ... }
    title: "Company Rental"
    description: "..."
    listingType: rent
  }) {
    id
    ownerType
    ownerId
    createdBy
  }
}

# Test 3: Query business listings
query {
  businessListings(businessId: "uuid-of-business") {
    id
    title
    createdBy
    createdByUser {
      name
      email
    }
  }
}

# Test 4: Query my individual listings
query {
  myIndividualListings {
    id
    title
    ownerType
  }
}
```

---

## Migration Strategy

### Phase 8a: Foundation (Non-Breaking)
1. ✅ Add `businessID` field to GraphQL input (optional)
2. ✅ Add new query methods (`businessListings`, `myIndividualListings`)
3. ✅ Add business service dependency to property service
4. ✅ Add permission validation helpers
5. ✅ Ensure `createdBy` is populated on new listings

**Impact**: Zero - existing functionality continues to work

### Phase 8b: Business Integration (Feature Addition)
1. ✅ Update `CreateListing` resolver to handle business context
2. ✅ Add permission checks for business listings
3. ✅ Update `UpdateListing` to check business permissions
4. ✅ Update `DeleteListing` to check business permissions

**Impact**: Low - only affects business listing creation (new feature)

### Phase 8c: Enhanced Queries (Optional)
1. ✅ Add `createdByUser` field resolver (needs profile service)
2. ✅ Add advanced business listing filters
3. ✅ Add analytics for business listings

**Impact**: Zero - new optional fields

---

## API Changes Summary

### New GraphQL Fields
```graphql
input CreateListingInput {
  businessID: UUID  # NEW: Optional
}

extend type Query {
  businessListings(businessId: UUID!, ...): [Listing!]!  # NEW
  myIndividualListings(...): [Listing!]!  # NEW
}

type Listing {
  createdByUser: User  # NEW: Resolves creator profile
}
```

### Behavior Changes
1. **Create Listing**: Now requires permission check for business-owned listings
2. **Update Listing**: Now requires permission check for business-owned listings
3. **Delete Listing**: Now requires permission check for business-owned listings
4. **CreatedBy**: Now automatically populated with creator's user ID

### Error Responses
New error codes:
- `NOT_BUSINESS_MEMBER`: User is not a member of the business
- `INSUFFICIENT_PERMISSIONS`: User lacks required permission
- `BUSINESS_REQUIRED`: businessID required for business-owned listings
- `BUSINESS_NOT_ALLOWED`: businessID not allowed for individual listings

---

## Rollback Plan

If issues arise:

1. **Quick Rollback**: Comment out permission checks in CreateListing resolver
2. **Database Rollback**: Revert migration if created_by constraints added
3. **Service Rollback**: Property service can function without business service (graceful degradation)

---

## Questions for Review

1. **Profile Service Integration**: Do we need to integrate with profile service for `createdByUser` resolver? Or should we defer this?

2. **Backward Compatibility**: Should we support creating business listings without permission checks temporarily (with deprecation warning)?

3. **Migration**: Do existing listings need to be updated with `created_by` values, or only new ones?

4. **Permission Caching**: Should we cache permission checks to reduce database queries?

5. **Soft Delete**: How should soft-deleted business listings behave when business is deleted?

6. **Transfer Listings**: Should we add a mutation to transfer listings between individual and business ownership?

---

## Files to Create/Modify

### New Files
- [ ] `internal/modules/property/service/business_auth.go` - Permission validation
- [ ] `internal/modules/property/service/business_auth_test.go` - Tests

### Modified Files
- [ ] `internal/modules/property/port/graphql/schema.graphqls` - Add businessID field and new queries
- [ ] `internal/modules/property/port/graphql/resolvers.go` - Update CreateListing, add new queries
- [ ] `internal/modules/property/port/graphql/helpers.go` - Update input mapping
- [ ] `internal/modules/property/service/interface.go` - Add new methods
- [ ] `internal/modules/property/service/service.go` - Add business service dependency
- [ ] `internal/modules/property/service/listing_crud.go` - Add context-aware methods
- [ ] `internal/modules/property/service/listing_query.go` - Add business query methods
- [ ] `internal/modules/property/domain/errors.go` - Add business-related errors
- [ ] `cmd/api/server/routes.go` - Wire business service to property service
- [ ] `internal/transport/graph/schema.resolvers.go` - Delegate new queries

### Documentation
- [ ] Update BUSINESS_ARCHITECTURE.md with property integration examples
- [ ] Update API documentation with new mutations/queries
- [ ] Create migration guide for frontend

---

## Success Criteria

Phase 8 is complete when:

1. ✅ Users can create listings owned by businesses they're members of
2. ✅ Permission checks prevent unauthorized business listing creation
3. ✅ Permission checks prevent unauthorized business listing editing/deletion
4. ✅ `createdBy` field is populated for all new listings
5. ✅ Business-specific queries work correctly
6. ✅ Individual listing flow remains unchanged
7. ✅ All tests pass
8. ✅ Documentation is updated
9. ✅ Frontend can distinguish between individual and business listings
10. ✅ No breaking changes to existing API

---

## Timeline Estimate

- **Step 1**: Schema updates - 30 minutes
- **Step 2**: Service layer updates - 2 hours
- **Step 3**: Resolver updates - 1.5 hours
- **Step 4**: Interface updates - 15 minutes
- **Step 5**: Initialization updates - 15 minutes
- **Step 6**: Migration (if needed) - 30 minutes
- **Testing**: 2 hours
- **Documentation**: 1 hour

**Total**: ~8 hours of development work
