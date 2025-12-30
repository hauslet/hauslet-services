# Listings Growth Features Implementation Plan

## 📊 Overview
This plan covers non-transactional growth features for sale/rental listings: lead management, messaging infrastructure, analytics tracking, and premium visibility. Combined with the [Listing Promotion System](LISTING_PROMOTION_SYSTEM_IMPLEMENTATION_PLAN.md), these features form the core monetization and engagement strategy for Sales/Rentals properties.

**Platform Position:** Listing host (not a broker) - We facilitate connections but don't handle transactions.

## 🎯 Current Status
**Phase:** Phase 0 - Planning
**Last Updated:** 2025-12-30
**Business Model Integration:** Works alongside Promotion System for complete monetization

---

## Goals
- **Increase Lead Quality**: Capture qualified leads with spam prevention and validation
- **Boost Engagement**: Deliver actionable analytics to owners/agents for better listing optimization
- **Monetize Visibility**: Enable premium placement through subscription-included promotions (see [Promotion System](LISTING_PROMOTION_SYSTEM_IMPLEMENTATION_PLAN.md))
- **Facilitate Communication**: Provide robust, reliable messaging infrastructure
- **Track Performance**: Comprehensive analytics for conversion funnel optimization

## Non-Goals
- ❌ No escrow, rent collection, contract signing, or closing workflows
- ❌ No public reviews for non-transactional listings (sales/rentals)
- ❌ No platform liability for transaction outcomes
- ❌ No direct transaction intermediation

## High-Level Scope
1. **Leads Module**: Inquiry capture, assignment, lifecycle management, spam controls
2. **Messaging Module**: Multi-party conversations, delivery state, moderation, attachments
3. **Analytics Module**: Event ingestion, aggregation, reporting
4. **Premium Visibility**: Integrated with Promotion System for boost tracking and impression analytics

---

## Module Boundaries (DDD Architecture)

### `internal/modules/leads`
**Purpose:** Lead capture, assignment, lifecycle management
**Domain Models:** Lead, LeadEvent, LeadAssignment
**Key Services:** LeadService, LeadAssignmentService, LeadValidationService
**External Dependencies:** property (listing metadata), business (routing), profile (contact info)

### `internal/modules/messaging`
**Purpose:** Airbnb-grade messaging infrastructure
**Domain Models:** Conversation, Message, MessageReceipt, Attachment
**Key Services:** ConversationService, MessageService, AttachmentService
**External Dependencies:** storage (attachments), queue (notifications), moderation (safety)

### `internal/modules/analytics`
**Purpose:** Event tracking and aggregation
**Domain Models:** ListingEvent, ListingDailyStats, BusinessDailyStats
**Key Services:** EventIngestionService, AggregationService, ReportingService
**External Dependencies:** queue (async aggregation), listing-promotion (boost performance)

### `internal/modules/scheduling` (Optional - Future)
**Purpose:** Viewing appointment management
**Domain Models:** ViewingRequest, Viewing
**Key Services:** SchedulingService, CalendarService
**External Dependencies:** messaging (confirmations), profile (availability)

---

## Dependencies Map

### Core Module Dependencies
```
leads ─────────┬─> property (listing metadata, ownership)
               ├─> business (routing, permissions)
               ├─> profile (contact info)
               └─> messaging (auto-create conversation)

messaging ─────┬─> storage (attachments)
               ├─> queue (notifications)
               └─> moderation (content safety)

analytics ─────┬─> property (listing data)
               ├─> listing-promotion (boost performance)
               └─> queue (async aggregation)

listing-promotion ─> analytics (impression/click tracking)
```

### Payment Integration
- **Promotion System**: Handles paid boosts (Featured, Premium) - see [Promotion Plan](LISTING_PROMOTION_SYSTEM_IMPLEMENTATION_PLAN.md)
- **Analytics**: Tracks boost performance (impressions, clicks, ROI)
- **Leads**: No payment integration (free for all tiers)
- **Messaging**: No payment integration (free for all tiers)

---

## Phase 1: Lead Capture & Management (Foundation)

**Goal:** Build core lead capture, validation, and assignment system
**Estimated Files:** ~10 files, ~800-1000 LOC
**Dependencies:** None (first module)

### 1.1 Domain Models

