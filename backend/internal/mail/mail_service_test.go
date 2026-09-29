package mail

import (
	"errors"
	"strings"
	"testing"
)

func TestNewService(t *testing.T) {
	mockSender := &MockSender{}
	mockAuditor := &MockAuditor{}

	tests := []struct {
		name    string
		sender  Sender
		auditor Auditor
		wantErr bool
	}{
		{"valid dependencies", mockSender, mockAuditor, false},
		{"nil sender", nil, mockAuditor, true},
		{"nil auditor", mockSender, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewService(tt.sender, tt.auditor)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewService() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestService_SendWelcomeEmail(t *testing.T) {
	tests := []struct {
		name              string
		mockSendError     error
		expectedErr       bool
		expectedAuditType string
	}{
		{"success logs email_sent", nil, false, "email_sent"},
		{"failure logs email_failed", errors.New("smtp down"), true, "email_failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &MockSender{SendFunc: func(to []string, subject, body string) error { return tt.mockSendError }}
			auditor := &MockAuditor{}
			service, _ := NewService(sender, auditor)

			err := service.SendWelcomeEmail("test@demo.local", "user-1")

			if (err != nil) != tt.expectedErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectedErr, err)
			}
			if sender.CalledCount != 1 {
				t.Errorf("Expected sender to be called 1 time, got %d", sender.CalledCount)
			}
			if auditor.LastEventType != tt.expectedAuditType {
				t.Errorf("Expected audit event %q, got %q", tt.expectedAuditType, auditor.LastEventType)
			}
		})
	}
}

func TestService_SendUserWelcomeEmail(t *testing.T) {
	tests := []struct {
		name              string
		mockSendError     error
		expectedErr       bool
		expectedAuditType string
	}{
		{"success logs email_sent", nil, false, "email_sent"}, // Note the lowercase here based on your code!
		{"failure logs email_failed", errors.New("timeout"), true, "email_failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &MockSender{SendFunc: func(to []string, subject, body string) error { return tt.mockSendError }}
			auditor := &MockAuditor{}
			service, _ := NewService(sender, auditor)

			err := service.SendUserWelcomeEmail("test@demo.local", "user-1", "Alice", "http://invite")

			if (err != nil) != tt.expectedErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectedErr, err)
			}
			// Verify the body actually contains the invite URL
			if sender.CalledCount > 0 && !strings.Contains(sender.LastBody, "http://invite") {
				t.Errorf("Expected email body to contain invite URL, got: %s", sender.LastBody)
			}
			if auditor.LastEventType != tt.expectedAuditType {
				t.Errorf("Expected audit event %q, got %q", tt.expectedAuditType, auditor.LastEventType)
			}
		})
	}
}

func TestService_SendSOPPublishedEmail(t *testing.T) {
	sender := &MockSender{SendFunc: func(to []string, subject, body string) error { return nil }}
	auditor := &MockAuditor{}
	service, _ := NewService(sender, auditor)

	err := service.SendSOPPublishedEmail("reader@demo.local", "Rae", "Water Protocol", 2, "Add the rinse step", "http://lab/sops/1/v/latest")
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if !strings.Contains(sender.LastSubject, "Water Protocol") || !strings.Contains(sender.LastBody, "Add the rinse step") {
		t.Fatalf("subject %q body %q", sender.LastSubject, sender.LastBody)
	}
	if auditor.LastEventType != "email_sent" {
		t.Fatalf("audit %q", auditor.LastEventType)
	}
}

func TestService_SendPasswordResetEmail(t *testing.T) {
	tests := []struct {
		name              string
		mockSendError     error
		expectedErr       bool
		expectedAuditType string
	}{
		{"success logs email_sent", nil, false, "email_sent"},
		{"failure logs email_failed", errors.New("auth failed"), true, "email_failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &MockSender{SendFunc: func(to []string, subject, body string) error { return tt.mockSendError }}
			auditor := &MockAuditor{}
			service, _ := NewService(sender, auditor)

			err := service.SendPasswordResetEmail("test@demo.local", "user-1", "Alice", "http://reset")

			if (err != nil) != tt.expectedErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectedErr, err)
			}
			if sender.CalledCount > 0 && !strings.Contains(sender.LastBody, "http://reset") {
				t.Errorf("Expected email body to contain reset URL, got: %s", sender.LastBody)
			}
			if auditor.LastEventType != tt.expectedAuditType {
				t.Errorf("Expected audit event %q, got %q", tt.expectedAuditType, auditor.LastEventType)
			}
		})
	}
}
