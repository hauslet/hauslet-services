# Multi-Property Messaging Service with AI Support - Implementation Plan

## Overview

Build a real-time messaging system for **all property types** (sales, rentals, shortlets) and future features (roomies, lawyers). Supports conversation contexts (Inquiry, Transaction, Support) with GraphQL Subscriptions for real-time delivery and Vertex AI Agent Builder for intelligent first-line support.

**Primary Integration**: `internal/modules/leads` (already handles inquiries for ALL property types)

**Property Types Supported**:

- **Sale**: Property sales inquiries and negotiations
- **Rent**: Long-term rental inquiries
- **Shortlet**: Short-term booking inquiries + active booking conversations
- **Future**: Roommate matching, legal consultations

## Architecture Summary

```
User ↔ Agent/Owner Messaging (ALL property types: sale, rent, shortlet)
    ├─ Inquiry conversations (from Leads module - pre-transaction)
    ├─ Transaction conversations (shortlet bookings, sales negotiations, rental applications)
    └─ Support conversations (User ↔ AI/Human for platform support)
    +
GraphQL Subscriptions (WebSocket real-time delivery)
    +
Vertex AI Agent Builder (FAQ answers, policy questions, escalation to humans)
    +
Future: Roommate matching chats, Lawyer consultations
```

---

## Domain Model (Clean Architecture + DDD)

### Core Entities

**Conversation (Aggregate Root)**

- File: `internal/modules/messaging/domain/conversation.go`
- Types:
  - `inquiry` (pre-transaction for ALL property types - linked to Lead)
  - `transaction` (active transaction: shortlet booking, rental application, sale negotiation, future: roommate chat, lawyer consult)
  - `support` (user ↔ AI/support for platform help)
- Rich behavior: `AddMessage()`, `RequestSupport()`, `EscalateToHuman()`, `CanUserParticipate()`
- Polymorphic context: Links to Lead ID, Booking ID, Transaction ID (future), or null for support
- **Key Integration**: Automatically created from Leads module for all property types

**Message**

- File: `internal/modules/messaging/domain/message.go`
- Types: `text`, `system`, `image`, `file`
- Sender types: `user`, `ai_agent`, `support_agent`
- AI context tracking: session ID, confidence, token usage

**Participant**

- File: `internal/modules/messaging/domain/participant.go`
- Tracks who's in conversation, when they joined, last read time
- `isVisible` flag for transparent support intervention

**SupportState**

- File: `internal/modules/messaging/domain/support_state.go`
- Tracks AI session, escalation status, response times
- Manages AI ↔ Human handoff workflow

### Enums & Errors

- File: `internal/modules/messaging/domain/enums.go`
- File: `internal/modules/messaging/domain/errors.go`
- Domain enums with `String()` methods (ConversationType, ParticipantType, MessageType, SupportStatus)
- Sentinel errors (ErrUnauthorizedAccess, ErrConversationClosed, ErrAIServiceUnavailable)

---

## Repository Layer (GORM + Mappers)

### Database Schemas

- File: `internal/modules/messaging/repository/schema/conversation.go`
- File: `internal/modules/messaging/repository/schema/message.go`
- File: `internal/modules/messaging/repository/schema/participant.go`

**Key Design Decisions:**

- **JSONB fields**: `unread_counts` (map[userID]count), `support_state`, `metadata`, `ai_context`
- **Polymorphic references**: `context_type` + `context_id` (points to leads or bookings)
- **Performance indexes**: conversation type, last_message_at, sender_id, created_at

### Mappers

- File: `internal/modules/messaging/domain/mapper.go`
- Pattern: `MapConversationFromSchema()`, `MapConversationToSchema()`
- Handles JSONB ↔ Domain struct conversion

### Repository Interfaces

- File: `internal/modules/messaging/repository/interface.go`
- `ConversationRepository`: CRUD + `GetByContext()`, `ListForUser()`, `MarkAsRead()`
- `MessageRepository`: CRUD + `ListByConversation()`, `ListByConversationSince()`
- `ParticipantRepository`: CRUD + `ListByConversation()`

### Repository Implementations

- File: `internal/modules/messaging/repository/conversation_repo.go`
- File: `internal/modules/messaging/repository/message_repo.go`
- File: `internal/modules/messaging/repository/participant_repo.go`
- Standard GORM wrappers following booking repository pattern

