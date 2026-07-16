package migrations

// steps_player_test.go — prouve que le target PLAYER (stats.duckdb) est provisionné
// pour Halo 5 avec le schéma Progression V2 (Ascension), vues _latest incluses.
//
// Contexte backlog « Sections Ascension ne s'affiche pas sur H5 » : le diagnostic
// initial (vue streak_latest absente → GET /players/{slug}/streaks renvoie 500) est
// PÉRIMÉ. Vérifié sur pièces (2026-07-16) : les 4 player DB h5 réelles portent bien
// streak/streak_history/streak_latest + record_history + milestone_earned (schéma
// tracé title_slug='halo_5'). Mécanisme : le set de migrations h5 ne possède QUE le
// target metadata (OwnsTarget) ; pour player, RunForTitleDB(halo_5, player) retombe
// sur le fallback HINF COMPLET (registre global + titleStepsProvider = StepsFor),
// qui inclut create_progression_player_schema (streak/record_history/milestone_earned)
// + create_streak_history_append_only (table append-only + vue streak_latest). ZÉRO
// duplication de SQL : les mêmes step-functions HINF servent les deux titres.
//
// Ce test VERROUILLE l'invariant : si un jour h5 possédait le target player (OwnsTarget
// player=true) sans réémettre ces steps, la couche Ascension h5 recasserait
// silencieusement (GET /streaks → 500, widget home auto-masqué). Pendant de
// TestHalo5Shared_InheritsInfiniteSchema (target shared) pour le target player.

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	halo5 "levelup/go-api/internal/games/halo_5"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

// viewExists indique si `name` existe en tant que VUE (table_type='VIEW'). Distinct
// de tableExists (metadata_test.go) qui compte aussi les tables de base : ici on
// exige explicitement une vue (les vues _latest append-only, ADR 0026).
func viewExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var n int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ? AND table_type = 'VIEW'",
		name,
	).Scan(&n); err != nil {
		t.Fatalf("viewExists(%s): %v", name, err)
	}
	return n == 1
}

// TestHalo5Player_InheritsProgressionV2Schema : le target player (NON possédé par le
// set h5) hérite du schéma player HINF complet, dont la couche Progression V2 Ascension
// (streaks/records/milestones) et sa vue append-only streak_latest.
func TestHalo5Player_InheritsProgressionV2Schema(t *testing.T) {
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	Register()

	// Le set h5 ne DOIT PAS posséder le target player : c'est la condition du fallback
	// HINF complet (registre global + StepsFor). L'inverse produirait un schéma player
	// partiel (sans progression) → régression Ascension h5.
	if Set().OwnsTarget(migration.TargetPlayer) {
		t.Fatal("le set h5 ne doit PAS posséder le target player (fallback HINF requis pour le schéma Progression V2)")
	}

	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := migration.RunForTitleDB(db, halo5.TitleSlug, migration.TargetPlayer); err != nil {
		t.Fatalf("RunForTitleDB(%s, player): %v", halo5.TitleSlug, err)
	}

	// Tables Progression V2 (create_progression_player_schema) + racine career player.
	for _, tbl := range []string{
		"streak", "streak_history", "record_history", "milestone_earned",
		"career_progression",
	} {
		if !tableExists(t, db, tbl) {
			t.Errorf("table player %q absente du player h5 — fallback HINF cassé (schéma Progression V2 non provisionné)", tbl)
		}
	}

	// Vue append-only streak_latest (create_streak_history_append_only) : lecture EXACTE
	// de StreaksRepo.List (FROM streak_latest). Son absence = GET /players/{slug}/streaks
	// 500 → HomeAscensionWidget auto-masqué (symptôme historique du backlog).
	if !viewExists(t, db, "streak_latest") {
		t.Error("vue streak_latest absente du player h5 — GET /streaks renverrait 500 (StreaksRepo.List lit FROM streak_latest)")
	}
}
