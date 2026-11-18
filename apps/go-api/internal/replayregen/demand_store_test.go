package replayregen

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
)

func TestFileDemandStore_AllerRetour(t *testing.T) {
	paths := title.NewPathResolver(t.TempDir())
	st := NewFileDemandStore(paths.ReplayGenerationQueuePath())
	ctx := context.Background()
	if got, err := st.LoadDemands(ctx); err != nil || got != nil {
		t.Fatalf("fichier absent : %v %v, attendu une liste vide sans erreur", got, err)
	}
	want := []domain.ReplayGenerationDemand{demandFor(1), demandFor(2)}
	if err := st.SaveDemands(ctx, want); err != nil {
		t.Fatal(err)
	}
	got, err := st.LoadDemands(ctx)
	if err != nil || len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("aller-retour : %+v %v", got, err)
	}
	// le fichier est écrit sous AdminStateDir, sans temporaire résiduel
	entries, _ := os.ReadDir(paths.AdminStateDir())
	if len(entries) != 1 || entries[0].Name() != "replay_generation_queue.json" {
		t.Fatalf("contenu de %s : %v", paths.AdminStateDir(), entries)
	}
	raw, _ := os.ReadFile(paths.ReplayGenerationQueuePath())
	if strings.Contains(strings.ToLower(string(raw)), "token") {
		t.Fatalf("le fichier ne doit porter aucun jeton : %s", raw)
	}
}

func TestFileDemandStore_FichierIllisibleOuDUneAutreVersion(t *testing.T) {
	p := filepath.Join(t.TempDir(), "q.json")
	st := NewFileDemandStore(p)
	for _, contenu := range []string{"pas du json", `{"version":99,"demands":[]}`} {
		if err := os.WriteFile(p, []byte(contenu), 0o644); err != nil {
			t.Fatal(err)
		}
		if got, err := st.LoadDemands(context.Background()); err == nil {
			t.Fatalf("%q : attendu une erreur, obtenu %v", contenu, got)
		}
	}
}

// Un fichier illisible : la reprise journalise en Warn et laisse la file vide.
func TestResumeDemands_FichierIllisibleSurFichier(t *testing.T) {
	logs := captureLogs(t)
	paths := title.NewPathResolver(t.TempDir())
	if err := os.MkdirAll(paths.AdminStateDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.ReplayGenerationQueuePath(), []byte("{pas du json"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	svc := New()
	svc.Bind(ctx, &wg, Deps{Paths: paths, Demands: NewFileDemandStore(paths.ReplayGenerationQueuePath()),
		GateEvery: time.Millisecond})
	svc.ResumeDemands()
	cancel()
	wg.Wait()
	if !strings.Contains(logs.String(), "level=WARN") || !strings.Contains(logs.String(), "illisibles") {
		t.Fatalf("Warn attendu : %s", logs.String())
	}
	if _, ok := svc.DemandStatus(testSlug, demandFor(1).MatchID); ok {
		t.Fatal("la file doit être vide")
	}
}

// L'heure de fin du match voyage avec la demande persistée (elle sert encore après un redémarrage) ; absente, elle
// n'écrit rien et se relit zéro.
func TestFileDemandStore_HeureDeFinDuMatch(t *testing.T) {
	st := NewFileDemandStore(filepath.Join(t.TempDir(), "q.json"))
	ctx := context.Background()
	fin := time.Date(2026, 11, 18, 11, 58, 0, 0, time.UTC)
	avec, sans := demandFor(1), demandFor(2)
	avec.MatchEndedAt = fin
	if err := st.SaveDemands(ctx, []domain.ReplayGenerationDemand{avec, sans}); err != nil {
		t.Fatal(err)
	}
	got, err := st.LoadDemands(ctx)
	if err != nil || len(got) != 2 || !got[0].MatchEndedAt.Equal(fin) || !got[1].MatchEndedAt.IsZero() {
		t.Fatalf("aller-retour : %+v %v", got, err)
	}
}
