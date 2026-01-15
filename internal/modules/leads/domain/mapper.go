package domain

import (
	"errors"
	"hauslet/internal/modules/leads/repository/schema"
)

// MapLeadFromSchema converts a GORM Lead schema to domain Lead
func MapLeadFromSchema(schemaLead *schema.Lead) *Lead {
	if schemaLead == nil {
		return nil
	}

	return &Lead{
		ID:              schemaLead.ID,
		ListingID:       schemaLead.ListingID,
		BusinessID:      schemaLead.BusinessID,
		UserID:          schemaLead.UserID,
		IsVerified:      schemaLead.IsVerified,
		Name:            schemaLead.Name,
		Email:           schemaLead.Email,
		PhoneNumber:     schemaLead.PhoneNumber,
		Message:         schemaLead.Message,
		Source:          LeadSource(schemaLead.Source),
		Status:          LeadStatus(schemaLead.Status),
		SpamScore:       schemaLead.SpamScore,
		IsSpam:          schemaLead.IsSpam,
		AssignedTo:      schemaLead.AssignedTo,
		AssignedAt:      schemaLead.AssignedAt,
		AutoAssigned:    schemaLead.AutoAssigned,
		UserAgent:       schemaLead.UserAgent,
		IPAddress:       schemaLead.IPAddress,
		ReferrerURL:     schemaLead.ReferrerURL,
		UTMParams:       schemaLead.UTMParams,
		CustomMetadata:  schemaLead.CustomMetadata,
		FirstResponseAt: schemaLead.FirstResponseAt,
		ResponseTime:    schemaLead.ResponseTime,
		ResponseCount:   schemaLead.ResponseCount,
		CreatedAt:       schemaLead.CreatedAt,
		UpdatedAt:       schemaLead.UpdatedAt,
		DeletedAt:       nil, // GORM DeletedAt is different type
	}
}

// MapLeadToSchema converts a domain Lead to GORM Lead schema
func MapLeadToSchema(domainLead *Lead) (*schema.Lead, error) {
	if domainLead == nil {
		return nil, errors.New("lead cannot be nil")
	}

	return &schema.Lead{
		ID:              domainLead.ID,
		ListingID:       domainLead.ListingID,
		UserID:          domainLead.UserID,
		BusinessID:      domainLead.BusinessID,
		Name:            domainLead.Name,
		Email:           domainLead.Email,
		PhoneNumber:     domainLead.PhoneNumber,
		Message:         domainLead.Message,
		Source:          string(domainLead.Source),
		Status:          string(domainLead.Status),
		SpamScore:       domainLead.SpamScore,
		IsSpam:          domainLead.IsSpam,
		AssignedTo:      domainLead.AssignedTo,
		AssignedAt:      domainLead.AssignedAt,
		AutoAssigned:    domainLead.AutoAssigned,
		UserAgent:       domainLead.UserAgent,
		IPAddress:       domainLead.IPAddress,
		ReferrerURL:     domainLead.ReferrerURL,
		UTMParams:       domainLead.UTMParams,
		CustomMetadata:  domainLead.CustomMetadata,
		FirstResponseAt: domainLead.FirstResponseAt,
		ResponseTime:    domainLead.ResponseTime,
		ResponseCount:   domainLead.ResponseCount,
		CreatedAt:       domainLead.CreatedAt,
		UpdatedAt:       domainLead.UpdatedAt,
	}, nil
}

// MapLeadsFromSchema converts a slice of GORM Lead schemas to domain Leads
func MapLeadsFromSchema(schemaLeads []*schema.Lead) []*Lead {
	if schemaLeads == nil {
		return nil
	}

	leads := make([]*Lead, 0, len(schemaLeads))
	for _, schemaLead := range schemaLeads {
		if mapped := MapLeadFromSchema(schemaLead); mapped != nil {
			leads = append(leads, mapped)
		}
	}
	return leads
}

