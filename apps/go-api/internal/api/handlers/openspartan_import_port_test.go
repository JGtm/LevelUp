package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/openspartan"
	"levelup/go-api/internal/port"
)

// fakeOpenSpartanImport : le service d'import, sans base ni fichier OpenSpartan.
type fakeOpenSpartanImport struct {
	result port.OpenSpartanImportResult
	err    error

	mu   sync.Mutex
	got  port.OpenSpartanImportOptions
	xuid string
}

func (f *fakeOpenSpartanImport) Import(_ context.Context, xuid, _ string, opts port.OpenSpartanImportOptions) (port.OpenSpartanImportResult, error) {
	f.mu.Lock()
	f.got, f.xuid = opts, xuid
	f.mu.Unlock()
	if opts.OnProgress != nil {
		opts.OnProgress(5, 10)
	}
	return f.result, f.err
}

// fakeOpenSpartanPostImport : le recalcul après import, qui garde ce qu'on lui a demandé.
type fakeOpenSpartanPostImport struct {
	result port.OpenSpartanPostImportResult

	mu       sync.Mutex
	calls    int
	gamertag string
	matchIDs []string
}

func (f *fakeOpenSpartanPostImport) Run(_ context.Context, _, gamertag string, matchIDs []string, _ port.OpenSpartanPostImportOptions) (port.OpenSpartanPostImportResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.gamertag, f.matchIDs = gamertag, matchIDs
	return f.result, nil
}

// runFakeImport envoie un téléversement et rend le job à son état terminal.
func runFakeImport(t *testing.T, imp port.OpenSpartanImportService, post port.OpenSpartanPostImportService) *domain.AsyncJobStatus {
	t.Helper()
	store := newJobStoreForTest(t)
	h := NewOpenSpartanImportHandler(OpenSpartanImportConfig{
		ImportService: imp, PostImportService: post, JobStore: store,
		TempDir: t.TempDir(), StashDir: filepath.Join(t.TempDir(), "players"),
	})
	sess := &domain.SessionData{LinkedHaloIdentity: &domain.HaloIdentity{XUID: testHandlerXUID, Gamertag: "Tester"}}
	rr := httptest.NewRecorder()
	h.StartImport(rr, buildMultipartRequest(t, "db", []byte("peu importe : le faux service ne lit pas le fichier"), sess))
	if rr.Code != http.StatusAccepted {
		t.Fatalf("statut %d, attendu 202 : %s", rr.Code, rr.Body.String())
	}
	deadline := time.After(10 * time.Second)
	for {
		for _, j := range store.List(10) {
			if j.Status == domain.JobStatusSucceeded || j.Status == domain.JobStatusFailed {
				return j
			}
		}
		select {
		case <-deadline:
			t.Fatal("le job n'a pas atteint un état terminal en 10 s")
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// TestOpenSpartanImport_FauxPort_Succes : le handler passe au port le xuid de la session, le dossier de
// remise des amis et un rappel de progression, lance le recalcul sur les matchs insérés, et range les
// comptes des deux services dans le résultat du job.
func TestOpenSpartanImport_FauxPort_Succes(t *testing.T) {
	imp := &fakeOpenSpartanImport{result: port.OpenSpartanImportResult{
		DetectedOwnerXUID: testHandlerXUID, Confidence: openspartan.Confidence(2), TotalMatches: 3, InsertedMatches: 2,
		InsertedMatchIDs: []string{"m1", "m2"},
	}}
	post := &fakeOpenSpartanPostImport{result: port.OpenSpartanPostImportResult{SessionsTouched: 4, PerfScoresTouched: 2}}

	job := runFakeImport(t, imp, post)

	if job.Status != domain.JobStatusSucceeded {
		t.Fatalf("job %s, attendu succeeded : %+v", job.Status, job.Error)
	}
	if imp.xuid != testHandlerXUID || imp.got.Source != "openspartan_import" || imp.got.StashDir == "" {
		t.Errorf("appel du port : xuid=%q options=%+v", imp.xuid, imp.got)
	}
	if post.calls != 1 || post.gamertag != "Tester" || fmt.Sprint(post.matchIDs) != "[m1 m2]" {
		t.Errorf("recalcul : %d appel(s), gamertag=%q, matchs=%v", post.calls, post.gamertag, post.matchIDs)
	}
	if job.Result["inserted_matches"] != 2 {
		t.Errorf("inserted_matches = %v, attendu 2", job.Result["inserted_matches"])
	}
	pi, _ := job.Result["post_import"].(map[string]any)
	if pi["sessions_touched"] != 4 || pi["perf_scores_touched"] != 2 {
		t.Errorf("post_import = %v", pi)
	}
}

// TestOpenSpartanImport_FauxPort_CodesDEchec : l'identité des erreurs du port fixe le code d'échec du
// job, sans recalcul.
func TestOpenSpartanImport_FauxPort_CodesDEchec(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"propriétaire différent", fmt.Errorf("détecté: %w", port.ErrOpenSpartanXUIDMismatch), "xuid_mismatch"},
		{"confiance faible", port.ErrOpenSpartanLowConfidence, "owner_low_confidence"},
		{"pas une base OpenSpartan", fmt.Errorf("ouverture: %w", openspartan.ErrNotOpenSpartanDB), "not_openspartan_db"},
		{"autre", errors.New("disque plein"), "import_error"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			post := &fakeOpenSpartanPostImport{}
			job := runFakeImport(t, &fakeOpenSpartanImport{err: c.err}, post)
			if job.Status != domain.JobStatusFailed || job.Error == nil || job.Error.Code != c.want {
				t.Fatalf("job %s erreur %+v, attendu failed/%s", job.Status, job.Error, c.want)
			}
			if post.calls != 0 {
				t.Errorf("recalcul lancé après un échec : %d appel(s)", post.calls)
			}
		})
	}
}