---

## Service Layer (Business Logic + Authorization)

### Service Interface

- File: `internal/modules/messaging/service/interface.go`

**Core Methods:**

- `GetOrCreateInquiryConversation(leadID, requesterID)` - Pre-transaction chat for ANY property type (sale, rent, shortlet)
- `GetOrCreateTransactionConversation(contextType, contextID, requesterID)` - Active transaction chat (booking, rental application, sale negotiation)
- `GetOrCreateSupportConversation(userID)` - Persistent "Hauslet Support" contact
- `SendMessage(conversationID, senderID, content, type)` - Send message with auto-AI-response (publishes `EventMessageSent`)
- `GetMessages(conversationID, requesterID, limit, offset)` - Fetch messages
- `MarkAsRead(conversationID, userID)` - Update unread counts (publishes `EventMessageRead`)
- `RequestSupport(conversationID, requesterID)` - Escalate to human
- `AssignSupportAgent(conversationID, agentID, assignedBy)` - Human takes over

**Service Dependencies:**

- `events.Publisher` - For publishing message and conversation events
- Hook interfaces for cross-module integration

**Hook Interfaces (Cross-Module):**

- `LeadHooks`: Get lead participants (guest, host)
- `BookingHooks`: Get booking participants (guest, host)
- `ProfileHooks`: Get user info for display names

### Service Implementation

- File: `internal/modules/messaging/service/service.go`

**Authorization Pattern (follows booking service):**

```go
func (s *messagingServiceImpl) canAccessConversation(ctx, convID, userID) (*Conversation, error) {
    conv := s.convRepo.GetByID(convID)
    if !conv.CanUserParticipate(userID) {
        return nil, ErrUnauthorizedAccess
    }
    return conv, nil
}
```

**Key Logic:**

- `SendMessage()`: Authorizes sender, saves message, updates conversation tracking, **publishes event via `events.Publisher`**, triggers AI response if support conversation
- `handleAIResponse()`: Async goroutine to get AI response, save it, check for escalation
- **Multi-party authorization**:
  - Inquiry conversations: Prospective buyer/renter AND property owner/agent (works for sale, rent, shortlet)
  - Transaction conversations: Transaction participants (buyer-seller, tenant-landlord, guest-host)
  - Support conversations: User AND support team (AI + humans)

**Event Publishing Pattern:**

```go
func (s *messagingServiceImpl) SendMessage(ctx context.Context, conversationID uuid.UUID, senderID uuid.UUID, content string, messageType domain.MessageType) (*domain.Message, error) {
    // ... authorization and business logic ...
    
    // Save message
    message, err := s.messageRepo.Create(ctx, msg)
    if err != nil {
        return nil, err
    }
    
    // Publish event using existing infrastructure
    s.eventPublisher.Publish(
        ctx,
        events.ChannelMessages,
        events.EventMessageSent,
        message.ID.String(),
        message, // payload
        &events.PublishOptions{
            ActorID:      &senderID.String(),
            FailSilently: true,
            Metadata: map[string]string{
                "conversation_id": conversationID.String(),
            },
        },
    )
    
    return message, nil
}
```

### Authorization Helpers

- File: `internal/modules/messaging/service/authorization.go`
- `canAccessInquiryConversation()`: Verify user is prospective buyer/renter OR property owner/agent from lead (works for ALL property types)
- `canAccessTransactionConversation()`: Verify user is transaction participant (booking guest/host, sale buyer/seller, rental tenant/landlord)
- `isSupportAgent()`: Check user role for support queue access

### AI Support Service

- File: `internal/modules/messaging/service/ai_support.go`

**Interface:**

```go
type AISupportService interface {
    ProcessUserMessage(ctx, conversation, message) (*Message, error)
    ShouldEscalateToHuman(ctx, conversation, aiResponse) (bool, string, error)
    CreateSession(ctx, userID) (string, error)
    CloseSession(ctx, sessionID) error
}
```

**Escalation Triggers:**

- Low AI confidence (< 0.6)
- User requests "human agent", "real person", "complaint"
- Repeated failed intents
- Sensitive topics (refunds, disputes)

---

## Vertex AI Integration

### Vertex AI Client Wrapper

