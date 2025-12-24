# Product Requirements Document (PRD): Hauslet Calendar & Pricing System

## Document Information
- **Product**: Hauslet Real Estate Platform
- **Feature**: Calendar Management & Dynamic Pricing System
- **Version**: 1.0
- **Date**: December 24, 2025
- **Author**: Product Team
- **Status**: Draft

---

## Executive Summary

Hauslet requires a comprehensive calendar and pricing system to manage different property types (short-term rentals, long-term rentals, and properties for sale) across West African markets. The system must handle booking availability, property showings, maintenance scheduling, and dynamic pricing while maintaining data consistency across multiple services.

---

## 1. Background & Context

### 1.1 Business Objectives
- Enable property owners to manage availability across multiple property types
- Provide dynamic pricing capabilities to maximize revenue for short-term rentals
- Streamline showing schedules for long-term rentals and properties for sale
- Reduce double-bookings and scheduling conflicts
- Support regional pricing strategies across West African markets

### 1.2 Target Users
- **Property Owners**: Manage availability, pricing, and maintenance
- **Property Managers**: Coordinate showings, bookings, and maintenance
- **Guests/Tenants**: Book properties and schedule viewings
- **Buyers**: Schedule property viewings
- **Vendors**: Access maintenance schedules

### 1.3 Success Metrics
- Zero double-bookings due to system conflicts
- 95% uptime for calendar and pricing services
- Property listing load time under 2 seconds
- Dynamic pricing adoption rate of 60% among shortlet properties
- 40% reduction in manual scheduling effort

---

## 2. Product Requirements

### 2.1 Calendar Management System

#### 2.1.1 Core Calendar Functionality

**Requirement CAL-001: Event Management**
- The system shall support creating, viewing, updating, and deleting calendar events
- Events must support multiple types: bookings, showings, maintenance, open houses
- Each event must track start time, end time, status, and participants
- Events must support timezone-aware scheduling
- System must prevent overlapping events for the same property

**Requirement CAL-002: Booking Management (Short-term Rentals)**
- System shall support instant booking and request-to-book workflows
- Bookings must include check-in time, check-out time, and guest count
- System must enforce minimum and maximum stay requirements
- Bookings must include buffer times for cleaning and preparation
- System shall support blocking dates for owner use

**Requirement CAL-003: Showing Management (Long-term Rentals & Sales)**
- System shall support scheduling private showings with specific time slots
- System must allow open house events with capacity limits
- Showings must support agent assignment
- System shall track showing history and attendance
- Multiple guests can schedule showings for the same time slot (open houses)

**Requirement CAL-004: Maintenance Scheduling**
- System shall support one-time and recurring maintenance events
- Maintenance events must categorize as routine, emergency, or repair
- System must support vendor assignment and coordination
- Maintenance events can overlap with availability (for vacant properties)
- System shall send notifications to relevant parties

**Requirement CAL-005: Recurring Events**
- System shall support weekly, monthly, and custom recurring patterns
- Recurring events must allow for exceptions (skip specific dates)
- Users can modify single instances or entire series
- System must handle recurring open houses and maintenance schedules

**Requirement CAL-006: Conflict Prevention**
- System must prevent double-bookings at the database level
- System shall validate conflicts before confirming reservations
- Buffer times must be respected when checking availability
- System must handle concurrent booking attempts gracefully

#### 2.1.2 Calendar Views and Search

**Requirement CAL-007: Availability Display**
- System shall display monthly calendar views with availability status
- Users can view availability for specific date ranges
- Calendar must show blocked dates and reason for blocking
- System shall indicate minimum stay requirements on calendar

**Requirement CAL-008: Search Functionality**
- Users can search for available properties by date range
- Search must filter by property type, location, and guest capacity
- System shall return properties with available slots for showings
- Search results must indicate next available dates

---

### 2.2 Pricing System

#### 2.2.1 Dynamic Pricing Rules

**Requirement PRC-001: Pricing Rule Types**
- System shall support base pricing (default nightly/monthly rates)
- System must support seasonal pricing variations
- System shall allow weekend and weekday pricing differences
- System must support holiday and special event pricing
- System shall enable length-of-stay discounts
- System must support last-minute and early-bird pricing
- System shall allow occupancy-based pricing variations

