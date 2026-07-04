//go:build cgo

// diag_session_map_winrate — diagnostic de l'écart de "taux historique par carte"
// entre :
//   - le tableau Match History (cellule win_rate_hist)
//   - le chart Squad/Synergies "Performance par carte — Session vs Historique"
//
// Pour chaque session repérée (par défaut 6 avril toutes années), affiche
// par carte trois winrates calculés sur l'historique du joueur principal :
//
//	[A] tableau MH      : tous les matchs (équivalent computeMapWinRates)
//	[B] chart Synergies : matchs avec is_with_friends=TRUE
//	                      (équivalent computeHistoricalMapWRByLabel)
//	[C] squad sélectionné : matchs avec exactement les xuids du squad de la session
//	                        (= ce que les libellés UI prétendent montrer)
//
// Usage : go run -tags cgo ./cmd/diag_session_map_winrate [MM-DD] [gamertag]
//
//	MM-DD     : filtre date (default "04-06")
//	gamertag  : facultatif, restreint à un joueur
package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

const tz = "Europe/Paris"

const (
	sharedPathRel = "../../data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb"
	playersDirRel = "../../data/titles/halo_infinite/players"
)

func main() {
	mmdd := "04-06"
	pinned := ""
	if len(os.Args) > 1 {
		mmdd = os.Args[1]
	}
	if len(os.Args) > 2 {
		pinned = os.Args[2]
	}

	if _, err := os.Stat(sharedPathRel); err != nil {
		log.Fatalf("shared DB introuvable : %s — lance depuis apps/go-api/", sharedPathRel)
	}
	shared := openDuckDB(sharedPathRel)
	defer shared.Close()

	entries, err := os.ReadDir(playersDirRel)
	if err != nil {
		log.Fatalf("lecture playersDir: %v", err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		gt := e.Name()
		if pinned != "" && !strings.EqualFold(gt, pinned) {
			continue
		}
		statsPath := filepath.Join(playersDirRel, gt, "stats.duckdb")
		if _, err := os.Stat(statsPath); err != nil {
			continue
		}
		diagPlayer(shared, gt, statsPath, mmdd)
	}
}

type histRow struct {
	matchID       string
	mapID         string
	mapName       string
	outcome       int
	isWithFriends bool
}

func diagPlayer(shared *sql.DB, gt, statsPath, mmdd string) {
	pdb := openDuckDB(statsPath)
	defer pdb.Close()

	xuid := lookupXUID(shared, gt)
	if xuid == "" {
		fmt.Printf("\n[%s] xuid introuvable — skip\n", gt)
		return
	}

	// 1. Sessions contenant au moins un match du 6 avril (toutes années).
	sessions := findSessionsOnDate(shared, pdb, xuid, mmdd)
	if len(sessions) == 0 {
		return
	}

	fmt.Printf("\n========================================\n")
	fmt.Printf("[%s]  xuid=%s\n", gt, xuid)
	fmt.Printf("========================================\n")

	// 2. Charger l'historique COMPLET du main (registry + outcome + pme).
	hist := loadHistory(shared, pdb, xuid)
	fmt.Printf("Historique total du main : %d matchs (avec map)\n", len(hist))

	// 3. Pour chaque session, calculer le squad et les 3 winrates par carte.
	for _, s := range sessions {
		diagSession(shared, pdb, xuid, hist, s)
	}
}

type mapInfo struct {
	mapID, mapName string
}

func diagSession(shared, pdb *sql.DB, mainXUID string, hist []histRow, label string) {
	matches := matchesOfSession(pdb, label)
	if len(matches) == 0 {
		return
	}

	// Cartes uniques de la session
	maps := mapsOfMatches(shared, matches)

	// Squad = intersection des xuids présents dans la même team que le main
	// sur tous les matchs de la session (hors main).
	squadX, squadGT := computeSquad(shared, mainXUID, matches)

	fmt.Printf("\n--- Session %q (%d matchs, %d cartes) ---\n", label, len(matches), len(maps))
	fmt.Printf("    Squad sélectionné (intersection) : %s\n", strings.Join(squadGT, ", "))

	// Pour chaque carte : 3 winrates
	fmt.Printf("    %-18s | %-18s | %-18s | %-18s\n", "carte", "[A] tableau MH", "[B] chart Synergies", "[C] squad sélection")
	fmt.Printf("    %s\n", strings.Repeat("-", 84))
	for _, m := range maps {
		a := winrateAll(hist, m.mapID)
		b := winrateWithFriends(hist, m.mapID)
		c := winrateWithSquad(shared, mainXUID, squadX, hist, m.mapID)
		fmt.Printf("    %-18s | %-18s | %-18s | %-18s\n",
			truncate(m.mapName, 18), fmt3(a), fmt3(b), fmt3(c))
	}
}

func fmt3(s stat) string {
	if s.total == 0 {
		return "n/a (0)"
	}
	pct := 100.0 * float64(s.wins) / float64(s.total)
	return fmt.Sprintf("%.1f%% (%d/%d)", pct, s.wins, s.total)
}

type stat struct{ wins, total int }

// [A] Tableau Match History : tous les matchs du main sur cette carte
func winrateAll(hist []histRow, mapID string) stat {
	var s stat
	for _, r := range hist {
		if r.mapID != mapID {
			continue
		}
		s.total++
		if r.outcome == 2 {
			s.wins++
		}
	}
	return s
}

// [B] Chart Synergies : matchs is_with_friends=TRUE
func winrateWithFriends(hist []histRow, mapID string) stat {
	var s stat
	for _, r := range hist {
		if r.mapID != mapID {
			continue
		}
		if !r.isWithFriends {
			continue
		}
		s.total++
		if r.outcome == 2 {
			s.wins++
		}
	}
	return s
}

// [C] Squad sélectionné : matchs avec EXACTEMENT le squad présent côté participants.
//
//	Si squad vide → renvoie n/a.
func winrateWithSquad(shared *sql.DB, mainXUID string, squadXUIDs []string, hist []histRow, mapID string) stat {
	if len(squadXUIDs) == 0 {
		return stat{}
	}
	// Pour chaque match du main sur cette carte, vérifier que TOUS les squadXUIDs
	// sont dans les participants.
	var s stat
	for _, r := range hist {
		if r.mapID != mapID {
			continue
		}
		if matchHasAllXUIDs(shared, r.matchID, squadXUIDs) {
			s.total++
			if r.outcome == 2 {
				s.wins++
			}
		}
	}
	return s
}

func matchHasAllXUIDs(shared *sql.DB, matchID string, xuids []string) bool {
	if len(xuids) == 0 {
		return false
	}
	placeholders := strings.Repeat("?,", len(xuids))
	placeholders = placeholders[:len(placeholders)-1]
	q := fmt.Sprintf(
		`SELECT COUNT(DISTINCT xuid) FROM match_participants WHERE match_id = ? AND xuid IN (%s)`,
		placeholders,
	)
	args := make([]any, 0, len(xuids)+1)
	args = append(args, matchID)
	for _, x := range xuids {
		args = append(args, x)
	}
	var n int
	if err := shared.QueryRow(q, args...).Scan(&n); err != nil {
		log.Fatalf("matchHasAllXUIDs: %v", err)
	}
	return n == len(xuids)
}

// ----------------------------------------------------------------------------
// Loaders
// ----------------------------------------------------------------------------

func lookupXUID(shared *sql.DB, gt string) string {
	var x sql.NullString
	err := shared.QueryRow(
		"SELECT xuid FROM xuid_aliases WHERE LOWER(gamertag) = LOWER(?) ORDER BY last_seen DESC NULLS LAST LIMIT 1",
		gt,
	).Scan(&x)
	if err == sql.ErrNoRows {
		return ""
	}
	if err != nil {
		log.Fatalf("lookupXUID: %v", err)
	}
	return x.String
}

func findSessionsOnDate(shared, pdb *sql.DB, xuid, mmdd string) []string {
	// Étape 1 : matchs du main dont le start_time tombe sur MM-DD.
	q := `
		SELECT mp.match_id
		FROM match_participants mp
		JOIN match_registry r ON r.match_id = mp.match_id
		WHERE mp.xuid = ?
		  AND strftime(` + analysis.SQLStartTimeCanonical("r") + `, '%m-%d') = ?
	`
	rows, err := shared.Query(q, xuid, mmdd)
	if err != nil {
		log.Fatalf("findSessionsOnDate registry: %v", err)
	}
	defer rows.Close()
	var matchIDs []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			log.Fatalf("scan: %v", err)
		}
		matchIDs = append(matchIDs, s)
	}
	if len(matchIDs) == 0 {
		return nil
	}

	// Étape 2 : récupérer les session_label correspondants côté pme.
	placeholders := strings.Repeat("?,", len(matchIDs))
	placeholders = placeholders[:len(placeholders)-1]
	q2 := fmt.Sprintf(
		`SELECT DISTINCT COALESCE(session_label, '')
		 FROM player_match_enrichment
		 WHERE match_id IN (%s)`,
		placeholders,
	)
	args := make([]any, len(matchIDs))
	for i, m := range matchIDs {
		args[i] = m
	}
	rows2, err := pdb.Query(q2, args...)
	if err != nil {
		log.Fatalf("findSessionsOnDate pme: %v", err)
	}
	defer rows2.Close()
	var labels []string
	seen := map[string]bool{}
	for rows2.Next() {
		var l sql.NullString
		_ = rows2.Scan(&l)
		v := strings.TrimSpace(l.String)
		if v == "" {
			v = "(no session)"
		}
		if !seen[v] {
			seen[v] = true
			labels = append(labels, v)
		}
	}
	sort.Strings(labels)
	return labels
}

