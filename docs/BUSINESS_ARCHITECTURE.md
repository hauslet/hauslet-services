# Business Module Architecture Guide

## Overview

The business module enables users to create and manage business profiles similar to GitHub Organizations. A single user can operate in two contexts:
1. **Individual User** - Personal account with individual listings
2. **Business Member** - Member of one or more businesses, with role-based permissions

## Table of Contents
- [Core Concepts](#core-concepts)
- [User-Business Relationship](#user-business-relationship)
- [Ownership Models](#ownership-models)
- [Dashboard Separation](#dashboard-separation)
- [Listings & Properties](#listings--properties)
- [Permission System](#permission-system)
- [GraphQL API Structure](#graphql-api-structure)
- [Frontend Integration Examples](#frontend-integration-examples)

---

## Core Concepts

### 1. User Account (Individual)
Every user has a personal account that exists independently of any business.

```
┌─────────────────────────────────────┐
│         User (Individual)           │
├─────────────────────────────────────┤
│ • Personal Profile                  │
│ • Individual Listings               │
│ • Personal Dashboard                │
│ • Can create businesses             │
│ • Can join multiple businesses      │
└─────────────────────────────────────┘
```

### 2. Business Profile
A business is a separate entity that can have multiple members.

```
┌─────────────────────────────────────┐
│            Business                 │
├─────────────────────────────────────┤
│ • Business Profile                  │
│ • Business Listings                 │
│ • Business Dashboard                │
│ • Team Members (with roles)         │
│ • Shared Resources                  │
└─────────────────────────────────────┘
```

---

## User-Business Relationship

### Relationship Diagram

```
┌──────────────┐
│   User A     │───────────┐
│ (Individual) │           │
└──────────────┘           │
       │                   ├──── Owner ────────┐
       │                   │                   ▼
       │                   │            ┌──────────────┐
       │                   │            │  Business X  │
       │                   │            └──────────────┘
       │                   │                   ▲
       │                   └──── Admin ────────┤
       │                                       │
┌──────────────┐                               │
│   User B     │───── Member ───────────────────┘
│ (Individual) │
└──────────────┘
       │
       │
       └──── Owner ────────┐
                           ▼
                    ┌──────────────┐
                    │  Business Y  │
                    └──────────────┘
```

### Multi-Context Example

**User A** can operate in 3 contexts:
1. **Individual Context**: Create personal listings, manage own profile
2. **Business X Context (Owner)**: Full control, can invite members, manage all listings
3. **Business Y Context (Member)**: Limited access based on assigned permissions

---

## Ownership Models

### Property Ownership Types

Properties and listings can be owned by either individuals OR businesses:

```
Property/Listing Ownership
├── Individual Owned
│   ├── Owner: User ID
│   ├── Owner Type: "individual"
│   └── Access: Only the user
│
└── Business Owned
    ├── Owner: Business ID
    ├── Owner Type: "business"
    └── Access: Business members (based on permissions)
```

### Database Schema

```go
// Existing in Property module
type Listing struct {
    ID          uuid.UUID
    OwnerID     uuid.UUID  // Can be UserID or BusinessID
    OwnerType   string     // "individual" or "business"
    CreatedBy   uuid.UUID  // The actual user who created it
    // ... other fields
}
```

### Ownership Flow Diagram

```
User creates a listing
        │
        ├───► "Create as Individual?"
        │            │
        │            ├─ Yes ──► OwnerType: "individual"
        │            │          OwnerID: {user_id}
        │            │
        │            └─ No ───► Select Business
        │                              │
        │                              ├─ Check: User has "CanCreateListings" permission?
        │                              │         │
        │                              │         ├─ Yes ──► OwnerType: "business"
        │                              │         │          OwnerID: {business_id}
        │                              │         │          CreatedBy: {user_id}
        │                              │         │
        │                              │         └─ No ───► Error: Insufficient permissions
        │                              │
        └──────────────────────────────┘
```

---

## Dashboard Separation

### Dashboard Access Pattern

```
┌─────────────────────────────────────────────────────────────┐
│                      User Dashboard                         │
│                   (Personal Context)                        │
├─────────────────────────────────────────────────────────────┤
│  Route: /dashboard                                          │
│                                                             │
│  • My Profile                                               │
│  • My Individual Listings (OwnerType = "individual")        │
│  • My Messages (individual conversations)                   │
│  • My Businesses (list of businesses I'm a member of)       │
│  • Create New Business                                      │
└─────────────────────────────────────────────────────────────┘

                        ▼ User selects a business

┌─────────────────────────────────────────────────────────────┐
│                   Business Dashboard                        │
│                  (Business Context)                         │
├─────────────────────────────────────────────────────────────┤
│  Route: /businesses/{businessId}                            │
│                                                             │
│  • Business Profile (if has "CanEditBusiness")              │
│  • Business Listings (OwnerType = "business")               │
│  • Team Members (if owner/admin)                            │
│  • Invitations (if has "CanManageMembers")                  │
│  • Analytics (if has "CanViewAnalytics")                    │
│  • Financials (if has "CanViewFinancials")                  │
│  • Business Messages (business conversations)               │
│                                                             │
│  [Switch Back to Personal Dashboard] button                │
└─────────────────────────────────────────────────────────────┘
```

### Navigation Example

```jsx
// User Navigation Component
<nav>
  {/* Always visible */}
  <NavItem to="/dashboard">My Dashboard</NavItem>

  {/* Show user's businesses */}
  <Dropdown label="My Businesses">
    {userBusinesses.map(business => (
      <DropdownItem
        key={business.id}
        to={`/businesses/${business.id}`}
      >
        {business.name} ({business.role})
      </DropdownItem>
    ))}
  </Dropdown>

  <NavItem to="/businesses/new">Create Business</NavItem>
</nav>
```

---

## Listings & Properties

### Filtering Listings by Context

#### Individual Dashboard Query
```graphql
# Get only MY individual listings
query MyIndividualListings {
  myListings(ownerType: INDIVIDUAL) {
    id
    title
    ownerType
    ownerID
    createdBy
  }
}
```

#### Business Dashboard Query
```graphql
# Get listings for a specific business
query BusinessListings($businessId: UUID!) {
  businessListings(businessId: $businessId) {
    id
    title
    ownerType
    ownerID
    createdBy  # Shows which team member created it
    createdByUser {
      name
      email
    }
  }
}
```

### Creating Listings - Context Selector

```
┌─────────────────────────────────────────────┐
│        Create New Listing Form              │
├─────────────────────────────────────────────┤
│                                             │
│  Create as:                                 │
│  ○ Personal Listing                         │
│  ○ Business Listing                         │
│                                             │
│  [If Business selected]                     │
│  ┌─────────────────────────────────┐       │
│  │ Select Business:                │       │
│  │ ▼ Acme Property Management      │       │
│  │   ├─ You are Owner              │       │
│  │   └─ ✓ Can create listings      │       │
│  │                                 │       │
│  │ ▼ Downtown Realty               │       │
│  │   ├─ You are Member             │       │
│  │   └─ ✗ Cannot create listings   │       │
│  └─────────────────────────────────┘       │
│                                             │
│  [Continue to listing details...]          │
└─────────────────────────────────────────────┘
```

---

## Permission System

### Roles and Default Permissions

```
┌──────────────────────────────────────────────────────────────┐
│                         OWNER                                │
├──────────────────────────────────────────────────────────────┤
│ • All permissions (cannot be restricted)                     │
│ • Can delete the business                                    │
│ • Can transfer ownership                                     │
│ • Can invite/remove any member including admins              │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                         ADMIN                                │
├──────────────────────────────────────────────────────────────┤
│ ✓ CanCreateListings                                          │
│ ✓ CanEditListings                                            │
│ ✓ CanDeleteListings                                          │
│ ✓ CanPublishListings                                         │
│ ✓ CanManageMedia                                             │
│ ✓ CanViewAnalytics                                           │
│ ✓ CanManageMembers                                           │
│ ✓ CanEditBusiness                                            │
│ ✗ CanViewFinancials (optional)                               │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                        MEMBER                                │
├──────────────────────────────────────────────────────────────┤
│ ✓ CanCreateListings                                          │
│ ✓ CanEditListings                                            │
│ ✗ CanDeleteListings                                          │
│ ✗ CanPublishListings                                         │
│ ✓ CanManageMedia                                             │
│ ✗ CanViewAnalytics                                           │
│ ✗ CanManageMembers                                           │
│ ✗ CanEditBusiness                                            │
│ ✗ CanViewFinancials                                          │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                        VIEWER                                │
├──────────────────────────────────────────────────────────────┤
│ ✗ CanCreateListings                                          │
│ ✗ CanEditListings                                            │
│ ✗ CanDeleteListings                                          │
│ ✗ CanPublishListings                                         │
│ ✗ CanManageMedia                                             │
│ ✓ CanViewAnalytics                                           │
│ ✗ CanManageMembers                                           │
│ ✗ CanEditBusiness                                            │
│ ✗ CanViewFinancials                                          │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                      ACCOUNTANT                              │
├──────────────────────────────────────────────────────────────┤
│ ✗ CanCreateListings                                          │
│ ✗ CanEditListings                                            │
│ ✗ CanDeleteListings                                          │
│ ✗ CanPublishListings                                         │
│ ✗ CanManageMedia                                             │
│ ✓ CanViewAnalytics                                           │
│ ✗ CanManageMembers                                           │
│ ✗ CanEditBusiness                                            │
│ ✓ CanViewFinancials                                          │
└──────────────────────────────────────────────────────────────┘
```

### Custom Permissions

Owners and admins can customize permissions for individual members:

```graphql
mutation UpdateMemberPermissions($businessId: UUID!, $memberId: UUID!, $permissions: MemberPermissionsInput!) {
  updateMemberPermissions(
    businessId: $businessId
    memberId: $memberId
    permissions: {
      canCreateListings: true
      canEditListings: true
      canDeleteListings: false
      canPublishListings: true
      canManageMedia: true
      canViewAnalytics: false
      canManageMembers: false
      canEditBusiness: false
      canViewFinancials: false
    }
  ) {
    id
    role
    permissions {
      canCreateListings
      canEditListings
      # ...
    }
  }
}
```

---

## GraphQL API Structure

### Queries

#### User Context Queries
```graphql
# Get my individual profile
query {
  me {
    id
    name
    email
  }
}

# Get businesses I'm a member of
query {
  myBusinesses {
    id
    name
    slug
    role          # My role in this business
    permissions   # My permissions in this business
  }
}

# Get my individual listings
query {
  myListings(ownerType: INDIVIDUAL) {
    id
    title
    ownerType
  }
}
```

#### Business Context Queries
```graphql
# Get specific business details
query GetBusiness($id: UUID!) {
  business(id: $id) {
    id
    name
    description
    businessType
    memberCount
    propertyCount
    listingCount
  }
}

# Get business listings
query GetBusinessListings($businessId: UUID!) {
  listings(
    filter: {
      ownerType: BUSINESS
      ownerID: $businessId
    }
  ) {
    id
    title
    createdBy
    createdByUser {
      name
      email
    }
  }
}

# Get business members (if authorized)
query GetBusinessMembers($businessId: UUID!) {
  businessMembers(businessId: $businessId) {
    id
    user {
      id
      name
      email
    }
    role
    permissions {
      canCreateListings
      canEditListings
      # ...
    }
    joinedAt
  }
}

# Check my permissions in a business
query MyBusinessPermissions($businessId: UUID!) {
  myBusinessPermissions(businessId: $businessId) {
    canCreateListings
    canEditListings
    canDeleteListings
    canPublishListings
    canManageMedia
    canViewAnalytics
    canManageMembers
    canEditBusiness
    canViewFinancials
  }
}
```

### Mutations

#### Business Management
```graphql
# Create a business
mutation CreateBusiness($input: CreateBusinessInput!) {
  createBusiness(input: {
    name: "Acme Property Management"
    displayName: "Acme PM"
    businessType: PROPERTY_MANAGEMENT
    email: "contact@acme.com"
    phone: "+234..."
    address: {
      street: "..."
      city: "Lagos"
      # ...
    }
  }) {
    id
    slug
    name
  }
}

# Update business profile (requires CanEditBusiness permission)
mutation UpdateBusiness($id: UUID!, $input: UpdateBusinessInput!) {
  updateBusiness(id: $id, input: {
    displayName: "New Display Name"
    description: "Updated description"
  }) {
    id
    displayName
    description
  }
}
```

#### Member Management
```graphql
# Invite a member (requires CanManageMembers permission)
mutation InviteMember($businessId: UUID!, $input: InviteMemberInput!) {
  inviteMember(
    businessId: $businessId
    input: {
      email: "newmember@example.com"
      role: MEMBER
      customPermissions: {
        canCreateListings: true
        canEditListings: true
        # ...
      }
    }
  ) {
    id
    email
    role
    token
    expiresAt
  }
}

# Accept invitation
mutation AcceptInvitation($token: String!) {
  acceptInvitation(token: $token) {
    id
    businessID
    role
    permissions {
      # ...
    }
  }
}

# Update member role (requires owner or admin)
mutation UpdateMemberRole($businessId: UUID!, $memberId: UUID!, $role: MemberRole!) {
  updateMemberRole(
    businessId: $businessId
    memberId: $memberId
    role: ADMIN
  ) {
    id
    role
    permissions {
      # ...
    }
  }
}
```

#### Listing Creation with Context
```graphql
# Create personal listing
mutation CreatePersonalListing($input: CreateListingInput!) {
  createListing(input: {
    ownerType: INDIVIDUAL
    # ... listing details
  }) {
    id
    ownerType
    ownerID
  }
}

# Create business listing (requires CanCreateListings permission)
mutation CreateBusinessListing($input: CreateListingInput!) {
  createListing(input: {
    ownerType: BUSINESS
    businessID: "uuid-of-business"
    # ... listing details
  }) {
    id
    ownerType
    ownerID
    createdBy  # Your user ID
  }
}
```

---

## Frontend Integration Examples

### 1. Context Provider Pattern

```tsx
// contexts/BusinessContext.tsx
import { createContext, useContext, useState } from 'react';

interface BusinessContextType {
  currentContext: 'individual' | 'business';
  currentBusinessId?: string;
  switchToIndividual: () => void;
  switchToBusiness: (businessId: string) => void;
}

const BusinessContext = createContext<BusinessContextType | null>(null);

export function BusinessProvider({ children }) {
  const [currentContext, setCurrentContext] = useState<'individual' | 'business'>('individual');
  const [currentBusinessId, setCurrentBusinessId] = useState<string>();

  const switchToIndividual = () => {
    setCurrentContext('individual');
    setCurrentBusinessId(undefined);
  };

  const switchToBusiness = (businessId: string) => {
    setCurrentContext('business');
    setCurrentBusinessId(businessId);
  };

  return (
    <BusinessContext.Provider value={{
      currentContext,
      currentBusinessId,
      switchToIndividual,
      switchToBusiness
    }}>
      {children}
    </BusinessContext.Provider>
  );
}

export const useBusinessContext = () => {
  const context = useContext(BusinessContext);
  if (!context) throw new Error('useBusinessContext must be used within BusinessProvider');
  return context;
};
```

### 2. Dashboard Component

```tsx
// pages/Dashboard.tsx
import { useBusinessContext } from '../contexts/BusinessContext';
import IndividualDashboard from '../components/IndividualDashboard';
import BusinessDashboard from '../components/BusinessDashboard';

export default function Dashboard() {
  const { currentContext, currentBusinessId } = useBusinessContext();

  if (currentContext === 'individual') {
    return <IndividualDashboard />;
  }

  return <BusinessDashboard businessId={currentBusinessId!} />;
}
```

### 3. Listing Creation Form

```tsx
// components/CreateListingForm.tsx
import { useState } from 'react';
import { useQuery } from '@apollo/client';
import { GET_MY_BUSINESSES } from '../graphql/queries';

export default function CreateListingForm() {
  const [ownerType, setOwnerType] = useState<'individual' | 'business'>('individual');
  const [selectedBusinessId, setSelectedBusinessId] = useState<string>();

  const { data: businessesData } = useQuery(GET_MY_BUSINESSES);

  return (
    <form>
      <h2>Create New Listing</h2>

      {/* Owner Type Selector */}
      <div>
        <label>Create as:</label>
        <select value={ownerType} onChange={(e) => setOwnerType(e.target.value)}>
          <option value="individual">Personal Listing</option>
          <option value="business">Business Listing</option>
        </select>
      </div>

      {/* Business Selector */}
      {ownerType === 'business' && (
        <div>
          <label>Select Business:</label>
          <select
            value={selectedBusinessId}
            onChange={(e) => setSelectedBusinessId(e.target.value)}
          >
            <option value="">-- Select a business --</option>
            {businessesData?.myBusinesses
              ?.filter(b => b.permissions.canCreateListings)
              .map(business => (
                <option key={business.id} value={business.id}>
                  {business.name} ({business.role})
                </option>
              ))
            }
          </select>

          {selectedBusinessId && (
            <p className="text-sm text-gray-500">
              You have permission to create listings for this business
            </p>
          )}
        </div>
      )}

      {/* Rest of the listing form */}
      <div>
        {/* Title, description, etc. */}
      </div>

      <button type="submit">Create Listing</button>
    </form>
  );
}
```

### 4. Permission-Based UI

```tsx
// components/BusinessDashboard.tsx
import { useQuery } from '@apollo/client';
import { GET_MY_BUSINESS_PERMISSIONS } from '../graphql/queries';

export default function BusinessDashboard({ businessId }: { businessId: string }) {
  const { data, loading } = useQuery(GET_MY_BUSINESS_PERMISSIONS, {
    variables: { businessId }
  });

  if (loading) return <div>Loading...</div>;

  const permissions = data?.myBusinessPermissions;

  return (
    <div>
      <h1>Business Dashboard</h1>

      {/* Conditionally render based on permissions */}
      {permissions?.canCreateListings && (
        <button>Create New Listing</button>
      )}

      {permissions?.canManageMembers && (
        <section>
          <h2>Team Management</h2>
          <button>Invite Member</button>
          {/* Team member list */}
        </section>
      )}

      {permissions?.canEditBusiness && (
        <section>
          <h2>Business Settings</h2>
          {/* Business settings form */}
        </section>
      )}

      {permissions?.canViewAnalytics && (
        <section>
          <h2>Analytics</h2>
          {/* Analytics dashboard */}
        </section>
      )}

      {permissions?.canViewFinancials && (
        <section>
          <h2>Financials</h2>
          {/* Financial reports */}
        </section>
      )}
    </div>
  );
}
```

### 5. Listing List with Context Filter

```tsx
// components/ListingsList.tsx
import { useQuery } from '@apollo/client';
import { useBusinessContext } from '../contexts/BusinessContext';
import { GET_MY_LISTINGS, GET_BUSINESS_LISTINGS } from '../graphql/queries';

export default function ListingsList() {
  const { currentContext, currentBusinessId } = useBusinessContext();

  // Query based on context
  const { data, loading } = useQuery(
    currentContext === 'individual' ? GET_MY_LISTINGS : GET_BUSINESS_LISTINGS,
    {
      variables: currentContext === 'business'
        ? { businessId: currentBusinessId }
        : { ownerType: 'INDIVIDUAL' }
    }
  );

  if (loading) return <div>Loading listings...</div>;

  const listings = currentContext === 'individual'
    ? data?.myListings
    : data?.businessListings;

  return (
    <div>
      <h2>
        {currentContext === 'individual' ? 'My Listings' : 'Business Listings'}
      </h2>

      <ul>
        {listings?.map(listing => (
          <li key={listing.id}>
            <h3>{listing.title}</h3>
            <p>Type: {listing.ownerType}</p>
            {listing.createdBy && currentContext === 'business' && (
              <p className="text-sm">
                Created by: {listing.createdByUser?.name}
              </p>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
}
```

---

## Future: Messaging Integration

### Message Context Separation

```
Messages will be separated by context:

┌──────────────────────────────────────────────────────────────┐
│                  Individual Messages                         │
├──────────────────────────────────────────────────────────────┤
│ Conversations tied to:                                       │
│ • Your personal listings                                     │
│ • Direct messages to you as an individual                    │
│                                                              │
│ Route: /messages                                             │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                   Business Messages                          │
├──────────────────────────────────────────────────────────────┤
│ Conversations tied to:                                       │
│ • Business listings                                          │
│ • Messages to the business                                   │
│ • Shared among authorized team members                       │
│                                                              │
│ Route: /businesses/{businessId}/messages                     │
│                                                              │
│ Access: Based on permissions (e.g., CanManageListings)       │
└──────────────────────────────────────────────────────────────┘
```

### Message Schema (Future)

```go
type Message struct {
    ID          uuid.UUID
    ConversationID uuid.UUID

    // Context
    OwnerType   string     // "individual" or "business"
    OwnerID     uuid.UUID  // UserID or BusinessID

    // If business context
    BusinessID  *uuid.UUID

    // Message details
    SenderID    uuid.UUID
    Content     string
    CreatedAt   time.Time
}
```

---

## Summary for Frontend Developer

### Key Points to Remember:

1. **Dual Context**: Users always operate in one of two contexts - individual or business
2. **Separate Dashboards**: Individual dashboard (`/dashboard`) and business dashboard (`/businesses/{id}`)
3. **Permission-Based UI**: Show/hide features based on user's permissions in the current business
4. **Context-Aware Queries**: Always filter data by context (individual vs business)
5. **Ownership Model**: Every resource (listing, message, etc.) has an `ownerType` and `ownerID`
6. **Role-Based Access**: Use GraphQL queries to check permissions before rendering UI elements

### Implementation Checklist:

- [ ] Create BusinessContext provider
- [ ] Implement context switcher in navigation
- [ ] Build separate dashboard components
- [ ] Add permission checks to all business-related UI
- [ ] Filter listings by context (individual vs business)
- [ ] Show business selector in creation forms
- [ ] Display team member info on business listings
- [ ] Implement invitation flow UI
- [ ] Add role/permission management UI (for owners/admins)
- [ ] Prepare for future messaging context separation

### Questions to Ask Backend:

1. Should there be a default business context when user has only one business?
2. How should we handle notifications between contexts?
3. What happens to business listings when a member is removed?
4. Can a user transfer individual listings to a business?
5. What's the search/filter behavior across contexts?

---

## Additional Resources

- GraphQL Schema: `/internal/modules/business/port/graphql/schema.graphqls`
- Domain Models: `/internal/modules/business/domain/`
- Permission System: `/internal/modules/business/middleware/`
- Email Notifications: `/internal/modules/business/notification/`

For questions or clarifications, reach out to the backend team!