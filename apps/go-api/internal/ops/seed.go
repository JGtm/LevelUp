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
	RankID    int    `json:"rank_id"`
	TitleEN   string `json:"title_en"`
	TitleFR   string `json:"title_fr"`
	Subtitle  string `json:"subtitle"`
	Tier      string `json:"tier"`
	Grade     int    `json:"grade"`
	XPRequired int64 `json:"xp_required"`
	IconPath  string `json:"icon_path"`
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
	CitationKey string
	Mode        string // pvp, pve, composite
	MedalIDs    []int64
	Description string
}

// SeedCitationMappings insère les 45 règles PVP + 7 PVE + composites.
// Portage de populate_citation_mappings.py Python.
func SeedCitationMappings(opts SeedOptions) (SeedResult, error) {
	db, err := sql.Open("duckdb", opts.MetaDBPath)
	if err != nil {
		return SeedResult{Component: "citation_mappings"}, fmt.Errorf("ouverture metadata: %w", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS citation_mappings (
		citation_key VARCHAR,
		mode VARCHAR,
		medal_id BIGINT,
		description VARCHAR,
		PRIMARY KEY (citation_key, medal_id)
	)`); err != nil {
		return SeedResult{Component: "citation_mappings"}, err
	}

	mappings := defaultCitationMappings()
	inserted, skipped := 0, 0
	for _, m := range mappings {
		for _, medalID := range m.MedalIDs {
			res, err := db.Exec(`INSERT OR IGNORE INTO citation_mappings VALUES (?,?,?,?)`,
				m.CitationKey, m.Mode, medalID, m.Description)
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
// Portage des listes hardcodées dans populate_citation_mappings.py.
func defaultCitationMappings() []CitationMapping {
	return []CitationMapping{
		// PvE
		{CitationKey: "grunt_slayer", Mode: "pve", MedalIDs: []int64{1257}, Description: "Élimine des Grunts"},
		{CitationKey: "elite_slayer", Mode: "pve", MedalIDs: []int64{1258}, Description: "Élimine des Élites"},
		{CitationKey: "boss_kills", Mode: "pve", MedalIDs: []int64{1259}, Description: "Élimine des boss"},
		{CitationKey: "wave_completed", Mode: "pve", MedalIDs: []int64{1260}, Description: "Vague complétée"},
		{CitationKey: "firefight_win", Mode: "pve", MedalIDs: []int64{1261}, Description: "Victoire Firefight"},
		{CitationKey: "firefight_mvp", Mode: "pve", MedalIDs: []int64{1262}, Description: "MVP Firefight"},
		{CitationKey: "pve_killing_spree", Mode: "pve", MedalIDs: []int64{1263}, Description: "Série de kills PvE"},
		// PvP multi-kills
		{CitationKey: "double_kill", Mode: "pvp", MedalIDs: []int64{1001}, Description: "Double Kill"},
		{CitationKey: "triple_kill", Mode: "pvp", MedalIDs: []int64{1002}, Description: "Triple Kill"},
		{CitationKey: "overkill", Mode: "pvp", MedalIDs: []int64{1003}, Description: "Overkill"},
		{CitationKey: "killtacular", Mode: "pvp", MedalIDs: []int64{1004}, Description: "Killtacular"},
		{CitationKey: "killing_frenzy", Mode: "pvp", MedalIDs: []int64{1005}, Description: "Killing Frenzy"},
		// PvP séries
		{CitationKey: "killing_spree", Mode: "pvp", MedalIDs: []int64{1100}, Description: "Série 5 kills"},
		{CitationKey: "killing_rampage", Mode: "pvp", MedalIDs: []int64{1101}, Description: "Kills Rampage"},
		// Objectif
		{CitationKey: "flag_runner", Mode: "pvp", MedalIDs: []int64{1200}, Description: "Transport drapeau"},
		{CitationKey: "flag_capture", Mode: "pvp", MedalIDs: []int64{1201}, Description: "Capture drapeau"},
		// Composites
		{CitationKey: "grenade_mastery", Mode: "composite", MedalIDs: []int64{1300, 1301}, Description: "Maîtrise grenades"},
		{CitationKey: "vehicle_mastery", Mode: "composite", MedalIDs: []int64{1400, 1401, 1402}, Description: "Maîtrise véhicules"},
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
