package domain

// LeadStatus represents the lifecycle state of a lead
type LeadStatus string

const (
	StatusNew       LeadStatus = "new"        // Just created, not yet contacted
	StatusAssigned  LeadStatus = "assigned"   // Assigned to an agent
	StatusContacted LeadStatus = "contacted"  // Agent has reached out
	StatusQualified LeadStatus = "qualified"  // Lead is qualified as serious
	StatusConverted LeadStatus = "converted"  // Deal closed (self-reported)
	StatusLost      LeadStatus = "lost"       // Lead did not convert
	StatusSpam      LeadStatus = "spam"       // Marked as spam
	StatusArchived  LeadStatus = "archived"   // Archived/closed
)

// String returns the string representation of the LeadStatus
func (s LeadStatus) String() string {
	return string(s)
}

// IsValid checks if the lead status is a valid value
func (s LeadStatus) IsValid() bool {
	switch s {
	case StatusNew, StatusAssigned, StatusContacted, StatusQualified,
		StatusConverted, StatusLost, StatusSpam, StatusArchived:
		return true
	}
	return false
}

// LeadSource represents where the lead originated from
type LeadSource string

const (
	SourceWebsite       LeadSource = "website"
	SourceMobileApp     LeadSource = "mobile_app"
	SourceAPI           LeadSource = "api"
	SourceWhatsApp      LeadSource = "whatsapp"
	SourcePhoneCall     LeadSource = "phone_call"
	SourceEmailCampaign LeadSource = "email_campaign"
)

// String returns the string representation of the LeadSource
func (s LeadSource) String() string {
	return string(s)
}

// IsValid checks if the lead source is a valid value
func (s LeadSource) IsValid() bool {
	switch s {
	case SourceWebsite, SourceMobileApp, SourceAPI,
		SourceWhatsApp, SourcePhoneCall, SourceEmailCampaign:
		return true
	}
	return false
}

// LeadEventType represents the type of event in a lead's history
type LeadEventType string

const (
	EventCreated      LeadEventType = "created"
	EventAssigned     LeadEventType = "assigned"
	EventStatusChange LeadEventType = "status_changed"
	EventContacted    LeadEventType = "contacted"
	EventNoteAdded    LeadEventType = "note_added"
	EventMarkedSpam   LeadEventType = "marked_spam"
)

// String returns the string representation of the LeadEventType
func (s LeadEventType) String() string {
	return string(s)
}

// IsValid checks if the lead event type is a valid value
func (s LeadEventType) IsValid() bool {
	switch s {
	case EventCreated, EventAssigned, EventStatusChange,
		EventContacted, EventNoteAdded, EventMarkedSpam:
		return true
	}
	return false
}

// ActorType represents who or what performed an action
type ActorType string

const (
	ActorUser       ActorType = "user"       // Human user action
	ActorSystem     ActorType = "system"     // System-initiated action
	ActorAutomation ActorType = "automation" // Automated process
)

// String returns the string representation of the ActorType
func (s ActorType) String() string {
	return string(s)
}

// IsValid checks if the actor type is a valid value
func (s ActorType) IsValid() bool {
	switch s {
	case ActorUser, ActorSystem, ActorAutomation:
		return true
	}
	return false
}

// AssignmentReason represents why a lead was assigned
type AssignmentReason string

const (
	ReasonAuto     AssignmentReason = "auto"     // Automatically assigned by system
	ReasonManual   AssignmentReason = "manual"   // Manually assigned by user
	ReasonReassign AssignmentReason = "reassign" // Reassigned from another agent
	ReasonEscalate AssignmentReason = "escalate" // Escalated to senior agent
)

// String returns the string representation of the AssignmentReason
func (s AssignmentReason) String() string {
	return string(s)
}

// IsValid checks if the assignment reason is a valid value
func (s AssignmentReason) IsValid() bool {
	switch s {
	case ReasonAuto, ReasonManual, ReasonReassign, ReasonEscalate:
		return true
	}
	return false
}
