# Business Module Cleanup Report

**Date**: 2025-12-15
**Status**: Analysis Complete

## Executive Summary

The Business module has grown organically and contains several redundant methods across the repository, service, and resolver layers. This report identifies specific redundancies and provides actionable recommendations for cleanup.

---

## 🔴 Critical Redundancies (High Priority)

### 1. **Duplicate Membership Checking**

**Issue**: Multiple methods perform the same membership check with different names.

**Affected Methods**:
- `BusinessRepository.IsMember()` - Repository layer
- `BusinessService.CanUserAccessBusiness()` - Service layer (calls `IsMember`)
- Used by: PropertyBusinessAdapter

**Analysis**:
```go
// Service layer (permissions.go:11-21)
func (s *BusinessServiceImpl) CanUserAccessBusiness(ctx context.Context, userID, businessID uuid.UUID) (bool, error) {
    isMember, err := s.repo.IsMember(ctx, businessID, userID)
    // ... essentially just wraps IsMember
}
```

**Recommendation**:
- ✅ **KEEP**: `BusinessRepository.IsMember()` - Core repository operation
- ❌ **REMOVE**: `BusinessService.CanUserAccessBusiness()` - Redundant wrapper
- 🔧 **ACTION**: Update PropertyBusinessAdapter to use `GetMember()` instead (already implemented correctly in property_hooks.go:28-33)

**Impact**: Low risk - Already not used by property module

---

### 2. **Redundant GraphQL Query Resolvers**

**Issue**: Overlapping data retrieval for user businesses.

**Affected Resolvers**:
- `MyBusinesses(ctx)` → Returns `[]domain.Business` - Gets businesses through memberships
- `MyMemberships(ctx)` → Returns `[]domain.BusinessMember` - Contains business data in each member

**Analysis**:
```go
// resolvers.go:60-80
func (r *Resolver) MyBusinesses(ctx context.Context) ([]domain.Business, error) {
    businesses, err := r.businessService.ListUserBusinesses(ctx, userID)
    // Returns business list
}

// resolvers.go:138-158
func (r *Resolver) MyMemberships(ctx context.Context) ([]domain.BusinessMember, error) {
    memberships, err := r.businessService.GetUserMemberships(ctx, userID)
    // Returns memberships with business data + role + permissions
}
```

**Frontend Impact**:
- `MyBusinesses` - Used for simple business lists (e.g., dropdown, switcher)
- `MyMemberships` - Provides business + role + permissions (more complete data)

**Recommendation**:
- ❌ **REMOVE**: `MyBusinesses` query resolver
- ✅ **KEEP**: `MyMemberships` - Provides richer data (includes role, permissions, joined_at)
- 🔧 **ACTION**: Update frontend to use `myMemberships` query and extract business from membership
- ⚠️ **NOTE**: This is a **BREAKING CHANGE** for frontend - requires coordination

**Frontend Migration**:
```graphql
# Before
query {
  myBusinesses {
    id
    name
    slug
  }
}

# After
query {
  myMemberships {
    business {
      id
      name
      slug
    }
    role
    permissions { ... }
  }
}
```

---

### 3. **Permission Check Inefficiency**

**Issue**: Multiple permission methods fetch the same member record separately.

**Affected Methods**:
```go
// All fetch member separately
HasPermission(ctx, userID, businessID, permission) // permissions.go:37-49
GetUserPermissions(ctx, userID, businessID)        // permissions.go:23-35
IsOwner(ctx, userID, businessID)                   // permissions.go:51-63
IsAdmin(ctx, userID, businessID)                   // permissions.go:65-77
```

**Analysis**: Each method independently calls `s.repo.GetMember()` and then checks different aspects of the same domain object.

**Recommendation**:
- ✅ **KEEP ALL**: These are different semantic operations used by different consumers
- 🔧 **OPTIMIZE**: Add caching layer or batch these calls when used together
- 💡 **SUGGESTION**: Add a helper method `getMemberWithCache()` for internal use

**Possible Optimization** (Optional):
```go
// Add internal helper
func (s *BusinessServiceImpl) getMemberWithPermissions(ctx context.Context, userID, businessID uuid.UUID) (*domain.BusinessMember, error) {
    // Could add context-based caching here
    member, err := s.repo.GetMember(ctx, businessID, userID)
    if err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, domain.ErrMemberNotFound
        }
        return nil, err
    }
    return domain.MapBusinessMemberFromSchema(member), nil
}
```