- File: `internal/modules/messaging/port/vertexai/client.go`

**Key Methods:**

```go
func (c *VertexAIClient) SendMessage(ctx, sessionID, message, history) (*AIResponse, error)
func (c *VertexAIClient) DetectIntent(ctx, sessionID, text) (*IntentDetectionResult, error)
```

**Context Building:**

- Last 10 messages as conversation history (token limit optimization)
- User profile info (name, verification status)
- Booking/listing context if available
- Platform policies from knowledge base

**Response Structure:**

- `Text`: AI response content
- `Confidence`: 0.0-1.0 score for escalation decisions
- `Intent`: Detected user intent (booking_question, refund_request, etc.)
- `RequiresEscalation`: Boolean flag from AI

---

## GraphQL Layer (Using Existing Events Infrastructure)

### GraphQL Schema

- File: `internal/modules/messaging/port/graphql/schema.graphqls`

**Types:**

```graphql
type Conversation {
  id: UUID!
  type: ConversationType!
  participants: [Participant!]!
  lastMessageAt: Time
  unreadCount: Int!
  supportState: SupportState
  messages(limit: Int, offset: Int): [Message!]!
}

type Message {
  id: UUID!
  senderID: UUID!
  senderType: ParticipantType!
  content: String!
  aiContext: AIMessageContext
  isRead: Boolean!
}
```

**Queries:**

```graphql
extend type Query {
  conversation(id: UUID!): Conversation
  myConversations(limit: Int, offset: Int): [Conversation!]!
  hausletSupport: Conversation!  # Get/create persistent support contact
  supportQueue(assignedToMe: Boolean, limit: Int): [Conversation!]!  # Admin/support only
}
```

**Mutations:**

```graphql
extend type Mutation {
  # Start inquiry conversation from lead (works for sale, rent, shortlet)
  startInquiryConversation(leadID: UUID!): Conversation!

  # Start transaction conversation (contextType: "booking", "rental_application", "sale_negotiation", etc.)
  startTransactionConversation(contextType: String!, contextID: UUID!): Conversation!

  # Messaging operations
  sendMessage(conversationID: UUID!, content: String!, type: MessageType!): Message!
  markConversationAsRead(conversationID: UUID!): Boolean!
  requestHumanSupport(conversationID: UUID!): Conversation!
}
```

**Subscriptions:**

```graphql
extend type Subscription {
  messageAdded(conversationID: UUID!): Message!
  conversationUpdated(conversationID: UUID!): Conversation!
  myConversationsUpdated: Conversation!
}
```

### Resolver Implementation

- File: `internal/modules/messaging/port/graphql/resolvers.go`

**Pattern (follows booking/leads):**

- Thin adapter: Extract user from context, delegate to service
- No authorization in resolver (service layer handles it)
- **Uses existing `events.Subscriber`** for real-time subscriptions
- Subscription authorization: Verify user can access conversation before subscribing

**Key Resolvers:**

```go
func (r *Resolver) HausletSupport(ctx context.Context) (*Conversation, error) {
    userID := getUserIDFromContext(ctx)
    return r.messagingSvc.GetOrCreateSupportConversation(ctx, userID)
}

func (r *Resolver) MessageAdded(ctx context.Context, conversationID uuid.UUID) (<-chan *Message, error) {
    userID := getUserIDFromContext(ctx)
    
    // Verify access first
    _, err := r.messagingSvc.GetConversation(ctx, conversationID, userID)
    if err != nil {
        return nil, err
    }
    
    // Subscribe using existing events infrastructure
    eventCh, err := r.eventSubscriber.SubscribeWithFilter(
        ctx,
        events.ChannelMessages,
        events.FilterByType(events.EventMessageSent),
        events.FilterByMetadata("conversation_id", conversationID.String()),
    )
    if err != nil {
        return nil, err
    }
    
    // Create output channel
    outCh := make(chan *Message, 1)
    
    // Process events
    go func() {
        defer close(outCh)
        
        for {
            select {
            case <-ctx.Done():
                return
                
            case event, ok := <-eventCh:
                if !ok {
                    return
                }
                
                // Parse message from payload
                var msg domain.Message
                if err := json.Unmarshal(event.Payload, &msg); err != nil {
                    continue
                }
                
                // Re-authorize: verify user can still access this conversation
                if !r.messagingSvc.CanAccessConversation(ctx, conversationID, userID) {
                    continue
                }
                
                // Convert and send
                outCh <- toGraphQLMessage(&msg)
            }
        }
    }()
    
    return outCh, nil
}
```