// MapLeadEventFromSchema converts a GORM LeadEvent schema to domain LeadEvent
func MapLeadEventFromSchema(schemaEvent *schema.LeadEvent) *LeadEvent {
	if schemaEvent == nil {
		return nil
	}

	var oldStatus *LeadStatus
	var newStatus *LeadStatus

	if schemaEvent.OldStatus != nil {
		status := LeadStatus(*schemaEvent.OldStatus)
		oldStatus = &status
	}

	if schemaEvent.NewStatus != nil {
		status := LeadStatus(*schemaEvent.NewStatus)
		newStatus = &status
	}

	return &LeadEvent{
		ID:        schemaEvent.ID,
		LeadID:    schemaEvent.LeadID,
		EventType: LeadEventType(schemaEvent.EventType),
		ActorID:   schemaEvent.ActorID,
		ActorType: ActorType(schemaEvent.ActorType),
		OldStatus: oldStatus,
		NewStatus: newStatus,
		Changes:   schemaEvent.Changes,
		Notes:     schemaEvent.Notes,
		CreatedAt: schemaEvent.CreatedAt,
	}
}

// MapLeadEventToSchema converts a domain LeadEvent to GORM LeadEvent schema
func MapLeadEventToSchema(domainEvent *LeadEvent) (*schema.LeadEvent, error) {
	if domainEvent == nil {
		return nil, errors.New("lead event cannot be nil")
	}

	var oldStatus *string
	var newStatus *string

	if domainEvent.OldStatus != nil {
		status := string(*domainEvent.OldStatus)
		oldStatus = &status
	}

	if domainEvent.NewStatus != nil {
		status := string(*domainEvent.NewStatus)
		newStatus = &status
	}

	return &schema.LeadEvent{
		ID:        domainEvent.ID,
		LeadID:    domainEvent.LeadID,
		EventType: string(domainEvent.EventType),
		ActorID:   domainEvent.ActorID,
		ActorType: string(domainEvent.ActorType),
		OldStatus: oldStatus,
		NewStatus: newStatus,
		Changes:   domainEvent.Changes,
		Notes:     domainEvent.Notes,
		CreatedAt: domainEvent.CreatedAt,
	}, nil
}

// MapLeadEventsFromSchema converts a slice of GORM LeadEvent schemas to domain LeadEvents
func MapLeadEventsFromSchema(schemaEvents []*schema.LeadEvent) []*LeadEvent {
	if schemaEvents == nil {
		return nil
	}

	events := make([]*LeadEvent, 0, len(schemaEvents))
	for _, schemaEvent := range schemaEvents {
		if mapped := MapLeadEventFromSchema(schemaEvent); mapped != nil {
			events = append(events, mapped)
		}
	}
	return events
}

// MapLeadAssignmentFromSchema converts a GORM LeadAssignment schema to domain LeadAssignment
func MapLeadAssignmentFromSchema(schemaAssignment *schema.LeadAssignment) *LeadAssignment {
	if schemaAssignment == nil {
		return nil
	}

	return &LeadAssignment{
		ID:         schemaAssignment.ID,
		LeadID:     schemaAssignment.LeadID,
		FromUserID: schemaAssignment.FromUserID,
		ToUserID:   schemaAssignment.ToUserID,
		Reason:     AssignmentReason(schemaAssignment.Reason),
		Notes:      schemaAssignment.Notes,
		AssignedBy: schemaAssignment.AssignedBy,
		AssignedAt: schemaAssignment.AssignedAt,
	}
}

// MapLeadAssignmentToSchema converts a domain LeadAssignment to GORM LeadAssignment schema
func MapLeadAssignmentToSchema(domainAssignment *LeadAssignment) (*schema.LeadAssignment, error) {
	if domainAssignment == nil {
		return nil, errors.New("lead assignment cannot be nil")
	}

	return &schema.LeadAssignment{
		ID:         domainAssignment.ID,
		LeadID:     domainAssignment.LeadID,
		FromUserID: domainAssignment.FromUserID,
		ToUserID:   domainAssignment.ToUserID,
		Reason:     string(domainAssignment.Reason),
		Notes:      domainAssignment.Notes,
		AssignedBy: domainAssignment.AssignedBy,
		AssignedAt: domainAssignment.AssignedAt,
	}, nil
}

// MapLeadAssignmentsFromSchema converts a slice of GORM LeadAssignment schemas to domain LeadAssignments
func MapLeadAssignmentsFromSchema(schemaAssignments []*schema.LeadAssignment) []*LeadAssignment {
	if schemaAssignments == nil {
		return nil
	}

	assignments := make([]*LeadAssignment, 0, len(schemaAssignments))
	for _, schemaAssignment := range schemaAssignments {
		if mapped := MapLeadAssignmentFromSchema(schemaAssignment); mapped != nil {
			assignments = append(assignments, mapped)
		}
	}
	return assignments
}