**Requirement PRC-002: Rule Configuration**
- Property owners can create multiple pricing rules per property
- Each rule must have a priority level for conflict resolution
- Rules can apply to specific date ranges
- Rules can target specific days of the week
- Rules must support minimum and maximum stay requirements
- System shall validate rules for conflicts before activation

**Requirement PRC-003: Pricing Components**
- System must support nightly/monthly base rates
- System shall calculate and include cleaning fees
- System must support service fees
- Pricing can include percentage-based modifiers
- System shall support absolute price overrides
- All pricing must specify currency (NGN, GHS, etc.)

#### 2.2.2 Price Calculation

**Requirement PRC-004: Real-time Price Calculation**
- System shall calculate accurate pricing based on check-in/out dates
- Calculation must consider all applicable pricing rules
- System must apply length-of-stay discounts automatically
- Price calculation must factor in number of guests
- System shall calculate lead-time based pricing (last-minute/early-bird)
- Calculation must provide detailed breakdown of price components

**Requirement PRC-005: Price Display**
- Property listings must show base starting price
- System shall indicate when dynamic pricing is enabled
- Detailed price breakdown must be available on request
- System must display nightly rates for each day in booking range
- Price quotes must show all fees separately (accommodation, cleaning, service)

**Requirement PRC-006: Price Caching and Performance**
- Base prices must be available for instant display in search results
- Calculated prices for specific dates can be cached temporarily
- Price changes must invalidate relevant caches
- System shall handle high-traffic pricing requests efficiently

---

### 2.3 Data Management and Consistency

#### 2.3.1 Service Integration

**Requirement INT-001: Property Data**
- Property listings must include base pricing information
- Property details must indicate if dynamic pricing is enabled
- Property service must track when pricing was last synchronized
- System shall maintain pricing information even when pricing service is unavailable

**Requirement INT-002: Cross-Service Communication**
- Calendar service must retrieve pricing from pricing service when creating bookings
- Property service must sync base prices from pricing service periodically
- Services must communicate availability and pricing for calendar displays
- All services must handle partial failures gracefully

**Requirement INT-003: Data Synchronization**
- Base prices must sync to property service every 6 hours
- Pricing rule changes must trigger immediate cache invalidation
- System shall maintain eventual consistency across services
- Manual sync operations must be available for administrators

#### 2.3.2 Data Integrity

**Requirement DAT-001: Version Control**
- Calendar events must use optimistic locking to prevent conflicts
- Concurrent updates must be detected and rejected
- System shall track version numbers for conflict resolution

**Requirement DAT-002: Audit Trail**
- All pricing calculations must be logged for debugging
- Calendar event changes must be tracked with timestamps
- System shall maintain history of pricing rule changes
- Revenue calculations must be auditable

---

### 2.4 User Experience Requirements

#### 2.4.1 Property Owner Experience

**Requirement UX-001: Calendar Management**
- Property owners can view all events for their properties in one calendar
- Owners can block dates manually with custom reasons
- System must show booking status (pending, confirmed, completed, cancelled)
- Owners receive notifications for new bookings and showing requests

**Requirement UX-002: Pricing Management**
- Owners can set base prices easily through intuitive interface
- System provides templates for common pricing strategies (weekend, seasonal)
- Owners can preview pricing calendar before applying rules
- System shows potential revenue impact of pricing changes

#### 2.4.2 Guest/Tenant Experience

**Requirement UX-003: Booking Flow**
- Guests see accurate pricing immediately when selecting dates
- System displays total cost breakdown before confirmation
- Guests receive clear confirmation with booking details
- System sends reminders before check-in

**Requirement UX-004: Showing Scheduling**
- Tenants/buyers can view available showing slots
- System allows selecting preferred time slots for viewings
- Users receive confirmation and reminders for scheduled showings
- System shows agent contact information for showings

---

### 2.5 Business Rules

#### 2.5.1 Calendar Business Rules

**Rule CAL-BR-001: Booking Minimums**
- Short-term rentals can enforce minimum stay requirements (e.g., 3 nights)
- Minimum stays can vary by season or day of week
- System must reject bookings that don't meet minimum requirements

**Rule CAL-BR-002: Buffer Times**
- Short-term rentals must include cleaning time between bookings
- Default buffer time is configurable per property
- Buffer times cannot be booked by guests

