package storage

import "database/sql"

type migration struct {
	version int
	up      func(tx *sql.Tx) error
}

// migrations defines the complete, append-only history of database schema changes.
//
// IMPORTANT:
// - Existing migrations must NEVER be modified or reordered.
// - New schema changes are added as new migrations with a higher version number.
// - If a mistake is discovered in an existing migration, fix it by adding a new migration.
// - This ensures deterministic upgrades and preserves data integrity.
var migrations = []migration{
	// v1: schema version tracking
	{
		version: 1,
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				CREATE TABLE IF NOT EXISTS schema_version (
					version INTEGER NOT NULL
				);
			`)
			if err != nil {
				return err
			}

			// initialize version if table is empty
			_, err = tx.Exec(`
				INSERT INTO schema_version (version)
				SELECT 0
				WHERE NOT EXISTS (SELECT 1 FROM schema_version);
			`)
			return err
		},
	},

	// v2: SOP metadata
	{
		version: 2,
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				CREATE TABLE sops (
					id TEXT PRIMARY KEY,
					title TEXT NOT NULL UNIQUE,
					created_at TEXT NOT NULL
				);
			`)
			return err
		},
	},

	// v3: immutable SOP versions
	{
		version: 3,
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
			CREATE TABLE sop_versions (
				id TEXT PRIMARY KEY,
				sop_id TEXT NOT NULL,
				version INTEGER NOT NULL,
				content_path TEXT NOT NULL,
				content_hash TEXT NOT NULL,
				created_at TEXT NOT NULL,
				FOREIGN KEY (sop_id) REFERENCES sops(id),
				UNIQUE (sop_id, version)
			);
		`)
			return err
		},
	},

	// v4: SOP assets (append-only, hashed)
	{
		version: 4,
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
			CREATE TABLE sop_assets (
				id TEXT PRIMARY KEY,
				sop_id TEXT NOT NULL,
				file_name TEXT NOT NULL,
				content_path TEXT NOT NULL,
				content_hash TEXT NOT NULL,
				created_at TEXT NOT NULL,
				FOREIGN KEY (sop_id) REFERENCES sops(id),
				UNIQUE (sop_id, content_path)
			);
		`)
			return err
		},
	},

	// v5: users
	{
		version: 5,
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
                CREATE TABLE roles (
                    id TEXT PRIMARY KEY,
                    description TEXT
                );
                INSERT INTO roles (id, description) VALUES 
                    ('admin', 'Full system access'),
                    ('editor', 'Can create SOP drafts.'),
                    ('approver', 'Can sign off on SOP release candidates'),
                    ('auditor', 'Can view logs and history'),
                    ('viewer', 'Read-only access + can sign SOPs as read');

                CREATE TABLE users (
                    id TEXT PRIMARY KEY,
                    display_name TEXT NOT NULL,
                    email TEXT UNIQUE NOT NULL,
                    password_hash TEXT NOT NULL,
                    role_id TEXT NOT NULL DEFAULT 'viewer',
                    is_active INTEGER NOT NULL DEFAULT 1,
                    created_at TEXT NOT NULL,
                    FOREIGN KEY (role_id) REFERENCES roles(id)
                );
            `)
			return err
		},
	},

	// v6: SOP version acknowledgments
	{
		version: 6,
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
			CREATE TABLE sop_acknowledgments (
				id TEXT PRIMARY KEY,
				sop_version_id TEXT NOT NULL,
				user_id TEXT NOT NULL,
				ack_type TEXT NOT NULL,
				created_at TEXT NOT NULL,
				FOREIGN KEY (sop_version_id) REFERENCES sop_versions(id),
				FOREIGN KEY (user_id) REFERENCES users(id),
				UNIQUE (sop_version_id, user_id, ack_type)
			);
		`)
			return err
		},
	},

	// v7: audit event log (append-only)
	{
		version: 7,
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
        CREATE TABLE audit_events (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            event_type TEXT NOT NULL,
            entity_type TEXT NOT NULL,
            entity_id TEXT NOT NULL,
            actor_user_id TEXT NULL,
            payload TEXT NOT NULL,
            created_at TEXT NOT NULL,
            hash TEXT NOT NULL,
            prev_hash TEXT NOT NULL,

            FOREIGN KEY (actor_user_id) REFERENCES users(id),
            UNIQUE(prev_hash)
        );

        CREATE INDEX idx_audit_events_created_at ON audit_events(created_at DESC);
        `)
			return err
		},
	},

	// v8: refresh tokens
	// is_active: -- 1 for active, 0 for revoked
	{
		version: 8,
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
			CREATE TABLE refresh_tokens (
				token_id TEXT PRIMARY KEY,
				user_id TEXT NOT NULL,
				expires_at DATETIME NOT NULL,
				created_at DATETIME NOT NULL,
				is_active INTEGER DEFAULT 1,
				FOREIGN KEY (user_id) REFERENCES users(id)
			);
		`)
			return err
		},
	},

	// v9: password reset tokens
	// is_active: -- 1 for active, 0 for revoked
	{
		version: 9,
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
			CREATE TABLE password_reset_tokens (
				user_id TEXT PRIMARY KEY,
				token_hash TEXT NOT NULL,
				expires_at DATETIME NOT NULL,
				FOREIGN KEY (user_id) REFERENCES users(id)
			);
		`)
			return err
		},
	},

	// v10: SOP version lifecycle states (append-only)
	{
		version: 10,
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
                CREATE TABLE sop_version_states (
                    id TEXT PRIMARY KEY,
                    sop_version_id TEXT NOT NULL,
                    state TEXT NOT NULL, -- 'draft', 'rc', 'published', 'rejected', 'archived'
                    actor_user_id TEXT NOT NULL,
                    created_at TEXT NOT NULL,
                    
                    FOREIGN KEY (sop_version_id) REFERENCES sop_versions(id),
                    FOREIGN KEY (actor_user_id) REFERENCES users(id)
                );

                -- Optimizes the lookup for the "current" state of any given version
                CREATE INDEX idx_sop_version_states_latest 
                ON sop_version_states(sop_version_id, created_at DESC);
            `)
			return err
		},
	},

	// v11: SOP tags
	{
		version: 11,
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				CREATE TABLE tags (
					id TEXT PRIMARY KEY,
					title TEXT NOT NULL UNIQUE,
					created_at TEXT NOT NULL,
					is_active INTEGER DEFAULT 1
				);
				
				CREATE TABLE sops_tags (
					sop_id TEXT NOT NULL,
					tag_id TEXT NOT NULL,
					PRIMARY KEY (sop_id, tag_id),
					FOREIGN KEY (sop_id) REFERENCES sops(id), 
					FOREIGN KEY (tag_id) REFERENCES tags(id)
				);
			`)
			return err
		},
	},
	// v12: normalize audit taxonomy casing
	{
		version: 12,
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				UPDATE audit_events
				SET
					event_type = lower(trim(event_type)),
					entity_type = lower(trim(entity_type))
				WHERE event_type != lower(trim(event_type))
				   OR entity_type != lower(trim(entity_type));
			`)
			return err
		},
	},

	// v13: SMTP settings (single row, password at rest encrypted with env key)
	{
		version: 13,
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				CREATE TABLE smtp_settings (
					id INTEGER PRIMARY KEY CHECK (id = 1),
					host TEXT NOT NULL,
					port TEXT NOT NULL,
					username TEXT NOT NULL,
					password_enc BLOB NOT NULL,
					from_addr TEXT NOT NULL,
					updated_at TEXT NOT NULL
				);
			`)
			return err
		},
	},

	// v14: enforce at most one concurrently published SOP version per SOP container
	{
		version: 14,
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				CREATE TRIGGER sop_version_states_guard_published_insert
				BEFORE INSERT ON sop_version_states
				WHEN NEW.state = 'published'
				BEGIN
					SELECT RAISE(ABORT, 'conflict: this version is already published')
					WHERE EXISTS (
						SELECT 1
						FROM sop_versions v
						WHERE v.id = NEW.sop_version_id
						  AND (
							SELECT st.state
							FROM sop_version_states st
							WHERE st.sop_version_id = v.id
							ORDER BY st.created_at DESC, st.rowid DESC
							LIMIT 1
						  ) = 'published'
					);
					SELECT RAISE(ABORT, 'conflict: another version of this SOP is still published')
					WHERE EXISTS (
						SELECT 1
						FROM sop_versions v
						WHERE v.sop_id = (SELECT sop_id FROM sop_versions WHERE id = NEW.sop_version_id)
						  AND v.id != NEW.sop_version_id
						  AND (
							SELECT st.state
							FROM sop_version_states st
							WHERE st.sop_version_id = v.id
							ORDER BY st.created_at DESC, st.rowid DESC
							LIMIT 1
						  ) = 'published'
					);
				END;
			`)
			return err
		},
	},
	// v15: mail mode setting (smtp|manual_links)
	{
		version: 15,
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				CREATE TABLE app_settings (
					id INTEGER PRIMARY KEY CHECK (id = 1),
					mail_mode TEXT NOT NULL DEFAULT 'smtp' CHECK (mail_mode IN ('smtp', 'manual_links')),
					updated_at TEXT NOT NULL
				);

				INSERT INTO app_settings (id, mail_mode, updated_at)
				SELECT 1, 'smtp', strftime('%Y-%m-%dT%H:%M:%fZ','now')
				WHERE NOT EXISTS (SELECT 1 FROM app_settings WHERE id = 1);
			`)
			return err
		},
	},
	// v16: append-only SOP PDF artifacts
	{
		version: 16,
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				CREATE TABLE sop_version_pdf_artifacts (
					id TEXT PRIMARY KEY,
					sop_id TEXT NOT NULL,
					sop_version_id TEXT NOT NULL,
					stage TEXT NOT NULL,
					generator_version TEXT NOT NULL,
					file_path TEXT NOT NULL,
					content_hash TEXT NOT NULL,
					size_bytes INTEGER NOT NULL,
					created_by TEXT NOT NULL,
					created_at TEXT NOT NULL,
					FOREIGN KEY (sop_id) REFERENCES sops(id),
					FOREIGN KEY (sop_version_id) REFERENCES sop_versions(id),
					FOREIGN KEY (created_by) REFERENCES users(id),
					UNIQUE (sop_version_id, stage, generator_version)
				);

				CREATE INDEX idx_sop_pdf_artifacts_lookup
				ON sop_version_pdf_artifacts(sop_version_id, stage, generator_version, created_at DESC);
			`)
			return err
		},
	},
	// v17: per-user SOP favorites (private)
	{
		version: 17,
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				CREATE TABLE sop_favorites (
					user_id TEXT NOT NULL,
					sop_id TEXT NOT NULL,
					created_at TEXT NOT NULL,
					PRIMARY KEY (user_id, sop_id),
					FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
					FOREIGN KEY (sop_id) REFERENCES sops(id) ON DELETE CASCADE
				);

				CREATE INDEX idx_sop_favorites_user_created
				ON sop_favorites(user_id, created_at DESC);
			`)
			return err
		},
	},
	// v18: optional Resend transport (API key encrypted like SMTP password)
	{
		version: 18,
		up: func(tx *sql.Tx) error {
			if _, err := tx.Exec(`
				ALTER TABLE app_settings ADD COLUMN mail_transport TEXT NOT NULL DEFAULT 'smtp';
			`); err != nil {
				return err
			}
			_, err := tx.Exec(`
				CREATE TABLE resend_settings (
					id INTEGER PRIMARY KEY CHECK (id = 1),
					api_key_enc BLOB NOT NULL,
					from_addr TEXT NOT NULL,
					updated_at TEXT NOT NULL
				);
			`)
			return err
		},
	},
	// v19: immutable change summary on each SOP version
	{
		version: 19,
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				ALTER TABLE sop_versions ADD COLUMN change_summary TEXT NOT NULL DEFAULT '';
			`)
			return err
		},
	},
	// v20: force password change for bootstrap (and any future flagged) accounts
	{
		version: 20,
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				ALTER TABLE users ADD COLUMN must_change_password INTEGER NOT NULL DEFAULT 0;
			`)
			return err
		},
	},
	// v21: outbound integration settings (Slack, Gotify, generic webhook)
	{
		version: 21,
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				CREATE TABLE integration_settings (
					id INTEGER PRIMARY KEY CHECK (id = 1),
					slack_enabled INTEGER NOT NULL DEFAULT 0,
					slack_webhook_url_enc BLOB,
					slack_events TEXT NOT NULL DEFAULT 'sop_published,sop_rc,sop_rejected',
					gotify_enabled INTEGER NOT NULL DEFAULT 0,
					gotify_url TEXT NOT NULL DEFAULT '',
					gotify_token_enc BLOB,
					gotify_events TEXT NOT NULL DEFAULT 'sop_published,sop_rc,sop_rejected,backup_s3_failed,integrity_check_failed',
					webhook_enabled INTEGER NOT NULL DEFAULT 0,
					webhook_url TEXT NOT NULL DEFAULT '',
					webhook_bearer_enc BLOB,
					webhook_events TEXT NOT NULL DEFAULT 'sop_published,sop_rc,sop_rejected,backup_s3_failed,integrity_check_failed',
					updated_at TEXT NOT NULL
				);
			`)
			return err
		},
	},
	// v22: per-user UI locale and organization default for shared outbound text
	{
		version: 22,
		up: func(tx *sql.Tx) error {
			if _, err := tx.Exec(`
				ALTER TABLE users ADD COLUMN locale TEXT NOT NULL DEFAULT 'en';
			`); err != nil {
				return err
			}
			_, err := tx.Exec(`
				ALTER TABLE app_settings ADD COLUMN default_locale TEXT NOT NULL DEFAULT 'en';
			`)
			return err
		},
	},
}

func LatestSchemaVersion() int {
	if len(migrations) == 0 {
		return 0
	}
	return migrations[len(migrations)-1].version
}

func runMigrations(db *sql.DB) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var current int

	err = tx.QueryRow(`SELECT version FROM schema_version`).Scan(&current)
	if err != nil {
		// If the table doesn't exist yet, we're at version 0
		current = 0
	}

	for _, m := range migrations {
		if m.version > current {
			if err := m.up(tx); err != nil {
				return 0, err
			}
			_, err := tx.Exec(`UPDATE schema_version SET version = ?`, m.version)
			if err != nil {
				return 0, err
			}
			current = m.version
		}
	}

	return current, tx.Commit()
}
