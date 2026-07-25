//go:build cgo

// Package prestige — prestige_metadata_reopen_test.go : régression « 500 permanent
// sur le catalogue Prestige » (metadata.duckdb).
//
// Contexte (V721-08.1, Découverte D1 du PLAN_SQUAD_CHALLENGES). Les repos
// référentiels (PrestigeTemplateRepo / PrestigePresetArcRepo) partagent l'unique
// handle RW metadata tenue par le serveur pour toute sa durée de vie. Tant qu'ils
// lisaient via `db.Query` / `db.QueryRow` PLATS, une invalidation FATAL DuckDB
// (bug ART) ou une handle périmée (`sql: database is closed` après un Close /
// Reopen concurrent) rendait TOUTES les lectures du catalogue définitivement en
// erreur jusqu'au prochain restart. Côté HTTP, l'erreur n'est aucune des
// sentinelles de PrestigeHandler.serviceError → branche `default` = 500 masqué :
// c'est le 500 constaté en prod sur « Proposer des défis »
// (POST /squads/{id}/challenges/pool/refresh → RefreshSquadPool → ListByTitle).
//
// Simulacre : db.Close() AVANT chaque appel (même procédé que
// prestige_player_reopen_test.go, dont le fix *Recovered est déjà en place côté
// player DB). Fichier réel (t.TempDir) car Reopen doit pouvoir rouvrir le DSN —
// :memory: ne survit pas à un close.
//
// Oracle : sur la handle fermée, chaque méthode doit réussir et rendre la ligne
// seedée. Le code pré-fix (Query/QueryRow plats) remonte « sql: database is
// closed » → chaque appel FAIL. Le code post-fix (*Recovered) PASS.
package prestige

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/prestige"
)

// newPrestigeMetadataReopenDB crée un metadata.duckdb temporaire (fichier réel)
// avec le schéma metadata complet (migrations réelles, title steps câblés par le
// TestMain du sous-package) et retourne la handle RW ouverte via le wrapper
// duckdb (cache + pool), seule apte au Reopen auto-réparant.
func newPrestigeMetadataReopenDB(t *testing.T) *duckdb.DB {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "metadata.duckdb")

	raw, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if err := migration.RunForDB(raw, migration.TargetMetadata); err != nil {
		raw.Close()
		t.Fatalf("RunForDB(metadata): %v", err)
	}
	raw.Close()

	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// seedReopenCatalog insère 2 templates + 1 arc preset (1 étape) sur la handle ouverte.
func seedReopenCatalog(t *testing.T, db *duckdb.DB) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	templates := []prestige.Template{
		{
			ID: "tpl_reopen_a", TitleSlug: "halo_infinite", Metric: "kills_total",
			WindowType: prestige.WindowSession, Cadence: prestige.CadenceWeekly,
			EvalType: prestige.EvalCumulative, ModeFilter: "universal",
			LabelEN: "A", LabelFR: "A",
			NormalTarget: 1, HeroicTarget: 2, LegendaryTarget: 3, MythicTarget: 4,
			SchemaVersion: 1, UpdatedAt: now,
		},
		{
			ID: "tpl_reopen_b", TitleSlug: "halo_infinite", Metric: "assists",
			WindowType: prestige.WindowSession, Cadence: prestige.CadenceWeekly,
			EvalType: prestige.EvalCumulative, ModeFilter: "universal",
			LabelEN: "B", LabelFR: "B",
			NormalTarget: 1, HeroicTarget: 2, LegendaryTarget: 3, MythicTarget: 4,
			SchemaVersion: 1, UpdatedAt: now,
		},
	}
	if err := NewPrestigeTemplateRepo(db).Replace(ctx, "halo_infinite", templates); err != nil {
		t.Fatalf("seed templates Replace: %v", err)
	}

	arcs := []prestige.PresetArc{{
		ID: "arc_reopen", TitleSlug: "halo_infinite",
		TitleEN: "Arc", TitleFR: "Arc", SchemaVersion: 1, UpdatedAt: now,
	}}
	steps := []prestige.PresetArcStep{{
		PresetArcID: "arc_reopen", Position: 1,
		TemplateID: "tpl_reopen_a", TargetTier: prestige.TierHeroic,
	}}
	if err := NewPrestigePresetArcRepo(db).Replace(ctx, "halo_infinite", arcs, steps); err != nil {
		t.Fatalf("seed preset arcs Replace: %v", err)
	}
}

