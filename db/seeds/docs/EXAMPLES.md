# Advanced Seeding Examples

This document provides implementation examples for the remaining seeders.

## Listing Seeder (Complete Implementation)

```go
package seeders

import (
 "fmt"
 "strings"
 "time"

 propertySchema "hauslet/internal/modules/property/repository/schema"
 "hauslet/db/seeds/data"
 "hauslet/db/seeds/utils"

 "github.com/google/uuid"
)

func SeedListing(ctx *SeedContext) error {
 config := ctx.Config.(*SeedConfig)

 // Get all properties
 var properties []struct {
  ID       uuid.UUID
  OwnerID  uuid.UUID
  City     string
  Area     string
  Type     string
  Bedrooms *int
 }

 if err := ctx.DB.Table("properties").
  Select("id, owner_id, city, area, property_type as type, bedrooms").
  Scan(&properties).Error; err != nil {
  return fmt.Errorf("failed to fetch properties: %w", err)
 }

 if len(properties) == 0 {
  return fmt.Errorf("no properties found - seed property module first")
 }

 // Calculate how many of each listing type
 shortletCount := int(float64(config.ListingCount) * config.ShortletRatio)
 rentCount := int(float64(config.ListingCount) * config.RentRatio)
 saleCount := config.ListingCount - shortletCount - rentCount

 // Create shortlet listings
 for i := 0; i < shortletCount; i++ {
  property := utils.RandomChoice(properties)
  if err := createListing(ctx, property, "shortlet"); err != nil {
   return err
  }
 }

 // Create rental listings
 for i := 0; i < rentCount; i++ {
  property := utils.RandomChoice(properties)
  if err := createListing(ctx, property, "rent"); err != nil {
   return err
  }
 }

 // Create sale listings
 for i := 0; i < saleCount; i++ {
  property := utils.RandomChoice(properties)
  if err := createListing(ctx, property, "sale"); err != nil {
   return err
  }
 }

 return nil
}

func createListing(ctx *SeedContext, property struct {
 ID       uuid.UUID
 OwnerID  uuid.UUID
 City     string
 Area     string
 Type     string
 Bedrooms *int
}, listingType string) error {

 // Get location tier for pricing
 tier := getLocationTier(property.City, property.Area)

 // Generate title
 bedroomCount := "1"
 if property.Bedrooms != nil && *property.Bedrooms > 0 {
  bedroomCount = fmt.Sprintf("%d", *property.Bedrooms)
 }

 titleTemplate := utils.RandomChoice(data.PropertyTitles[property.Type])
 title := utils.FormatTemplate(titleTemplate, map[string]string{
  "bedroom": bedroomCount,
  "area":    property.Area,
 })

 // Generate description
 description := utils.RandomChoice(data.PropertyDescriptions)
 description = strings.ReplaceAll(description, "{area}", property.Area)

 // Generate slug from title
 slug := strings.ToLower(title)
 slug = strings.ReplaceAll(slug, " ", "-")
 slug = slug + "-" + utils.RandomString(6)

 // Get price based on tier and type
 price := data.GetPriceForListing(tier, listingType)

 // Determine status (90% published, 10% draft)
 status := propertySchema.ListingStatusActive
 published := true
 if utils.RandomBoolWithProbability(0.10) {
  status = propertySchema.ListingStatusDraft
  published = false
 }

 // Create listing
 listing := propertySchema.Listing{
  ID:         uuid.New(),
  PropertyID: property.ID,
  OwnerID:    property.OwnerID,
  OwnerType:  propertySchema.OwnerLandlord, // Simplify for now

  Slug:        slug,
  Title:       title,
  Description: description,
  Currency:    propertySchema.CurrencyNGN,

  ListingType: propertySchema.ListingType(listingType),
  Status:      status,
  Published:   published,

  LatestReviewStatus: propertySchema.ReviewStatusApproved,

  ViewCount: utils.RandomInt(0, 500),

  CreatedAt: utils.RandomPastDate(180),
  UpdatedAt: time.Now(),
 }

 if published {
  publishedAt := listing.CreatedAt.Add(24 * time.Hour)
  listing.PublishedAt = &publishedAt
 }

 // Add type-specific details
 switch listingType {
 case "shortlet":
  listing.ShortletDetails = &propertySchema.ShortletDetail{
   PricePerNight:     float64(price),
   MinNights:         utils.RandomInt(1, 3),
   MaxNights:         utils.RandomInt(14, 90),
   CleaningFee:       float64(utils.RandomInt(5000, 20000)),
   CautionFee:        float64(utils.RandomInt(10000, 50000)),
   MaxGuests:         utils.RandomInt(2, 8),
   CheckInTime:       "14:00",
   CheckOutTime:      "11:00",
   InstantBookable:   utils.RandomBool(),
   AllowPets:         utils.RandomBoolWithProbability(0.3),
   AllowSmoking:      utils.RandomBoolWithProbability(0.2),
   AllowEvents:       utils.RandomBoolWithProbability(0.4),
  }
  listing.HasCalendar = true

 case "rent":
  paymentFrequency := utils.RandomChoice([]string{"monthly", "yearly"})
  listing.RentalDetails = &propertySchema.RentalDetail{
   RentAmount:        float64(price),
   PaymentFrequency:  paymentFrequency,
   SecurityDeposit:   float64(price), // Usually same as rent
   ServiceCharge:     float64(utils.Percentage(float64(price), 5, 15)),
   MinLeasePeriod:    12, // months
   AllowPets:         utils.RandomBoolWithProbability(0.3),
   FurnishingStatus:  string(utils.RandomChoice(data.FurnishingTypes)),
  }

 case "sale":
  listing.SaleDetails = &propertySchema.SaleDetail{
   SalePrice:        float64(price),
   LegalFees:        float64(utils.Percentage(float64(price), 1, 3)),
   PropertyTaxYearly: float64(utils.RandomInt(50000, 500000)),
   Negotiable:       utils.RandomBool(),
   TitleType:        utils.RandomChoice([]string{"C of O", "Deed of Assignment", "Governor's Consent"}),
  }
 }

 if err := ctx.DB.Create(&listing).Error; err != nil {
  return fmt.Errorf("failed to create listing: %w", err)
 }

 // Create media for listing
 if ctx.Config.(*SeedConfig).SeedMedia {
  if err := createListingMedia(ctx, listing.ID, property.Type); err != nil {
   return fmt.Errorf("failed to create media: %w", err)
  }
 }

 return nil
}

func createListingMedia(ctx *SeedContext, listingID uuid.UUID, propertyType string) error {
 // Create 3-8 images
 imageCount := utils.RandomInt(3, 8)

 for i := 0; i < imageCount; i++ {
  media := propertySchema.ListingMedia{
   ID:         uuid.New(),
   ListingID:  listingID,
   MediaType:  propertySchema.MediaTypeImage,
   URL:        fmt.Sprintf("https://picsum.photos/800/600?random=%d", utils.RandomInt(1, 1000)),
   Caption:    generateMediaCaption(i, propertyType),
   DisplayOrder: i,
   IsPrimary:  i == 0, // First image is primary
   CreatedAt:  time.Now(),
  }

  if err := ctx.DB.Create(&media).Error; err != nil {
   return err
  }
 }

 return nil
}

func generateMediaCaption(index int, propertyType string) string {
 captions := []string{
  "Living Room",
  "Master Bedroom",
  "Kitchen",
  "Bathroom",
  "Exterior View",
  "Dining Area",
  "Balcony",
  "Parking Space",
 }

 if index < len(captions) {
  return captions[index]
 }

 return "Property View"
}

func getLocationTier(city, area string) string {
 // Premium areas
 premiumAreas := []string{
  "Victoria Island", "Ikoyi", "Lekki Phase 1", "Banana Island",
  "Maitama", "Asokoro", "Wuse 2", "GRA Phase 2", "Old GRA",
 }

 for _, premium := range premiumAreas {
  if strings.Contains(area, premium) {
   return "premium"
  }
 }

 // Budget areas
 budgetAreas := []string{"Gbagada", "Ojota"}
 for _, budget := range budgetAreas {
  if strings.Contains(area, budget) {
   return "budget"
  }
 }

 return "mid"
}
```

