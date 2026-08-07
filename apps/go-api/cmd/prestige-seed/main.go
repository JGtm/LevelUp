//go:build ignore

// cmd/prestige-seed — initialise des données Prestige de test pour un joueur.
//
// Usage (depuis apps/go-api/, SERVEUR ARRÊTÉ) :
//
//	go run ./cmd/prestige-seed/main.go
//	go run ./cmd/prestige-seed/main.go --player Chocoboflor --pp 3500
//	go run ./cmd/prestige-seed/main.go --delete
//
// Ce que ça fait :
//   - Upsert dans user_prestige  (shared_social.duckdb) : PP + niveau
//   - Upsert dans prestige_events (shared_social.duckdb) : 1 événement source="seed"
//   - Insert  dans arc            (stats.duckdb joueur)   : 1 arc actif
//   - Insert  dans challenge      (stats.duckdb joueur)   : 1 défi actif Heroic
//
// Prérequis : SERVEUR ARRÊTÉ (DuckDB = 1 writer à la fois).
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	player := flag.String("player", "JGtm", "Gamertag du joueur à initialiser")
	pp := flag.Int("pp", 1500, "Total PP à attribuer (1500 = Vétéran niv.2)")
	del := flag.Bool("delete", false, "Supprime les données de seed pour ce joueur")
	flag.Parse()

	socialDB := fmt.Sprintf("../../data/titles/halo_infinite/warehouse/shared_social.duckdb")
	playerDB := fmt.Sprintf("../../data/titles/halo_infinite/players/%s/stats.duckdb", *player)

	if *del {
		runDelete(socialDB, playerDB, *player)
		return
	}
	runSeed(socialDB, playerDB, *player, *pp)
}

func runSeed(socialDB, playerDB, player string, pp int) {
	// ── 1. shared_social.duckdb : user_prestige + prestige_events ──────────────

	social, err := sql.Open("duckdb", socialDB)
	if err != nil {
		fatalf("social open: %v", err)
	}
	defer social.Close()

	mustExec(social, `
		CREATE TABLE IF NOT EXISTS user_prestige (
			user_id       VARCHAR NOT NULL,
			title_slug    VARCHAR NOT NULL,
			total_pp      INTEGER NOT NULL DEFAULT 0,
			current_level INTEGER NOT NULL DEFAULT 0,
			updated_at    TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			PRIMARY KEY (user_id, title_slug)
		)`)

	mustExec(social, `
		CREATE TABLE IF NOT EXISTS prestige_events (
			id          VARCHAR PRIMARY KEY,
			user_id     VARCHAR NOT NULL,
			title_slug  VARCHAR NOT NULL,
			source_type VARCHAR NOT NULL,
			source_id   VARCHAR,
			pp_amount   INTEGER NOT NULL DEFAULT 0,
			tier        VARCHAR,
			created_at  TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		)`)

	level := levelFromPP(pp)

	_, err = social.Exec(`
		INSERT INTO user_prestige (user_id, title_slug, total_pp, current_level, updated_at)
		VALUES (?, 'halo_infinite', ?, ?, ?)
		ON CONFLICT (user_id, title_slug) DO UPDATE SET
			total_pp      = EXCLUDED.total_pp,
			current_level = EXCLUDED.current_level,
			updated_at    = EXCLUDED.updated_at`,
		player, pp, level, time.Now())
	if err != nil {
		fatalf("upsert user_prestige: %v", err)
	}
	fmt.Printf("✓ user_prestige : %s → %d PP (niv. %d)\n", player, pp, level)

	evtID := fmt.Sprintf("seed_%s_%d", player, time.Now().UnixMilli())
	_, err = social.Exec(`
		INSERT INTO prestige_events (id, user_id, title_slug, source_type, pp_amount, created_at)
		VALUES (?, ?, 'halo_infinite', 'seed', ?, ?)
		ON CONFLICT (id) DO NOTHING`,
		evtID, player, pp, time.Now())
	if err != nil {
		fatalf("insert prestige_events: %v", err)
	}
	fmt.Printf("✓ prestige_events: événement seed inséré (%s)\n", evtID)

	// ── 2. stats.duckdb joueur : arc + challenge ────────────────────────────────

	if _, err := os.Stat(playerDB); os.IsNotExist(err) {
		fatalf("DB joueur introuvable : %s\n  → vérifiez que le joueur existe", playerDB)
	}

	pdb, err := sql.Open("duckdb", playerDB)
	if err != nil {
		fatalf("player open: %v", err)
	}
	defer pdb.Close()

	mustExec(pdb, `
		CREATE TABLE IF NOT EXISTS arc (
			id           VARCHAR PRIMARY KEY,
			user_id      VARCHAR NOT NULL,
			title_slug   VARCHAR NOT NULL,
			title        VARCHAR NOT NULL,
			description  VARCHAR,
			is_preset    BOOLEAN NOT NULL DEFAULT FALSE,
			preset_id    VARCHAR,
			created_at   TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			completed_at TIMESTAMP
		)`)

	mustExec(pdb, `
		CREATE TABLE IF NOT EXISTS challenge (
			id                       VARCHAR PRIMARY KEY,
			user_id                  VARCHAR NOT NULL,
			title_slug               VARCHAR NOT NULL,
			arc_id                   VARCHAR,
			position                 INTEGER,
			template_id              VARCHAR,
			metric                   VARCHAR NOT NULL,
			target                   DOUBLE NOT NULL,
			target_per_member        DOUBLE,
			window_type              VARCHAR NOT NULL,
			window_value             VARCHAR,
			cadence                  VARCHAR NOT NULL DEFAULT 'free',
			eval_type                VARCHAR NOT NULL DEFAULT 'threshold',
			mode                     VARCHAR NOT NULL DEFAULT 'libre',
			tier                     VARCHAR,
			data_tier                VARCHAR NOT NULL DEFAULT 'full',
			label                    VARCHAR,
			status                   VARCHAR NOT NULL DEFAULT 'draft',
			created_at               TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			committed_at             TIMESTAMP,
			completed_at             TIMESTAMP,
			expired_at               TIMESTAMP,
			abandoned_at             TIMESTAMP,
			last_palier_recompute_at TIMESTAMP,
			is_private               BOOLEAN DEFAULT FALSE
		)`)

	arcID := fmt.Sprintf("arc_seed_%s", player)
	_, err = pdb.Exec(`
		INSERT INTO arc (id, user_id, title_slug, title, description, is_preset, created_at)
		VALUES (?, ?, 'halo_infinite', 'Ascension Heroic', 'Arc de test — progresser en mode Heroic', FALSE, ?)
		ON CONFLICT (id) DO NOTHING`,
		arcID, player, time.Now())
	if err != nil {
		fatalf("insert arc: %v", err)
	}
	fmt.Printf("✓ arc      : %s (id=%s)\n", player, arcID)

	chID := fmt.Sprintf("ch_seed_%s", player)
	_, err = pdb.Exec(`
		INSERT INTO challenge
			(id, user_id, title_slug, arc_id, metric, target, window_type, cadence, eval_type, mode,
			 tier, data_tier, label, status, created_at, committed_at)
		VALUES (?, ?, 'halo_infinite', ?, 'FieldKDA', 1.5, 'rolling_days', 'weekly', 'threshold', 'libre',
			'heroic', 'full', 'Maintenir un FDA ≥ 1.5 sur la semaine', 'active', ?, ?)
		ON CONFLICT (id) DO NOTHING`,
		chID, player, arcID, time.Now(), time.Now())
	if err != nil {
		fatalf("insert challenge: %v", err)
	}
	fmt.Printf("✓ challenge: %s (id=%s, Heroic FDA≥1.5 hebdo)\n", player, chID)

	fmt.Println("\nSeed terminé. Le module Prestige est activé par défaut — il suffit de redémarrer le serveur.")
}