### Authorization Helpers

- File: `internal/modules/messaging/port/graphql/authorization.go`
- `getUserIDFromContext(ctx)`: Extract viewer from context
- `isAdminRole(role)`: Check for admin/root access
- `isSupportRole(role)`: Check for support agent role

---

## Database Migration

### Migration File

- File: `db/migrations/004_create_messaging_tables.sql`

**Note:** Check latest migration number. Currently migrations go up to 003, so this should be 004.

**Tables:**

1. **conversations**: id, type, status, context_type, context_id, last_message_at, unread_counts (JSONB), support_state (JSONB)
2. **messages**: id, conversation_id, sender_id, sender_type, message_type, content, metadata (JSONB), read_by (JSONB), ai_context (JSONB)
3. **conversation_participants**: id, conversation_id, user_id, type, joined_at, left_at, last_read_at, is_muted, is_visible

**Indexes:**

- `conversations`: type, status, context (type+id), last_message_at, support_status
- `messages`: conversation_id+created_at, sender_id, created_at
- `participants`: conversation_id, user_id

**Constraints:**

- Unique: One active conversation per context (lead/booking)
- Unique: One active participant record per user per conversation
- Foreign keys: context_id → leads/bookings, conversation_id CASCADE delete

**Triggers:**

- `updated_at` auto-update on conversations and messages

---

## GraphQL Subscriptions Setup

### WebSocket Transport Configuration

- File: `internal/transport/graph/server.go` **(ALREADY EXISTS)**

**Note:** WebSocket transport is already configured (lines 104-115). No changes needed. The existing setup includes:

- Keep-alive ping interval (10 seconds)
- Origin validation
- Support for development and production environments

**Optional Enhancement:** Add `InitFunc` for WebSocket authentication if needed:

```go
// In SetupGraphQL, modify existing WebSocket transport:
srv.AddTransport(transport.Websocket{
    KeepAlivePingInterval: 10 * time.Second,
    Upgrader: websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool {
            origin := r.Header.Get("Origin")
            return origin == "" || 
                origin == r.Header.Get("Host") || 
                origin == cfg.App.Client || 
                cfg.App.Env != "production"
        },
    },
    // Optional: Add InitFunc for WebSocket auth
    InitFunc: func(ctx context.Context, initPayload transport.InitPayload) (context.Context, *transport.InitPayload, error) {
        // Extract auth token from connection init
        authToken := initPayload.Authorization()
        if authToken == "" {
            return nil, nil, errors.New("missing auth token")
        }
        // Validate JWT and set viewer context
        // Return context with authenticated user
        return ctx, &initPayload, nil
    },
})
```

### Events Infrastructure Integration

- File: `internal/platform/events/types.go` **(MODIFY)**

**Add messaging event types:**

```go
// Add to EventType constants:
EventMessageSent          EventType = "message.sent"
EventMessageRead          EventType = "message.read"
EventConversationUpdated  EventType = "conversation.updated"
EventConversationCreated  EventType = "conversation.created"

// ChannelMessages already exists in types.go!
```

**Note:** The events infrastructure (`internal/platform/events`) is already set up:

- `events.Broker` - Redis pub/sub broker
- `events.Publisher` - Service-friendly event publishing
- `events.Subscriber` - Resolver-friendly event subscription with filtering
- Already wired in `cmd/api/server/container.go` as `EventPublisher` and `EventSubscriber`

**Optional Enhancement:** Add convenience method to `events.Publisher` (similar to `PublishBookingEvent`, `PublishLeadEvent`):

```go
// In internal/platform/events/publisher.go
func (p *Publisher) PublishMessageEvent(ctx context.Context, eventType EventType, messageID string, payload interface{}, conversationID *string, actorID *string) error {
    opts := &PublishOptions{
        ActorID:      actorID,
        FailSilently: true,
    }
    if conversationID != nil {
        opts.Metadata = map[string]string{
            "conversation_id": *conversationID,
        }
    }
    return p.Publish(ctx, ChannelMessages, eventType, messageID, payload, opts)
}
```

