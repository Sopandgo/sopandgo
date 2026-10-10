package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/sop"
	"github.com/sopandgo/sopandgo/backend/internal/storage"
	_ "modernc.org/sqlite"
)

// seedDemoEnabled reports whether demo users and the sample SOP should be
// inserted. Only the value "false" (case-insensitive, surrounding space
// ignored) turns seeding off. Empty or unset means seed.
func seedDemoEnabled(value string) bool {
	return !strings.EqualFold(strings.TrimSpace(value), "false")
}

// Demo user keys, as used in sop.json ("read_by", "favorited_by").
const (
	userManager    = "manager"
	userQA         = "qa"
	userResearcher = "researcher"
	demoTag        = "Demo"
)

// withDemoTag returns tags with the Demo tag first. Manifest entries that
// already spell Demo (any case) are dropped so the tag is attached once.
func withDemoTag(tags []string) []string {
	out := make([]string, 0, len(tags)+1)
	out = append(out, demoTag)
	for _, tag := range tags {
		if strings.EqualFold(tag, demoTag) {
			continue
		}
		out = append(out, tag)
	}
	return out
}

type demoUser struct {
	key   string
	name  string
	email string
	role  string
}

var demoUsers = []demoUser{
	{userManager, "Demo Lab Manager", "manager@demo.local", auth.RoleEditor},
	{userQA, "Demo QA Officer", "qa@demo.local", auth.RoleApprover},
	{userResearcher, "Demo Research Assistant", "researcher@demo.local", auth.RoleViewer},
}

// demoManifest is the optional sop.json next to an SOP's version-N.md files.
// See backend/demo/README.md for the format.
type demoManifest struct {
	// Tags attached to the SOP. Created on first use.
	Tags []string `json:"tags"`
	// FinalState is where the last version stops: "published" (default),
	// "rc", or "draft". Earlier versions are always published.
	FinalState string `json:"final_state"`
	// ChangeSummaries holds one summary per version, in file order.
	ChangeSummaries []string `json:"change_summaries"`
	// ReadBy maps a demo user key to the version numbers that user signed.
	ReadBy map[string][]int `json:"read_by"`
	// FavoritedBy lists demo user keys that favorite the SOP.
	FavoritedBy []string `json:"favorited_by"`
}

func main() {
	if !seedDemoEnabled(os.Getenv("SEED_DEMO_DATA")) {
		log.Println("SEED_DEMO_DATA=false, skipping demo data")
		return
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	if err := seed(dataDir, filepath.Join("demo", "sops")); err != nil {
		log.Fatal(err)
	}
}

// seed inserts demo users and every SOP under sopsRoot into dataDir. It does
// nothing when dataDir already holds app.db.
func seed(dataDir, sopsRoot string) error {
	log.Println("seeding demo data")

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("cannot create data directory: %w", err)
	}

	// If the database file already exists, we skip seeding entirely.
	if _, err := os.Stat(filepath.Join(dataDir, "app.db")); err == nil {
		log.Println("database already exists, skipping seeding")
		return nil
	}

	store, err := storage.Open(dataDir)
	if err != nil {
		return fmt.Errorf("cannot open storage: %w", err)
	}
	defer store.DB.Close()

	auditLogger, err := audit.New(store.DB)
	if err != nil {
		return fmt.Errorf("cannot init audit logger: %w", err)
	}

	sopService := sop.NewService(store.DB, auditLogger, dataDir, "1", false, nil)
	authService := auth.NewService(store.DB, auditLogger, dataDir)

	avatarsDir := filepath.Join(filepath.Dir(sopsRoot), "avatars")
	userIDs, err := ensureDemoUsers(authService, avatarsDir)
	if err != nil {
		return fmt.Errorf("failed to create demo users: %w", err)
	}

	sopFolders, err := os.ReadDir(sopsRoot)
	if err != nil {
		return fmt.Errorf("failed to read sops root directory %s: %w", sopsRoot, err)
	}

	// A broken SOP folder does not stop the others from seeding.
	var failures []error
	tagIDs := map[string]string{}
	for _, f := range sopFolders {
		if !f.IsDir() {
			continue
		}

		sopDir := filepath.Join(sopsRoot, f.Name())
		if err := seedOneSOP(sopService, sopDir, userIDs, tagIDs); err != nil {
			log.Printf("ERR: failed seeding %s: %v", f.Name(), err)
			failures = append(failures, fmt.Errorf("%s: %w", f.Name(), err))
		}
	}

	log.Println("demo data seeding process finished")
	return errors.Join(failures...)
}

