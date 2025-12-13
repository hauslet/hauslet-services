# Business Module - Quick Reference Guide

## 🎯 TL;DR

- Users can have **individual accounts** AND belong to **multiple businesses**
- Everything has a **context**: individual OR business
- Businesses work like **GitHub Organizations**: team collaboration with role-based permissions
- Listings, messages, and dashboards are **separated by context**

---

## 📊 Quick Visual: User Operating Contexts

```
         YOU (as a User)
              │
    ┌─────────┴─────────┐
    │                   │
Individual          Businesses
Dashboard          (You're a member)
    │                   │
    │              ┌────┼────┐
    │              │    │    │
    ├─ My         Business Business Business
    │  Listings      A       B       C
    │               │       │       │
    ├─ My          Role:   Role:   Role:
    │  Messages    Owner   Admin   Member
    │               │       │       │
    └─ Create     Access: Access: Access:
       Business    Full   Most    Limited
```

---

## 🔑 Key Concepts (30-second version)

### 1. Context Switching
Users switch between contexts like switching GitHub accounts:
- **`/dashboard`** → Individual context (your personal stuff)
- **`/businesses/xyz`** → Business context (team stuff)

### 2. Ownership Model
Everything is owned by EITHER an individual OR a business:

```
Listing A              Listing B
├─ ownerType: "individual"    ├─ ownerType: "business"
├─ ownerID: {user-id}         ├─ ownerID: {business-id}
└─ Only you can access        ├─ createdBy: {user-id}
                              └─ Team can access (if permitted)
```

### 3. Permissions
What you can do in a business depends on your **role** + **custom permissions**:

| Role | Can Do Everything? | Default Access |
|------|-------------------|----------------|
| **Owner** | ✅ Yes (unlimited) | All permissions |
| **Admin** | ⚠️ Almost everything | Most permissions |
| **Member** | ❌ No | Limited permissions |
| **Viewer** | ❌ No | Read-only |
| **Accountant** | ❌ No | Financials only |

---

## 🚀 Implementation Patterns

### Pattern 1: Context Provider (MUST HAVE)

```tsx
// 1. Wrap your app
<BusinessProvider>
  <App />
</BusinessProvider>

// 2. Use anywhere
const { currentContext, currentBusinessId, switchToBusiness } = useBusinessContext();

// 3. Switch context
<button onClick={() => switchToBusiness(businessId)}>
  Switch to Business X
</button>
```

### Pattern 2: Dashboard Router

```tsx
function Dashboard() {
  const { currentContext, currentBusinessId } = useBusinessContext();

  // Show different dashboard based on context
  return currentContext === 'individual'
    ? <IndividualDashboard />
    : <BusinessDashboard businessId={currentBusinessId} />;
}
```

### Pattern 3: Permission-Gated UI

```tsx
function BusinessActions({ businessId }) {
  const { data } = useQuery(GET_MY_PERMISSIONS, {
    variables: { businessId }
  });

  const perms = data?.myBusinessPermissions;

  return (
    <>
      {perms?.canCreateListings && <CreateListingButton />}
      {perms?.canManageMembers && <InviteMemberButton />}
      {perms?.canEditBusiness && <EditBusinessButton />}
    </>
  );
}
```

### Pattern 4: Context-Aware Form

```tsx
function CreateListingForm() {
  const [ownerType, setOwnerType] = useState('individual');
  const [selectedBusiness, setSelectedBusiness] = useState(null);

  return (
    <>
      <select onChange={(e) => setOwnerType(e.target.value)}>
        <option value="individual">Personal Listing</option>
        <option value="business">Business Listing</option>
      </select>

      {ownerType === 'business' && (
        <BusinessSelector
          onSelect={setSelectedBusiness}
          filter={(b) => b.permissions.canCreateListings}
        />
      )}

      {/* Rest of form */}
    </>
  );
}
```

---

## 📋 Essential GraphQL Queries

### Get User's Businesses
```graphql
query MyBusinesses {
  myBusinesses {
    id
    name
    slug
    role              # "owner" | "admin" | "member"
    permissions {     # What you can do
      canCreateListings
      canEditListings
      canManageMembers
      # ... etc
    }
  }
}
```

### Get My Permissions in a Business
```graphql
query MyPerms($businessId: UUID!) {
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

### Get Context-Specific Listings
```graphql
# Individual listings
query MyListings {
  myListings(ownerType: INDIVIDUAL) {
    id
    title
  }
}

# Business listings
query BusinessListings($businessId: UUID!) {
  listings(filter: {
    ownerType: BUSINESS
    ownerID: $businessId
  }) {
    id
    title
    createdBy          # Which team member created it
    createdByUser {
      name
      email
    }
  }
}
```

---

## ⚡ Common UI Flows

### Flow 1: User Joins a Business

```
1. Business owner sends invitation
   └─> Mutation: inviteMember(email, role)

2. User receives email with token

3. User clicks "Accept" link
   └─> Mutation: acceptInvitation(token)

4. User now sees business in "My Businesses" dropdown

5. User clicks business name
   └─> Navigate to /businesses/{id}
   └─> Context switches to business
```

### Flow 2: Creating a Business Listing

```
1. User is on business dashboard (/businesses/xyz)

2. Clicks "Create Listing"

3. Form already knows context = "business"
   └─> ownerType: "business"
   └─> businessID: xyz

4. Submit mutation
   └─> Backend checks: Does user have CanCreateListings?
       ├─ Yes → Create listing
       └─ No  → Error: Insufficient permissions

