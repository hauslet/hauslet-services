package leads

import "github.com/google/uuid"

// QualificationJobType identifies the lead qualification job.
const QualificationJobType = "leads:qualify"

// QualificationJobPayload defines the data for lead qualification.
type QualificationJobPayload struct {
	LeadID uuid.UUID `json:"lead_id"`
}