// ensureDemoUsers creates the demo users and returns their IDs by user key.
// When avatarsDir contains <key>.jpg (manager.jpg, qa.jpg, researcher.jpg),
// that file is set as the user's profile picture. The bootstrap admin is never
// given a picture here.
func ensureDemoUsers(authService *auth.Service, avatarsDir string) (map[string]string, error) {
	ids := map[string]string{}

	for _, d := range demoUsers {
		// Just create the user. No need to check "if exists".
		id, err := authService.RegisterUser(
			d.name,
			d.email,
			d.role,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to register demo user %s: %w", d.name, err)
		}

		if err := authService.UpdateUserPassword(id, "12345", nil); err != nil {
			return nil, fmt.Errorf("failed to set bootstrap password: %w", err)
		}

		avatarPath := filepath.Join(avatarsDir, d.key+".jpg")
		if raw, err := os.ReadFile(avatarPath); err == nil {
			if err := authService.SetAvatar(id, raw, nil); err != nil {
				return nil, fmt.Errorf("failed to set avatar for %s: %w", d.key, err)
			}
			log.Printf("created demo user: %s (%s) with avatar", d.name, d.role)
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("failed to read avatar %s: %w", avatarPath, err)
		} else {
			log.Printf("created demo user: %s (%s)", d.name, d.role)
		}
		ids[d.key] = id
	}

	return ids, nil
}

func isDemoUserKey(key string) bool {
	return slices.ContainsFunc(demoUsers, func(d demoUser) bool {
		return d.key == key
	})
}

// loadManifest reads sop.json from sopDir. A missing file yields the defaults:
// every version published, no tags, no reader signatures, no favorites.
func loadManifest(sopDir string, versionCount int) (demoManifest, error) {
	b, err := os.ReadFile(filepath.Join(sopDir, "sop.json"))
	if errors.Is(err, os.ErrNotExist) {
		return parseManifest(nil, versionCount)
	}
	if err != nil {
		return demoManifest{}, err
	}
	return parseManifest(b, versionCount)
}

// parseManifest decodes and validates a sop.json body. A nil body yields defaults.
func parseManifest(b []byte, versionCount int) (demoManifest, error) {
	var m demoManifest
	if b != nil {
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&m); err != nil {
			return demoManifest{}, fmt.Errorf("invalid sop.json: %w", err)
		}
	}

	if versionCount < 1 {
		return demoManifest{}, fmt.Errorf("no version-N.md files found")
	}

	switch m.FinalState {
	case "":
		m.FinalState = sop.StatePublished
	case sop.StatePublished, sop.StateRC, sop.StateDraft:
	default:
		return demoManifest{}, fmt.Errorf("final_state %q must be published, rc, or draft", m.FinalState)
	}

	if len(m.ChangeSummaries) > versionCount {
		return demoManifest{}, fmt.Errorf("%d change_summaries for %d versions", len(m.ChangeSummaries), versionCount)
	}

	lastPublished := versionCount
	if m.FinalState != sop.StatePublished {
		lastPublished = versionCount - 1
	}
	for key, versions := range m.ReadBy {
		if !isDemoUserKey(key) {
			return demoManifest{}, fmt.Errorf("read_by: unknown demo user %q", key)
		}
		for _, v := range versions {
			if v < 1 || v > lastPublished {
				return demoManifest{}, fmt.Errorf("read_by %s: version %d is never published", key, v)
			}
		}
	}

	for _, key := range m.FavoritedBy {
		if !isDemoUserKey(key) {
			return demoManifest{}, fmt.Errorf("favorited_by: unknown demo user %q", key)
		}
	}

	return m, nil
}