- [ ] Create `internal/modules/leads/domain/enums.go`
  ```go
  // Lead Status
  type LeadStatus string
  const (
      LeadStatusNew       LeadStatus = "new"        // Just submitted
      LeadStatusContacted LeadStatus = "contacted"  // Agent reached out
      LeadStatusScheduled LeadStatus = "scheduled"  // Viewing scheduled
      LeadStatusConverted LeadStatus = "converted"  // Deal closed (self-reported)
      LeadStatusClosed    LeadStatus = "closed"     // No longer pursuing
      LeadStatusSpam      LeadStatus = "spam"       // Marked as spam
  )

  // Lead Source
  type LeadSource string
  const (
      LeadSourceWebForm    LeadSource = "web_form"
      LeadSourceMobile     LeadSource = "mobile_app"
      LeadSourceAPI        LeadSource = "api"
      LeadSourceImported   LeadSource = "imported"
  )

  // Lead Event Type
  type LeadEventType string
  const (
      EventLeadCreated    LeadEventType = "created"
      EventLeadAssigned   LeadEventType = "assigned"
      EventLeadContacted  LeadEventType = "contacted"
      EventStatusChanged  LeadEventType = "status_changed"
      EventNoteAdded      LeadEventType = "note_added"
  )
  ```

- [ ] Create `internal/modules/leads/domain/lead.go`
  ```go
  type Lead struct {
      ID          uuid.UUID   `json:"id"`
      ListingID   uuid.UUID   `json:"listing_id"`
      BusinessID  *uuid.UUID  `json:"business_id,omitempty"` // If listing owned by business
      UserID      *uuid.UUID  `json:"user_id,omitempty"`     // If lead is registered user

      // Contact Info
      Name        string      `json:"name"`
      Email       string      `json:"email"`
      Phone       *string     `json:"phone,omitempty"`
      Message     string      `json:"message"`

      // Classification
      Source      LeadSource  `json:"source"`
      Status      LeadStatus  `json:"status"`

      // Assignment
      AssignedTo  *uuid.UUID  `json:"assigned_to,omitempty"`
      AssignedAt  *time.Time  `json:"assigned_at,omitempty"`

      // Metadata
      IPAddress   *string     `json:"ip_address,omitempty"`
      UserAgent   *string     `json:"user_agent,omitempty"`
      ReferralURL *string     `json:"referral_url,omitempty"`

      // Spam Detection
      SpamScore   float64     `json:"spam_score"`
      IsVerified  bool        `json:"is_verified"`

      CreatedAt   time.Time   `json:"created_at"`
      UpdatedAt   time.Time   `json:"updated_at"`
      DeletedAt   *time.Time  `json:"deleted_at,omitempty"`
  }

  // Business logic methods
  func (l *Lead) CanBeContacted() bool
  func (l *Lead) Assign(agentID uuid.UUID) error
  func (l *Lead) UpdateStatus(newStatus LeadStatus) error
  func (l *Lead) MarkAsSpam() error
  func (l *Lead) Verify() error
  ```

- [ ] Create `internal/modules/leads/domain/lead_event.go`
  ```go
  type LeadEvent struct {
      ID        uuid.UUID     `json:"id"`
      LeadID    uuid.UUID     `json:"lead_id"`
      Type      LeadEventType `json:"type"`
      ActorID   *uuid.UUID    `json:"actor_id,omitempty"` // Who performed action
      Metadata  map[string]interface{} `json:"metadata,omitempty"`
      CreatedAt time.Time     `json:"created_at"`
  }
  ```

- [ ] Create `internal/modules/leads/domain/lead_assignment.go`
  ```go
  type LeadAssignment struct {
      ID         uuid.UUID  `json:"id"`
      LeadID     uuid.UUID  `json:"lead_id"`
      AssigneeID uuid.UUID  `json:"assignee_id"`
      AssignedBy uuid.UUID  `json:"assigned_by"`
      Reason     *string    `json:"reason,omitempty"`
      AssignedAt time.Time  `json:"assigned_at"`
  }
  ```

- [ ] Create `internal/modules/leads/domain/errors.go`
  ```go
  var (
      ErrLeadNotFound        = errors.New("lead not found")
      ErrDuplicateLead       = errors.New("duplicate lead detected")
      ErrInvalidEmail        = errors.New("invalid email address")
      ErrInvalidPhone        = errors.New("invalid phone number")
      ErrSpamDetected        = errors.New("spam detected")
      ErrRateLimitExceeded   = errors.New("rate limit exceeded")
      ErrCannotAssign        = errors.New("cannot assign lead")
      ErrAlreadyAssigned     = errors.New("lead already assigned")
  )
  ```

### 1.2 GORM Schemas

