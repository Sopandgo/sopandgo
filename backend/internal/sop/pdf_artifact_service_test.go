package sop_test

import (
	"errors"
	"testing"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/pdfgen"
	"github.com/sopandgo/sopandgo/backend/internal/sop"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

type failingRenderer struct{}

func (failingRenderer) Render(pdfgen.Input) ([]byte, error) {
	return nil, errors.New("renderer unreachable")
}

// A committed lifecycle change must be reported as a success even when the PDF renderer is down.
func TestService_VersionLifecycle_SurvivesPDFRendererFailure(t *testing.T) {
	env := testenv.New(t)
	actorID := "pdf-failure-user"
	env.SeedTestUser(t, actorID, "pdf-failure@demo.local", "admin")

	svc := sop.NewService(env.Store.DB, env.AuditLogger, env.DataDir, "test-generator", true, failingRenderer{})
	if !svc.IsPDFExportEnabled() {
		t.Fatal("expected PDF export to be enabled")
	}

	sopID, err := svc.RegisterSOP("PDF Failure SOP", &actorID)
	if err != nil {
		t.Fatalf("RegisterSOP failed: %v", err)
	}

	vID, vNum, err := svc.RegisterSOPVersion(sopID, "# Body", "Initial procedure", &actorID)
	if err != nil {
		t.Fatalf("RegisterSOPVersion should succeed when PDF generation fails: %v", err)
	}
	if vNum != 1 {
		t.Fatalf("expected version 1, got %d", vNum)
	}

	time.Sleep(15 * time.Millisecond)
	if err := svc.TransitionVersionState(vID, sop.StateRC, actorID); err != nil {
		t.Fatalf("TransitionVersionState should succeed when PDF generation fails: %v", err)
	}

	time.Sleep(15 * time.Millisecond)
	if _, err := svc.ApproveSOPVersion(vID, actorID); err != nil {
		t.Fatalf("ApproveSOPVersion should succeed when PDF generation fails: %v", err)
	}

	v, err := svc.GetSOPVersionByID(vID)
	if err != nil {
		t.Fatalf("GetSOPVersionByID failed: %v", err)
	}
	if v.Status != sop.StatePublished {
		t.Fatalf("expected status %q, got %q", sop.StatePublished, v.Status)
	}

	if _, _, err := svc.GetVersionPDFArtifactPath(vID, ""); err == nil {
		t.Fatal("expected no PDF artifact after renderer failure")
	}

	processed, generated, err := svc.BackfillPDFArtifactsForGenerator(actorID)
	if err != nil {
		t.Fatalf("backfill should skip failing versions, got: %v", err)
	}
	if processed == 0 || generated != 0 {
		t.Fatalf("expected processed > 0 and generated == 0, got processed=%d generated=%d", processed, generated)
	}
}
