package domain

import (
	"encoding/json"
	"hauslet/internal/modules/finance/repository/schema"
)

// MapWalletFromSchema converts schema.Wallet to domain.Wallet
func MapWalletFromSchema(s *schema.Wallet) *Wallet {
	if s == nil {
		return nil
	}

	var metadata map[string]interface{}
	if s.Metadata != "" {
		json.Unmarshal([]byte(s.Metadata), &metadata)
	}

	return &Wallet{
		ID:         s.ID,
		OwnerType:  OwnerType(s.OwnerType),
		OwnerID:    s.OwnerID,
		WalletType: WalletType(s.WalletType),
		Balance:    s.Balance,
		Currency:   s.Currency,
		Status:     WalletStatus(s.Status),
		Metadata:   metadata,
		CreatedAt:  s.CreatedAt,
		UpdatedAt:  s.UpdatedAt,
	}
}

// MapWalletToSchema converts domain.Wallet to schema.Wallet
func MapWalletToSchema(d *Wallet) *schema.Wallet {
	if d == nil {
		return nil
	}

	var metadataJSON string
	if d.Metadata != nil {
		bytes, _ := json.Marshal(d.Metadata)
		metadataJSON = string(bytes)
	}

	return &schema.Wallet{
		ID:         d.ID,
		OwnerType:  d.OwnerType.String(),
		OwnerID:    d.OwnerID,
		WalletType: d.WalletType.String(),
		Balance:    d.Balance,
		Currency:   d.Currency,
		Status:     d.Status.String(),
		Metadata:   metadataJSON,
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	}
}

// MapLedgerEntryFromSchema converts schema.LedgerEntry to domain.LedgerEntry
func MapLedgerEntryFromSchema(s *schema.LedgerEntry) *LedgerEntry {
	if s == nil {
		return nil
	}

	return &LedgerEntry{
		ID:             s.ID,
		TransactionID:  s.TransactionID,
		Reference:      s.Reference,
		DebitWalletID:  s.DebitWalletID,
		CreditWalletID: s.CreditWalletID,
		Amount:         s.Amount,
		Currency:       s.Currency,
		ResourceType:   ResourceType(s.ResourceType),
		ResourceID:     s.ResourceID,
		Memo:           s.Memo,
		CreatedAt:      s.CreatedAt,
	}
}

// MapLedgerEntryToSchema converts domain.LedgerEntry to schema.LedgerEntry
func MapLedgerEntryToSchema(d *LedgerEntry) *schema.LedgerEntry {
	if d == nil {
		return nil
	}

	return &schema.LedgerEntry{
		ID:             d.ID,
		TransactionID:  d.TransactionID,
		Reference:      d.Reference,
		DebitWalletID:  d.DebitWalletID,
		CreditWalletID: d.CreditWalletID,
		Amount:         d.Amount,
		Currency:       d.Currency,
		ResourceType:   d.ResourceType.String(),
		ResourceID:     d.ResourceID,
		Memo:           d.Memo,
		CreatedAt:      d.CreatedAt,
	}
}