func runDelete(socialDB, playerDB, player string) {
	social, err := sql.Open("duckdb", socialDB)
	if err != nil {
		fatalf("social open: %v", err)
	}
	defer social.Close()
	mustExec(social, `DELETE FROM user_prestige    WHERE user_id = ? AND title_slug = 'halo_infinite'`, player)
	mustExec(social, `DELETE FROM prestige_events  WHERE user_id = ? AND title_slug = 'halo_infinite' AND source_type = 'seed'`, player)
	fmt.Printf("✓ supprimé user_prestige + events seed pour %s\n", player)

	if _, err := os.Stat(playerDB); !os.IsNotExist(err) {
		pdb, err := sql.Open("duckdb", playerDB)
		if err == nil {
			defer pdb.Close()
			mustExec(pdb, `DELETE FROM arc       WHERE id LIKE 'arc_seed_%'  AND user_id = ?`, player)
			mustExec(pdb, `DELETE FROM challenge  WHERE id LIKE 'ch_seed_%'   AND user_id = ?`, player)
			fmt.Printf("✓ supprimé arc + challenge seed pour %s\n", player)
		}
	}
}

// levelFromPP reproduit DefaultTuning().Levels.Thresholds = {0,500,1500,3000,6000,12000}.
func levelFromPP(pp int) int {
	thresholds := []int{0, 500, 1500, 3000, 6000, 12000}
	level := 0
	for i, t := range thresholds {
		if pp >= t {
			level = i
		}
	}
	return level
}

func mustExec(db *sql.DB, query string, args ...any) {
	if _, err := db.Exec(query, args...); err != nil {
		fatalf("exec [%.60s…]: %v", query, err)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ERREUR: "+format+"\n", args...)
	os.Exit(1)
}