## Booking Seeder Pattern

```go
func SeedBooking(ctx *SeedContext) error {
 config := ctx.Config.(*SeedConfig)

 // Get shortlet listings with calendars
 var listings []struct {
  ID       uuid.UUID
  OwnerID  uuid.UUID
  Price    float64
 }

 ctx.DB.Raw(`
  SELECT l.id, l.owner_id, 
         COALESCE(l.shortlet_details->>'price_per_night', '0')::float as price
  FROM listings l
  WHERE l.listing_type = 'shortlet'
    AND l.has_calendar = true
    AND l.status = 'active'
 `).Scan(&listings)

 // Get guests (non-owners)
 var guests []uuid.UUID
 ctx.DB.Table("users").
  Select("id").
  Where("role = ?", "user").
  Limit(20).
  Scan(&guests)

 for i := 0; i < config.BookingCount; i++ {
  listing := utils.RandomChoice(listings)
  guest := utils.RandomChoice(guests)

  // Ensure guest != owner
  if guest == listing.OwnerID {
   continue
  }

  if err := createBooking(ctx, listing, guest); err != nil {
   return err
  }
 }

 return nil
}

func createBooking(ctx *SeedContext, listing struct {
 ID       uuid.UUID
 OwnerID  uuid.UUID
 Price    float64
}, guestID uuid.UUID) error {
 
 // Random check-in/out dates
 checkIn := utils.RandomDateInRange(-60, 60) // 60 days past to 60 days future
 nights := utils.RandomInt(2, 7)
 checkOut := checkIn.AddDate(0, 0, nights)

 totalPrice := listing.Price * float64(nights)

 // Determine status based on dates
 status := determineBookingStatus(checkIn, checkOut)

 booking := bookingSchema.Booking{
  ID:         uuid.New(),
  ListingID:  listing.ID,
  GuestID:    guestID,
  GuestCount: utils.RandomInt(1, 4),
  Status:     status,
  CheckIn:    checkIn,
  CheckOut:   checkOut,
  TotalPrice: totalPrice,
  Currency:   "NGN",
  CreatedAt:  checkIn.Add(-7 * 24 * time.Hour), // Booked 7 days before
 }

 return ctx.DB.Create(&booking).Error
}

func determineBookingStatus(checkIn, checkOut time.Time) bookingSchema.BookingStatus {
 now := time.Now()

 if checkOut.Before(now) {
  return bookingSchema.BookingStatusCompleted
 } else if checkIn.Before(now) && checkOut.After(now) {
  return bookingSchema.BookingStatusActive
 } else {
  return bookingSchema.BookingStatusConfirmed
 }
}
```

