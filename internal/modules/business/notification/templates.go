package notification

import (
	"fmt"
	"strings"
	"time"
)

// TemplateBuilder builds email templates for business notifications
type TemplateBuilder struct {
	baseURL string
}

// NewTemplateBuilder creates a new template builder
func NewTemplateBuilder(baseURL string) *TemplateBuilder {
	return &TemplateBuilder{
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

// BuildInvitationEmail builds an invitation email template
func (tb *TemplateBuilder) BuildInvitationEmail(data InvitationEmailData) *EmailTemplate {
	acceptURL := fmt.Sprintf("%s/invitations/accept?token=%s", tb.baseURL, data.Token)
	declineURL := fmt.Sprintf("%s/invitations/decline?token=%s", tb.baseURL, data.Token)

	expiresIn := time.Until(data.ExpiresAt).Hours() / 24

	subject := fmt.Sprintf("You've been invited to join %s on Hauslet", data.BusinessName)

	textBody := fmt.Sprintf(`Hi there,

%s has invited you to join %s as a %s on Hauslet.

Accept this invitation by clicking the link below:
%s

Or decline by clicking here:
%s

This invitation will expire in %.0f days (%s).

If you didn't expect this invitation, you can safely ignore this email.

Best regards,
The Hauslet Team
`, data.InviterName, data.BusinessName, data.Role, acceptURL, declineURL, expiresIn, data.ExpiresAt.Format("Jan 02, 2006"))

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Business Invitation</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f8f9fa; padding: 20px; border-radius: 10px;">
        %s
        <h2 style="color: #2c3e50; margin-top: 0;">You've been invited to join %s</h2>

        <p>Hi there,</p>

        <p><strong>%s</strong> has invited you to join <strong>%s</strong> as a <strong>%s</strong> on Hauslet.</p>

        <div style="margin: 30px 0;">
            <a href="%s" style="background-color: #3498db; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block; margin-right: 10px;">Accept Invitation</a>
            <a href="%s" style="background-color: #95a5a6; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Decline</a>
        </div>

        <p style="color: #7f8c8d; font-size: 14px;">This invitation will expire in <strong>%.0f days</strong> (%s).</p>

        <hr style="border: none; border-top: 1px solid #e0e0e0; margin: 30px 0;">

        <p style="color: #7f8c8d; font-size: 12px;">If you didn't expect this invitation, you can safely ignore this email.</p>

        <p style="color: #7f8c8d; font-size: 12px;">Best regards,<br>The Hauslet Team</p>
    </div>
</body>
</html>`, tb.buildLogoHTML(data.BusinessLogoURL), data.BusinessName, data.InviterName, data.BusinessName, data.Role, acceptURL, declineURL, expiresIn, data.ExpiresAt.Format("Jan 02, 2006"))

	return &EmailTemplate{
		Subject:  subject,
		TextBody: textBody,
		HTMLBody: htmlBody,
		Variables: map[string]interface{}{
			"business_name": data.BusinessName,
			"inviter_name":  data.InviterName,
			"role":          data.Role,
			"accept_url":    acceptURL,
			"decline_url":   declineURL,
			"expires_at":    data.ExpiresAt,
		},
	}
}

// BuildMemberAddedEmail builds a member added notification email
func (tb *TemplateBuilder) BuildMemberAddedEmail(data MemberAddedEmailData) *EmailTemplate {
	subject := fmt.Sprintf("You've been added to %s", data.BusinessName)

	textBody := fmt.Sprintf(`Hi %s,

Great news! You've been added to %s as a %s by %s.

You can now access the business dashboard and start managing properties and listings.

Visit your dashboard: %s/businesses/%s

Best regards,
The Hauslet Team
`, data.MemberName, data.BusinessName, data.Role, data.AddedBy, tb.baseURL, data.BusinessID)

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Added to Business</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f8f9fa; padding: 20px; border-radius: 10px;">
        %s
        <h2 style="color: #27ae60; margin-top: 0;">Welcome to %s!</h2>

        <p>Hi %s,</p>

        <p>Great news! You've been added to <strong>%s</strong> as a <strong>%s</strong> by <strong>%s</strong>.</p>

        <p>You can now access the business dashboard and start managing properties and listings.</p>

        <div style="margin: 30px 0;">
            <a href="%s/businesses/%s" style="background-color: #27ae60; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Go to Dashboard</a>
        </div>

        <p style="color: #7f8c8d; font-size: 12px;">Best regards,<br>The Hauslet Team</p>
    </div>
</body>
</html>`, tb.buildLogoHTML(data.BusinessLogoURL), data.BusinessName, data.MemberName, data.BusinessName, data.Role, data.AddedBy, tb.baseURL, data.BusinessID)

	return &EmailTemplate{
		Subject:  subject,
		TextBody: textBody,
		HTMLBody: htmlBody,
		Variables: map[string]interface{}{
			"business_name": data.BusinessName,
			"member_name":   data.MemberName,
			"role":          data.Role,
			"added_by":      data.AddedBy,
		},
	}
}

// BuildMemberRemovedEmail builds a member removed notification email
func (tb *TemplateBuilder) BuildMemberRemovedEmail(data MemberRemovedEmailData) *EmailTemplate {
	subject := fmt.Sprintf("You've been removed from %s", data.BusinessName)

	textBody := fmt.Sprintf(`Hi %s,

This is to inform you that you've been removed from %s by %s.

You no longer have access to the business dashboard and its resources.

If you believe this was done in error, please contact the business owner.

Best regards,
The Hauslet Team
`, data.MemberName, data.BusinessName, data.RemovedBy)

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Removed from Business</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f8f9fa; padding: 20px; border-radius: 10px;">
        %s
        <h2 style="color: #e74c3c; margin-top: 0;">Access Removed</h2>

        <p>Hi %s,</p>

        <p>This is to inform you that you've been removed from <strong>%s</strong> by <strong>%s</strong>.</p>

        <p>You no longer have access to the business dashboard and its resources.</p>

        <p style="color: #7f8c8d; font-size: 14px;">If you believe this was done in error, please contact the business owner.</p>

        <p style="color: #7f8c8d; font-size: 12px;">Best regards,<br>The Hauslet Team</p>
    </div>
</body>
</html>`, tb.buildLogoHTML(data.BusinessLogoURL), data.MemberName, data.BusinessName, data.RemovedBy)

	return &EmailTemplate{
		Subject:  subject,
		TextBody: textBody,
		HTMLBody: htmlBody,
		Variables: map[string]interface{}{
			"business_name": data.BusinessName,
			"member_name":   data.MemberName,
			"removed_by":    data.RemovedBy,
		},
	}
}

// BuildRoleChangedEmail builds a role changed notification email
func (tb *TemplateBuilder) BuildRoleChangedEmail(data RoleChangedEmailData) *EmailTemplate {
	subject := fmt.Sprintf("Your role in %s has been updated", data.BusinessName)

	textBody := fmt.Sprintf(`Hi %s,

Your role in %s has been updated by %s.

Old Role: %s
New Role: %s

Your new role grants you different permissions within the business. Please review the updated permissions in your dashboard.

Visit your dashboard: %s/businesses/%s

Best regards,
The Hauslet Team
`, data.MemberName, data.BusinessName, data.ChangedBy, data.OldRole, data.NewRole, tb.baseURL, data.BusinessID)

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Role Updated</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f8f9fa; padding: 20px; border-radius: 10px;">
        %s
        <h2 style="color: #3498db; margin-top: 0;">Your Role Has Been Updated</h2>

        <p>Hi %s,</p>

        <p>Your role in <strong>%s</strong> has been updated by <strong>%s</strong>.</p>

        <div style="background-color: white; padding: 15px; border-radius: 5px; margin: 20px 0;">
            <p style="margin: 5px 0;"><strong>Old Role:</strong> %s</p>
            <p style="margin: 5px 0;"><strong>New Role:</strong> %s</p>
        </div>

        <p>Your new role grants you different permissions within the business. Please review the updated permissions in your dashboard.</p>

        <div style="margin: 30px 0;">
            <a href="%s/businesses/%s" style="background-color: #3498db; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">View Dashboard</a>
        </div>

        <p style="color: #7f8c8d; font-size: 12px;">Best regards,<br>The Hauslet Team</p>
    </div>
</body>
</html>`, tb.buildLogoHTML(data.BusinessLogoURL), data.MemberName, data.BusinessName, data.ChangedBy, data.OldRole, data.NewRole, tb.baseURL, data.BusinessID)

	return &EmailTemplate{
		Subject:  subject,
		TextBody: textBody,
		HTMLBody: htmlBody,
		Variables: map[string]interface{}{
			"business_name": data.BusinessName,
			"member_name":   data.MemberName,
			"old_role":      data.OldRole,
			"new_role":      data.NewRole,
			"changed_by":    data.ChangedBy,
		},
	}
}

