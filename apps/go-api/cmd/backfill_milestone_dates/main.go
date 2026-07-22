//go:build cgo

// backfill_milestone_dates — backfill one-off des dates de franchissement des
// jalons (A6). Toutes les earned_at ont été estampillées à la date du premier run
// (seed) ; on recalcule la vraie date de franchissement par jalon en rejouant les
// matchs du joueur (cumul par métrique, fragment timezone canonique). Non
// dérivable → earned_at NULL (le front n'affiche alors pas de date).
//
//	Usage : go run -tags cgo ./cmd/backfill_milestone_dates \
//		    [--data-root ../../data/titles/halo_infinite] [--apply]
//
// Sans --apply : DRY-RUN (aucune mutation). Le serveur Go DOIT être stoppé.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/ops"
)

func main() {
	dataRoot := flag.String("data-root", "../../data/titles/halo_infinite", "racine du titre (data/titles/{slug})")
	apply := flag.Bool("apply", false, "applique le backfill (sinon dry-run, aucune mutation)")
	flag.Parse()

	slug := filepath.Base(filepath.Clean(*dataRoot))
	hp := games.EffectiveHpToKill(slug)
	mode := "DRY-RUN (aucune mutation)"
	if *apply {
		mode = "APPLY (recalcul earned_at)"
	}
	fmt.Printf("=== backfill_milestone_dates ===\n")
	fmt.Printf("data_root: %s (slug=%s, hp=%g)\n", *dataRoot, slug, hp)
	fmt.Printf("mode     : %s\n\n", mode)

	ctx := context.Background()

	// Catalogue (metadata) : milestone_id -> métrique + seuil.
	metaPath := filepath.Join(*dataRoot, "warehouse", "metadata.duckdb")
	catalog := loadCatalog(ctx, metaPath, slug)
	fmt.Printf("catalogue : %d jalons\n", len(catalog))

	// Matchs (shared) : réutilisé pour tous les joueurs du titre.
	sharedPath := filepath.Join(*dataRoot, "warehouse", "shared_matches_v2.duckdb")
	shared, err := sql.Open("duckdb", sharedPath)
	if err != nil {
		log.Fatalf("open shared: %v", err)
	}
	defer shared.Close()

	playersDir := filepath.Join(*dataRoot, "players")
	entries, err := os.ReadDir(playersDir)
	if err != nil {
		log.Fatalf("lecture playersDir: %v", err)
	}
	totalUpdated, totalNulled := 0, 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		statsPath := filepath.Join(playersDir, e.Name(), "stats.duckdb")
		if _, statErr := os.Stat(statsPath); statErr != nil {
			continue
		}
		u, n := runBackfill(ctx, shared, statsPath, e.Name(), slug, catalog, hp, *apply)
		totalUpdated += u
		totalNulled += n
	}

	fmt.Println()
	fmt.Printf("=== Total : %d date(s) recalculée(s), %d NULL (non dérivable) ===\n", totalUpdated, totalNulled)
	if !*apply {
		fmt.Println("\nRelance avec --apply pour confirmer (serveur Go stoppé).")
	}
}

// loadCatalog lit milestone_catalog (metadata) pour un titre.
func loadCatalog(ctx context.Context, metaPath, slug string) map[string]ops.MilestoneTarget {
	db, err := sql.Open("duckdb", metaPath)
	if err != nil {
		log.Fatalf("open metadata: %v", err)
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx,
		`SELECT id, metric, threshold FROM milestone_catalog WHERE title_slug = ?`, slug)
	if err != nil {
		log.Fatalf("load catalog: %v", err)
	}
	defer rows.Close()
	out := map[string]ops.MilestoneTarget{}
	for rows.Next() {
		var t ops.MilestoneTarget
		if err := rows.Scan(&t.MilestoneID, &t.Metric, &t.Threshold); err != nil {
			log.Fatalf("scan catalog: %v", err)
		}
		out[t.MilestoneID] = t
	}
	return out
}

// runBackfill ouvre la stats.duckdb du joueur et exécute le backfill. Imprime le
// résultat par xuid. Retourne (updated, nulled) agrégés.
func runBackfill(ctx context.Context, shared *sql.DB, statsPath, gamertag, slug string,
	catalog map[string]ops.MilestoneTarget, hp float64, apply bool) (int, int) {
	stats, err := sql.Open("duckdb", statsPath)
	if err != nil {
		log.Fatalf("open %s: %v", statsPath, err)
	}
	stats.SetMaxOpenConns(1)
	defer stats.Close()

	results, err := ops.BackfillMilestoneDates(ctx, shared, stats, slug, catalog, hp, apply)
	if err != nil {
		log.Fatalf("backfill %s: %v", statsPath, err)
	}
	updated, nulled := 0, 0
	if len(results) == 0 {
		fmt.Printf("--- %s : aucun jalon débloqué\n", gamertag)
		return 0, 0
	}
	for _, r := range results {
		fmt.Printf("--- %s (xuid=%s) : %d recalculé(s), %d NULL / %d jalon(s)\n",
			gamertag, r.XUID, r.Updated, r.Nulled, r.Total)
		updated += r.Updated
		nulled += r.Nulled
	}
	return updated, nulled
}
