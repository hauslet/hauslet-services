package service

import (
	"context"
	"fmt"
	"net/http"

	"hauslet/internal/modules/verification/domain"
)

// SubmitListingVerification submits listing verification documents for manual review
func (s *verificationService) SubmitListingVerification(ctx context.Context, req SubmitListingVerificationRequest) (*SubmitVerificationResponse, error) {
	// Get session and authorize
	session, err := s.GetSession(ctx, req.SessionID, req.UserID)
	if err != nil {
		return nil, err
	}

	// Validate session state
	if session.Status != domain.SessionPending && session.Status != domain.SessionInProgress {
		return nil, domain.ErrInvalidSessionStatus
	}
	if session.Type != domain.VerificationListing {
		return nil, domain.ErrSessionTypeMismatch
	}

	// Map document type string to evidence type
	var evidenceType domain.EvidenceType
	switch req.DocumentType {
	case "title_deed":
		evidenceType = domain.EvidenceTitleDeed
	case "property_tax":
		evidenceType = domain.EvidencePropertyTax
	case "geo_tagged_photo":
		evidenceType = domain.EvidenceGeoTaggedPhoto
	default:
		return nil, fmt.Errorf("invalid document type: %s", req.DocumentType)
	}

	// Update session data with proof type
	listingData, err := session.GetListingData()
	if err != nil {
		return nil, err
	}
	listingData.ProofType = req.DocumentType

	// Detect MIME type from document bytes
	mimeType := http.DetectContentType(req.ProofDocument)

	// Submit via shared manual review helper
	return s.submitManualReview(ctx, session, req.UserID, []ManualReviewEvidence{
		{
			Type:     evidenceType,
			Data:     req.ProofDocument,
			MimeType: mimeType,
		},
	}, req.IPAddress)
}