- [ ] Create `internal/modules/leads/repository/schema/gorm.go`
  ```go
  type Lead struct {
      ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
      ListingID   uuid.UUID  `gorm:"type:uuid;not null;index:idx_lead_listing"`
      BusinessID  *uuid.UUID `gorm:"type:uuid;index"`
      UserID      *uuid.UUID `gorm:"type:uuid;index"`

      Name        string     `gorm:"type:varchar(255);not null"`
      Email       string     `gorm:"type:varchar(255);not null;index:idx_lead_email"`
      Phone       *string    `gorm:"type:varchar(50)"`
      Message     string     `gorm:"type:text;not null"`

      Source      string     `gorm:"type:varchar(50);not null"`
      Status      string     `gorm:"type:varchar(50);not null;index:idx_lead_status"`

      AssignedTo  *uuid.UUID `gorm:"type:uuid;index"`
      AssignedAt  *time.Time

      IPAddress   *string    `gorm:"type:varchar(100)"`
      UserAgent   *string    `gorm:"type:text"`
      ReferralURL *string    `gorm:"type:text"`

      SpamScore   float64    `gorm:"not null;default:0.0"`
      IsVerified  bool       `gorm:"not null;default:false"`

      CreatedAt   time.Time
      UpdatedAt   time.Time
      DeletedAt   *time.Time `gorm:"index"`
  }

  type LeadEvent struct {
      ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
      LeadID    uuid.UUID  `gorm:"type:uuid;not null;index:idx_event_lead"`
      Type      string     `gorm:"type:varchar(50);not null"`
      ActorID   *uuid.UUID `gorm:"type:uuid"`
      Metadata  string     `gorm:"type:jsonb"`
      CreatedAt time.Time  `gorm:"index"`
  }

  type LeadAssignment struct {
      ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
      LeadID     uuid.UUID  `gorm:"type:uuid;not null;index"`
      AssigneeID uuid.UUID  `gorm:"type:uuid;not null;index"`
      AssignedBy uuid.UUID  `gorm:"type:uuid;not null"`
      Reason     *string    `gorm:"type:text"`
      AssignedAt time.Time  `gorm:"not null"`
  }
  ```

### 1.3 Repositories

- [ ] Create `internal/modules/leads/repository/interface.go`
  ```go
  type LeadRepository interface {
      Create(ctx context.Context, lead *schema.Lead) error
      GetByID(ctx context.Context, id uuid.UUID) (*schema.Lead, error)
      GetByEmail(ctx context.Context, email string, listingID uuid.UUID) (*schema.Lead, error)
      Update(ctx context.Context, lead *schema.Lead) error
      UpdateStatus(ctx context.Context, id uuid.UUID, status string) error

      ListByListing(ctx context.Context, listingID uuid.UUID, limit, offset int) ([]*schema.Lead, error)
      ListByBusiness(ctx context.Context, businessID uuid.UUID, status *string, limit, offset int) ([]*schema.Lead, error)
      ListByAssignee(ctx context.Context, assigneeID uuid.UUID, status *string, limit, offset int) ([]*schema.Lead, error)

      CountByListing(ctx context.Context, listingID uuid.UUID) (int, error)
      CountByIPToday(ctx context.Context, ipAddress string) (int, error)
      CountByEmailToday(ctx context.Context, email string) (int, error)
  }

  type LeadEventRepository interface {
      Create(ctx context.Context, event *schema.LeadEvent) error
      ListByLead(ctx context.Context, leadID uuid.UUID, limit, offset int) ([]*schema.LeadEvent, error)
  }

  type LeadAssignmentRepository interface {
      Create(ctx context.Context, assignment *schema.LeadAssignment) error
      ListByLead(ctx context.Context, leadID uuid.UUID) ([]*schema.LeadAssignment, error)
  }
  ```

- [ ] Implement repositories with GORM
- [ ] Create mappers in `domain/mapper.go`

### 1.4 Service Layer

- [ ] Create `internal/modules/leads/service/interface.go`
  ```go
  type LeadService interface {
      // Create lead with validation and spam checks
      CreateLead(ctx context.Context, input CreateLeadInput) (*domain.Lead, error)

      // Get lead details
      GetLead(ctx context.Context, leadID uuid.UUID) (*domain.Lead, error)

      // Update lead status
      UpdateLeadStatus(ctx context.Context, leadID uuid.UUID, status domain.LeadStatus) error

      // Assign lead to agent
      AssignLead(ctx context.Context, leadID, assigneeID, assignedBy uuid.UUID, reason *string) error

      // List leads (with filters)
      ListLeads(ctx context.Context, filter LeadFilter) ([]*domain.Lead, error)

      // Mark as spam
      MarkAsSpam(ctx context.Context, leadID uuid.UUID) error

      // Verify lead (e.g., email verification)
      VerifyLead(ctx context.Context, leadID uuid.UUID) error
  }

  type CreateLeadInput struct {
      ListingID   uuid.UUID
      Name        string
      Email       string
      Phone       *string
      Message     string
      Source      domain.LeadSource
      IPAddress   *string
      UserAgent   *string
      ReferralURL *string
  }

  type LeadFilter struct {
      ListingID  *uuid.UUID
      BusinessID *uuid.UUID
      AssignedTo *uuid.UUID
      Status     *domain.LeadStatus
      Limit      int
      Offset     int
  }
  ```

