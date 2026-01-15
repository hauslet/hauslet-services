package domain

// ConversationType defines the high-level category of a chat
type ConversationType string

const (
	ConversationTypeInquiry     ConversationType = "inquiry"
	ConversationTypeTransaction ConversationType = "transaction"
	ConversationTypeSupport     ConversationType = "support"
)

// ConversationContextType defines what entity this conversation is about
// Polymorphic link to other modules (Leads, Bookings, Roommates, etc.)
type ConversationContextType string

const (
	ContextTypeLead              ConversationContextType = "lead"
	ContextTypeBooking           ConversationContextType = "booking"
	ContextTypeRentalApplication ConversationContextType = "rental_application"
	ContextTypeSaleNegotiation   ConversationContextType = "sale_negotiation"
	ContextTypeRoommateListing   ConversationContextType = "roommate_listing"   // Future
	ContextTypeLegalConsultation ConversationContextType = "legal_consultation" // Future
	ContextTypeNone              ConversationContextType = "none"               // For general support
)

// ParticipantType defines the role of a user in a specific conversation
type ParticipantType string

const (
	ParticipantTypeUser         ParticipantType = "user"          // Standard user
	ParticipantTypeHost         ParticipantType = "host"          // Property owner/manager
	ParticipantTypeGuest        ParticipantType = "guest"         // Renter/Buyer/Inquirer
	ParticipantTypeSupportAgent ParticipantType = "support_agent" // Human support staff
	ParticipantTypeAIAgent      ParticipantType = "ai_agent"      // Vertex AI Bot
	ParticipantTypeSystem       ParticipantType = "system"        // Automated system messages
)

// MessageType defines the content structure of a message
type MessageType string

const (
	MessageTypeText   MessageType = "text"
	MessageTypeImage  MessageType = "image"
	MessageTypeFile   MessageType = "file"
	MessageTypeSystem MessageType = "system" // e.g., "Booking Confirmed"
)

// SupportStatus tracks the state of AI/Human intervention
type SupportStatus string

const (
	SupportStatusAIActive    SupportStatus = "ai_active"
	SupportStatusEscalated   SupportStatus = "escalated"    // Waiting for human
	SupportStatusAgentActive SupportStatus = "agent_active" // Human is handling
	SupportStatusResolved    SupportStatus = "resolved"
)