// TestPrestigeMetadataRepos_RecoverAfterHandleClose ferme la handle metadata AVANT
// chaque appel puis vérifie que les 6 lectures du fichier récupèrent (Reopen +
// retry) au lieu d'échouer : ListByTitle / GetByID / Suggest (templates) et
// ListByTitle / GetByID / GetSteps (preset arcs). Chaque close est ré-appliqué
// pour prouver la recovery de CHAQUE call site indépendamment.
func TestPrestigeMetadataRepos_RecoverAfterHandleClose(t *testing.T) {
	db := newPrestigeMetadataReopenDB(t)
	seedReopenCatalog(t, db)

	tplRepo := NewPrestigeTemplateRepo(db)
	arcRepo := NewPrestigePresetArcRepo(db)
	ctx := context.Background()

	// 1. Templates.ListByTitle — la lecture du 500 prod (RefreshSquadPool).
	closeHandle(t, db, "Templates.ListByTitle")
	list, err := tplRepo.ListByTitle(ctx, "halo_infinite")
	if err != nil {
		t.Fatalf("Templates.ListByTitle après close: attendu recovery, got %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("Templates.ListByTitle: got %d templates, want 2", len(list))
	}

	// 2. Templates.GetByID (QueryRowRecovered).
	closeHandle(t, db, "Templates.GetByID")
	tpl, err := tplRepo.GetByID(ctx, "tpl_reopen_a")
	if err != nil {
		t.Fatalf("Templates.GetByID après close: attendu recovery, got %v", err)
	}
	if tpl.ID != "tpl_reopen_a" || tpl.Metric != "kills_total" {
		t.Fatalf("Templates.GetByID: got %q/%q want tpl_reopen_a/kills_total", tpl.ID, tpl.Metric)
	}

	// 3. Templates.Suggest (Query avec exclusions).
	closeHandle(t, db, "Templates.Suggest")
	suggested, err := tplRepo.Suggest(ctx, "halo_infinite", []string{"tpl_reopen_a"}, 5)
	if err != nil {
		t.Fatalf("Templates.Suggest après close: attendu recovery, got %v", err)
	}
	if len(suggested) != 1 || suggested[0].ID != "tpl_reopen_b" {
		t.Fatalf("Templates.Suggest: got %d templates, want 1 (tpl_reopen_b)", len(suggested))
	}

	// 4. PresetArcs.ListByTitle.
	closeHandle(t, db, "PresetArcs.ListByTitle")
	arcs, err := arcRepo.ListByTitle(ctx, "halo_infinite")
	if err != nil {
		t.Fatalf("PresetArcs.ListByTitle après close: attendu recovery, got %v", err)
	}
	if len(arcs) != 1 || arcs[0].ID != "arc_reopen" {
		t.Fatalf("PresetArcs.ListByTitle: got %d arcs, want 1 (arc_reopen)", len(arcs))
	}

	// 5. PresetArcs.GetByID (QueryRowRecovered).
	closeHandle(t, db, "PresetArcs.GetByID")
	arc, err := arcRepo.GetByID(ctx, "arc_reopen")
	if err != nil {
		t.Fatalf("PresetArcs.GetByID après close: attendu recovery, got %v", err)
	}
	if arc.ID != "arc_reopen" {
		t.Fatalf("PresetArcs.GetByID: got %q want arc_reopen", arc.ID)
	}

	// 6. PresetArcs.GetSteps.
	closeHandle(t, db, "PresetArcs.GetSteps")
	steps, err := arcRepo.GetSteps(ctx, "arc_reopen")
	if err != nil {
		t.Fatalf("PresetArcs.GetSteps après close: attendu recovery, got %v", err)
	}
	if len(steps) != 1 || steps[0].TemplateID != "tpl_reopen_a" {
		t.Fatalf("PresetArcs.GetSteps: got %d étapes, want 1 (tpl_reopen_a)", len(steps))
	}

	// 7. Not-found : la recovery ne doit pas masquer sql.ErrNoRows en succès — le
	//    mapping domaine ErrChallengeNotFound doit survivre au Reopen.
	closeHandle(t, db, "Templates.GetByID(absent)")
	if _, err := tplRepo.GetByID(ctx, "tpl_absent"); !errors.Is(err, prestige.ErrChallengeNotFound) {
		t.Fatalf("Templates.GetByID(absent) après close: want ErrChallengeNotFound, got %v", err)
	}
}

// TestTranslateLockErr_MapsFileLockToErrDBLocked verrouille la 2e moitié de la
// chaîne d'erreur : une contention fichier DuckDB (un AUTRE process tient
// metadata.duckdb, ou Reopen ne peut pas rouvrir le DSN) doit ressortir en
// dblease.ErrDBLocked, seule sentinelle que PrestigeHandler.serviceError mappe en
// 503 + Retry-After (au lieu du 500 masqué de la branche `default`).
//
// La contention réelle exige un 2e process détenteur du fichier : non simulable
// in-process. On teste donc la traduction sur les signatures d'erreur DuckDB
// exactes reconnues par duckdb.IsFileLockError, et on vérifie que le reste passe
// INCHANGÉ (sql.ErrNoRows notamment, dont dépend le mapping ErrChallengeNotFound
// / ErrArcNotFound). Le maillon final (ErrDBLocked → 503) est couvert par
// api/handlers/prestige_test.go (TestPrestigeHandler_*_DBLocked_Returns503).
func TestTranslateLockErr_MapsFileLockToErrDBLocked(t *testing.T) {
	ctx := context.Background()

	locked := []string{
		"Could not set lock on file \"metadata.duckdb\"",
		"Conflicting lock is held in levelup-server.exe (PID 1234)",
		"IO Error: File is already open in server.exe (PID 4242)",
	}
	for _, msg := range locked {
		got := translateLockErr(ctx, errors.New(msg))
		if !errors.Is(got, dblease.ErrDBLocked) {
			t.Errorf("translateLockErr(%q) ne wrappe pas ErrDBLocked (→ 500 au lieu de 503), got %v", msg, got)
		}
	}

	// Pass-through : identité des erreurs non-lock préservée.
	if got := translateLockErr(ctx, nil); got != nil {
		t.Errorf("translateLockErr(nil) = %v, want nil", got)
	}
	if got := translateLockErr(ctx, sql.ErrNoRows); !errors.Is(got, sql.ErrNoRows) {
		t.Errorf("translateLockErr(sql.ErrNoRows) doit préserver ErrNoRows, got %v", got)
	}
	if got := translateLockErr(ctx, sql.ErrNoRows); errors.Is(got, dblease.ErrDBLocked) {
		t.Error("translateLockErr(sql.ErrNoRows) ne doit pas devenir ErrDBLocked")
	}
	boom := errors.New("syntax error near SELECT")
	if got := translateLockErr(ctx, boom); got != boom {
		t.Errorf("translateLockErr(err non-lock) = %v, want l'erreur d'origine", got)
	}
}
