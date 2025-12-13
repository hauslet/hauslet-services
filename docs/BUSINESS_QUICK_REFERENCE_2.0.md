Here is the updated **Business Module - Quick Reference Guide** adapted for the **Subdomain + Global Store** architecture.

-----

# Business Module - Quick Reference Guide

## 🎯 TL;DR

  - **Multi-Zone Architecture:**
      - **Individual Context:** Lives at `app.domain.com`
      - **Business Context:** Lives at `business.domain.com/:slug`
  - **Context is Implicit:** The URL (Web) or Global Store (Mobile) determines where you are.
  - **Strict Separation:** You cannot view business listings while on the individual dashboard, and vice-versa.
  - **API Magic:** Context is passed via headers (`X-Tenant-Slug`), not just query arguments.

-----

## 📊 Quick Visual: The Two Zones

```
       YOU (User)
           │
    ┌──────┴──────┐
    ▼             ▼
  ZONE A        ZONE B
(Individual)   (Business)
 app.com       business.app.com
    │             │
    │             ├─ /acme-corp
    │             │  (Owner)
    │             │
    └─ /dashboard ├─ /beta-ltd
                  │  (Member)
                  │
                  └─ /city-realty
                     (Admin)
```

-----

## 🔑 Key Concepts (30-second version)

### 1\. Context Switching = Domain Switching

Moving from "Personal" to "Business" is a **full navigation event**, not just a router state change.

  - **`app.domain.com`** → Your personal space.
  - **`business.domain.com/acme`** → The Acme workspace.

### 2\. Ownership is Automatic

You don't "select" who owns a listing anymore. **Where you are determines who owns it.**

```
User is at: app.com             User is at: business.app.com/acme
       │                                       │
       ▼                                       ▼
Creates Listing                         Creates Listing
       │                                       │
Owner = YOU (Individual)                Owner = ACME (Business)
```

### 3\. Permissions (RBAC)

Permissions are checked against the **active context**.

| Role | Access Level |
|------|-------------------|
| **Owner** | Full control of the workspace. |
| **Admin** | Can manage listings & members, but can't delete business. |
| **Member** | Can create/edit listings, but no administrative powers. |
| **Viewer** | Read-only access. |

-----

## 🚀 Implementation Patterns

### Pattern 1: Global Store (The Brain)

```tsx
// store.ts (Zustand example)
interface AppState {
  activeContext: { type: 'INDIVIDUAL' | 'BUSINESS'; slug?: string };
}

// 1. Web: Syncs from URL
// 2. Mobile: Syncs from Storage
```

### Pattern 2: Context Switcher (Web)

```tsx
// ⚠️ MUST use <a> tags, not <Link>
<Dropdown>
  <div className="label">Personal</div>
  <a href="https://app.domain.com/dashboard">
    My Profile
  </a>

  <div className="label">My Businesses</div>
  {myBusinesses.map(biz => (
    <a href={`https://business.domain.com/${biz.slug}/dashboard`}>
      {biz.name}
    </a>
  ))}
</Dropdown>
```

### Pattern 3: Permission-Gated UI

```tsx
function BusinessActions() {
  // 1. Get permissions for CURRENT context
  const { data } = useQuery(GET_CURRENT_PERMISSIONS);
  const perms = data?.currentPermissions;

  return (
    <>
      {perms?.canCreateListings && <CreateListingButton />}
      {perms?.canManageMembers && <InviteMemberButton />}
    </>
  );
}
```

### Pattern 4: Context-Aware Form

```tsx
function CreateListingForm() {
  const { activeContext } = useStore();

  return (
    <form>
      {/* Visual Feedback Only */}
      <Banner>
        Creating listing for: 
        <strong>
          {activeContext.type === 'BUSINESS' 
            ? activeContext.slug 
            : 'Personal Profile'}
        </strong>
      </Banner>

      {/* No "Owner Select" needed - Backend handles it */}
      <input name="title" />
      <button>Submit</button>
    </form>
  );
}
```

-----

## 📋 Essential GraphQL Queries

### Get User's Businesses (For Navigation)

```graphql
query MyBusinesses {
  myBusinesses {
    id
    name
    slug
    role
  }
}
```

### Get Context-Specific Listings

*Note: The query is the same, but the **result changes** based on where the user is.*

```graphql
# Headers: X-Tenant-Slug (injected automatically)
query GetListings {
  listings {
    id
    title
    ownerType # Returns "INDIVIDUAL" or "BUSINESS"
    createdByUser { name }
  }
}
```

-----

## ⚡ Common UI Flows

### Flow 1: User Switches Context

```
1. User clicks "Acme Corp" in navbar
   └─> <a href="https://business.app.com/acme-corp/dashboard">

