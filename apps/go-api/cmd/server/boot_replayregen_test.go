package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/replayartifact"
	"levelup/go-api/internal/replayregen"
)

type pendingEnMemoire struct {
	mu sync.Mutex
	on bool
}

func (p *pendingEnMemoire) ReplayRegenPending() (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.on, nil
}

func (p *pendingEnMemoire) SetReplayRegenPending(on bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.on = on
	return nil
}

// regenFixture : deux rejeux en retard (schéma 1) dont le film est présent, une reconstruction qui compte ses appels.
func regenFixture(t *testing.T, pending bool) (replayregen.Deps, *atomic.Int32, *atomic.Bool) {
	t.Helper()
	paths := title.NewPathResolver(t.TempDir())
	films := map[string]bool{}
	for _, id := range []string{"00000100", "00000101"} {
		p := paths.ReplayArtifactPath("halo_infinite", id)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(map[string]any{"matchId": id, "schemaVersion": 1})
		if _, _, err := replayartifact.Write(context.Background(), p, raw); err != nil {
			t.Fatal(err)
		}
		films[id] = true
	}
	var calls atomic.Int32
	var heavy atomic.Bool
	heavy.Store(true)
	return replayregen.Deps{
		Paths:        paths,
		Titles:       func() []string { return []string{"halo_infinite"} },
		FilmPresent:  func(_, id string) bool { return films[id] },
		Rebuild:      func(context.Context, string, string) error { calls.Add(1); return nil },
		Pending:      &pendingEnMemoire{on: pending},
		HeavyAllowed: heavy.Load,
		GateEvery:    10 * time.Millisecond,
	}, &calls, &heavy
}

func startedRegen(t *testing.T, deps replayregen.Deps) *replayregen.Service {
	return startedRegenModes(t, deps, replayRegenModes{Batch: true})
}

func startedRegenModes(t *testing.T, deps replayregen.Deps, modes replayRegenModes) *replayregen.Service {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	svc := replayregen.New()
	startReplayRegen(ctx, &wg, svc, deps, modes)
	t.Cleanup(func() { cancel(); wg.Wait() })
	return svc
}

// TestBootReplayRegen_SansDrapeauAucuneFile : sans drapeau de reprise, la file ne part jamais seule au démarrage ; seul
// le décompte se fait.
func TestBootReplayRegen_SansDrapeauAucuneFile(t *testing.T) {
	deps, calls, _ := regenFixture(t, false)
	svc := startedRegen(t, deps)
	deadline := time.Now().Add(5 * time.Second)
	for !svc.Status().Checked {
		if time.Now().After(deadline) {
			t.Fatal("le décompte de démarrage ne s'est pas fait")
		}
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(80 * time.Millisecond)
	if st := svc.Status(); st.Running || st.Stale != 2 || calls.Load() != 0 {
		t.Fatalf("état = %+v, reconstructions = %d : file arrêtée, 2 en retard comptés", st, calls.Load())
	}
}

// TestBootReplayRegen_AvecDrapeauReprendSousLaPorteDuJeu : avec le drapeau, la file reprend ; une partie en cours
// (porte fermée, comme avant le premier relevé de `armGameWatch`) la retient jusqu'à son ouverture.
func TestBootReplayRegen_AvecDrapeauReprendSousLaPorteDuJeu(t *testing.T) {
	deps, calls, heavy := regenFixture(t, true)
	heavy.Store(false)
	svc := startedRegen(t, deps)
	time.Sleep(100 * time.Millisecond)
	if !svc.Status().Running || calls.Load() != 0 {
		t.Fatalf("état = %+v, reconstructions = %d : reprise en attente de la porte", svc.Status(), calls.Load())
	}
	heavy.Store(true)
	deadline := time.Now().Add(5 * time.Second)
	for calls.Load() != 2 {
		if time.Now().After(deadline) {
			t.Fatalf("reconstructions = %d, attendu 2 après l'ouverture de la porte", calls.Load())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// TestBootReplayRegen_ApresArmGameWatch : l'ordre de main() est une propriété du démarrage : la porte du mode jeu est
// armée AVANT que la file reprenne.
func TestBootReplayRegen_ApresArmGameWatch(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	arm := strings.Index(string(src), "\tarmGameWatch(st)")
	bind := strings.Index(string(src), "\tbindReplayRegen(st)")
	if arm < 0 || bind < 0 || arm > bind {
		t.Fatalf("armGameWatch (%d) doit précéder bindReplayRegen (%d) dans main()", arm, bind)
	}
}

// demandesEnMemoire : le magasin des demandes de génération en vol, en mémoire.
type demandesEnMemoire struct {
	demands []domain.ReplayGenerationDemand
}

func (d *demandesEnMemoire) LoadDemands(context.Context) ([]domain.ReplayGenerationDemand, error) {
	return d.demands, nil
}

func (d *demandesEnMemoire) SaveDemands(_ context.Context, demands []domain.ReplayGenerationDemand) error {
	d.demands = demands
	return nil
}

// TestBootReplayRegen_PosteDeDeveloppementSeulementLesDemandes : sans la capacité du lot, le démarrage ne compte rien
// et ne reprend aucune file de régénération ; avec celle des demandes, une demande en vol reprend.
func TestBootReplayRegen_PosteDeDeveloppementSeulementLesDemandes(t *testing.T) {
	deps, calls, _ := regenFixture(t, true) // un drapeau de lot posé : il ne doit pas reprendre
	deps.Demands = &demandesEnMemoire{demands: []domain.ReplayGenerationDemand{{
		TitleSlug: "halo_infinite", MatchID: "0000ab00-0000-0000-0000-000000000001", PlayerSlug: "P", XUID: "1",
	}}}
	var fetched atomic.Int32
	deps.FetchFilm = func(context.Context, domain.ReplayGenerationDemand) error { fetched.Add(1); return nil }
	svc := startedRegenModes(t, deps, replayRegenModes{Demands: true})
	deadline := time.Now().Add(5 * time.Second)
	for calls.Load() != 1 {
		if time.Now().After(deadline) {
			t.Fatalf("reconstructions = %d, attendu 1 (la demande reprise)", calls.Load())
		}
		time.Sleep(5 * time.Millisecond)
	}
	if st := svc.Status(); st.Running || st.Checked {
		t.Fatalf("état du lot = %+v : ni file reprise ni décompte hors de la capacité du lot", st)
	}
	if fetched.Load() != 1 {
		t.Fatalf("téléchargements = %d, attendu 1", fetched.Load())
	}
}