- [ ] Create `internal/modules/leads/service/lead_service.go`
  ```go
  type LeadServiceImpl struct {
      leadRepo       repository.LeadRepository
      eventRepo      repository.LeadEventRepository
      assignmentRepo repository.LeadAssignmentRepository
      validator      *LeadValidator
      spamDetector   *SpamDetector
      db             *gorm.DB
      log            *lgr.Logger
  }

  // CreateLead implementation:
  // 1. Validate email, phone, message
  // 2. Check rate limits (IP-based, email-based)
  // 3. Run spam detection (SpamDetector)
  // 4. Check for duplicates (same email + listing in last 24h)
  // 5. Create lead + lead event
  // 6. Auto-assign if business has routing rules
  // 7. Trigger notification (via messaging hook or queue)
  ```

- [ ] Create `internal/modules/leads/service/validation.go`
  ```go
  type LeadValidator struct {
      emailRegex *regexp.Regexp
      phoneRegex *regexp.Regexp
  }

  func (v *LeadValidator) ValidateEmail(email string) error
  func (v *LeadValidator) ValidatePhone(phone string) error
  func (v *LeadValidator) ValidateMessage(message string) error
  func (v *LeadValidator) CheckRateLimit(ctx context.Context, ipAddress, email string) error
  ```

- [ ] Create `internal/modules/leads/service/spam_detector.go`
  ```go
  type SpamDetector struct {
      // Pattern-based spam detection
      suspiciousPatterns []*regexp.Regexp
  }

  func (s *SpamDetector) CalculateSpamScore(input CreateLeadInput) float64
  // Factors:
  // - Message length and quality
  // - Email domain reputation
  // - IP address history
  // - Suspicious patterns (URLs, repeated chars)
  // Returns score 0.0-1.0 (>0.8 = likely spam)
  ```

### 1.5 GraphQL Schema

- [ ] Create `internal/modules/leads/port/graphql/schema.graphqls`
  ```graphql
  enum LeadStatus {
      new
      contacted
      scheduled
      converted
      closed
      spam
  }

  enum LeadSource {
      web_form
      mobile_app
      api
      imported
  }

  type Lead {
      id: UUID!
      listingId: UUID!
      businessId: UUID
      userId: UUID
      name: String!
      email: String!
      phone: String
      message: String!
      source: LeadSource!
      status: LeadStatus!
      assignedTo: UUID
      assignedAt: Time
      spamScore: Float!
      isVerified: Boolean!
      events: [LeadEvent!]!
      createdAt: Time!
      updatedAt: Time!
  }

  type LeadEvent {
      id: UUID!
      type: String!
      actorId: UUID
      metadata: JSON
      createdAt: Time!
  }

  input CreateLeadInput {
      listingId: UUID!
      name: String!
      email: String!
      phone: String
      message: String!
      source: LeadSource!
  }

  input UpdateLeadStatusInput {
      leadId: UUID!
      status: LeadStatus!
  }

  input AssignLeadInput {
      leadId: UUID!
      assigneeId: UUID!
      reason: String
  }

  extend type Query {
      lead(id: UUID!): Lead
      myLeads(status: LeadStatus, limit: Int, offset: Int): [Lead!]!
      listingLeads(listingId: UUID!, status: LeadStatus, limit: Int, offset: Int): [Lead!]!
      businessLeads(businessId: UUID!, status: LeadStatus, limit: Int, offset: Int): [Lead!]! @requireBusinessMember
  }

  extend type Mutation {
      createLead(input: CreateLeadInput!): Lead!
      updateLeadStatus(input: UpdateLeadStatusInput!): Lead!
      assignLead(input: AssignLeadInput!): Lead!
      markLeadAsSpam(leadId: UUID!): Lead!
      verifyLead(leadId: UUID!): Lead!
  }
  ```

### Phase 1 Verification
- [ ] Create lead via GraphQL
- [ ] Verify spam detection works
- [ ] Test rate limiting (IP + email)
- [ ] Test duplicate prevention
- [ ] Assign lead to agent
- [ ] Verify events recorded

**Completion Criteria:** Lead capture system functional with spam protection

---

## Phase 2: Messaging Infrastructure (Airbnb-Grade)

**Goal:** Build robust messaging system with delivery tracking and safety features
**Estimated Files:** ~12 files, ~1200-1500 LOC
**Dependencies:** Phase 1 (leads integration), storage module (attachments)

### 2.1 Domain Models

