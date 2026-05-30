//go:build cgo

// diag_lusr_slots — audit READ-ONLY de la couverture LUSR v2 dans match_skill_rank.
//
// Contexte : la table match_skill_rank est append-only et porte, sous Stratégie C
// (ADR 0024), DEUX rows par match v2 — rating_type='LUSR' (slot lu par l'UI) et
// rating_type='LUSR_V2' (audit). Ce diag répond à deux questions :
//
//   - ANOMALIE A : des matchs dont la SEULE row est 'LUSR_V2' (slot 'LUSR'
//     manquant) ? La vue match_skill_rank_latest les afficherait alors avec la
//     note "LUSR_V2". Doit être 0.
//   - ANOMALIE B : des matchs avec un slot 'LUSR' mais SANS audit 'LUSR_V2'
//     (= jamais passés par le pipeline v2 canonical → valeur encore v1). C'est
//     le "v1 résiduel" réel : ces matchs attendent le backfill canonical
//     (cmd/lusr_v2_canonical_backfill --commit).
//
// Lecture seule, ouverture READ_ONLY : à lancer SERVEUR ARRÊTÉ (le watcher tient
// un lock RW exclusif sur les player DBs). Sortie texte console.
//
// Usage (depuis apps/go-api) : go run ./cmd/diag_lusr_slots [-data-root ../..] [GT...]
package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"flag"
	"fmt"
	"path/filepath"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

var defaultPlayers = []string{"Madina97294", "Chocoboflor", "JGtm", "XxDaemonGamerxX"}

func openRO(path string) (*sql.DB, error) {
	connector, err := duckdb.NewConnector(path+"?access_mode=READ_ONLY", func(driver.ExecerContext) error { return nil })
	if err != nil {
		return nil, err
	}
	return sql.OpenDB(connector), nil
}

func main() {
	dataRoot := flag.String("data-root", "../..", "racine du repo (depuis apps/go-api : ../..)")
	flag.Parse()
	players := flag.Args()
	if len(players) == 0 {
		players = defaultPlayers
	}
	ctx := context.Background()

	for _, player := range players {
		path := filepath.Join(*dataRoot, "data", "titles", "halo_infinite", "players", player, "stats.duckdb")
		db, err := openRO(path)
		if err != nil {
			fmt.Printf("\n══ %s ══\n   open échoué (serveur lancé ? DB absente ?) : %v\n", player, err)
			continue
		}
		fmt.Printf("\n══════════ %s ══════════\n", player)
		reportRaw(ctx, db)
		reportLatest(ctx, db)
		reportAnomalyA(ctx, db)
		reportAnomalyB(ctx, db)
		db.Close()
	}
}

// reportRaw : répartition physique par rating_type (table append-only).
func reportRaw(ctx context.Context, db *sql.DB) {
	fmt.Println("\n-- [1] match_skill_rank (brut, append-only) --")
	rows, err := db.QueryContext(ctx, `
		SELECT COALESCE(rating_type,'NULL') AS rt, COUNT(*) nb,
		       COUNT(DISTINCT match_id) nb_match,
		       ROUND(MIN(rating_value),1) mn, ROUND(MAX(rating_value),1) mx,
		       MIN(written_at) first_w, MAX(written_at) last_w
		FROM match_skill_rank GROUP BY 1 ORDER BY 1`)
	if err != nil {
		fmt.Printf("   erreur: %v\n", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var rt string
		var nb, nbMatch int
		var mn, mx sql.NullFloat64
		var firstW, lastW sql.NullString
		_ = rows.Scan(&rt, &nb, &nbMatch, &mn, &mx, &firstW, &lastW)
		fmt.Printf("   %-8s rows=%-5d matchs=%-5d val=[%.0f..%.0f] written=[%s .. %s]\n",
			rt, nb, nbMatch, mn.Float64, mx.Float64, short(firstW), short(lastW))
	}
}

// reportLatest : ce que l'UI lit (1 ligne / match, priorité CSR>LUSR>LUSR_V2).
func reportLatest(ctx context.Context, db *sql.DB) {
	fmt.Println("-- [2] match_skill_rank_latest (vue UI) --")
	rows, err := db.QueryContext(ctx, `
		SELECT rating_type, COUNT(*) nb FROM match_skill_rank_latest
		GROUP BY 1 ORDER BY 1`)
	if err != nil {
		fmt.Printf("   erreur: %v\n", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var rt string
		var nb int
		_ = rows.Scan(&rt, &nb)
		fmt.Printf("   %-8s matchs affichés=%d\n", rt, nb)
	}
}

// reportAnomalyA : matchs dont la note affichée serait 'LUSR_V2' = slot 'LUSR'
// (et 'CSR') physiquement absent. Doit être 0.
func reportAnomalyA(ctx context.Context, db *sql.DB) {
	var n int
	_ = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM (
			SELECT match_id FROM match_skill_rank
			GROUP BY match_id
			HAVING SUM(CASE WHEN rating_type='LUSR_V2' THEN 1 ELSE 0 END) > 0
			   AND SUM(CASE WHEN rating_type='LUSR'    THEN 1 ELSE 0 END) = 0
			   AND SUM(CASE WHEN rating_type='CSR'     THEN 1 ELSE 0 END) = 0
		)`).Scan(&n)
	fmt.Printf("-- [3] ANOMALIE A : matchs LUSR_V2 sans slot LUSR (ni CSR) = %d --\n", n)
}

// reportAnomalyB : matchs avec slot 'LUSR' mais SANS audit 'LUSR_V2' = v1 pur
// (jamais traités par le pipeline v2 canonical). C'est le "v1 résiduel" réel.
func reportAnomalyB(ctx context.Context, db *sql.DB) {
	var n int
	_ = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM (
			SELECT match_id FROM match_skill_rank
			GROUP BY match_id
			HAVING SUM(CASE WHEN rating_type='LUSR'    THEN 1 ELSE 0 END) > 0
			   AND SUM(CASE WHEN rating_type='LUSR_V2' THEN 1 ELSE 0 END) = 0
			   AND SUM(CASE WHEN rating_type='CSR'     THEN 1 ELSE 0 END) = 0
		)`).Scan(&n)
	fmt.Printf("-- [4] ANOMALIE B : matchs LUSR sans audit LUSR_V2 (= v1 résiduel) = %d --\n", n)
}

func short(s sql.NullString) string {
	if !s.Valid {
		return "—"
	}
	if len(s.String) > 19 {
		return s.String[:19]
	}
	return s.String
}
