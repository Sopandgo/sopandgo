package mail

import (
	"fmt"
)

func NewService(sender Sender, auditor Auditor) (*Service, error) {
	if sender == nil {
		return nil, fmt.Errorf("mail sender cannot be nil")
	}
	if auditor == nil {
		return nil, fmt.Errorf("audit logger cannot be nil")
	}
	return &Service{
		sender:  sender,
		auditor: auditor,
	}, nil
}

// Silence makes every send a no-op that fails with ErrSilenced, whatever the
// stored SMTP or Resend settings say. A public demo calls it before serving.
func (s *Service) Silence() {
	s.sender = silencedSender{}
}

type silencedSender struct{}

func (silencedSender) Send([]string, string, string) error { return ErrSilenced }
