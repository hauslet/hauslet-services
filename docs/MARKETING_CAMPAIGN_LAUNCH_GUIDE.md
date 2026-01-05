# Marketing Campaign Launch Guide for Hauslet

A practical, step-by-step guide for launching marketing campaigns - especially for those new to digital marketing.

---

## **🎯 Campaign Launch Strategy**

### **Phase 1: Start Small (Testing Phase)**

Don't spend big money initially! Start with small budgets to learn what works.

---

## **📱 Facebook & Instagram Ads (Best for Real Estate)**

### **Step 1: Set Up Facebook Ads Manager**

1. **Create a Facebook Business Page** (if you don't have one)
   - Go to facebook.com/pages/create
   - Name: "Hauslet - Find Your Dream Home"
   - Category: Real Estate

2. **Set Up Ads Manager**
   - Go to business.facebook.com
   - Click "Create Account" → Create a Business Manager account
   - Add your Facebook Page
   - Add a payment method (Naira credit card or bank account)

3. **Install Facebook Pixel** (Optional but recommended)
   - This tracks visitors on your website
   - Shows you which ads bring users who actually browse properties

---

### **Step 2: Create Your First Campaign**

#### **Campaign Structure:**
```
Campaign Level: "Hauslet - Property Inquiries"
├─ Ad Set 1: Lagos - Luxury Apartments
│  ├─ Ad A: Carousel (Pool + Gym + Interior)
│  └─ Ad B: Single Image (Living Room)
│
├─ Ad Set 2: Lagos - Budget Friendly
│  ├─ Ad A: Video Walkthrough
│  └─ Ad B: Static Image
│
└─ Ad Set 3: Abuja - New Listings
   ├─ Ad A: Carousel
   └─ Ad B: Single Image
```

---

### **Step 3: Set Up Campaign in Ads Manager**

**1. Click "Create" → Choose Campaign Objective:**
- Select: **"Traffic"** (drive people to your website)
- Name: `Hauslet_LagosProp_Jan2026`

**2. Set Budget:**
- **Testing Phase:** ₦5,000 - ₦10,000 per day
- **Scaling Phase:** ₦30,000 - ₦100,000 per day (after you find what works)

**3. Create Ad Set (Targeting):**

**Locations:**
- Target: Nigeria
- Or specific: Lagos, Abuja, Port Harcourt

**Demographics:**
- Age: 25-45 (prime renters/buyers)
- Gender: All
- Language: English

**Detailed Targeting:**
```
Interests to target:
- Real Estate
- Property Investment
- Interior Design
- Home Decor
- Zillow (yes, target people who search for Zillow!)
- Apartment Living
- House Hunting

Behaviors:
- Likely to move
- Recently moved
- Expats in Nigeria
```

**4. Create Your Ad:**

**Format Options:**

**Option A: Carousel Ad** (Best for multiple properties)
```
Headline: "Find Your Dream Home in Lagos"
Text: "Browse 500+ verified apartments and houses. 
      From ₦400k to ₦5M. Free viewings. 
      No Agent Fees. 🏡"

Card 1: Luxury 3BR with Pool - ₦2.5M/year
Card 2: Cozy 2BR in Lekki - ₦800k/year
Card 3: Modern Studio - ₦450k/year

Button: "Learn More"
```

**Option B: Single Image/Video Ad**
```
Image: Beautiful living room photo
Headline: "2BR Apartment in Lekki - ₦800k/year"
Text: "✅ Gated Estate ✅ 24/7 Security ✅ Gym & Pool
      Click to schedule a viewing!"
Button: "Learn More"
```

**5. Set Destination URL (CRITICAL!):**

This is where UTM tracking happens:

```
Destination URL Format:
https://hauslet.com/listings/[PROPERTY-ID]?utm_source=facebook&utm_medium=cpc&utm_campaign=lagos_apartments_jan2026&utm_content={{ad.id}}

Example:
https://hauslet.com/listings/abc-123-def?utm_source=facebook&utm_medium=cpc&utm_campaign=lekki_luxury_q1&utm_content=carousel_pool_v1
```

**Facebook Dynamic Parameters** (auto-fills ad info):
```
utm_source=facebook
utm_medium=cpc
utm_campaign=lagos_apartments_jan2026
utm_content={{ad.name}}
utm_term={{placement}}
```

---

### **Step 4: Launch & Monitor**

**Day 1-3: Let It Run**
- Don't touch anything
- Facebook needs time to optimize delivery

**Day 4-7: Analyze**
Check Ads Manager:
- **CTR (Click-Through Rate):** Above 1% is good for real estate
- **Cost Per Click:** ₦50-₦200 is typical in Nigeria
- **Cost Per Lead:** Track in Hauslet backend

**Week 2: Optimize**
- Turn off underperforming ads (low CTR, high cost)
- Double budget on winning ads
- Test new creatives

---

## **🎨 Instagram Ads (Same Platform as Facebook)**

Good news: **Instagram ads run through Facebook Ads Manager!**

Just check "Instagram" placement when creating ad set.

**Instagram-Specific Tips:**
- Use **Stories format** (9:16 vertical video/image)
- Shorter text (Instagram users scroll fast)
- High-quality, lifestyle images (not just property photos)

**Example Instagram Story Ad:**
```
[Video: 15-second pan through beautiful apartment]
Text Overlay: "Your Next Home Awaits 🏡"
Swipe Up: "View Property"
Link: hauslet.com/listings/xyz?utm_source=instagram&utm_medium=story
```

---

## **🔍 Google Ads (Search Intent)**

People actively searching "apartments for rent in Lagos" → HIGH INTENT!

### **Set Up Google Ads:**

1. **Go to ads.google.com**
2. **Create Campaign → Search**
3. **Goal:** Website Traffic

**Keywords to Target:**
```
Exact Match:
- [apartments for rent in lagos]
- [houses for sale in lekki]
- [2 bedroom apartment lagos]

Phrase Match:
- "cheap apartments in lagos"
- "luxury homes nigeria"

Broad Match Modifier:
- +apartments +rent +lagos +2bedroom
```

**Ad Example:**
```
Headline 1: Find Apartments in Lagos
Headline 2: 500+ Verified Properties
Headline 3: Free Viewings | No Agent Fees
Description: Browse 2BR, 3BR apartments from ₦400k/year. 
            Gated estates, security, amenities. View today!
URL: hauslet.com/search/lagos?utm_source=google&utm_medium=cpc&utm_campaign=search_lagos
```

**Budget:**
- Start: ₦10,000/day
- Cost Per Click: ₦100-₦500 (higher than Facebook, but MUCH higher intent)

---

## **📧 Email Marketing (Lowest Cost)**

### **Build an Email List:**

**Method 1: Lead Magnets**
- "Download our Free Lagos Rental Guide PDF"
- "Get Weekly New Listings Alert"
- "Landlord's Guide to Renting Out Property"

**Method 2: Existing Users**
- Users who saved properties but didn't contact owners
- Users who viewed 3+ properties this week

**Email Campaign Example:**
```
Subject: "15 New Apartments in Lekki - Just Added This Week!"

Body:
Hi [Name],

We just added 15 new verified apartments in Lekki Phase 1. 
Here are 3 that match what you were looking for:

1. Modern 2BR with Gym - ₦850k/year [View Property]
2. Spacious 3BR in Gated Estate - ₦1.2M/year [View Property]
3. Cozy 1BR for Singles - ₦500k/year [View Property]

CTA: Browse All New Listings
Link: hauslet.com/new-listings?utm_source=email&utm_campaign=weekly_newsletter_jan2026
```

**Tools:**
- **Mailchimp** (free up to 500 subscribers)
- **SendGrid** (developer-friendly, pay-as-you-go)
- **ConvertKit** (good for content creators)

---

## **👥 Influencer Marketing**

Partner with Nigerian lifestyle/real estate influencers.

### **How to Find Influencers:**

**Instagram:**
- Search: #LagosHomes #NaijaRealEstate #LagosLifestyle
- Look for: 10k-100k followers (micro-influencers are cheaper and more engaged)

**TikTok:**
- Search: "Lagos apartments" "moving to Lagos"
- Trending creators showing lifestyle content

### **Collaboration Types:**

**Option 1: Sponsored Post**
```
Cost: ₦50,000 - ₦300,000 per post (depends on follower count)

Example:
Influencer posts:
"Hey guys! Just found this amazing platform for finding apartments in Lagos. 
No agent fees, verified listings, and free viewings! 🏡
Check out @hauslet_ng or click the link in my bio!
[Link: hauslet.com/featured?utm_source=instagram&utm_medium=influencer&utm_campaign=lifestyle_collab_jan2026&utm_content=@influencername]"
```

**Option 2: Affiliate Program**
```
Pay per conversion instead of flat fee:
- Influencer gets custom link: hauslet.com?ref=influencername
- Track conversions (successful rentals/sales)
- Pay: ₦10,000 - ₦50,000 per successful conversion
```

**Option 3: Content Partnership**
```
Free apartment listing for 2 months in exchange for:
- 3 Instagram posts
- 5 Stories
- 1 TikTok video
- Property walkthrough video content
```

---

## **📊 Measuring Success (Analytics)**

### **Track These Metrics:**

**1. Lead Volume:**
```sql
SELECT 
  DATE(created_at) as date,
  utm_params->>'utm_campaign' as campaign,
  COUNT(*) as leads
FROM leads
WHERE created_at >= '2026-01-01'
GROUP BY DATE(created_at), utm_params->>'utm_campaign'
ORDER BY date DESC;
```

**2. Campaign ROI:**
```
Campaign: Facebook - Lagos Apartments Jan 2026
├─ Ad Spend: ₦150,000
├─ Leads Generated: 87
├─ Cost Per Lead: ₦1,724
├─ Viewing Requests: 23 (26% conversion)
├─ Actual Viewings: 12 (52% show-up rate)
└─ Signed Leases: 4 (33% close rate)

Revenue (if charging landlords):
4 conversions × ₦50,000 commission = ₦200,000
ROI: (₦200k - ₦150k) / ₦150k = 33% profit 🎉
```

**3. Channel Comparison:**
```
Channel           | Leads | Cost/Lead | Conversion | Best For
------------------|-------|-----------|------------|------------------
Facebook Ads      |  87   |  ₦1,724   |    26%     | Volume + Awareness
Instagram Stories |  45   |  ₦2,111   |    31%     | Young professionals
Google Search     |  23   |  ₦4,348   |    48%     | High intent buyers
Email Newsletter  | 156   |    ₦96    |    12%     | Nurturing leads
Influencer Post   |  67   |  ₦1,493   |    19%     | Brand awareness
```

---

## **💰 Budget Recommendations**

### **Starting Out (Month 1-3):**
```
Total: ₦300,000/month

Breakdown:
├─ Facebook/Instagram: ₦150,000 (50%)
├─ Google Ads: ₦80,000 (27%)
├─ Influencer Test: ₦50,000 (17%)
└─ Email Marketing Tool: ₦20,000 (6%)
```

### **Growth Phase (Month 4-6):**
```
Total: ₦1,000,000/month

Breakdown:
├─ Facebook/Instagram: ₦500,000 (50%)
├─ Google Ads: ₦300,000 (30%)
├─ Influencer Partnerships: ₦150,000 (15%)
└─ Email + Content: ₦50,000 (5%)
```

---

## **🎯 Quick Win Strategies**

### **Strategy 1: Retargeting**
Target people who visited your site but didn't inquire:

**Facebook Pixel Setup:**
1. Install Facebook Pixel on hauslet.com
2. Create Custom Audience: "Visited property page but didn't contact owner"
3. Run ads to this audience with: "Still looking? Schedule a free viewing!"

**Cost:** Much cheaper (₦30-₦50 per click vs. ₦100-₦200)
**Conversion:** 2-3x higher than cold traffic

### **Strategy 2: Lookalike Audiences**
After getting 100+ leads from Facebook:
1. Upload your lead email list to Facebook
2. Create "Lookalike Audience" (1% - 5%)
3. Facebook finds similar people
4. Target them with new ads

**Result:** Higher conversion rates at lower cost

### **Strategy 3: WhatsApp Business Integration**
Add "Contact via WhatsApp" button on listings:
```
Link: https://wa.me/2348012345678?text=Hi,%20I'm%20interested%20in%20the%202BR%20apartment%20in%20Lekki%20(Property%20ID:%20abc-123)

Track as:
source: "whatsapp"
utm_campaign: "direct_messaging"
```

**Why:** Nigerians LOVE WhatsApp. Higher response rates than email!

---

## **🚨 Common Mistakes to Avoid**

| ❌ Don't Do This | ✅ Do This Instead |
|------------------|-------------------|
| Targeting too broad: "All of Nigeria, ages 18-65" = wasted money | Target specific: "Lagos, 25-40, interested in real estate" |
| No UTM tracking: Can't tell which ads work | Always use UTM parameters: Know your ROI |
| One ad creative only: Audience gets blind to it | Test 3-5 variations: Different images, headlines, copy |
| Checking daily, making changes: Panic mode kills campaigns | Wait 5-7 days: Let Facebook optimize before judging |
| Ignoring mobile: 80% of Nigerian traffic is mobile | Mobile-first design: Fast loading, easy forms |

---

## **📚 Learning Resources**

### **Free Courses:**
- **Facebook Blueprint:** facebook.com/business/learn (Official Facebook training)
- **Google Skillshop:** skillshop.withgoogle.com (Free Google Ads certification)
- **HubSpot Academy:** academy.hubspot.com (Marketing fundamentals)

### **YouTube Channels:**
- **Ben Heath** - Facebook Ads expert
- **Loves Data** - Google Ads & Analytics
- **Neil Patel** - General digital marketing

### **Nigerian-Specific:**
- Follow Nigerian digital marketing agencies on LinkedIn
- Join Facebook groups: "Digital Marketing Nigeria", "Nigerian Business Owners"

---

## **🎉 Your 30-Day Action Plan**

### **Week 1: Setup**
- [ ] Create Facebook Business Manager
- [ ] Set up Facebook Page
- [ ] Add payment method
- [ ] Install Facebook Pixel on Hauslet
- [ ] Create Google Ads account

### **Week 2: First Campaign**
- [ ] Create 1 Facebook campaign (₦10k/day budget)
- [ ] 2 ad sets (different audiences)
- [ ] 2 ads per set (A/B test)
- [ ] Track UTM parameters
- [ ] Monitor leads in Hauslet backend

### **Week 3: Analyze & Optimize**
- [ ] Check which ads perform best
- [ ] Turn off losers, boost winners
- [ ] Test new ad creatives
- [ ] Launch Google Ads test campaign (₦5k/day)

### **Week 4: Scale**
- [ ] Double budget on winning campaigns
- [ ] Set up retargeting
- [ ] Plan influencer outreach
- [ ] Set up email marketing tool

---

## **🔗 UTM Parameter Best Practices**

Always structure your tracking URLs consistently:

```
Base URL: https://hauslet.com/listings/{listing-id}

UTM Parameters:
?utm_source=     [facebook | instagram | google | email | whatsapp]
&utm_medium=     [cpc | organic | email | referral | social]
&utm_campaign=   [descriptive_name_with_underscores]
&utm_content=    [ad_variant_name]
&utm_term=       [keywords_or_targeting_info]

Example:
https://hauslet.com/listings/abc-123?utm_source=facebook&utm_medium=cpc&utm_campaign=lagos_luxury_q1_2026&utm_content=carousel_pool_v2&utm_term=25-35_property_investors
```

### **Naming Conventions:**

**utm_source:** Where traffic comes from
- `facebook`, `instagram`, `google`, `email`, `linkedin`, `twitter`, `tiktok`

**utm_medium:** Type of traffic
- `cpc` (paid ads), `organic`, `email`, `social`, `referral`, `affiliate`

**utm_campaign:** Specific campaign identifier
- Use underscores: `lagos_apartments_jan2026`
- Include: location + property_type + time_period
- Examples: `lekki_luxury_homes_q1`, `abuja_budget_rentals_promo`

**utm_content:** Ad variation (for A/B testing)
- `carousel_v1`, `single_image_v2`, `video_walkthrough`
- `headline_a`, `headline_b`

**utm_term:** Keywords or audience info (optional)
- `2bedroom_lekki`, `luxury_properties`
- `age_25-35_investors`

---

## **📱 WhatsApp Marketing Strategy**

### **Set Up WhatsApp Business API:**

**Option 1: WhatsApp Business App** (Free, basic)
- Download from Play Store/App Store
- Set up business profile
- Create quick replies for common questions
- Use WhatsApp Business Catalog for listings

**Option 2: WhatsApp Business API** (Advanced, paid)
- Integrate with Hauslet backend
- Auto-responses
- Bulk messaging (with user consent)
- Analytics

### **WhatsApp Campaign Ideas:**

**1. Direct Property Alerts:**
```
Message Template:
"Hi [Name]! 🏡

New 2BR apartment just listed in Lekki Phase 1:
✅ ₦850k/year
✅ Gated estate
✅ Pool & Gym
✅ Available now

Interested? Reply YES to schedule viewing
or click: hauslet.com/listings/xyz?utm_source=whatsapp
```

**2. WhatsApp Status Updates:**
```
Post daily property highlights in your Status
"🔥 Hot Deal: 3BR Duplex - ₦1.5M/year"
+ Link to listing with UTM tracking
```

**3. WhatsApp Groups:**
```
Create groups by interest:
- "Lagos Apartments Under 1M"
- "Luxury Properties Nigeria"
- "Buy & Invest Lagos"

Rules:
- Weekly new listings only
- No spam
- Ask before adding members
```

---

## **🎯 Conversion Rate Optimization (CRO)**

### **Landing Page Best Practices:**

**Must-Have Elements:**
1. **Clear Headline:** "Find Your Dream Apartment in Lagos"
2. **High-Quality Images:** Professional photos, virtual tours
3. **Trust Signals:** "500+ Verified Properties", "No Agent Fees"
4. **Simple Contact Form:** Name, Email, Phone, Message (4 fields max)
5. **Mobile Optimized:** 80% of traffic is mobile
6. **Fast Loading:** Under 3 seconds
7. **Clear CTA:** "Contact Owner", "Schedule Viewing"

### **A/B Testing Ideas:**

**Test 1: CTA Button Color**
- Version A: Green button "Contact Owner"
- Version B: Blue button "Contact Owner"
- Measure: Which gets more clicks?

**Test 2: Form Length**
- Version A: 4 fields (Name, Email, Phone, Message)
- Version B: 2 fields (Email, Message) - Phone optional
- Measure: Which has higher completion rate?

**Test 3: Headline Copy**
- Version A: "Find Apartments in Lagos"
- Version B: "Find Your Dream Home - No Agent Fees"
- Version C: "500+ Verified Listings - Browse Free"
- Measure: Which drives more leads?

---

## **💡 Advanced Tactics**

### **1. Seasonal Campaigns**

**Back to School (August-September):**
```
Target: Parents, university students
Campaign: "Student Housing Near UNILAG"
Ad: "Safe, Affordable Student Apartments"
```

**New Year (January-February):**
```
Target: People with "New Year, New Home" goals
Campaign: "Fresh Start 2026 - New Apartments"
Ad: "New Year, New Address 🎉"
```

**Rainy Season (June-July):**
```
Target: People tired of flooding issues
Campaign: "Elevated Properties - No Flood Risk"
Ad: "Say Goodbye to Flooded Apartments"
```

### **2. Geo-Fencing**

Target people physically near your properties:

**Facebook Location Targeting:**
- Set radius: 5km around luxury apartment
- Target: "People who live in this area" or "People who were recently in this area"
- Ad: "Your Neighbor Just Moved In - See Why!"

### **3. Video Marketing**

**TikTok/Instagram Reels Ideas:**
```
1. "Day in the Life" - Show property walkthrough
2. "Before vs. After" - Renovation transformations
3. "Hidden Gems" - Unique features of properties
4. "Area Tour" - Neighborhood amenities
5. "Landlord Tips" - Educational content
```

**YouTube Strategy:**
```
Create channel: "Hauslet - Real Estate Nigeria"

Content:
- Property tours (15-minute detailed walkthroughs)
- Area guides ("Living in Lekki Phase 1: Pros & Cons")
- How-to videos ("How to Find Apartment in Lagos")
- Market insights ("Lagos Rental Prices 2026")

SEO optimize titles with keywords:
"2 Bedroom Apartment Tour Lekki Lagos Nigeria 2026"
```

---

## **📊 Dashboard Setup (Future Implementation)**

### **Recommended Metrics to Track:**

**1. Lead Source Performance**
```
Dashboard Widget: Lead Sources (Pie Chart)
- Facebook: 35%
- Instagram: 20%
- Google: 15%
- Direct/Organic: 18%
- Email: 7%
- Influencer: 5%
```

**2. Campaign ROI**
```
Table View:
Campaign Name         | Spend    | Leads | Cost/Lead | Conversions | ROI
---------------------|----------|-------|-----------|-------------|------
FB_Lagos_Jan2026     | ₦150k    | 87    | ₦1,724    | 4           | 33%
IG_Stories_Lekki     | ₦95k     | 45    | ₦2,111    | 3           | 58%
Google_Search_Lagos  | ₦100k    | 23    | ₦4,348    | 8           | 400%
```

**3. Funnel Visualization**
```
Ad Impressions: 50,000
↓ (2% CTR)
Clicks: 1,000
↓ (15% form start)
Form Started: 150
↓ (58% completion)
Leads: 87
↓ (26% contact)
Contacted: 23
↓ (17% conversion)
Conversions: 4
```

---

## **🎓 Key Takeaways**

✅ **Start small** - Test with ₦10k-₦20k/day budgets  
✅ **Track everything** - UTM parameters are mandatory  
✅ **Wait 7 days** - Let algorithms optimize before judging  
✅ **Test creatives** - Always run 3-5 ad variations  
✅ **Mobile-first** - 80% of Nigerian traffic is mobile  
✅ **WhatsApp wins** - Highest engagement channel in Nigeria  
✅ **Retargeting** - Cheapest conversions come from retargeting  
✅ **Measure ROI** - Know your cost per lead and cost per conversion  

---

**The Hauslet lead system is already built to track all of this automatically. Start small, test everything, scale what works!** 🚀📊
