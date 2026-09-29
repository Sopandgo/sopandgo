package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
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

func main() {
	if !seedDemoEnabled(os.Getenv("SEED_DEMO_DATA")) {
		log.Println("SEED_DEMO_DATA=false, skipping demo data")
		return
	}

	log.Println("seeding demo data")

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("cannot create data directory: %v", err)
	}

	// If the database file already exists, we skip seeding entirely.
	if _, err := os.Stat(filepath.Join(dataDir, "app.db")); err == nil {
		log.Println("database already exists, skipping seeding")
		return
	}

	store, err := storage.Open(dataDir)
	if err != nil {
		log.Fatalf("cannot open storage: %v", err)
	}
	defer store.DB.Close()

	auditLogger, err := audit.New(store.DB)
	if err != nil {
		log.Fatalf("cannot init audit logger: %v", err)
	}

	sopService := sop.NewService(store.DB, auditLogger, dataDir, "1", false, nil)

	sopsRoot := filepath.Join("demo", "sops")

	authService := auth.NewService(store.DB, auditLogger)

	userIDs, err := ensureDemoUsers(authService)
	if err != nil {
		log.Fatalf("failed to create demo users: %v", err)
	}

	sopFolders, err := os.ReadDir(sopsRoot)
	if err != nil {
		log.Fatalf("failed to read sops root directory %s: %v", sopsRoot, err)
	}

	for _, f := range sopFolders {
		if !f.IsDir() {
			continue
		}

		sopDir := filepath.Join(sopsRoot, f.Name())
		if err := seedOneSOP(sopService, sopDir, userIDs); err != nil {
			log.Printf("ERR: failed seeding %s: %v", f.Name(), err)
			continue
		}
	}

	log.Println("demo data seeding process finished")
}

func ensureDemoUsers(authService *auth.Service) ([]string, error) {
	demos := []struct {
		name  string
		email string
		role  string
	}{
		{"Demo Lab Manager", "manager@demo.local", auth.RoleEditor},
		{"Demo QA Officer", "qa@demo.local", auth.RoleApprover},
		{"Demo Research Assistant", "researcher@demo.local", auth.RoleViewer},
	}

	var ids []string

	for _, d := range demos {
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

		log.Printf("created demo user: %s (%s)", d.name, d.role)
		ids = append(ids, id)
	}

	return ids, nil
}

func seedOneSOP(
	sopService *sop.Service,
	sopDir string,
	userIDs []string,
) error {
	log.Printf("processing SOP from %s", sopDir)

	// Ensure we have our specific roles from the ensureDemoUsers array
	if len(userIDs) < 3 {
		return fmt.Errorf("not enough demo users provided to seed workflow")
	}
	editorID := userIDs[0]
	approverID := userIDs[1]
	viewerID := userIDs[2]

	// 1. find version-1.md to get the title ---
	version1Path := filepath.Join(sopDir, "version-1.md")
	title, err := extractTitleFromMarkdown(version1Path)
	if err != nil {
		return fmt.Errorf("failed to extract SOP title from %s: %w", version1Path, err)
	}

	// 2. create the SOP (We can attribute the container creation to the Editor)
	sopID, err := sopService.RegisterSOP(title, &editorID)
	if err != nil {
		return err
	}

	// 3. add assets (images/diagrams) ---
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

	// 4. add versions
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

	var lastVersionID string
	for _, f := range mdFiles {
		content, err := os.ReadFile(filepath.Join(sopDir, f))
		if err != nil {
			return err
		}

		// STEP A: Editor creates the Draft
		vID, _, err := sopService.RegisterSOPVersion(sopID, string(content), "Updated procedure", &editorID)
		if err != nil {
			return err
		}

		// STEP B: Editor promotes Draft -> RC
		if err := sopService.TransitionVersionState(vID, sop.StateRC, editorID); err != nil {
			return fmt.Errorf("failed to promote to RC: %w", err)
		}

		// STEP C: Approver publishes RC -> Published
		if _, err := sopService.ApproveSOPVersion(vID, approverID); err != nil {
			return fmt.Errorf("failed to publish: %w", err)
		}

		lastVersionID = vID
		log.Printf("  -> seeded and published version: %s", f)
	}

	// Viewer reads the latest published version. The approver acknowledgment
	// is already recorded by ApproveSOPVersion.
	if lastVersionID != "" {
		if _, err := sopService.AddAcknowledgment(lastVersionID, viewerID, sop.AckTypeRead); err != nil {
			return fmt.Errorf("failed to record reader acknowledgment: %w", err)
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
