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
	"fmt"
	"os"

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
		CREATE INDEX IF NOT EXISTS idx_citation_mappings_norm ON citation_mappings(citation_name_norm);
		CREATE INDEX IF NOT EXISTS idx_citation_mappings_medal ON citation_mappings(medal_id);
		CREATE INDEX IF NOT EXISTS idx_citation_mappings_type ON citation_mappings(mapping_type);
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

	const upsert = `
		INSERT INTO citation_mappings (
			citation_name_norm, citation_name_display, mapping_type,
			medal_id, medal_ids, stat_name, award_name, award_category,
			custom_function, composite_children, enabled,
			image_path, category, description, tier_targets, subcategory
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (citation_name_norm) DO UPDATE SET
			citation_name_display = EXCLUDED.citation_name_display,
			mapping_type          = EXCLUDED.mapping_type,
			medal_id              = EXCLUDED.medal_id,
			medal_ids             = EXCLUDED.medal_ids,
			stat_name             = EXCLUDED.stat_name,
			award_name            = EXCLUDED.award_name,
			award_category        = EXCLUDED.award_category,
			custom_function       = EXCLUDED.custom_function,
			composite_children    = EXCLUDED.composite_children,
			enabled               = EXCLUDED.enabled,
			image_path            = EXCLUDED.image_path,
			category              = EXCLUDED.category,
			description           = EXCLUDED.description,
			tier_targets          = EXCLUDED.tier_targets,
			subcategory           = EXCLUDED.subcategory`

	mappings := defaultCitationMappings()
	inserted, skipped := 0, 0
	for _, m := range mappings {
		var medalArg interface{}
		if m.MedalID > 0 {
			medalArg = uint64(m.MedalID) //nolint:gosec
		}
		if _, err := db.ExecContext(ctx, upsert,
			m.Norm, m.Display, m.MappingType,
			medalArg, nullStr(m.MedalIDs), nullStr(m.StatName),
			nullStr(m.AwardName), nullStr(m.AwardCategory),
			nullStr(m.CustomFunction), nullStr(m.CompositeChildren), m.Enabled,
			nullStr(m.ImagePath), m.Category, m.Description,
			nullStr(m.TierTargets), nullStr(m.Subcategory),
		); err != nil {
			return SeedResult{Component: componentCitationMappings}, fmt.Errorf("upsert %s: %w", m.Norm, err)
		}
		if _, present := existing[m.Norm]; present {
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
func defaultCitationMappings() []CitationMapping {
	// Phase 6.5 (2026-04-28, commit a1d25325) a renommé les sous-dossiers vers
	// les slugs canoniques longs. Tout chemin émis ici doit utiliser ces slugs,
	// sinon le seed UPSERT écrase image_path avec un chemin 404.
	const wpH5 = "static/commendations/halo_5_guardians/"
	const wpHI = "static/commendations/halo_infinite/"
	return []CitationMapping{
		// ── PVP — Mode de jeu (11) ────────────────────────────────────
		{Norm: "charge", Display: "À la charge", MappingType: mappingTypeAward,
			AwardName: "zone_captured", AwardCategory: awardCategoryObjective, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_À_la_charge.png",
			Category:    citationCatModeJeu,
			Description: "Prenez le contrôle d'une base dans n'importe quelle partie matchmaking Bases.",
			TierTargets: tierTargets10_20_30_50_100},
		{Norm: "forced_annexation", Display: "Annexion forcée", MappingType: mappingTypeCustom,
			CustomFunction: "compute_annexion_forcee", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Annexion_forcée.png",
			Category:    citationCatModeJeu,
			Description: "Prenez le contrôle de 3 bases sans mourir dans n'importe quelle partie matchmaking Bases.",
			TierTargets: tierTargets3_6_9_15_30},
		{Norm: "assistant", Display: "Assistant", MappingType: mappingTypeStat,
			StatName: "assists", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Assistant.png",
			Category:    citationCatModeJeu,
			Description: "Remportez n'importe quelle médaille d'assistance en Zone de combat.",
			TierTargets: "25,50,75,125,250"},
		{Norm: "bulldozer", Display: "Bulldozer", MappingType: mappingTypeCustom,
			CustomFunction: "compute_bulldozer", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Bulldozer.png",
			Category:    citationCatModeJeu,
			Description: "Terminez n'importe quelle partie matchmaking Assassin avec un FDA supérieur à 8.",
			TierTargets: tierTargets3_6_9_15_30},
		{Norm: "flag_defender", Display: "Défenseur du drapeau", MappingType: mappingTypeAward,
			AwardName: "carrier_killed", AwardCategory: awardCategoryObjective, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Défenseur_du_drapeau.png",
			Category:    citationCatModeJeu,
			Description: "Protégez le drapeau de votre équipe dans n'importe quelle partie matchmaking Capture du drapeau.",
			TierTargets: tierTargets10_20_30_50_100},
		{Norm: "got_you", Display: "Je te tiens !", MappingType: mappingTypeAward,
			AwardName: "flag_returned", AwardCategory: awardCategoryObjective, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Je_te_tiens_!.png",
			Category:    citationCatModeJeu,
			Description: "Rapportez le drapeau de votre équipe dans n'importe quelle partie matchmaking Capture du drapeau.",
			TierTargets: tierTargets10_20_30_50_100},
		{Norm: "stakeholder", Display: "Partie prenante", MappingType: mappingTypeAward,
			AwardName: "zone_secured", AwardCategory: awardCategoryObjective, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Partie_prenante.png",
			Category:    citationCatModeJeu,
			Description: "Défendez une base appartenant à votre équipe dans n'importe quelle partie matchmaking Bases.",
			TierTargets: tierTargets10_20_30_50_100},
		{Norm: "flag_carrier_hunter", Display: "Sus au porteur du drapeau", MappingType: mappingTypeAward,
			AwardName: "carrier_killed", AwardCategory: awardCategoryObjective, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Sus_au_porteur_du_drapeau.png",
			Category:    citationCatModeJeu,
			Description: "Tuez un porte-drapeau ennemi dans n'importe quelle partie matchmaking Capture du drapeau.",
			TierTargets: tierTargets10_20_30_50_100},
		{Norm: "flag_victory", Display: "Victoire au drapeau", MappingType: mappingTypeCustom,
			CustomFunction: "compute_wins_ctf", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Victoire_au_drapeau.png",
			Category:    citationCatModeJeu,
			Description: "Remportez n'importe quelle partie matchmaking Capture du drapeau.",
			TierTargets: tierTargets5_10_15_25_50},
		{Norm: "slayer_victory", Display: "Victoire en assassin", MappingType: mappingTypeCustom,
			CustomFunction: "compute_wins_slayer", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Victoire_en_Assassin.png",
			Category:    citationCatModeJeu,
			Description: "Remportez une partie d'Assassin en équipe.",
			TierTargets: tierTargets5_10_15_25_50},
		{Norm: "strongholds_victory", Display: "Victoire en bases", MappingType: mappingTypeCustom,
			CustomFunction: "compute_wins_strongholds", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Victoire_en_Bases.png",
			Category:    citationCatModeJeu,
			Description: "Remportez n'importe quelle partie matchmaking Bases.",
			TierTargets: tierTargets5_10_15_25_50},

		// ── PVP — Véhicule (2) + Grenade (2) ──────────────────────────
		{Norm: "splatter", Display: "Écrasement", MappingType: mappingTypeMedal, MedalID: 221693153, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Écrasement.png",
			Category:    citationCatVehicule,
			Description: "Écrasez un Spartan adverse avec un véhicule.",
			TierTargets: tierTargets10_20_30_50_100, Subcategory: citationSubGeneral},
		{Norm: "driver", Display: "Pilote", MappingType: mappingTypeMedal, MedalID: 3169118333, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Pilote.png",
			Category:    citationCatVehicule,
			Description: "Décrochez des médailles de pilote.",
			TierTargets: tierTargets10_20_30_50_100, Subcategory: citationSubGeneral},
		{Norm: "frag_grenade", Display: "Grenade à fragmentation", MappingType: mappingTypeMedal, MedalID: 2648272972, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Grenade_à_fragmentation.png",
			Category:    citationCatArme,
			Description: "Tuez un Spartan adverse à l'aide d'une grenade à fragmentation.",
			TierTargets: tierTargets5_10_15_25_50, Subcategory: "Grenade"},
		{Norm: "plasma_grenade", Display: "Grenade à plasma", MappingType: mappingTypeMedal, MedalID: 3655682764, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Grenade_à_plasma.png",
			Category:    citationCatArme,
			Description: "Tuez un Spartan adverse à l'aide d'une grenade à plasma.",
			TierTargets: "2,4,6,10,20", Subcategory: "Grenade"},

		// ── PVP — Multijoueur (10) ────────────────────────────────────
		{Norm: "assassin", Display: "Assassin", MappingType: mappingTypeMedal, MedalID: 548533137, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Assassin.png",
			Category:    citationCatMultijoueur,
			Description: "Assassinez des Spartans adverses.",
			TierTargets: tierTargets5_10_15_25_50},
		{Norm: "spartan_carnage", Display: "Carnage de Spartans", MappingType: mappingTypeMedal,
			MedalIDs:    "2780740615,4261842076,418532952,1486797009,710323196,1720896992,2567026752,2875941471",
			Enabled:     true,
			ImagePath:   wpH5 + "H5G_citation_Carnage_de_Spartans.png",
			Category:    citationCatMultijoueur,
			Description: "Tuez plusieurs Spartans adverses sans mourir.",
			TierTargets: tierTargets3_6_9_15_30},
		{Norm: "close_combat", Display: "Combat rapproché", MappingType: mappingTypeStat,
			StatName: "melee_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Combat_rapproché.png",
			Category:    citationCatMultijoueur,
			Description: "Remportez n'importe quelle médaille de combat rapproché.",
			TierTargets: tierTargets10_20_30_50_100},
		{Norm: "opportunist", Display: "Combattant opportuniste", MappingType: mappingTypeMedal,
			MedalIDs:    "622331684,2063152177,4261842076,2137071619,1486797009,1430343434,2242633421",
			Enabled:     true,
			ImagePath:   wpH5 + "H5G_citation_Combattant_opportuniste.png",
			Category:    citationCatMultijoueur,
			Description: "Remportez n'importe quelle médaille d'aptitude au combat.",
			TierTargets: tierTargets10_20_30_50_100},
		{Norm: "multikill", Display: "Multifrag", MappingType: mappingTypeMedal, MedalID: 622331684, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Multifrag.png",
			Category:    citationCatMultijoueur,
			Description: "Tuez rapidement plusieurs Spartans adverses.",
			TierTargets: tierTargets3_6_9_15_30},
		{Norm: "melee_fighter", Display: "Pugilat", MappingType: mappingTypeStat,
			StatName: "melee_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Pugilat.png",
			Category:    citationCatMultijoueur,
			Description: "Tuez un Spartan ennemi d'une attaque rapprochée.",
			TierTargets: tierTargets5_10_15_25_50},
		{Norm: "headshot", Display: "Tir à la tête", MappingType: mappingTypeStat,
			StatName: "headshot_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Tir_à_la_tête.png",
			Category:    citationCatMultijoueur,
			Description: "Tuez un Spartan ennemi d'un tir à la tête.",
			TierTargets: tierTargets10_20_30_50_100},
		{Norm: "spartan_killer", Display: "Tueur de Spartans", MappingType: mappingTypeStat,
			StatName: "kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Tueur_de_Spartans.png",
			Category:    citationCatMultijoueur,
			Description: "Éliminez les Spartans ennemis.",
			TierTargets: "20,40,60,100,200"},
		{Norm: "eagle_eye", Display: "Œil de lynx", MappingType: mappingTypeMedal, MedalID: 1512363953, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Œil_de_lynx.png",
			Category:    citationCatMultijoueur,
			Description: "Tuez un Spartan adverse en pleine santé à l'aide d'une arme de précision sans manquer un seul coup.",
			TierTargets: tierTargets10_20_30_50_100},
		{Norm: "avenger", Display: "Vengeur", MappingType: mappingTypeMedal, MedalID: 9000000001, Enabled: true,
			ImagePath:   wpH5 + "Assassin_commendation.png",
			Category:    citationCatMultijoueur,
			Description: "Tuez l'ennemi responsable de votre mort précédente.",
			TierTargets: "5,15,30,55,105"},

		// ── PVP — Spartan Companies (15) ──────────────────────────────
		{Norm: "flag_em_down", Display: "Sors les drapeaux", MappingType: mappingTypeCustom,
			CustomFunction: "compute_flag_em_down", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Flag_'em_down.png",
			Category:    citationCatSpartanCompanies,
			Description: "Obtenir une des médailles suivantes : Interception, Bataille de drapeaux, Frag du porteur, Retour du drapeau, Défense du drapeau",
			TierTargets: "1000,2000,3000,4800,9700"},
		{Norm: "grand_theft", Display: "Vol à la tire", MappingType: mappingTypeCustom,
			CustomFunction: "compute_hijack", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Grand_Theft.png",
			Category:    citationCatSpartanCompanies,
			Description: "Aborder un véhicule ou un véhicule aérien",
			TierTargets: "200,400,600,960,1940"},
		{Norm: "helping_hand", Display: "Coup de main", MappingType: mappingTypeStat,
			StatName: "assists", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Helping_Hand.png",
			Category:    citationCatSpartanCompanies,
			Description: "Obtenir n'importe quelle assistance",
			TierTargets: "20000,40000,60000,96000,194400"},
		{Norm: "im_just_perfect", Display: "Zéro défaut", MappingType: mappingTypeMedal, MedalID: 1512363953, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_I'm_just_perfect.png",
			Category:    citationCatSpartanCompanies,
			Description: "Tuer un joueur avec une arme de précision sans manquer un tir",
			TierTargets: "2000,4000,6000,9600,19400"},
		{Norm: "lawnmower", Display: "Tondeuse", MappingType: mappingTypeMedal, MedalID: 221693153, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Lawnmower.png",
			Category:    citationCatSpartanCompanies,
			Description: "Tuer un adversaire en l'écrasant avec un véhicule",
			TierTargets: "500,1000,1500,2400,4900"},
		{Norm: "look_ma_no_pin", Display: "Regarde maman, sans goupille", MappingType: mappingTypeStat,
			StatName: "grenade_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Look_ma_no_pin.png",
			Category:    citationCatSpartanCompanies,
			Description: "Tuer un Spartan ennemi avec n'importe quelle grenade",
			TierTargets: "4000,8000,12000,19200,38900"},
		{Norm: "lucky", Display: "Lucky", MappingType: mappingTypeMedal,
			MedalIDs: "3905838030,3091261182", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Lucky.png",
			Category:    citationCatSpartanCompanies,
			Description: "Obtenir les médailles La chance, Chargeur vide, Sayonara ou Boulet de canon",
			TierTargets: "400,800,1200,1920,3880"},
		{Norm: "no_hard_feelings", Display: "Sans rancune", MappingType: mappingTypeStat,
			StatName: "kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_No_Hard_Feelings.png",
			Category:    citationCatSpartanCompanies,
			Description: "Tuer des Spartans ennemis",
			TierTargets: "50000,100000,150000,240000,486000"},
		{Norm: "positive_contribution", Display: "Positive contribution", MappingType: mappingTypeCustom,
			CustomFunction: "compute_bulldozer", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Positive_contribution.png",
			Category:    citationCatSpartanCompanies,
			Description: "Finir avec FDA supérieur à 8 dans n'importe quelle partie Assassin en matchmaking",
			TierTargets: "300,600,900,1440,2960"},
		{Norm: "power_play", Display: "Coup de force", MappingType: mappingTypeStat,
			StatName: "power_weapon_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Power_play.png",
			Category:    citationCatSpartanCompanies,
			Description: "Tuer un Spartan ennemi avec une arme puissante",
			TierTargets: "10000,20000,30000,48000,97200"},
		{Norm: "road_trip", Display: "Virée sur la route", MappingType: mappingTypeMedal, MedalID: 3169118333, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Road_Trip.png",
			Category:    citationCatSpartanCompanies,
			Description: "Tuer un Spartan ennemi avec un véhicule terrestre",
			TierTargets: "3000,6000,9000,14400,29200"},
		{Norm: "sting_like_a_bee", Display: "Pique comme une abeille", MappingType: mappingTypeStat,
			StatName: "melee_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Sting_like_a_bee.png",
			Category:    citationCatSpartanCompanies,
			Description: "Tuer un Spartan ennemi en combat rapproché",
			TierTargets: "5000,10000,15000,24000,48600"},
		{Norm: "the_reaper", Display: "Le faucheur", MappingType: mappingTypeMedal, MedalID: 2625820422, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_The_Reaper.png",
			Category:    citationCatSpartanCompanies,
			Description: "Tuer un adversaire d'outre-tombe",
			TierTargets: "500,1000,1500,2400,4850"},
		{Norm: "too_fast_for_you", Display: "Trop rapide pour toi", MappingType: mappingTypeMedal, MedalID: 2123530881, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Too_fast_for_you.png",
			Category:    citationCatSpartanCompanies,
			Description: "Tuer un adversaire qui vous a tiré dessus en premier",
			TierTargets: "2000,4000,6000,9600,19400"},
		{Norm: "vandalism", Display: "Vandalisme", MappingType: mappingTypeCustom,
			CustomFunction: "compute_vandalism", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Vandalism.png",
			Category:    citationCatSpartanCompanies,
			Description: "Détruire un véhicule ennemi",
			TierTargets: "1200,2400,3600,5760,11640"},

		// ── PVP — Véhicules destructeurs (7) ─────────────────────────
		{Norm: "wraith_destroyer", Display: "Destructeur d'apparitions", MappingType: mappingTypeCustom,
			CustomFunction: "compute_wraith_destroyer", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Destructeur_d'apparitions.png",
			Category:    citationCatVehicule,
			Description: "Détruisez les apparitions occupées par l'adversaire.",
			TierTargets: tierTargets3_6_9_15_30, Subcategory: citationSubCovenant},
		{Norm: "banshee_destroyer", Display: "Destructeur de banshees", MappingType: mappingTypeAward,
			AwardName: "destroyed_banshee", AwardCategory: awardCategoryVehicle, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Destructeur_de_banshees.png",
			Category:    citationCatVehicule,
			Description: "Détruisez les banshees occupés par l'adversaire.",
			TierTargets: tierTargets3_6_9_15_30, Subcategory: citationSubCovenant},
		{Norm: "ghost_destroyer", Display: "Destructeur de ghosts", MappingType: mappingTypeAward,
			AwardName: "destroyed_ghost", AwardCategory: awardCategoryVehicle, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Destructeur_de_ghosts.png",
			Category:    citationCatVehicule,
			Description: "Détruisez les ghosts occupés par l'adversaire.",
			TierTargets: tierTargets5_10_15_25_50, Subcategory: citationSubCovenant},
		{Norm: "mongoose_destroyer", Display: "Destructeur de mongooses", MappingType: mappingTypeCustom,
			CustomFunction: "compute_mongoose_destroyer", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Destructeur_de_mongooses.png",
			Category:    citationCatVehicule,
			Description: "Détruisez les mongooses occupées par l'adversaire.",
			TierTargets: tierTargets5_10_15_25_50, Subcategory: citationSubUNSC},
		{Norm: "scorpion_destroyer", Display: "Destructeur de scorpions", MappingType: mappingTypeAward,
			AwardName: "destroyed_scorpion", AwardCategory: awardCategoryVehicle, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Destructeur_de_scorpions.png",
			Category:    citationCatVehicule,
			Description: "Détruisez les scorpions occupés par l'adversaire.",
			TierTargets: "1,3,5,7,10", Subcategory: citationSubUNSC},
		{Norm: "warthog_destroyer", Display: "Destructeur de warthogs", MappingType: mappingTypeCustom,
			CustomFunction: "compute_warthog_destroyer", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Destructeur_de_warthogs.png",
			Category:    citationCatVehicule,
			Description: "Détruisez les warthogs occupés par l'adversaire.",
			TierTargets: tierTargets5_10_15_25_50, Subcategory: citationSubUNSC},
		{Norm: "wasp_destroyer", Display: "Destructeur de wasps", MappingType: mappingTypeAward,
			AwardName: "destroyed_wasp", AwardCategory: awardCategoryVehicle, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Destructeur_de_wasps.png",
			Category:    citationCatVehicule,
			Description: "Détruisez les wasps occupés par l'adversaire.",
			TierTargets: tierTargets3_6_9_15_30, Subcategory: citationSubUNSC},

		// ── PVE — Firefight (10, dont 4 désactivées) ──────────────────
		{Norm: "grunt_slayer", Display: "Tueur de Grognards", MappingType: mappingTypePVEStat,
			StatName: "grunt_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Tueur_de_Grognards.png",
			Category:    citationCatEnnemi,
			Description: "Tuez des Grognards.",
			TierTargets: tierTargets10_20_30_50_100, Subcategory: citationSubCovenant},
		{Norm: "elite_slayer", Display: "Tueur d'Élites", MappingType: mappingTypePVEStat,
			StatName: "elite_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Tueur_d'Élites.png",
			Category:    citationCatEnnemi,
			Description: "Tuez des Élites.",
			TierTargets: tierTargets10_20_30_50_100, Subcategory: citationSubCovenant},
		{Norm: "jackal_slayer", Display: "Tueur de Rapaces", MappingType: mappingTypePVEStat,
			StatName: "jackal_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Tueur_de_Rapaces.png",
			Category:    citationCatEnnemi,
			Description: "Tuez des Rapaces.",
			TierTargets: tierTargets5_10_15_25_50, Subcategory: citationSubCovenant},
		{Norm: "hunter_slayer", Display: "Tueur de Chasseurs", MappingType: mappingTypePVEStat,
			StatName: "hunter_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Tueur_de_Chasseurs.png",
			Category:    citationCatEnnemi,
			Description: "Tuez des Chasseurs.",
			TierTargets: "2,4,6,10,20", Subcategory: citationSubCovenant},
		{Norm: "sentinel_slayer", Display: "Tueur de sentinelles", MappingType: mappingTypePVEStat,
			StatName: "sentinel_kills", Enabled: false,
			ImagePath:   wpH5 + "H5G_citation_Tueur_de_sentinelles.png",
			Category:    citationCatEnnemi,
			Description: "Tuez des sentinelles.",
			TierTargets: tierTargets5_10_15_25_50, Subcategory: citationSubBanished},
		{Norm: "like_a_boss", Display: "Comme un Boss", MappingType: mappingTypePVEStat,
			StatName: "boss_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Like_a_boss.png",
			Category:    citationCatSpartanCompanies,
			Description: "Tuer un boss dans une partie Baptême du feu zone de combat",
			TierTargets: "250,500,750,1200,2400"},
		{Norm: "player_vs_everything", Display: "Éliminations Firefight", MappingType: mappingTypePVEStat,
			StatName: "total_enemy_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Player_vs_Everything.png",
			Category:    citationCatSpartanCompanies,
			Description: "Gagner des parties en Baptême du feu",
			TierTargets: "200,400,600,960,1940"},
		{Norm: "brute_slayer", Display: "Tueur de Brutes", MappingType: mappingTypePVEStat,
			StatName:    "brute_kills",
			Enabled:     false,
			Category:    citationCatEnnemi,
			Description: "Tuez des Brutes.",
			TierTargets: "10,20,30,50,100", Subcategory: citationSubBanished},
		{Norm: "skimmer_slayer", Display: "Tueur de Skimmers", MappingType: mappingTypePVEStat,
			StatName:    "skimmer_kills",
			Enabled:     false,
			Category:    citationCatEnnemi,
			Description: "Tuez des Skimmers.",
			TierTargets: "10,20,30,50,100", Subcategory: citationSubBanished},
		{Norm: "marine_slayer", Display: "Tueur de Marines", MappingType: mappingTypePVEStat,
			StatName: "marine_kills", Enabled: false,
			ImagePath:   wpH5 + "H5G_citation_Tueur_de_répliques_de_Marines.png",
			Category:    citationCatEnnemi,
			Description: "Tuez les Marines ennemis.",
			TierTargets: "20,40,60,100,200", Subcategory: citationSubBanished},

		// ── Armes — UNSC (10) ─────────────────────────────────────────
		{Norm: "br75_mastery", Display: "Maîtrise du BR75", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:BR75", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_BR75.png",
			Category:    citationCatArme,
			Description: "Éliminez des Spartans avec le BR75.",
			TierTargets: "25,50,100,200,500", Subcategory: citationSubUNSC},
		{Norm: "ma40_mastery", Display: "Maîtrise du MA40 AR", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:MA40 AR", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_MA40.png",
			Category:    citationCatArme,
			Description: "Éliminez des Spartans avec le MA40 AR.",
			TierTargets: "25,50,100,200,500", Subcategory: citationSubUNSC},
		{Norm: "sidekick_mastery", Display: "Maîtrise du MK50 Sidekick", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Mk51 Sidekick", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Sidekick.png",
			Category:    citationCatArme,
			Description: "Éliminez des Spartans avec le MK50 Sidekick.",
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubUNSC},
		{Norm: "commando_mastery", Display: "Maîtrise du VK78 Commando", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:VK78 Commando", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Commando.png",
			Category:    citationCatArme,
			Description: "Éliminez des Spartans avec le VK78 Commando.",
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubUNSC},
		{Norm: "sniper_mastery", Display: "Maîtrise du S7 Sniper", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:S7 Sniper", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Sniper-S7.png",
			Category:    citationCatArme,
			Description: "Éliminez des Spartans avec le S7 Sniper.",
			TierTargets: tierTargets10_20_40_80_200, Subcategory: citationSubUNSC},
		{Norm: "spnkr_mastery", Display: "Maîtrise du M41 SPNKr", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:M41 SPNKr", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_SPNKR.png",
			Category:    citationCatArme,
			Description: "Éliminez des Spartans avec le M41 SPNKr.",
			TierTargets: tierTargets10_20_40_80_200, Subcategory: citationSubUNSC},
		{Norm: "bulldog_mastery", Display: "Maîtrise du CQS48 Bulldog", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:CQS48 Bulldog", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Bulldog.png",
			Category:    citationCatArme,
			Description: "Éliminez des Spartans avec le CQS48 Bulldog.",
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubUNSC},
		{Norm: "bandit_mastery", Display: "Maîtrise du Bandit EVO", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Bandit Evo", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Bandit.png",
			Category:    citationCatArme,
			Description: "Éliminez des Spartans avec le Bandit EVO.",
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubUNSC},
		{Norm: "hydra_mastery", Display: "Maîtrise du MLRS-2 Hydra", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:MLRS-2 Hydra", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Hydra.png",
			Category:    citationCatArme,
			Description: "Éliminez des Spartans avec le MLRS-2 Hydra.",
			TierTargets: "5,10,20,40,100", Subcategory: citationSubUNSC},
		{Norm: "mutilator_mastery", Display: "Maîtrise du Mutilateur", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Mutilator", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Mutilator.png",
			Category:    citationCatArme,
			Description: "Éliminez des Spartans avec le Mutilateur.",
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubUNSC},

		// ── Armes — Paria (9) ─────────────────────────────────────────
		{Norm: "stalker_mastery", Display: "Maîtrise du Fusil traqueur", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Stalker Rifle", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Stalker.png",
			Category:    citationCatArme,
			Description: "Éliminez des Spartans avec le Fusil traqueur.",
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubParia},
		{Norm: "needler_mastery", Display: "Maîtrise du Needler", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Needler", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Needler.png",
			Category:    citationCatArme,
			Description: "Éliminez des Spartans avec le Needler.",
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubParia},
		{Norm: "energy_sword_mastery", Display: "Maîtrise de l'Épée à énergie", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Energy Sword", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Épée_à_énergie.png",
			Category:    citationCatArme,
			Description: "Éliminez des Spartans avec l'Épée à énergie.",
			TierTargets: tierTargets10_20_40_80_200, Subcategory: citationSubParia},
		{Norm: "mangler_mastery", Display: "Maîtrise du Déchiqueteur", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Mangler", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Mangler.png",
			Category:    citationCatArme,
			Description: "Éliminez des Spartans avec le Déchiqueteur.",
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubParia},
		{Norm: "skewer_mastery", Display: "Maîtrise de l'Empaleur", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Skewer", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Skewer.png",
			Category:    citationCatArme,
			Description: "Éliminez des Spartans avec l'Empaleur.",
			TierTargets: "5,10,20,40,100", Subcategory: citationSubParia},
		{Norm: "gravity_hammer_mastery", Display: "Maîtrise du Marteau antigravité", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Gravity Hammer", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Marteau_antigrav.png",
			Category:    citationCatArme,
			Description: "Éliminez des Spartans avec le Marteau antigravité.",
			TierTargets: tierTargets10_20_40_80_200, Subcategory: citationSubParia},
		{Norm: "pulse_carbine_mastery", Display: "Maîtrise de la Carabine à impulsion", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Pulse Carbine", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Carabine.png",
			Category:    citationCatArme,
			Description: "Éliminez des Spartans avec la Carabine à impulsion.",
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubParia},
		{Norm: "ravager_mastery", Display: "Maîtrise du Ravageur", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Ravager", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Ravager.png",
			Category:    citationCatArme,
			Description: "Éliminez des Spartans avec le Ravageur.",
			TierTargets: "5,10,20,40,100", Subcategory: citationSubParia},
		{Norm: "plasma_pistol_mastery", Display: "Maîtrise du Pistolet à plasma", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Plasma Pistol", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Plasma.png",
			Category:    citationCatArme,
			Description: "Éliminez des Spartans avec le Pistolet à plasma.",
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubParia},

		// ── Armes — Forerunner (5) ────────────────────────────────────
		{Norm: "heatwave_mastery", Display: "Maîtrise du Calcineur", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Heatwave", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Heatwave.png",
			Category:    citationCatArme,
			Description: "Éliminez des Spartans avec le Calcineur.",
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubForerunner},
		{Norm: "cindershot_mastery", Display: "Maîtrise du Crémateur", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Cindershot", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Cremator.png",
			Category:    citationCatArme,
			Description: "Éliminez des Spartans avec le Crémateur.",
			TierTargets: tierTargets10_20_40_80_200, Subcategory: citationSubForerunner},
		{Norm: "sentinel_beam_mastery", Display: "Maîtrise du Rayon de Sentinelle", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Sentinel Beam", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Sentinel.png",
			Category:    citationCatArme,
			Description: "Éliminez des Spartans avec le Rayon de Sentinelle.",
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubForerunner},
		{Norm: "disruptor_mastery", Display: "Maîtrise du Disrupteur", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Disruptor", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Disruptor.png",
			Category:    citationCatArme,
			Description: "Éliminez des Spartans avec le Disrupteur.",
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubForerunner},
		{Norm: "shock_rifle_mastery", Display: "Maîtrise du Fusil électrique", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Shock Rifle", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Shock.png",
			Category:    citationCatArme,
			Description: "Éliminez des Spartans avec le Fusil électrique.",
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubForerunner},

		// ── Composites (7) ────────────────────────────────────────────
		{Norm: "covenant_destroyer", Display: "Destructeur de Covenants", MappingType: mappingTypeComposite,
			CompositeChildren: `["grunt_slayer","elite_slayer","jackal_slayer","hunter_slayer","like_a_boss","brute_slayer","skimmer_slayer"]`,
			Enabled:           true,
			ImagePath:         wpH5 + "H5G_maîtrise_Destructeur_de_Covenants.png",
			Category:          citationCatEnnemi,
			Description:       "Obtenez toutes les citations d'élimination de Covenants",
			Subcategory:       citationSubCovenant},
		{Norm: "grenade_mastery", Display: "Maîtrise des grenades", MappingType: mappingTypeComposite,
			CompositeChildren: `["frag_grenade","plasma_grenade"]`,
			Enabled:           true,
			ImagePath:         wpH5 + "H5G_Maîtrise_des_grenades.png",
			Category:          citationCatArme,
			Description:       "Obtenez toutes les citations de grenade.",
			Subcategory:       "Grenade"},
		{Norm: "vehicle_mastery", Display: "Maîtrise de véhicule", MappingType: mappingTypeComposite,
			CompositeChildren: `["splatter","driver","wraith_destroyer","banshee_destroyer","ghost_destroyer","mongoose_destroyer","scorpion_destroyer","warthog_destroyer","wasp_destroyer"]`,
			Enabled:           true,
			ImagePath:         wpH5 + "Vehicle_Mastery.png",
			Category:          citationCatVehicule,
			Description:       "Obtenez toutes les citations de véhicule.",
			Subcategory:       citationSubGeneral},
		{Norm: "human_weapons_mastery", Display: "Maîtrise des armes UNSC", MappingType: mappingTypeComposite,
			CompositeChildren: `["br75_mastery","ma40_mastery","sidekick_mastery","commando_mastery","sniper_mastery","spnkr_mastery","bulldog_mastery","bandit_mastery","hydra_mastery","mutilator_mastery"]`,
			Enabled:           true,
			ImagePath:         wpH5 + "H5G_Maîtrise_en_armes_UNSC.png",
			Category:          citationCatArme,
			Description:       "Obtenez toutes les citations de maîtrise d'armes UNSC.",
			Subcategory:       citationSubUNSC},
		{Norm: "paria_weapons_mastery", Display: "Maîtrise des armes Parias", MappingType: mappingTypeComposite,
			CompositeChildren: `["stalker_mastery","needler_mastery","energy_sword_mastery","mangler_mastery","skewer_mastery","gravity_hammer_mastery","pulse_carbine_mastery","ravager_mastery","plasma_pistol_mastery"]`,
			Enabled:           true,
			ImagePath:         wpHI + "HI_Maîtrise_en_armes_lourdes_Parias.png",
			Category:          citationCatArme,
			Description:       "Obtenez toutes les citations de maîtrise d'armes Parias.",
			Subcategory:       citationSubParia},
		{Norm: "forerunner_weapons_mastery", Display: "Maîtrise des armes Forerunner", MappingType: mappingTypeComposite,
			CompositeChildren: `["heatwave_mastery","cindershot_mastery","sentinel_beam_mastery","disruptor_mastery","shock_rifle_mastery"]`,
			Enabled:           true,
			ImagePath:         wpH5 + "H5G_Maîtrise_en_armes_lourdes_forerunners.png",
			Category:          citationCatArme,
			Description:       "Obtenez toutes les citations de maîtrise d'armes Forerunner.",
			Subcategory:       citationSubForerunner},
		{Norm: "all_weapons_mastery", Display: "Maîtrise en armement", MappingType: mappingTypeComposite,
			CompositeChildren: `["human_weapons_mastery","paria_weapons_mastery","forerunner_weapons_mastery","grenade_mastery"]`,
			Enabled:           true,
			ImagePath:         wpH5 + "H5G_Maîtrise_en_armement.png",
			Category:          citationCatArme,
			Description:       "Obtenez toutes les citations d'armement.",
			Subcategory:       citationSubGeneral},
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// medal_definitions
// ─────────────────────────────────────────────────────────────────────────────

// SeedMedalDefinitions vérifie que medal_definitions est peuplé.
// La logique complète de population vient de l'API 343i (async) — ici on injecte
// les médailles custom si la table est vide.
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