### gqlgen Configuration

- File: `gqlgen.yml` **(MODIFY)**

**Add messaging schema:**

```yaml
schema:
  - internal/modules/messaging/port/graphql/*.graphqls

models:
  Conversation:
    model: hauslet/internal/modules/messaging/domain.Conversation
  Message:
    model: hauslet/internal/modules/messaging/domain.Message
  Participant:
    model: hauslet/internal/modules/messaging/domain.Participant
  # ... all messaging types
```

---

## Integration with Existing Modules

### Hooks Adapters

- File: `internal/modules/messaging/service/hooks_adapters.go`

**Lead Hooks (PRIMARY INTEGRATION):**

```go
type leadHooksAdapter struct {
    leadSvc leadsservice.LeadService
}

func (a *leadHooksAdapter) GetLeadParticipants(ctx, leadID) (prospectID, ownerID, err) {
    // Use lead service to get lead
    // Extract prospect (buyer/renter/guest) and owner/agent IDs
    // Works for ALL property types: sale, rent, shortlet
}

func (a *leadHooksAdapter) GetLeadContext(ctx, leadID) (*LeadContext, error) {
    // Get listing ID, property type, business ID from lead
    // This gives conversation full context about what property type it's for
}
```

**Booking Hooks:** For shortlet transaction conversations (guest-host during active booking)

**Profile Hooks:** Get user display names, verification status

**Property Hooks:** Get listing type (sale, rent, shortlet) for conversation context

### Container Wiring

- File: `cmd/api/server/container.go` **(MODIFY)**

**Add to Container struct:**

```go
MessagingSvc messagingservice.MessagingService
AISupportSvc messagingservice.AISupportService
```

**Add initialization method:**

```go
func (c *Container) initMessaging() error {
    // Initialize repos
    convRepo := messagingrepository.NewConversationRepository(c.DB)
    msgRepo := messagingrepository.NewMessageRepository(c.DB)
    partRepo := messagingrepository.NewParticipantRepository(c.DB)

    // Initialize Vertex AI client
    vertexAIClient := vertexai.NewVertexAIClient(ctx, projectID, agentID, location)

    // Initialize services
    c.AISupportSvc = messagingservice.NewAISupportService(vertexAIClient, c.Logger)

    // Initialize hook adapters
    leadHooks := messagingservice.NewLeadHooksAdapter(c.LeadSvc)
    bookingHooks := messagingservice.NewBookingHooksAdapter(c.BookingSvc)
    profileHooks := messagingservice.NewProfileHooksAdapter(c.ProfileSvc)

    // Initialize messaging service with existing EventPublisher
    c.MessagingSvc = messagingservice.NewMessagingService(
        convRepo, msgRepo, partRepo,
        c.AISupportSvc,
        leadHooks, bookingHooks, profileHooks,
        c.EventPublisher, // Use existing EventPublisher from container
        c.Logger,
    )

    return nil
}
```

**Note:** `EventPublisher` and `EventSubscriber` already exist in the Container (lines 131-132). No need to create new instances.

### GraphQL Resolver Wiring

- File: `internal/transport/graph/resolver.go` **(MODIFY)**

**Add to Resolver struct:**

```go
MessagingResolver *messaginggraphql.Resolver
```

**Pass services in NewResolver:**

```go
MessagingResolver: messaginggraphql.NewResolver(
    messagingSvc, 
    aiSupportSvc, 
    eventSubscriber, // Pass existing EventSubscriber for subscriptions
    log,
)
```

---

## Configuration

### Add Vertex AI Config

- File: `config/config.go` **(MODIFY)**

**Add struct:**

```go
type VertexAIConfig struct {
    ProjectID       string
    AgentID         string
    Location        string
    CredentialsPath string
}

type MessagingConfig struct {
    AIContextMessages      int
    AIConfidenceThreshold  float64
    AITimeoutSeconds       int
    MaxConversationAgeDays int
}
```

### Environment Variables

- File: `.env.example` **(ADD)**

