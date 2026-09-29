package sop

import (
	"fmt"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
)

func createTagRecord(db audit.DBTX, id, title, createdAt string) error {
	_, err := db.Exec(`
		INSERT INTO tags (id, title, created_at)
		VALUES (?, ?, ?)
	`, id, title, createdAt)
	return err
}

func getTagsRecord(db audit.DBTX) ([]Tag, error) {
	const query = `
		SELECT id, title, is_active 
		FROM tags 
		ORDER BY title ASC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		err := rows.Scan(&t.ID, &t.Title, &t.IsActive)
		if err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}

	if tags == nil {
		tags = []Tag{}
	}
	return tags, nil
}

// --- SOP Tags ---

func attachTagToSOPRecord(db audit.DBTX, sopID, tagID string) error {
	_, err := db.Exec(`
		INSERT INTO sops_tags (sop_id, tag_id)
		VALUES (?, ?)
		ON CONFLICT(sop_id, tag_id) DO NOTHING
	`, sopID, tagID)
	return err
}

func detachTagFromSOPRecord(db audit.DBTX, sopID, tagID string) error {
	_, err := db.Exec(`
		DELETE FROM sops_tags 
		WHERE sop_id = ? AND tag_id = ?
	`, sopID, tagID)
	return err
}

func getTagsForSOPRecord(db audit.DBTX, sopID string) ([]Tag, error) {
	const query = `
		SELECT t.id, t.title, t.is_active
		FROM tags t
		JOIN sops_tags st ON t.id = st.tag_id
		WHERE st.sop_id = ? AND t.is_active = 1
		ORDER BY t.title ASC`

	rows, err := db.Query(query, sopID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tags for sop: %w", err)
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		err := rows.Scan(&t.ID, &t.Title, &t.IsActive)
		if err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}

	if tags == nil {
		tags = []Tag{}
	}
	return tags, nil
}

// setTagActiveStatusRecord updates the is_active flag for a tag.
func setTagActiveStatusRecord(db audit.DBTX, tagID string, isActive bool) error {
	val := 0
	if isActive {
		val = 1
	}

	res, err := db.Exec(`UPDATE tags SET is_active = ? WHERE id = ?`, val, tagID)
	if err != nil {
		return err
	}

	// Defensive check
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("tag not found: %s", tagID)
	}

	return nil
}

// --- Utility ---
func checkTagExistsCaseInsensitiveRecord(db audit.DBTX, title string) (bool, error) {
	var exists bool
	// COLLATE NOCASE tells SQLite to ignore case for this specific check
	query := `SELECT EXISTS(SELECT 1 FROM tags WHERE title = ? COLLATE NOCASE)`
	err := db.QueryRow(query, title).Scan(&exists)
	return exists, err
}