---

## 🟡 Medium Priority Redundancies

### 4. **Unused Repository Methods**

**Issue**: Repository methods that are never called by service layer.

**Affected Methods**:

| Method | Defined In | Used By | Status |
|--------|------------|---------|--------|
| `PatchBusiness()` | business_repo.go:40-43 | ❌ None | Unused |
| `PatchMember()` | membership_repo.go:41-47 | ❌ None | Unused |
| `PatchInvitation()` | invitation_repo.go | ❌ None | Unused |
| `ListBusinessesByCreator()` | business_repo.go:80-88 | ❌ None | Unused |
| `BusinessExists()` | business_repo.go:55-60 | ❌ None | Unused |
| `GetActiveMemberCount()` | membership_repo.go:77-85 | ❌ None | Unused |
| `ListPendingInvitations()` | invitation_repo.go | ❌ None | Unused |

**Recommendation**:
- ❌ **REMOVE ALL** - These methods were added "just in case" but are never used
- 🔧 **ACTION**: Run a full text search to confirm no usage, then delete
- 📝 **NOTE**: If needed in future, can be easily re-added

**Verification Command**:
```bash
# Check if any method is used
grep -r "PatchBusiness" --include="*.go" internal/modules/business/service/
grep -r "ListBusinessesByCreator" --include="*.go" internal/modules/business/
```

---

### 5. **Update Method Duplication**

**Issue**: Service layer has both full update and field-specific methods.

**Affected Methods**:
- `UpdateBusiness()` - Full update with selective field updates (business_crud.go:156-223)
- Repository has both `UpdateBusiness()` and `PatchBusiness()` but service only uses `UpdateBusiness()`

**Analysis**: The service layer's `UpdateBusiness` already implements selective updates (checks for nil), making `PatchBusiness` unnecessary.

**Recommendation**:
- ✅ **KEEP**: Service `UpdateBusiness()` - Handles selective updates
- ✅ **KEEP**: Repository `UpdateBusiness()` - Used by service
- ❌ **REMOVE**: Repository `PatchBusiness()` - Never used

---

## 🟢 Low Priority (Code Smell, Not Urgent)

### 6. **GraphQL Resolver Duplication**

**Issue**: Some resolvers duplicate authentication/authorization logic.

**Pattern Found**:
```go
// Repeated in multiple resolvers
v := viewer.FromContext(ctx)
if v == nil || v.UserID == "" {
    r.log.Logf("WARN Unauthenticated attempt...")
    return nil, fmt.Errorf("unauthenticated")
}

userID, err := uuid.Parse(v.UserID)
if err != nil {
    return nil, fmt.Errorf("invalid user ID")
}
```

**Files**: All mutation/query resolvers repeat this pattern.

**Recommendation**:
- 💡 **SUGGEST**: Extract to helper method `requireAuthenticatedUser(ctx) (uuid.UUID, error)`
- ⏸️ **NOT URGENT**: This is a code smell, not a functional redundancy

**Suggested Helper**:
```go
// Add to resolvers.go
func (r *Resolver) requireAuthenticatedUser(ctx context.Context) (uuid.UUID, error) {
    v := viewer.FromContext(ctx)
    if v == nil || v.UserID == "" {
        return uuid.Nil, fmt.Errorf("unauthenticated")
    }

    userID, err := uuid.Parse(v.UserID)
    if err != nil {
        return uuid.Nil, fmt.Errorf("invalid user ID")
    }

    return userID, nil
}
```

---

### 7. **Repository Hard Delete Method**

**Issue**: `HardDeleteBusiness()` exists but is never used (soft delete is always used).

**Method**: `business_repo.go:50-53`

**Recommendation**:
- ⏸️ **KEEP FOR NOW**: Might be needed for admin operations or GDPR compliance
- 📝 **DOCUMENT**: Add comment explaining when/why hard delete would be used
- 🔐 **SECURE**: Ensure it's never exposed through GraphQL API

---

## 📊 Quantitative Summary

### Methods to Remove (Immediate)
| Layer | Method Count | Status |
|-------|--------------|--------|
| Service | 1 | `CanUserAccessBusiness()` |
| Repository | 7 | All `Patch*`, unused list methods |
| Resolver | 1 | `MyBusinesses()` ⚠️ Breaking change |
| **TOTAL** | **9 methods** | **Can be removed** |

