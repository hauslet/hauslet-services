package middleware

import (
	"fmt"
	"net/http"
	"regexp"

	"hauslet/internal/modules/business/domain"
	"hauslet/internal/modules/business/service"
	"hauslet/internal/transport/graph/viewer"

	"github.com/go-chi/chi/v5"
	"github.com/go-pkgz/lgr"
	"github.com/google/uuid"
)

// BusinessAuthMiddleware handles business-level authorization
type BusinessAuthMiddleware struct {
	businessService service.BusinessService
	log             *lgr.Logger
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// NewBusinessAuthMiddleware creates a new business authorization middleware
func NewBusinessAuthMiddleware(businessService service.BusinessService, log *lgr.Logger) *BusinessAuthMiddleware {
	return &BusinessAuthMiddleware{
		businessService: businessService,
		log:             log,
	}
}

// RequireBusinessMember ensures the user is a member of the specified business
// Expects businessID to be in the URL parameters as "businessID"
func (m *BusinessAuthMiddleware) RequireBusinessMember(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get viewer from context
		v := viewer.FromContext(ctx)
		if v == nil || v.UserID == "" {
			m.log.Logf("WARN Unauthenticated request to business resource")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get business ID from URL
		businessIDStr := chi.URLParam(r, "businessID")
		if businessIDStr == "" {
			m.log.Logf("WARN Missing businessID in URL")
			http.Error(w, "Business ID required", http.StatusBadRequest)
			return
		}

		businessID, err := uuid.Parse(businessIDStr)
		if err != nil {
			m.log.Logf("WARN Invalid business ID: %v", err)
			http.Error(w, "Invalid business ID", http.StatusBadRequest)
			return
		}

		userID, err := uuid.Parse(v.UserID)
		if err != nil {
			m.log.Logf("ERROR Invalid user ID: %v", err)
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		// Check membership
		membership, err := m.businessService.GetMember(ctx, businessID, userID)
		if err != nil {
			if err == domain.ErrMemberNotFound {
				m.log.Logf("WARN User %s is not a member of business %s", userID, businessID)
				http.Error(w, "Forbidden: Not a business member", http.StatusForbidden)
				return
			}
			m.log.Logf("ERROR Failed to get membership: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Get business
		business, err := m.businessService.GetBusiness(ctx, businessID)
		if err != nil {
			m.log.Logf("ERROR Failed to get business: %v", err)
			http.Error(w, "Business not found", http.StatusNotFound)
			return
		}

		// Add to context
		bc := &BusinessContext{
			BusinessID:  businessID,
			Business:    business,
			Membership:  membership,
			Permissions: &membership.Permissions,
			IsOwner:     membership.IsOwner(),
			IsAdmin:     membership.IsAdmin(),
		}

		ctx = WithBusinessContext(ctx, bc)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireBusinessOwner ensures the user is an owner of the specified business
func (m *BusinessAuthMiddleware) RequireBusinessOwner(next http.Handler) http.Handler {
	return m.RequireBusinessMember(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		bc, ok := GetBusinessContext(ctx)
		if !ok {
			m.log.Logf("ERROR Business context not found")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if !bc.IsOwner {
			m.log.Logf("WARN User is not an owner of business %s", bc.BusinessID)
			http.Error(w, "Forbidden: Owner access required", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}))
}

// RequireBusinessAdmin ensures the user is an admin or owner of the specified business
func (m *BusinessAuthMiddleware) RequireBusinessAdmin(next http.Handler) http.Handler {
	return m.RequireBusinessMember(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		bc, ok := GetBusinessContext(ctx)
		if !ok {
			m.log.Logf("ERROR Business context not found")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if !bc.IsAdmin && !bc.IsOwner {
			m.log.Logf("WARN User is not an admin/owner of business %s", bc.BusinessID)
			http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}))
}

// RequirePermission ensures the user has the specified permission
func (m *BusinessAuthMiddleware) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return m.RequireBusinessMember(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			bc, ok := GetBusinessContext(ctx)
			if !ok {
				m.log.Logf("ERROR Business context not found")
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Owners and admins have all permissions
			if bc.IsOwner || bc.IsAdmin {
				next.ServeHTTP(w, r)
				return
			}

			// Check specific permission
			if bc.Permissions == nil {
				m.log.Logf("WARN No permissions found for user in business %s", bc.BusinessID)
				http.Error(w, "Forbidden: Insufficient permissions", http.StatusForbidden)
				return
			}

			hasPermission := false
			switch permission {
			case "CanCreateListings":
				hasPermission = bc.Permissions.CanCreateListings
			case "CanEditListings":
				hasPermission = bc.Permissions.CanEditListings
			case "CanDeleteListings":
				hasPermission = bc.Permissions.CanDeleteListings
			case "CanPublishListings":
				hasPermission = bc.Permissions.CanPublishListings
			case "CanManageMedia":
				hasPermission = bc.Permissions.CanManageMedia
			case "CanViewAnalytics":
				hasPermission = bc.Permissions.CanViewAnalytics
			case "CanManageMembers":
				hasPermission = bc.Permissions.CanManageMembers
			case "CanEditBusiness":
				hasPermission = bc.Permissions.CanEditBusiness
			case "CanViewFinancials":
				hasPermission = bc.Permissions.CanViewFinancials
			default:
				m.log.Logf("WARN Unknown permission: %s", permission)
				http.Error(w, "Forbidden: Invalid permission", http.StatusForbidden)
				return
			}

			if !hasPermission {
				m.log.Logf("WARN User lacks permission %s in business %s", permission, bc.BusinessID)
				http.Error(w, fmt.Sprintf("Forbidden: %s permission required", permission), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		}))
	}
}

// WithTenantSlug resolves X-Tenant-Slug to a business, enforces membership, and injects business context.
// If the header is absent, it simply forwards the request.
func (m *BusinessAuthMiddleware) WithTenantSlug(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		slug := r.Header.Get("X-Tenant-Slug")
		if slug == "" {
			next.ServeHTTP(w, r)
			return
		}

		if !slugPattern.MatchString(slug) {
			m.log.Logf("WARN Invalid tenant slug format")
			http.Error(w, "Invalid tenant slug", http.StatusBadRequest)
			return
		}

		// Resolve business by slug
		business, err := m.businessService.GetBusinessBySlug(ctx, slug)
		if err != nil || business == nil {
			m.log.Logf("WARN Business not found for slug %s", slug)
			http.Error(w, "Business not found", http.StatusNotFound)
			return
		}

		// Must be authenticated to use business context
		v := viewer.FromContext(ctx)
		if v == nil || v.UserID == "" {
			m.log.Logf("WARN Unauthenticated request with tenant slug %s", slug)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userID, err := uuid.Parse(v.UserID)
		if err != nil {
			m.log.Logf("ERROR Invalid user ID: %v", err)
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		// Validate membership
		membership, err := m.businessService.GetMember(ctx, business.ID, userID)
		if err != nil {
			if err == domain.ErrMemberNotFound {
				m.log.Logf("WARN User %s is not a member of business %s (slug %s)", userID, business.ID, slug)
				http.Error(w, "Forbidden: Not a business member", http.StatusForbidden)
				return
			}
			m.log.Logf("ERROR Failed to get membership for slug %s: %v", slug, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Inject context and continue
		ctx = WithBusinessContext(ctx, &BusinessContext{
			BusinessID:  business.ID,
			Business:    business,
			Membership:  membership,
			Permissions: &membership.Permissions,
			IsOwner:     membership.IsOwner(),
			IsAdmin:     membership.IsAdmin(),
		})

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LoadBusinessContext loads business context from businessID URL parameter
// This is a lightweight version that doesn't enforce membership
func (m *BusinessAuthMiddleware) LoadBusinessContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get business ID from URL
		businessIDStr := chi.URLParam(r, "businessID")
		if businessIDStr == "" {
			// No business ID, continue without context
			next.ServeHTTP(w, r)
			return
		}

		businessID, err := uuid.Parse(businessIDStr)
		if err != nil {
			m.log.Logf("WARN Invalid business ID: %v", err)
			http.Error(w, "Invalid business ID", http.StatusBadRequest)
			return
		}

		// Get business
		business, err := m.businessService.GetBusiness(ctx, businessID)
		if err != nil {
			m.log.Logf("ERROR Failed to get business: %v", err)
			http.Error(w, "Business not found", http.StatusNotFound)
			return
		}

		// Add business to context (without membership info)
		ctx = WithBusiness(ctx, business)

		// Try to load membership if user is authenticated
		v := viewer.FromContext(ctx)
		if v != nil && v.UserID != "" {
			userID, err := uuid.Parse(v.UserID)
			if err == nil {
				membership, err := m.businessService.GetMember(ctx, businessID, userID)
				if err == nil {
					ctx = WithMembership(ctx, membership)
					ctx = WithPermissions(ctx, &membership.Permissions)
				}
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
