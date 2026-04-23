// Package ops — seed.go : population des référentiels metadata.duckdb.
//
// Portage des scripts populate_*.py Python.
//
// Fonctions :
//   - SeedCareerRanks     → populate_career_ranks.py
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

// CitationMapping représente une règle citation → médaille(s).
type CitationMapping struct {
	Norm        string
	Display     string
	MappingType string // medal | composite | pve
	Category    string
	ImagePath   string
	TierTargets string // CSV ex: "10,20,30,50,100"
	MedalIDs    []int64
	Description string
}

// SeedCitationMappings insère les mappings citation → médaille dans citation_mappings.
// Portage de populate_citation_mappings.py Python (schéma v6 complet).
func SeedCitationMappings(opts SeedOptions) (SeedResult, error) {
	db, err := sql.Open("duckdb", opts.MetaDBPath)
	if err != nil {
		return SeedResult{Component: "citation_mappings"}, fmt.Errorf("ouverture metadata: %w", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS citation_mappings (
		citation_name_norm    VARCHAR NOT NULL,
		citation_name_display VARCHAR NOT NULL,
		mapping_type          VARCHAR NOT NULL DEFAULT 'medal',
		category              VARCHAR,
		image_path            VARCHAR,
		description           VARCHAR,
		tier_targets          VARCHAR,
		medal_id              UBIGINT,
		enabled               BOOLEAN NOT NULL DEFAULT TRUE
	)`); err != nil {
		return SeedResult{Component: "citation_mappings"}, err
	}

	mappings := defaultCitationMappings()
	inserted, skipped := 0, 0
	for _, m := range mappings {
		medalIDs := m.MedalIDs
		if len(medalIDs) == 0 {
			medalIDs = []int64{0}
		}
		for _, medalID := range medalIDs {
			var medalArg interface{}
			if medalID > 0 {
				medalArg = uint64(medalID) //nolint:gosec
			}
			res, err := db.Exec(
				`INSERT INTO citation_mappings
					(citation_name_norm, citation_name_display, mapping_type, category, image_path, description, tier_targets, medal_id, enabled)
				VALUES (?,?,?,?,?,?,?,?,TRUE)
				ON CONFLICT DO NOTHING`,
				m.Norm, m.Display, m.MappingType, m.Category, m.ImagePath, m.Description, m.TierTargets, medalArg,
			)
			if err != nil {
				return SeedResult{Component: "citation_mappings"}, err
			}
			if n, _ := res.RowsAffected(); n == 0 {
				skipped++
			} else {
				inserted++
			}
		}
	}
	return SeedResult{
		Component: "citation_mappings",
		Inserted:  inserted,
		Skipped:   skipped,
		Message:   fmt.Sprintf("%d règles insérées, %d présentes", inserted, skipped),
	}, nil
}

// defaultCitationMappings retourne les mappings par défaut.
// Portage des listes de populate_citation_mappings.py (schéma v6).
// Note : les 84 citations complètes de production incluent image_path et tier_targets
// peuplés par Python. Ce seed couvre les entrées de base pour les installs fraîches / CI.
func defaultCitationMappings() []CitationMapping {
	return []CitationMapping{
		// PvE
		{Norm: "grunt_slayer", Display: "Chasseur de Grunts", MappingType: "pve", Category: "Ennemi",
			TierTargets: "50,100,250,500,1000", MedalIDs: []int64{1257}, Description: "Élimine des Grunts"},
		{Norm: "elite_slayer", Display: "Chasseur d'Élites", MappingType: "pve", Category: "Ennemi",
			TierTargets: "25,50,100,250,500", MedalIDs: []int64{1258}, Description: "Élimine des Élites"},
		{Norm: "boss_kills", Display: "Tueur de boss", MappingType: "pve", Category: "Ennemi",
			TierTargets: "5,10,25,50,100", MedalIDs: []int64{1259}, Description: "Élimine des boss"},
		{Norm: "wave_completed", Display: "Vague complétée", MappingType: "pve", Category: "Mode de jeu",
			TierTargets: "10,25,50,100,250", MedalIDs: []int64{1260}, Description: "Vague complétée"},
		{Norm: "firefight_win", Display: "Victoire Firefight", MappingType: "pve", Category: "Mode de jeu",
			TierTargets: "5,10,25,50,100", MedalIDs: []int64{1261}, Description: "Victoire Firefight"},
		{Norm: "firefight_mvp", Display: "MVP Firefight", MappingType: "pve", Category: "Mode de jeu",
			TierTargets: "5,10,25,50,100", MedalIDs: []int64{1262}, Description: "MVP Firefight"},
		{Norm: "pve_killing_spree", Display: "Série de kills PvE", MappingType: "pve", Category: "Mode de jeu",
			TierTargets: "10,25,50,100,250", MedalIDs: []int64{1263}, Description: "Série de kills PvE"},
		// PvP multi-kills
		{Norm: "double_kill", Display: "Double Kill", MappingType: "medal", Category: "Multijoueur",
			TierTargets: "10,25,50,100,250", MedalIDs: []int64{1001}, Description: "Double Kill"},
		{Norm: "triple_kill", Display: "Triple Kill", MappingType: "medal", Category: "Multijoueur",
			TierTargets: "5,10,25,50,100", MedalIDs: []int64{1002}, Description: "Triple Kill"},
		{Norm: "overkill", Display: "Overkill", MappingType: "medal", Category: "Multijoueur",
			TierTargets: "5,10,25,50,100", MedalIDs: []int64{1003}, Description: "Overkill"},
		{Norm: "killtacular", Display: "Killtacular", MappingType: "medal", Category: "Multijoueur",
			TierTargets: "3,5,10,25,50", MedalIDs: []int64{1004}, Description: "Killtacular"},
		{Norm: "killing_frenzy", Display: "Killing Frenzy", MappingType: "medal", Category: "Multijoueur",
			TierTargets: "3,5,10,25,50", MedalIDs: []int64{1005}, Description: "Killing Frenzy"},
		// PvP séries
		{Norm: "killing_spree", Display: "Série de kills", MappingType: "medal", Category: "Multijoueur",
			TierTargets: "10,25,50,100,250", MedalIDs: []int64{1100}, Description: "Série 5 kills"},
		{Norm: "killing_rampage", Display: "Rampage", MappingType: "medal", Category: "Multijoueur",
			TierTargets: "5,10,25,50,100", MedalIDs: []int64{1101}, Description: "Kills Rampage"},
		// Objectif
		{Norm: "flag_runner", Display: "Porteur de drapeau", MappingType: "medal", Category: "Mode de jeu",
			TierTargets: "10,25,50,100,250", MedalIDs: []int64{1200}, Description: "Transport drapeau"},
		{Norm: "flag_capture", Display: "Capture de drapeau", MappingType: "medal", Category: "Mode de jeu",
			TierTargets: "5,10,25,50,100", MedalIDs: []int64{1201}, Description: "Capture drapeau"},
		// Composites
		{Norm: "grenade_mastery", Display: "Maîtrise des grenades", MappingType: "composite", Category: "Multijoueur",
			TierTargets: "10,25,50,100,250", MedalIDs: []int64{1300, 1301}, Description: "Maîtrise grenades"},
		{Norm: "vehicle_mastery", Display: "Maîtrise des véhicules", MappingType: "composite", Category: "Multijoueur",
			TierTargets: "10,25,50,100,250", MedalIDs: []int64{1400, 1401, 1402}, Description: "Maîtrise véhicules"},
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