// BuildInvitationAcceptedEmail builds an invitation accepted notification email (for business owners)
func (tb *TemplateBuilder) BuildInvitationAcceptedEmail(data InvitationAcceptedEmailData) *EmailTemplate {
	subject := fmt.Sprintf("%s accepted your invitation to %s", data.AcceptedBy, data.BusinessName)

	textBody := fmt.Sprintf(`Hi,

Good news! %s (%s) has accepted your invitation to join %s as a %s.

They can now access the business dashboard and collaborate with your team.

View team members: %s/businesses/%s/members

Best regards,
The Hauslet Team
`, data.AcceptedBy, data.AcceptedEmail, data.BusinessName, data.Role, tb.baseURL, data.BusinessID)

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Invitation Accepted</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f8f9fa; padding: 20px; border-radius: 10px;">
        %s
        <h2 style="color: #27ae60; margin-top: 0;">Invitation Accepted!</h2>

        <p>Hi,</p>

        <p>Good news! <strong>%s</strong> (%s) has accepted your invitation to join <strong>%s</strong> as a <strong>%s</strong>.</p>

        <p>They can now access the business dashboard and collaborate with your team.</p>

        <div style="margin: 30px 0;">
            <a href="%s/businesses/%s/members" style="background-color: #27ae60; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">View Team Members</a>
        </div>

        <p style="color: #7f8c8d; font-size: 12px;">Best regards,<br>The Hauslet Team</p>
    </div>
</body>
</html>`, tb.buildLogoHTML(data.BusinessLogoURL), data.AcceptedBy, data.AcceptedEmail, data.BusinessName, data.Role, tb.baseURL, data.BusinessID)

	return &EmailTemplate{
		Subject:  subject,
		TextBody: textBody,
		HTMLBody: htmlBody,
		Variables: map[string]interface{}{
			"business_name":  data.BusinessName,
			"accepted_by":    data.AcceptedBy,
			"accepted_email": data.AcceptedEmail,
			"role":           data.Role,
		},
	}
}

// BuildInvitationDeclinedEmail builds an invitation declined notification email (for business owners)
func (tb *TemplateBuilder) BuildInvitationDeclinedEmail(data InvitationDeclinedEmailData) *EmailTemplate {
	subject := fmt.Sprintf("%s declined your invitation to %s", data.DeclinedBy, data.BusinessName)

	textBody := fmt.Sprintf(`Hi,

%s (%s) has declined your invitation to join %s.

You can invite someone else or try reaching out to them directly.

Manage invitations: %s/businesses/%s/invitations

Best regards,
The Hauslet Team
`, data.DeclinedBy, data.DeclinedEmail, data.BusinessName, tb.baseURL, data.BusinessID)

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Invitation Declined</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f8f9fa; padding: 20px; border-radius: 10px;">
        %s
        <h2 style="color: #e67e22; margin-top: 0;">Invitation Declined</h2>

        <p>Hi,</p>

        <p><strong>%s</strong> (%s) has declined your invitation to join <strong>%s</strong>.</p>

        <p>You can invite someone else or try reaching out to them directly.</p>

        <div style="margin: 30px 0;">
            <a href="%s/businesses/%s/invitations" style="background-color: #3498db; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Manage Invitations</a>
        </div>

        <p style="color: #7f8c8d; font-size: 12px;">Best regards,<br>The Hauslet Team</p>
    </div>
</body>
</html>`, tb.buildLogoHTML(data.BusinessLogoURL), data.DeclinedBy, data.DeclinedEmail, data.BusinessName, tb.baseURL, data.BusinessID)

	return &EmailTemplate{
		Subject:  subject,
		TextBody: textBody,
		HTMLBody: htmlBody,
		Variables: map[string]interface{}{
			"business_name":  data.BusinessName,
			"declined_by":    data.DeclinedBy,
			"declined_email": data.DeclinedEmail,
		},
	}
}

