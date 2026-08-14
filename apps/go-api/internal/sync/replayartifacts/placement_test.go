// Package replayartifacts — placement_test.go : le fil de l'eau prend UN chemin,
// et un seul.
//
// Ce qui est vérifié : le hook décide d'après le réglage vivant (relu à chaque
// cycle), « off » ne touche même pas la base, et le chemin « ouvrier » n'enfile
// que ce qui a besoin de l'être — la fenêtre de rétention et l'idempotence
// s'appliquent AVANT la file, pas après.
package replayartifacts

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/replaybuild"
)

// hookAvec construit un hook au réglage donné, sans store de configuration.
func hookAvec(location string, env replaybuild.PlacementEnv, enqueue EnqueueFunc) *Hook {
	return &Hook{
		Location: func() string { return location },
		Env:      env,
		Enqueue:  enqueue,
	}
}

func noopEnqueue(context.Context, string, string) error { return nil }

// TestHookPlacement : le réglage commande, et l'absence de file ne fait JAMAIS
// retomber sur une construction locale (le VPS web ne décode jamais).
func TestHookPlacement(t *testing.T) {
	dev := replaybuild.PlacementEnv{WorkerConfigured: true}
	prod := replaybuild.PlacementEnv{Production: true, WorkerConfigured: true}
	cases := map[string]struct {
		hook   *Hook
		attend replaybuild.Placement
	}{
		"local en dev":            {hookAvec("local", dev, noopEnqueue), replaybuild.PlacementLocal},
		"worker en dev":           {hookAvec("worker", dev, noopEnqueue), replaybuild.PlacementWorker},
		"off":                     {hookAvec("off", dev, noopEnqueue), replaybuild.PlacementOff},
		"local en prod refusé":    {hookAvec("local", prod, noopEnqueue), replaybuild.PlacementOff},
		"worker sans file câblée": {hookAvec("worker", dev, nil), replaybuild.PlacementOff},
		"hook nil":                {nil, replaybuild.PlacementOff},
	}
	for nom, c := range cases {
		if got := c.hook.Placement(); got != c.attend {
			t.Errorf("%s : placement = %q, attendu %q", nom, got, c.attend)
		}
	}
}

// TestRun_Off_NeTouchePasLaBase : « ne rien faire » veut dire ne rien faire —
// pas même une lecture du registre pour sélectionner du travail à jeter.
func TestRun_Off_NeTouchePasLaBase(t *testing.T) {
	lu := false
	Run(context.Background(), Deps{
		Placement: replaybuild.PlacementOff,
		WithRead:  func(context.Context, string, func(*sql.DB)) { lu = true },
	}, []string{"m1"})
	if lu {
		t.Fatal("placement off : la base a été lue alors qu'aucune construction n'est demandée")
	}
}

// TestRun_Worker_SelectionnePuisEnfile : le chemin ouvrier sélectionne (donc
// applique la fenêtre de rétention) avant de mettre en file.
func TestRun_Worker_SelectionnePuisEnfile(t *testing.T) {
	lu := false
	Run(context.Background(), Deps{
		Placement: replaybuild.PlacementWorker,
		Enqueue:   noopEnqueue,
		WithRead:  func(context.Context, string, func(*sql.DB)) { lu = true },
	}, []string{"m1"})
	if !lu {
		t.Fatal("placement worker : la sélection n'a pas eu lieu — la fenêtre de rétention ne s'appliquerait pas")
	}
}

// TestEnqueueAll_SauteLesArtefactsAJour : on n'enfile pas un match dont
// l'artefact est déjà là et à jour — la file et la purge ne se courent pas après.
func TestEnqueueAll_SauteLesArtefactsAJour(t *testing.T) {
	repoRoot := t.TempDir()
	dejaLa := "aaaaaaaa-1111-4000-8000-000000000000"
	ecrireArtefact(t, repoRoot, dejaLa)

	var enfiles []string
	d := Deps{
		RepoRoot:  repoRoot,
		TitleSlug: titlePkg.DefaultSlug,
		Enqueue: func(_ context.Context, _, matchID string) error {
			enfiles = append(enfiles, matchID)
			return nil
		},
	}
	enqueueAll(context.Background(), d, []buildWork{
		{matchID: dejaLa},
		{matchID: "bbbbbbbb-2222-4000-8000-000000000000"},
	})
	if len(enfiles) != 1 || enfiles[0] != "bbbbbbbb-2222-4000-8000-000000000000" {
		t.Fatalf("mis en file = %v, attendu le seul match sans artefact à jour", enfiles)
	}
}

// TestEnqueueAll_SansFile_NeFaitRien : un chemin de sync sans file câblée ne
// construit pas non plus — il s'abstient.
func TestEnqueueAll_SansFile_NeFaitRien(t *testing.T) {
	enqueueAll(context.Background(), Deps{RepoRoot: t.TempDir()}, []buildWork{{matchID: "m1"}})
}

// ecrireArtefact pose un artefact à la version de schéma courante.
func ecrireArtefact(t *testing.T, repoRoot, matchID string) {
	t.Helper()
	path := titlePkg.NewPathResolver(repoRoot).ReplayArtifactPath(titlePkg.DefaultSlug, matchID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	blob, err := json.Marshal(replay.ReplayDocument{SchemaVersion: replay.SchemaVersion, MatchID: matchID})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, blob, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
