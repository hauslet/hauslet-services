# Business Module – Frontend User Stories (Subdomain + Global Store)

Stories are scoped to the new two-zone architecture: `app.domain.com` (individual) and `business.domain.com/:slug` (business workspace). Each story includes acceptance criteria for web; mobile variants call out deep-link/state specifics.

## Context Detection & State

- **As a returning user, I want the app to know which context I’m in based on the URL so layouts and data match my location.**
  - Accept: On `app.domain.com/*`, store sets `{type: 'INDIVIDUAL'}`; on `business.domain.com/{slug}/*`, store sets `{type: 'BUSINESS', slug}`.
  - Accept: Refreshing the page preserves the derived context without flicker.
  - Accept: Context derives from URL only (web); no manual dropdown for owner selection.

- **As a mobile user, I want the app to remember my last business workspace.**
  - Accept: Global store persists `activeContext` to storage; cold start restores it.
  - Accept: Deep links like `https://business.domain.com/acme/...` switch state to `{type: 'BUSINESS', slug: 'acme'}` and open the corresponding screen.

## Navigation & Switching

- **As a user, I want to switch between personal and business zones via full navigations so the correct domain loads.**
  - Accept: Context switch uses `<a href>` to the target domain (`app.domain.com`, `business.domain.com/{slug}`), not SPA routing.
  - Accept: After switch, user remains authenticated (shared cookie domain, `credentials: 'include'` on API calls).
  - Accept: UI shows a clear “Exit to Personal” link on business layouts and a “My Businesses” list linking to `business.domain.com/{slug}/dashboard`.

## Requests & Headers

- **As an API consumer, I want the current business context sent automatically so the backend routes to the right tenant.**
  - Accept: For `activeContext.type === 'BUSINESS'`, requests include header `X-Tenant-Slug: {slug}`.
  - Accept: For individual context, header is omitted.
  - Accept: Apollo/REST client is configured once (interceptor/link) to add/remove the header based on store state.

## Dashboards & Layouts

- **As an individual user, I want a personal dashboard that only shows my individual data.**
  - Accept: Personal dashboard loads on `app.domain.com/dashboard`.
  - Accept: Listings query filters to individual context (no business items shown).
  - Accept: “Create Business” entry is present.

- **As a business member, I want a workspace dashboard scoped to the active business.**
  - Accept: Business dashboard loads on `business.domain.com/{slug}/dashboard`.
  - Accept: Shows a context banner with business name/slug and a link back to personal dashboard.
  - Accept: All data widgets call APIs with `X-Tenant-Slug`; no manual business ID inputs.

## Listings

- **As a user, I want listing creation to inherit the current context automatically.**
  - Accept: In business zone, create listing mutation is sent without owner selectors; backend infers from header and sets `ownerType=BUSINESS`.
  - Accept: In personal zone, ownerType is individual by default; no business picker rendered.
  - Accept: Submit button is disabled if permissions disallow creation (based on current permissions query).

- **As a user, I want listing lists to reflect my active zone.**
  - Accept: In personal zone, list shows only individual listings.
  - Accept: In business zone, list shows only listings for the active business and includes “Created by” user info.
  - Accept: Loading/error states are scoped per zone and do not reuse cached cross-zone data.

## Permissions & RBAC

- **As a business member, I want the UI to hide or disable actions I’m not allowed to perform.**
  - Accept: Fetch `currentPermissions` (or equivalent) on business dashboards; gate create/edit/delete/publish/media/member/analytics/financial actions accordingly.
  - Accept: Owner/Admin see all actions; Member lacks delete/publish/manage-members; Viewer is read-only; Accountant only sees financial/analytics.
  - Accept: Buttons/toolbars reflect disabled state with explanatory tooltips when permission is missing.

## Authentication & Session Continuity

- **As a user, I want to stay logged in when moving between `app` and `business` subdomains.**
  - Accept: Frontend requests are sent with `credentials: 'include'`; session cookies issued on `.domain.com` are honored across subdomains.
  - Accept: On failed auth due to missing cookie, user is prompted to log in and then redirected to the intended domain/route.

## Invitations & Membership (Business Zone)

- **As an owner/admin, I want to invite members from the business dashboard.**
  - Accept: Invite form submits under business context (header set); permissions checked client-side for visibility.
  - Accept: Success state shows invite status/token details; errors surface permission failures.

- **As a member, I want to view the team roster with roles and permissions.**
  - Accept: Members list rendered only in business zone; shows role and key permission flags.
  - Accept: Manage-member actions (change role/permissions) visible only to Owner/Admin.

## Messaging (Future-ready)

- **As a user, I want messaging screens to separate personal and business conversations.**
  - Accept: Personal messages live at `app.domain.com/messages`; business inbox at `business.domain.com/{slug}/messages`.
  - Accept: Header injection follows the same context rules when loading business conversations.

## Mobile Parity Checklist

- Accept: Global store persists `activeContext`; API interceptor reads it.
- Accept: Deep links for both zones adjust state before rendering.
- Accept: Context switch UI (drawer) mirrors web’s personal/business options.

