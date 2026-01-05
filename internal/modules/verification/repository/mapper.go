package repository

import (
	"time"

	"hauslet/internal/modules/verification/domain"
	"hauslet/internal/modules/verification/repository/schema"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// SessionToSchema converts domain.VerificationSession to schema
func SessionToSchema(s *domain.VerificationSession) *schema.VerificationSessionSchema {
	sch := &schema.VerificationSessionSchema{
		ID:           s.ID,
		UserID:       s.UserID,
		Type:         s.Type.String(),
		Tier:         s.Tier.String(),
		Status:       s.Status.String(),
		Country:      s.Country,
		MaxAttempts:  s.MaxAttempts,
		AttemptsUsed: s.AttemptsUsed,
		LastAttemptAt: s.LastAttemptAt,
		ApprovedAt:    s.ApprovedAt,
		RejectedAt:    s.RejectedAt,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
		ExpiresAt:     s.ExpiresAt,
		CompletedAt:   s.CompletedAt,
		IPAddress:     s.IPAddress,
		UserAgent:     s.UserAgent,
	}

	// Map rejection reason
	if s.RejectionReason != nil {
		reason := s.RejectionReason.String()
		sch.RejectionReason = &reason
	}
	sch.RejectionNotes = s.RejectionNotes

	// Serialize verification data to JSONB
	sch.VerificationData = verificationDataToMap(s.Data, s.Type)

	return sch
}

// SessionFromSchema converts schema to domain.VerificationSession
func SessionFromSchema(sch *schema.VerificationSessionSchema) (*domain.VerificationSession, error) {
	vType := domain.VerificationType(sch.Type)

	s := &domain.VerificationSession{
		ID:             sch.ID,
		UserID:         sch.UserID,
		Type:           vType,
		Tier:           domain.VerificationTier(sch.Tier),
		Status:         domain.SessionStatus(sch.Status),
		Country:        sch.Country,
		MaxAttempts:    sch.MaxAttempts,
		AttemptsUsed:   sch.AttemptsUsed,
		LastAttemptAt:  sch.LastAttemptAt,
		ApprovedAt:     sch.ApprovedAt,
		RejectedAt:     sch.RejectedAt,
		RejectionNotes: sch.RejectionNotes,
		CreatedAt:      sch.CreatedAt,
		UpdatedAt:      sch.UpdatedAt,
		ExpiresAt:      sch.ExpiresAt,
		CompletedAt:    sch.CompletedAt,
		IPAddress:      sch.IPAddress,
		UserAgent:      sch.UserAgent,
	}

	// Deserialize rejection reason
	if sch.RejectionReason != nil {
		reason := domain.RejectionReason(*sch.RejectionReason)
		s.RejectionReason = &reason
	}

	// Deserialize verification data from JSONB
	s.Data = verificationDataFromMap(sch.VerificationData, vType)

	return s, nil
}

// AttemptToSchema converts domain.VerificationAttempt to schema
func AttemptToSchema(a *domain.VerificationAttempt) *schema.VerificationAttemptSchema {
	sch := &schema.VerificationAttemptSchema{
		ID:                a.ID,
		SessionID:         a.SessionID,
		ProviderName:      a.ProviderName,
		Status:            a.Status.String(),
		SubmittedAt:       a.SubmittedAt,
		CompletedAt:       a.CompletedAt,
		ProviderSessionID: a.ProviderSessionID,
		WebhookReceived:   a.WebhookReceived,
		WebhookReceivedAt: a.WebhookReceivedAt,
		CreatedAt:         a.CreatedAt,
		UpdatedAt:         a.UpdatedAt,
	}

	// Convert processing time
	if a.ProcessingTime != nil {
		nanos := int64(*a.ProcessingTime)
		sch.ProcessingTime = &nanos
	}

	// Convert evidence IDs
	evidenceStrings := make([]string, len(a.EvidenceIDs))
	for i, id := range a.EvidenceIDs {
		evidenceStrings[i] = id.String()
	}
	sch.EvidenceIDs = pq.StringArray(evidenceStrings)

	// Serialize result to JSONB
	if a.Result != nil {
		sch.Result = verificationResultToMap(*a.Result)
	}

	return sch
}

// AttemptFromSchema converts schema to domain.VerificationAttempt
func AttemptFromSchema(sch *schema.VerificationAttemptSchema) (*domain.VerificationAttempt, error) {
	a := &domain.VerificationAttempt{
		ID:                sch.ID,
		SessionID:         sch.SessionID,
		ProviderName:      sch.ProviderName,
		Status:            domain.AttemptStatus(sch.Status),
		SubmittedAt:       sch.SubmittedAt,
		CompletedAt:       sch.CompletedAt,
		ProviderSessionID: sch.ProviderSessionID,
		WebhookReceived:   sch.WebhookReceived,
		WebhookReceivedAt: sch.WebhookReceivedAt,
		CreatedAt:         sch.CreatedAt,
		UpdatedAt:         sch.UpdatedAt,
	}

	// Convert processing time
	if sch.ProcessingTime != nil {
		duration := time.Duration(*sch.ProcessingTime)
		a.ProcessingTime = &duration
	}

	// Convert evidence IDs
	evidenceIDs := make([]uuid.UUID, 0, len(sch.EvidenceIDs))
	for _, idStr := range sch.EvidenceIDs {
		id, err := uuid.Parse(idStr)
		if err == nil {
			evidenceIDs = append(evidenceIDs, id)
		}
	}
	a.EvidenceIDs = evidenceIDs

	// Deserialize result from JSONB
	if sch.Result != nil {
		result := verificationResultFromMap(sch.Result)
		a.Result = &result
	}

	return a, nil
}

// EvidenceToSchema converts domain.Evidence to schema
func EvidenceToSchema(e *domain.Evidence) *schema.VerificationEvidenceSchema {
	return &schema.VerificationEvidenceSchema{
		ID:         e.ID,
		SessionID:  e.SessionID,
		AttemptID:  e.AttemptID,
		Type:       e.Type.String(),
		Metadata:   evidenceMetadataToMap(e.Metadata),
		Verified:   e.Verified,
		VerifiedAt: e.VerifiedAt,
		CreatedAt:  e.CreatedAt,
		UpdatedAt:  e.UpdatedAt,
		DeletedAt:  e.DeletedAt,
	}
}

// EvidenceFromSchema converts schema to domain.Evidence
func EvidenceFromSchema(sch *schema.VerificationEvidenceSchema) (*domain.Evidence, error) {
	return &domain.Evidence{
		ID:         sch.ID,
		SessionID:  sch.SessionID,
		AttemptID:  sch.AttemptID,
		Type:       domain.EvidenceType(sch.Type),
		Metadata:   evidenceMetadataFromMap(sch.Metadata),
		Verified:   sch.Verified,
		VerifiedAt: sch.VerifiedAt,
		CreatedAt:  sch.CreatedAt,
		UpdatedAt:  sch.UpdatedAt,
		DeletedAt:  sch.DeletedAt,
	}, nil
}

// Helper functions for serialization/deserialization

func applicantInfoToMap(info domain.ApplicantInfo) schema.JSONBMap {
	m := schema.JSONBMap{
		"first_name":   info.FirstName,
		"last_name":    info.LastName,
		"date_of_birth": info.DateOfBirth.Format(time.RFC3339),
		"nationality":  info.Nationality,
	}
	if info.Email != nil {
		m["email"] = *info.Email
	}
	if info.PhoneNumber != nil {
		m["phone_number"] = *info.PhoneNumber
	}
	if info.Address != nil {
		m["address"] = addressToMap(*info.Address)
	}
	return m
}

func applicantInfoFromMap(m schema.JSONBMap) domain.ApplicantInfo {
	info := domain.ApplicantInfo{
		FirstName:  getString(m, "first_name"),
		LastName:   getString(m, "last_name"),
		Nationality: getString(m, "nationality"),
	}
	if dobStr := getString(m, "date_of_birth"); dobStr != "" {
		dob, _ := time.Parse(time.RFC3339, dobStr)
		info.DateOfBirth = dob
	}
	if email := getString(m, "email"); email != "" {
		info.Email = &email
	}
	if phone := getString(m, "phone_number"); phone != "" {
		info.PhoneNumber = &phone
	}
	if addrMap, ok := m["address"].(map[string]any); ok {
		addr := addressFromMap(addrMap)
		info.Address = &addr
	}
	return info
}

func addressToMap(addr domain.Address) map[string]any {
	m := map[string]any{
		"line1":       addr.Line1,
		"city":        addr.City,
		"postal_code": addr.PostalCode,
		"country":     addr.Country,
	}
	if addr.Line2 != nil {
		m["line2"] = *addr.Line2
	}
	if addr.State != nil {
		m["state"] = *addr.State
	}
	return m
}

func addressFromMap(m map[string]any) domain.Address {
	addr := domain.Address{
		Line1:      getString(m, "line1"),
		City:       getString(m, "city"),
		PostalCode: getString(m, "postal_code"),
		Country:    getString(m, "country"),
	}
	if line2 := getString(m, "line2"); line2 != "" {
		addr.Line2 = &line2
	}
	if state := getString(m, "state"); state != "" {
		addr.State = &state
	}
	return addr
}

func documentInfoToMap(doc domain.DocumentInfo) schema.JSONBMap {
	m := schema.JSONBMap{
		"type":            doc.Type.String(),
		"issuing_country": doc.IssuingCountry,
	}
	if doc.Number != nil {
		m["number"] = *doc.Number
	}
	if doc.IssueDate != nil {
		m["issue_date"] = doc.IssueDate.Format(time.RFC3339)
	}
	if doc.ExpiryDate != nil {
		m["expiry_date"] = doc.ExpiryDate.Format(time.RFC3339)
	}
	return m
}

func documentInfoFromMap(m schema.JSONBMap) domain.DocumentInfo {
	doc := domain.DocumentInfo{
		Type:           domain.DocumentType(getString(m, "type")),
		IssuingCountry: getString(m, "issuing_country"),
	}
	if number := getString(m, "number"); number != "" {
		doc.Number = &number
	}
	if issueStr := getString(m, "issue_date"); issueStr != "" {
		issueDate, _ := time.Parse(time.RFC3339, issueStr)
		doc.IssueDate = &issueDate
	}
	if expiryStr := getString(m, "expiry_date"); expiryStr != "" {
		expiryDate, _ := time.Parse(time.RFC3339, expiryStr)
		doc.ExpiryDate = &expiryDate
	}
	return doc
}

func verificationResultToMap(result domain.VerificationResult) schema.JSONBMap {
	m := schema.JSONBMap{
		"success":       result.Success,
		"provider_name": result.ProviderName,
		"provider_ref_id": result.ProviderRefID,
		"processed_at":  result.ProcessedAt.Format(time.RFC3339),
	}
	if result.Score != nil {
		m["score"] = *result.Score
	}
	if result.RejectionReason != nil {
		m["rejection_reason"] = result.RejectionReason.String()
	}
	if result.RejectionNotes != nil {
		m["rejection_notes"] = *result.RejectionNotes
	}
	if result.RawResponse != nil {
		m["raw_response"] = result.RawResponse
	}
	return m
}

func verificationResultFromMap(m schema.JSONBMap) domain.VerificationResult {
	result := domain.VerificationResult{
		Success:       getBool(m, "success"),
		ProviderName:  getString(m, "provider_name"),
		ProviderRefID: getString(m, "provider_ref_id"),
	}
	if processedStr := getString(m, "processed_at"); processedStr != "" {
		processed, _ := time.Parse(time.RFC3339, processedStr)
		result.ProcessedAt = processed
	}
	if score, ok := m["score"].(float64); ok {
		result.Score = &score
	}
	if reasonStr := getString(m, "rejection_reason"); reasonStr != "" {
		reason := domain.RejectionReason(reasonStr)
		result.RejectionReason = &reason
	}
	if notes := getString(m, "rejection_notes"); notes != "" {
		result.RejectionNotes = &notes
	}
	if raw, ok := m["raw_response"].(map[string]any); ok {
		result.RawResponse = raw
	}
	return result
}

func evidenceMetadataToMap(meta domain.EvidenceMetadata) schema.JSONBMap {
	m := schema.JSONBMap{
		"evidence_id": meta.EvidenceID.String(),
		"type":        meta.Type.String(),
		"url":         meta.URL,
		"hash":        meta.Hash,
		"size":        meta.Size,
		"mime_type":   meta.MimeType,
		"uploaded_at": meta.UploadedAt.Format(time.RFC3339),
	}
	if meta.UploadedByIP != nil {
		m["uploaded_by_ip"] = *meta.UploadedByIP
	}
	return m
}

func evidenceMetadataFromMap(m schema.JSONBMap) domain.EvidenceMetadata {
	meta := domain.EvidenceMetadata{
		Type:     domain.EvidenceType(getString(m, "type")),
		URL:      getString(m, "url"),
		Hash:     getString(m, "hash"),
		MimeType: getString(m, "mime_type"),
	}
	if idStr := getString(m, "evidence_id"); idStr != "" {
		id, _ := uuid.Parse(idStr)
		meta.EvidenceID = id
	}
	if size, ok := m["size"].(float64); ok {
		meta.Size = int64(size)
	}
	if uploadedStr := getString(m, "uploaded_at"); uploadedStr != "" {
		uploaded, _ := time.Parse(time.RFC3339, uploadedStr)
		meta.UploadedAt = uploaded
	}
	if ip := getString(m, "uploaded_by_ip"); ip != "" {
		meta.UploadedByIP = &ip
	}
	return meta
}

// Helper functions for VerificationData union type

func verificationDataToMap(data domain.VerificationData, vType domain.VerificationType) schema.JSONBMap {
	m := schema.JSONBMap{}

	switch vType {
	case domain.VerificationIdentity:
		if data.Identity != nil {
			m["identity"] = identityDataToMap(*data.Identity)
		}
	case domain.VerificationPhone:
		if data.Phone != nil {
			m["phone"] = phoneDataToMap(*data.Phone)
		}
	case domain.VerificationAddress:
		if data.Address != nil {
			m["address"] = addressDataToMap(*data.Address)
		}
	case domain.VerificationBusiness:
		if data.Business != nil {
			m["business"] = businessDataToMap(*data.Business)
		}
	}

	return m
}

func verificationDataFromMap(m schema.JSONBMap, vType domain.VerificationType) domain.VerificationData {
	data := domain.VerificationData{}

	switch vType {
	case domain.VerificationIdentity:
		if idMap, ok := m["identity"].(map[string]any); ok {
			identity := identityDataFromMap(idMap)
			data.Identity = &identity
		}
	case domain.VerificationPhone:
		if phoneMap, ok := m["phone"].(map[string]any); ok {
			phone := phoneDataFromMap(phoneMap)
			data.Phone = &phone
		}
	case domain.VerificationAddress:
		if addrMap, ok := m["address"].(map[string]any); ok {
			address := addressDataFromMap(addrMap)
			data.Address = &address
		}
	case domain.VerificationBusiness:
		if bizMap, ok := m["business"].(map[string]any); ok {
			business := businessDataFromMap(bizMap)
			data.Business = &business
		}
	}

	return data
}

func identityDataToMap(data domain.IdentityData) map[string]any {
	return map[string]any{
		"applicant_info": applicantInfoToMap(data.ApplicantInfo),
		"document_info":  documentInfoToMap(data.DocumentInfo),
	}
}

func identityDataFromMap(m map[string]any) domain.IdentityData {
	data := domain.IdentityData{}

	if applicantMap, ok := m["applicant_info"].(map[string]any); ok {
		data.ApplicantInfo = applicantInfoFromMap(applicantMap)
	}
	if docMap, ok := m["document_info"].(map[string]any); ok {
		data.DocumentInfo = documentInfoFromMap(docMap)
	}

	return data
}

func phoneDataToMap(data domain.PhoneData) map[string]any {
	m := map[string]any{
		"phone_number":      data.PhoneNumber,
		"country_code":      data.CountryCode,
		"otp_attempts":      data.OTPAttempts,
		"max_otp_attempts":  data.MaxOTPAttempts,
	}

	// OTP code is NOT stored in DB - it's in Redis with TTL
	if data.OTPGeneratedAt != nil {
		m["otp_generated_at"] = data.OTPGeneratedAt.Format(time.RFC3339)
	}
	if data.OTPExpiresAt != nil {
		m["otp_expires_at"] = data.OTPExpiresAt.Format(time.RFC3339)
	}
	if data.VerifiedAt != nil {
		m["verified_at"] = data.VerifiedAt.Format(time.RFC3339)
	}
	if data.SMSProvider != nil {
		m["sms_provider"] = *data.SMSProvider
	}

	return m
}

func phoneDataFromMap(m map[string]any) domain.PhoneData {
	data := domain.PhoneData{
		PhoneNumber:    getString(m, "phone_number"),
		CountryCode:    getString(m, "country_code"),
		OTPAttempts:    getInt(m, "otp_attempts"),
		MaxOTPAttempts: getInt(m, "max_otp_attempts"),
	}

	// OTP code is NOT stored in DB - it's in Redis with TTL
	if genStr := getString(m, "otp_generated_at"); genStr != "" {
		gen, _ := time.Parse(time.RFC3339, genStr)
		data.OTPGeneratedAt = &gen
	}
	if expStr := getString(m, "otp_expires_at"); expStr != "" {
		exp, _ := time.Parse(time.RFC3339, expStr)
		data.OTPExpiresAt = &exp
	}
	if verStr := getString(m, "verified_at"); verStr != "" {
		ver, _ := time.Parse(time.RFC3339, verStr)
		data.VerifiedAt = &ver
	}
	if provider := getString(m, "sms_provider"); provider != "" {
		data.SMSProvider = &provider
	}

	return data
}

func addressDataToMap(data domain.AddressData) map[string]any {
	m := map[string]any{
		"full_address":  data.FullAddress,
		"street":        data.Street,
		"city":          data.City,
		"state":         data.State,
		"postal_code":   data.PostalCode,
		"country":       data.Country,
		"document_type": data.DocumentType,
	}

	if data.IssueDate != nil {
		m["issue_date"] = data.IssueDate.Format(time.RFC3339)
	}
	if data.VerifiedAt != nil {
		m["verified_at"] = data.VerifiedAt.Format(time.RFC3339)
	}
	if data.MatchScore != nil {
		m["match_score"] = *data.MatchScore
	}

	return m
}

func addressDataFromMap(m map[string]any) domain.AddressData {
	data := domain.AddressData{
		FullAddress:  getString(m, "full_address"),
		Street:       getString(m, "street"),
		City:         getString(m, "city"),
		State:        getString(m, "state"),
		PostalCode:   getString(m, "postal_code"),
		Country:      getString(m, "country"),
		DocumentType: getString(m, "document_type"),
	}

	if issueStr := getString(m, "issue_date"); issueStr != "" {
		issue, _ := time.Parse(time.RFC3339, issueStr)
		data.IssueDate = &issue
	}
	if verStr := getString(m, "verified_at"); verStr != "" {
		ver, _ := time.Parse(time.RFC3339, verStr)
		data.VerifiedAt = &ver
	}
	if score, ok := m["match_score"].(float64); ok {
		data.MatchScore = &score
	}

	return data
}

func businessDataToMap(data domain.BusinessData) map[string]any {
	m := map[string]any{
		"business_name":       data.BusinessName,
		"registration_number": data.RegistrationNumber,
		"business_type":       data.BusinessType,
		"country":             data.Country,
		"business_address":    addressToMap(data.BusinessAddress),
	}

	if data.TaxID != nil {
		m["tax_id"] = *data.TaxID
	}
	if data.IncorporationDate != nil {
		m["incorporation_date"] = data.IncorporationDate.Format(time.RFC3339)
	}
	if data.OwnerUserID != nil {
		m["owner_user_id"] = data.OwnerUserID.String()
	}
	if len(data.BeneficialOwners) > 0 {
		m["beneficial_owners"] = data.BeneficialOwners
	}
	if len(data.Directors) > 0 {
		m["directors"] = data.Directors
	}
	if data.VerifiedAt != nil {
		m["verified_at"] = data.VerifiedAt.Format(time.RFC3339)
	}
	if data.VerificationProvider != nil {
		m["verification_provider"] = *data.VerificationProvider
	}

	return m
}

func businessDataFromMap(m map[string]any) domain.BusinessData {
	data := domain.BusinessData{
		BusinessName:       getString(m, "business_name"),
		RegistrationNumber: getString(m, "registration_number"),
		BusinessType:       getString(m, "business_type"),
		Country:            getString(m, "country"),
	}

	if taxID := getString(m, "tax_id"); taxID != "" {
		data.TaxID = &taxID
	}
	if incStr := getString(m, "incorporation_date"); incStr != "" {
		inc, _ := time.Parse(time.RFC3339, incStr)
		data.IncorporationDate = &inc
	}
	if ownerStr := getString(m, "owner_user_id"); ownerStr != "" {
		ownerID, _ := uuid.Parse(ownerStr)
		data.OwnerUserID = &ownerID
	}
	if addrMap, ok := m["business_address"].(map[string]any); ok {
		data.BusinessAddress = addressFromMap(addrMap)
	}
	if owners, ok := m["beneficial_owners"].([]any); ok {
		data.BeneficialOwners = make([]string, 0, len(owners))
		for _, o := range owners {
			if s, ok := o.(string); ok {
				data.BeneficialOwners = append(data.BeneficialOwners, s)
			}
		}
	}
	if dirs, ok := m["directors"].([]any); ok {
		data.Directors = make([]string, 0, len(dirs))
		for _, d := range dirs {
			if s, ok := d.(string); ok {
				data.Directors = append(data.Directors, s)
			}
		}
	}
	if verStr := getString(m, "verified_at"); verStr != "" {
		ver, _ := time.Parse(time.RFC3339, verStr)
		data.VerifiedAt = &ver
	}
	if provider := getString(m, "verification_provider"); provider != "" {
		data.VerificationProvider = &provider
	}

	return data
}

// Helper functions
func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	if v, ok := m[key].(int); ok {
		return v
	}
	return 0
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}