func matchesOfSession(pdb *sql.DB, label string) []string {
	var rows *sql.Rows
	var err error
	if label == "(no session)" {
		rows, err = pdb.Query(
			`SELECT match_id FROM player_match_enrichment WHERE session_label IS NULL OR session_label = ''`,
		)
	} else {
		rows, err = pdb.Query(
			`SELECT match_id FROM player_match_enrichment WHERE session_label = ?`,
			label,
		)
	}
	if err != nil {
		log.Fatalf("matchesOfSession: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		_ = rows.Scan(&s)
		out = append(out, s)
	}
	return out
}

func mapsOfMatches(shared *sql.DB, matchIDs []string) []mapInfo {
	if len(matchIDs) == 0 {
		return nil
	}
	placeholders := strings.Repeat("?,", len(matchIDs))
	placeholders = placeholders[:len(placeholders)-1]
	q := fmt.Sprintf(
		`SELECT DISTINCT COALESCE(map_id,''), COALESCE(map_name,'')
		 FROM match_registry
		 WHERE match_id IN (%s)
		 ORDER BY map_name`,
		placeholders,
	)
	args := make([]any, len(matchIDs))
	for i, m := range matchIDs {
		args[i] = m
	}
	rows, err := shared.Query(q, args...)
	if err != nil {
		log.Fatalf("mapsOfMatches: %v", err)
	}
	defer rows.Close()
	var out []mapInfo
	for rows.Next() {
		var mi mapInfo
		_ = rows.Scan(&mi.mapID, &mi.mapName)
		if mi.mapID == "" && mi.mapName == "" {
			continue
		}
		out = append(out, mi)
	}
	return out
}

func computeSquad(shared *sql.DB, mainXUID string, matches []string) (xuids, gts []string) {
	if len(matches) == 0 {
		return nil, nil
	}
	placeholders := strings.Repeat("?,", len(matches))
	placeholders = placeholders[:len(placeholders)-1]

	// 1. team du main par match
	qTeam := fmt.Sprintf(
		`SELECT match_id, team_id FROM match_participants
		 WHERE xuid = ? AND match_id IN (%s)`,
		placeholders,
	)
	args := make([]any, 0, len(matches)+1)
	args = append(args, mainXUID)
	for _, m := range matches {
		args = append(args, m)
	}
	rows, err := shared.Query(qTeam, args...)
	if err != nil {
		log.Fatalf("computeSquad team main: %v", err)
	}
	teamByMatch := map[string]int{}
	for rows.Next() {
		var mid string
		var t sql.NullInt64
		_ = rows.Scan(&mid, &t)
		teamByMatch[mid] = int(t.Int64)
	}
	rows.Close()

	// 2. coéquipiers présents par match (même team que le main, ≠ main)
	teammatesPerMatch := map[string]map[string]bool{}
	gtByXUID := map[string]string{}
	for mid, team := range teamByMatch {
		q := `SELECT mp.xuid, COALESCE(va.gamertag, mp.xuid)
			  FROM match_participants mp
			  LEFT JOIN xuid_aliases va ON va.xuid = mp.xuid
			  WHERE mp.match_id = ? AND mp.team_id = ? AND mp.xuid <> ?`
		r, err := shared.Query(q, mid, team, mainXUID)
		if err != nil {
			log.Fatalf("computeSquad teammates: %v", err)
		}
		set := map[string]bool{}
		for r.Next() {
			var x, g string
			_ = r.Scan(&x, &g)
			set[x] = true
			gtByXUID[x] = g
		}
		r.Close()
		teammatesPerMatch[mid] = set
	}

	// 3. intersection sur tous les matchs
	if len(teammatesPerMatch) == 0 {
		return nil, nil
	}
	var inter map[string]bool
	first := true
	for _, set := range teammatesPerMatch {
		if first {
			inter = make(map[string]bool, len(set))
			for k := range set {
				inter[k] = true
			}
			first = false
			continue
		}
		for k := range inter {
			if !set[k] {
				delete(inter, k)
			}
		}
	}

	for x := range inter {
		xuids = append(xuids, x)
		gts = append(gts, gtByXUID[x])
	}
	sort.Strings(xuids)
	sort.Strings(gts)
	return xuids, gts
}

func loadHistory(shared, pdb *sql.DB, xuid string) []histRow {
	q := `
		SELECT mp.match_id,
		       COALESCE(r.map_id, ''),
		       COALESCE(r.map_name, ''),
		       COALESCE(mp.outcome, 0)
		FROM match_participants mp
		JOIN match_registry r ON r.match_id = mp.match_id
		WHERE mp.xuid = ?
	`
	rows, err := shared.Query(q, xuid)
	if err != nil {
		log.Fatalf("loadHistory shared: %v", err)
	}
	defer rows.Close()
	var out []histRow
	matchIDs := []string{}
	for rows.Next() {
		var h histRow
		_ = rows.Scan(&h.matchID, &h.mapID, &h.mapName, &h.outcome)
		out = append(out, h)
		matchIDs = append(matchIDs, h.matchID)
	}

	// Charger pme.is_with_friends pour chacun
	wfBy := map[string]bool{}
	if len(matchIDs) > 0 {
		// chunked IN (DuckDB tolère beaucoup mais soyons sûrs)
		chunkSize := 500
		for i := 0; i < len(matchIDs); i += chunkSize {
			j := i + chunkSize
			if j > len(matchIDs) {
				j = len(matchIDs)
			}
			ch := matchIDs[i:j]
			placeholders := strings.Repeat("?,", len(ch))
			placeholders = placeholders[:len(placeholders)-1]
			q2 := fmt.Sprintf(
				`SELECT match_id, COALESCE(is_with_friends, FALSE)
				 FROM player_match_enrichment WHERE match_id IN (%s)`,
				placeholders,
			)
			args := make([]any, len(ch))
			for k, m := range ch {
				args[k] = m
			}
			r2, err := pdb.Query(q2, args...)
			if err != nil {
				log.Fatalf("loadHistory pme: %v", err)
			}
			for r2.Next() {
				var mid string
				var b bool
				_ = r2.Scan(&mid, &b)
				wfBy[mid] = b
			}
			r2.Close()
		}
	}
	for i := range out {
		out[i].isWithFriends = wfBy[out[i].matchID]
	}
	// Garde uniquement ceux avec un map valide
	filtered := out[:0:0]
	for _, h := range out {
		if h.mapID == "" && h.mapName == "" {
			continue
		}
		filtered = append(filtered, h)
	}
	return filtered
}

func openDuckDB(path string) *sql.DB {
	connector, err := duckdb.NewConnector(path+"?access_mode=READ_ONLY", func(execer driver.ExecerContext) error {
		_, e := execer.ExecContext(context.Background(), "SET TimeZone='"+tz+"'", nil)
		return e
	})
	if err != nil {
		log.Fatalf("connector(%s): %v", path, err)
	}
	return sql.OpenDB(connector)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "."
}