5. Listing appears in business dashboard
   └─> Shows "Created by: {user.name}"
```

### Flow 3: Managing Team Permissions

```
1. Owner/Admin goes to /businesses/xyz/team

2. Sees list of members with their roles

3. Clicks "Edit" on a member

4. Toggle specific permissions:
   [x] Can create listings
   [x] Can edit listings
   [ ] Can delete listings  ← Disable this
   [ ] Can manage members

5. Save
   └─> Mutation: updateMemberPermissions(...)

6. Member's UI updates automatically
   └─> Delete button disappears from their view
```

---

## 🎨 UI Components You'll Need

### Navigation
```tsx
<Header>
  <Logo />

  {/* Context switcher */}
  <ContextMenu>
    <MenuItem onClick={switchToIndividual}>
      My Dashboard
    </MenuItem>
    <Divider />
    {myBusinesses.map(b => (
      <MenuItem onClick={() => switchToBusiness(b.id)}>
        {b.name} ({b.role})
      </MenuItem>
    ))}
    <Divider />
    <MenuItem>+ Create Business</MenuItem>
  </ContextMenu>
</Header>
```

### Dashboard Switcher
```tsx
{currentContext === 'business' && (
  <Alert>
    You're viewing <strong>{businessName}</strong>
    <button onClick={switchToIndividual}>
      Switch to personal dashboard
    </button>
  </Alert>
)}
```

### Permission-Aware Button
```tsx
<PermissionGate
  businessId={currentBusinessId}
  permission="canCreateListings"
>
  <CreateListingButton />
</PermissionGate>
```

### Business Selector (for forms)
```tsx
<BusinessSelect
  value={selectedBusiness}
  onChange={setSelectedBusiness}
  filter={(business) =>
    business.permissions.canCreateListings
  }
  placeholder="Select a business..."
  renderOption={(business) => (
    <div>
      <strong>{business.name}</strong>
      <span className="text-gray-500">
        {business.role} • {business.memberCount} members
      </span>
    </div>
  )}
/>
```

---

## 🔮 Future: Messaging

When messaging is implemented, it will follow the same pattern:

```
Individual Messages          Business Messages
/messages                    /businesses/{id}/messages
├─ Conversations about       ├─ Conversations about
│  YOUR personal listings    │  BUSINESS listings
│                           │
└─ Direct messages to you    └─ Shared inbox for team
                                (if has permission)
```

Messages will have:
- `ownerType`: "individual" | "business"
- `ownerID`: user ID or business ID
- `businessID`: (if business context)

---

## ⚠️ Common Pitfalls to Avoid

### ❌ DON'T: Mix contexts
```tsx
// BAD: Showing business listings on individual dashboard
<IndividualDashboard>
  <MyListings />
  <BusinessListings businessId={someId} /> {/* ❌ Wrong! */}
</IndividualDashboard>
```

### ✅ DO: Keep contexts separate
```tsx
// GOOD: Clear separation
{currentContext === 'individual' ? (
  <IndividualDashboard>
    <MyListings />
  </IndividualDashboard>
) : (
  <BusinessDashboard businessId={currentBusinessId}>
    <BusinessListings businessId={currentBusinessId} />
  </BusinessDashboard>
)}
```

### ❌ DON'T: Show actions without permission check
```tsx
// BAD: Always showing delete button
<ListingCard>
  <DeleteButton /> {/* ❌ User might not have permission! */}
</ListingCard>
```

### ✅ DO: Check permissions first
```tsx
// GOOD: Permission-gated action
<ListingCard>
  {permissions?.canDeleteListings && <DeleteButton />}
</ListingCard>
```

### ❌ DON'T: Assume ownerType
```tsx
// BAD: Hardcoded assumption
const ownerId = user.id; // ❌ Might be business ID!
```

### ✅ DO: Use the actual ownerType
```tsx
// GOOD: Check ownerType first
const ownerId = listing.ownerType === 'individual'
  ? user.id
  : selectedBusinessId;
```

---

## 📞 Quick Help

### "How do I know if I'm in business context?"
```tsx
const { currentContext } = useBusinessContext();
if (currentContext === 'business') {
  // You're in business context
}
```

### "How do I check if user can do X in a business?"
```tsx
const { data } = useQuery(GET_MY_PERMISSIONS, {
  variables: { businessId }
});

if (data?.myBusinessPermissions?.canCreateListings) {
  // User can create listings
}
```

### "How do I get all businesses user belongs to?"
```graphql
query {
  myBusinesses {
    id
    name
    role
    permissions { ... }
  }
}
```

### "How do I filter listings by context?"
```tsx
// Individual
const { data } = useQuery(GET_LISTINGS, {
  variables: { ownerType: 'INDIVIDUAL' }
});

// Business
const { data } = useQuery(GET_LISTINGS, {
  variables: {
    ownerType: 'BUSINESS',
    ownerID: businessId
  }
});
```

---

## 🎓 Mental Model

Think of it like this:

**You as a user** = One person with many hats

**Individual context** = Your personal hat (your private stuff)

**Business contexts** = Different team hats you wear
- Hat 1: Owner at Company A (can do everything)
- Hat 2: Admin at Company B (can do most things)
- Hat 3: Member at Company C (can do some things)

**The app needs to know which hat you're wearing** to show the right dashboard and permissions.

---

## 📚 For More Details

See the full architecture guide: [BUSINESS_ARCHITECTURE.md](./BUSINESS_ARCHITECTURE.md)
