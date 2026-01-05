package domain

import (
	"hauslet/internal/modules/payments/repository/schema"
	"hauslet/internal/platform/payment"
	"strings"

	"github.com/google/uuid"
)

// ============================================================================
// Payment Mappers
// ============================================================================

// MapPaymentFromSchema converts schema.Payment to domain.Payment
func MapPaymentFromSchema(sp *schema.Payment) *Payment {
	if sp == nil {
		return nil
	}

	p := &Payment{
		ID:                sp.ID,
		Reference:         sp.Reference,
		ProviderRef:       sp.ProviderRef,
		BookingID:         sp.BookingID,
		BusinessID:        sp.BusinessID,
		ResourceType:      ResourceType(sp.ResourceType),
		ResourceID:        sp.ResourceID,
		PayerID:           sp.PayerID,
		PayerEmail:        sp.PayerEmail,
		PayerName:         sp.PayerName,
		Amount:            sp.Amount,
		Currency:          payment.Currency(sp.Currency),
		Market:            NormalizeMarket(Market(sp.Market)),
		Status:            PaymentStatus(sp.Status),
		PaymentMethod:     PaymentMethodType(sp.PaymentMethod),
		Provider:          sp.Provider,
		AuthorizationCode: sp.AuthorizationCode,
		RedirectURL:       sp.RedirectURL,
		RequiresAction:    sp.RequiresAction,
		Description:       sp.Description,
		Metadata:          metadataArrayToMap(sp.Metadata),
		RefundedAmount:    sp.RefundedAmount,
		RefundedAt:        sp.RefundedAt,
		CreatedAt:         sp.CreatedAt,
		UpdatedAt:         sp.UpdatedAt,
	}

	if sp.DeletedAt.Valid {
		p.DeletedAt = &sp.DeletedAt.Time
	}

	return p
}

// MapPaymentToSchema converts domain.Payment to schema.Payment
func MapPaymentToSchema(p *Payment) *schema.Payment {
	if p == nil {
		return nil
	}

	sp := &schema.Payment{
		ID:                p.ID,
		Reference:         p.Reference,
		ProviderRef:       p.ProviderRef,
		BookingID:         p.BookingID,
		BusinessID:        p.BusinessID,
		ResourceType:      schema.ResourceType(p.ResourceType),
		ResourceID:        p.ResourceID,
		PayerID:           p.PayerID,
		PayerEmail:        p.PayerEmail,
		PayerName:         p.PayerName,
		Amount:            p.Amount,
		Currency:          string(p.Currency),
		Market:            schema.Market(NormalizeMarket(p.Market)),
		Status:            schema.PaymentStatus(p.Status),
		PaymentMethod:     schema.PaymentMethodType(p.PaymentMethod),
		Provider:          p.Provider,
		AuthorizationCode: p.AuthorizationCode,
		RedirectURL:       p.RedirectURL,
		RequiresAction:    p.RequiresAction,
		Description:       p.Description,
		Metadata:          metadataMapToArray(p.Metadata),
		RefundedAmount:    p.RefundedAmount,
		RefundedAt:        p.RefundedAt,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
	}

	return sp
}

// MapPaymentsFromSchema converts multiple schema.Payment to domain.Payment
func MapPaymentsFromSchema(sps []*schema.Payment) []Payment {
	payments := make([]Payment, 0, len(sps))
	for _, sp := range sps {
		if p := MapPaymentFromSchema(sp); p != nil {
			payments = append(payments, *p)
		}
	}
	return payments
}

// ============================================================================
// Transaction Mappers
// ============================================================================

// MapTransactionFromSchema converts schema.Transaction to domain.Transaction
func MapTransactionFromSchema(st *schema.Transaction) *Transaction {
	if st == nil {
		return nil
	}

	return &Transaction{
		ID:           st.ID,
		PaymentID:    st.PaymentID,
		BookingID:    st.BookingID,
		BusinessID:   st.BusinessID,
		Type:         TransactionType(st.Type),
		Reference:    st.Reference,
		Amount:       st.Amount,
		Currency:     payment.Currency(st.Currency),
		Status:       TransactionStatus(st.Status),
		Provider:     st.Provider,
		ProviderTxID: st.ProviderTxID,
		Description:  st.Description,
		Metadata:     metadataArrayToMap(st.Metadata),
		ErrorMessage: st.ErrorMessage,
		ErrorCode:    st.ErrorCode,
		ProcessedAt:  st.ProcessedAt,
		CreatedAt:    st.CreatedAt,
		UpdatedAt:    st.UpdatedAt,
	}
}

