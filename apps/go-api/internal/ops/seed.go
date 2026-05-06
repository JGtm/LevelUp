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

// ─────────────────────────────────────────────────────────────────────────────
// career_ranks
// ─────────────────────────────────────────────────────────────────────────────

// careerRankJSON représente une entrée dans career_ranks_metadata.json.
type careerRankJSON struct {
	RankID     int    `json:"rank_id"`
	TitleEN    string `json:"title_en"`
	TitleFR    string `json:"title_fr"`
	Subtitle   string `json:"subtitle"`
	Tier       string `json:"tier"`
	Grade      int    `json:"grade"`
	XPRequired int64  `json:"xp_required"`
	IconPath   string `json:"icon_path"`
}

// SeedCareerRanks peuple la table career_ranks depuis le JSON source.
// Portage de populate_career_ranks() Python.
func SeedCareerRanks(opts SeedOptions) (SeedResult, error) {
	jsonPath := opts.DataDir + "/cache/career_ranks_metadata.json"
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return SeedResult{Component: "career_ranks"}, fmt.Errorf("lecture %s: %w", jsonPath, err)
	}
	var ranks []careerRankJSON
	if err := json.Unmarshal(data, &ranks); err != nil {
		return SeedResult{Component: "career_ranks"}, fmt.Errorf("parse JSON: %w", err)
	}

	db, err := sql.Open("duckdb", opts.MetaDBPath)
	if err != nil {
		return SeedResult{Component: "career_ranks"}, fmt.Errorf("ouverture metadata: %w", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS career_ranks (
		rank_id INTEGER PRIMARY KEY,
		title_en VARCHAR, title_fr VARCHAR,
		subtitle VARCHAR, tier VARCHAR, grade INTEGER,
		xp_required BIGINT, icon_path VARCHAR
	)`); err != nil {
		return SeedResult{Component: "career_ranks"}, err
	}

	inserted, skipped := 0, 0
	for _, r := range ranks {
		res, err := db.Exec(`INSERT OR IGNORE INTO career_ranks VALUES (?,?,?,?,?,?,?,?)`,
			r.RankID, r.TitleEN, r.TitleFR, r.Subtitle, r.Tier, r.Grade, r.XPRequired, r.IconPath)
		if err != nil {
			return SeedResult{Component: "career_ranks"}, fmt.Errorf("insert rank %d: %w", r.RankID, err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			skipped++
		} else {
			inserted++
		}
	}
	return SeedResult{
		Component: "career_ranks",
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
func SeedCitationMappings(opts SeedOptions) (SeedResult, error) {
	db, err := sql.Open("duckdb", opts.MetaDBPath)
	if err != nil {
		return SeedResult{Component: "citation_mappings"}, fmt.Errorf("ouverture metadata: %w", err)
	}
	defer db.Close()

	// Schéma v2 complet : compatible migrations add_citation_mappings +
	// add_citation_mappings_pk + add_citation_mappings_v2_fields.
	if _, err := db.Exec(`
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
		return SeedResult{Component: "citation_mappings"}, fmt.Errorf("create schema: %w", err)
	}

	existing := make(map[string]struct{})
	rows, err := db.Query(`SELECT citation_name_norm FROM citation_mappings`)
	if err != nil {
		return SeedResult{Component: "citation_mappings"}, fmt.Errorf("scan existing: %w", err)
	}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			rows.Close()
			return SeedResult{Component: "citation_mappings"}, err
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
		if _, err := db.Exec(upsert,
			m.Norm, m.Display, m.MappingType,
			medalArg, nullStr(m.MedalIDs), nullStr(m.StatName),
			nullStr(m.AwardName), nullStr(m.AwardCategory),
			nullStr(m.CustomFunction), nullStr(m.CompositeChildren), m.Enabled,
			nullStr(m.ImagePath), m.Category, m.Description,
			nullStr(m.TierTargets), nullStr(m.Subcategory),
		); err != nil {
			return SeedResult{Component: "citation_mappings"}, fmt.Errorf("upsert %s: %w", m.Norm, err)
		}
		if _, present := existing[m.Norm]; present {
			skipped++
		} else {
			inserted++
		}
	}
	return SeedResult{
		Component: "citation_mappings",
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
		{Norm: "charge", Display: "À la charge", MappingType: "award",
			AwardName: "zone_captured", AwardCategory: "objective", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_%C3%80_la_charge.png",
			Category:    "Mode de jeu",
			Description: "Prenez le contrôle d'une base dans n'importe quelle partie matchmaking Bases.",
			TierTargets: "10,20,30,50,100"},
		{Norm: "forced_annexation", Display: "Annexion forcée", MappingType: "custom",
			CustomFunction: "compute_annexion_forcee", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Annexion_forc%C3%A9e.png",
			Category:    "Mode de jeu",
			Description: "Prenez le contrôle de 3 bases sans mourir dans n'importe quelle partie matchmaking Bases.",
			TierTargets: "3,6,9,15,30"},
		{Norm: "assistant", Display: "Assistant", MappingType: "stat",
			StatName: "assists", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Assistant.png",
			Category:    "Mode de jeu",
			Description: "Remportez n'importe quelle médaille d'assistance en Zone de combat.",
			TierTargets: "25,50,75,125,250"},
		{Norm: "bulldozer", Display: "Bulldozer", MappingType: "custom",
			CustomFunction: "compute_bulldozer", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Bulldozer.png",
			Category:    "Mode de jeu",
			Description: "Terminez n'importe quelle partie matchmaking Assassin avec un FDA supérieur à 8.",
			TierTargets: "3,6,9,15,30"},
		{Norm: "flag_defender", Display: "Défenseur du drapeau", MappingType: "award",
			AwardName: "carrier_killed", AwardCategory: "objective", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_D%C3%A9fenseur_du_drapeau.png",
			Category:    "Mode de jeu",
			Description: "Protégez le drapeau de votre équipe dans n'importe quelle partie matchmaking Capture du drapeau.",
			TierTargets: "10,20,30,50,100"},
		{Norm: "got_you", Display: "Je te tiens !", MappingType: "award",
			AwardName: "flag_returned", AwardCategory: "objective", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Je_te_tiens_%21.png",
			Category:    "Mode de jeu",
			Description: "Rapportez le drapeau de votre équipe dans n'importe quelle partie matchmaking Capture du drapeau.",
			TierTargets: "10,20,30,50,100"},
		{Norm: "stakeholder", Display: "Partie prenante", MappingType: "award",
			AwardName: "zone_secured", AwardCategory: "objective", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Partie_prenante.png",
			Category:    "Mode de jeu",
			Description: "Défendez une base appartenant à votre équipe dans n'importe quelle partie matchmaking Bases.",
			TierTargets: "10,20,30,50,100"},
		{Norm: "flag_carrier_hunter", Display: "Sus au porteur du drapeau", MappingType: "award",
			AwardName: "carrier_killed", AwardCategory: "objective", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Sus_au_porteur_du_drapeau.png",
			Category:    "Mode de jeu",
			Description: "Tuez un porte-drapeau ennemi dans n'importe quelle partie matchmaking Capture du drapeau.",
			TierTargets: "10,20,30,50,100"},
		{Norm: "flag_victory", Display: "Victoire au drapeau", MappingType: "custom",
			CustomFunction: "compute_wins_ctf", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Victoire_au_drapeau.png",
			Category:    "Mode de jeu",
			Description: "Remportez n'importe quelle partie matchmaking Capture du drapeau.",
			TierTargets: "5,10,15,25,50"},
		{Norm: "slayer_victory", Display: "Victoire en assassin", MappingType: "custom",
			CustomFunction: "compute_wins_slayer", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Victoire_en_Assassin.png",
			Category:    "Mode de jeu",
			Description: "Remportez une partie d'Assassin en équipe.",
			TierTargets: "5,10,15,25,50"},
		{Norm: "strongholds_victory", Display: "Victoire en bases", MappingType: "custom",
			CustomFunction: "compute_wins_strongholds", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Victoire_en_Bases.png",
			Category:    "Mode de jeu",
			Description: "Remportez n'importe quelle partie matchmaking Bases.",
			TierTargets: "5,10,15,25,50"},

		// ── PVP — Véhicule (2) + Grenade (2) ──────────────────────────
		{Norm: "splatter", Display: "Écrasement", MappingType: "medal", MedalID: 221693153, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_%C3%89crasement.png",
			Category:    "Véhicule",
			Description: "Écrasez un Spartan adverse avec un véhicule.",
			TierTargets: "10,20,30,50,100", Subcategory: "Général"},
		{Norm: "driver", Display: "Pilote", MappingType: "medal", MedalID: 3169118333, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Pilote.png",
			Category:    "Véhicule",
			Description: "Décrochez des médailles de pilote.",
			TierTargets: "10,20,30,50,100", Subcategory: "Général"},
		{Norm: "frag_grenade", Display: "Grenade à fragmentation", MappingType: "medal", MedalID: 2648272972, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Grenade_%C3%A0_fragmentation.png",
			Category:    "Arme",
			Description: "Tuez un Spartan adverse à l'aide d'une grenade à fragmentation.",
			TierTargets: "5,10,15,25,50", Subcategory: "Grenade"},
		{Norm: "plasma_grenade", Display: "Grenade à plasma", MappingType: "medal", MedalID: 3655682764, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Grenade_%C3%A0_plasma.png",
			Category:    "Arme",
			Description: "Tuez un Spartan adverse à l'aide d'une grenade à plasma.",
			TierTargets: "2,4,6,10,20", Subcategory: "Grenade"},

		// ── PVP — Multijoueur (10) ────────────────────────────────────
		{Norm: "assassin", Display: "Assassin", MappingType: "medal", MedalID: 548533137, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Assassin.png",
			Category:    "Multijoueur",
			Description: "Assassinez des Spartans adverses.",
			TierTargets: "5,10,15,25,50"},
		{Norm: "spartan_carnage", Display: "Carnage de Spartans", MappingType: "medal",
			MedalIDs:    "2780740615,4261842076,418532952,1486797009,710323196,1720896992,2567026752,2875941471",
			Enabled:     true,
			ImagePath:   wpH5 + "H5G_citation_Carnage_de_Spartans.png",
			Category:    "Multijoueur",
			Description: "Tuez plusieurs Spartans adverses sans mourir.",
			TierTargets: "3,6,9,15,30"},
		{Norm: "close_combat", Display: "Combat rapproché", MappingType: "stat",
			StatName: "melee_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Combat_rapproch%C3%A9.png",
			Category:    "Multijoueur",
			Description: "Remportez n'importe quelle médaille de combat rapproché.",
			TierTargets: "10,20,30,50,100"},
		{Norm: "opportunist", Display: "Combattant opportuniste", MappingType: "medal",
			MedalIDs:    "622331684,2063152177,4261842076,2137071619,1486797009,1430343434,2242633421",
			Enabled:     true,
			ImagePath:   wpH5 + "H5G_citation_Combattant_opportuniste.png",
			Category:    "Multijoueur",
			Description: "Remportez n'importe quelle médaille d'aptitude au combat.",
			TierTargets: "10,20,30,50,100"},
		{Norm: "multikill", Display: "Multifrag", MappingType: "medal", MedalID: 622331684, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Multifrag.png",
			Category:    "Multijoueur",
			Description: "Tuez rapidement plusieurs Spartans adverses.",
			TierTargets: "3,6,9,15,30"},
		{Norm: "melee_fighter", Display: "Pugilat", MappingType: "stat",
			StatName: "melee_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Pugilat.png",
			Category:    "Multijoueur",
			Description: "Tuez un Spartan ennemi d'une attaque rapprochée.",
			TierTargets: "5,10,15,25,50"},
		{Norm: "headshot", Display: "Tir à la tête", MappingType: "stat",
			StatName: "headshot_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Tir_%C3%A0_la_t%C3%AAte.png",
			Category:    "Multijoueur",
			Description: "Tuez un Spartan ennemi d'un tir à la tête.",
			TierTargets: "10,20,30,50,100"},
		{Norm: "spartan_killer", Display: "Tueur de Spartans", MappingType: "stat",
			StatName: "kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Tueur_de_Spartans.png",
			Category:    "Multijoueur",
			Description: "Éliminez les Spartans ennemis.",
			TierTargets: "20,40,60,100,200"},
		{Norm: "eagle_eye", Display: "Œil de lynx", MappingType: "medal", MedalID: 1512363953, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_%C5%92il_de_lynx.png",
			Category:    "Multijoueur",
			Description: "Tuez un Spartan adverse en pleine santé à l'aide d'une arme de précision sans manquer un seul coup.",
			TierTargets: "10,20,30,50,100"},
		{Norm: "avenger", Display: "Vengeur", MappingType: "medal", MedalID: 9000000001, Enabled: true,
			ImagePath:   wpH5 + "Assassin_commendation.png",
			Category:    "Multijoueur",
			Description: "Tuez l'ennemi responsable de votre mort précédente.",
			TierTargets: "5,15,30,55,105"},

		// ── PVP — Spartan Companies (15) ──────────────────────────────
		{Norm: "flag_em_down", Display: "Sors les drapeaux", MappingType: "custom",
			CustomFunction: "compute_flag_em_down", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Flag_%27em_down.png",
			Category:    "Spartan Companies",
			Description: "Obtenir une des médailles suivantes : Interception, Bataille de drapeaux, Frag du porteur, Retour du drapeau, Défense du drapeau",
			TierTargets: "1000,2000,3000,4800,9700"},
		{Norm: "grand_theft", Display: "Vol à la tire", MappingType: "custom",
			CustomFunction: "compute_hijack", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Grand_Theft.png",
			Category:    "Spartan Companies",
			Description: "Aborder un véhicule ou un véhicule aérien",
			TierTargets: "200,400,600,960,1940"},
		{Norm: "helping_hand", Display: "Coup de main", MappingType: "stat",
			StatName: "assists", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Helping_Hand.png",
			Category:    "Spartan Companies",
			Description: "Obtenir n'importe quelle assistance",
			TierTargets: "20000,40000,60000,96000,194400"},
		{Norm: "im_just_perfect", Display: "Zéro défaut", MappingType: "medal", MedalID: 1512363953, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_I%27m_just_perfect.png",
			Category:    "Spartan Companies",
			Description: "Tuer un joueur avec une arme de précision sans manquer un tir",
			TierTargets: "2000,4000,6000,9600,19400"},
		{Norm: "lawnmower", Display: "Tondeuse", MappingType: "medal", MedalID: 221693153, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Lawnmower.png",
			Category:    "Spartan Companies",
			Description: "Tuer un adversaire en l'écrasant avec un véhicule",
			TierTargets: "500,1000,1500,2400,4900"},
		{Norm: "look_ma_no_pin", Display: "Regarde maman, sans goupille", MappingType: "stat",
			StatName: "grenade_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Look_ma_no_pin.png",
			Category:    "Spartan Companies",
			Description: "Tuer un Spartan ennemi avec n'importe quelle grenade",
			TierTargets: "4000,8000,12000,19200,38900"},
		{Norm: "lucky", Display: "Lucky", MappingType: "medal",
			MedalIDs: "3905838030,3091261182", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Lucky.png",
			Category:    "Spartan Companies",
			Description: "Obtenir les médailles La chance, Chargeur vide, Sayonara ou Boulet de canon",
			TierTargets: "400,800,1200,1920,3880"},
		{Norm: "no_hard_feelings", Display: "Sans rancune", MappingType: "stat",
			StatName: "kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_No_Hard_Feelings.png",
			Category:    "Spartan Companies",
			Description: "Tuer des Spartans ennemis",
			TierTargets: "50000,100000,150000,240000,486000"},
		{Norm: "positive_contribution", Display: "Positive contribution", MappingType: "custom",
			CustomFunction: "compute_bulldozer", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Positive_contribution.png",
			Category:    "Spartan Companies",
			Description: "Finir avec FDA supérieur à 8 dans n'importe quelle partie Assassin en matchmaking",
			TierTargets: "300,600,900,1440,2960"},
		{Norm: "power_play", Display: "Coup de force", MappingType: "stat",
			StatName: "power_weapon_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Power_play.png",
			Category:    "Spartan Companies",
			Description: "Tuer un Spartan ennemi avec une arme puissante",
			TierTargets: "10000,20000,30000,48000,97200"},
		{Norm: "road_trip", Display: "Virée sur la route", MappingType: "medal", MedalID: 3169118333, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Road_Trip.png",
			Category:    "Spartan Companies",
			Description: "Tuer un Spartan ennemi avec un véhicule terrestre",
			TierTargets: "3000,6000,9000,14400,29200"},
		{Norm: "sting_like_a_bee", Display: "Pique comme une abeille", MappingType: "stat",
			StatName: "melee_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Sting_like_a_bee.png",
			Category:    "Spartan Companies",
			Description: "Tuer un Spartan ennemi en combat rapproché",
			TierTargets: "5000,10000,15000,24000,48600"},
		{Norm: "the_reaper", Display: "Le faucheur", MappingType: "medal", MedalID: 2625820422, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_The_Reaper.png",
			Category:    "Spartan Companies",
			Description: "Tuer un adversaire d'outre-tombe",
			TierTargets: "500,1000,1500,2400,4850"},
		{Norm: "too_fast_for_you", Display: "Trop rapide pour toi", MappingType: "medal", MedalID: 2123530881, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Too_fast_for_you.png",
			Category:    "Spartan Companies",
			Description: "Tuer un adversaire qui vous a tiré dessus en premier",
			TierTargets: "2000,4000,6000,9600,19400"},
		{Norm: "vandalism", Display: "Vandalisme", MappingType: "custom",
			CustomFunction: "compute_vandalism", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Vandalism.png",
			Category:    "Spartan Companies",
			Description: "Détruire un véhicule ennemi",
			TierTargets: "1200,2400,3600,5760,11640"},

		// ── PVP — Véhicules destructeurs (7) ─────────────────────────
		{Norm: "wraith_destroyer", Display: "Destructeur d'apparitions", MappingType: "custom",
			CustomFunction: "compute_wraith_destroyer", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Destructeur_d%27apparitions.png",
			Category:    "Véhicule",
			Description: "Détruisez les apparitions occupées par l'adversaire.",
			TierTargets: "3,6,9,15,30", Subcategory: "Covenant"},
		{Norm: "banshee_destroyer", Display: "Destructeur de banshees", MappingType: "award",
			AwardName: "destroyed_banshee", AwardCategory: "vehicle", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Destructeur_de_banshees.png",
			Category:    "Véhicule",
			Description: "Détruisez les banshees occupés par l'adversaire.",
			TierTargets: "3,6,9,15,30", Subcategory: "Covenant"},
		{Norm: "ghost_destroyer", Display: "Destructeur de ghosts", MappingType: "award",
			AwardName: "destroyed_ghost", AwardCategory: "vehicle", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Destructeur_de_ghosts.png",
			Category:    "Véhicule",
			Description: "Détruisez les ghosts occupés par l'adversaire.",
			TierTargets: "5,10,15,25,50", Subcategory: "Covenant"},
		{Norm: "mongoose_destroyer", Display: "Destructeur de mongooses", MappingType: "custom",
			CustomFunction: "compute_mongoose_destroyer", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Destructeur_de_mongooses.png",
			Category:    "Véhicule",
			Description: "Détruisez les mongooses occupées par l'adversaire.",
			TierTargets: "5,10,15,25,50", Subcategory: "UNSC"},
		{Norm: "scorpion_destroyer", Display: "Destructeur de scorpions", MappingType: "award",
			AwardName: "destroyed_scorpion", AwardCategory: "vehicle", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Destructeur_de_scorpions.png",
			Category:    "Véhicule",
			Description: "Détruisez les scorpions occupés par l'adversaire.",
			TierTargets: "1,3,5,7,10", Subcategory: "UNSC"},
		{Norm: "warthog_destroyer", Display: "Destructeur de warthogs", MappingType: "custom",
			CustomFunction: "compute_warthog_destroyer", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Destructeur_de_warthogs.png",
			Category:    "Véhicule",
			Description: "Détruisez les warthogs occupés par l'adversaire.",
			TierTargets: "5,10,15,25,50", Subcategory: "UNSC"},
		{Norm: "wasp_destroyer", Display: "Destructeur de wasps", MappingType: "award",
			AwardName: "destroyed_wasp", AwardCategory: "vehicle", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Destructeur_de_wasps.png",
			Category:    "Véhicule",
			Description: "Détruisez les wasps occupés par l'adversaire.",
			TierTargets: "3,6,9,15,30", Subcategory: "UNSC"},

		// ── PVE — Firefight (10, dont 4 désactivées) ──────────────────
		{Norm: "grunt_slayer", Display: "Tueur de Grognards", MappingType: "pve_stat",
			StatName: "grunt_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Tueur_de_Grognards.png",
			Category:    "Ennemi",
			Description: "Tuez des Grognards.",
			TierTargets: "10,20,30,50,100", Subcategory: "Covenant"},
		{Norm: "elite_slayer", Display: "Tueur d'Élites", MappingType: "pve_stat",
			StatName: "elite_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Tueur_d%27%C3%89lites.png",
			Category:    "Ennemi",
			Description: "Tuez des Élites.",
			TierTargets: "10,20,30,50,100", Subcategory: "Covenant"},
		{Norm: "jackal_slayer", Display: "Tueur de Rapaces", MappingType: "pve_stat",
			StatName: "jackal_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Tueur_de_Rapaces.png",
			Category:    "Ennemi",
			Description: "Tuez des Rapaces.",
			TierTargets: "5,10,15,25,50", Subcategory: "Covenant"},
		{Norm: "hunter_slayer", Display: "Tueur de Chasseurs", MappingType: "pve_stat",
			StatName: "hunter_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Tueur_de_Chasseurs.png",
			Category:    "Ennemi",
			Description: "Tuez des Chasseurs.",
			TierTargets: "2,4,6,10,20", Subcategory: "Covenant"},
		{Norm: "sentinel_slayer", Display: "Tueur de sentinelles", MappingType: "pve_stat",
			StatName: "sentinel_kills", Enabled: false,
			ImagePath:   wpH5 + "H5G_citation_Tueur_de_sentinelles.png",
			Category:    "Ennemi",
			Description: "Tuez des sentinelles.",
			TierTargets: "5,10,15,25,50", Subcategory: "Banished"},
		{Norm: "like_a_boss", Display: "Comme un Boss", MappingType: "pve_stat",
			StatName: "boss_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Like_a_boss.png",
			Category:    "Spartan Companies",
			Description: "Tuer un boss dans une partie Baptême du feu zone de combat",
			TierTargets: "250,500,750,1200,2400"},
		{Norm: "player_vs_everything", Display: "Éliminations Firefight", MappingType: "pve_stat",
			StatName: "total_enemy_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Player_vs_Everything.png",
			Category:    "Spartan Companies",
			Description: "Gagner des parties en Baptême du feu",
			TierTargets: "200,400,600,960,1940"},
		{Norm: "brute_slayer", Display: "Tueur de Brutes", MappingType: "pve_stat",
			StatName:    "brute_kills",
			Enabled:     false,
			Category:    "Ennemi",
			Description: "Tuez des Brutes.",
			Subcategory: "Banished"},
		{Norm: "skimmer_slayer", Display: "Tueur de Skimmers", MappingType: "pve_stat",
			StatName:    "skimmer_kills",
			Enabled:     false,
			Category:    "Ennemi",
			Description: "Tuez des Skimmers.",
			Subcategory: "Banished"},
		{Norm: "marine_slayer", Display: "Tueur de Marines", MappingType: "pve_stat",
			StatName: "marine_kills", Enabled: false,
			ImagePath:   wpH5 + "H5G_citation_Tueur_de_r%C3%A9pliques_de_Marines.png",
			Category:    "Ennemi",
			Description: "Tuez les Marines ennemis.",
			TierTargets: "20,40,60,100,200", Subcategory: "Banished"},

		// ── Armes — UNSC (10) ─────────────────────────────────────────
		{Norm: "br75_mastery", Display: "Maîtrise du BR75", MappingType: "weapon_stat",
			StatName: "weapon_kills:BR75", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_BR75.png",
			Category:    "Arme",
			Description: "Éliminez des Spartans avec le BR75.",
			TierTargets: "25,50,100,200,500", Subcategory: "UNSC"},
		{Norm: "ma40_mastery", Display: "Maîtrise du MA40 AR", MappingType: "weapon_stat",
			StatName: "weapon_kills:MA40 AR", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_MA40.png",
			Category:    "Arme",
			Description: "Éliminez des Spartans avec le MA40 AR.",
			TierTargets: "25,50,100,200,500", Subcategory: "UNSC"},
		{Norm: "sidekick_mastery", Display: "Maîtrise du MK50 Sidekick", MappingType: "weapon_stat",
			StatName: "weapon_kills:Mk51 Sidekick", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Sidekick.png",
			Category:    "Arme",
			Description: "Éliminez des Spartans avec le MK50 Sidekick.",
			TierTargets: "10,25,50,100,250", Subcategory: "UNSC"},
		{Norm: "commando_mastery", Display: "Maîtrise du VK78 Commando", MappingType: "weapon_stat",
			StatName: "weapon_kills:VK78 Commando", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Commando.png",
			Category:    "Arme",
			Description: "Éliminez des Spartans avec le VK78 Commando.",
			TierTargets: "10,25,50,100,250", Subcategory: "UNSC"},
		{Norm: "sniper_mastery", Display: "Maîtrise du S7 Sniper", MappingType: "weapon_stat",
			StatName: "weapon_kills:S7 Sniper", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Sniper-S7.png",
			Category:    "Arme",
			Description: "Éliminez des Spartans avec le S7 Sniper.",
			TierTargets: "10,20,40,80,200", Subcategory: "UNSC"},
		{Norm: "spnkr_mastery", Display: "Maîtrise du M41 SPNKr", MappingType: "weapon_stat",
			StatName: "weapon_kills:M41 SPNKr", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_SPNKR.png",
			Category:    "Arme",
			Description: "Éliminez des Spartans avec le M41 SPNKr.",
			TierTargets: "10,20,40,80,200", Subcategory: "UNSC"},
		{Norm: "bulldog_mastery", Display: "Maîtrise du CQS48 Bulldog", MappingType: "weapon_stat",
			StatName: "weapon_kills:CQS48 Bulldog", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Bulldog.png",
			Category:    "Arme",
			Description: "Éliminez des Spartans avec le CQS48 Bulldog.",
			TierTargets: "10,25,50,100,250", Subcategory: "UNSC"},
		{Norm: "bandit_mastery", Display: "Maîtrise du Bandit EVO", MappingType: "weapon_stat",
			StatName: "weapon_kills:Bandit Evo", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Bandit.png",
			Category:    "Arme",
			Description: "Éliminez des Spartans avec le Bandit EVO.",
			TierTargets: "10,25,50,100,250", Subcategory: "UNSC"},
		{Norm: "hydra_mastery", Display: "Maîtrise du MLRS-2 Hydra", MappingType: "weapon_stat",
			StatName: "weapon_kills:MLRS-2 Hydra", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Hydra.png",
			Category:    "Arme",
			Description: "Éliminez des Spartans avec le MLRS-2 Hydra.",
			TierTargets: "5,10,20,40,100", Subcategory: "UNSC"},
		{Norm: "mutilator_mastery", Display: "Maîtrise du Mutilateur", MappingType: "weapon_stat",
			StatName: "weapon_kills:Mutilator", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Mutilator.png",
			Category:    "Arme",
			Description: "Éliminez des Spartans avec le Mutilateur.",
			TierTargets: "10,25,50,100,250", Subcategory: "UNSC"},

		// ── Armes — Paria (9) ─────────────────────────────────────────
		{Norm: "stalker_mastery", Display: "Maîtrise du Fusil traqueur", MappingType: "weapon_stat",
			StatName: "weapon_kills:Stalker Rifle", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Stalker.png",
			Category:    "Arme",
			Description: "Éliminez des Spartans avec le Fusil traqueur.",
			TierTargets: "10,25,50,100,250", Subcategory: "Paria"},
		{Norm: "needler_mastery", Display: "Maîtrise du Needler", MappingType: "weapon_stat",
			StatName: "weapon_kills:Needler", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Needler.png",
			Category:    "Arme",
			Description: "Éliminez des Spartans avec le Needler.",
			TierTargets: "10,25,50,100,250", Subcategory: "Paria"},
		{Norm: "energy_sword_mastery", Display: "Maîtrise de l'Épée à énergie", MappingType: "weapon_stat",
			StatName: "weapon_kills:Energy Sword", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_%C3%89p%C3%A9e_%C3%A0_%C3%A9nergie.png",
			Category:    "Arme",
			Description: "Éliminez des Spartans avec l'Épée à énergie.",
			TierTargets: "10,20,40,80,200", Subcategory: "Paria"},
		{Norm: "mangler_mastery", Display: "Maîtrise du Déchiqueteur", MappingType: "weapon_stat",
			StatName: "weapon_kills:Mangler", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Mangler.png",
			Category:    "Arme",
			Description: "Éliminez des Spartans avec le Déchiqueteur.",
			TierTargets: "10,25,50,100,250", Subcategory: "Paria"},
		{Norm: "skewer_mastery", Display: "Maîtrise de l'Empaleur", MappingType: "weapon_stat",
			StatName: "weapon_kills:Skewer", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Skewer.png",
			Category:    "Arme",
			Description: "Éliminez des Spartans avec l'Empaleur.",
			TierTargets: "5,10,20,40,100", Subcategory: "Paria"},
		{Norm: "gravity_hammer_mastery", Display: "Maîtrise du Marteau antigravité", MappingType: "weapon_stat",
			StatName: "weapon_kills:Gravity Hammer", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Marteau_antigrav.png",
			Category:    "Arme",
			Description: "Éliminez des Spartans avec le Marteau antigravité.",
			TierTargets: "10,20,40,80,200", Subcategory: "Paria"},
		{Norm: "pulse_carbine_mastery", Display: "Maîtrise de la Carabine à impulsion", MappingType: "weapon_stat",
			StatName: "weapon_kills:Pulse Carbine", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Carabine.png",
			Category:    "Arme",
			Description: "Éliminez des Spartans avec la Carabine à impulsion.",
			TierTargets: "10,25,50,100,250", Subcategory: "Paria"},
		{Norm: "ravager_mastery", Display: "Maîtrise du Ravageur", MappingType: "weapon_stat",
			StatName: "weapon_kills:Ravager", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Ravager.png",
			Category:    "Arme",
			Description: "Éliminez des Spartans avec le Ravageur.",
			TierTargets: "5,10,20,40,100", Subcategory: "Paria"},
		{Norm: "plasma_pistol_mastery", Display: "Maîtrise du Pistolet à plasma", MappingType: "weapon_stat",
			StatName: "weapon_kills:Plasma Pistol", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Plasma.png",
			Category:    "Arme",
			Description: "Éliminez des Spartans avec le Pistolet à plasma.",
			TierTargets: "10,25,50,100,250", Subcategory: "Paria"},

		// ── Armes — Forerunner (5) ────────────────────────────────────
		{Norm: "heatwave_mastery", Display: "Maîtrise du Calcineur", MappingType: "weapon_stat",
			StatName: "weapon_kills:Heatwave", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Heatwave.png",
			Category:    "Arme",
			Description: "Éliminez des Spartans avec le Calcineur.",
			TierTargets: "10,25,50,100,250", Subcategory: "Forerunner"},
		{Norm: "cindershot_mastery", Display: "Maîtrise du Crémateur", MappingType: "weapon_stat",
			StatName: "weapon_kills:Cindershot", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Cremator.png",
			Category:    "Arme",
			Description: "Éliminez des Spartans avec le Crémateur.",
			TierTargets: "10,20,40,80,200", Subcategory: "Forerunner"},
		{Norm: "sentinel_beam_mastery", Display: "Maîtrise du Rayon de Sentinelle", MappingType: "weapon_stat",
			StatName: "weapon_kills:Sentinel Beam", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Sentinel.png",
			Category:    "Arme",
			Description: "Éliminez des Spartans avec le Rayon de Sentinelle.",
			TierTargets: "10,25,50,100,250", Subcategory: "Forerunner"},
		{Norm: "disruptor_mastery", Display: "Maîtrise du Disrupteur", MappingType: "weapon_stat",
			StatName: "weapon_kills:Disruptor", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Disruptor.png",
			Category:    "Arme",
			Description: "Éliminez des Spartans avec le Disrupteur.",
			TierTargets: "10,25,50,100,250", Subcategory: "Forerunner"},
		{Norm: "shock_rifle_mastery", Display: "Maîtrise du Fusil électrique", MappingType: "weapon_stat",
			StatName: "weapon_kills:Shock Rifle", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Shock.png",
			Category:    "Arme",
			Description: "Éliminez des Spartans avec le Fusil électrique.",
			TierTargets: "10,25,50,100,250", Subcategory: "Forerunner"},

		// ── Composites (7) ────────────────────────────────────────────
		{Norm: "covenant_destroyer", Display: "Destructeur de Covenants", MappingType: "composite",
			CompositeChildren: `["grunt_slayer","elite_slayer","jackal_slayer","hunter_slayer","like_a_boss","brute_slayer","skimmer_slayer"]`,
			Enabled:           true,
			ImagePath:         wpH5 + "H5G_maîtrise_Destructeur_de_Covenants.png",
			Category:          "Ennemi",
			Description:       "Obtenez toutes les citations d'élimination de Covenants",
			Subcategory:       "Covenant"},
		{Norm: "grenade_mastery", Display: "Maîtrise des grenades", MappingType: "composite",
			CompositeChildren: `["frag_grenade","plasma_grenade"]`,
			Enabled:           true,
			ImagePath:         wpH5 + "H5G_Maîtrise_des_grenades.png",
			Category:          "Arme",
			Description:       "Obtenez toutes les citations de grenade.",
			Subcategory:       "Grenade"},
		{Norm: "vehicle_mastery", Display: "Maîtrise de véhicule", MappingType: "composite",
			CompositeChildren: `["splatter","driver","wraith_destroyer","banshee_destroyer","ghost_destroyer","mongoose_destroyer","scorpion_destroyer","warthog_destroyer","wasp_destroyer"]`,
			Enabled:           true,
			ImagePath:         wpH5 + "Vehicle_Mastery.png",
			Category:          "Véhicule",
			Description:       "Obtenez toutes les citations de véhicule.",
			Subcategory:       "Général"},
		{Norm: "human_weapons_mastery", Display: "Maîtrise des armes UNSC", MappingType: "composite",
			CompositeChildren: `["br75_mastery","ma40_mastery","sidekick_mastery","commando_mastery","sniper_mastery","spnkr_mastery","bulldog_mastery","bandit_mastery","hydra_mastery","mutilator_mastery"]`,
			Enabled:           true,
			ImagePath:         wpH5 + "H5G_Maîtrise_en_armes_UNSC.png",
			Category:          "Arme",
			Description:       "Obtenez toutes les citations de maîtrise d'armes UNSC.",
			Subcategory:       "UNSC"},
		{Norm: "paria_weapons_mastery", Display: "Maîtrise des armes Parias", MappingType: "composite",
			CompositeChildren: `["stalker_mastery","needler_mastery","energy_sword_mastery","mangler_mastery","skewer_mastery","gravity_hammer_mastery","pulse_carbine_mastery","ravager_mastery","plasma_pistol_mastery"]`,
			Enabled:           true,
			ImagePath:         wpHI + "HI_Maîtrise_en_armes_lourdes_Parias.png",
			Category:          "Arme",
			Description:       "Obtenez toutes les citations de maîtrise d'armes Parias.",
			Subcategory:       "Paria"},
		{Norm: "forerunner_weapons_mastery", Display: "Maîtrise des armes Forerunner", MappingType: "composite",
			CompositeChildren: `["heatwave_mastery","cindershot_mastery","sentinel_beam_mastery","disruptor_mastery","shock_rifle_mastery"]`,
			Enabled:           true,
			ImagePath:         wpH5 + "H5G_Maîtrise_en_armes_lourdes_forerunners.png",
			Category:          "Arme",
			Description:       "Obtenez toutes les citations de maîtrise d'armes Forerunner.",
			Subcategory:       "Forerunner"},
		{Norm: "all_weapons_mastery", Display: "Maîtrise en armement", MappingType: "composite",
			CompositeChildren: `["human_weapons_mastery","paria_weapons_mastery","forerunner_weapons_mastery","grenade_mastery"]`,
			Enabled:           true,
			ImagePath:         wpH5 + "H5G_Maîtrise_en_armement.png",
			Category:          "Arme",
			Description:       "Obtenez toutes les citations d'armement.",
			Subcategory:       "Général"},
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// medal_definitions
// ─────────────────────────────────────────────────────────────────────────────

// SeedMedalDefinitions vérifie que medal_definitions est peuplé.
// La logique complète de population vient de l'API 343i (async) — ici on injecte
// les médailles custom si la table est vide.
func SeedMedalDefinitions(opts SeedOptions) (SeedResult, error) {
	db, err := sql.Open("duckdb", opts.MetaDBPath)
	if err != nil {
		return SeedResult{Component: "medal_definitions"}, fmt.Errorf("ouverture metadata: %w", err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'medal_definitions'`).Scan(&count); err != nil || count == 0 {
		return SeedResult{
			Component: "medal_definitions",
			Message:   "table absente — lancer d'abord les migrations",
		}, nil
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM medal_definitions").Scan(&count); err != nil {
		return SeedResult{Component: "medal_definitions"}, err
	}
	return SeedResult{
		Component: "medal_definitions",
		Skipped:   count,
		Message:   fmt.Sprintf("%d médailles présentes", count),
	}, nil
}
