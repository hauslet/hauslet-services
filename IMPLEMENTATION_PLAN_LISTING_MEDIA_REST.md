# Implementation Plan: REST Handlers for Listing Media

## Overview
This plan outlines the implementation of REST API endpoints to consume the listing media methods from `internal/property/service/interface.go`. The implementation will follow the existing architectural patterns observed in the auth module.

## Current State Analysis

### Existing Listing Media Service Methods
From `internal/property/service/interface.go`:
1. `UploadListingMedia(ctx, listingID, media)` - Generates signed upload URLs for media
2. `UpdateListingMedia(ctx, listingID, mediaID, updates)` - Updates media metadata
3. `DeleteListingMedia(ctx, listingID, media)` - Deletes media files
4. `ListListingMedia(ctx, listingID)` - Lists all media for a listing
5. `FinalizeListingMedia(ctx, data)` - Finalizes media after upload completion

### Architectural Patterns Observed
From `internal/auth/port/http/`:
- **Handler Structure**: HTTPHandler struct with service, context, and logger dependencies
- **Helper Methods**: `sendError()` and `sendSuccess()` for standardized responses
- **DTOs**: Request/Response types in domain package (`domain/dto.go`)
- **Routing**: Separate `routes.go` file with `SetupRoutes()` and optional `SetupRoutesWithRateLimiting()`
- **Handlers**: Individual files per functionality group (e.g., `register.go`, `user.go`)
- **Authentication**: Uses `go-pkgz/auth` middleware with `token.GetUserInfo(r)` for user extraction
- **Validation**: Domain-level validation with `ValidationError` type
- **Error Handling**: Standardized `ErrorResponse` with optional field-level errors

### Technology Stack
- **Router**: Chi (github.com/go-chi/chi/v5)
- **Auth**: go-pkgz/auth with JWT tokens
- **Logging**: go-pkgz/lgr
- **Storage**: R2Storage with presigned URLs
- **Queue**: NATS (for thumbnail generation jobs)

## Implementation Plan

### Phase 1: Domain Layer DTOs

**File**: `internal/property/domain/dto.go` (new file)

Create request/response DTOs following the auth module pattern:

```go
// Request DTOs
- UploadMediaRequest
  - media []MediaUploadInput
  - Fields: type, group, caption, mime_type, size_bytes, is_primary, is_group_cover, order, filename, duration

- UpdateMediaRequest
  - caption *string
  - is_primary *bool
  - is_group_cover *bool
  - order *int

- DeleteMediaRequest
  - media []MediaDeleteInput
  - Fields: media_id, key

- FinalizeMediaRequest
  - media_keys []string

// Response DTOs
- UploadMediaResponse
  - media []MediaUploadResult
  - Fields: id, filename, url, key

- MediaResponse
  - id, listing_id, url, type, thumbnails, group, caption, mime_type, size_bytes
  - is_primary, is_group_cover, order, created_at, updated_at

- ListMediaResponse
  - media []MediaResponse
  - total int

- SuccessResponse
  - message string
```

**Validation Methods**: Add `Validate()` methods to each request DTO (similar to `RegisterRequest.Validate()`)

### Phase 2: HTTP Handler Structure

**Directory**: `internal/property/port/http/` (new directory)

#### File 1: `handler.go`
```go
package http

type HTTPHandler struct {
    propertyService service.Service
    ctx             context.Context
    log             *lgr.Logger
}

func NewHTTPHandler(ctx context.Context, svc service.Service, log *lgr.Logger) *HTTPHandler

// Helper methods
func (h *HTTPHandler) sendError(w, message, statusCode, field)
func (h *HTTPHandler) sendSuccess(w, data, statusCode)
func (h *HTTPHandler) extractUserID(r) (uuid.UUID, error) // Extract from JWT
```

#### File 2: `routes.go`
```go
package http

func (h *HTTPHandler) SetupRoutes(r chi.Router)
func (h *HTTPHandler) SetupRoutesWithRateLimiting(r chi.Router, redisClient)

// Route structure:
// POST   /api/listings/:id/media          - Upload media (get presigned URLs)
// GET    /api/listings/:id/media          - List all media
// PATCH  /api/listings/:id/media/:mediaId - Update media metadata
// DELETE /api/listings/:id/media          - Delete media (bulk)
// POST   /api/listings/:id/media/finalize - Finalize uploaded media
```

