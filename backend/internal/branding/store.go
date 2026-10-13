package branding

import (
	"database/sql"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
)

type record struct {
	name, light, dark, hash string
	hasLogo                 bool
}

func getRecord(db audit.DBTX) (record, error) {
	var (
		r                       record
		name, light, dark, hash sql.NullString
	)
	err := db.QueryRow(`
		SELECT name, primary_light, primary_dark, logo_hash, logo_png IS NOT NULL
		FROM branding WHERE id = 1
	`).Scan(&name, &light, &dark, &hash, &r.hasLogo)
	if err == sql.ErrNoRows {
		return record{}, nil
	}
	if err != nil {
		return record{}, err
	}
	r.name, r.light, r.dark, r.hash = name.String, light.String, dark.String, hash.String
	return r, nil
}

// nullIfEmpty stores defaults as NULL so "not set" stays distinct from a value.
func nullIfEmpty(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func setTextRecord(db audit.DBTX, name, light, dark string, now time.Time) error {
	_, err := db.Exec(`
		INSERT INTO branding (id, name, primary_light, primary_dark, updated_at)
		VALUES (1, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET name = excluded.name,
			primary_light = excluded.primary_light,
			primary_dark = excluded.primary_dark,
			updated_at = excluded.updated_at
	`, nullIfEmpty(name), nullIfEmpty(light), nullIfEmpty(dark), now.UTC().Format(time.RFC3339Nano))
	return err
}

func setLogoRecord(db audit.DBTX, logo, icon []byte, hash string, now time.Time) error {
	_, err := db.Exec(`
		INSERT INTO branding (id, logo_png, icon_png, logo_hash, updated_at)
		VALUES (1, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET logo_png = excluded.logo_png,
			icon_png = excluded.icon_png,
			logo_hash = excluded.logo_hash,
			updated_at = excluded.updated_at
	`, logo, icon, hash, now.UTC().Format(time.RFC3339Nano))
	return err
}

func clearLogoRecord(db audit.DBTX, now time.Time) error {
	_, err := db.Exec(`
		UPDATE branding SET logo_png = NULL, icon_png = NULL, logo_hash = NULL, updated_at = ?
		WHERE id = 1
	`, now.UTC().Format(time.RFC3339Nano))
	return err
}

func deleteRecord(db audit.DBTX) error {
	_, err := db.Exec(`DELETE FROM branding WHERE id = 1`)
	return err
}

func getImageRecord(db audit.DBTX, variant string) ([]byte, string, error) {
	column := "logo_png"
	if variant == VariantIcon {
		column = "icon_png"
	}
	var (
		data []byte
		hash sql.NullString
	)
	// column is one of two constants above, never user input.
	err := db.QueryRow(`SELECT `+column+`, logo_hash FROM branding WHERE id = 1`).Scan(&data, &hash)
	if err == sql.ErrNoRows || (err == nil && len(data) == 0) {
		return nil, "", ErrNoLogo
	}
	if err != nil {
		return nil, "", err
	}
	return data, hash.String, nil
}
