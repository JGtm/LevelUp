package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// Un seul import OpenSpartan à la fois par compte : un second envoi pendant qu'un import
// est actif répond 409 import_already_active, sans enregistrer le fichier ni créer de job.
func TestStartImport_SecondImportWhileActiveReturns409(t *testing.T) {
	h := newHandlerForTest(t)
	active := h.jobStore.Create(domain.JobTypeOpenSpartanImport, testHandlerXUID)

	req := buildMultipartRequest(t, "db", []byte("peu importe"), sessionWithXUID(testHandlerXUID))
	rr := httptest.NewRecorder()
	h.StartImport(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status: want 409, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "import_already_active") {
		t.Errorf("body should carry import_already_active, got %s", rr.Body.String())
	}
	entries, err := os.ReadDir(h.tempDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("le fichier d'un import refusé a été enregistré: %d entrée(s) dans %s", len(entries), h.tempDir)
	}
	if got := h.jobStore.FindActiveJob(domain.JobTypeOpenSpartanImport, testHandlerXUID); got == nil || got.JobID != active.JobID {
		t.Errorf("le job actif d'origine doit rester le seul actif: %+v", got)
	}
}

// L'import d'un autre compte, ou après la fin du précédent, n'est pas bloqué.
func TestStartImport_OtherAccountOrFinishedJobIsNotBlocked(t *testing.T) {
	dir := t.TempDir()
	fixturePath := buildMinimalOpenSpartanFile(t, dir, testHandlerXUID)
	fileBytes, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}

	h := newHandlerForTest(t)
	other := h.jobStore.Create(domain.JobTypeOpenSpartanImport, "2533274800000001")
	finished := h.jobStore.Create(domain.JobTypeOpenSpartanImport, testHandlerXUID)
	h.jobStore.SetStatus(finished.JobID, domain.JobStatusSucceeded, nil)
	t.Cleanup(func() { _ = other })

	rr := httptest.NewRecorder()
	h.StartImport(rr, buildMultipartRequest(t, "db", fileBytes, sessionWithXUID(testHandlerXUID)))
	if rr.Code != http.StatusAccepted {
		t.Fatalf("status: want 202, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	// Attendre la fin de l'import lancé en tâche de fond avant de rendre la main.
	for _, j := range h.jobStore.List(10) {
		if j.JobID != other.JobID && j.JobID != finished.JobID {
			pollJobUntilDone(t, h.jobStore, j.JobID, 30*time.Second)
		}
	}
}