**Authentication**: All routes require authentication via `authMiddleware.Auth`

**Rate Limits** (production only):
- Upload: 20 requests/minute
- Update: 30 requests/minute
- Delete: 20 requests/minute
- List: 100 requests/minute
- Finalize: 20 requests/minute

#### File 3: `media.go`
```go
package http

// Handler methods
func (h *HTTPHandler) UploadListingMedia(w, r)
func (h *HTTPHandler) ListListingMedia(w, r)
func (h *HTTPHandler) UpdateListingMedia(w, r)
func (h *HTTPHandler) DeleteListingMedia(w, r)
func (h *HTTPHandler) FinalizeListingMedia(w, r)
```

### Phase 3: Handler Implementation Details

#### UploadListingMedia
- **Method**: POST
- **Path**: `/api/listings/:id/media`
- **Auth**: Required
- **Request**: JSON body with `UploadMediaRequest`
- **Process**:
  1. Extract and validate listing ID from URL params
  2. Decode and validate request body
  3. Extract user ID from JWT token
  4. Verify user owns the listing (authorization check)
  5. Call `service.UploadListingMedia()`
  6. Return presigned URLs with 201 status
- **Errors**: 400 (invalid input), 401 (unauthorized), 403 (forbidden), 404 (listing not found), 500 (server error)

#### ListListingMedia
- **Method**: GET
- **Path**: `/api/listings/:id/media`
- **Auth**: Required
- **Process**:
  1. Extract and validate listing ID from URL params
  2. Extract user ID from JWT token
  3. Verify user can access the listing (owner or public listing)
  4. Call `service.ListListingMedia()`
  5. Return media list with 200 status
- **Errors**: 401, 403, 404, 500

#### UpdateListingMedia
- **Method**: PATCH
- **Path**: `/api/listings/:id/media/:mediaId`
- **Auth**: Required
- **Request**: JSON body with `UpdateMediaRequest`
- **Process**:
  1. Extract and validate listing ID and media ID from URL params
  2. Decode and validate request body
  3. Extract user ID from JWT token
  4. Verify user owns the listing
  5. Call `service.UpdateListingMedia()`
  6. Return success message with 200 status
- **Errors**: 400, 401, 403, 404, 500

#### DeleteListingMedia
- **Method**: DELETE
- **Path**: `/api/listings/:id/media`
- **Auth**: Required
- **Request**: JSON body with `DeleteMediaRequest`
- **Process**:
  1. Extract and validate listing ID from URL params
  2. Decode and validate request body
  3. Extract user ID from JWT token
  4. Verify user owns the listing
  5. Call `service.DeleteListingMedia()`
  6. Return success message with 200 status
- **Errors**: 400, 401, 403, 404, 500

#### FinalizeListingMedia
- **Method**: POST
- **Path**: `/api/listings/:id/media/finalize`
- **Auth**: Required
- **Request**: JSON body with `FinalizeMediaRequest`
- **Process**:
  1. Extract and validate listing ID from URL params
  2. Decode and validate request body
  3. Extract user ID from JWT token
  4. Verify user owns the listing
  5. Build `FinalizedListingMedia` domain object
  6. Call `service.FinalizeListingMedia()`
  7. Trigger thumbnail generation job (via service)
  8. Return success message with 200 status
- **Errors**: 400, 401, 403, 404, 500

### Phase 4: Authorization Helper

**File**: `internal/property/port/http/authorization.go` (new file)

```go
// Helper function to verify listing ownership
func (h *HTTPHandler) verifyListingOwnership(ctx, listingID, userID) error
  - Get listing by ID
  - Check if listing.OwnerID == userID
  - Return domain.ErrForbidden if not owner
```

### Phase 5: Integration with Server

**File**: `cmd/api/server/routes.go`

Update `setupRoutes()`:
```go
// Add after property service initialization
propertyHTTP := propertyport.NewHTTPHandler(ctx, propertyService, log)

// Setup property routes
r.Route("/api", func(r chi.Router) {
    if cfg.App.Env == "production" {
        propertyHTTP.SetupRoutesWithRateLimiting(r, *rds)
    } else {
        propertyHTTP.SetupRoutes(r)
    }
})
```

### Phase 6: Error Handling Updates

**File**: `internal/property/domain/errors.go`

Add missing errors if needed:
```go
var (
    ErrForbidden = errors.New("forbidden: insufficient permissions")
    ErrUnauthorized = errors.New("unauthorized")
    ErrInvalidMediaInput = errors.New("invalid media input")
    ErrMediaUploadFailed = errors.New("media upload failed")
)
```

