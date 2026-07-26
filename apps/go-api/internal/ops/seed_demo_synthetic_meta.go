// Package ops — seed_demo_synthetic_meta.go : metadata.duckdb synthétique.
//
// Recette (verdict d'investigation 2026-07-13) : une metadata vierge MIGRÉE
// (RunForTitleDB TargetMetadata) porte déjà weapon_labels, weapon_registry,
// mode_name_tr, csr_placement_thresholds, playlists ranked+FR, milestones,
// prestige, et le SCHÉMA vide du reste. On complète avec :
//   - `seed citation-mappings` + `seed rank-translations` (sources Go embarquées,
//     CI-safe, aucun fichier externe) ;
//   - INSERT synthétiques minimaux dans medal_definitions (les medal_name_id
//     référencés par le corpus) et career_ranks (créée à la main : aucune migration
//     ne la crée, et son seed lit un JSON non committé).
//
// Les NOMS carte/mode/playlist ne passent PAS par la metadata : ils sont
// dénormalisés dans match_registry (map_name/playlist_name/..._fr) — écrits par le
// générateur shared.
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"

	_ "github.com/duckdb/duckdb-go/v2"
)

// writeSyntheticMetadata crée metadata.duckdb : migrations title + seeds Go +
// medals/ranks synthétiques. Fresh (idempotence reseed).
func writeSyntheticMetadata(ctx context.Context, metaPath string) error {
	if err := os.MkdirAll(filepath.Dir(metaPath), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	if err := removeDuckDBForFreshWrite(metaPath); err != nil {
		return err
	}

	// 1. Migrations metadata du titre par défaut (title steps inclus : weapon_labels,
	//    mode_name_tr, csr thresholds, ranked playlists, schéma citation/career_rank...).
	//    Les providers (SetTitleStepsProvider, SetCareerRankTranslationsProvider) sont
	//    posés au boot de la CLI (cmd/levelup/main.go).
	if err := func() error {
		db, err := sql.Open("duckdb", metaPath)
		if err != nil {
			return fmt.Errorf("open: %w", err)
		}
		defer db.Close()
		if err := migration.RunForTitleDB(db, titlePkg.DefaultSlug, migration.TargetMetadata); err != nil {
			return fmt.Errorf("migrations metadata: %w", err)
		}
		return nil
	}(); err != nil {
		return err
	}

	// 2. Seeds Go embarqués (citations + libellés de rangs FR/EN). Ces fonctions
	//    ouvrent leur propre connexion.
	if _, err := SeedCitationMappings(ctx, SeedOptions{MetaDBPath: metaPath}); err != nil {
		return fmt.Errorf("seed citation-mappings: %w", err)
	}
	if _, err := SeedRankTranslations(ctx, SeedOptions{MetaDBPath: metaPath}); err != nil {
		return fmt.Errorf("seed rank-translations: %w", err)
	}

	// 3. medals + career_ranks synthétiques.
	db, err := sql.Open("duckdb", metaPath)
	if err != nil {
		return fmt.Errorf("reopen: %w", err)
	}
	defer db.Close()
	if err := insertSyntheticMedals(ctx, db); err != nil {
		return fmt.Errorf("medals: %w", err)
	}
	if err := insertSyntheticCareerRanks(ctx, db); err != nil {
		return fmt.Errorf("career_ranks: %w", err)
	}
	return nil
}

// insertSyntheticMedals peuple medal_definitions pour les medal_name_id du corpus
// (le seed migration ne pose que la médaille custom Vengeur). INSERT-or-ignore sur
// la PK medal_name_id (table fraîche → pas de surface ART concurrente).
func insertSyntheticMedals(ctx context.Context, db *sql.DB) error {
	type med struct {
		id              int64
		fr, en          string
		diffIdx, typIdx int
	}
	meds := []med{
		{synthMedalIDs[0], "Double meurtre", "Double Kill", 0, 2},
		{synthMedalIDs[1], "Triple meurtre", "Triple Kill", 1, 2},
		{synthMedalIDs[2], "Série de meurtres", "Killing Spree", 0, 0},
		{synthMedalIDs[3], "Tir à la tête", "Headshot", 0, 4},
	}
	for _, m := range meds {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO medal_definitions
				(medal_name_id, name_fr, name_en, description_fr, description_en,
				 is_custom, difficulty_index, type_index, personal_score)
			VALUES (?, ?, ?, '', '', FALSE, ?, ?, 0)
			ON CONFLICT (medal_name_id) DO NOTHING`,
			m.id, m.fr, m.en, m.diffIdx, m.typIdx); err != nil {
			return fmt.Errorf("insert medal %d: %w", m.id, err)
		}
	}
	// Renseigner difficulty/medal_type (colonnes ajoutées par enrich_medal_definitions_v2)
	// pour les lignes synthétiques (le seed migration ne les recalcule que sur son passage).
	if _, err := db.ExecContext(ctx, `
		UPDATE medal_definitions SET
			difficulty = CASE difficulty_index WHEN 1 THEN 'Heroic' WHEN 2 THEN 'Legendary' ELSE 'Normal' END,
			medal_type = CASE type_index WHEN 0 THEN 'spree' WHEN 2 THEN 'multikill' WHEN 4 THEN 'skill' ELSE '' END
		WHERE difficulty IS NULL`); err != nil {
		return fmt.Errorf("enrich medals: %w", err)
	}
	return nil
}

// insertSyntheticCareerRanks crée + peuple career_ranks (aucune migration ne la
// crée ; son seed lit un JSON non committé). Schéma COMPLET (tier_type,
// large_icon_path, adornment_icon_path) — sinon EnrichFromMetadata plante en
// Binder Error (piège documenté ops/seed.go). 4 rangs suffisent pour l'enrichissement.
func insertSyntheticCareerRanks(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS career_ranks (
		rank_id INTEGER PRIMARY KEY,
		title_en VARCHAR, subtitle_en VARCHAR,
		tier VARCHAR, tier_type VARCHAR, grade INTEGER,
		xp_required INTEGER, icon_path VARCHAR,
		large_icon_path VARCHAR, adornment_icon_path VARCHAR
	)`); err != nil {
		return fmt.Errorf("create career_ranks: %w", err)
	}
	type rk struct {
		id      int
		titleEN string
		tier    string
		grade   int
		xp      int
	}
	ranks := []rk{
		{40, "Corporal", tierNameBronze, 3, 400000},
		{120, "Sergeant", "Silver", 1, 1200000},
		{200, "Lieutenant", "Gold", 2, 2400000},
		{272, "Hero", "Onyx", 1, 8000000},
	}
	for _, r := range ranks {
		icon := fmt.Sprintf("career_rank/rank_%d.png", r.id)
		if _, err := db.ExecContext(ctx, `INSERT OR IGNORE INTO career_ranks
			(rank_id, title_en, subtitle_en, tier, tier_type, grade, xp_required, icon_path, large_icon_path, adornment_icon_path)
			VALUES (?, ?, '', ?, 'Normal', ?, ?, ?, ?, '')`,
			r.id, r.titleEN, r.tier, r.grade, r.xp, icon, icon); err != nil {
			return fmt.Errorf("insert rank %d: %w", r.id, err)
		}
	}
	return nil
}