## Review Seeder Pattern

```go
func SeedReview(ctx *SeedContext) error {
 // Get completed bookings
 var bookings []struct {
  ID        uuid.UUID
  ListingID uuid.UUID
  GuestID   uuid.UUID
  OwnerID   uuid.UUID
 }

 ctx.DB.Raw(`
  SELECT b.id, b.listing_id, b.guest_id, l.owner_id
  FROM bookings b
  JOIN listings l ON b.listing_id = l.id
  WHERE b.status = 'completed'
    AND b.check_out < NOW()
 `).Scan(&bookings)

 for _, booking := range bookings {
  // 70% chance guest reviews the listing
  if utils.RandomBoolWithProbability(0.70) {
   createReview(ctx, booking, "guest_to_listing")
  }

  // 60% chance host reviews the guest
  if utils.RandomBoolWithProbability(0.60) {
   createReview(ctx, booking, "host_to_guest")
  }
 }

 return nil
}

func createReview(ctx *SeedContext, booking struct {
 ID        uuid.UUID
 ListingID uuid.UUID
 GuestID   uuid.UUID
 OwnerID   uuid.UUID
}, reviewType string) error {

 // Generate rating (normal distribution around 4-5)
 rating := generateRating()

 var review reviewSchema.Review
 
 if reviewType == "guest_to_listing" {
  review = reviewSchema.Review{
   ID:         uuid.New(),
   ReviewerID: booking.GuestID,
   TargetType: "listing",
   TargetID:   booking.ListingID,
   BookingID:  &booking.ID,
   Rating:     rating,
   Comment:    generateReviewComment(rating, "guest"),
   CreatedAt:  time.Now().Add(-time.Duration(utils.RandomInt(1, 30)) * 24 * time.Hour),
  }
 } else {
  review = reviewSchema.Review{
   ID:         uuid.New(),
   ReviewerID: booking.OwnerID,
   TargetType: "user",
   TargetID:   booking.GuestID,
   BookingID:  &booking.ID,
   Rating:     rating,
   Comment:    generateReviewComment(rating, "host"),
   CreatedAt:  time.Now().Add(-time.Duration(utils.RandomInt(1, 30)) * 24 * time.Hour),
  }
 }

 return ctx.DB.Create(&review).Error
}

func generateRating() int {
 // 60% chance of 5 stars, 25% of 4 stars, 10% of 3 stars, 5% of 1-2 stars
 r := utils.RandomFloat(0, 1)
 if r < 0.60 {
  return 5
 } else if r < 0.85 {
  return 4
 } else if r < 0.95 {
  return 3
 }
 return utils.RandomInt(1, 2)
}

func generateReviewComment(rating int, reviewerType string) string {
 if reviewerType == "guest" {
  positive := []string{
   "Amazing property! Very clean and exactly as described.",
   "Loved our stay here. Host was very responsive and helpful.",
   "Perfect location, great amenities. Would definitely book again!",
   "The property exceeded our expectations. Highly recommend!",
  }

  neutral := []string{
   "Good property overall. A few minor issues but nothing major.",
   "Decent stay. Property was clean but a bit dated.",
   "Fine for the price. Could use some updates.",
  }

  negative := []string{
   "Property was not as described. Several issues with cleanliness.",
   "Had problems with the power supply. Host was slow to respond.",
   "Would not recommend. Better options available for the price.",
  }

  if rating >= 4 {
   return utils.RandomChoice(positive)
  } else if rating == 3 {
   return utils.RandomChoice(neutral)
  }
  return utils.RandomChoice(negative)
 }

 // Host reviewing guest
 positive := []string{
  "Great guest! Very respectful and left the property in excellent condition.",
  "Wonderful guests. Great communication and followed all house rules.",
  "Excellent guests. Would happily host them again!",
 }

 neutral := []string{
  "Good guests overall. Left the property reasonably clean.",
  "Decent guests. A few minor issues but nothing serious.",
 }

 if rating >= 4 {
  return utils.RandomChoice(positive)
 }
 return utils.RandomChoice(neutral)
}
```