- [ ] Create `internal/modules/messaging/domain/enums.go`
  ```go
  // Conversation State
  type ConversationState string
  const (
      ConversationStateActive   ConversationState = "active"
      ConversationStateArchived ConversationState = "archived"
      ConversationStateBlocked  ConversationState = "blocked"
  )

  // Message Type
  type MessageType string
  const (
      MessageTypeText              MessageType = "text"
      MessageTypeSystem            MessageType = "system"
      MessageTypeAttachment        MessageType = "attachment"
      MessageTypeViewingRequest    MessageType = "viewing_request"
      MessageTypeViewingConfirmed  MessageType = "viewing_confirmed"
  )

  // Delivery State
  type DeliveryState string
  const (
      DeliveryStateSent      DeliveryState = "sent"
      DeliveryStateDelivered DeliveryState = "delivered"
      DeliveryStateRead      DeliveryState = "read"
  )

  // Participant Role
  type ParticipantRole string
  const (
      ParticipantRoleOwner ParticipantRole = "owner"
      ParticipantRoleAgent ParticipantRole = "agent"
      ParticipantRoleLead  ParticipantRole = "lead"
  )
  ```

- [ ] Create `internal/modules/messaging/domain/conversation.go`
  ```go
  type Conversation struct {
      ID          uuid.UUID         `json:"id"`
      LeadID      *uuid.UUID        `json:"lead_id,omitempty"`
      ListingID   uuid.UUID         `json:"listing_id"`
      BusinessID  *uuid.UUID        `json:"business_id,omitempty"`
      Subject     *string           `json:"subject,omitempty"`
      State       ConversationState `json:"state"`
      CreatedBy   uuid.UUID         `json:"created_by"`
      CreatedAt   time.Time         `json:"created_at"`
      UpdatedAt   time.Time         `json:"updated_at"`
  }

  // Business logic
  func (c *Conversation) CanParticipantSend(participantID uuid.UUID) bool
  func (c *Conversation) Archive() error
  func (c *Conversation) Block() error
  func (c *Conversation) Unblock() error
  ```

- [ ] Create `internal/modules/messaging/domain/participant.go`
  ```go
  type ConversationParticipant struct {
      ID             uuid.UUID       `json:"id"`
      ConversationID uuid.UUID       `json:"conversation_id"`
      UserID         uuid.UUID       `json:"user_id"`
      Role           ParticipantRole `json:"role"`
      JoinedAt       time.Time       `json:"joined_at"`
      LastReadAt     *time.Time      `json:"last_read_at,omitempty"`
      Muted          bool            `json:"muted"`
      Archived       bool            `json:"archived"`
  }

  func (p *ConversationParticipant) MarkRead(at time.Time)
  func (p *ConversationParticipant) Mute()
  func (p *ConversationParticipant) Unmute()
  ```

- [ ] Create `internal/modules/messaging/domain/message.go`
  ```go
  type Message struct {
      ID             uuid.UUID      `json:"id"`
      ConversationID uuid.UUID      `json:"conversation_id"`
      SenderID       uuid.UUID      `json:"sender_id"`
      SenderRole     ParticipantRole `json:"sender_role"`
      Type           MessageType    `json:"type"`
      Body           string         `json:"body"`
      Metadata       map[string]interface{} `json:"metadata,omitempty"`
      DeliveryState  DeliveryState  `json:"delivery_state"`

      // Idempotency
      ClientID       *string        `json:"client_id,omitempty"` // Client-generated for dedup

      CreatedAt      time.Time      `json:"created_at"`
  }

  func (m *Message) MarkDelivered()
  func (m *Message) MarkRead()
  ```

- [ ] Create `internal/modules/messaging/domain/message_receipt.go`
  ```go
  type MessageReceipt struct {
      ID            uuid.UUID  `json:"id"`
      MessageID     uuid.UUID  `json:"message_id"`
      ParticipantID uuid.UUID  `json:"participant_id"`
      DeliveredAt   *time.Time `json:"delivered_at,omitempty"`
      ReadAt        *time.Time `json:"read_at,omitempty"`
  }
  ```

- [ ] Create `internal/modules/messaging/domain/attachment.go`
  ```go
  type Attachment struct {
      ID         uuid.UUID `json:"id"`
      MessageID  uuid.UUID `json:"message_id"`
      StorageKey string    `json:"storage_key"`
      FileName   string    `json:"file_name"`
      MimeType   string    `json:"mime_type"`
      Size       int64     `json:"size"`
      Checksum   string    `json:"checksum"`
      ScanStatus string    `json:"scan_status"` // pending, clean, rejected
      CreatedAt  time.Time `json:"created_at"`
  }
  ```

### 2.2 Service Layer

- [ ] Create `internal/modules/messaging/service/interface.go`
  ```go
  type ConversationService interface {
      CreateConversation(ctx context.Context, input CreateConversationInput) (*domain.Conversation, error)
      GetConversation(ctx context.Context, conversationID uuid.UUID) (*domain.Conversation, error)
      ListUserConversations(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Conversation, error)
      ArchiveConversation(ctx context.Context, conversationID, userID uuid.UUID) error
      BlockConversation(ctx context.Context, conversationID, userID uuid.UUID) error
  }

  type MessageService interface {
      SendMessage(ctx context.Context, input SendMessageInput) (*domain.Message, error)
      ListMessages(ctx context.Context, conversationID uuid.UUID, limit, offset int) ([]*domain.Message, error)
      MarkAsRead(ctx context.Context, messageID, participantID uuid.UUID) error
      MarkConversationAsRead(ctx context.Context, conversationID, participantID uuid.UUID) error
  }

  type SendMessageInput struct {
      ConversationID uuid.UUID
      SenderID       uuid.UUID
      Type           domain.MessageType
      Body           string
      ClientID       *string // For idempotency
      Metadata       map[string]interface{}
  }
  ```