```env
VERTEX_AI_PROJECT_ID=hauslet-production
VERTEX_AI_AGENT_ID=hauslet-support-agent
VERTEX_AI_LOCATION=us-central1
VERTEX_AI_CREDENTIALS_PATH=/path/to/credentials.json

MESSAGING_AI_CONTEXT_MESSAGES=10
MESSAGING_AI_CONFIDENCE_THRESHOLD=0.6
MESSAGING_AI_TIMEOUT_SECONDS=15
```

---

## Implementation Phases

### Phase 1: Core Messaging (No AI) - **Start Here**

**Duration:** 1-2 weeks

**Deliverables:**

1. Domain models with rich behavior (conversation.go, message.go, participant.go)
2. Repository layer with GORM schemas
3. Service layer with authorization (inquiry and booking conversations)
4. Database migration (004_create_messaging_tables.sql)
5. Basic GraphQL queries/mutations (NO subscriptions yet)

**Testing:**

- Unit tests for domain logic (CanUserParticipate, AddMessage behavior)
- Service layer integration tests (authorization checks)
- Manual GraphQL Playground testing

**Success Criteria:**

- Prospective buyer/renter can message property owner/agent about inquiry (works for sale, rent, shortlet listings)
- Messages persist to database correctly
- Unread counts update correctly
- Authorization prevents unauthorized access
- Conversations automatically created from Leads module integration

---

### Phase 2: GraphQL Subscriptions

**Duration:** 1 week

**Deliverables:**

1. Add messaging event types to `internal/platform/events/types.go` (EventMessageSent, EventConversationUpdated, etc.)
2. Update service layer to publish events via existing `events.Publisher` when messages are sent
3. Implement subscription resolvers using existing `events.Subscriber` with filters
4. Wire `EventSubscriber` into messaging GraphQL resolver
5. Optional: Add WebSocket `InitFunc` for enhanced authentication (WebSocket transport already exists)

**Testing:**

- WebSocket connection tests (using existing transport)
- Subscription auth tests (reject unauthenticated)
- Real-time delivery latency tests
- Concurrent subscription load tests
- Multi-instance subscription tests (events automatically work across instances via Redis)

**Success Criteria:**

- New messages appear in real-time (< 500ms)
- Subscriptions work across multiple API instances (via existing Redis pub/sub)
- Unauthorized users can't subscribe to others' conversations
- Events are properly filtered by conversation ID

---

### Phase 3: Vertex AI Integration

**Duration:** 1 week

**Deliverables:**

1. Vertex AI client wrapper (port/vertexai/client.go)
2. AI support service implementation
3. Support conversation type
4. "Hauslet Support" persistent contact
5. Escalation logic (confidence threshold, keywords)

**Testing:**

- AI response quality testing (common questions)
- Escalation trigger testing
- Fallback when AI service unavailable
- Token usage monitoring

**Success Criteria:**

- User messages "Hauslet Support" and gets instant AI response
- AI answers FAQ questions correctly (policies, pricing, etc.)
- AI escalates low-confidence or complaint messages
- Human agent can take over seamlessly

---

### Phase 4: Polish & Integration

**Duration:** 1 week

**Deliverables:**

1. Notification integration (email/push for new messages when offline)
2. Message read receipts
3. Typing indicators (optional)
4. File attachments support
5. Admin support dashboard

**Testing:**

- End-to-end flow testing
- Cross-module integration tests
- Performance testing (1000s of concurrent conversations)

---

## Critical Files Summary

### New Files to Create (40+ files)

**Domain Layer:**

- `internal/modules/messaging/domain/conversation.go`
- `internal/modules/messaging/domain/message.go`
- `internal/modules/messaging/domain/participant.go`
- `internal/modules/messaging/domain/support_state.go`
- `internal/modules/messaging/domain/enums.go`
- `internal/modules/messaging/domain/errors.go`
- `internal/modules/messaging/domain/mapper.go`

**Repository Layer:**

- `internal/modules/messaging/repository/schema/conversation.go`
- `internal/modules/messaging/repository/schema/message.go`
- `internal/modules/messaging/repository/schema/participant.go`
- `internal/modules/messaging/repository/interface.go`
- `internal/modules/messaging/repository/conversation_repo.go`
- `internal/modules/messaging/repository/message_repo.go`
- `internal/modules/messaging/repository/participant_repo.go`

**Service Layer:**

