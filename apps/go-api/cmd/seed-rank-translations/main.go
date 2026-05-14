//go:build cgo

// cmd/seed-rank-translations — Peuple career_rank_translations offline à partir
// des libellés codés en dur (correspondant au CSV Halo_Ranks_FR).
//
// Utile quand refresh-career-ranks n'est pas jouable (tokens invalides, API
// GameCMS down) et que la table est vide.
//
// Usage :
//
//	go run -tags cgo ./cmd/seed-rank-translations
//	go run -tags cgo ./cmd/seed-rank-translations --dry-run
//
// IMPORTANT : stopper le serveur API avant de lancer (metadata.duckdb est
// ouvert en RW au boot — le pool détient un verrou exclusif).
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
)

// Grades militaires — 15 paliers par tranche de rang (tiers × grades × 3 niveaux)
var gradesFR = [15]string{
	"Cadet", "Soldat", "Caporal suppléant", "Caporal",
	"Sergent", "Sergent-chef", "Sergent d'artillerie", "Adjudant",
	"Lieutenant", "Capitaine", "Lieutenant-major", "Lieutenant-colonel",
	"Colonel", "Général de brigade", "Général",
}
var gradesEN = [15]string{
	"Cadet", "Private", "Lance Corporal", "Corporal",
	"Sergeant", "Staff Sergeant", "Gunnery Sergeant", "Master Sergeant",
	"Lieutenant", "Captain", "Major", "Lieutenant Colonel",
	"Colonel", "Brigadier General", "General",
}

// 6 tranches de rang (Bronze→Onyx), chacune = 15 grades × 3 niveaux = 45 rangs
var tiersFR = [6]string{"Bronze", "Argent", "Or", "Platine", "Diamant", "Onyx"}
var tiersEN = [6]string{"Bronze", "Silver", "Gold", "Platinum", "Diamond", "Onyx"}

type rankRow struct {
	id      int
	titleFR string
	titleEN string
	tierFR  string
	tierEN  string
}

// buildRanks génère les 272 rangs (0 + 270 normaux + Hero).
//
// Algorithme (rangs 1–270) :
//
//	tierIdx   = (i-1) / 45          → 0=Bronze … 5=Onyx
//	posInTier = (i-1) % 45
//	gradeIdx  = posInTier / 3       → 0=Cadet … 14=Général
//	level     = posInTier%3 + 1     → 1, 2 ou 3
func buildRanks() []rankRow {
	rows := make([]rankRow, 0, 272)

	// Rang 0 — Recrue
	rows = append(rows, rankRow{0, "Recrue", "Recruit", "Bronze", "Bronze"})

	// Rangs 1–270
	for i := 1; i <= 270; i++ {
		tierIdx := (i - 1) / 45
		posInTier := (i - 1) % 45
		gradeIdx := posInTier / 3
		level := posInTier%3 + 1

		titleFR := fmt.Sprintf("%s %d", gradesFR[gradeIdx], level)
		titleEN := fmt.Sprintf("%s %d", gradesEN[gradeIdx], level)

		rows = append(rows, rankRow{
			id:      i,
			titleFR: titleFR,
			titleEN: titleEN,
			tierFR:  tiersFR[tierIdx],
			tierEN:  tiersEN[tierIdx],
		})
	}

	// Rang 271 — Héros
	rows = append(rows, rankRow{271, "Héros", "Hero", "Onyx", "Onyx"})

	return rows
}

func main() {
	fs := flag.NewFlagSet("seed-rank-translations", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Affiche les rangs sans écrire en base")
	titleID := fs.String("title-id", titlePkg.DefaultSlug, "Title ID (ex: halo_infinite)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(1)
	}

	if err := run(*titleID, *dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "erreur : %v\n", err)
		os.Exit(1)
	}
}

func run(titleID string, dryRun bool) error {
	ranks := buildRanks()

	if dryRun {
		for _, r := range ranks {
			fmt.Printf("rank %3d | FR: %-30s [%s] | EN: %-30s [%s]\n",
				r.id, r.titleFR, r.tierFR, r.titleEN, r.tierEN)
		}
		fmt.Printf("\n%d rangs (dry-run — rien écrit)\n", len(ranks))
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config.Load: %w", err)
	}
	if cfg.RepoRoot == "" {
		return fmt.Errorf("LEVELUP_REPO_ROOT non défini")
	}

	metaPath := titlePkg.NewPathResolver(cfg.RepoRoot).MetadataDBPath(titleID)
	if _, err := os.Stat(metaPath); err != nil {
		return fmt.Errorf("metadata.duckdb introuvable (%s): %w", metaPath, err)
	}

	db, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		return fmt.Errorf("ouverture metadata.duckdb: %w", err)
	}
	defer db.Close()

	ctx := context.Background()

	if _, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS career_rank_translations (
			rank_id    INTEGER NOT NULL,
			lang       VARCHAR NOT NULL,
			title      VARCHAR,
			subtitle   VARCHAR,
			tier       VARCHAR,
			fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (rank_id, lang)
		)
	`); err != nil {
		return fmt.Errorf("ensure table: %w", err)
	}

	var before int
	_ = db.QueryRow(ctx, "SELECT COUNT(*) FROM career_rank_translations").Scan(&before)
	fmt.Printf("Avant : %d lignes dans career_rank_translations\n", before)

	inserted := 0
	for _, r := range ranks {
		for _, lang := range []struct {
			code  string
			title string
			tier  string
		}{
			{"fr", r.titleFR, r.tierFR},
			{"en", r.titleEN, r.tierEN},
		} {
			if _, err := db.Exec(ctx, `
				INSERT OR REPLACE INTO career_rank_translations
					(rank_id, lang, title, subtitle, tier, fetched_at)
				VALUES (?, ?, ?, '', ?, CURRENT_TIMESTAMP)
			`, r.id, lang.code, lang.title, lang.tier); err != nil {
				return fmt.Errorf("upsert rank %d lang %s: %w", r.id, lang.code, err)
			}
			inserted++
		}
	}

	var after int
	_ = db.QueryRow(ctx, "SELECT COUNT(*) FROM career_rank_translations").Scan(&after)
	fmt.Printf("Apres  : %d lignes — %d upsertées (%d rangs × 2 langues)\n",
		after, inserted, len(ranks))
	return nil
}
