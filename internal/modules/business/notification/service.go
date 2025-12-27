package notification

import (
	"context"
	"fmt"
	"time"

	"hauslet/internal/modules/business/domain"
	businesstemplates "hauslet/internal/modules/business/templates"
	"hauslet/internal/platform/email"
	"hauslet/internal/platform/queue"
	emailJob "hauslet/internal/queue/jobs/emails"

	"github.com/go-pkgz/lgr"
)

// NotificationService handles business notifications
type NotificationService struct {
	mailClient   *email.Client
	queueClient  *queue.Client
	queueSubject string
	baseURL      string
	log          *lgr.Logger
}

// NewNotificationService creates a new notification service
func NewNotificationService(
	mailClient *email.Client,
	queueClient *queue.Client,
	queueSubject string,
	baseURL string,
	log *lgr.Logger,
) *NotificationService {
	return &NotificationService{
		mailClient:   mailClient,
		queueClient:  queueClient,
		queueSubject: queueSubject,
		baseURL:      baseURL,
		log:          log,
	}
}

// sendEmailAsync runs the provided send function in a goroutine and logs errors
func (s *NotificationService) sendEmailAsync(label string, fn func() error) {
	go func() {
		if err := fn(); err != nil && s.log != nil {
			s.log.Logf("[WARN] %s: %v", label, err)
		}
	}()
}

// publishEmailJob tries to enqueue the email job and returns an error on failure.
// It uses a short-lived background context so request cancellation does not
// prevent publishing.
func (s *NotificationService) publishEmailJob(job emailJob.EmailJob) error {
	if s.queueClient == nil || s.queueSubject == "" {
		return fmt.Errorf("queue not configured")
	}

	pubCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.queueClient.Publish(pubCtx, s.queueSubject, job); err != nil {
		if s.log != nil {
			s.log.Logf("[WARN] failed to publish business email job to %s: %v", s.queueSubject, err)
		}
		return err
	}

	return nil
}

// SendInvitationEmail sends an invitation email to the invitee
func (s *NotificationService) SendInvitationEmail(
	ctx context.Context,
	invitation *domain.BusinessInvitation,
	business *domain.Business,
	inviterName string,
) error {
	subject := fmt.Sprintf("You've been invited to join %s on Hauslet", business.Name)
	preview := fmt.Sprintf("%s has invited you to join %s", inviterName, business.Name)

	acceptURL := fmt.Sprintf("%s/invitations/accept?token=%s", s.baseURL, invitation.Token)
	declineURL := fmt.Sprintf("%s/invitations/decline?token=%s", s.baseURL, invitation.Token)

	emailData := map[string]any{
		"BusinessName":    business.Name,
		"BusinessLogoURL": business.LogoURL,
		"InviterName":     inviterName,
		"Role":            invitation.Role,
		"AcceptURL":       acceptURL,
		"DeclineURL":      declineURL,
		"ExpiresAt":       invitation.ExpiresAt.Format("January 2, 2006 at 3:04 PM"),

		// Required for the Layout
		"Subject": subject,
		"Preview": preview,
		"Year":    time.Now().Year(),
	}

	htmlBody, err := s.mailClient.RenderTemplate(
		businesstemplates.FS,
		"invitation.html",
		emailData,
	)
	if err != nil {
		return err
	}

	s.sendEmailAsync("send invitation email", func() error {
		job := emailJob.EmailJob{
			To:      invitation.Email,
			Subject: subject,
			HTML:    htmlBody,
		}
		if err := s.publishEmailJob(job); err == nil {
			return nil
		} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
			return err
		}
		return s.mailClient.SendHTML(ctx, invitation.Email, subject, htmlBody)
	})

	return nil
}

