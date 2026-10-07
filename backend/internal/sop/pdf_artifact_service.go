package sop

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/i18n"
	"github.com/sopandgo/sopandgo/backend/internal/integrity"
	"github.com/sopandgo/sopandgo/backend/internal/pdfgen"
	"github.com/sopandgo/sopandgo/backend/internal/storage"
)

var ErrPDFExportDisabled = fmt.Errorf("pdf export disabled")

func (s *Service) ensureVersionPDFArtifact(versionID, stage, actorUserID string) error {
	if !s.IsPDFExportEnabled() {
		return nil
	}
	if actorUserID == "" {
		return fmt.Errorf("actor user id is required for pdf artifact creation")
	}
	if s.pdfGeneratorVersion == "" {
		return fmt.Errorf("pdf generator version is not configured")
	}

	// Hard no-op for existing artifact key (append-only, no overwrite).
	existing, err := getSOPVersionPDFArtifactByKeyRecord(s.db, versionID, stage, s.pdfGeneratorVersion)
	if err == nil && existing != nil {
		return nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check pdf artifact existence: %w", err)
	}

	version, err := getSOPVersionByIDRecord(s.db, versionID)
	if err != nil {
		return fmt.Errorf("failed to load sop version: %w", err)
	}
	sopMeta, err := getSOPByIDRecord(s.db, version.SOPID)
	if err != nil {
		return fmt.Errorf("failed to load sop metadata: %w", err)
	}
	absVersionPath, err := storage.SafeJoin(s.dataDir, version.ContentPath)
	if err != nil {
		return fmt.Errorf("invalid version path: %w", err)
	}
	contentBytes, err := os.ReadFile(absVersionPath)
	if err != nil {
		return fmt.Errorf("failed to read version content: %w", err)
	}
	assets, err := getSOPAssetsBySOPIDRecord(s.db, version.SOPID)
	if err != nil {
		return fmt.Errorf("failed to load sop assets: %w", err)
	}
	contentForPDF, err := s.embedLocalAssetReferences(string(contentBytes), assets)
	if err != nil {
		return fmt.Errorf("failed to embed sop assets: %w", err)
	}

	pdfBytes, err := s.pdfGenerator.Render(pdfgen.Input{
		Markdown: contentForPDF,
		Meta: pdfgen.Metadata{
			SOPTitle:         sopMeta.Title,
			SOPID:            version.SOPID,
			SOPVersionID:     versionID,
			SOPVersionNumber: version.Version,
			Stage:            stage,
			ContentHash:      version.ContentHash,
			GeneratorVersion: s.pdfGeneratorVersion,
			GeneratedAt:      time.Now().UTC(),
			Locale:           i18n.ReadDefaultLocale(s.db),
		},
	})
	if err != nil {
		return fmt.Errorf("pdf generation failed: %w", err)
	}

	fileName := fmt.Sprintf("%s__v%d__%s__g%s.pdf",
		slugify(sopMeta.Title),
		version.Version,
		stage,
		slugify(s.pdfGeneratorVersion),
	)

	relPath := filepath.ToSlash(filepath.Join(
		"sops",
		version.SOPID,
		"exports",
		"pdf",
		fmt.Sprintf("version-%d", version.Version),
		stage,
		"g"+s.pdfGeneratorVersion,
		fileName,
	))

	absPath, err := storage.SafeJoin(s.dataDir, relPath)
	if err != nil {
		return fmt.Errorf("invalid path construction: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return fmt.Errorf("failed to create pdf directory: %w", err)
	}

	f, err := os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil
		}
		return fmt.Errorf("failed to create pdf file: %w", err)
	}
	if _, err := f.Write(pdfBytes); err != nil {
		f.Close()
		return fmt.Errorf("failed writing pdf file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed closing pdf file: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	artifactID := uuid.NewString()
	pdfHash := integrity.HashBytes(pdfBytes)
	if err := createSOPVersionPDFArtifactRecord(
		s.db,
		artifactID,
		version.SOPID,
		versionID,
		stage,
		s.pdfGeneratorVersion,
		relPath,
		pdfHash,
		int64(len(pdfBytes)),
		actorUserID,
		now,
	); err != nil {
		_ = os.Remove(absPath)
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil
		}
		return fmt.Errorf("failed to persist pdf artifact: %w", err)
	}

	if err := s.auditLogger.Log(
		s.db,
		audit.EventSOPVersionPDFArtifactCreated,
		audit.EntitySOPVersion,
		versionID,
		&actorUserID,
		map[string]any{
			"stage":             stage,
			"generator_version": s.pdfGeneratorVersion,
			"artifact_id":       artifactID,
			"file_path":         relPath,
		},
	); err != nil {
		log.Printf("audit write failed for PDF artifact %s: %v", artifactID, err)
	}
	return nil
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func (s *Service) GetVersionPDFArtifactPath(versionID string, stage string) (string, string, error) {
	if !s.IsPDFExportEnabled() {
		return "", "", ErrPDFExportDisabled
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	version, err := getSOPVersionByIDRecord(s.db, versionID)
	if err != nil {
		return "", "", err
	}
	if stage == "" {
		stage = version.Status
	}

	artifact, err := getSOPVersionPDFArtifactByKeyRecord(s.db, versionID, stage, s.pdfGeneratorVersion)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", fmt.Errorf("pdf artifact not found")
		}
		return "", "", err
	}

	absPath, err := storage.SafeJoin(s.dataDir, artifact.FilePath)
	if err != nil {
		return "", "", err
	}
	return absPath, artifact.GeneratorVersion, nil
}

