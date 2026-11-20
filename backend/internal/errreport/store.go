package errreport

import (
	"database/sql"
	"errors"
)

// Settings is the admin's error reporting configuration. The DSN is not a
// secret (docs/adr/0004), so it is stored and returned as is.
type Settings struct {
	Enabled        bool   `json:"enabled"`
	DSN            string `json:"dsn"`
	Environment    string `json:"environment"`
	BrowserEnabled bool   `json:"browser_enabled"`
	UpdatedAt      string `json:"updated_at"`
}

// DefaultEnvironment is the environment events carry until an admin sets one.
const DefaultEnvironment = "production"

func defaultSettings() Settings {
	return Settings{Environment: DefaultEnvironment}
}

type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// loadRow returns the stored row, or nil when there is none.
func loadRow(db *sql.DB) (*Settings, error) {
	var st Settings
	var enabled, browser int
	err := db.QueryRow(`
		SELECT enabled, dsn, environment, browser_enabled, updated_at
		FROM error_reporting_settings WHERE id = 1
	`).Scan(&enabled, &st.DSN, &st.Environment, &browser, &st.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	st.Enabled = enabled == 1
	st.BrowserEnabled = browser == 1
	return &st, nil
}

func load(db *sql.DB) (Settings, error) {
	r, err := loadRow(db)
	if err != nil || r == nil {
		return defaultSettings(), err
	}
	return *r, nil
}

func writeRow(db execer, st Settings) error {
	_, err := db.Exec(`
		INSERT INTO error_reporting_settings (id, enabled, dsn, environment, browser_enabled, updated_at)
		VALUES (1, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			enabled = excluded.enabled,
			dsn = excluded.dsn,
			environment = excluded.environment,
			browser_enabled = excluded.browser_enabled,
			updated_at = excluded.updated_at
	`, boolInt(st.Enabled), st.DSN, st.Environment, boolInt(st.BrowserEnabled), st.UpdatedAt)
	return err
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