- [ ] Implement services with:
  - Idempotency check via ClientID
  - Delivery receipt creation for all participants
  - Push notification triggering
  - Content moderation hooks

### 2.3 GraphQL Schema

- [ ] Create comprehensive messaging schema with mutations for send, read, archive

### Phase 2 Verification
- [ ] Create conversation
- [ ] Send message with idempotency key
- [ ] Verify duplicate send rejected
- [ ] Mark as read, verify receipt updated
- [ ] Test attachment upload
- [ ] Verify notifications sent

**Completion Criteria:** Reliable messaging with delivery tracking

---

## Phase 3: Analytics & Event Tracking

**Goal:** Track listing performance and aggregate metrics
**Estimated Files:** ~8 files, ~600-800 LOC
**Dependencies:** listing-promotion (boost tracking)

### 3.1 Domain Models

- [ ] Create `internal/modules/analytics/domain/enums.go`
  ```go
  type EventType string
  const (
      EventTypeView              EventType = "view"
      EventTypeSave              EventType = "save"
      EventTypeInquiry           EventType = "inquiry"
      EventTypeMessage           EventType = "message"
      EventTypeViewing           EventType = "viewing"
      EventTypePromotionImpression EventType = "promotion_impression"
      EventTypePromotionClick      EventType = "promotion_click"
  )
  ```

- [ ] Create `internal/modules/analytics/domain/listing_event.go`
  ```go
  type ListingEvent struct {
      ID        uuid.UUID  `json:"id"`
      ListingID uuid.UUID  `json:"listing_id"`
      UserID    *uuid.UUID `json:"user_id,omitempty"` // Anonymous if nil
      Type      EventType  `json:"type"`
      Source    string     `json:"source"` // web, mobile, api

      // Promotion tracking
      PromotionID *uuid.UUID `json:"promotion_id,omitempty"`

      // Session tracking
      SessionID   *string    `json:"session_id,omitempty"`

      Metadata    map[string]interface{} `json:"metadata,omitempty"`
      CreatedAt   time.Time  `json:"created_at"`
  }
  ```

- [ ] Create `internal/modules/analytics/domain/daily_stats.go`
  ```go
  type ListingDailyStats struct {
      ListingID   uuid.UUID `json:"listing_id"`
      Date        time.Time `json:"date"`

      Views       int64     `json:"views"`
      Saves       int64     `json:"saves"`
      Inquiries   int64     `json:"inquiries"`
      Messages    int64     `json:"messages"`
      Viewings    int64     `json:"viewings"`

      // Promotion metrics (if promoted)
      PromotionImpressions int64 `json:"promotion_impressions"`
      PromotionClicks      int64 `json:"promotion_clicks"`

      // Conversion metrics
      InquiryToViewingRate float64 `json:"inquiry_to_viewing_rate"`

      UpdatedAt   time.Time `json:"updated_at"`
  }

  type BusinessDailyStats struct {
      BusinessID  uuid.UUID `json:"business_id"`
      Date        time.Time `json:"date"`

      TotalViews      int64 `json:"total_views"`
      TotalInquiries  int64 `json:"total_inquiries"`
      TotalViewings   int64 `json:"total_viewings"`
      ActiveListings  int   `json:"active_listings"`

      UpdatedAt   time.Time `json:"updated_at"`
  }
  ```

### 3.2 Service Layer

- [ ] Create `internal/modules/analytics/service/interface.go`
  ```go
  type EventIngestionService interface {
      TrackEvent(ctx context.Context, event *domain.ListingEvent) error
      TrackBatch(ctx context.Context, events []*domain.ListingEvent) error
  }

  type AggregationService interface {
      AggregateDaily(ctx context.Context, date time.Time) error
      AggregateListingStats(ctx context.Context, listingID uuid.UUID, date time.Time) error
      AggregateBusinessStats(ctx context.Context, businessID uuid.UUID, date time.Time) error
  }

  type ReportingService interface {
      GetListingStats(ctx context.Context, listingID uuid.UUID, startDate, endDate time.Time) ([]*domain.ListingDailyStats, error)
      GetBusinessStats(ctx context.Context, businessID uuid.UUID, startDate, endDate time.Time) ([]*domain.BusinessDailyStats, error)
      GetTopPerformingListings(ctx context.Context, limit int) ([]*domain.ListingDailyStats, error)
  }
  ```

