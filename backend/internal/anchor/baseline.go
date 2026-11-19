package anchor

import (
	"database/sql"
	"encoding/json"
	"sort"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
)

// Baseline is the integrity_baseline payload written each time anchoring is
// switched on. It puts the hash of every SOP version and asset that already
// exists into the audit chain, so the first anchor covers content created
// before version and acknowledgment events carried content hashes. It also
// records chain rows that already failed verification, which the admin
// confirmed, so later scans can tell those known breaks from new ones.
type Baseline struct {
	InstanceID  string               `json:"instance_id"`
	Versions    []BaselineVersion    `json:"versions"`
	Assets      []BaselineAsset      `json:"assets"`
	KnownBroken []audit.ChainFailure `json:"known_broken"`
}

type BaselineVersion struct {
	ID          string `json:"id"`
	SOPID       string `json:"sop_id"`
	Version     int    `json:"version"`
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
}

type BaselineAsset struct {
	ID          string `json:"id"`
	SOPID       string `json:"sop_id"`
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
}

type querier interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

func buildBaseline(db querier, instanceID string, knownBroken []audit.ChainFailure) (Baseline, error) {
	b := Baseline{InstanceID: instanceID, Versions: []BaselineVersion{}, Assets: []BaselineAsset{}, KnownBroken: knownBroken}
	if b.KnownBroken == nil {
		b.KnownBroken = []audit.ChainFailure{}
	}

	rows, err := db.Query(`SELECT id, sop_id, version, content_path, content_hash FROM sop_versions ORDER BY id`)
	if err != nil {
		return Baseline{}, err
	}
	for rows.Next() {
		var v BaselineVersion
		if err := rows.Scan(&v.ID, &v.SOPID, &v.Version, &v.Path, &v.ContentHash); err != nil {
			rows.Close()
			return Baseline{}, err
		}
		b.Versions = append(b.Versions, v)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Baseline{}, err
	}

	rows, err = db.Query(`SELECT id, sop_id, content_path, content_hash FROM sop_assets ORDER BY id`)
	if err != nil {
		return Baseline{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var a BaselineAsset
		if err := rows.Scan(&a.ID, &a.SOPID, &a.Path, &a.ContentHash); err != nil {
			return Baseline{}, err
		}
		b.Assets = append(b.Assets, a)
	}
	return b, rows.Err()
}

// knownBroken returns every chain row an admin accepted as already broken
// when switching anchoring on, across all baselines.
func knownBroken(db querier) (map[int64]string, error) {
	rows, err := db.Query(`SELECT payload FROM audit_events WHERE event_type = ? ORDER BY id`, audit.EventIntegrityBaseline)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]string{}
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var b struct {
			KnownBroken []audit.ChainFailure `json:"known_broken"`
		}
		if json.Unmarshal([]byte(payload), &b) != nil {
			continue
		}
		for _, f := range b.KnownBroken {
			out[f.EventID] = f.Error
		}
	}
	return out, rows.Err()
}

// unexpectedFailures drops failures an admin already accepted.
func unexpectedFailures(failed []audit.ChainFailure, known map[int64]string) []audit.ChainFailure {
	var out []audit.ChainFailure
	for _, f := range failed {
		if _, ok := known[f.EventID]; !ok {
			out = append(out, f)
		}
	}
	return out
}

// sameFailureIDs reports whether failed lists exactly the ids in accepted.
func sameFailureIDs(failed []audit.ChainFailure, accepted []int64) bool {
	a := make([]int64, 0, len(failed))
	for _, f := range failed {
		a = append(a, f.EventID)
	}
	b := append([]int64(nil), accepted...)
	sort.Slice(a, func(i, j int) bool { return a[i] < a[j] })
	sort.Slice(b, func(i, j int) bool { return b[i] < b[j] })
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
