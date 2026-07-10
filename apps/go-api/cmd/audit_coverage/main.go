// audit_coverage — outil de diagnostic exhaustif pour la couverture des données
// par match (shared) et par (player, match) (player DBs).
//
// Pour chaque match dans match_registry, vérifie les tables critiques.
// IMPORTANT : doit être lancé serveur arrêté (DuckDB lock fichier sur Windows).
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"levelup/go-api/internal/analysis"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	MBitWeaponKills       uint64 = 1 << 21
	MBitWeaponKillsNoFilm uint64 = 1 << 22
)

var (
	dataRoot  = flag.String("data", `../../data`, "Racine du dossier data/")
	titleSlug = flag.String("title", "halo_infinite", "Title slug")
	limit     = flag.Int("limit", 25, "Nombre de matchs récents à afficher en détail")
)

type matchRow struct {
	matchID         string
	startTime       time.Time
	durationDays    float64
	isFirefight     bool
	bitmask         uint64
	participants    int
	weaponKills     int
	highlightEvents int
	medals          int
	kvp             int
}

func main() {
	flag.Parse()

	sharedPath := filepath.Join(*dataRoot, "titles", *titleSlug, "warehouse", "shared_matches_v2.duckdb")
	shared, err := sql.Open("duckdb", sharedPath+"?access_mode=read_only")
	if err != nil {
		log.Fatal("open shared:", err)
	}
	defer shared.Close()

	matches, err := loadMatches(shared)
	if err != nil {
		log.Fatal("loadMatches:", err)
	}
	fmt.Printf("Total matchs dans match_registry : %d\n\n", len(matches))

	printGlobalSynthesis(matches)

	fmt.Printf("\n=== Détail des %d matchs les plus récents ===\n", *limit)
	printRecentDetails(matches, *limit)

	printActionable(matches)

	playersDir := filepath.Join(*dataRoot, "titles", *titleSlug, "players")
	entries, err := os.ReadDir(playersDir)
	if err != nil {
		log.Printf("readdir players: %v", err)
		return
	}
	fmt.Println("\n=== Audit player-side ===")
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		auditPlayer(filepath.Join(playersDir, e.Name(), "stats.duckdb"), e.Name(), shared)
	}
}

