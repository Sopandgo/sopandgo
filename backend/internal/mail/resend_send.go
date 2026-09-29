package mail

import (
	"fmt"

	"github.com/resend/resend-go/v3"
)

func (s *SMTPSettingsStore) sendResendEmail(to []string, subject, body string) error {
	apiKey, from, err := s.loadResendDecrypted()
	if err != nil {
		return err
	}
	client := resend.NewClient(apiKey)
	_, err = client.Emails.Send(&resend.SendEmailRequest{
		From:    from,
		To:      to,
		Subject: subject,
		Text:    body,
	})
	if err != nil {
		return fmt.Errorf("resend: %w", err)
	}
	return nil
}