// BuildBusinessCreatedEmail builds a business creation notification email
func (tb *TemplateBuilder) BuildBusinessCreatedEmail(data BusinessCreatedEmailData) *EmailTemplate {
	subject := fmt.Sprintf("Welcome to %s - Your business is ready!", data.BusinessName)

	textBody := fmt.Sprintf(`Hi %s,

Congratulations! Your business "%s" has been successfully created on Hauslet.

You can now:
- Invite team members to collaborate
- Create and manage property listings
- Access business analytics and insights

Get started: %s

Best regards,
The Hauslet Team
`, data.CreatorName, data.BusinessName, data.DashboardURL)

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Business Created</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f8f9fa; padding: 20px; border-radius: 10px;">
        %s
        <h2 style="color: #27ae60; margin-top: 0;">Welcome to %s!</h2>

        <p>Hi %s,</p>

        <p>Congratulations! Your business <strong>"%s"</strong> has been successfully created on Hauslet.</p>

        <div style="background-color: white; padding: 20px; border-radius: 5px; margin: 20px 0;">
            <h3 style="margin-top: 0; color: #2c3e50;">You can now:</h3>
            <ul style="list-style-type: none; padding-left: 0;">
                <li style="margin: 10px 0;">✓ Invite team members to collaborate</li>
                <li style="margin: 10px 0;">✓ Create and manage property listings</li>
                <li style="margin: 10px 0;">✓ Access business analytics and insights</li>
            </ul>
        </div>

        <div style="margin: 30px 0;">
            <a href="%s" style="background-color: #27ae60; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block;">Get Started</a>
        </div>

        <p style="color: #7f8c8d; font-size: 12px;">Best regards,<br>The Hauslet Team</p>
    </div>
</body>
</html>`, tb.buildLogoHTML(data.BusinessLogoURL), data.BusinessName, data.CreatorName, data.BusinessName, data.DashboardURL)

	return &EmailTemplate{
		Subject:  subject,
		TextBody: textBody,
		HTMLBody: htmlBody,
		Variables: map[string]interface{}{
			"business_name":  data.BusinessName,
			"creator_name":   data.CreatorName,
			"dashboard_url":  data.DashboardURL,
		},
	}
}

// buildLogoHTML builds HTML for business logo if available
func (tb *TemplateBuilder) buildLogoHTML(logoURL *string) string {
	if logoURL != nil && *logoURL != "" {
		return fmt.Sprintf(`<div style="text-align: center; margin-bottom: 20px;">
            <img src="%s" alt="Business Logo" style="max-width: 150px; height: auto;">
        </div>`, *logoURL)
	}
	return ""
}
