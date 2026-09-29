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