- `internal/modules/messaging/service/interface.go`
- `internal/modules/messaging/service/service.go` (includes event publishing)
- `internal/modules/messaging/service/authorization.go`
- `internal/modules/messaging/service/ai_support.go`
- `internal/modules/messaging/service/hooks_adapters.go`

**Port Layer:**

- `internal/modules/messaging/port/graphql/schema.graphqls`
- `internal/modules/messaging/port/graphql/resolvers.go` (uses events.Subscriber)
- `internal/modules/messaging/port/graphql/authorization.go`
- `internal/modules/messaging/port/vertexai/client.go`

**Events Integration:**

- `internal/platform/events/types.go` (add messaging event types)

**Database:**

- `db/migrations/004_create_messaging_tables.sql`

### Files to Modify (6 files)

1. **`gqlgen.yml`** - Add messaging schema paths and type bindings
2. **`internal/transport/graph/server.go`** - Optional: Add WebSocket InitFunc for auth (WebSocket transport already exists)
3. **`internal/transport/graph/resolver.go`** - Wire messaging resolver (pass EventSubscriber)
4. **`cmd/api/server/container.go`** - Initialize messaging services (use existing EventPublisher)
5. **`config/config.go`** - Add Vertex AI and messaging config
6. **`internal/platform/events/types.go`** - Add messaging event types (EventMessageSent, etc.)

---

## Key Design Decisions

### 1. Support Contact Pattern (Airbnb-style)

**Decision:** Persistent "Hauslet Support" conversation per user (lazy creation)
**Why:** Familiar UX, maintains conversation history, simpler than per-issue tickets

### 2. Message Storage

**Decision:** Separate messages table with foreign key
**Why:** Better performance for large conversations, easier pagination, standard pattern

### 3. Subscription Scalability

**Decision:** Use existing `internal/platform/events` infrastructure (Redis PubSub)
**Why:**

- Real-time delivery via existing `events.Publisher`/`events.Subscriber`
- Horizontally scalable (works across API instances automatically)
- Already wired in container, no new infrastructure needed
- Consistent with other modules (bookings, payments, etc.)

### 4. AI Context Window

**Decision:** Send last 10 messages to Vertex AI
**Why:** Token limit constraints, most relevant context is recent, cost optimization

### 5. Unread Counts

**Decision:** JSONB map in conversations table
**Why:** Atomic updates, simpler queries, typical conversation has <10 participants

### 6. Authorization

**Decision:** Service layer checks permissions (not GraphQL resolvers)
**Why:** Follows Clean Architecture, single source of truth, testable

---

## Success Metrics

**Phase 1 Success:**

- [ ] Guest-host messaging works for inquiries and bookings
- [ ] Messages persist and retrieve correctly
- [ ] Authorization prevents unauthorized access
- [ ] Unread counts update correctly

**Phase 2 Success:**

- [ ] Real-time message delivery (< 500ms latency)
- [ ] Subscriptions work across multiple API instances
- [ ] WebSocket authentication works

**Phase 3 Success:**

- [ ] AI responds to 80%+ of common questions
- [ ] AI escalation triggers work (low confidence, keywords)
- [ ] Human agents can take over conversations
- [ ] "Hauslet Support" contact appears in every user's inbox

**Production Ready:**

- [ ] 10,000+ concurrent WebSocket connections supported
- [ ] 95th percentile message delivery < 1 second
- [ ] AI response time < 3 seconds
- [ ] Zero unauthorized conversation access in security audit

---

## Future Extensibility (Roomies & Lawyers)

### How to Add New Conversation Types

The system is designed to be extensible for future features:

**For Roommate Matching ("Roomies"):**

1. Add new conversation context type: `roommate_matching`
2. Create `internal/modules/roommates/` module
3. Use messaging service with context: `startTransactionConversation(contextType: "roommate_matching", contextID: roommateListingID)`
4. Authorization: Both users seeking roommates can participate

**For Legal Consultations:**

1. Add new conversation context type: `legal_consultation`
2. Create `internal/modules/legal/` module
3. Use messaging service with context: `startTransactionConversation(contextType: "legal_consultation", contextID: consultationID)`
4. Authorization: User AND assigned lawyer can participate
5. AI support can answer common legal FAQs before escalating to lawyer

**No changes needed to core messaging module** - it already supports polymorphic contexts via `context_type` + `context_id` pattern.
