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

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/migration"

	_ "github.com/duckdb/duckdb-go/v2"
)

// SeedOptions configure une opération de seed.
type SeedOptions struct {
	MetaDBPath string
	DataDir    string // data/ root pour les JSON sources
}

// SeedResult résume le résultat d'un seed.
//
// Pour citation_mappings : Inserted = citations nouvelles, Skipped = citations déjà présentes,
// Updated = citations déjà présentes dont une colonne ou un texte différait et a été réécrit.
type SeedResult struct {
	Component string
	Inserted  int
	Updated   int
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
	mappingTypeAward         = "award"
	mappingTypeStat          = "stat"
	mappingTypePVEStat       = "pve_stat"
	mappingTypeObjectiveStat = "objective_stat"
	mappingTypeWeaponStat    = "weapon_stat"
	mappingTypeMedal         = "medal"
	mappingTypeComposite     = "composite"
	mappingTypeCustom        = "custom"
)

// Norms des 4 citations d'objectif basculées en source objective_stat (v7.2) —
// constantes pour éviter la duplication littérale du norm (goconst) entre la
// définition seedée (defaultCitationMappings) et son garde-rail de test
// (TestObjectiveStatCitations_MappedToColumns).
const (
	citationNormCharge            = "charge"
	citationNormGotYou            = "got_you"
	citationNormStakeholder       = "stakeholder"
	citationNormFlagCarrierHunter = "flag_carrier_hunter"
)

// Norms des 10 citations d'objectif ajoutées en v7.2.1 (V721-03). Même raison que
// le bloc ci-dessus : chaque norm est cité 4 fois (ligne seedée + clé du nom EN +
// clé de la description EN + garde-rail de test), ce que goconst refuse en littéral.
// 9 autres colonnes exploitables ont été écartées par l'utilisateur — ne pas les
// re-proposer sans arbitrage.
const (
	citationNormFlagCaptures         = "flag_captures"
	citationNormFlagSecures          = "flag_secures"
	citationNormFlagSteals           = "flag_steals"
	citationNormReturnerTakedown     = "returner_takedown"
	citationNormUnstoppableCarrier   = "unstoppable_carrier"
	citationNormAggressiveReturn     = "aggressive_return"
	citationNormZoneDefense          = "zone_defense"
	citationNormUntouchableCarrier   = "untouchable_carrier"
	citationNormSkullCarrierTakedown = "skull_carrier_takedown"
	citationNormSkullGrabs           = "skull_grabs"
)

// Catégories citation_mappings.category — CLÉS STABLES (snake_case), jamais des
// libellés humains. Le seed écrivait des libellés FR (« Mode de jeu ») jusqu'au
// 2026-08-04 : la valeur stockée était alors de la donnée localisée, normalisée à
// chaque lecture. La colonne porte désormais directement la clé canonique
// (migration de données `normalize_citation_mappings_category_keys`), la
// traduction FR/EN vivant dans le manifeste web citations.toml.
//
// Une seule source pour ces clés : internal/games/canonical — ajouter une
// catégorie impose donc de l'ajouter là-bas (sinon elle est servie comme "other").
const (
	citationCatModeJeu          = canonical.CommendationCategoryGameMode
	citationCatVehicule         = canonical.CommendationCategoryVehicle
	citationCatArme             = canonical.CommendationCategoryWeapon
	citationCatMultijoueur      = canonical.CommendationCategoryMultiplayer
	citationCatSpartanCompanies = canonical.CommendationCategorySpartanCompanies
	citationCatEnnemi           = canonical.CommendationCategoryEnemy
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

// Tier targets des citations d'objectif ajoutées en v7.2.1 (source objective_stat,
// colonnes match_objective_stats_latest). CALIBRÉS sur les données réelles au 2026-07-25
// (joueur de référence : 304 matchs à objectifs), PAS repris des paliers génériques
// ci-dessus — un réalignement rendrait quatre de ces citations inatteignables.
//
// Deux régimes assumés :
//   - citations à volume (captures/sécurisations/vols de drapeau, défense de zone) :
//     palier maître ≈ 1,2–1,35 × le total actuel du meilleur joueur ;
//   - exploits intrinsèquement rares (frags en portant le drapeau ou le crâne : 3 et 1
//     occurrences au total) et mode peu joué (26 matchs Oddball en base) : paliers bas
//     (1,2,3,5,10 / 2,5,10,20,40) — volontaire, ce n'est pas une erreur de calibration.
//
// Toute retouche de ces valeurs impose un `backfill --citations-recompute-all` (la
// progression est recalculée depuis l'historique, cf. docs/COMMENDATIONS.md).
const (
	tierTargets1_2_3_5_10         = "1,2,3,5,10"
	tierTargets2_5_10_20_40       = "2,5,10,20,40"
	tierTargets5_10_20_30_50      = "5,10,20,30,50"
	tierTargets5_10_20_35_60      = "5,10,20,35,60"
	tierTargets10_25_50_75_125    = "10,25,50,75,125"
	tierTargets25_50_100_175_300  = "25,50,100,175,300"
	tierTargets25_50_100_200_350  = "25,50,100,200,350"
	tierTargets50_100_150_250_500 = "50,100,150,250,500"
	tierTargets50_100_200_350_600 = "50,100,200,350,600"
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
	Title         string `json:"title_en"` // clé du fichier source : le titre ANGLAIS du rang (format d entrée figé)
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
			r.RankID, r.Title, r.Subtitle, r.Tier, r.TierType, r.Grade,
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
	TierTargets       string // CSV de seuils, "" = NULL
	Subcategory       string // "" = NULL
}

// SeedCitationMappings insère ou met à jour les règles citation_mappings et leurs textes par
// langue dans la base MetaDBPath (commande `levelup seed citation-mappings`). Les noms et
// descriptions de chaque langue viennent de citation_texts.toml (validé : toutes les langues
// `required`) : le seed n'a ni colonne de texte ni texte en Go.
func SeedCitationMappings(ctx context.Context, opts SeedOptions) (SeedResult, error) {
	// Validé AVANT l'ouverture : un fichier incomplet ne crée pas la base.
	if err := validateCitationTexts(); err != nil {
		return SeedResult{Component: componentCitationMappings}, err
	}
	db, err := sql.Open("duckdb", opts.MetaDBPath)
	if err != nil {
		return SeedResult{Component: componentCitationMappings}, fmt.Errorf("ouverture metadata: %w", err)
	}
	defer db.Close()
	return seedCitationMappingsOn(ctx, db)
}

// SeedMedalDefinitions ne peuple PLUS rien : medal_definitions est désormais
// alimentée par les migrations (portage historique de populate_medal_metadata.py).
// Cette fonction se limite à vérifier que la table existe et à retourner son
// décompte de lignes (Skipped) — un compte à 0 avec la table présente signale
// des migrations non jouées, pas une absence de code de seed.
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
