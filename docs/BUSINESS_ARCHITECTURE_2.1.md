Here is the comprehensive **Business Module Architecture Guide (v2.1)**.

I have restored the original **Ownership Flow Diagram**, **Relationship Diagrams**, and **Role Tables**, but adapted them to accurately reflect the new **Subdomain + Global Store** architecture.

-----

# Business Module Architecture Guide

## Overview

The business module enables users to create and manage business profiles similar to GitHub Organizations. The architecture uses a **Multi-Zone approach** to strictly separate personal and business contexts via subdomains.

A single user can operate in two contexts:

1.  **Individual Context** (`app.domain.com`) - Personal account with individual listings.
2.  **Business Context** (`business.domain.com/:slug`) - Member of a specific business workspace.

## Table of Contents

  - [Core Concepts](https://www.google.com/search?q=%23core-concepts)
  - [User-Business Relationship](https://www.google.com/search?q=%23user-business-relationship)
  - [Ownership Models & Flow](https://www.google.com/search?q=%23ownership-models--flow)
  - [Dashboard & Domain Separation](https://www.google.com/search?q=%23dashboard--domain-separation)
  - [Context Detection & State](https://www.google.com/search?q=%23context-detection--state)
  - [Listings & Properties](https://www.google.com/search?q=%23listings--properties)
  - [Permission System](https://www.google.com/search?q=%23permission-system)
  - [GraphQL API Structure](https://www.google.com/search?q=%23graphql-api-structure)
  - [Mobile Integration Strategy](https://www.google.com/search?q=%23mobile-integration-strategy)
  - [Frontend Integration Examples](https://www.google.com/search?q=%23frontend-integration-examples)

-----

## Core Concepts

### 1\. User Account (Individual)

Every user has a personal account that acts as the anchor.

```
┌─────────────────────────────────────┐
│    Context: INDIVIDUAL (Root)       │
│      URL: app.domain.com            │
├─────────────────────────────────────┤
│ • Personal Profile                  │
│ • Individual Listings               │
│ • Personal Dashboard                │
│ • Can create businesses             │
│ • Can join multiple businesses      │
└─────────────────────────────────────┘
```

### 2\. Business Profile (Workspace)

A business is a separate entity accessed via a dedicated URL structure.

```
┌─────────────────────────────────────┐
│    Context: BUSINESS (Workspace)    │
│  URL: business.domain.com/:slug     │
├─────────────────────────────────────┤
│ • Business Profile                  │
│ • Business Listings                 │
│ • Business Dashboard                │
│ • Team Members (with roles)         │
│ • Shared Resources                  │
└─────────────────────────────────────┘
```

-----

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
```

### Multi-Context Example

**User A** can operate in different contexts based on the URL:

1.  **Individual Context**: `app.domain.com/dashboard`
2.  **Business X Context**: `business.domain.com/business-x/dashboard`

-----

## Ownership Models & Flow

### Property Ownership Types

Properties and listings can be owned by either individuals OR businesses.

```
Property/Listing Ownership
├── Individual Owned
│   ├── Owner: User ID
│   ├── Owner Type: "individual"
│   └── Context: Created while on app.domain.com
│
└── Business Owned
    ├── Owner: Business ID
    ├── Owner Type: "business"
    └── Context: Created while on business.domain.com/:slug
```

### Database Schema

```go
// Existing in Property module
type Listing struct {
    ID          uuid.UUID
    OwnerID     uuid.UUID  // Can be UserID or BusinessID
    OwnerType   string     // "individual" or "business"
    CreatedBy   uuid.UUID  // The actual user who created it
}
```

### Ownership Flow Diagram

This flow determines who owns a listing based on the user's **Current Active Context** (URL or Mobile State).

```
User clicks "Create Listing"
        │
        ▼
Check Active Context (Global Store / URL)
        │
        ├───► Context: INDIVIDUAL (app.domain.com)
        │       │
        │       └─► OwnerType: "individual"
        │           OwnerID: {user_id}
        │
        └───► Context: BUSINESS (business.domain.com/slug)
                │
                ├─ Check: User has "CanCreateListings" permission?
                │         │
                │         ├─ Yes ──► OwnerType: "business"
                │         │          OwnerID: {business_id}
                │         │          CreatedBy: {user_id}
                │         │
                │         └─ No ───► Error: Insufficient permissions
```

-----

## Dashboard & Domain Separation

### Architecture Requirements

1.  **Shared Cookies:** The auth cookie domain must be set to `.domain.com` so the user stays logged in across `app` and `business` subdomains.
2.  **Cross-Domain Navigation:** Switching contexts requires a full URL navigation (using `href`), not just a React Router push.

### Dashboard Access Pattern

```
┌─────────────────────────────────────────────────────────────┐
│                      User Dashboard                         │
│                   (Individual Context)                      │
├─────────────────────────────────────────────────────────────┤
│  Host: app.domain.com                                       │
│  Route: /dashboard                                          │
│                                                             │
│  • My Profile                                               │
│  • My Individual Listings                                   │
│  • My Businesses (Links to business.domain.com/...)         │
│  • Create New Business                                      │
└─────────────────────────────────────────────────────────────┘

         ▼ User clicks "Acme Corp" (Full Navigation)

┌─────────────────────────────────────────────────────────────┐
│                   Business Dashboard                        │
│                  (Business Context)                         │
├─────────────────────────────────────────────────────────────┤
│  Host: business.domain.com                                  │
│  Route: /acme-corp/dashboard                                │
│                                                             │
│  • Context Header: X-Tenant-Slug: acme-corp                 │
│  • Business Profile                                         │
│  • Business Listings                                        │
│  • Team Members                                             │
│  • [Exit to Personal Dashboard] (Link to app.domain.com)    │
└─────────────────────────────────────────────────────────────┘
```

-----

## Context Detection & State

To unify Web and Mobile logic, we use a **Global Store** (e.g., Zustand/Redux), but the initialization logic differs per platform.

### Web Detection Logic

The frontend must parse the URL to determine the current context.

```typescript
// utils/contextDetection.ts
export const detectContext = () => {
  const hostname = window.location.hostname; // e.g., "business.domain.com"
  
  // Check if we are on the business subdomain
  if (hostname.startsWith('business.')) {
    const pathParts = window.location.pathname.split('/').filter(Boolean);
    const slug = pathParts[0]; // e.g., "acme-corp"
    
    if (slug) return { type: 'BUSINESS', slug };
  }

  return { type: 'INDIVIDUAL' };
};
```

-----

## Listings & Properties

### Filtering Listings by Context

#### Individual Dashboard Query

```graphql
# Get only MY individual listings
# Headers: None or explicit context
query MyIndividualListings {
  myListings(ownerType: INDIVIDUAL) {
    id
    title
  }
}
```

#### Business Dashboard Query

```graphql
# Get listings for the current business
# Headers: X-Tenant-Slug: acme-corp
query CurrentContextListings {
  # Backend infers Business ID from the Slug header
  listings {
    id
    title
    ownerType # Returns BUSINESS
    createdByUser { name }
  }
}
```

### Creating Listings - Context UI

Unlike the previous plan where the user selected a business from a dropdown, the **Context is now inferred**.

```
┌─────────────────────────────────────────────┐
│        Create New Listing Form              │
├─────────────────────────────────────────────┤
│                                             │
│  [BANNER] You are creating a listing for:   │
│  🏢 Acme Property Management                │
│                                             │
│  ┌───────────────────────────────────────┐  │
│  │ Title:                                │  │
│  │ _____________________________________ │  │
│  └───────────────────────────────────────┘  │
│                                             │
│  [Create Listing]                           │
│     └── Backend checks X-Tenant-Slug        │
│         and assigns Owner: Acme             │
└─────────────────────────────────────────────┘
```

-----

## Permission System

### Roles and Default Permissions

```
┌──────────────────────────────────────────────────────────────┐
│                         OWNER                                │
├──────────────────────────────────────────────────────────────┤
│ • All permissions (cannot be restricted)                     │
│ • Can delete the business                                    │
│ • Can invite/remove any member including admins              │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                         ADMIN                                │
├──────────────────────────────────────────────────────────────┤
│ ✓ CanCreateListings    ✓ CanViewAnalytics                    │
│ ✓ CanEditListings      ✓ CanManageMembers                    │
│ ✓ CanDeleteListings    ✓ CanEditBusiness                     │
│ ✓ CanPublishListings   ✗ CanViewFinancials (optional)        │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                        MEMBER                                │
├──────────────────────────────────────────────────────────────┤
│ ✓ CanCreateListings    ✗ CanViewAnalytics                    │
│ ✓ CanEditListings      ✗ CanManageMembers                    │
│ ✗ CanDeleteListings    ✗ CanEditBusiness                     │
│ ✗ CanPublishListings   ✗ CanViewFinancials                   │
└──────────────────────────────────────────────────────────────┘
```

-----

## GraphQL API Structure

### Header Middleware

We introduce a middleware layer to handle context.

  * **Request Header:** `X-Tenant-Slug: acme-corp`
  * **Middleware Logic:**
    1.  Resolve Slug to `BusinessID`.
    2.  Validate User is a member of this Business.
    3.  Inject `BusinessID` into the GraphQL Context.

### Mutations

#### Business Management

```graphql
mutation CreateBusiness($input: CreateBusinessInput!) {
  createBusiness(input: {
    name: "Acme Corp"
    slug: "acme-corp" # Explicitly defining the URL slug
  }) {
    id
    slug
  }
}
```

#### Listing Creation

```graphql
# Input does NOT need OwnerType/OwnerID if header is present
mutation CreateListing($input: CreateListingInput!) {
  createListing(input: {
    title: "New Property"
  }) {
    id
    ownerType # Backend sets this based on the Context Header
    ownerID
  }
}
```

-----

## Mobile Integration Strategy

Since Mobile Apps do not have URLs/Subdomains, they rely on **Internal State** and **Deep Linking**.

### 1\. Global Store (The "Master")

On mobile, the app stores the `activeContext` in memory/storage.

```typescript
// Mobile State
state = {
  activeContext: { type: 'BUSINESS', slug: 'acme-corp', id: '123' }
}
```

### 2\. API Injection

An API Interceptor injects the header manually based on the state.

```typescript
// Mobile API Client
client.interceptors.request.use(config => {
  if (state.activeContext.type === 'BUSINESS') {
    config.headers['X-Tenant-Slug'] = state.activeContext.slug;
  }
  return config;
});
```

### 3\. Deep Linking (Universal Links)

We map Web URLs to App State changes.

| Web URL | App Action |
| :--- | :--- |
| `app.domain.com/dashboard` | Switch State to **INDIVIDUAL** → Go to Home |
| `business.domain.com/acme/...` | Switch State to **BUSINESS (Acme)** → Go to Dashboard |

-----

## Frontend Integration Examples

### 1\. Context Provider (Universal)

This provider handles the logic for both Web (URL-based) and Mobile (Store-based).

```tsx
// contexts/BusinessContext.tsx
import { create } from 'zustand';
import { detectContext } from '../utils/contextDetection';

interface AppState {
  activeContext: { type: 'INDIVIDUAL' | 'BUSINESS'; slug?: string };
  setContext: (ctx: any) => void;
}

export const useStore = create<AppState>((set) => ({
  activeContext: { type: 'INDIVIDUAL' },
  setContext: (ctx) => set({ activeContext: ctx }),
}));

// Web Component to sync URL to Store
export const WebContextSyncer = () => {
  const { setContext } = useStore();
  
  useEffect(() => {
    // On mount or navigation, read the URL
    const context = detectContext();
    setContext(context);
  }, [window.location.href]);

  return null;
};
```

### 2\. Context-Aware Layouts

```tsx
// App.tsx
export default function App() {
  const { activeContext } = useStore();

  return (
    <>
      <WebContextSyncer /> {/* Keeps store in sync with URL */}
      
      {activeContext.type === 'INDIVIDUAL' ? (
        <IndividualLayout>
           <Routes ... />
        </IndividualLayout>
      ) : (
        <BusinessLayout slug={activeContext.slug}>
           <Routes ... />
        </BusinessLayout>
      )}
    </>
  );
}
```

### 3\. Navigation / Context Switcher

```tsx
// Context Switcher Component (Web)
// Note: Uses standard <a> tags for cross-domain linking
<Dropdown label="Switch Context">
  <div className="label">Personal</div>
  <a href="https://app.domain.com/dashboard">
     My Individual Profile
  </a>

  <div className="label">My Businesses</div>
  {userBusinesses.map(business => (
    <a 
      key={business.id} 
      href={`https://business.domain.com/${business.slug}/dashboard`}
    >
      {business.name} ({business.role})
    </a>
  ))}
</Dropdown>
```