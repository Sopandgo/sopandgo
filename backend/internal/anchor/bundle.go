package anchor

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// bundleEvent is one line of audit_events.jsonl: every column the row hash
// covers, exactly as stored, so an auditor can recompute the chain.
type bundleEvent struct {
	ID          int64   `json:"id"`
	EventType   string  `json:"event_type"`
	EntityType  string  `json:"entity_type"`
	EntityID    string  `json:"entity_id"`
	ActorUserID *string `json:"actor_user_id"`
	Payload     string  `json:"payload"`
	CreatedAt   string  `json:"created_at"`
	Hash        string  `json:"hash"`
	PrevHash    string  `json:"prev_hash"`
	HashVersion int     `json:"hash_version"`
}

const bundleReadme = `SOPandGO audit evidence bundle
==============================

Contents
  audit_events.jsonl   every audit row in chain order, exactly as stored
  anchors/*.tsr        RFC 3161 TimeStampResp receipts (DER)
  anchors/*.json       which audit row and hash each receipt covers
  tsa-ca.pem           the TSA CA certificate(s) SOPandGO pinned

Check it without SOPandGO
  1. Get your TSA's CA certificate from the TSA itself (its website or
     trust list), not only from this bundle, and compare it with tsa-ca.pem.

  2. Check each receipt with OpenSSL, using the tip_hash from its .json:

       openssl ts -verify -digest <tip_hash> -in anchors/<name>.tsr \
         -CAfile tsa-ca.pem -partial_chain

  3. Recompute the chain and match it to the receipts:

       python3 -I verify_anchors.py <this-bundle.zip>

     The script and the hash algorithm are described in SOPandGO's
     docs/ops/verifying-anchors.md.

A receipt proves the audit history up to its row existed, unchanged, at the
time the TSA signed it. Rows written after the newest receipt are not covered.
`

// WriteBundle streams the evidence bundle zip to w.
func (s *Service) WriteBundle(w io.Writer) error {
	st := s.snapshot()
	events, err := s.logger.ChainEvents()
	if err != nil {
		return err
	}
	anchors, err := s.store.List()
	if err != nil {
		return err
	}

	zw := zip.NewWriter(w)
	now := time.Now().UTC()
	create := func(name string) (io.Writer, error) {
		return zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate, Modified: now})
	}

	f, err := create("README.txt")
	if err != nil {
		return err
	}
	if _, err := io.WriteString(f, bundleReadme); err != nil {
		return err
	}

	f, err = create("audit_events.jsonl")
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	for _, e := range events {
		if err := enc.Encode(bundleEvent{
			ID: e.ID, EventType: e.EventType, EntityType: e.EntityType, EntityID: e.EntityID,
			ActorUserID: e.ActorID, Payload: e.Payload, CreatedAt: e.CreatedAtRaw(),
			Hash: e.Hash, PrevHash: e.PrevHash, HashVersion: e.HashVersion,
		}); err != nil {
			return err
		}
	}

	var cas []string
	seenCA := map[string]bool{}
	for _, pem := range append([]string{st.PinnedCAPEM}, pemsOf(anchors)...) {
		if pem != "" && !seenCA[pem] {
			seenCA[pem] = true
			cas = append(cas, pem)
		}
	}
	f, err = create("tsa-ca.pem")
	if err != nil {
		return err
	}
	if _, err := io.WriteString(f, strings.Join(cas, "")); err != nil {
		return err
	}

	for _, a := range anchors {
		base := "anchors/" + receiptBaseName(a)
		f, err := create(base + ".tsr")
		if err != nil {
			return err
		}
		if _, err := f.Write(a.TSR); err != nil {
			return err
		}
		side, err := json.MarshalIndent(sidecarFor(st.InstanceID, a), "", "  ")
		if err != nil {
			return err
		}
		f, err = create(base + ".json")
		if err != nil {
			return err
		}
		if _, err := f.Write(side); err != nil {
			return err
		}
	}
	return zw.Close()
}

func pemsOf(anchors []Anchor) []string {
	out := make([]string, 0, len(anchors))
	for _, a := range anchors {
		out = append(out, a.TSACAPEM)
	}
	return out
}

// BundleFileName is the download name for the evidence bundle.
func BundleFileName(now time.Time) string {
	return fmt.Sprintf("sopandgo-audit-evidence-%s.zip", now.UTC().Format("20060102-150405"))
}
