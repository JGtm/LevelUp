// Package sync — pve.go : extraction et insertion des stats PvE Firefight.
//
// Portage de :
//   - src/data/sync/models.py         → PveMatchStatsRow
//   - src/data/sync/transformers.py   → ExtractPveStats, findPveStatsDict
//   - src/data/sync/batch_insert.py   → InsertPveStats
//   - src/data/sync/engine.py         → MarkPveStatsDone
//
// La détection is_firefight est faite dans transforms.go (isFirefightMatch).
// Le bitmask MBitPVEStats est dans backfill_flags.go.
package sync

import (
	"database/sql"
	"fmt"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// Modèle
// ─────────────────────────────────────────────────────────────────────────────

// PveMatchStatsRow représente une ligne de pve_match_stats (shared_pve.duckdb).
// Portage de PveMatchStatsRow dataclass (Python models.py).
type PveMatchStatsRow struct {
	MatchID        string
	XUID           string
	WavesCompleted int
	BossKills      int
	GruntKills     int
	EliteKills     int
	JackalKills    int
	BruteKills     int
	HunterKills    int
	SkimmerKills   int
	CrawlerKills   int
	SoldierKills   int
	KnightKills    int
	WardenKills    int
	SentinelKills  int
	MarineKills    int
	TotalKills     int
	Deaths         int
	DamageDealt    float64
	PveBitsValue   int64
}

// ─────────────────────────────────────────────────────────────────────────────
// Extraction API → struct
// ─────────────────────────────────────────────────────────────────────────────

// ExtractPveStats extrait les stats PvE depuis le JSON d'un match Halo Infinite.
// Portage de extract_pve_stats() Python (transformers.py ~150 lignes).
//
// Supporte deux layouts API selon la saison :
//   - EnemyKillsByType dict (v1)
//   - Champs directs GruntKills, EliteKills… (v2)
func ExtractPveStats(matchID string, matchJSON map[string]any) []PveMatchStatsRow {
	players, _ := matchJSON["Players"].([]any)
	if len(players) == 0 {
		return nil
	}
	var rows []PveMatchStatsRow
	for _, p := range players {
		playerMap, ok := p.(map[string]any)
		if !ok {
			continue
		}
		xuid := extractPlayerXUID(playerMap)
		if xuid == "" {
			continue
		}
		pveDict := findPveStatsDict(playerMap)
		if pveDict == nil {
			continue
		}
		rows = append(rows, buildPveRow(matchID, xuid, playerMap, pveDict))
	}
	return rows
}

// findPveStatsDict cherche le sous-dict contenant les stats PvE.
// L'API Halo peut retourner : FirefightStats | PveStats | EliminationStats
// ou les champs directement dans PlayerStats.
func findPveStatsDict(playerMap map[string]any) map[string]any {
	pts, _ := playerMap["PlayerTeamStats"].([]any)
	if len(pts) == 0 {
		return nil
	}
	first, ok := pts[0].(map[string]any)
	if !ok {
		return nil
	}
	ps, _ := first["PlayerStats"].(map[string]any)
	if ps == nil {
		return nil
	}
	for _, key := range []string{"FirefightStats", "PveStats", "EliminationStats"} {
		if sub, ok := ps[key].(map[string]any); ok {
			return sub
		}
	}
	// Les champs sont parfois inlinés directement dans PlayerStats
	if _, found := ps["GruntKills"]; found {
		return ps
	}
	return nil
}

// extractPlayerXUID extrait le XUID depuis une map joueur (vs extractXUID(string)).
func extractPlayerXUID(playerMap map[string]any) string {
	// Format : Xuid ou PlayerId = "xuid(1234567890)"
	for _, key := range []string{"Xuid", "PlayerId"} {
		if raw, ok := playerMap[key].(string); ok && raw != "" {
			return cleanXUID(raw)
		}
	}
	// Fallback : PlayerProperties[0].PlayerId
	props, _ := playerMap["PlayerProperties"].([]any)
	if len(props) > 0 {
		if first, ok := props[0].(map[string]any); ok {
			if id, ok := first["PlayerId"].(string); ok {
				return cleanXUID(id)
			}
		}
	}
	return ""
}

// cleanXUID retire le préfixe/suffixe "xuid(…)".
func cleanXUID(raw string) string {
	raw = strings.TrimPrefix(raw, "xuid(")
	raw = strings.TrimSuffix(raw, ")")
	return raw
}

// buildPveRow construit une PveMatchStatsRow depuis les dicts API.
func buildPveRow(matchID, xuid string, playerMap, pveDict map[string]any) PveMatchStatsRow {
	row := PveMatchStatsRow{
		MatchID:        matchID,
		XUID:           xuid,
		WavesCompleted: intFrom(pveDict, "WavesCompleted"),
		BossKills:      intFrom(pveDict, "BossKills"),
		Deaths:         intFrom(playerMap, "Deaths"),
		DamageDealt:    float64From(playerMap, "TotalDamageDealt"),
	}
	fillEnemyKills(&row, pveDict)
	row.TotalKills = row.GruntKills + row.EliteKills + row.JackalKills +
		row.BruteKills + row.HunterKills + row.SkimmerKills +
		row.CrawlerKills + row.SoldierKills + row.KnightKills +
		row.WardenKills + row.SentinelKills + row.MarineKills
	row.PveBitsValue = computePveBits(row)
	return row
}

// fillEnemyKills remplit les kills par type d'ennemi (2 layouts API).
func fillEnemyKills(row *PveMatchStatsRow, pveDict map[string]any) {
	if ebt, ok := pveDict["EnemyKillsByType"].(map[string]any); ok {
		row.GruntKills = intFrom(ebt, "Grunt")
		row.EliteKills = intFrom(ebt, "Elite")
		row.JackalKills = intFrom(ebt, "Jackal")
		row.BruteKills = intFrom(ebt, "Brute")
		row.HunterKills = intFrom(ebt, "Hunter")
		row.SkimmerKills = intFrom(ebt, "Skimmer")
		row.CrawlerKills = intFrom(ebt, "Crawler")
		row.SoldierKills = intFrom(ebt, "Soldier")
		row.KnightKills = intFrom(ebt, "Knight")
		row.WardenKills = intFrom(ebt, "Warden")
		row.SentinelKills = intFrom(ebt, "Sentinel")
		row.MarineKills = intFrom(ebt, "Marine")
		return
	}
	// Champs directs
	row.GruntKills = intFrom(pveDict, "GruntKills")
	row.EliteKills = intFrom(pveDict, "EliteKills")
	row.JackalKills = intFrom(pveDict, "JackalKills")
	row.BruteKills = intFrom(pveDict, "BruteKills")
	row.HunterKills = intFrom(pveDict, "HunterKills")
	row.SkimmerKills = intFrom(pveDict, "SkimmerKills")
	row.CrawlerKills = intFrom(pveDict, "CrawlerKills")
	row.SoldierKills = intFrom(pveDict, "SoldierKills")
	row.KnightKills = intFrom(pveDict, "KnightKills")
	row.WardenKills = intFrom(pveDict, "WardenKills")
	row.SentinelKills = intFrom(pveDict, "SentinelKills")
	row.MarineKills = intFrom(pveDict, "MarineKills")
}

// computePveBits calcule le bitmask PveBits pour une ligne PvE.
func computePveBits(row PveMatchStatsRow) int64 {
	var bits int64
	if row.TotalKills > 0 {
		bits |= PveBitTotalKills
	}
	if row.BossKills > 0 {
		bits |= PveBitBossKills
	}
	if row.GruntKills > 0 {
		bits |= PveBitGrunt
	}
	if row.EliteKills > 0 {
		bits |= PveBitElite
	}
	if row.JackalKills > 0 {
		bits |= PveBitJackal
	}
	if row.BruteKills > 0 {
		bits |= PveBitBrute
	}
	if row.HunterKills > 0 {
		bits |= PveBitHunter
	}
	if row.SkimmerKills > 0 {
		bits |= PveBitSkimmer
	}
	if row.CrawlerKills > 0 {
		bits |= PveBitCrawler
	}
	if row.SoldierKills > 0 {
		bits |= PveBitSoldier
	}
	if row.KnightKills > 0 {
		bits |= PveBitKnight
	}
	if row.WardenKills > 0 {
		bits |= PveBitWarden
	}
	if row.SentinelKills > 0 {
		bits |= PveBitSentinel
	}
	if row.MarineKills > 0 {
		bits |= PveBitMarine
	}
	return bits
}

// float64From lit un float64 depuis une map JSON (les nombres JSON sont float64).
func float64From(m map[string]any, key string) float64 {
	v, _ := m[key].(float64)
	return v
}

// ─────────────────────────────────────────────────────────────────────────────
// Insertion DuckDB
// ─────────────────────────────────────────────────────────────────────────────

// InsertPveStats insère (INSERT OR REPLACE) des lignes dans pve_match_stats.
// Portage de batch_insert_pve_stats() Python (batch_insert.py).
func InsertPveStats(db *sql.DB, rows []PveMatchStatsRow) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	const q = `
		INSERT OR REPLACE INTO pve_match_stats (
			match_id, xuid,
			waves_completed, boss_kills,
			grunt_kills, elite_kills, jackal_kills, brute_kills,
			hunter_kills, skimmer_kills, crawler_kills, soldier_kills,
			knight_kills, warden_kills, sentinel_kills, marine_kills,
			total_kills, deaths, damage_dealt, pve_bits
		) VALUES (
			?, ?,
			?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?
		)`
	inserted := 0
	for _, r := range rows {
		if _, err := db.Exec(q,
			r.MatchID, r.XUID,
			r.WavesCompleted, r.BossKills,
			r.GruntKills, r.EliteKills, r.JackalKills, r.BruteKills,
			r.HunterKills, r.SkimmerKills, r.CrawlerKills, r.SoldierKills,
			r.KnightKills, r.WardenKills, r.SentinelKills, r.MarineKills,
			r.TotalKills, r.Deaths, r.DamageDealt, r.PveBitsValue,
		); err != nil {
			return inserted, fmt.Errorf("insert pve_match_stats %s/%s: %w", r.MatchID, r.XUID, err)
		}
		inserted++
	}
	return inserted, nil
}

// MarkPveStatsDone marque MBitPVEStats dans match_registry.backfill_completed.
// Portage de _try_insert_pve_stats() Python (engine.py).
func MarkPveStatsDone(sharedDB *sql.DB, matchID string) error {
	_, err := sharedDB.Exec(`
		UPDATE match_registry
		SET backfill_completed = COALESCE(backfill_completed, 0) | ?
		WHERE match_id = ?
	`, MBitPVEStats, matchID)
	return err
}
