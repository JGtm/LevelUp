// Package ops — seed.go : population des référentiels metadata.duckdb.
//
// Portage des scripts populate_*.py Python.
//
// Fonctions :
//   - SeedCareerRanks      → populate_career_ranks.py
//   - SeedCitationMappings → populate_citation_mappings.py
//   - SeedWeaponLabels     → populate_weapon_labels (via migration), ici validation
//   - SeedMedalDefinitions → populate_medal_metadata.py
package ops

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"levelup/go-api/internal/migration"

	_ "github.com/duckdb/duckdb-go/v2"
)

// SeedOptions configure une opération de seed.
type SeedOptions struct {
	MetaDBPath string
	DataDir    string // data/ root pour les JSON sources
}

// SeedResult résume le résultat d'un seed.
type SeedResult struct {
	Component string
	Inserted  int
	Skipped   int
	Message   string
}

// Composants Seed*. Centralisés ici pour éviter la duplication littérale
// du nom dans chaque SeedResult retourné par les SeedXxx (cf. lint goconst).
const (
	componentCareerRanks      = "career_ranks"
	componentCitationMappings = "citation_mappings"
	componentMedalDefinitions = "medal_definitions"
	componentRankTranslations = "career_rank_translations"
)

// Mapping types pour citation_mappings.mapping_type.
const (
	mappingTypeAward      = "award"
	mappingTypeStat       = "stat"
	mappingTypePVEStat    = "pve_stat"
	mappingTypeWeaponStat = "weapon_stat"
	mappingTypeMedal      = "medal"
	mappingTypeComposite  = "composite"
	mappingTypeCustom     = "custom"
)

// Catégories citation_mappings.category (libellés FR — UI Cockpit).
const (
	citationCatModeJeu          = "Mode de jeu"
	citationCatVehicule         = "Véhicule"
	citationCatArme             = "Arme"
	citationCatMultijoueur      = "Multijoueur"
	citationCatSpartanCompanies = "Spartan Companies"
	citationCatEnnemi           = "Ennemi"
)

// Subcategory libellés FR récurrents pour citation_mappings.subcategory.
const (
	citationSubGeneral    = "Général"
	citationSubUNSC       = "UNSC"
	citationSubCovenant   = "Covenant"
	citationSubBanished   = "Banished"
	citationSubParia      = "Paria"
	citationSubForerunner = "Forerunner"
)

// Award categories tag (pour award_category dans la fonction d'award seed).
const (
	awardCategoryObjective = "objective"
	awardCategoryVehicle   = "vehicle"
)

// Tier targets récurrents (CSV de seuils par défaut, partagés par plusieurs
// catégories de citations) — centralisés pour réduire la duplication littérale.
const (
	tierTargets3_6_9_15_30      = "3,6,9,15,30"
	tierTargets5_10_15_25_50    = "5,10,15,25,50"
	tierTargets10_20_30_50_100  = "10,20,30,50,100"
	tierTargets10_20_40_80_200  = "10,20,40,80,200"
	tierTargets10_25_50_100_250 = "10,25,50,100,250"
)

// ─────────────────────────────────────────────────────────────────────────────
// career_ranks
// ─────────────────────────────────────────────────────────────────────────────

// careerRankJSON représente une entrée dans career_ranks_metadata.json.
//
// tier_type / large_icon_path / adornment_icon_path peuvent être absents des
// JSON legacy (champs facultatifs) → insérés vides. Le schéma career_ranks doit
// néanmoins les déclarer : c'est le contrat lu par EnrichFromMetadata (tier_type,
// adornment_icon_path) et LoadCareerRankImageURLs (large_icon_path). Sans ces
// colonnes, l'enrichissement carrière (page + home) plante en Binder Error.
type careerRankJSON struct {
	RankID        int    `json:"rank_id"`
	TitleEN       string `json:"title_en"`
	Subtitle      string `json:"subtitle"`
	Tier          string `json:"tier"`
	TierType      string `json:"tier_type"`
	Grade         int    `json:"grade"`
	XPRequired    int64  `json:"xp_required"`
	IconPath      string `json:"icon_path"`
	LargeIconPath string `json:"large_icon_path"`
	AdornmentPath string `json:"adornment_icon_path"`
}