// SendMemberAddedEmail sends a notification to a newly added member
func (s *NotificationService) SendMemberAddedEmail(
	ctx context.Context,
	member *domain.BusinessMember,
	business *domain.Business,
	memberName string,
	memberEmail string,
	addedByName string,
) error {
	subject := fmt.Sprintf("You've been added to %s", business.Name)
	preview := fmt.Sprintf("You're now a %s in %s", member.Role, business.Name)

	dashboardURL := fmt.Sprintf("%s/businesses/%s", s.baseURL, business.ID)

	emailData := map[string]any{
		"BusinessName":    business.Name,
		"BusinessLogoURL": business.LogoURL,
		"MemberName":      memberName,
		"Role":            member.Role,
		"AddedBy":         addedByName,
		"DashboardURL":    dashboardURL,

		// Required for the Layout
		"Subject": subject,
		"Preview": preview,
		"Year":    time.Now().Year(),
	}

	htmlBody, err := s.mailClient.RenderTemplate(
		businesstemplates.FS,
		"member_added.html",
		emailData,
	)
	if err != nil {
		return err
	}

	s.sendEmailAsync("send member added email", func() error {
		job := emailJob.EmailJob{
			To:      memberEmail,
			Subject: subject,
			HTML:    htmlBody,
		}
		if err := s.publishEmailJob(job); err == nil {
			return nil
		} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
			return err
		}
		return s.mailClient.SendHTML(ctx, memberEmail, subject, htmlBody)
	})

	return nil
}

// SendMemberRemovedEmail sends a notification to a removed member
func (s *NotificationService) SendMemberRemovedEmail(
	ctx context.Context,
	business *domain.Business,
	memberName string,
	memberEmail string,
	removedByName string,
) error {
	subject := fmt.Sprintf("You've been removed from %s", business.Name)
	preview := "Your access to the business has been removed"

	emailData := map[string]any{
		"BusinessName":    business.Name,
		"BusinessLogoURL": business.LogoURL,
		"MemberName":      memberName,
		"RemovedBy":       removedByName,

		// Required for the Layout
		"Subject": subject,
		"Preview": preview,
		"Year":    time.Now().Year(),
	}

	htmlBody, err := s.mailClient.RenderTemplate(
		businesstemplates.FS,
		"member_removed.html",
		emailData,
	)
	if err != nil {
		return err
	}

	s.sendEmailAsync("send member removed email", func() error {
		job := emailJob.EmailJob{
			To:      memberEmail,
			Subject: subject,
			HTML:    htmlBody,
		}
		if err := s.publishEmailJob(job); err == nil {
			return nil
		} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
			return err
		}
		return s.mailClient.SendHTML(ctx, memberEmail, subject, htmlBody)
	})

	return nil
}

// SendRoleChangedEmail sends a notification when a member's role changes
func (s *NotificationService) SendRoleChangedEmail(
	ctx context.Context,
	business *domain.Business,
	memberName string,
	memberEmail string,
	oldRole domain.MemberRole,
	newRole domain.MemberRole,
	changedByName string,
) error {
	subject := fmt.Sprintf("Your role in %s has been updated", business.Name)
	preview := fmt.Sprintf("Your role has been changed from %s to %s", oldRole, newRole)

	dashboardURL := fmt.Sprintf("%s/businesses/%s", s.baseURL, business.ID)

	emailData := map[string]any{
		"BusinessName":    business.Name,
		"BusinessLogoURL": business.LogoURL,
		"MemberName":      memberName,
		"OldRole":         oldRole,
		"NewRole":         newRole,
		"ChangedBy":       changedByName,
		"DashboardURL":    dashboardURL,

		// Required for the Layout
		"Subject": subject,
		"Preview": preview,
		"Year":    time.Now().Year(),
	}

	htmlBody, err := s.mailClient.RenderTemplate(
		businesstemplates.FS,
		"role_changed.html",
		emailData,
	)
	if err != nil {
		return err
	}

	s.sendEmailAsync("send role changed email", func() error {
		job := emailJob.EmailJob{
			To:      memberEmail,
			Subject: subject,
			HTML:    htmlBody,
		}
		if err := s.publishEmailJob(job); err == nil {
			return nil
		} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
			return err
		}
		return s.mailClient.SendHTML(ctx, memberEmail, subject, htmlBody)
	})

	return nil
}