**Rule CAL-BR-003: Lead Time**
- Properties can require minimum advance booking time
- Last-minute bookings (< 24 hours) may require special approval
- Same-day bookings can be disabled per property

**Rule CAL-BR-004: Cancellation Windows**
- Bookings can be cancelled according to cancellation policy
- Cancellations within certain windows may incur fees
- Cancellations must free up calendar availability immediately

#### 2.5.2 Pricing Business Rules

**Rule PRC-BR-001: Rule Priority**
- When multiple rules apply, highest priority rule wins
- Base price rule has lowest priority
- Holiday pricing overrides weekend pricing
- Exact date rules override day-of-week rules

**Rule PRC-BR-002: Discount Stacking**
- Length-of-stay discounts apply after other pricing rules
- Weekly discount: 15% off for 7+ nights
- Monthly discount: 30% off for 28+ nights
- Discounts cannot reduce price below minimum threshold

**Rule PRC-BR-003: Price Validity**
- Quoted prices must be honored for 24 hours
- Prices can change for future dates without affecting existing quotes
- Confirmed bookings are protected from price changes

---

## 3. Technical Requirements

### 3.1 Performance Requirements

**Requirement PERF-001: Response Times**
- Property listing pages must load within 2 seconds
- Price calculations must complete within 500ms
- Calendar availability checks must complete within 300ms
- Search results must display within 3 seconds

**Requirement PERF-002: Scalability**
- System must support 10,000 concurrent users
- Must handle 100 bookings per minute during peak times
- Database must support 1 million properties
- System must scale across West African markets

### 3.2 Reliability Requirements

**Requirement REL-001: Availability**
- Calendar service must maintain 99.9% uptime
- Pricing service must maintain 99.5% uptime
- System must gracefully degrade when services are unavailable
- Base pricing must always be available from property listings

**Requirement REL-002: Data Consistency**
- No double-bookings are permitted
- Price calculations must be mathematically accurate
- Calendar conflicts must be detected 100% of the time
- Revenue calculations must match to the cent

### 3.3 Security Requirements

**Requirement SEC-001: Access Control**
- Only property owners can modify pricing for their properties
- Only authorized users can create/modify calendar events
- Pricing calculations must validate property ownership
- Personal information in bookings must be protected

**Requirement SEC-002: Data Privacy**
- Guest personal information must be encrypted at rest
- Payment information must not be stored in calendar/pricing services
- Audit logs must track all pricing and booking changes
- GDPR/data protection compliance for international guests

---

## 4. User Stories

### 4.1 Property Owner Stories

#### Story PO-001: Set Base Pricing
**As a** property owner  
**I want to** set a base nightly rate for my short-term rental  
**So that** guests can see the starting price when browsing properties

**Acceptance Criteria:**
- I can enter a base price per night in my local currency
- The price displays on my property listing immediately
- I can update the base price at any time
- The system validates that price is a positive number

---

#### Story PO-002: Create Seasonal Pricing
**As a** property owner  
**I want to** set higher prices during peak season  
**So that** I can maximize revenue during high-demand periods

**Acceptance Criteria:**
- I can define a date range for peak season
- I can set a specific nightly rate or percentage increase
- The system shows me a preview of how pricing will look
- Seasonal pricing overrides base pricing for those dates
- I can create multiple seasonal pricing rules

---

#### Story PO-003: Block Dates for Personal Use
**As a** property owner  
**I want to** block certain dates on my calendar  
**So that** I can use my property for personal purposes

**Acceptance Criteria:**
- I can select dates to block on a calendar interface
- I can add a note explaining why dates are blocked
- Blocked dates are not available for booking
- I can unblock dates if my plans change
- System prevents new bookings during blocked dates

---

#### Story PO-004: View All Bookings
**As a** property owner  
**I want to** see all bookings across my properties in one calendar  
**So that** I can manage my portfolio efficiently

**Acceptance Criteria:**
- I see a unified calendar showing all my properties
- I can filter by property, date range, or booking status
- Each booking shows guest name, dates, and total revenue
- I can click on a booking to see full details
- Calendar color-codes different event types

---

#### Story PO-005: Set Minimum Stay Requirements
**As a** property owner  
**I want to** require a minimum 3-night stay on weekends  
**So that** I reduce turnover costs and cleaning frequency

**Acceptance Criteria:**
- I can set minimum nights for specific days of the week
- I can set different minimums for different seasons
- System rejects bookings that don't meet minimum requirements
- Guests see the minimum stay requirement before booking
- I can change minimum stay requirements at any time

