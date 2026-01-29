package domain

// VerificationType represents the type of verification being performed
type VerificationType string

const (
	VerificationIdentity VerificationType = "identity" // KYC/ID verification
	VerificationPhone    VerificationType = "phone"    // Phone number + OTP verification
	VerificationAddress  VerificationType = "address"  // Physical address verification
	VerificationBusiness VerificationType = "business" // Business/entity verification
	VerificationListing  VerificationType = "listing"  // Listing verification
)

func (v VerificationType) String() string {
	return string(v)
}

func (v VerificationType) IsValid() bool {
	switch v {
	case VerificationIdentity, VerificationPhone, VerificationAddress, VerificationBusiness, VerificationListing:
		return true
	}
	return false
}

// SessionStatus represents the overall verification session state
type SessionStatus string

const (
	SessionPending    SessionStatus = "pending"     // Initial state
	SessionInProgress SessionStatus = "in_progress" // KYC submission in flight
	SessionApproved   SessionStatus = "approved"    // Verification passed
	SessionRejected   SessionStatus = "rejected"    // Verification failed
	SessionExpired    SessionStatus = "expired"     // Session timeout
)

func (s SessionStatus) String() string {
	return string(s)
}

func (s SessionStatus) IsValid() bool {
	switch s {
	case SessionPending, SessionInProgress, SessionApproved, SessionRejected, SessionExpired:
		return true
	}
	return false
}

func (s SessionStatus) IsFinal() bool {
	return s == SessionApproved || s == SessionRejected || s == SessionExpired
}

// AttemptStatus represents individual KYC attempt state
type AttemptStatus string

const (
	AttemptPending    AttemptStatus = "pending"    // Queued for submission
	AttemptProcessing AttemptStatus = "processing" // Submitted to provider
	AttemptSuccess    AttemptStatus = "success"    // Provider returned success
	AttemptFailed     AttemptStatus = "failed"     // Provider returned failure
)

func (a AttemptStatus) String() string {
	return string(a)
}

func (a AttemptStatus) IsValid() bool {
	switch a {
	case AttemptPending, AttemptProcessing, AttemptSuccess, AttemptFailed:
		return true
	}
	return false
}

func (a AttemptStatus) IsFinal() bool {
	return a == AttemptSuccess || a == AttemptFailed
}

// VerificationTier defines the level of verification required
type VerificationTier string

const (
	TierBasic    VerificationTier = "basic"    // Document + selfie
	TierStandard VerificationTier = "standard" // Basic + liveness check
	TierEnhanced VerificationTier = "enhanced" // Standard + address verification
)

func (t VerificationTier) String() string {
	return string(t)
}

func (t VerificationTier) IsValid() bool {
	switch t {
	case TierBasic, TierStandard, TierEnhanced:
		return true
	}
	return false
}

// DocumentType represents the type of identity document
type DocumentType string

const (
	DocumentPassport        DocumentType = "passport"
	DocumentNationalID      DocumentType = "national_id"
	DocumentDriversLicense  DocumentType = "drivers_license"
	DocumentResidencePermit DocumentType = "residence_permit"
	// Nigeria-specific
	DocumentVotersCard DocumentType = "voters_card"
	DocumentNIN        DocumentType = "NIN" // National Identification Number
	DocumentBVN        DocumentType = "BVN" // Bank Verification Number
	DocumentVIN        DocumentType = "VIN" // Voter Identification Number
)

func (d DocumentType) String() string {
	return string(d)
}

func (d DocumentType) IsValid() bool {
	switch d {
	case DocumentPassport, DocumentNationalID, DocumentDriversLicense, DocumentResidencePermit,
		DocumentVotersCard, DocumentNIN, DocumentBVN, DocumentVIN:
		return true
	}
	return false
}

// RejectionReason codes for failed verifications
type RejectionReason string

const (
	RejectionDocumentExpired    RejectionReason = "document_expired"
	RejectionDocumentInvalid    RejectionReason = "document_invalid"
	RejectionDocumentUnreadable RejectionReason = "document_unreadable"
	RejectionPhotoMismatch      RejectionReason = "photo_mismatch"
	RejectionLivenessFailed     RejectionReason = "liveness_failed"
	RejectionUnderAge           RejectionReason = "under_age"
	RejectionSanctionedCountry  RejectionReason = "sanctioned_country"
	RejectionDuplicateAccount   RejectionReason = "duplicate_account"
	RejectionProviderError      RejectionReason = "provider_error"
	RejectionOther              RejectionReason = "other"
)

func (r RejectionReason) String() string {
	return string(r)
}

func (r RejectionReason) IsValid() bool {
	switch r {
	case RejectionDocumentExpired, RejectionDocumentInvalid, RejectionDocumentUnreadable,
		RejectionPhotoMismatch, RejectionLivenessFailed, RejectionUnderAge,
		RejectionSanctionedCountry, RejectionDuplicateAccount, RejectionProviderError, RejectionOther:
		return true
	}
	return false
}

// EvidenceType categorizes stored evidence
type EvidenceType string

const (
	// Identity verification evidence
	EvidenceDocumentFront EvidenceType = "document_front"
	EvidenceDocumentBack  EvidenceType = "document_back"
	EvidenceSelfie        EvidenceType = "selfie"
	EvidenceLiveness      EvidenceType = "liveness_video"

	// Address verification evidence
	EvidenceAddressUtilityBill   EvidenceType = "address_utility_bill"
	EvidenceAddressBankStatement EvidenceType = "address_bank_statement"
	EvidenceAddressLease         EvidenceType = "address_lease"
	EvidenceAddressOther         EvidenceType = "address_other"

	// Business verification evidence
	EvidenceBusinessRegistration EvidenceType = "business_registration"
	EvidenceBusinessTaxID        EvidenceType = "business_tax_id"
	EvidenceBusinessLicense      EvidenceType = "business_license"
	EvidenceBusinessAddress      EvidenceType = "business_address"

	// Phone verification evidence (metadata/logs)
	EvidencePhoneOTP EvidenceType = "phone_otp"

	// Listing verification evidence
	EvidenceTitleDeed      EvidenceType = "title_deed"
	EvidencePropertyTax    EvidenceType = "property_tax_receipt"
	EvidenceGeoTaggedPhoto EvidenceType = "geo_tagged_photo"
)

func (e EvidenceType) String() string {
	return string(e)
}

func (e EvidenceType) IsValid() bool {
	switch e {
	case EvidenceDocumentFront, EvidenceDocumentBack, EvidenceSelfie, EvidenceLiveness,
		EvidenceAddressUtilityBill, EvidenceAddressBankStatement, EvidenceAddressLease, EvidenceAddressOther,
		EvidenceBusinessRegistration, EvidenceBusinessTaxID, EvidenceBusinessLicense, EvidenceBusinessAddress,
		EvidencePhoneOTP,
		EvidenceTitleDeed, EvidencePropertyTax, EvidenceGeoTaggedPhoto:
		return true
	}
	return false
}

// TargetType defines what entity is being verified
type TargetType string

const (
	TargetUser    TargetType = "user"
	TargetListing TargetType = "listing"
)

func (t TargetType) String() string {
	return string(t)
}

func (t TargetType) IsValid() bool {
	switch t {
	case TargetUser, TargetListing:
		return true
	}
	return false
}
