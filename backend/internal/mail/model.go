package mail

import "github.com/sopandgo/sopandgo/backend/internal/audit"

type Sender interface {
	Send(to []string, subject string, body string) error
}

type Auditor interface {
	Log(executor audit.DBTX, eventType, entityType, entityID string, actorUserID *string, payload any) error
}

// Service handles email business logic and audit logging.
type Service struct {
	sender  Sender
	auditor Auditor
}

type SmtpSender struct {
	host     string
	port     string
	username string
	password string
	from     string
}