## Business Seeder Pattern

```go
func SeedBusiness(ctx *SeedContext) error {
 config := ctx.Config.(*SeedConfig)

 // Get users who can be business owners
 var users []uuid.UUID
 ctx.DB.Table("users").
  Select("id").
  Where("role IN (?)", []string{"user", "admin"}).
  Limit(10).
  Scan(&users)

 for i := 0; i < config.BusinessCount; i++ {
  owner := utils.RandomChoice(users)
  if err := createBusiness(ctx, owner); err != nil {
   return err
  }
 }

 return nil
}

func createBusiness(ctx *SeedContext, ownerID uuid.UUID) error {
 name := utils.RandomChoice(data.NigerianBusinessNames)
 
 business := businessSchema.Business{
  ID:          uuid.New(),
  Name:        name,
  Type:        businessSchema.TypePropertyManagement,
  Description: fmt.Sprintf("%s is a leading property management company in Nigeria", name),
  Email:       utils.RandomEmail(name),
  Phone:       utils.RandomPhoneNG(),
  IsVerified:  utils.RandomBool(),
  CreatedAt:   utils.RandomPastDate(365),
 }

 if err := ctx.DB.Create(&business).Error; err != nil {
  return err
 }

 // Add owner as member
 member := businessSchema.BusinessMember{
  ID:         uuid.New(),
  BusinessID: business.ID,
  UserID:     ownerID,
  Role:       businessSchema.RoleOwner,
  CreatedAt:  business.CreatedAt,
 }

 return ctx.DB.Create(&member).Error
}
```

These patterns can be adapted and extended for your specific needs!