---

### 4.2 Guest/Tenant Stories

#### Story GT-001: Search Available Properties
**As a** guest  
**I want to** search for available properties for my travel dates  
**So that** I can find accommodation that fits my schedule

**Acceptance Criteria:**
- I can enter check-in and check-out dates
- Search shows only properties available for my dates
- I see the total price for my date range
- Results show nightly rate and total cost
- I can see if properties have minimum stay requirements

---

#### Story GT-002: See Price Breakdown
**As a** guest  
**I want to** see a detailed breakdown of my booking cost  
**So that** I understand what I'm paying for

**Acceptance Criteria:**
- I see nightly rate for each night of my stay
- Cleaning fee is shown separately
- Service fees are itemized
- Any discounts (weekly, monthly) are clearly shown
- Total price is prominently displayed
- Currency is clearly indicated

---

#### Story GT-003: Schedule Property Viewing
**As a** prospective tenant  
**I want to** schedule a viewing for a rental property  
**So that** I can see the property before applying

**Acceptance Criteria:**
- I can see available showing time slots
- I can select my preferred date and time
- I receive confirmation of my scheduled showing
- I get a reminder 24 hours before the showing
- I receive agent contact information
- I can reschedule or cancel if needed

---

#### Story GT-004: Book Last-Minute Property
**As a** guest  
**I want to** book a property for tonight  
**So that** I can find accommodation when traveling unexpectedly

**Acceptance Criteria:**
- I can search for properties available same-day
- I see if last-minute booking is allowed
- Last-minute pricing is clearly displayed
- I can complete booking instantly if property allows instant booking
- I receive immediate confirmation

---

#### Story GT-005: Receive Booking Confirmation
**As a** guest  
**I want to** receive detailed confirmation of my booking  
**So that** I have all necessary information for my stay

**Acceptance Criteria:**
- I receive email confirmation immediately after booking
- Confirmation includes property address and directions
- Check-in and check-out times are clearly stated
- Host contact information is provided
- Booking reference number is included
- Price breakdown is included in confirmation

---

### 4.3 Property Manager Stories

#### Story PM-001: Manage Multiple Property Showings
**As a** property manager  
**I want to** schedule showings across multiple properties  
**So that** I can efficiently coordinate my daily schedule

**Acceptance Criteria:**
- I see all showing requests in one dashboard
- I can approve or decline showing requests
- I can propose alternative times for showings
- System prevents scheduling conflicts
- I receive notifications of new showing requests
- I can view my daily schedule of all showings

---

#### Story PM-002: Schedule Recurring Maintenance
**As a** property manager  
**I want to** schedule monthly HVAC maintenance  
**So that** properties are maintained without impacting bookings

**Acceptance Criteria:**
- I can create recurring maintenance events
- I can specify frequency (weekly, monthly, etc.)
- I can assign vendors to maintenance tasks
- System warns me if maintenance conflicts with bookings
- I can track maintenance costs per property
- I can mark maintenance as complete

---

#### Story PM-003: Handle Open Houses
**As a** property manager  
**I want to** schedule open houses with capacity limits  
**So that** multiple prospects can view properties efficiently

**Acceptance Criteria:**
- I can create open house events with specific time windows
- I can set maximum attendee capacity
- System tracks RSVP count
- Prospects can register for open houses
- I receive list of registered attendees
- System prevents over-booking beyond capacity

---

### 4.4 System Administrator Stories

#### Story SA-001: Monitor System Performance
**As a** system administrator  
**I want to** monitor calendar and pricing service health  
**So that** I can ensure system reliability

**Acceptance Criteria:**
- I can view real-time metrics for response times
- I see error rates for pricing calculations
- I can track cache hit rates
- System alerts me when services are down
- I can view audit logs for debugging

---

#### Story SA-002: Sync Pricing Data
**As a** system administrator  
**I want to** manually trigger pricing synchronization  
**So that** I can resolve data inconsistencies

**Acceptance Criteria:**
- I can trigger sync for specific properties
- I can trigger bulk sync for all properties
- System shows sync status and last sync time
- I receive confirmation when sync completes
- System logs all sync operations

---

### 4.5 Revenue Optimization Stories