- [ ] Implement async aggregation via worker jobs

### 3.3 Worker Jobs

- [ ] Create `internal/queue/jobs/analytics/daily_aggregation.go`
  ```go
  type DailyAggregationJob struct {
      aggregationService analytics.AggregationService
      log                *lgr.Logger
  }

  func (j *DailyAggregationJob) Run(ctx context.Context) error {
      yesterday := time.Now().AddDate(0, 0, -1)
      return j.aggregationService.AggregateDaily(ctx, yesterday)
  }
  ```

- [ ] Register to run daily at 2 AM

### 3.4 GraphQL Schema

- [ ] Create analytics queries:
  ```graphql
  extend type Query {
      listingStats(listingId: UUID!, startDate: Time!, endDate: Time!): [ListingDailyStats!]!
      myListingsPerformance(startDate: Time!, endDate: Time!): [ListingDailyStats!]!
      businessStats(businessId: UUID!, startDate: Time!, endDate: Time!): [BusinessDailyStats!]! @requireBusinessMember
  }
  ```

### Phase 3 Verification
- [ ] Track listing view events
- [ ] Run aggregation job
- [ ] Verify daily stats computed
- [ ] Query stats via GraphQL
- [ ] Verify promotion impression tracking

**Completion Criteria:** Analytics pipeline functional with daily aggregation

---

## Integration with Listing Promotion System

### Cross-Module Integration Points

1. **Analytics → Promotion System**
   - Track promotion impressions and clicks
   - Calculate promotion ROI
   - Report performance in promotion dashboard

2. **Leads → Promotion System**
   - Included in conversion funnel metrics
   - Premium/Featured listings may get priority in lead display

