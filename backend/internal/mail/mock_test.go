package mail

import "github.com/sopandgo/sopandgo/backend/internal/audit"

// MockSender implements the mail.Sender interface
type MockSender struct {
	SendFunc func(to []string, subject, body string) error
	// Helper fields to track what was called during the test
	CalledCount int
	LastTo      []string
	LastSubject string
	LastBody    string
}

func (m *MockSender) Send(to []string, subject, body string) error {
	m.CalledCount++
	m.LastTo = to
	m.LastSubject = subject
	m.LastBody = body

	if m.SendFunc != nil {
		return m.SendFunc(to, subject, body)
	}
	return nil // Default is success
}

// MockAuditor implements the mail.Auditor interface
type MockAuditor struct {
	LogFunc func(executor audit.DBTX, eventType, entityType, entityID string, actorUserID *string, payload any) error
	// Helper fields
	CalledCount   int
	LastEventType string
}

func (m *MockAuditor) Log(executor audit.DBTX, eventType, entityType, entityID string, actorUserID *string, payload any) error {
	m.CalledCount++
	m.LastEventType = eventType

	if m.LogFunc != nil {
		return m.LogFunc(executor, eventType, entityType, entityID, actorUserID, payload)
	}
	return nil
}
