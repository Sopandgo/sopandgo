package sop

import (
	"log"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/integrity"
	"github.com/sopandgo/sopandgo/backend/internal/storage"
)

func (s *Service) RunSystemIntegrityCheck() (*SystemIntegrityReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	report := &SystemIntegrityReport{
		CheckedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}

	// 1. Check SOP Versions
	versions, err := getAllSOPVersionsRecord(s.db)
	if err != nil {
		return nil, err
	}

	for _, v := range versions {
		report.Versions.Checked++

		fullPath, err := storage.SafeJoin(s.dataDir, v.ContentPath)

		// Helper closure to add failure
		addFailure := func(msg string) {
			report.Versions.Failed = append(report.Versions.Failed, IntegrityFailure{
				SOPID:   v.SOPID,
				Version: v.Version,
				Error:   msg,
			})
		}

		if err != nil {
			addFailure("unsafe path")
			continue
		}

		valid, err := integrity.VerifyFile(fullPath, v.ContentHash)
		if err != nil || !valid {
			log.Printf("Integrity Fail: Version %d (SOP %s)", v.Version, v.SOPID)
			addFailure("hash mismatch or missing file")
			continue
		}
		report.Versions.Valid++
	}
	report.Versions.Invalid = report.Versions.Checked - report.Versions.Valid

	// 2. Check Assets
	assets, err := getAllSOPAssetsRecord(s.db)
	if err != nil {
		return nil, err
	}

	for _, a := range assets {
		report.Assets.Checked++

		fullPath, err := storage.SafeJoin(s.dataDir, a.ContentPath)

		// Helper closure to add failure
		addFailure := func(msg string) {
			report.Assets.Failed = append(report.Assets.Failed, IntegrityFailure{
				SOPID:   a.SOPID,
				AssetID: a.ID,
				Error:   msg,
			})
		}

		if err != nil {
			addFailure("unsafe path")
			continue
		}

		valid, err := integrity.VerifyFile(fullPath, a.ContentHash)
		if err != nil || !valid {
			log.Printf("Integrity Fail: Asset %s (SOP %s)", a.ID, a.SOPID)
			addFailure("hash mismatch or missing file")
			continue
		}
		report.Assets.Valid++
	}
	report.Assets.Invalid = report.Assets.Checked - report.Assets.Valid

	// 3. Final Status
	report.OK = (report.Versions.Invalid == 0) && (report.Assets.Invalid == 0)

	return report, nil
}
