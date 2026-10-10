package sop

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
)

// currentVersionStateSQL is a version's current state: its newest state row, with
// rowid breaking ties between rows written at the same instant. Every query that
// asks "which version is published" uses it, so the list, the SOP detail and
// /version-latest cannot disagree. Expects the version aliased as v.
const currentVersionStateSQL = `COALESCE((SELECT state FROM sop_version_states WHERE sop_version_id = v.id ORDER BY created_at DESC, rowid DESC LIMIT 1), '')`

// --- SOP ---

func createSOPRecord(db audit.DBTX, id, title, createdAt string) error {
	_, err := db.Exec(`
		INSERT INTO sops (id, title, created_at)
		VALUES (?, ?, ?)
	`, id, title, createdAt)
	return err
}

func getSOPByIDRecord(db audit.DBTX, id string) (*SOP, error) {
	const query = `
		SELECT id, title, created_at
		FROM sops WHERE id = ?`

	var s SOP
	var createdAt string

	err := db.QueryRow(query, id).Scan(
		&s.ID, &s.Title, &createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("sop not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	s.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	return &s, nil
}

func listSOPsRecord(db audit.DBTX, userID string, limit, offset int, tagID string, searchQuery string, favoritesOnly, favoritesFirst bool) ([]SOPListItem, int, error) {
	var args []any
	var conditions []string
	joins := ""

	useFav := userID != ""

	// JOIN order: tag association first (no placeholder in JOIN), then favorites (placeholder in JOIN).
	if tagID != "" {
		joins += ` JOIN sops_tags st ON s.id = st.sop_id`
		conditions = append(conditions, "st.tag_id = ?")
	}
	if useFav {
		joins += ` LEFT JOIN sop_favorites fav ON fav.sop_id = s.id AND fav.user_id = ?`
		args = append(args, userID)
	}
	if tagID != "" {
		args = append(args, tagID)
	}

	if favoritesOnly && useFav {
		conditions = append(conditions, "fav.user_id IS NOT NULL")
	}

	if searchQuery != "" {
		conditions = append(conditions, "s.title LIKE ?")
		args = append(args, "%"+searchQuery+"%")
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// 3. Execute the Count Query (for pagination totals)
	countSQL := `SELECT COUNT(DISTINCT s.id) FROM sops s` + joins + whereClause
	var total int
	if err := db.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count sops: %w", err)
	}

	// 4. Execute the actual Select Query
	if limit <= 0 {
		limit = -1 // SQLite treats LIMIT -1 as "return all rows"
	}

	isFavExpr := `0`
	if useFav {
		isFavExpr = `CASE WHEN fav.user_id IS NOT NULL THEN 1 ELSE 0 END`
	}

	orderBy := `s.created_at DESC, s.rowid DESC`
	if favoritesFirst && useFav {
		orderBy = `CASE WHEN fav.user_id IS NOT NULL THEN 0 ELSE 1 END, ` + orderBy
	}

	querySQL := `SELECT DISTINCT s.id, s.title, s.created_at, ` + isFavExpr + ` AS is_favorite FROM sops s` +
		joins +
		whereClause +
		` ORDER BY ` + orderBy + ` LIMIT ? OFFSET ?`

	// Copy args for the select query and append limit/offset
	queryArgs := append(args, limit, offset)

	rows, err := db.Query(querySQL, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list sops: %w", err)
	}
	defer rows.Close()

	var sops []SOPListItem
	for rows.Next() {
		var s SOPListItem
		var createdAt string
		var isFavInt int
		if err := rows.Scan(&s.ID, &s.Title, &createdAt, &isFavInt); err != nil {
			return nil, 0, err
		}
		s.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		s.IsFavorite = isFavInt != 0
		s.Tags = []Tag{} // Initialize so it safely serializes as [] in JSON, not null
		sops = append(sops, s)
	}

	// If no SOPs matched, exit early
	if len(sops) == 0 {
		return []SOPListItem{}, total, nil
	}

	// 5. Bulk Fetch Tags for the retrieved SOPs
	sopIDs := make([]any, len(sops))
	placeholders := make([]string, len(sops))
	for i, s := range sops {
		sopIDs[i] = s.ID
		placeholders[i] = "?"
	}

	tagQuery := `
		SELECT st.sop_id, t.id, t.title, t.is_active 
		FROM tags t
		JOIN sops_tags st ON t.id = st.tag_id
		WHERE st.sop_id IN (` + strings.Join(placeholders, ",") + `)
		ORDER BY t.title ASC
	`

	tagRows, err := db.Query(tagQuery, sopIDs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch tags for sops: %w", err)
	}
	defer tagRows.Close()

	// Map the tags to their respective SOP IDs
	tagsBySOP := make(map[string][]Tag)
	for tagRows.Next() {
		var sopID string
		var t Tag
		if err := tagRows.Scan(&sopID, &t.ID, &t.Title, &t.IsActive); err != nil {
			return nil, 0, err
		}
		tagsBySOP[sopID] = append(tagsBySOP[sopID], t)
	}

	// 6. Map tags back to their SOPs
	for i, s := range sops {
		if tags, ok := tagsBySOP[s.ID]; ok {
			sops[i].Tags = tags
		}
	}

	// 7. Bulk fetch versions (newest first) for the latest and published version per SOP
	versionQuery := `
		SELECT v.sop_id, v.id, v.version, v.created_at, ` + currentVersionStateSQL + ` AS status
		FROM sop_versions v
		WHERE v.sop_id IN (` + strings.Join(placeholders, ",") + `)
		ORDER BY v.sop_id, v.version DESC
	`

	versionRows, err := db.Query(versionQuery, sopIDs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch versions for sops: %w", err)
	}
	defer versionRows.Close()

	latestBySOP := make(map[string]*SOPListVersion)
	publishedBySOP := make(map[string]int)
	for versionRows.Next() {
		var sopID, createdAt string
		var v SOPListVersion
		if err := versionRows.Scan(&sopID, &v.ID, &v.Version, &createdAt, &v.Status); err != nil {
			return nil, 0, err
		}
		v.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		if _, ok := latestBySOP[sopID]; !ok {
			latestBySOP[sopID] = &v
		}
		if _, ok := publishedBySOP[sopID]; !ok && v.Status == StatePublished {
			publishedBySOP[sopID] = v.Version
		}
	}
	if err := versionRows.Err(); err != nil {
		return nil, 0, err
	}

	for i, s := range sops {
		sops[i].LatestVersion = latestBySOP[s.ID]
		if n, ok := publishedBySOP[s.ID]; ok {
			sops[i].PublishedVersion = &n
		}
	}

	return sops, total, nil
}

// getSOPVersionPointersRecord returns the newest version of an SOP and the version
// readers see. Either is nil when the SOP has no such version.
func getSOPVersionPointersRecord(db audit.DBTX, sopID string) (latest, published *SOPListVersion, err error) {
	query := `
		SELECT v.id, v.version, v.created_at, ` + currentVersionStateSQL + ` AS status
		FROM sop_versions v
		WHERE v.sop_id = ?
		ORDER BY v.version DESC
	`

	rows, err := db.Query(query, sopID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch versions for sop: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var v SOPListVersion
		var createdAt string
		if err := rows.Scan(&v.ID, &v.Version, &createdAt, &v.Status); err != nil {
			return nil, nil, err
		}
		v.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		if latest == nil {
			latest = &v
		}
		if published == nil && v.Status == StatePublished {
			published = &v
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return latest, published, nil
}

// --- SOP Version ---

func createSOPVersionRecord(db audit.DBTX, id, sopID, version, contentPath, contentHash, changeSummary, createdAt string) error {
	_, err := db.Exec(`
		INSERT INTO sop_versions (
			id,
			sop_id,
			version,
			content_path,
			content_hash,
			change_summary,
			created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`,
		id,
		sopID,
		version,
		contentPath,
		contentHash,
		changeSummary,
		createdAt,
	)
	return err
}

// getLatestSOPVersionRecord gets the highest version number regardless of state.
// It resolves the current state for idempotency and UI.
func getLatestSOPVersionRecord(tx *sql.Tx, sopID string) (*SOPVersion, error) {
	var v SOPVersion
	var createdAt string

	query := `
        SELECT v.id, v.sop_id, v.version, v.content_path, v.content_hash, v.change_summary, v.created_at,
               COALESCE((SELECT state FROM sop_version_states WHERE sop_version_id = v.id ORDER BY created_at DESC, rowid DESC LIMIT 1), '') as status
        FROM sop_versions v 
        WHERE v.sop_id = ? 
        ORDER BY v.version DESC 
        LIMIT 1
    `

	err := tx.QueryRow(query, sopID).Scan(
		&v.ID,
		&v.SOPID,
		&v.Version,
		&v.ContentPath,
		&v.ContentHash,
		&v.ChangeSummary,
		&createdAt,
		&v.Status,
	)

	if err != nil {
		return nil, err
	}

	v.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	return &v, nil
}

// hasReleaseCandidateForSOPRecord reports whether any version of this SOP is currently a release candidate
// (latest lifecycle state = rc). Used to enforce a single in-flight RC while allowing multiple drafts.
func hasReleaseCandidateForSOPRecord(db audit.DBTX, sopID string) (bool, error) {
	var n int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM sop_versions v
		WHERE v.sop_id = ?
		  AND (
			SELECT state FROM sop_version_states
			WHERE sop_version_id = v.id
			ORDER BY created_at DESC, rowid DESC
			LIMIT 1
		  ) = ?
	`, sopID, StateRC).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// hasAnotherReleaseCandidateForSOPRecord reports whether a different version (not excludeVersionID) is RC.
func hasAnotherReleaseCandidateForSOPRecord(db audit.DBTX, sopID, excludeVersionID string) (bool, error) {
	var n int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM sop_versions v
		WHERE v.sop_id = ? AND v.id != ?
		  AND (
			SELECT state FROM sop_version_states
			WHERE sop_version_id = v.id
			ORDER BY created_at DESC, rowid DESC
			LIMIT 1
		  ) = ?
	`, sopID, excludeVersionID, StateRC).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Replace your existing getSOPVersionByIDRecord with this:
func getSOPVersionByIDRecord(db audit.DBTX, id string) (*SOPVersion, error) {
	const query = `
        SELECT v.id, v.sop_id, v.version, v.content_path, v.content_hash, v.change_summary, v.created_at,
               (SELECT state FROM sop_version_states WHERE sop_version_id = v.id ORDER BY created_at DESC, rowid DESC LIMIT 1) as status
        FROM sop_versions v WHERE v.id = ?`

	var s SOPVersion
	var createdAt string

	err := db.QueryRow(query, id).Scan(
		&s.ID, &s.SOPID, &s.Version, &s.ContentPath, &s.ContentHash, &s.ChangeSummary, &createdAt, &s.Status,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("sop version not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	s.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	return &s, nil
}

func getSOPVersionsBySOPIDRecord(db audit.DBTX, sopID string) ([]SOPVersion, error) {
	const query = `
        SELECT 
            v.id, 
            v.sop_id, 
            v.version, 
            v.content_path, 
            v.content_hash,
            v.change_summary,
            v.created_at,
            (SELECT state FROM sop_version_states WHERE sop_version_id = v.id ORDER BY created_at DESC, rowid DESC LIMIT 1) as status
        FROM sop_versions v 
        WHERE v.sop_id = ? 
        ORDER BY v.created_at DESC`

	rows, err := db.Query(query, sopID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sop versions: %w", err)
	}
	defer rows.Close()

	var sopVersions []SOPVersion
	for rows.Next() {
		var s SOPVersion
		var createdAt string

		// Make sure the scan order exactly matches the SELECT statement above
		err := rows.Scan(
			&s.ID,
			&s.SOPID,
			&s.Version,
			&s.ContentPath,
			&s.ContentHash,
			&s.ChangeSummary,
			&createdAt,
			&s.Status,
		)
		if err != nil {
			return nil, err
		}

		s.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		sopVersions = append(sopVersions, s)
	}

	// Optional: Return an empty slice instead of nil so JSON marshals as [] instead of null
	if sopVersions == nil {
		sopVersions = []SOPVersion{}
	}

	return sopVersions, nil
}

// getLatestPublishedSOPVersionRecord ensures regular Viewers ONLY see published versions.
// It checks if the absolute latest state of the version is 'published'.
func getLatestPublishedSOPVersionRecord(tx *sql.Tx, sopID string) (*SOPVersion, error) {
	var v SOPVersion
	var createdAt string

	// This query finds the highest version number WHERE the latest state entry is 'published'
	query := `
        SELECT v.id, v.sop_id, v.version, v.content_path, v.content_hash, v.change_summary, v.created_at, s.state
        FROM sop_versions v
        JOIN sop_version_states s ON v.id = s.sop_version_id
        WHERE v.sop_id = ? AND s.state = 'published'
        AND s.created_at = (
            SELECT MAX(created_at) FROM sop_version_states WHERE sop_version_id = v.id
        )
        ORDER BY v.version DESC 
        LIMIT 1
    `

	err := tx.QueryRow(query, sopID).Scan(
		&v.ID, &v.SOPID, &v.Version, &v.ContentPath, &v.ContentHash, &v.ChangeSummary, &createdAt, &v.Status,
	)

	if err != nil {
		return nil, err
	}

	v.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	return &v, nil
}

// --- SOP Version States ---

func createSOPVersionStateRecord(db audit.DBTX, id, sopVersionID, state, actorUserID, createdAt string) error {
	_, err := db.Exec(`
        INSERT INTO sop_version_states (id, sop_version_id, state, actor_user_id, created_at)
        VALUES (?, ?, ?, ?, ?)
    `, id, sopVersionID, state, actorUserID, createdAt)
	return err
}

// --- SOP Version PDF Artifacts ---

func createSOPVersionPDFArtifactRecord(
	db audit.DBTX,
	id, sopID, sopVersionID, stage, generatorVersion, filePath, contentHash string,
	sizeBytes int64,
	createdBy, createdAt string,
) error {
	_, err := db.Exec(`
		INSERT INTO sop_version_pdf_artifacts (
			id, sop_id, sop_version_id, stage, generator_version, file_path, content_hash, size_bytes, created_by, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		id, sopID, sopVersionID, stage, generatorVersion, filePath, contentHash, sizeBytes, createdBy, createdAt,
	)
	return err
}

func getSOPVersionPDFArtifactByKeyRecord(db audit.DBTX, sopVersionID, stage, generatorVersion string) (*SOPVersionPDFArtifact, error) {
	const query = `
		SELECT id, sop_id, sop_version_id, stage, generator_version, file_path, content_hash, size_bytes, created_at, created_by
		FROM sop_version_pdf_artifacts
		WHERE sop_version_id = ? AND stage = ? AND generator_version = ?
		LIMIT 1
	`
	var a SOPVersionPDFArtifact
	var createdAt string
	if err := db.QueryRow(query, sopVersionID, stage, generatorVersion).Scan(
		&a.ID, &a.SOPID, &a.SOPVersionID, &a.Stage, &a.GeneratorVersion, &a.FilePath, &a.ContentHash, &a.SizeBytes, &createdAt, &a.CreatedBy,
	); err != nil {
		return nil, err
	}
	a.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	return &a, nil
}

func listPDFEligibleStatesForVersionRecord(db audit.DBTX, sopVersionID string) ([]string, error) {
	rows, err := db.Query(`
		SELECT DISTINCT state
		FROM sop_version_states
		WHERE sop_version_id = ?
		  AND state IN (?, ?, ?)
	`, sopVersionID, StateDraft, StateRC, StatePublished)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var state string
		if err := rows.Scan(&state); err != nil {
			return nil, err
		}
		out = append(out, state)
	}
	return out, nil
}

// --- SOP Asset ---

func createSOPAssetRecord(db audit.DBTX, id, sopID, filename, contentPath, contentHash, createdAt string) error {
	_, err := db.Exec(`
		INSERT INTO sop_assets (
			id,
			sop_id,
			file_name,
			content_path,
			content_hash,
			created_at
		) VALUES (?, ?, ?, ?, ?, ?)
	`,
		id,
		sopID,
		filename,
		contentPath,
		contentHash,
		createdAt,
	)
	return err
}

func getSOPAssetByIDRecord(db audit.DBTX, id string) (*SOPAsset, error) {
	const query = `
		SELECT id, sop_id, file_name, content_path, content_hash, created_at
		FROM sop_assets WHERE id = ?`

	var s SOPAsset
	var createdAt string

	err := db.QueryRow(query, id).Scan(
		&s.ID, &s.SOPID, &s.Filename, &s.ContentPath, &s.ContentHash, &createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("sop not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	s.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	return &s, nil
}

func getSOPAssetsBySOPIDRecord(db audit.DBTX, sopID string) ([]SOPAsset, error) {
	const query = `
		SELECT id, sop_id, file_name, content_path, content_hash, created_at
		FROM sop_assets WHERE sop_id = ? ORDER BY created_at DESC, rowid DESC`

	rows, err := db.Query(query, sopID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sop assets: %w", err)
	}
	defer rows.Close()

	var sopAssets []SOPAsset
	for rows.Next() {
		var s SOPAsset
		var createdAt string

		err := rows.Scan(
			&s.ID, &s.SOPID, &s.Filename, &s.ContentPath, &s.ContentHash, &createdAt,
		)
		if err != nil {
			return nil, err
		}

		s.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		sopAssets = append(sopAssets, s)
	}

	return sopAssets, nil
}

// --- Acknowledgement ---

func createSOPAcknowledgmentRecord(db audit.DBTX, id, sopVersionID, userId, acknowledgmentType, createdAt string) error {
	_, err := db.Exec(`
		INSERT INTO sop_acknowledgments (
			id,
			sop_version_id,
			user_id,
			ack_type,
			created_at
		) VALUES (?, ?, ?, ?, ?)
	`,
		id,
		sopVersionID,
		userId,
		acknowledgmentType,
		createdAt,
	)
	return err
}

func getSOPAcknowledgmentByIDRecord(db audit.DBTX, id string) (*SOPAcknowledgment, error) {
	const query = `
		SELECT id, sop_version_id, user_id, ack_type, created_at
		FROM sop_acknowledgments WHERE id = ?`

	var s SOPAcknowledgment
	var createdAt string

	err := db.QueryRow(query, id).Scan(
		&s.ID, &s.SOPVersionID, &s.UserID, &s.AcknowledgmentType, &createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("sop acknowledgment not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	s.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	return &s, nil
}

func getSOPAcknowledgmentsBySOPVersionIDRecord(db audit.DBTX, sopVersionID string) ([]SOPAcknowledgment, error) {
	const query = `
		SELECT id, sop_version_id, user_id, ack_type, created_at
		FROM sop_acknowledgments WHERE sop_version_id = ? ORDER BY created_at DESC, rowid DESC`

	rows, err := db.Query(query, sopVersionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sop acknowledgments: %w", err)
	}
	defer rows.Close()

	var sopAcknowledgments []SOPAcknowledgment
	for rows.Next() {
		var s SOPAcknowledgment
		var createdAt string

		err := rows.Scan(
			&s.ID, &s.SOPVersionID, &s.UserID, &s.AcknowledgmentType, &createdAt,
		)
		if err != nil {
			return nil, err
		}

		s.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		sopAcknowledgments = append(sopAcknowledgments, s)
	}

	return sopAcknowledgments, nil
}

func getSOPAcknowledgmentsWithUserBySOPVersionIDRecord(db audit.DBTX, sopVersionID string) ([]SOPAcknowledgmentWithUser, error) {
	const query = `
        SELECT 
            a.id, a.sop_version_id, a.user_id, a.ack_type, a.created_at,
            u.id, u.display_name, u.email, u.role_id, u.is_active, u.created_at
        FROM sop_acknowledgments a
        JOIN users u ON a.user_id = u.id
        WHERE a.sop_version_id = ? 
        ORDER BY a.created_at DESC`

	rows, err := db.Query(query, sopVersionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sop acknowledgments with users: %w", err)
	}
	defer rows.Close()

	var results []SOPAcknowledgmentWithUser

	for rows.Next() {
		var r SOPAcknowledgmentWithUser
		var ackCreatedAt string
		var userCreatedAt string

		err := rows.Scan(
			// Acknowledgment fields
			&r.ID,
			&r.SOPVersionID,
			&r.UserID,
			&r.AcknowledgmentType,
			&ackCreatedAt,

			// Nested User fields
			&r.User.ID,
			&r.User.DisplayName,
			&r.User.Email,
			&r.User.Role,
			&r.User.IsActive,
			&userCreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Parse time strings back to time.Time objects
		r.CreatedAt, _ = time.Parse(time.RFC3339Nano, ackCreatedAt)
		r.User.CreatedAt, _ = time.Parse(time.RFC3339Nano, userCreatedAt)

		results = append(results, r)
	}

	return results, nil
}

func getSOPAcknowledgmentsByUserIDRecord(db audit.DBTX, userID string) ([]SOPAcknowledgment, error) {
	const query = `
		SELECT id, sop_version_id, user_id, ack_type, created_at
		FROM sop_acknowledgments WHERE user_id = ? ORDER BY created_at DESC, rowid DESC`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sop acknowledgments: %w", err)
	}
	defer rows.Close()

	var sopAcknowledgments []SOPAcknowledgment
	for rows.Next() {
		var s SOPAcknowledgment
		var createdAt string

		err := rows.Scan(
			&s.ID, &s.SOPVersionID, &s.UserID, &s.AcknowledgmentType, &createdAt,
		)
		if err != nil {
			return nil, err
		}

		s.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		sopAcknowledgments = append(sopAcknowledgments, s)
	}

	return sopAcknowledgments, nil
}

// integrity

func getAllSOPVersionsRecord(db audit.DBTX) ([]SOPVersion, error) {
	// Used for Admin Integrity Check
	const query = `SELECT id, sop_id, version, content_path, content_hash FROM sop_versions`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SOPVersion
	for rows.Next() {
		var s SOPVersion
		if err := rows.Scan(&s.ID, &s.SOPID, &s.Version, &s.ContentPath, &s.ContentHash); err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, nil
}

func getAllSOPAssetsRecord(db audit.DBTX) ([]SOPAsset, error) {
	// Used for Admin Integrity Check
	const query = `SELECT id, sop_id, content_path, content_hash FROM sop_assets`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SOPAsset
	for rows.Next() {
		var s SOPAsset
		if err := rows.Scan(&s.ID, &s.SOPID, &s.ContentPath, &s.ContentHash); err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, nil
}

// getActivePublishedVersionIDRecord finds the currently published version for an SOP container,
// explicitly excluding the version we are currently operating on.
func getActivePublishedVersionIDRecord(db audit.DBTX, sopID string, excludeVersionID string) (string, error) {
	var oldPublishedVersionID string

	query := `
        SELECT v.id
        FROM sop_versions v
        WHERE v.sop_id = ? AND ` + currentVersionStateSQL + ` = 'published'
        AND v.id != ?
        ORDER BY v.version DESC LIMIT 1
    `

	err := db.QueryRow(query, sopID, excludeVersionID).Scan(&oldPublishedVersionID)
	if err != nil {
		return "", err
	}

	return oldPublishedVersionID, nil
}

func getSignatureStatusByUserIDRecord(db audit.DBTX, userID string) ([]UserSignatureStatus, error) {
	const query = `
		WITH RankedPublished AS (
			SELECT
				v.id AS version_id,
				v.sop_id,
				v.version,
				s.title,
				ROW_NUMBER() OVER (PARTITION BY v.sop_id ORDER BY v.version DESC) AS rn
			FROM sop_versions v
			JOIN sops s ON v.sop_id = s.id
			WHERE EXISTS (
				SELECT 1 FROM sop_version_states st
				WHERE st.sop_version_id = v.id
				  AND st.state = 'published'
				  AND st.created_at = (
					  SELECT MAX(created_at)
					  FROM sop_version_states
					  WHERE sop_version_id = v.id
				  )
			)
		),
		LatestPublished AS (
			SELECT version_id, sop_id, version, title
			FROM RankedPublished
			WHERE rn = 1
		),
		UserAcks AS (
			SELECT sop_version_id, MAX(created_at) as created_at
			FROM sop_acknowledgments
			WHERE user_id = ?
			GROUP BY sop_version_id
		)
		SELECT 
			lp.sop_id,
			lp.title,
			lp.version_id,
			lp.version,
			CASE WHEN ua_latest.sop_version_id IS NOT NULL THEN 1 ELSE 0 END as has_signed_latest,
			CASE WHEN EXISTS (
				SELECT 1 FROM sop_versions old_v 
				JOIN UserAcks ua_old ON old_v.id = ua_old.sop_version_id
				WHERE old_v.sop_id = lp.sop_id
			) THEN 1 ELSE 0 END as signed_older_version,
			ua_latest.created_at as last_sign_date
		FROM LatestPublished lp
		LEFT JOIN UserAcks ua_latest ON lp.version_id = ua_latest.sop_version_id
		ORDER BY lp.title ASC
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get signature status: %w", err)
	}
	defer rows.Close()

	var statuses []UserSignatureStatus
	for rows.Next() {
		var s UserSignatureStatus
		var lastSignDate sql.NullString
		
		err := rows.Scan(
			&s.SOPID,
			&s.Title,
			&s.LatestVersionID,
			&s.LatestVersion,
			&s.HasSignedLatest,
			&s.SignedOlderVersion,
			&lastSignDate,
		)
		if err != nil {
			return nil, err
		}

		if lastSignDate.Valid {
			dateStr := lastSignDate.String
			s.LastSignDate = &dateStr
		}

		statuses = append(statuses, s)
	}

	if statuses == nil {
		statuses = []UserSignatureStatus{}
	}

	return statuses, nil
}

func addSOPFavoriteRecord(db audit.DBTX, userID, sopID, createdAt string) (int64, error) {
	res, err := db.Exec(`
		INSERT INTO sop_favorites (user_id, sop_id, created_at)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id, sop_id) DO NOTHING
	`, userID, sopID, createdAt)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func removeSOPFavoriteRecord(db audit.DBTX, userID, sopID string) (int64, error) {
	res, err := db.Exec(`DELETE FROM sop_favorites WHERE user_id = ? AND sop_id = ?`, userID, sopID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func isSOPFavoritedRecord(db audit.DBTX, userID, sopID string) (bool, error) {
	if userID == "" {
		return false, nil
	}
	var one int
	err := db.QueryRow(`
		SELECT 1 FROM sop_favorites WHERE user_id = ? AND sop_id = ? LIMIT 1
	`, userID, sopID).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// previousComparisonVersionIDRecord prefers the newest older published version,
// then any older version, so a new draft is compared with what the lab last published.
func previousComparisonVersionIDRecord(db audit.DBTX, sopID string, versionNum int) (string, error) {
	const publishedQuery = `
		SELECT v.id
		FROM sop_versions v
		WHERE v.sop_id = ? AND v.version < ?
		  AND (
			SELECT state FROM sop_version_states
			WHERE sop_version_id = v.id
			ORDER BY created_at DESC, rowid DESC
			LIMIT 1
		  ) = 'published'
		ORDER BY v.version DESC
		LIMIT 1
	`
	var id string
	err := db.QueryRow(publishedQuery, sopID, versionNum).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}

	err = db.QueryRow(`
		SELECT id FROM sop_versions
		WHERE sop_id = ? AND version < ?
		ORDER BY version DESC
		LIMIT 1
	`, sopID, versionNum).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return id, nil
}

func listRecentPublishesRecord(db audit.DBTX, limit int) ([]PublishedActivity, error) {
	const query = `
		SELECT s.id, s.title, v.id, v.version, v.change_summary, st.created_at,
		       COALESCE(u.display_name, '')
		FROM sop_version_states st
		JOIN sop_versions v ON v.id = st.sop_version_id
		JOIN sops s ON s.id = v.sop_id
		LEFT JOIN users u ON u.id = st.actor_user_id
		WHERE st.state = 'published'
		  AND st.rowid = (
			SELECT st2.rowid
			FROM sop_version_states st2
			WHERE st2.sop_version_id = v.id AND st2.state = 'published'
			ORDER BY st2.created_at DESC, st2.rowid DESC
			LIMIT 1
		  )
		  AND (
			SELECT state FROM sop_version_states
			WHERE sop_version_id = v.id
			ORDER BY created_at DESC, rowid DESC
			LIMIT 1
		  ) = 'published'
		ORDER BY st.created_at DESC
		LIMIT ?
	`
	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list recent publishes: %w", err)
	}
	defer rows.Close()

	var items []PublishedActivity
	for rows.Next() {
		var item PublishedActivity
		var publishedAt string
		if err := rows.Scan(
			&item.SOPID,
			&item.Title,
			&item.VersionID,
			&item.Version,
			&item.ChangeSummary,
			&publishedAt,
			&item.PublishedBy,
		); err != nil {
			return nil, err
		}
		item.PublishedAt, _ = time.Parse(time.RFC3339Nano, publishedAt)
		items = append(items, item)
	}
	if items == nil {
		items = []PublishedActivity{}
	}
	return items, nil
}

type publishedVersionRow struct {
	SOPID     string
	Title     string
	VersionID string
	Version   int
}

func listCurrentlyPublishedVersionsRecord(db audit.DBTX, sopID string) ([]publishedVersionRow, error) {
	query := `
		SELECT s.id, s.title, v.id, v.version
		FROM sop_versions v
		JOIN sops s ON s.id = v.sop_id
		WHERE (
			SELECT state FROM sop_version_states
			WHERE sop_version_id = v.id
			ORDER BY created_at DESC, rowid DESC
			LIMIT 1
		) = 'published'
	`
	args := []any{}
	if sopID != "" {
		query += ` AND v.sop_id = ?`
		args = append(args, sopID)
	}
	query += ` ORDER BY s.title ASC`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []publishedVersionRow
	for rows.Next() {
		var row publishedVersionRow
		if err := rows.Scan(&row.SOPID, &row.Title, &row.VersionID, &row.Version); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, nil
}

func listReaderAcksByVersionIDsRecord(db audit.DBTX, versionIDs []string) (map[string]map[string]string, error) {
	result := map[string]map[string]string{}
	if len(versionIDs) == 0 {
		return result, nil
	}
	placeholders := make([]string, len(versionIDs))
	args := make([]any, len(versionIDs))
	for i, id := range versionIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	query := `
		SELECT sop_version_id, user_id, created_at
		FROM sop_acknowledgments
		WHERE ack_type = ? AND sop_version_id IN (` + strings.Join(placeholders, ",") + `)
	`
	args = append([]any{AckTypeRead}, args...)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var versionID, userID, createdAt string
		if err := rows.Scan(&versionID, &userID, &createdAt); err != nil {
			return nil, err
		}
		if result[versionID] == nil {
			result[versionID] = map[string]string{}
		}
		result[versionID][userID] = createdAt
	}
	return result, nil
}
