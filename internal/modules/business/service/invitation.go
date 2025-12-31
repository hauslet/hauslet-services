package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"hauslet/internal/modules/business/domain"
	"hauslet/internal/modules/business/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	invitationTokenLength = 32
	invitationExpiryDays  = 7
)

// InviteUser creates an invitation for a user to join a business
func (s *BusinessServiceImpl) InviteUser(ctx context.Context, businessID uuid.UUID, email string, role domain.MemberRole, invitedBy uuid.UUID, customPermissions *domain.MemberPermissions) (*domain.BusinessInvitation, error) {
	// Check if inviter has permission
	hasPermission, err := s.HasPermission(ctx, invitedBy, businessID, "CanManageMembers")
	if err != nil {
		return nil, err
	}
	if !hasPermission {
		return nil, domain.ErrInsufficientPermissions
	}

	// Check if there's already a pending invitation
	hasPending, err := s.repo.HasPendingInvitation(ctx, businessID, email)
	if err != nil {
		return nil, err
	}
	if hasPending {
		return nil, domain.ErrUserAlreadyInvited
	}

	// Generate secure token
	token, err := generateInvitationToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate invitation token: %w", err)
	}

	// Determine permissions
	permissions := domain.GetDefaultPermissions(role)
	if customPermissions != nil {
		permissions = *customPermissions
	}

	// Create invitation
	now := time.Now()
	invitation := &domain.BusinessInvitation{
		ID:          uuid.New(),
		BusinessID:  businessID,
		Email:       email,
		Role:        role,
		Permissions: permissions,
		InvitedBy:   invitedBy,
		Token:       token,
		Status:      domain.InvitationPending,
		ExpiresAt:   now.Add(invitationExpiryDays * 24 * time.Hour),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	schemaInvitation, err := domain.MapBusinessInvitationToSchema(invitation)
	if err != nil {
		return nil, fmt.Errorf("failed to map invitation: %w", err)
	}

	if err := s.repo.CreateInvitation(ctx, schemaInvitation); err != nil {
		s.log.Error("Failed to create invitation", "error", err)
		return nil, fmt.Errorf("failed to create invitation: %w", err)
	}

	s.log.Info(" Invitation created for %s to join business %s", email, businessID)
	if s.notifier != nil {
		business, err := s.GetBusiness(ctx, businessID)
		if err != nil {
			s.log.Warn("Invitation created but failed to load business", "business_id", businessID, "error", err)
		} else {
			inviterName := s.getProfileName(ctx, invitedBy)
			if inviterName == "" {
				inviterName = business.DisplayName
			}
			if inviterName == "" {
				inviterName = business.Name
			}
			if err := s.notifier.SendInvitationEmail(ctx, invitation, business, inviterName); err != nil {
				s.log.Warn("Failed to send invitation email", "business_id", businessID, "error", err)
			}
		}
	}
	return invitation, nil
}

// AcceptInvitation accepts an invitation and adds the user as a member
func (s *BusinessServiceImpl) AcceptInvitation(ctx context.Context, token string, userID uuid.UUID) (*domain.BusinessMember, error) {
	// Get invitation by token
	schemaInvitation, err := s.repo.GetInvitationByToken(ctx, token)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrInvitationNotFound
		}
		return nil, err
	}

	invitation := domain.MapBusinessInvitationFromSchema(schemaInvitation)

	// Check if invitation can be accepted
	if err := invitation.Accept(); err != nil {
		return nil, err
	}

	// Check if user is already a member
	isMember, err := s.repo.IsMember(ctx, invitation.BusinessID, userID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if isMember {
		return nil, domain.ErrMemberAlreadyExists
	}

	var member *domain.BusinessMember

	// Create member and update invitation in a transaction
	err = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		// Create member
		now := time.Now()
		member = &domain.BusinessMember{
			ID:          uuid.New(),
			BusinessID:  invitation.BusinessID,
			UserID:      userID,
			Role:        invitation.Role,
			Permissions: invitation.Permissions,
			IsActive:    true,
			InvitedBy:   &invitation.InvitedBy,
			JoinedAt:    now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		schemaMember, err := domain.MapBusinessMemberToSchema(member)
		if err != nil {
			return fmt.Errorf("failed to map member: %w", err)
		}

		if err := s.repo.AddMember(ctx, schemaMember); err != nil {
			return fmt.Errorf("failed to add member: %w", err)
		}

		// Update invitation status
		schemaInvitation.Status = schema.InvitationAccepted
		schemaInvitation.AcceptedAt = &now
		schemaInvitation.UpdatedAt = now

		if err := s.repo.UpdateInvitation(ctx, schemaInvitation); err != nil {
			return fmt.Errorf("failed to update invitation: %w", err)
		}

		return nil
	})

	if err != nil {
		s.log.Error("Failed to accept invitation", "error", err)
		return nil, err
	}

	if s.notifier != nil {
		business, err := s.GetBusiness(ctx, invitation.BusinessID)
		if err != nil {
			s.log.Warn("Invitation accepted but failed to load business", "business_id", invitation.BusinessID, "error", err)
		} else {
			acceptedName := s.getProfileName(ctx, userID)
			if acceptedName == "" {
				acceptedName = invitation.Email
			}
			ownerEmails := make([]string, 0, 2)
			if business.Email != "" {
				ownerEmails = append(ownerEmails, business.Email)
			}
			if business.BillingEmail != nil && *business.BillingEmail != "" {
				ownerEmails = append(ownerEmails, *business.BillingEmail)
			}

			if len(ownerEmails) > 0 {
				if err := s.notifier.SendInvitationAcceptedEmail(ctx, business, acceptedName, invitation.Email, invitation.Role, ownerEmails); err != nil {
					s.log.Warn("Failed to send invitation accepted email", "business_id", business.ID, "error", err)
				}
			}

			if err := s.notifier.SendMemberAddedEmail(ctx, member, business, acceptedName, invitation.Email, "Invitation accepted"); err != nil {
				s.log.Warn("Failed to send member added email", "business_id", business.ID, "error", err)
			}
		}
	}

	s.log.Info(" User accepted invitation to business", "user_id", userID, "business_id", invitation.BusinessID)
	return member, nil
}