### Lines of Code Impact
- **Estimated removal**: ~150-200 lines
- **Test cleanup needed**: ~50-100 lines
- **Documentation updates**: ~20 lines

---

## 🎯 Recommended Action Plan

### Phase 1: Safe Removals (No Breaking Changes)
**Priority**: High
**Risk**: Low
**Effort**: 1-2 hours

1. ✅ Remove unused repository methods:
   - `PatchBusiness()`
   - `PatchMember()`
   - `PatchInvitation()`
   - `ListBusinessesByCreator()`
   - `BusinessExists()`
   - `GetActiveMemberCount()`
   - `ListPendingInvitations()`

2. ✅ Remove service redundancy:
   - `CanUserAccessBusiness()` (already not used by property module)

3. ✅ Update interface files to reflect removals

4. ✅ Run tests to ensure nothing breaks

**Files to Modify**:
- `internal/modules/business/repository/interface.go`
- `internal/modules/business/repository/business_repo.go`
- `internal/modules/business/repository/membership_repo.go`
- `internal/modules/business/repository/invitation_repo.go`
- `internal/modules/business/service/interface.go`
- `internal/modules/business/service/permissions.go`

---

### Phase 2: Breaking Changes (Requires Frontend Coordination)
**Priority**: Medium
**Risk**: High
**Effort**: 2-4 hours (including frontend updates)

1. ⚠️ Remove `MyBusinesses` GraphQL resolver
2. ⚠️ Update GraphQL schema
3. ⚠️ Update frontend to use `myMemberships` instead
4. ⚠️ Run integration tests

**Files to Modify**:
- `internal/modules/business/port/graphql/schema.graphqls`
- `internal/modules/business/port/graphql/resolvers.go`
- Frontend GraphQL queries

---

### Phase 3: Code Quality Improvements (Optional)
**Priority**: Low
**Risk**: Low
**Effort**: 2-3 hours

1. 💡 Extract `requireAuthenticatedUser()` helper
2. 💡 Add permission check caching/batching
3. 💡 Document `HardDeleteBusiness()` usage

---

## 🧪 Testing Strategy

### Before Removal
```bash
# Run all business module tests
go test ./internal/modules/business/... -v

# Check for usages
grep -r "CanUserAccessBusiness" --include="*.go" .
grep -r "MyBusinesses" --include="*.go" .
```

### After Removal
```bash
# Ensure compilation
go build ./...

# Run tests
go test ./internal/modules/business/... -v

# Run integration tests
go test ./... -tags=integration
```

---

## 💰 Expected Benefits

### Maintainability
- ✅ Fewer methods to maintain
- ✅ Clearer separation of concerns
- ✅ Less cognitive load for new developers

### Performance
- ⚡ Slightly less dead code in binary
- ⚡ Reduced attack surface (fewer unused endpoints)

### Code Quality
- 📈 Improved test coverage focus
- 📈 Clearer API boundaries
- 📈 Better documentation signal-to-noise ratio

---

## ⚠️ Risks and Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Frontend breaks after `MyBusinesses` removal | High | High | Coordinate with frontend team, staged rollout |
| Hidden usage of "unused" methods | Low | Medium | Thorough grep search, staged removal |
| Tests break unexpectedly | Low | Low | Run full test suite before/after |
| Integration breaks | Very Low | High | Run integration tests, staging environment test |

---

## 📝 Notes

1. **No Performance Impact**: These redundancies don't cause runtime performance issues, only maintenance burden.

2. **Breaking Changes**: Only the `MyBusinesses` removal is a breaking change. Everything else is internal.

3. **Future-Proofing**: Some methods (like `HardDeleteBusiness`) might seem unused but could be needed for compliance/admin tools.

4. **Property Module Integration**: Already using the correct methods (`GetMember` not `CanUserAccessBusiness`), so no impact there.

---

## 🤝 Next Steps

1. **Review**: Team reviews this report
2. **Prioritize**: Decide on phase execution order
3. **Coordinate**: Sync with frontend team on breaking changes
4. **Execute**: Implement Phase 1 (safe removals) first
5. **Test**: Comprehensive testing after each phase
6. **Document**: Update API documentation

---

**Report Generated By**: Claude Code (Phase 8 Analysis)
**Reviewed By**: [Pending]
**Approved By**: [Pending]
**Implementation Date**: [TBD]