// MapTransactionFromSchema converts schema.Transaction to domain.Transaction
func MapTransactionFromSchema(s *schema.Transaction) *Transaction {
	if s == nil {
		return nil
	}

	var metadata map[string]interface{}
	if s.Metadata != "" {
		json.Unmarshal([]byte(s.Metadata), &metadata)
	}

	return &Transaction{
		ID:           s.ID,
		Type:         TransactionType(s.Type),
		Status:       TransactionStatus(s.Status),
		ResourceType: ResourceType(s.ResourceType),
		ResourceID:   s.ResourceID,
		Amount:       s.Amount,
		Currency:     s.Currency,
		PaymentID:    s.PaymentID,
		ErrorMessage: s.ErrorMessage,
		Metadata:     metadata,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

// MapTransactionToSchema converts domain.Transaction to schema.Transaction
func MapTransactionToSchema(d *Transaction) *schema.Transaction {
	if d == nil {
		return nil
	}

	var metadataJSON string
	if d.Metadata != nil {
		bytes, _ := json.Marshal(d.Metadata)
		metadataJSON = string(bytes)
	}

	return &schema.Transaction{
		ID:           d.ID,
		Type:         d.Type.String(),
		Status:       d.Status.String(),
		ResourceType: d.ResourceType.String(),
		ResourceID:   d.ResourceID,
		Amount:       d.Amount,
		Currency:     d.Currency,
		PaymentID:    d.PaymentID,
		ErrorMessage: d.ErrorMessage,
		Metadata:     metadataJSON,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
}

// MapDisbursementFromSchema converts schema.Disbursement to domain.Disbursement
func MapDisbursementFromSchema(s *schema.Disbursement) *Disbursement {
	if s == nil {
		return nil
	}

	return &Disbursement{
		ID:               s.ID,
		WalletID:         s.WalletID,
		TransactionID:    s.TransactionID,
		Amount:           s.Amount,
		Currency:         s.Currency,
		Provider:         s.Provider,
		TransferCode:     s.TransferCode,
		ProviderResponse: s.ProviderResponse,
		Status:           DisbursementStatus(s.Status),
		Attempts:         s.Attempts,
		NextRetryAt:      s.NextRetryAt,
		CompletedAt:      s.CompletedAt,
		FailureReason:    s.FailureReason,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
}

// MapDisbursementToSchema converts domain.Disbursement to schema.Disbursement
func MapDisbursementToSchema(d *Disbursement) *schema.Disbursement {
	if d == nil {
		return nil
	}

	return &schema.Disbursement{
		ID:               d.ID,
		WalletID:         d.WalletID,
		TransactionID:    d.TransactionID,
		Amount:           d.Amount,
		Currency:         d.Currency,
		Provider:         d.Provider,
		TransferCode:     d.TransferCode,
		ProviderResponse:d.ProviderResponse,
		Status:           d.Status.String(),
		Attempts:         d.Attempts,
		NextRetryAt:      d.NextRetryAt,
		CompletedAt:      d.CompletedAt,
		FailureReason:    d.FailureReason,
		CreatedAt:        d.CreatedAt,
		UpdatedAt:        d.UpdatedAt,
	}
}

// MapDisputeFromSchema converts schema.Dispute to domain.Dispute
func MapDisputeFromSchema(s *schema.Dispute) *Dispute {
	if s == nil {
		return nil
	}

	var evidence []DisputeEvidence
	if s.Evidence != "" {
		json.Unmarshal([]byte(s.Evidence), &evidence)
	}

	var resolution *DisputeResolution
	if s.Resolution != nil && *s.Resolution != "" {
		resolution = &DisputeResolution{}
		json.Unmarshal([]byte(*s.Resolution), resolution)
	}

	return &Dispute{
		ID:            s.ID,
		BookingID:     s.BookingID,
		WalletID:      s.WalletID,
		PaymentID:     s.PaymentID,
		FiledBy:       DisputeParty(s.FiledBy),
		FiledByID:     s.FiledByID,
		Reason:        DisputeReason(s.Reason),
		Status:        DisputeStatus(s.Status),
		Description:   s.Description,
		Amount:        s.Amount,
		Currency:      s.Currency,
		Evidence:      evidence,
		AdminNotes:    s.AdminNotes,
		Resolution:    resolution,
		ResolvedByID:  s.ResolvedByID,
		ResolvedAt:    s.ResolvedAt,
		RefundAmount:  s.RefundAmount,
		TransactionID: s.TransactionID,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}

// MapDisputeToSchema converts domain.Dispute to schema.Dispute
func MapDisputeToSchema(d *Dispute) *schema.Dispute {
	if d == nil {
		return nil
	}

	var evidenceJSON string
	if d.Evidence != nil {
		bytes, _ := json.Marshal(d.Evidence)
		evidenceJSON = string(bytes)
	}

	var resolutionJSON *string
	if d.Resolution != nil {
		bytes, _ := json.Marshal(d.Resolution)
		resJSON := string(bytes)
		resolutionJSON = &resJSON
	}

	return &schema.Dispute{
		ID:            d.ID,
		BookingID:     d.BookingID,
		WalletID:      d.WalletID,
		PaymentID:     d.PaymentID,
		FiledBy:       d.FiledBy.String(),
		FiledByID:     d.FiledByID,
		Reason:        d.Reason.String(),
		Status:        d.Status.String(),
		Description:   d.Description,
		Amount:        d.Amount,
		Currency:      d.Currency,
		Evidence:      evidenceJSON,
		AdminNotes:    d.AdminNotes,
		Resolution:    resolutionJSON,
		ResolvedByID:  d.ResolvedByID,
		ResolvedAt:    d.ResolvedAt,
		RefundAmount:  d.RefundAmount,
		TransactionID: d.TransactionID,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}
