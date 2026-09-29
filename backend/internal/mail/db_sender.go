package mail

// DatabaseSender sends mail using settings and transport from SMTPSettingsStore.
type DatabaseSender struct {
	store *SMTPSettingsStore
}

// NewDatabaseSender creates a sender backed by the database.
func NewDatabaseSender(store *SMTPSettingsStore) *DatabaseSender {
	return &DatabaseSender{store: store}
}

// Send implements Sender.
func (d *DatabaseSender) Send(to []string, subject string, body string) error {
	return d.store.SendOperational(to, subject, body)
}
