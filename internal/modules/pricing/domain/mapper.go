package domain

import (
	"hauslet/internal/modules/pricing/repository/schema"
)

// --- PricingRule Mappers ---

func MapRuleFromSchemaToEntity(s *schema.PricingRule) *PricingRule {
	if s == nil {
		return nil
	}

	d := &PricingRule{
		ID:              s.ID,
		ListingID:       s.ListingID,
		Name:            s.Name,
		Description:     s.Description,
		RuleType:        RuleType(s.RuleType),
		Active:          s.Active,
		Priority:        s.Priority,
		StartDate:       s.StartDate,
		EndDate:         s.EndDate,
		DaysOfWeek:      s.DaysOfWeek,
		ModifierType:    ModifierType(s.ModifierType),
		ModifierValue:   s.ModifierValue,
		MinNights:       s.MinNights,
		MaxNights:       s.MaxNights,
		MinLeadTimeDays: s.MinLeadTimeDays,
		MaxLeadTimeDays: s.MaxLeadTimeDays,
		CreatedBy:       s.CreatedBy,
		UpdatedBy:       s.UpdatedBy,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
	}

	if s.DeletedAt.Valid {
		deletedAt := s.DeletedAt.Time
		d.DeletedAt = &deletedAt
	}

	return d
}

func MapRuleFromEntityToSchema(d *PricingRule) *schema.PricingRule {
	if d == nil {
		return nil
	}

	return &schema.PricingRule{
		ID:              d.ID,
		ListingID:       d.ListingID,
		Name:            d.Name,
		Description:     d.Description,
		RuleType:        schema.RuleType(d.RuleType),
		Active:          d.Active,
		Priority:        d.Priority,
		StartDate:       d.StartDate,
		EndDate:         d.EndDate,
		DaysOfWeek:      d.DaysOfWeek,
		ModifierType:    schema.ModifierType(d.ModifierType),
		ModifierValue:   d.ModifierValue,
		MinNights:       d.MinNights,
		MaxNights:       d.MaxNights,
		MinLeadTimeDays: d.MinLeadTimeDays,
		MaxLeadTimeDays: d.MaxLeadTimeDays,
		CreatedBy:       d.CreatedBy,
		UpdatedBy:       d.UpdatedBy,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
	}
}

// --- MultiPropertyDiscount Mappers ---

func MapDiscountFromSchemaToEntity(s *schema.MultiPropertyDiscount) *MultiPropertyDiscount {
	if s == nil {
		return nil
	}

	d := &MultiPropertyDiscount{
		ID:              s.ID,
		OwnerID:         s.OwnerID,
		Name:            s.Name,
		Description:     s.Description,
		ListingIDs:      s.ListingIDs,
		MinProperties:   s.MinProperties,
		DiscountPercent: s.DiscountPercent,
		ValidFrom:       s.ValidFrom,
		ValidUntil:      s.ValidUntil,
		Active:          s.Active,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
	}

	if s.DeletedAt.Valid {
		deletedAt := s.DeletedAt.Time
		d.DeletedAt = &deletedAt
	}

	return d
}

func MapDiscountFromEntityToSchema(d *MultiPropertyDiscount) *schema.MultiPropertyDiscount {
	if d == nil {
		return nil
	}

	return &schema.MultiPropertyDiscount{
		ID:              d.ID,
		OwnerID:         d.OwnerID,
		Name:            d.Name,
		Description:     d.Description,
		ListingIDs:      d.ListingIDs,
		MinProperties:   d.MinProperties,
		DiscountPercent: d.DiscountPercent,
		ValidFrom:       d.ValidFrom,
		ValidUntil:      d.ValidUntil,
		Active:          d.Active,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
	}
}

// --- Batch Mappers ---

func MapRulesFromSchema(rules []*schema.PricingRule) []*PricingRule {
	if rules == nil {
		return nil
	}

	result := make([]*PricingRule, len(rules))
	for i, rule := range rules {
		result[i] = MapRuleFromSchemaToEntity(rule)
	}
	return result
}

func MapRulesToSchema(rules []*PricingRule) []*schema.PricingRule {
	if rules == nil {
		return nil
	}

	result := make([]*schema.PricingRule, len(rules))
	for i, rule := range rules {
		result[i] = MapRuleFromEntityToSchema(rule)
	}
	return result
}