func (s *Service) BackfillPDFArtifactsForGenerator(actorUserID string) (int, int, error) {
	if !s.IsPDFExportEnabled() {
		return 0, 0, nil
	}
	if actorUserID == "" {
		return 0, 0, fmt.Errorf("actor user id is required for backfill")
	}

	versions, err := getAllSOPVersionsRecord(s.db)
	if err != nil {
		return 0, 0, err
	}

	processed := 0
	generated := 0
	for _, v := range versions {
		states, err := listPDFEligibleStatesForVersionRecord(s.db, v.ID)
		if err != nil {
			return processed, generated, err
		}

		for _, stage := range states {
			processed++
			_, err := getSOPVersionPDFArtifactByKeyRecord(s.db, v.ID, stage, s.pdfGeneratorVersion)
			if err == nil {
				continue
			}
			if !errors.Is(err, sql.ErrNoRows) {
				return processed, generated, err
			}
			if err := s.ensureVersionPDFArtifact(v.ID, stage, actorUserID); err != nil {
				return processed, generated, err
			}
			generated++
		}
	}
	return processed, generated, nil
}

func slugify(in string) string {
	v := strings.ToLower(strings.TrimSpace(in))
	v = slugRe.ReplaceAllString(v, "-")
	v = strings.Trim(v, "-")
	if v == "" {
		return "sop"
	}
	return v
}

var markdownImageRefRegex = regexp.MustCompile(`!\[[^\]]*\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)
var htmlImageSrcRegex = regexp.MustCompile(`(?i)(<img[^>]+src=["'])([^"']+)(["'])`)

func (s *Service) embedLocalAssetReferences(markdown string, assets []SOPAsset) (string, error) {
	assetDataByRef := make(map[string]string, len(assets))

	for _, asset := range assets {
		assetAbsPath, err := storage.SafeJoin(s.dataDir, asset.ContentPath)
		if err != nil {
			return "", err
		}
		raw, err := os.ReadFile(assetAbsPath)
		if err != nil {
			return "", err
		}
		mimeType := "application/octet-stream"
		if len(raw) > 0 {
			mimeType = strings.SplitN(http.DetectContentType(raw), ";", 2)[0]
		}
		dataURL := "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(raw)
		assetDataByRef[filepath.ToSlash(path.Join("assets", asset.Filename))] = dataURL
		assetDataByRef[filepath.ToSlash(path.Join("./assets", asset.Filename))] = dataURL
	}

	resolveAssetDataURL := func(rawRef string) (string, bool) {
		decodedRef, err := url.QueryUnescape(strings.TrimSpace(rawRef))
		if err != nil {
			return "", false
		}
		cleanRef := filepath.ToSlash(path.Clean(decodedRef))
		cleanRef = strings.TrimPrefix(cleanRef, "/")
		if !strings.HasPrefix(cleanRef, "assets/") {
			return "", false
		}
		dataURL, ok := assetDataByRef[cleanRef]
		return dataURL, ok
	}

	out := markdownImageRefRegex.ReplaceAllStringFunc(markdown, func(fullMatch string) string {
		parts := markdownImageRefRegex.FindStringSubmatch(fullMatch)
		if len(parts) < 2 {
			return fullMatch
		}
		rawRef := parts[1]
		dataURL, ok := resolveAssetDataURL(rawRef)
		if !ok {
			return fullMatch
		}
		return strings.Replace(fullMatch, rawRef, dataURL, 1)
	})

	out = htmlImageSrcRegex.ReplaceAllStringFunc(out, func(fullMatch string) string {
		parts := htmlImageSrcRegex.FindStringSubmatch(fullMatch)
		if len(parts) != 4 {
			return fullMatch
		}
		dataURL, ok := resolveAssetDataURL(parts[2])
		if !ok {
			return fullMatch
		}
		return parts[1] + dataURL + parts[3]
	})

	return out, nil
}