func loadMatches(db *sql.DB) ([]matchRow, error) {
	q := `
		SELECT r.match_id,
		       ` + analysis.SQLStartTimeCanonical("r") + ` AS start_time,
		       COALESCE(r.is_firefight, FALSE) AS is_firefight,
		       COALESCE(r.backfill_completed, 0) AS bitmask,
		       (SELECT COUNT(*) FROM match_participants p WHERE p.match_id = r.match_id) AS participants,
		       (SELECT COUNT(*) FROM weapon_kills wk WHERE wk.match_id = r.match_id) AS wk,
		       (SELECT COUNT(*) FROM highlight_events hv WHERE hv.match_id = r.match_id) AS hev,
		       (SELECT COUNT(*) FROM medals_earned m WHERE m.match_id = r.match_id) AS medals,
		       (SELECT COUNT(*) FROM killer_victim_pairs kvp WHERE kvp.match_id = r.match_id) AS kvp
		FROM match_registry r
		ORDER BY start_time DESC NULLS LAST`
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	now := time.Now()
	var out []matchRow
	for rows.Next() {
		var m matchRow
		var st sql.NullTime
		var bm sql.NullInt64
		if err := rows.Scan(&m.matchID, &st, &m.isFirefight, &bm, &m.participants,
			&m.weaponKills, &m.highlightEvents, &m.medals, &m.kvp); err != nil {
			return nil, err
		}
		if st.Valid {
			m.startTime = st.Time
			m.durationDays = now.Sub(st.Time).Hours() / 24
		}
		if bm.Valid {
			m.bitmask = uint64(bm.Int64)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func printGlobalSynthesis(matches []matchRow) {
	total := len(matches)
	if total == 0 {
		return
	}
	var participantsOK, wkOK, wkNoFilm, hevOK, medalsOK, kvpOK int
	for _, m := range matches {
		if m.participants > 0 {
			participantsOK++
		}
		if m.weaponKills > 0 {
			wkOK++
		}
		if m.bitmask&MBitWeaponKillsNoFilm != 0 {
			wkNoFilm++
		}
		if m.highlightEvents > 0 {
			hevOK++
		}
		if m.medals > 0 {
			medalsOK++
		}
		if m.kvp > 0 {
			kvpOK++
		}
	}
	fmt.Println("=== Synthèse globale ===")
	fmt.Printf("  match_participants  : %4d/%d  (%5.1f%%)\n", participantsOK, total, pct(participantsOK, total))
	fmt.Printf("  weapon_kills        : %4d/%d  (%5.1f%%)  + %d marqués 'film expiré'\n", wkOK, total, pct(wkOK, total), wkNoFilm)
	fmt.Printf("  highlight_events    : %4d/%d  (%5.1f%%)\n", hevOK, total, pct(hevOK, total))
	fmt.Printf("  medals_earned       : %4d/%d  (%5.1f%%)\n", medalsOK, total, pct(medalsOK, total))
	fmt.Printf("  killer_victim_pairs : %4d/%d  (%5.1f%%)\n", kvpOK, total, pct(kvpOK, total))
}

func pct(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(n) * 100 / float64(total)
}

func printRecentDetails(matches []matchRow, n int) {
	if n > len(matches) {
		n = len(matches)
	}
	fmt.Printf("%-10s  %-37s  %-3s  %-4s  %-3s  %-6s  %-4s  %-4s  %-4s\n",
		"Date", "MatchID", "PVE", "part", "wk", "wkNoF", "hev", "med", "kvp")
	for _, m := range matches[:n] {
		date := "<nil>"
		if !m.startTime.IsZero() {
			date = m.startTime.Format("2006-01-02")
		}
		wkNoFilm := "no"
		if m.bitmask&MBitWeaponKillsNoFilm != 0 {
			wkNoFilm = "YES"
		}
		fmt.Printf("%-10s  %-37s  %-3v  %-4d  %-3d  %-6s  %-4d  %-4d  %-4d\n",
			date, m.matchID, m.isFirefight, m.participants, m.weaponKills, wkNoFilm,
			m.highlightEvents, m.medals, m.kvp)
	}
}

func printActionable(matches []matchRow) {
	const filmExpiryDays = 28.0

	var actionableWK, actionableHEV, actionableKVP, actionableMedals []string
	for _, m := range matches {
		if m.participants == 0 {
			continue
		}
		isFresh := m.durationDays >= 0 && m.durationDays <= filmExpiryDays
		if m.medals == 0 {
			actionableMedals = append(actionableMedals, m.matchID)
		}
		if m.weaponKills == 0 && m.bitmask&MBitWeaponKillsNoFilm == 0 && isFresh {
			actionableWK = append(actionableWK, m.matchID)
		}
		if m.highlightEvents == 0 && isFresh {
			actionableHEV = append(actionableHEV, m.matchID)
		}
		if m.kvp == 0 && isFresh {
			actionableKVP = append(actionableKVP, m.matchID)
		}
	}

	fmt.Println("\n=== Actionable (backfill recommandé) ===")
	fmt.Printf("  medals_earned       : %d match(s) sans médailles\n", len(actionableMedals))
	fmt.Printf("  weapon_kills        : %d match(s) frais (<28j) sans wk et film non marqué expiré\n", len(actionableWK))
	fmt.Printf("  highlight_events    : %d match(s) frais (<28j) sans hev\n", len(actionableHEV))
	fmt.Printf("  killer_victim_pairs : %d match(s) frais (<28j) sans kvp\n", len(actionableKVP))

	if len(actionableWK) > 0 && len(actionableWK) <= 30 {
		fmt.Println("\n  Match IDs à backfill weapons :")
		for _, id := range actionableWK {
			fmt.Printf("    %s\n", id)
		}
	}
}

func auditPlayer(path, name string, shared *sql.DB) {
	pdb, err := sql.Open("duckdb", path+"?access_mode=read_only")
	if err != nil {
		log.Printf("  [%s] open: %v", name, err)
		return
	}
	defer pdb.Close()

	var xuid string
	row := pdb.QueryRow(`SELECT value FROM sync_meta WHERE key = 'player_xuid' LIMIT 1`)
	if err := row.Scan(&xuid); err != nil {
		row2 := shared.QueryRow(`SELECT xuid FROM xuid_aliases WHERE gamertag = ? LIMIT 1`, name)
		_ = row2.Scan(&xuid)
	}

	var participated int
	if xuid != "" {
		_ = shared.QueryRow(`SELECT COUNT(DISTINCT match_id) FROM match_participants WHERE xuid = ?`, xuid).Scan(&participated)
	}

	tables := []string{"match_skill_rank", "match_citations", "personal_score_awards", "player_match_enrichment"}
	counts := map[string]int{}
	for _, t := range tables {
		var exists int
		_ = pdb.QueryRow(`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?`, t).Scan(&exists)
		if exists == 0 {
			counts[t] = -1
			continue
		}
		var n int
		_ = pdb.QueryRow(fmt.Sprintf(`SELECT COUNT(DISTINCT match_id) FROM %s`, t)).Scan(&n)
		counts[t] = n
	}

	fmt.Printf("\n  [%s] xuid=%s — %d matchs participés\n", name, xuid, participated)
	for _, t := range tables {
		c := counts[t]
		if c < 0 {
			fmt.Printf("    %-25s : table absente\n", t)
		} else {
			gap := participated - c
			fmt.Printf("    %-25s : %4d distinct match_id (gap %d, %.1f%% couvert)\n",
				t, c, gap, pct(c, participated))
		}
	}

	if xuid != "" && counts["personal_score_awards"] >= 0 && counts["player_match_enrichment"] >= 0 {
		var missing []string
		rows, err := shared.Query(`
			SELECT DISTINCT mp.match_id, mr.start_time
			FROM match_participants mp
			LEFT JOIN match_registry mr ON mp.match_id = mr.match_id
			WHERE mp.xuid = ?
			ORDER BY mr.start_time DESC NULLS LAST
			LIMIT 50`, xuid)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var mid string
				var ts sql.NullTime
				_ = rows.Scan(&mid, &ts)
				var hasPSA, hasEnrich, hasCit, hasRank int
				_ = pdb.QueryRow(`SELECT COUNT(*) FROM personal_score_awards WHERE match_id = ?`, mid).Scan(&hasPSA)
				_ = pdb.QueryRow(`SELECT COUNT(*) FROM player_match_enrichment WHERE match_id = ?`, mid).Scan(&hasEnrich)
				_ = pdb.QueryRow(`SELECT COUNT(*) FROM match_citations WHERE match_id = ?`, mid).Scan(&hasCit)
				_ = pdb.QueryRow(`SELECT COUNT(*) FROM match_skill_rank WHERE match_id = ?`, mid).Scan(&hasRank)
				if hasPSA == 0 || hasEnrich == 0 {
					missing = append(missing, fmt.Sprintf("    %s  PSA=%d enrich=%d cit=%d rank=%d", mid, hasPSA, hasEnrich, hasCit, hasRank))
				}
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			fmt.Printf("    Sample missing (sur 50 derniers matchs) — %d trous :\n", len(missing))
			max := 5
			if max > len(missing) {
				max = len(missing)
			}
			for _, line := range missing[:max] {
				fmt.Println(line)
			}
		}
	}
}