#### Story RO-001: Analyze Pricing Performance
**As a** property owner  
**I want to** see how my pricing strategy affects bookings  
**So that** I can optimize for maximum revenue

**Acceptance Criteria:**
- I can view revenue by month
- I see occupancy rates over time
- System shows which pricing rules generated most revenue
- I can compare different time periods
- I see average nightly rate and booking value

---

#### Story RO-002: Implement Gap-Filler Pricing
**As a** property owner  
**I want to** offer discounts for single-night gaps between bookings  
**So that** I maximize occupancy and reduce lost revenue

**Acceptance Criteria:**
- System identifies 1-2 night gaps in calendar
- I can set discount percentage for gap dates
- Discount applies automatically to applicable dates
- Guests see discounted price when booking gap dates
- I can disable gap-filler pricing if desired

---

## 5. Non-Functional Requirements

### 5.1 Usability
- System must be accessible on mobile devices
- Calendar interface must be intuitive for non-technical users
- Pricing setup must be completable in under 5 minutes
- Error messages must be clear and actionable

### 5.2 Localization
- System must support multiple currencies (NGN, GHS, XOF, etc.)
- Date formats must adapt to regional preferences
- Timezone handling must be accurate across West Africa
- Multi-language support for future expansion

### 5.3 Compliance
- Must comply with local short-term rental regulations
- Must support tax calculation for different jurisdictions
- Must maintain audit trails for financial reporting
- Must support data export for accounting purposes

---

## 6. Assumptions and Constraints

### 6.1 Assumptions
- Property owners have reliable internet access to manage listings
- Users understand basic calendar concepts
- Most short-term rentals will use dynamic pricing
- Payment processing is handled by separate service

### 6.2 Constraints
- Must integrate with existing property service
- Must work within current infrastructure budget
- Cannot require complete system rewrite
- Must maintain backward compatibility with existing bookings

### 6.3 Dependencies
- Requires functional property management service
- Requires user authentication service
- Requires notification service for emails/SMS
- May require payment service integration for deposits

---

## 7. Success Criteria

### 7.1 Launch Criteria
- Zero critical bugs in production
- All core user stories implemented
- Performance requirements met
- Security audit passed
- User acceptance testing completed

### 7.2 Post-Launch Metrics (3 months)
- 80% of shortlet properties using dynamic pricing
- Average revenue increase of 20% for properties using dynamic pricing
- Double-booking rate below 0.1%
- User satisfaction score above 4.0/5.0
- Property owner adoption rate above 70%

---

## 8. Future Enhancements (Out of Scope for v1.0)

- Integration with external calendar services (Google Calendar, iCal)
- AI-powered pricing recommendations
- Multi-currency automatic conversion
- Group booking management
- Loyalty programs and repeat guest discounts
- Integration with property management systems (PMS)
- Mobile applications for property managers
- Advanced analytics and forecasting
- Integration with smart locks for check-in
- Revenue management dashboard with recommendations

---

## 9. Glossary

**Base Price**: Default nightly or monthly rate before any pricing rules applied

**Buffer Time**: Mandatory gap between bookings for cleaning and preparation

**Dynamic Pricing**: Pricing that varies based on demand, season, or other factors

**Gap-Filler**: Discount offered for short gaps between existing bookings

**Instant Book**: Booking confirmed immediately without owner approval

**Length-of-Stay Discount**: Reduced rate for bookings exceeding certain duration

**Minimum Stay**: Shortest duration allowed for a booking

**Open House**: Showing event allowing multiple prospects simultaneously

**Pricing Rule**: Configured condition that modifies property pricing

**Short-term Rental/Shortlet**: Property rented for less than 30 days

---

## 10. Appendices

### Appendix A: Service Architecture Overview
- Calendar Service: Manages events, availability, and scheduling
- Pricing Service: Handles pricing rules and calculations
- Property Service: Stores property data and base pricing
- Services communicate via REST APIs
- Data synchronization occurs asynchronously

### Appendix B: Data Flow Examples
- **Booking Creation**: Guest searches → Property service returns listings → Guest selects dates → Pricing service calculates cost → Calendar service validates availability → Booking created
- **Price Display**: User views listing → Property service shows base price → User enters dates → Pricing service calculates exact price → User sees breakdown
- **Pricing Update**: Owner modifies rule → Pricing service updates → Cache invalidated → Property service synced → Listings updated