2. Browser loads new subdomain
   └─> Auth Cookie (shared) keeps user logged in

3. React App initializes
   └─> detectContext() sees "business." + "acme-corp"
   └─> Sets Store: { type: 'BUSINESS', slug: 'acme-corp' }

4. API Client configures itself
   └─> Adds Header: "X-Tenant-Slug: acme-corp"
```

### Flow 2: Creating a Listing

```
1. User is on business.app.com/acme/dashboard

2. Clicks "Create Listing"

3. Submits Form
   └─> Mutation sent with header X-Tenant-Slug: acme

4. Backend Middleware
   └─> Resolves 'acme' -> BusinessID
   └─> Checks: Does User have permission in BusinessID?
   └─> Creates listing with OwnerID = BusinessID
```

-----

## 🎨 UI Components You'll Need

### Context Banner

```tsx
// Shows at top of Business Dashboard
<div className="bg-blue-100 p-2 flex justify-between">
  <span>🏢 <strong>{businessName}</strong> Workspace</span>
  <a href="https://app.domain.com/dashboard">Exit to Personal</a>
</div>
```

### Mobile Switcher (Drawer)

```tsx
// Mobile uses STATE, not URLs
<Drawer>
  <Pressable onPress={() => setContext({ type: 'INDIVIDUAL' })}>
    <Avatar src={user.photo} />
    <Text>Personal</Text>
  </Pressable>

  {businesses.map(b => (
    <Pressable onPress={() => setContext({ type: 'BUSINESS', slug: b.slug })}>
      <BusinessLogo src={b.logo} />
      <Text>{b.name}</Text>
    </Pressable>
  ))}
</Drawer>
```

-----

## 🔮 Future: Messaging

Messaging follows the same subdomain logic:

```
Context: app.domain.com       Context: business.domain.com/acme
Route:   /messages            Route:   /messages
    │                            │
    ▼                            ▼
Personal DMs                 Team Inbox
& Individual Listing         & Acme Listing
Inquiries                    Inquiries
```

-----

## ⚠️ Common Pitfalls to Avoid

### ❌ DON'T: Use React Router `<Link>` between contexts

```tsx
// BAD: This won't change the subdomain!
<Link to="/acme/dashboard">Switch to Acme</Link>
```

### ✅ DO: Use native anchor tags

```tsx
// GOOD: Forces browser to load new domain
<a href="https://business.domain.com/acme/dashboard">Switch to Acme</a>
```

### ❌ DON'T: Hardcode Business IDs in Mutations

```tsx
// BAD: Relying on form state
createListing({ variables: { businessId: selectedId } })
```

### ✅ DO: Trust the Header

```tsx
// GOOD: The interceptor handles the context
createListing({ variables: { title: "..." } })
```

### ❌ DON'T: Forget Mobile Deep Links

```tsx
// BAD: Email link only works on web
<a href="https://business.domain.com/acme">View</a>
```

### ✅ DO: Configure Universal Links

Ensure `business.domain.com/*` is mapped in your iOS/Android project config to open the app and trigger a state switch.

-----

## 📞 Quick Help

### "How do I know where I am?"

```tsx
const { activeContext } = useStore();
// { type: 'BUSINESS', slug: 'acme-corp' }
```

### "How do I get the current Business ID?"

On the web, you often only have the **slug** from the URL initially. You may need to resolve it:

```graphql
query {
  currentBusiness { id, name, permissions }
}
```

### "Why am I logged out when switching to business?"

Check your **Cookie Domain**. It must be set to `.domain.com` (note the leading dot) to be shared across subdomains.

-----

## 🎓 Mental Model

**Think of "Slack Workspaces"**

  * **Individual Context** is your "Home" screen where you can see all workspaces.
  * **Business Context** is inside a specific Slack Workspace.
  * To go to another business, you leave the current one and enter the next.
  * You are the same "User," but your environment changes completely.