## File Structure Summary

```
internal/property/
├── domain/
│   ├── dto.go          (new - request/response DTOs)
│   ├── errors.go       (update - add new errors)
│   └── property.go     (existing)
├── port/
│   ├── http/           (new directory)
│   │   ├── handler.go     (new - HTTPHandler struct and helpers)
│   │   ├── routes.go      (new - route setup)
│   │   ├── media.go       (new - media handlers)
│   │   └── authorization.go (new - ownership verification)
│   └── graphql/        (existing)
├── service/            (existing)
└── repository/         (existing)

cmd/api/server/
└── routes.go          (update - integrate property HTTP handlers)
```

## Testing Strategy

### Unit Tests
**File**: `internal/property/port/http/media_test.go`

Test cases:
1. Upload media - success
2. Upload media - invalid listing ID
3. Upload media - unauthorized user
4. Upload media - user doesn't own listing
5. List media - success
6. List media - listing not found
7. Update media - success
8. Update media - media not found
9. Delete media - success
10. Delete media - bulk deletion
11. Finalize media - success
12. Finalize media - object not found in storage

### Integration Tests
- Test complete upload flow (upload -> finalize)
- Test media ordering updates
- Test primary media switching
- Test group cover management
- Test concurrent operations

## Security Considerations

1. **Authentication**: All endpoints require valid JWT token
2. **Authorization**: Verify user owns the listing before any mutation
3. **Input Validation**: Validate all inputs at domain layer
4. **File Size Limits**: Enforce max file size (already handled by domain.ListingMediaInput)
5. **Rate Limiting**: Apply rate limits in production to prevent abuse
6. **CORS**: Already configured at server level
7. **Presigned URL Expiry**: 15min for images, 30min for videos (already in service)

## Migration Path

No database migrations required - all schemas already exist.

## Rollout Plan

1. **Development**:
   - Implement Phase 1-6
   - Unit and integration tests
   - Manual testing with Postman/curl

2. **Staging**:
   - Deploy to staging environment
   - End-to-end testing
   - Performance testing

3. **Production**:
   - Deploy with rate limiting enabled
   - Monitor error rates and latency
   - Monitor NATS queue for thumbnail jobs

## API Documentation

Add Swagger/OpenAPI annotations to each handler (following auth module pattern):
```go
// @Summary Upload listing media
// @Description Generate presigned upload URLs for listing media
// @Tags listings
// @Accept json
// @Produce json
// @Param id path string true "Listing ID"
// @Param request body domain.UploadMediaRequest true "Media upload details"
// @Success 201 {object} domain.UploadMediaResponse
// @Failure 400 {object} domain.ErrorResponse
// @Router /api/listings/{id}/media [post]
// @Security BearerAuth
```

## Open Questions

1. **Public Access**: Should `ListListingMedia` be accessible for public (published) listings without authentication?
   - **Recommendation**: Yes, add a conditional check - if listing is published, allow public access. Otherwise, require ownership.
  - **User Answer** : DO NOT PUBLISH THIS ROUTE I WILL EXPOSE IT VIA GRAPH

2. **Bulk Operations**: Should there be a limit on bulk delete operations?
   - **Recommendation**: Yes, limit to 50 media items per delete request to prevent abuse.

3. **Media Ordering**: Should the API auto-manage media ordering (e.g., auto-increment)?
   - **Recommendation**: No, client should manage ordering explicitly. Service validates uniqueness within listing.
   - **User Answer** : Use Recommendation

4. **Webhook Notifications**: Should media finalization trigger webhooks?
   - **Recommendation**: Not in MVP. Add later if needed for client-side progress tracking.
  - **User Answer** : Use Recommendation

## Success Criteria

- [ ] All 5 endpoints implemented and tested
- [ ] Request/Response DTOs created with validation
- [ ] Authorization properly enforced
- [ ] Rate limiting configured for production
- [ ] Unit test coverage > 80%
- [ ] Integration tests passing
- [ ] API documentation complete
- [ ] No breaking changes to existing GraphQL API
- [ ] Thumbnail generation jobs triggered successfully

## Estimated Complexity

- **LOC**: ~800-1000 lines
- **Files**: 7 new files, 2 updated files
- **Time**: 1-2 days for implementation + testing
