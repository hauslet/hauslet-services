# Hauslet Leads: Hybrid Authentication + Marketing Integration Guide

## 🎯 Overview

The Hauslet Leads system now supports **Hybrid Authentication** - working seamlessly with both authenticated and anonymous users while providing powerful marketing campaign tracking.

---

## 🔄 How Hybrid Authentication Works

### Scenario 1: **Anonymous User** (No Login)

```txt
User clicks Facebook Ad → Hauslet Listing Page → "Contact Owner" Button
↓
Contact Form:
- Name: [Manual Entry]
- Email: [Manual Entry]
- Phone: [Manual Entry]
- Message: [Manual Entry]

System applies:
✅ Email validation
✅ Spam detection (full scoring)
✅ Rate limiting (3 inquiries/hour)
✅ Duplicate checking (1 hour window)
✅ UTM tracking captured

Database Record:
user_id: NULL
is_verified: FALSE
spam_score: (calculated based on content)
```

### Scenario 2: **Authenticated User** (Logged In)

```
User clicks Facebook Ad → Hauslet Listing Page → "Contact Owner" Button
↓
Contact Form:
- Name: [Auto-filled from profile] ✅
- Email: [Auto-filled from profile] ✅
- Phone: [Auto-filled from profile] ✅
- Message: [User types]

System applies:
✅ Auto-fill from verified profile
✅ Spam score reduced by 50% (trusted user)
✅ Same rate limiting
✅ UTM tracking captured

Database Record:
user_id: abc-123-def-456
is_verified: TRUE
spam_score: (reduced by 50%)
```

---

## 📢 Marketing Campaign Integration

### ✅ What Already Works

#### 1. **UTM Parameter Tracking**

The system automatically captures all UTM parameters from URLs:

```
Example URL from Facebook Ad:
https://hauslet.com/listings/xyz-789?
  utm_source=facebook
  &utm_medium=cpc
  &utm_campaign=lagos_apartments_jan2026
  &utm_content=ad_variant_a
  &utm_term=2bedroom+lekki

Stored in database as:
{
  "utm_source": "facebook",
  "utm_medium": "cpc",
  "utm_campaign": "lagos_apartments_jan2026",
  "utm_content": "ad_variant_a",
  "utm_term": "2bedroom+lekki"
}
```

#### 2. **Source Tracking**

Lead source field tracks the channel:

- `website` - Direct website visits
- `mobile_app` - Mobile app inquiries
- `facebook` - Facebook ads/posts
- `instagram` - Instagram campaigns
- `whatsapp` - WhatsApp Business integration
- `email_campaign` - Email marketing

#### 3. **Referrer Tracking**

Automatically captures which page/site sent the user:

```
referrer_url: "https://facebook.com/ads/..." 
```

#### 4. **Device/Browser Tracking**

```
user_agent: "Mozilla/5.0 (iPhone; CPU iPhone OS 15_0...)"
ip_address: "102.89.23.45"
```

---

## 📊 Marketing Use Cases

### Use Case 1: **Facebook/Instagram Ads Performance**

**Setup:**

```graphql
mutation CreateLead {
  createLead(input: {
    listingID: "listing-uuid-here"
    name: "Amaka Johnson"
    email: "amaka@example.com"
    message: "I'm interested in viewing this property"
    source: "facebook"
    utmParams: {
      utm_source: "facebook"
      utm_medium: "cpc"
      utm_campaign: "summer_rentals_2026"
      utm_content: "carousel_ad"
    }
  }) {
    id
    name
    email
    isVerified
    isAuthenticatedUser
    utmParams
    spamScore
  }
}
```

**Analytics Query (Later Implementation):**

```sql
-- Which campaign generated most leads?
SELECT 
  utm_params->>'utm_campaign' as campaign,
  COUNT(*) as total_leads,
  COUNT(CASE WHEN status = 'converted' THEN 1 END) as conversions,
  ROUND(COUNT(CASE WHEN status = 'converted' THEN 1 END)::decimal / COUNT(*) * 100, 2) as conversion_rate,
  COUNT(CASE WHEN is_verified = true THEN 1 END) as verified_leads
FROM leads
WHERE utm_params->>'utm_source' = 'facebook'
  AND created_at >= '2026-01-01'
GROUP BY utm_params->>'utm_campaign'
ORDER BY total_leads DESC;

-- Expected Output:
campaign                   | total_leads | conversions | conversion_rate | verified_leads
---------------------------|-------------|-------------|-----------------|---------------
summer_rentals_2026        |     124     |     18      |     14.52%      |      78
lekki_luxury_homes         |      89     |     12      |     13.48%      |      45
mainland_budget_friendly   |      67     |      5      |      7.46%      |      23
```

### Use Case 2: **A/B Testing Ad Creatives**

```
Campaign: "Lagos Apartments - January 2026"

Ad Variant A (Carousel with Pool Photo):
utm_content=carousel_pool
Leads: 45 | Conversions: 8 (17.8%)

Ad Variant B (Single Image with Interior):
utm_content=single_interior
Leads: 32 | Conversions: 3 (9.4%)

Winner: Variant A! 🎉
```

### Use Case 3: **Influencer Marketing Tracking**

```
Instagram Influencer Post:
https://hauslet.com/listings/xyz?
  utm_source=instagram
  &utm_medium=influencer
  &utm_campaign=lifestyle_collab_jan2026
  &utm_content=@lagoslifestyle_post123

Track:
- How many leads from @lagoslifestyle?
- Conversion rate vs. direct Instagram ads?
- ROI: ₦150,000 paid to influencer → 23 leads → 4 conversions
```

### Use Case 4: **Retargeting Campaign Optimization**