// MapTransactionToSchema converts domain.Transaction to schema.Transaction
func MapTransactionToSchema(t *Transaction) *schema.Transaction {
	if t == nil {
		return nil
	}

	return &schema.Transaction{
		ID:           t.ID,
		PaymentID:    t.PaymentID,
		BookingID:    t.BookingID,
		BusinessID:   t.BusinessID,
		Type:         schema.TransactionType(t.Type),
		Reference:    t.Reference,
		Amount:       t.Amount,
		Currency:     string(t.Currency),
		Status:       schema.TransactionStatus(t.Status),
		Provider:     t.Provider,
		ProviderTxID: t.ProviderTxID,
		Description:  t.Description,
		Metadata:     metadataMapToArray(t.Metadata),
		ErrorMessage: t.ErrorMessage,
		ErrorCode:    t.ErrorCode,
		ProcessedAt:  t.ProcessedAt,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
}

// MapTransactionsFromSchema converts multiple schema.Transaction to domain.Transaction
func MapTransactionsFromSchema(sts []*schema.Transaction) []Transaction {
	transactions := make([]Transaction, 0, len(sts))
	for _, st := range sts {
		if t := MapTransactionFromSchema(st); t != nil {
			transactions = append(transactions, *t)
		}
	}
	return transactions
}

// ============================================================================
// PaymentMethod Mappers
// ============================================================================

// MapPaymentMethodFromSchema converts schema.PaymentMethod to domain.PaymentMethod
func MapPaymentMethodFromSchema(spm *schema.PaymentMethod) *PaymentMethod {
	if spm == nil {
		return nil
	}

	pm := &PaymentMethod{
		ID:                spm.ID,
		UserID:            spm.UserID,
		Type:              PaymentMethodType(spm.Type),
		Provider:          spm.Provider,
		AuthorizationCode: spm.AuthorizationCode,
		Currency:          payment.Currency(spm.Currency),
		Last4Digits:       spm.Last4Digits,
		CardType:          spm.CardType,
		Brand:             spm.Brand,
		ExpiryMonth:       spm.ExpiryMonth,
		ExpiryYear:        spm.ExpiryYear,
		BankName:          spm.BankName,
		IsDefault:         spm.IsDefault,
		IsActive:          spm.IsActive,
		CustomerCode:      spm.CustomerCode,
		CreatedAt:         spm.CreatedAt,
		UpdatedAt:         spm.UpdatedAt,
	}

	if spm.DeletedAt.Valid {
		pm.DeletedAt = &spm.DeletedAt.Time
	}

	return pm
}

// MapPaymentMethodToSchema converts domain.PaymentMethod to schema.PaymentMethod
func MapPaymentMethodToSchema(pm *PaymentMethod) *schema.PaymentMethod {
	if pm == nil {
		return nil
	}

	return &schema.PaymentMethod{
		ID:                pm.ID,
		UserID:            pm.UserID,
		Type:              schema.PaymentMethodType(pm.Type),
		Provider:          pm.Provider,
		AuthorizationCode: pm.AuthorizationCode,
		Currency:          string(pm.Currency),
		Last4Digits:       pm.Last4Digits,
		CardType:          pm.CardType,
		Brand:             pm.Brand,
		ExpiryMonth:       pm.ExpiryMonth,
		ExpiryYear:        pm.ExpiryYear,
		BankName:          pm.BankName,
		IsDefault:         pm.IsDefault,
		IsActive:          pm.IsActive,
		CustomerCode:      pm.CustomerCode,
		CreatedAt:         pm.CreatedAt,
		UpdatedAt:         pm.UpdatedAt,
	}
}

// MapPaymentMethodsFromSchema converts multiple schema.PaymentMethod to domain.PaymentMethod
func MapPaymentMethodsFromSchema(spms []*schema.PaymentMethod) []PaymentMethod {
	methods := make([]PaymentMethod, 0, len(spms))
	for _, spm := range spms {
		if pm := MapPaymentMethodFromSchema(spm); pm != nil {
			methods = append(methods, *pm)
		}
	}
	return methods
}

// ============================================================================
// PayoutDetail Mappers
// ============================================================================

// MapPayoutDetailFromSchema converts schema.PayoutDetail to domain.PayoutDetail
func MapPayoutDetailFromSchema(spd *schema.PayoutDetail) *PayoutDetail {
	if spd == nil {
		return nil
	}

	pd := &PayoutDetail{
		ID:            spd.ID,
		UserID:        spd.UserID,
		BusinessID:    spd.BusinessID,
		BankCode:      spd.BankCode,
		BankName:      spd.BankName,
		AccountNumber: spd.AccountNumber,
		AccountName:   spd.AccountName,
		Currency:      payment.Currency(spd.Currency),
		Market:        NormalizeMarket(Market(spd.Market)),
		Provider:      spd.Provider,
		RecipientCode: spd.RecipientCode,
		IsVerified:    spd.IsVerified,
		VerifiedAt:    spd.VerifiedAt,
		IsDefault:     spd.IsDefault,
		IsActive:      spd.IsActive,
		Metadata:      metadataArrayToMap(spd.Metadata),
		CreatedAt:     spd.CreatedAt,
		UpdatedAt:     spd.UpdatedAt,
	}

	if spd.DeletedAt.Valid {
		pd.DeletedAt = &spd.DeletedAt.Time
	}

	return pd
}

// MapPayoutDetailToSchema converts domain.PayoutDetail to schema.PayoutDetail
func MapPayoutDetailToSchema(pd *PayoutDetail) *schema.PayoutDetail {
	if pd == nil {
		return nil
	}

	return &schema.PayoutDetail{
		ID:            pd.ID,
		UserID:        pd.UserID,
		BusinessID:    pd.BusinessID,
		BankCode:      pd.BankCode,
		BankName:      pd.BankName,
		AccountNumber: pd.AccountNumber,
		AccountName:   pd.AccountName,
		Currency:      string(pd.Currency),
		Market:        schema.Market(NormalizeMarket(pd.Market)),
		Provider:      pd.Provider,
		RecipientCode: pd.RecipientCode,
		IsVerified:    pd.IsVerified,
		VerifiedAt:    pd.VerifiedAt,
		IsDefault:     pd.IsDefault,
		IsActive:      pd.IsActive,
		Metadata:      metadataMapToArray(pd.Metadata),
		CreatedAt:     pd.CreatedAt,
		UpdatedAt:     pd.UpdatedAt,
	}
}

// MapPayoutDetailsFromSchema converts multiple schema.PayoutDetail to domain.PayoutDetail
func MapPayoutDetailsFromSchema(spds []*schema.PayoutDetail) []PayoutDetail {
	details := make([]PayoutDetail, 0, len(spds))
	for _, spd := range spds {
		if pd := MapPayoutDetailFromSchema(spd); pd != nil {
			details = append(details, *pd)
		}
	}
	return details
}

// ============================================================================
// Helper Functions
// ============================================================================

// metadataArrayToMap converts PostgreSQL string array to map
func metadataArrayToMap(arr []string) map[string]string {
	metadata := make(map[string]string)
	for _, item := range arr {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) == 2 {
			metadata[parts[0]] = parts[1]
		}
	}
	return metadata
}

// metadataMapToArray converts map to PostgreSQL string array
func metadataMapToArray(m map[string]string) []string {
	arr := make([]string, 0, len(m))
	for key, value := range m {
		arr = append(arr, key+"="+value)
	}
	return arr
}

// NewPaymentID generates a new UUID for payment
func NewPaymentID() uuid.UUID {
	return uuid.New()
}

// NewTransactionID generates a new UUID for transaction
func NewTransactionID() uuid.UUID {
	return uuid.New()
}