// DeclineInvitation declines an invitation
func (s *BusinessServiceImpl) DeclineInvitation(ctx context.Context, token string, userID uuid.UUID) error {
	// Get invitation by token
	schemaInvitation, err := s.repo.GetInvitationByToken(ctx, token)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return domain.ErrInvitationNotFound
		}
		return err
	}

	invitation := domain.MapBusinessInvitationFromSchema(schemaInvitation)

	// Decline invitation
	if err := invitation.Decline(); err != nil {
		return err
	}

	// Update invitation status
	schemaInvitation.Status = schema.InvitationDeclined
	schemaInvitation.UpdatedAt = time.Now()

	if err := s.repo.UpdateInvitation(ctx, schemaInvitation); err != nil {
		s.log.Error("Failed to decline invitation", "error", err)
		return fmt.Errorf("failed to decline invitation: %w", err)
	}

	if s.notifier != nil {
		business, err := s.GetBusiness(ctx, invitation.BusinessID)
		if err != nil {
			s.log.Warn("Invitation declined but failed to load business", "business_id", invitation.BusinessID, "error", err)
		} else {
			declinerName := s.getProfileName(ctx, userID)
			if declinerName == "" {
				declinerName = invitation.Email
			}
			ownerEmails := make([]string, 0, 2)
			if business.Email != "" {
				ownerEmails = append(ownerEmails, business.Email)
			}
			if business.BillingEmail != nil && *business.BillingEmail != "" {
				ownerEmails = append(ownerEmails, *business.BillingEmail)
			}
			if len(ownerEmails) > 0 {
				if err := s.notifier.SendInvitationDeclinedEmail(ctx, business, declinerName, invitation.Email, ownerEmails); err != nil {
					s.log.Warn("Failed to send invitation declined email", "business_id", business.ID, "error", err)
				}
			}
		}
	}

	s.log.Info(" User declined invitation to business", "user_id", userID, "business_id", invitation.BusinessID)
	return nil
}

// RevokeInvitation revokes an invitation
func (s *BusinessServiceImpl) RevokeInvitation(ctx context.Context, invitationID, revokedBy uuid.UUID) error {
	// Get invitation
	schemaInvitation, err := s.repo.GetInvitationByID(ctx, invitationID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return domain.ErrInvitationNotFound
		}
		return err
	}

	// Check if revoker has permission
	hasPermission, err := s.HasPermission(ctx, revokedBy, schemaInvitation.BusinessID, "CanManageMembers")
	if err != nil {
		return err
	}
	if !hasPermission {
		return domain.ErrInsufficientPermissions
	}

	invitation := domain.MapBusinessInvitationFromSchema(schemaInvitation)

	// Revoke invitation
	if err := invitation.Revoke(); err != nil {
		return err
	}

	// Update invitation status
	schemaInvitation.Status = schema.InvitationRevoked
	schemaInvitation.UpdatedAt = time.Now()

	if err := s.repo.UpdateInvitation(ctx, schemaInvitation); err != nil {
		s.log.Error("Failed to revoke invitation", "error", err)
		return fmt.Errorf("failed to revoke invitation: %w", err)
	}

	s.log.Info(" Invitation revoked", "invitation_id", invitationID, "revoked_by", revokedBy)
	return nil
}

// GetBusinessInvitations retrieves all invitations for a business
func (s *BusinessServiceImpl) GetBusinessInvitations(ctx context.Context, businessID uuid.UUID) ([]domain.BusinessInvitation, error) {
	schemaInvitations, err := s.repo.ListBusinessInvitations(ctx, businessID)
	if err != nil {
		return nil, fmt.Errorf("failed to get business invitations: %w", err)
	}

	return domain.MapBusinessInvitationsFromSchema(schemaInvitations), nil
}

// GetUserInvitations retrieves all invitations for a user email
func (s *BusinessServiceImpl) GetUserInvitations(ctx context.Context, email string) ([]domain.BusinessInvitation, error) {
	schemaInvitations, err := s.repo.ListUserInvitations(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user invitations: %w", err)
	}

	return domain.MapBusinessInvitationsFromSchema(schemaInvitations), nil
}

// generateInvitationToken generates a secure random token for invitations
func generateInvitationToken() (string, error) {
	bytes := make([]byte, invitationTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
