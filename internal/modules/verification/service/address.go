package service

import (
	"context"
	"fmt"

	"hauslet/internal/modules/verification/domain"
)

// =======================
// Address Verification
// =======================

func (s *verificationService) SubmitAddressVerification(ctx context.Context, req SubmitAddressVerificationRequest) (*SubmitVerificationResponse, error) {
	// Get session and authorize
	session, err := s.GetSession(ctx, req.SessionID, req.UserID)
	if err != nil {
		return nil, err
	}

	// Ensure session is address verification type
	if session.Type != domain.VerificationAddress {
		return nil, fmt.Errorf("session is not address verification type")
	}

	// Build evidence items and delegate to shared manual review flow
	evidence := []ManualReviewEvidence{
		{
			Type:     s.mapAddressDocTypeToEvidence(req.DocumentType),
			Data:     req.ProofDocument,
			MimeType: "application/pdf",
		},
	}

	return s.submitManualReview(ctx, session, req.UserID, evidence, req.IPAddress)
}
