package hooks

import (
	"context"
	"hauslet/internal/modules/auth/domain"
	"hauslet/internal/modules/auth/service"
)

// FinanceAuthAdapter adapts auth service to provide admin emails for finance notifications
type FinanceAuthAdapter struct {
	authSvc service.AuthService
	roles   []string
}

// NewFinanceAuthAdapter creates a new auth adapter for finance
func NewFinanceAuthAdapter(authSvc service.AuthService, roles []string) *FinanceAuthAdapter {
	return &FinanceAuthAdapter{
		authSvc: authSvc,
		roles:   roles,
	}
}

// GetAdminEmails retrieves emails of all active users with configured admin roles
func (a *FinanceAuthAdapter) GetAdminEmails(ctx context.Context) ([]string, error) {
	emails := make([]string, 0)
	seen := make(map[string]bool)

	for _, role := range a.roles {
		users, err := a.authSvc.GetUsersByRole(ctx, domain.UserRole(role))
		if err != nil {
			// Log but continue with other roles
			continue
		}

		for _, user := range users {
			if !user.IsActive {
				continue
			}
			if !seen[user.PrimaryEmail] {
				emails = append(emails, user.PrimaryEmail)
				seen[user.PrimaryEmail] = true
			}
		}
	}

	return emails, nil
}
