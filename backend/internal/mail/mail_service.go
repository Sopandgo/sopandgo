package mail

import (
	"fmt"
	"log"
	"strings"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/i18n"
)

func (s *Service) SendWelcomeEmail(recipientEmail string, userID string, locale string) error {
	// 1. Prepare Content
	subject := i18n.T(locale, "mail.welcome.subject", nil)
	body := i18n.T(locale, "mail.welcome.body", nil)

	// 2. Attempt to Send
	// If this fails, we generally do NOT want to audit log a "success" or "sent" event.
	if err := s.sender.Send([]string{recipientEmail}, subject, body); err != nil {
		// Audit Failure
		payload := map[string]string{
			"recipient": recipientEmail,
			"template":  "welcome_email_v1",
			"status":    "failed",
		}

		err := s.auditor.Log(nil, audit.EventEmailFailed, audit.EntityEmail, userID, nil, payload)
		if err != nil {
			// In production, use a proper logger like zap/logrus here
			fmt.Printf("WARNING: Failed to write audit log for email sent: %v\n", err)
		}

		return fmt.Errorf("failed to send welcome email: %w", err)
	}

	// 3. Audit the Success
	// We run this asynchronously (go func) OR synchronously depending on strictness.
	// Usually, for "Welcome Emails", if the audit fails, we don't want to crash the request,
	// but we should log the error to the console.
	payload := map[string]string{
		"recipient": recipientEmail,
		"template":  "welcome_email_v1",
		"status":    "sent",
	}

	// Helper to handle pointer for actorID (System sent this, so actor is nil or specific system ID)
	// Passing 'nil' as executor uses the audit logger's default DB connection.
	err := s.auditor.Log(nil, audit.EventEmailSent, audit.EntityUser, userID, nil, payload)
	if err != nil {
		// In production, use a proper logger like zap/logrus here
		fmt.Printf("WARNING: Failed to write audit log for email sent: %v\n", err)
	}

	return nil
}

func (s *Service) SendUserWelcomeEmail(recipientEmail, userID, userDisplayName, inviteURL, locale string) error {
	// 1. Prepare Content
	subject := i18n.T(locale, "mail.invite.subject", nil)
	body := i18n.T(locale, "mail.invite.body", map[string]string{
		"name": userDisplayName,
		"url":  inviteURL,
	})
	// 2. Attempt to Send
	// If this fails, we generally do NOT want to audit log a "success" or "sent" event.
	if err := s.sender.Send([]string{recipientEmail}, subject, body); err != nil {
		// Audit Failure
		payload := map[string]string{
			"recipient": recipientEmail,
			"template":  "welcome_email_v1",
			"status":    "failed",
		}

		err := s.auditor.Log(nil, audit.EventEmailFailed, audit.EntityEmail, userID, nil, payload)
		if err != nil {
			// In production, use a proper logger like zap/logrus here
			fmt.Printf("WARNING: Failed to write audit log for email send failure: %v\n", err)
		}

		return fmt.Errorf("failed to send welcome email: %w", err)
	}

	// 3. Audit the Success
	payload := map[string]string{
		"recipient": recipientEmail,
		"template":  "welcome_email_v1",
		"status":    "sent",
	}

	err := s.auditor.Log(nil, audit.EventEmailSent, audit.EntityEmail, userID, nil, payload)
	if err != nil {
		// In production, use a proper logger like zap/logrus here
		fmt.Printf("WARNING: Failed to write audit log for email sent: %v\n", err)
	}

	return nil
}

// SendSOPPublishedEmail tells one reader that a version is now the published procedure.
// Delivery failure is the caller's concern; publish itself must not depend on this succeeding.
func (s *Service) SendSOPPublishedEmail(recipientEmail, recipientName, sopTitle string, version int, summary, link, locale string) error {
	if strings.TrimSpace(summary) == "" {
		summary = i18n.T(locale, "mail.published.no_summary", nil)
	}
	subject := i18n.T(locale, "mail.published.subject", map[string]string{
		"title":   sopTitle,
		"version": fmt.Sprintf("%d", version),
	})
	body := i18n.T(locale, "mail.published.body", map[string]string{
		"name":    recipientName,
		"title":   sopTitle,
		"version": fmt.Sprintf("%d", version),
		"summary": summary,
		"url":     link,
	})
	if err := s.sender.Send([]string{recipientEmail}, subject, body); err != nil {
		payload := map[string]string{
			"recipient": recipientEmail,
			"template":  "sop_published_v1",
			"status":    "failed",
		}
		if logErr := s.auditor.Log(nil, audit.EventEmailFailed, audit.EntityEmail, recipientEmail, nil, payload); logErr != nil {
			log.Printf("WARNING: Failed to write audit log for publish notice failure: %v", logErr)
		}
		return fmt.Errorf("failed to send publish notice: %w", err)
	}
	payload := map[string]string{
		"recipient": recipientEmail,
		"template":  "sop_published_v1",
		"status":    "sent",
	}
	if err := s.auditor.Log(nil, audit.EventEmailSent, audit.EntityEmail, recipientEmail, nil, payload); err != nil {
		log.Printf("WARNING: Failed to write audit log for publish notice: %v", err)
	}
	return nil
}

// SendPasswordResetEmail is sent when an admin manually triggers a reset for an EXISTING user.
func (s *Service) SendPasswordResetEmail(recipientEmail, userID, userDisplayName, resetURL, locale string) error {
	// 1. Prepare Content
	subject := i18n.T(locale, "mail.reset.subject", nil)
	body := i18n.T(locale, "mail.reset.body", map[string]string{
		"name": userDisplayName,
		"url":  resetURL,
	})

	// 2. Attempt to Send
	if err := s.sender.Send([]string{recipientEmail}, subject, body); err != nil {
		// Audit Failure
		payload := map[string]string{
			"recipient": recipientEmail,
			"template":  "password_reset_v1",
			"status":    "failed",
		}

		err := s.auditor.Log(nil, audit.EventEmailFailed, audit.EntityEmail, userID, nil, payload)
		if err != nil {
			// In production, use a proper logger like zap/logrus here
			fmt.Printf("WARNING: Failed to write audit log for email send failure: %v\n", err)
		}

		return fmt.Errorf("failed to send password reset email: %w", err)
	}

	// 3. Audit the Success
	payload := map[string]string{
		"recipient": recipientEmail,
		"template":  "password_reset_v1",
		"status":    "sent",
	}

	err := s.auditor.Log(nil, audit.EventEmailSent, audit.EntityUser, userID, nil, payload)
	if err != nil {
		log.Printf("COMPLIANCE WARNING: Reset email sent to %s (User %s) but audit log failed: %v", recipientEmail, userID, err)
	}

	return nil
}