func seedOneSOP(
	sopService *sop.Service,
	sopDir string,
	userIDs map[string]string,
	tagIDs map[string]string,
) error {
	log.Printf("processing SOP from %s", sopDir)

	editorID := userIDs[userManager]
	approverID := userIDs[userQA]

	entries, err := os.ReadDir(sopDir)
	if err != nil {
		return err
	}

	var mdFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			mdFiles = append(mdFiles, e.Name())
		}
	}
	sort.Strings(mdFiles) // Ensures v1 is created before v2

	manifest, err := loadManifest(sopDir, len(mdFiles))
	if err != nil {
		return err
	}

	// 1. find version-1.md to get the title ---
	version1Path := filepath.Join(sopDir, mdFiles[0])
	title, err := extractTitleFromMarkdown(version1Path)
	if err != nil {
		return fmt.Errorf("failed to extract SOP title from %s: %w", version1Path, err)
	}

	// 2. create the SOP (We can attribute the container creation to the Editor)
	sopID, err := sopService.RegisterSOP(title, &editorID)
	if err != nil {
		return err
	}

	// 3. attach tags, creating each one the first time it is used.
	// Every demo SOP gets the Demo tag so evaluators can filter the library.
	for _, tag := range withDemoTag(manifest.Tags) {
		tagID, ok := tagIDs[tag]
		if !ok {
			tagID, err = sopService.CreateTag(tag, &editorID)
			if err != nil {
				return fmt.Errorf("failed to create tag %q: %w", tag, err)
			}
			tagIDs[tag] = tagID
		}
		if err := sopService.AttachTagToSOP(sopID, tagID, &editorID); err != nil {
			return err
		}
	}

	// 4. add assets (images/diagrams) ---
	assetsDir := filepath.Join(sopDir, "assets")
	if entries, err := os.ReadDir(assetsDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			content, err := os.ReadFile(filepath.Join(assetsDir, e.Name()))
			if err != nil {
				return err
			}
			if _, err := sopService.AddAsset(sopID, e.Name(), content, &editorID); err != nil {
				return err
			}
		}
	}

	// 5. add versions; the last one stops at manifest.FinalState
	for i, f := range mdFiles {
		versionNumber := i + 1
		isLast := versionNumber == len(mdFiles)

		content, err := os.ReadFile(filepath.Join(sopDir, f))
		if err != nil {
			return err
		}

		summary := "Updated procedure"
		if i < len(manifest.ChangeSummaries) {
			summary = manifest.ChangeSummaries[i]
		}

		// STEP A: Editor creates the Draft
		vID, _, err := sopService.RegisterSOPVersion(sopID, string(content), summary, &editorID)
		if err != nil {
			return err
		}
		if isLast && manifest.FinalState == sop.StateDraft {
			log.Printf("  -> seeded draft version: %s", f)
			break
		}

		// STEP B: Editor promotes Draft -> RC
		if err := sopService.TransitionVersionState(vID, sop.StateRC, editorID); err != nil {
			return fmt.Errorf("failed to promote to RC: %w", err)
		}
		if isLast && manifest.FinalState == sop.StateRC {
			log.Printf("  -> seeded release candidate: %s", f)
			break
		}

		// STEP C: Approver publishes RC -> Published. This also records the
		// approver acknowledgment.
		if _, err := sopService.ApproveSOPVersion(vID, approverID); err != nil {
			return fmt.Errorf("failed to publish: %w", err)
		}
		log.Printf("  -> seeded and published version: %s", f)

		// STEP D: Readers sign while this version is the published one.
		for _, d := range demoUsers {
			if !slices.Contains(manifest.ReadBy[d.key], versionNumber) {
				continue
			}
			if _, err := sopService.AddAcknowledgment(vID, userIDs[d.key], sop.AckTypeRead); err != nil {
				return fmt.Errorf("failed to record reader acknowledgment: %w", err)
			}
		}
	}

	// 6. favorites
	for _, key := range manifest.FavoritedBy {
		if err := sopService.FavoriteSOP(sopID, userIDs[key]); err != nil {
			return fmt.Errorf("failed to favorite: %w", err)
		}
	}

	return nil
}

func extractTitleFromMarkdown(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(b), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# ")), nil
		}
		// If we hit text before an H1, the file is malformed for this seeder
		break
	}
	return "", fmt.Errorf("no H1 title (# Title) found at start of %s", path)
}