// SendInvitationAcceptedEmail notifies business owners when an invitation is accepted
func (s *NotificationService) SendInvitationAcceptedEmail(
	ctx context.Context,
	business *domain.Business,
	acceptedByName string,
	acceptedEmail string,
	role domain.MemberRole,
	ownerEmails []string,
) error {
	subject := fmt.Sprintf("%s accepted your invitation to %s", acceptedByName, business.Name)
	preview := fmt.Sprintf("%s is now a %s", acceptedByName, role)

	membersURL := fmt.Sprintf("%s/businesses/%s/members", s.baseURL, business.ID)

	emailData := map[string]any{
		"BusinessName":    business.Name,
		"BusinessLogoURL": business.LogoURL,
		"AcceptedBy":      acceptedByName,
		"AcceptedEmail":   acceptedEmail,
		"Role":            role,
		"MembersURL":      membersURL,

		// Required for the Layout
		"Subject": subject,
		"Preview": preview,
		"Year":    time.Now().Year(),
	}

	htmlBody, err := s.mailClient.RenderTemplate(
		businesstemplates.FS,
		"invitation_accepted.html",
		emailData,
	)
	if err != nil {
		return err
	}

	// Send to all owners
	for _, ownerEmail := range ownerEmails {
		email := ownerEmail
		s.sendEmailAsync("send invitation accepted email", func() error {
			job := emailJob.EmailJob{
				To:      email,
				Subject: subject,
				HTML:    htmlBody,
			}
			if err := s.publishEmailJob(job); err == nil {
				return nil
			} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
				return err
			}
			return s.mailClient.SendHTML(ctx, email, subject, htmlBody)
		})
	}

	return nil
}

// SendInvitationDeclinedEmail notifies business owners when an invitation is declined
func (s *NotificationService) SendInvitationDeclinedEmail(
	ctx context.Context,
	business *domain.Business,
	declinedByName string,
	declinedEmail string,
	ownerEmails []string,
) error {
	subject := fmt.Sprintf("%s declined your invitation to %s", declinedByName, business.Name)
	preview := fmt.Sprintf("%s declined to join %s", declinedByName, business.Name)

	invitationsURL := fmt.Sprintf("%s/businesses/%s/invitations", s.baseURL, business.ID)

	emailData := map[string]any{
		"BusinessName":    business.Name,
		"BusinessLogoURL": business.LogoURL,
		"DeclinedBy":      declinedByName,
		"DeclinedEmail":   declinedEmail,
		"InvitationsURL":  invitationsURL,

		// Required for the Layout
		"Subject": subject,
		"Preview": preview,
		"Year":    time.Now().Year(),
	}

	htmlBody, err := s.mailClient.RenderTemplate(
		businesstemplates.FS,
		"invitation_declined.html",
		emailData,
	)
	if err != nil {
		return err
	}

	// Send to all owners
	for _, ownerEmail := range ownerEmails {
		email := ownerEmail
		s.sendEmailAsync("send invitation declined email", func() error {
			job := emailJob.EmailJob{
				To:      email,
				Subject: subject,
				HTML:    htmlBody,
			}
			if err := s.publishEmailJob(job); err == nil {
				return nil
			} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
				return err
			}
			return s.mailClient.SendHTML(ctx, email, subject, htmlBody)
		})
	}

	return nil
}

// SendBusinessCreatedEmail sends a welcome email when a business is created
func (s *NotificationService) SendBusinessCreatedEmail(
	ctx context.Context,
	business *domain.Business,
	creatorName string,
	creatorEmail string,
) error {
	subject := fmt.Sprintf("Welcome to %s - Your business is ready!", business.Name)
	preview := "Your business has been successfully created on Hauslet"

	dashboardURL := fmt.Sprintf("%s/businesses/%s", s.baseURL, business.ID)

	emailData := map[string]any{
		"BusinessName":    business.Name,
		"BusinessLogoURL": business.LogoURL,
		"CreatorName":     creatorName,
		"DashboardURL":    dashboardURL,

		// Required for the Layout
		"Subject": subject,
		"Preview": preview,
		"Year":    time.Now().Year(),
	}

	htmlBody, err := s.mailClient.RenderTemplate(
		businesstemplates.FS,
		"business_created.html",
		emailData,
	)
	if err != nil {
		return err
	}

	s.sendEmailAsync("send business created email", func() error {
		job := emailJob.EmailJob{
			To:      creatorEmail,
			Subject: subject,
			HTML:    htmlBody,
		}
		if err := s.publishEmailJob(job); err == nil {
			return nil
		} else if s.queueClient != nil && !s.queueClient.AllowFallback() {
			return err
		}
		return s.mailClient.SendHTML(ctx, creatorEmail, subject, htmlBody)
	})

	return nil
}