```sql
-- Find users who inquired but didn't convert (retargeting list)
SELECT DISTINCT email
FROM leads
WHERE status IN ('new', 'contacted', 'qualified')
  AND status != 'converted'
  AND created_at >= NOW() - INTERVAL '30 days'
  AND user_id IS NULL -- Anonymous users
ORDER BY created_at DESC;

-- Export to Facebook Custom Audience for retargeting
```

---

## 🎁 Benefits of Hybrid Approach

### For Users

✅ **No friction** - Anonymous users can inquire instantly  
✅ **Better UX** - Logged-in users get auto-filled forms  
✅ **Privacy choice** - Users decide when to authenticate

### For Property Owners

✅ **More leads** - No signup barrier means more inquiries  
✅ **Higher quality** - Verified leads marked clearly  
✅ **Filter options** - Can prioritize verified leads  
✅ **Campaign tracking** - Know which ads work best

### For the Platform

✅ **Spam protection** - Robust detection + rate limiting  
✅ **Marketing attribution** - Full UTM tracking  
✅ **User growth** - Authenticated users get benefits (auto-fill, notifications)  
✅ **Data quality** - Verified vs. anonymous clearly marked

---

## 🚀 API Examples

### Example 1: Anonymous Lead from Facebook Ad

```graphql
mutation {
  createLead(input: {
    listingID: "abc-123-def-456"
    name: "John Doe"
    email: "john@example.com"
    phone: "08012345678"
    message: "Is this 2BR apartment still available?"
    source: "website"
    utmParams: {
      utm_source: "facebook"
      utm_medium: "cpc"
      utm_campaign: "lekki_apartments"
      utm_content: "ad_version_1"
    }
  }) {
    id
    name
    email
    isVerified          # FALSE (anonymous)
    isAuthenticatedUser # FALSE
    spamScore           # Full score (e.g., 0.12)
    status             # "new"
  }
}
```

### Example 2: Authenticated User Lead (Auto-fill)

```graphql
# User already logged in (JWT token in header)
mutation {
  createLead(input: {
    listingID: "abc-123-def-456"
    name: ""              # Empty - will auto-fill from profile
    email: ""             # Empty - will auto-fill from profile
    phone: ""             # Empty - will auto-fill from profile
    message: "I'd like to schedule a viewing for this weekend"
    source: "mobile_app"
    utmParams: {
      utm_source: "app_notification"
      utm_campaign: "saved_searches_alert"
    }
  }) {
    id
    name               # "Amaka Johnson" (from profile)
    email              # "amaka@example.com" (from profile)
    isVerified         # TRUE (authenticated user)
    isAuthenticatedUser # TRUE
    spamScore          # Reduced score (e.g., 0.06 instead of 0.12)
    status            # "new"
  }
}
```

---

## 📈 Dashboard Metrics (Future Enhancement)

Potential analytics dashboard showing:

```
Campaign Performance:
├─ Total Leads: 342
├─ Verified Leads: 187 (54.7%)
├─ Anonymous Leads: 155 (45.3%)
├─ Spam Filtered: 23 (6.7%)
└─ Conversions: 47 (13.7%)

Top Campaigns:
1. Facebook - Lagos Rentals: 124 leads (18 conversions)
2. Instagram - Luxury Homes: 89 leads (12 conversions)
3. Google Ads - Buy Property: 67 leads (9 conversions)
4. Email Campaign - Q1 Newsletter: 45 leads (6 conversions)
5. Organic Search: 17 leads (2 conversions)

Conversion Funnel:
New (342) → Contacted (156) → Qualified (89) → Converted (47)
          45.6%               57.1%              52.8%
```

---

## 🔒 Security & Spam Protection

### Multi-Layer Protection

1. **Rate Limiting**
   - 3 inquiries/hour per email
   - 5 inquiries/hour per IP address
   - 10 inquiries/day per email/listing combo

2. **Spam Detection** (Pattern-Based)
   - Keyword scanning: "bitcoin", "viagra", "casino", etc.
   - URL density checks
   - Disposable email detection
   - All-caps message detection
   - Reduced by 50% for verified users

3. **Duplicate Prevention**
   - 1-hour window: Same email + listing = return existing lead
   - Prevents spamming landlords

4. **Data Validation**
   - Email format validation
   - Phone number format (optional)
   - Message length limits (10-5000 characters)

---

## 🎯 Best Practices for Marketing

### 1. Consistent UTM Naming

```
Good:
utm_campaign=q1_2026_lagos_rentals
utm_campaign=facebook_carousel_test_v2

Bad:
utm_campaign=test123
utm_campaign=campaign1
```

### 2. Source Attribution

```
Facebook Ads     → source: "website"  + utm_source: "facebook"
Instagram Ads    → source: "website"  + utm_source: "instagram"
Mobile App       → source: "mobile_app"
WhatsApp Button  → source: "whatsapp"
Email Campaign   → source: "email_campaign"
```

### 3. Track Everything

Always include full UTM parameters:

- utm_source (required): facebook, instagram, google, email
- utm_medium (required): cpc, display, email, organic
- utm_campaign (required): descriptive campaign name
- utm_content (optional): ad variant identifier
- utm_term (optional): keywords/targeting info

---

## 🎉 Summary

✅ **Hybrid authentication** - Works with and without login  
✅ **Marketing tracking** - Full UTM parameter support  
✅ **Spam protection** - Multi-layer security  
✅ **Better UX** - Auto-fill for authenticated users  
✅ **Higher conversions** - No friction for anonymous users  
✅ **Campaign ROI** - Track which ads actually work  

**The endpoint is PERFECT for Facebook/Instagram ads!** 🚀
