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
	"levelup/go-api/internal/port"
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

// TestEnqueueAll_ReEnfileUnArtefactAppauvri — LE PIÈGE DE FRAÎCHEUR, FERMÉ.
//
// Un artefact construit sans les faits du match porte la BONNE version de schéma : sur le seul
// critère de version il est « à jour », donc plus rien ne le re-cuit jamais. C'est ainsi qu'un
// ouvrier sans faits empoisonnerait le cache de rejeu à demeure. Ici la base connaît des
// participants : l'artefact appauvri doit repartir en file.
func TestEnqueueAll_ReEnfileUnArtefactAppauvri(t *testing.T) {
	repoRoot := t.TempDir()
	appauvri := "cccccccc-3333-4000-8000-000000000000"
	ecrireArtefact(t, repoRoot, appauvri) // schéma courant, mais sans joueurs de score

	var enfiles []string
	d := Deps{
		RepoRoot:  repoRoot,
		TitleSlug: titlePkg.DefaultSlug,
		Enqueue: func(_ context.Context, _, matchID string) error {
			enfiles = append(enfiles, matchID)
			return nil
		},
	}
	enqueueAll(context.Background(), d, []buildWork{{
		matchID: appauvri,
		facts: port.MatchFacts{
			GameVariantName: "CTF:Arena",
			Players:         []port.MatchPlayerFact{{XUID: "2533274819954312", Kills: 12}},
		},
	}})
	if len(enfiles) != 1 || enfiles[0] != appauvri {
		t.Fatalf("mis en file = %v, attendu le match dont l'artefact est appauvri", enfiles)
	}
}

// TestEnqueueAll_ArtefactCompletResteSaute : l'artefact qui PORTE déjà les faits ne se re-cuit
// pas. Sans ce cas, le prédicat de fraîcheur enfilerait tout le cache à chaque cycle.
func TestEnqueueAll_ArtefactCompletResteSaute(t *testing.T) {
	repoRoot := t.TempDir()
	complet := "dddddddd-4444-4000-8000-000000000000"
	ecrireArtefactAvecFaits(t, repoRoot, complet)

	var enfiles []string
	d := Deps{
		RepoRoot: repoRoot, TitleSlug: titlePkg.DefaultSlug,
		Enqueue: func(_ context.Context, _, matchID string) error {
			enfiles = append(enfiles, matchID)
			return nil
		},
	}
	enqueueAll(context.Background(), d, []buildWork{{
		matchID: complet,
		facts:   port.MatchFacts{Players: []port.MatchPlayerFact{{XUID: "2533274819954312"}}},
	}})
	if len(enfiles) != 0 {
		t.Fatalf("mis en file = %v, attendu aucun (l'artefact porte déjà les faits)", enfiles)
	}
}

// TestEnqueueAll_SansFaitsConnus_NeReEnfilePas : un match dont la base ne sait RIEN a
// légitimement un artefact sans joueurs. Le re-cuire serait une boucle perpétuelle — le
// prédicat ne doit pas transformer une absence de données en travail sans fin.
func TestEnqueueAll_SansFaitsConnus_NeReEnfilePas(t *testing.T) {
	repoRoot := t.TempDir()
	horsRegistre := "eeeeeeee-5555-4000-8000-000000000000"
	ecrireArtefact(t, repoRoot, horsRegistre)

	var enfiles []string
	d := Deps{
		RepoRoot: repoRoot, TitleSlug: titlePkg.DefaultSlug,
		Enqueue: func(_ context.Context, _, matchID string) error {
			enfiles = append(enfiles, matchID)
			return nil
		},
	}
	enqueueAll(context.Background(), d, []buildWork{{matchID: horsRegistre}}) // faits vides
	if len(enfiles) != 0 {
		t.Fatalf("mis en file = %v, attendu aucun (aucun fait n'existe pour ce match)", enfiles)
	}
}

// ecrireArtefactAvecFaits pose un artefact COMPLET : schéma courant ET joueurs de courbe de
// score, la marque que les lignes de match ont été fournies au constructeur.
func ecrireArtefactAvecFaits(t *testing.T, repoRoot, matchID string) {
	t.Helper()
	path := titlePkg.NewPathResolver(repoRoot).ReplayArtifactPath(titlePkg.DefaultSlug, matchID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	blob, err := json.Marshal(replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion,
		MatchID:       matchID,
		ScoreTimeline: &replay.ScoreTimeline{Players: []replay.PlayerScore{{XUID: "2533274819954312"}}},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, blob, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
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