// SeedCareerRanks peuple la table career_ranks depuis le JSON source.
// Portage de populate_career_ranks() Python.
func SeedCareerRanks(ctx context.Context, opts SeedOptions) (SeedResult, error) {
	jsonPath := opts.DataDir + "/cache/career_ranks_metadata.json"
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return SeedResult{Component: componentCareerRanks}, fmt.Errorf("lecture %s: %w", jsonPath, err)
	}
	var ranks []careerRankJSON
	if err := json.Unmarshal(data, &ranks); err != nil {
		return SeedResult{Component: componentCareerRanks}, fmt.Errorf("parse JSON: %w", err)
	}

	db, err := sql.Open("duckdb", opts.MetaDBPath)
	if err != nil {
		return SeedResult{Component: componentCareerRanks}, fmt.Errorf("ouverture metadata: %w", err)
	}
	defer db.Close()

	// Schéma aligné sur la prod metadata.duckdb (contrat lu par EnrichFromMetadata,
	// LoadCareerRankImageURLs, populate-career-rank-images). INSERT à colonnes
	// explicites : robuste à l'ordre + compatible avec une table existante.
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS career_ranks (
		rank_id INTEGER PRIMARY KEY,
		title_en VARCHAR, subtitle_en VARCHAR,
		tier VARCHAR, tier_type VARCHAR, grade INTEGER,
		xp_required INTEGER, icon_path VARCHAR,
		large_icon_path VARCHAR, adornment_icon_path VARCHAR
	)`); err != nil {
		return SeedResult{Component: componentCareerRanks}, err
	}

	inserted, skipped := 0, 0
	for _, r := range ranks {
		res, err := db.ExecContext(ctx, `INSERT OR IGNORE INTO career_ranks
			(rank_id, title_en, subtitle_en, tier, tier_type, grade,
			 xp_required, icon_path, large_icon_path, adornment_icon_path)
			VALUES (?,?,?,?,?,?,?,?,?,?)`,
			r.RankID, r.TitleEN, r.Subtitle, r.Tier, r.TierType, r.Grade,
			r.XPRequired, r.IconPath, r.LargeIconPath, r.AdornmentPath)
		if err != nil {
			return SeedResult{Component: componentCareerRanks}, fmt.Errorf("insert rank %d: %w", r.RankID, err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			skipped++
		} else {
			inserted++
		}
	}
	return SeedResult{
		Component: componentCareerRanks,
		Inserted:  inserted,
		Skipped:   skipped,
		Message:   fmt.Sprintf("%d insérés, %d déjà présents", inserted, skipped),
	}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// career_rank_translations
// ─────────────────────────────────────────────────────────────────────────────

// SeedRankTranslations peuple career_rank_translations (272 rangs × FR/EN) depuis
// le provider title-owned (migration.CareerRankTranslationRows, posé au boot via
// migration.SetCareerRankTranslationsProvider — MT-07). Données de référence
// statiques : sans elles, le rang carrière s'affiche en EN faute de fetch GameCMS
// (career_rank_translations vide → fallback RankName EN côté home). Provider non
// posé → 0 ligne (dégradation gracieuse ; le câblage du binaire doit le poser).
// Idempotent : INSERT OR REPLACE sur la PK (rank_id, lang). La table est créée par
// la migration add_career_rank_translations (appliquée avant le seed via runSeed).
func SeedRankTranslations(ctx context.Context, opts SeedOptions) (SeedResult, error) {
	db, err := sql.Open("duckdb", opts.MetaDBPath)
	if err != nil {
		return SeedResult{Component: componentRankTranslations}, fmt.Errorf("ouverture metadata: %w", err)
	}
	defer db.Close()

	rows := migration.CareerRankTranslationRows()
	inserted := 0
	for _, t := range rows {
		// ART-safe : SELECT-then-UPDATE-or-INSERT (pas d'INSERT OR REPLACE, =
		// ON CONFLICT DO UPDATE toutes colonnes → réécrit via l'index ART de la PK
		// et FATAL-invalide metadata.duckdb). L'index lookup est sur la PK (rank_id,
		// lang), non mutée par l'UPDATE.
		var dummy int
		err := db.QueryRowContext(ctx,
			`SELECT 1 FROM career_rank_translations WHERE rank_id = ? AND lang = ?`,
			t.RankID, t.Lang).Scan(&dummy)
		switch {
		case err == nil:
			_, err = db.ExecContext(ctx, `UPDATE career_rank_translations
				SET title = ?, subtitle = '', tier = ?, fetched_at = CURRENT_TIMESTAMP
				WHERE rank_id = ? AND lang = ?`, t.Title, t.Tier, t.RankID, t.Lang)
		case errors.Is(err, sql.ErrNoRows):
			_, err = db.ExecContext(ctx, `INSERT INTO career_rank_translations
				(rank_id, lang, title, subtitle, tier, fetched_at)
				VALUES (?,?,?,'',?,CURRENT_TIMESTAMP)`, t.RankID, t.Lang, t.Title, t.Tier)
		}
		if err != nil {
			return SeedResult{Component: componentRankTranslations},
				fmt.Errorf("upsert rank %d lang %s: %w", t.RankID, t.Lang, err)
		}
		inserted++
	}
	return SeedResult{
		Component: componentRankTranslations,
		Inserted:  inserted,
		Message:   fmt.Sprintf("%d lignes upsertées (272 rangs × FR/EN)", inserted),
	}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// citation_mappings
// ─────────────────────────────────────────────────────────────────────────────

// CitationMapping représente une règle citation complète (parité avec
// scripts/populate_citation_mappings.py main).
//
// Chaque champ optionnel utilise la string vide pour signifier NULL ; le
// chargement SQL convertit en NULL avant INSERT.
type CitationMapping struct {
	Norm              string
	Display           string
	MappingType       string // medal|composite|stat|pve_stat|weapon_stat|award|custom
	MedalID           int64  // 0 = NULL (medal_id simple)
	MedalIDs          string // CSV, "" = NULL (medal_ids multi)
	StatName          string // colonne mp ou "weapon_kills:<NameEN>", "" = NULL
	AwardName         string // award_name (personal_score_awards), "" = NULL
	AwardCategory     string // "" = NULL
	CustomFunction    string // nom de fonction custom, "" = NULL
	CompositeChildren string // JSON array de norm enfants, "" = NULL
	Enabled           bool
	ImagePath         string // "" = NULL
	Category          string
	Description       string
	TierTargets       string // CSV de seuils, "" = NULL
	Subcategory       string // "" = NULL
}

// SeedCitationMappings insère/met à jour les règles citation_mappings.
// UPSERT sur citation_name_norm (PK depuis migration add_citation_mappings_pk),
// donc relancer le seed met à jour les colonnes ajoutées par la migration v2.
//
// Compte distinctement Inserted (nouvelles citations) et Skipped (citations
// déjà présentes — leurs colonnes sont quand même rafraîchies via DO UPDATE).
func SeedCitationMappings(ctx context.Context, opts SeedOptions) (SeedResult, error) {
	db, err := sql.Open("duckdb", opts.MetaDBPath)
	if err != nil {
		return SeedResult{Component: componentCitationMappings}, fmt.Errorf("ouverture metadata: %w", err)
	}
	defer db.Close()

	// Schéma v2 complet : compatible migrations add_citation_mappings +
	// add_citation_mappings_pk + add_citation_mappings_v2_fields.
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS citation_mappings (
			citation_name_norm    VARCHAR PRIMARY KEY,
			citation_name_display VARCHAR NOT NULL,
			mapping_type          VARCHAR NOT NULL DEFAULT 'medal',
			medal_id              UBIGINT,
			medal_ids             VARCHAR,
			stat_name             VARCHAR,
			award_name            VARCHAR,
			award_category        VARCHAR,
			custom_function       VARCHAR,
			composite_children    VARCHAR,
			enabled               BOOLEAN NOT NULL DEFAULT TRUE,
			image_path            VARCHAR,
			category              VARCHAR,
			description           VARCHAR,
			tier_targets          VARCHAR,
			subcategory           VARCHAR
		);
		-- PAS d'index sur medal_id/mapping_type : mutés par SeedCitationMappings
		-- (SELECT-then-write) → surface ART. Drop DBs existantes : drop_metadata_art_surface_indexes_v4.
		CREATE INDEX IF NOT EXISTS idx_citation_mappings_norm ON citation_mappings(citation_name_norm);
			-- Nom anglais des citations (Infinite = copies de commendations H5 ; seul le
			-- calcul diffère). Locale-aware au read via ctxkeys.Locale. NULL → fallback FR.
			ALTER TABLE citation_mappings ADD COLUMN IF NOT EXISTS citation_name_display_en VARCHAR;
			-- Description anglaise des citations (source : commendations H5 officielles +
			-- trad fidèle Infinite). Locale-aware au read ; NULL → tooltip = nom seul (GH4).
			ALTER TABLE citation_mappings ADD COLUMN IF NOT EXISTS description_en VARCHAR;
	`); err != nil {
		return SeedResult{Component: componentCitationMappings}, fmt.Errorf("create schema: %w", err)
	}

	existing := make(map[string]struct{})
	rows, err := db.QueryContext(ctx, `SELECT citation_name_norm FROM citation_mappings`)
	if err != nil {
		return SeedResult{Component: componentCitationMappings}, fmt.Errorf("scan existing: %w", err)
	}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			rows.Close()
			return SeedResult{Component: componentCitationMappings}, err
		}
		existing[n] = struct{}{}
	}
	rows.Close()

	// ART-safe : SELECT-then-INSERT-or-UPDATE (pas d'ON CONFLICT, qui réécrit via les
	// index ART). La map `existing` (déjà construite ci-dessus) donne la décision
	// insert/update sans SELECT par ligne. L'UPDATE mute medal_id/mapping_type, indexés
	// (idx_citation_mappings_medal/type) → ces index sont droppés par
	// drop_metadata_art_surface_indexes_v4. metadata.duckdb = blast-radius MAX même au seed.
	const insertQ = `
		INSERT INTO citation_mappings (
			citation_name_norm, citation_name_display, citation_name_display_en, mapping_type,
			medal_id, medal_ids, stat_name, award_name, award_category,
			custom_function, composite_children, enabled,
			image_path, category, description, description_en, tier_targets, subcategory
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	const updateQ = `
		UPDATE citation_mappings SET
			citation_name_display = ?, citation_name_display_en = ?, mapping_type = ?, medal_id = ?, medal_ids = ?,
			stat_name = ?, award_name = ?, award_category = ?, custom_function = ?,
			composite_children = ?, enabled = ?, image_path = ?, category = ?,
			description = ?, description_en = ?, tier_targets = ?, subcategory = ?
		WHERE citation_name_norm = ?`

	mappings := defaultCitationMappings()
	inserted, skipped := 0, 0
	for _, m := range mappings {
		var medalArg interface{}
		if m.MedalID > 0 {
			medalArg = uint64(m.MedalID) //nolint:gosec
		}
		_, present := existing[m.Norm]
		var execErr error
		if present {
			_, execErr = db.ExecContext(ctx, updateQ,
				m.Display, citationDisplayENOr(m.Norm, m.Display), m.MappingType, medalArg, nullStr(m.MedalIDs),
				nullStr(m.StatName), nullStr(m.AwardName), nullStr(m.AwardCategory),
				nullStr(m.CustomFunction), nullStr(m.CompositeChildren), m.Enabled,
				nullStr(m.ImagePath), m.Category, m.Description, citationDescriptionENOr(m.Norm),
				nullStr(m.TierTargets), nullStr(m.Subcategory), m.Norm)
		} else {
			_, execErr = db.ExecContext(ctx, insertQ,
				m.Norm, m.Display, citationDisplayENOr(m.Norm, m.Display), m.MappingType,
				medalArg, nullStr(m.MedalIDs), nullStr(m.StatName),
				nullStr(m.AwardName), nullStr(m.AwardCategory),
				nullStr(m.CustomFunction), nullStr(m.CompositeChildren), m.Enabled,
				nullStr(m.ImagePath), m.Category, m.Description, citationDescriptionENOr(m.Norm),
				nullStr(m.TierTargets), nullStr(m.Subcategory))
		}
		if execErr != nil {
			return SeedResult{Component: componentCitationMappings}, fmt.Errorf("upsert %s: %w", m.Norm, execErr)
		}
		if present {
			skipped++
		} else {
			inserted++
		}
	}
	return SeedResult{
		Component: componentCitationMappings,
		Inserted:  inserted,
		Skipped:   skipped,
		Message:   fmt.Sprintf("%d insérées, %d mises à jour", inserted, skipped),
	}, nil
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// citationDisplayENOr retourne le nom EN d'une citation (map citationDisplayEN, clé =
// norm) ou, à défaut, le libellé FR fourni — on ne stocke jamais de NULL/vide pour
// citation_name_display_en quand un FR existe, afin que la cascade locale-aware
// retombe proprement sur le FR plutôt que sur du vide.
func citationDisplayENOr(norm, fallbackFR string) interface{} {
	if en, ok := citationDisplayEN[norm]; ok && en != "" {
		return en
	}
	if fallbackFR == "" {
		return nil
	}
	return fallbackFR
}

// citationDescriptionENOr retourne la description EN d'une citation (map
// citationDescriptionEN, clé = norm) ou nil si absente. Contrairement à
// citationDisplayENOr, on NE retombe PAS sur le FR : description_en reste NULL et le read
// locale-aware sert alors le nom seul sous UI EN (principe GH-5b « EN n'injecte jamais de
// FR »). La map couvre les 88 citations → nil en pratique = citation ajoutée sans EN.
func citationDescriptionENOr(norm string) interface{} {
	if en, ok := citationDescriptionEN[norm]; ok && en != "" {
		return en
	}
	return nil
}

// defaultCitationMappings — 88 règles citations portées de
// scripts/populate_citation_mappings.py (branche main, source de vérité).
//
// Catégories : Mode de jeu, Multijoueur, Spartan Companies, Véhicule, Arme,
// Ennemi (PVE) + composites.
//
// Toute modification ici doit être reportée dans le script Python pour garder
// les deux branches en parité.
//
//nolint:funlen,maintidx // Données de seed — préfère lisibilité linéaire.
func SeedMedalDefinitions(ctx context.Context, opts SeedOptions) (SeedResult, error) {
	db, err := sql.Open("duckdb", opts.MetaDBPath)
	if err != nil {
		return SeedResult{Component: componentMedalDefinitions}, fmt.Errorf("ouverture metadata: %w", err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'medal_definitions'`).Scan(&count); err != nil || count == 0 {
		return SeedResult{
			Component: componentMedalDefinitions,
			Message:   "table absente — lancer d'abord les migrations",
		}, nil
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM medal_definitions").Scan(&count); err != nil {
		return SeedResult{Component: componentMedalDefinitions}, err
	}
	return SeedResult{
		Component: componentMedalDefinitions,
		Skipped:   count,
		Message:   fmt.Sprintf("%d médailles présentes", count),
	}, nil
}