3. **Property → Promotion System**
   - Limit enforcement via PromotionChecker interface
   - See [Promotion Plan Phase 3](LISTING_PROMOTION_SYSTEM_IMPLEMENTATION_PLAN.md#phase-3-integration-patterns---property-module-interface)

### Implementation Example

```go
// In analytics service
func (s *EventIngestionService) TrackEvent(ctx context.Context, event *domain.ListingEvent) error {
    // If event has promotion_id, also track in promotion analytics
    if event.PromotionID != nil {
        if err := s.promotionAnalytics.TrackImpression(ctx, *event.PromotionID); err != nil {
            s.log.Logf("WARN failed to track promotion impression: %v", err)
            // Don't fail the event ingestion
        }
    }

    return s.eventRepo.Create(ctx, mappers.MapEventToSchema(event))
}
```

---

## Permissions & Security

### Business Role-Based Access Control (RBAC)

```go
// Lead permissions
- CreateLead: Public (with rate limits)
- ViewLead: Lead creator OR listing owner OR business admin
- AssignLead: Business admin OR listing owner
- UpdateStatus: Assigned agent OR business admin

// Messaging permissions
- CreateConversation: Lead creator OR listing owner
- SendMessage: Conversation participant
- ViewMessages: Conversation participant
- ArchiveConversation: Conversation participant

// Analytics permissions
- ViewListingStats: Listing owner OR business member
- ViewBusinessStats: Business admin only
```

### Rate Limiting Strategy

```yaml
# config/defaults/features.yaml
rate_limits:
  leads:
    per_ip_per_day: 5
    per_email_per_day: 3
  messaging:
    per_user_per_minute: 10
  analytics:
    track_events_per_second: 100
```

### PII Handling

- Lead contact info encrypted at rest
- Email/phone masked in analytics events
- GDPR deletion support (soft delete with cascading cleanup)
- Audit logs for all PII access

---

## Rollout Phases (Updated with Dependencies)

### Phase 1: Lead Capture (4-5 days)
**Dependencies:** None
- [ ] Domain models, repositories, services
- [ ] GraphQL API
- [ ] Spam detection
- [ ] Rate limiting
- [ ] Email notifications

### Phase 2: Messaging Infrastructure (6-8 days)
**Dependencies:** Phase 1 (lead integration), storage module
- [ ] Conversation & message models
- [ ] Delivery tracking
- [ ] Attachments with virus scanning
- [ ] Push notifications
- [ ] GraphQL API

### Phase 3: Analytics & Tracking (4-5 days)
**Dependencies:** listing-promotion module (for promotion tracking)
- [ ] Event ingestion
- [ ] Daily aggregation worker
- [ ] Stats queries
- [ ] Integration with promotion system

### Phase 4: Scheduling (Optional - Future) (3-4 days)
**Dependencies:** Phase 2 (messaging confirmations)
- [ ] Viewing request models
- [ ] Calendar integration
- [ ] Confirmation workflow

### Phase 5: Premium Visibility Analytics (2-3 days)
**Dependencies:** Phase 3, listing-promotion system deployed
- [ ] Boost performance tracking
- [ ] ROI calculations
- [ ] Performance reports

**Total Estimated Effort:** 19-25 days for Phases 1-3 (core system)

---

## Success Metrics

### Lead Management
- **Lead Response Time**: Median time to first contact < 2 hours
- **Lead Quality Score**: >70% verified leads (not spam)
- **Assignment Efficiency**: >90% auto-assigned within 5 minutes

### Messaging
- **Message Delivery Rate**: >99.9% delivered within 1 second
- **Read Rate**: >60% messages read within 24 hours
- **Spam/Abuse Rate**: <0.1% of messages flagged

### Analytics
- **Inquiry-to-Viewing Conversion**: Track by listing and business
- **Listing Engagement**: Views, saves, inquiries per day
- **Promotion ROI**: Revenue per featured/premium listing

### Premium Visibility (Integrated with Promotion System)
- **Boost Adoption**: % of listings using promotions
- **CTR on Featured Listings**: >5% click-through rate
- **Premium Conversion**: % of free users upgrading to paid plans

---

## Risks & Mitigations

### Technical Risks
| Risk | Impact | Mitigation |
|------|--------|------------|
| Spam leads overwhelming system | High | CAPTCHA, rate limits, ML-based detection |
| Message delivery failures | High | Retry logic, delivery receipts, monitoring |
| Analytics drift/inaccuracy | Medium | Daily reconciliation, audit logs |
| Attachment storage costs | Medium | Size limits, compression, CDN caching |

### Business Risks
| Risk | Impact | Mitigation |
|------|--------|------------|
| Low lead quality | High | Verification workflow, quality scoring |
| Messaging abuse | High | Moderation pipeline, user blocking |
| Analytics not actionable | Medium | User testing, iterative dashboard design |
| Low boost/promotion adoption | High | Clear ROI metrics, free trials, education |

---

## 🔧 Integration Checklist

- [ ] Leads service initialized in `cmd/api/main.go`
- [ ] Messaging service initialized in `cmd/api/main.go`
- [ ] Analytics service initialized in `cmd/worker/main.go`
- [ ] GraphQL schema includes all modules
- [ ] Worker jobs registered (aggregation, notifications)
- [ ] Cloud Scheduler configured
- [ ] Migration applied to database
- [ ] Storage service configured for attachments
- [ ] Promotion system integration complete (analytics hooks)

---

## 📝 Testing Strategy

### Unit Tests
- [ ] Domain model validation
- [ ] Spam detection algorithm
- [ ] Rate limiting logic
- [ ] Message delivery state machine
- [ ] Analytics aggregation calculations

### Integration Tests
- [ ] End-to-end lead submission → assignment → notification
- [ ] Messaging flow with delivery tracking
- [ ] Event ingestion → aggregation → reporting
- [ ] Attachment upload → virus scan → storage

### Load Tests
- [ ] 100 leads/second submission
- [ ] 1000 messages/second delivery
- [ ] 10k events/second ingestion

### Manual Testing
- [ ] Submit lead, verify assignment
- [ ] Send messages, verify delivery receipts
- [ ] Track events, verify stats accuracy
- [ ] Test spam detection with various inputs
- [ ] Verify notifications sent correctly

---

## 📊 Progress Tracking

**Current Phase:** Phase 0 - Planning
**Completion:** 0/5 phases
**Related Systems:** Listing Promotion System (monetization)

**Phase Summary:**
- ⏸️  Phase 1: Lead Capture - 0%
- ⏸️  Phase 2: Messaging Infrastructure - 0%
- ⏸️  Phase 3: Analytics & Tracking - 0%
- ⏸️  Phase 4: Scheduling (Optional) - 0%
- ⏸️  Phase 5: Premium Visibility Analytics - 0%

**Next Steps:**
1. Review and approve plan
2. Create leads module foundation
3. Implement spam detection
4. Build GraphQL API for leads

---

## 📌 Architecture Decisions

- **Messaging Pattern:** Airbnb-style with delivery receipts (not real-time chat)
- **Lead Assignment:** Auto-assignment via business routing rules
- **Spam Detection:** Pattern-based + IP reputation (ML future enhancement)
- **Analytics Storage:** Daily rollups for efficiency (raw events retained 90 days)
- **Attachment Storage:** Cloud Storage with virus scanning via Cloud Functions
- **Notification Strategy:** Email + push (in-app future enhancement)
- **Rate Limiting:** Redis-based distributed rate limiter
- **Idempotency:** Client-generated IDs for message deduplication

---

**Last Updated:** 2025-12-30
**Estimated Total Effort:** 19-25 days for core system (Phases 1-3)
**Risk Level:** Medium (messaging reliability, analytics accuracy)
**Business Impact:** HIGH - Core engagement and monetization features
**Related Plans:** [Listing Promotion System](LISTING_PROMOTION_SYSTEM_IMPLEMENTATION_PLAN.md)
