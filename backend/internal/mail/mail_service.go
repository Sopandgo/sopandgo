package mail

import (
	"fmt"
	"log"
	"strings"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
)

func (s *Service) SendWelcomeEmail(recipientEmail string, userID string) error {
	// 1. Prepare Content
	subject := "Welcome to SOPandGO!"
	body := "Hello! We are glad to have you."

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

func (s *Service) SendUserWelcomeEmail(recipientEmail, userID, userDisplayName, inviteURL string) error {
	// 1. Prepare Content
	subject := "Welcome to SOPandGO!"
	body := fmt.Sprintf("Hi %s,\n\nYour SOPandGO user account has been created by your admin.\n\nPlease use this link to set your password:\n%s\n\nCheers!",
		userDisplayName,
		inviteURL,
	)
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
func (s *Service) SendSOPPublishedEmail(recipientEmail, recipientName, sopTitle string, version int, summary, link string) error {
	if strings.TrimSpace(summary) == "" {
		summary = "No change summary was recorded."
	}
	subject := fmt.Sprintf("Published: %s (version %d)", sopTitle, version)
	body := fmt.Sprintf("Hi %s,\n\n%s version %d was published.\n\nWhat changed:\n%s\n\nOpen the SOP:\n%s\n",
		recipientName,
		sopTitle,
		version,
		summary,
		link,
	)
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
func (s *Service) SendPasswordResetEmail(recipientEmail, userID, userDisplayName, resetURL string) error {
	// 1. Prepare Content
	subject := "Reset your SOPandGO password"
	body := fmt.Sprintf("Hi %s,\n\nAn administrator has requested a password reset for your account.\n\nPlease use the link below to set a new password:\n%s\n\nThis link will expire in 24 hours.\n\nBest,\nThe SOPandGO Team",
		userDisplayName,
		resetURL,
	)

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
