//go:build cgo

// diag_lusr_player — comparaison LUSR ancien (carry-adj) vs proposé (sans).
//
// Replay TrueSkill séquentiel pour chaque joueur en deux modes :
//   - "old" : score kills_vs_expected compressé par carry-adj (prod actuelle)
//   - "new" : score kills_vs_expected nu (sigmoidRatio sans carry-adj)
//
// Identifie ensuite les N derniers matchs où les 3 joueurs ont joué ensemble
// et produit un tableau comparatif (KE, score brut, composite, delta mu, mu).
//
// Read-only sur shared_matches_v2.duckdb — compatible serveur tournant.
//
// Usage : go run -tags cgo ./cmd/diag_lusr_player [-n 15] gt1 gt2 gt3
package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"flag"
	"fmt"
	"log"
	"sort"
	"strings"

	"levelup/go-api/internal/games/halo_infinite/skillchain"
	lusync "levelup/go-api/internal/sync"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

const sharedDBPath = "data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb"

func main() {
	lusync.SetLUSRChainClassifier(skillchain.ClassifyLUSRChain)        // MT-15 (fail-loud)
	lusync.SetObjectiveFamilyClassifier(skillchain.IsObjectiveSubMode) // famille de la chaîne de perf classée

	nMatches := flag.Int("n", 15, "nombre de matchs communs récents à afficher")
	verbose := flag.Bool("v", false, "affiche le breakdown des 8 composantes du composite")
	flag.Parse()
	gamertags := flag.Args()
	if len(gamertags) == 0 {
		gamertags = []string{"Chocoboflor", "Madina97294", "JGtm"}
	}

	db := openShared()
	defer db.Close()

	xuidByGT := resolveXUIDs(db, gamertags)
	printAliasReport(db, gamertags, xuidByGT)
	printMatchParticipantsKind(db)
	players := make([]playerData, 0, len(gamertags))
	allMatchIDs := map[string]bool{}

	for _, gt := range gamertags {
		xuid := xuidByGT[strings.ToLower(gt)]
		if xuid == "" {
			fmt.Printf("⚠ gamertag %q : xuid introuvable, ignoré.\n", gt)
			continue
		}
		matches := loadMatches(db, xuid)
		if len(matches) == 0 {
			fmt.Printf("⚠ %s (xuid=%s) : 0 match LUSR-éligible.\n", gt, xuid)
			continue
		}
		ids := make([]string, len(matches))
		for i, m := range matches {
			ids[i] = m.matchID
			allMatchIDs[m.matchID] = true
		}
		parts := loadParticipants(db, ids)
		old := replay(matches, parts, true)
		neu := replay(matches, parts, false)
		players = append(players, playerData{
			gamertag:      gt,
			xuid:          xuid,
			replayOld:     old,
			replayNew:     neu,
			finalMUOldByC: finalMUByChain(old),
			finalMUNewByC: finalMUByChain(neu),
		})
	}

	if len(players) < 2 {
		log.Fatal("moins de 2 joueurs résolus, intersection impossible")
	}

	// Diagnostic : pour chaque joueur, ré-exécuter le même IN(...) que loadParticipants,
	// et compter combien de rows reviennent pour le match [1] (50cd2d8c-...).
	const targetMatch = "50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e"
	fmt.Printf("\n══ Diagnostic loadParticipants : rows réellement chargées pour le match %s ══\n", targetMatch)
	fmt.Println("(via le SQL exact de loadLUSRParticipants, contexte = liste d'IDs propre à chaque joueur)")
	for _, p := range players {
		probeLoadParticipantsForMatch(db, p, targetMatch)
	}
	// Affine pour Madina : essayer chunks plus petits pour identifier le seuil.
	fmt.Println("\n══ Affinement : test du seuil chunk pour Madina ══")
	for _, p := range players {
		if p.gamertag != "Madina97294" {
			continue
		}
		for _, chunkSize := range []int{50, 100, 200, 500, 1000} {
			probeChunked(db, p, targetMatch, chunkSize)
		}
		// Test concat trick : "match_id || '' IN (?,?,...)" pour Madina
		probeChunkedConcat(db, p, targetMatch, 500)
	}

	commonMatches := intersectRecentCommon(players, *nMatches)
	if len(commonMatches) == 0 {
		fmt.Println("Aucun match commun trouvé entre ces joueurs.")
	} else {
		fmt.Printf("\n══ %d derniers matchs communs (%s) ══\n",
			len(commonMatches), strings.Join(gamertags, " ∩ "))
		if len(commonMatches) > 0 {
			dumpParticipants(db, commonMatches[0].matchID, xuidByGT)
		}
		printMatchTable(commonMatches, players, *verbose)
	}

	printFinalSummary(players)
}

// probeLoadParticipantsForMatch simule loadLUSRParticipants pour un joueur
// (mêmes chunks de 500, même SQL IN (...)). Filtre ensuite la sortie sur
// targetMatch pour voir quelles rows sont visibles en pratique.
func probeLoadParticipantsForMatch(db *sql.DB, p playerData, targetMatch string) {
	// Collecte ses match_ids depuis le replay (équivalent loadMatches)
	allIDs := make([]string, 0, len(p.replayOld))
	for _, r := range p.replayOld {
		allIDs = append(allIDs, r.matchID)
	}
	// + tous ceux qui n'ont pas passé GetLUSRChain ? Non, replayRow ne contient
	// que les chains valides — mais loadLUSRParticipants charge AVANT filtrage,
	// donc on prend une marge. On reconstruit en relisant loadMatches.
	rawMatches := loadMatches(db, p.xuid)
	allIDs = allIDs[:0]
	for _, m := range rawMatches {
		allIDs = append(allIDs, m.matchID)
	}

	totalRows := 0
	visibleRowsForTarget := 0
	visibleXUIDs := []string{}

	const chunk = 500
	for start := 0; start < len(allIDs); start += chunk {
		end := start + chunk
		if end > len(allIDs) {
			end = len(allIDs)
		}
		batch := allIDs[start:end]
		ph := strings.Repeat("?,", len(batch))
		ph = ph[:len(ph)-1]
		args := make([]interface{}, len(batch))
		batchHasTarget := false
		for i, id := range batch {
			args[i] = id
			if id == targetMatch {
				batchHasTarget = true
			}
		}
		rows, err := db.Query("SELECT match_id, xuid FROM match_participants WHERE match_id IN ("+ph+")", args...)
		if err != nil {
			fmt.Printf("  %s: erreur chunk[%d:%d] : %v\n", p.gamertag, start, end, err)
			continue
		}
		for rows.Next() {
			totalRows++
			var mid, x string
			_ = rows.Scan(&mid, &x)
			if mid == targetMatch {
				visibleRowsForTarget++
				visibleXUIDs = append(visibleXUIDs, x)
			}
		}
		rows.Close()
		_ = batchHasTarget
	}
	fmt.Printf("  %s (xuid=%s) : %d ids → total rows ramenées=%d, pour %s visibles=%d %v\n",
		p.gamertag, p.xuid, len(allIDs), totalRows, targetMatch, visibleRowsForTarget, visibleXUIDs)
}

// probeChunked teste loadParticipants avec une taille de chunk donnée.
func probeChunked(db *sql.DB, p playerData, targetMatch string, chunkSize int) {
	rawMatches := loadMatches(db, p.xuid)
	allIDs := make([]string, len(rawMatches))
	for i, m := range rawMatches {
		allIDs[i] = m.matchID
	}
	visible := 0
	visibleX := []string{}
	for start := 0; start < len(allIDs); start += chunkSize {
		end := start + chunkSize
		if end > len(allIDs) {
			end = len(allIDs)
		}
		batch := allIDs[start:end]
		ph := strings.Repeat("?,", len(batch))
		ph = ph[:len(ph)-1]
		args := make([]interface{}, len(batch))
		for i, id := range batch {
			args[i] = id
		}
		rows, err := db.Query("SELECT match_id, xuid FROM match_participants WHERE match_id IN ("+ph+")", args...)
		if err != nil {
			continue
		}
		for rows.Next() {
			var mid, x string
			_ = rows.Scan(&mid, &x)
			if mid == targetMatch {
				visible++
				visibleX = append(visibleX, x)
			}
		}
		rows.Close()
	}
	fmt.Printf("  Madina chunk=%-5d : visibles pour %s = %d %v\n", chunkSize, truncate(targetMatch, 12), visible, visibleX)
}

// probeChunkedConcat utilise le trick "match_id || ”" pour défaire le pushdown.
func probeChunkedConcat(db *sql.DB, p playerData, targetMatch string, chunkSize int) {
	rawMatches := loadMatches(db, p.xuid)
	allIDs := make([]string, len(rawMatches))
	for i, m := range rawMatches {
		allIDs[i] = m.matchID
	}
	visible := 0
	for start := 0; start < len(allIDs); start += chunkSize {
		end := start + chunkSize
		if end > len(allIDs) {
			end = len(allIDs)
		}
		batch := allIDs[start:end]
		ph := strings.Repeat("?,", len(batch))
		ph = ph[:len(ph)-1]
		args := make([]interface{}, len(batch))
		for i, id := range batch {
			args[i] = id
		}
		rows, err := db.Query("SELECT match_id, xuid FROM match_participants WHERE match_id || '' IN ("+ph+")", args...)
		if err != nil {
			continue
		}
		for rows.Next() {
			var mid, x string
			_ = rows.Scan(&mid, &x)
			if mid == targetMatch {
				visible++
			}
		}
		rows.Close()
	}
	fmt.Printf("  Madina chunk=%-5d (concat trick) : visibles pour %s = %d\n", chunkSize, truncate(targetMatch, 12), visible)
}

// dumpRawXUIDs imprime le résultat d'une query SELECT xuid (simple) avec label.
func dumpRawXUIDs(db *sql.DB, label, query string) {
	rows, err := db.Query(query)
	fmt.Printf("  [%s] err=%v :\n", label, err)
	if rows == nil {
		return
	}
	defer rows.Close()
	i := 0
	for rows.Next() {
		i++
		var x string
		if e := rows.Scan(&x); e != nil {
			fmt.Printf("    scan err=%v\n", e)
			continue
		}
		fmt.Printf("    row %d : %q\n", i, x)
	}
	if e := rows.Err(); e != nil {
		fmt.Printf("    rows.Err()=%v\n", e)
	}
	fmt.Printf("    total=%d\n", i)
}

// printMatchParticipantsKind interroge information_schema pour savoir si
// match_participants est une table ou une view (et son SQL si view).
func printMatchParticipantsKind(db *sql.DB) {
	fmt.Println("\n══ Nature de match_participants ══")
	// Liste les databases attachées (au cas où des views ou tables externes brouillent les pistes)
	rows0, _ := db.Query(`SELECT database_name FROM duckdb_databases()`)
	if rows0 != nil {
		fmt.Println("  databases attachées :")
		for rows0.Next() {
			var d string
			_ = rows0.Scan(&d)
			fmt.Printf("    - %s\n", d)
		}
		rows0.Close()
	}
	// Liste toutes les tables/views nommées match_participants (peu importe le schéma)
	rowsT, _ := db.Query(`SELECT table_catalog, table_schema, table_name, table_type FROM information_schema.tables WHERE table_name='match_participants'`)
	if rowsT != nil {
		fmt.Println("  match_participants occurrences :")
		for rowsT.Next() {
			var c, s, n, t string
			_ = rowsT.Scan(&c, &s, &n, &t)
			fmt.Printf("    - %s.%s.%s (%s)\n", c, s, n, t)
		}
		rowsT.Close()
	}
	// Test 1 — WHERE match_id = literal
	dumpRawXUIDs(db, "= literal", `SELECT xuid FROM match_participants WHERE match_id = '50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e'`)
	// Test 2 — WHERE match_id LIKE literal
	dumpRawXUIDs(db, "LIKE literal", `SELECT xuid FROM match_participants WHERE match_id LIKE '50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e'`)
	// Test 3 — CTE force table scan
	dumpRawXUIDs(db, "CTE", `WITH t AS (SELECT xuid, match_id FROM match_participants) SELECT xuid FROM t WHERE match_id = '50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e'`)
	// Test 4 — ORDER BY rowid (force scan)
	dumpRawXUIDs(db, "ORDER BY rowid", `SELECT xuid FROM match_participants WHERE match_id = '50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e' ORDER BY rowid`)
	// Test 5 — OR avec dummy
	dumpRawXUIDs(db, "OR FALSE", `SELECT xuid FROM match_participants WHERE match_id = '50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e' OR FALSE`)
	// Test 6 — concat trick pour défaire l'index pushdown
	dumpRawXUIDs(db, "concat trick", `SELECT xuid FROM match_participants WHERE match_id || '' = '50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e'`)
	// Test 7 — SUBSTRING trick
	dumpRawXUIDs(db, "substring", `SELECT xuid FROM match_participants WHERE substring(match_id, 1, 36) = '50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e'`)
	// Test 8 — IN clause (utilisé par loadLUSRParticipants en prod)
	dumpRawXUIDs(db, "IN literal", `SELECT xuid FROM match_participants WHERE match_id IN ('50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e')`)
	// Test 9 — IN avec doublon (force list)
	dumpRawXUIDs(db, "IN 2 ids", `SELECT xuid FROM match_participants WHERE match_id IN ('50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e', 'dummy')`)
	// Test 10 — placeholder ? (comme en prod via loadLUSRParticipants)
	rows10, _ := db.Query(`SELECT xuid FROM match_participants WHERE match_id IN (?)`, "50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e")
	fmt.Println("  [IN (?)] :")
	if rows10 != nil {
		i := 0
		for rows10.Next() {
			i++
			var x string
			_ = rows10.Scan(&x)
			fmt.Printf("    row %d : %q\n", i, x)
		}
		fmt.Printf("    total=%d\n", i)
		rows10.Close()
	}
	// Test 11 — placeholder ? sur = simple
	rows11, _ := db.Query(`SELECT xuid FROM match_participants WHERE match_id = ?`, "50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e")
	fmt.Println("  [= ? (placeholder)] :")
	if rows11 != nil {
		i := 0
		for rows11.Next() {
			i++
			var x string
			_ = rows11.Scan(&x)
			fmt.Printf("    row %d : %q\n", i, x)
		}
		fmt.Printf("    total=%d\n", i)
		rows11.Close()
	}
	// Test 12 — IN avec 5 match_ids placeholders (comme loadLUSRParticipants prod)
	args := []interface{}{
		"50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e",
		"5a1c0c9c-0c0e-4339-a7b4-2daa9593d543",
		"de3cec8b-edf1-4edc-ad87-830369e0a358",
		"d3b139c6-ebd8-4d40-bf5d-aa3c6b76582a",
		"1ff0bc89-c078-47dd-920b-b96eb69f21bc",
	}
	rows12, _ := db.Query(`SELECT xuid FROM match_participants WHERE match_id IN (?,?,?,?,?)`, args...)
	fmt.Println("  [IN 5 placeholders] :")
	if rows12 != nil {
		i := 0
		uniq := map[string]bool{}
		for rows12.Next() {
			i++
			var x string
			_ = rows12.Scan(&x)
			uniq[x] = true
		}
		fmt.Printf("    total rows=%d, unique xuids=%d\n", i, len(uniq))
		rows12.Close()
	}
	// Test 13 — IN avec 1000 match_ids (force hash join)
	allIDsQ, _ := db.Query(`SELECT match_id FROM match_participants WHERE match_id || '' LIKE '50cd2d8c%' OR rowid >= 0 LIMIT 1000`)
	var allIDs []interface{}
	for allIDsQ.Next() {
		var m string
		_ = allIDsQ.Scan(&m)
		allIDs = append(allIDs, m)
	}
	allIDsQ.Close()
	// Force 50cd2d8c en première position
	allIDs = append([]interface{}{"50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e"}, allIDs...)
	placeholders := strings.Repeat("?,", len(allIDs))
	placeholders = placeholders[:len(placeholders)-1]
	rows13, _ := db.Query(`SELECT xuid FROM match_participants WHERE match_id = '50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e' AND match_id IN (`+placeholders+`)`, allIDs...)
	fmt.Printf("  [WHERE = literal AND IN %d placeholders] :\n", len(allIDs))
	if rows13 != nil {
		i := 0
		for rows13.Next() {
			i++
			var x string
			_ = rows13.Scan(&x)
			if i <= 12 {
				fmt.Printf("    row %d : %q\n", i, x)
			}
		}
		fmt.Printf("    total=%d\n", i)
		rows13.Close()
	}
	// Test 14 — concat trick dans la condition
	rows14, _ := db.Query(`SELECT xuid FROM match_participants WHERE match_id || '' IN (?)`, "50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e")
	fmt.Println("  [concat IN (?)] :")
	if rows14 != nil {
		i := 0
		for rows14.Next() {
			i++
			var x string
			_ = rows14.Scan(&x)
			fmt.Printf("    row %d : %q\n", i, x)
		}
		fmt.Printf("    total=%d\n", i)
		rows14.Close()
	}
	// SELECT pour Madina seule, sur ce match : direct row inspection
	rowsM, _ := db.Query(`SELECT xuid, match_id, team_id, kills, kills_expected FROM match_participants WHERE xuid = '2533274858283686' AND match_id = '50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e'`)
	if rowsM != nil {
		fmt.Println("  SELECT direct (Madina, 50cd2d8c) :")
		for rowsM.Next() {
			var x, mid string
			var tid sql.NullInt64
			var kills sql.NullInt64
			var ke sql.NullFloat64
			_ = rowsM.Scan(&x, &mid, &tid, &kills, &ke)
			fmt.Printf("    - xuid=%s match_id=%s team=%v kills=%v ke=%v\n", x, mid, tid, kills, ke)
		}
		rowsM.Close()
	}
	// Liste tous les match_ids de Madina contenant '50cd2d8c' (en cas de typo/suffixe)
	rowsL, _ := db.Query(`SELECT match_id FROM match_participants WHERE xuid = '2533274858283686' AND match_id LIKE '50cd2d8c%'`)
	if rowsL != nil {
		fmt.Println("  Madina match_ids LIKE '50cd2d8c%' :")
		for rowsL.Next() {
			var m string
			_ = rowsL.Scan(&m)
			fmt.Printf("    - [%s] (len=%d)\n", m, len(m))
		}
		rowsL.Close()
	}
	rows, err := db.Query(`SELECT table_type FROM information_schema.tables WHERE table_name='match_participants'`)
	if err != nil {
		fmt.Printf("  erreur SQL : %v\n", err)
		return
	}
	for rows.Next() {
		var t string
		_ = rows.Scan(&t)
		fmt.Printf("  table_type = %s\n", t)
	}
	rows.Close()
	cnt := db.QueryRow(`SELECT COUNT(*) FROM match_participants`)
	var n int64
	_ = cnt.Scan(&n)
	fmt.Printf("  COUNT(*) match_participants = %d\n", n)
	cnt2 := db.QueryRow(`SELECT COUNT(DISTINCT match_id) FROM match_participants`)
	var n2 int64
	_ = cnt2.Scan(&n2)
	fmt.Printf("  COUNT(DISTINCT match_id) = %d\n", n2)
	cnt3 := db.QueryRow(`SELECT COUNT(*) FROM match_participants WHERE match_id = '50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e'`)
	var n3 int64
	_ = cnt3.Scan(&n3)
	fmt.Printf("  COUNT(*) WHERE match_id=50cd2d8c... = %d\n", n3)
	cnt4 := db.QueryRow(`SELECT COUNT(*) FROM match_participants WHERE xuid = '2533274858283686'`)
	var n4 int64
	_ = cnt4.Scan(&n4)
	fmt.Printf("  COUNT(*) WHERE xuid=Madina = %d\n", n4)
	cnt5 := db.QueryRow(`SELECT COUNT(*) FROM match_participants WHERE xuid = '2533274858283686' AND match_id = '50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e'`)
	var n5 int64
	_ = cnt5.Scan(&n5)
	fmt.Printf("  COUNT(*) (Madina, 50cd2d8c) = %d\n", n5)
}

// dumpParticipants imprime tous les participants d'un match (xuid, team_id, KE,
// kills, deaths) pour diagnostiquer un éventuel mismatch xuid / team.
func dumpParticipants(db *sql.DB, matchID string, picked map[string]string) {
	fmt.Printf("\n══ Dump tous les participants de %s ══\n", matchID)
	// Inline le match_id (literal SQL) — diagnostic d'un soupçon de bug placeholder.
	q := `SELECT mp.xuid, mp.team_id, COALESCE(mp.kills, 0), COALESCE(mp.deaths, 0),
		       COALESCE(mp.kills_expected, 0), COALESCE(a.gamertag, '?')
		FROM match_participants mp
		LEFT JOIN xuid_aliases a ON a.xuid = mp.xuid
		WHERE mp.match_id = '` + matchID + `'
		ORDER BY mp.team_id, mp.kills DESC`
	rows, err := db.Query(q)
	if err != nil {
		fmt.Printf("  erreur SQL : %v\n", err)
		return
	}
	defer rows.Close()
	highlight := make(map[string]string)
	for gt, x := range picked {
		highlight[x] = gt
	}
	fmt.Println("  ┌────────────────────┬─────┬──────┬──────┬───────┬─────────────────────┬─────────┐")
	fmt.Println("  │ xuid               │ T   │ K    │ D    │ KE    │ gamertag            │ tracked │")
	fmt.Println("  ├────────────────────┼─────┼──────┼──────┼───────┼─────────────────────┼─────────┤")
	for rows.Next() {
		var xuid, gt string
		var team sql.NullInt64
		var kills, deaths int
		var ke float64
		if rows.Scan(&xuid, &team, &kills, &deaths, &ke, &gt) != nil {
			continue
		}
		teamStr := "NULL"
		if team.Valid {
			teamStr = fmt.Sprintf("%d", team.Int64)
		}
		tracked := ""
		if name, ok := highlight[xuid]; ok {
			tracked = "← " + name
		}
		fmt.Printf("  │ %-18s │ %-3s │ %4d │ %4d │ %5.1f │ %-19s │ %-7s │\n",
			truncate(xuid, 18), teamStr, kills, deaths, ke, truncate(gt, 19), tracked)
	}
	fmt.Println("  └────────────────────┴─────┴──────┴──────┴───────┴─────────────────────┴─────────┘")
}

// printAliasReport affiche pour chaque gamertag :
//   - le nombre de xuids distincts dans xuid_aliases (alias count)
//   - le xuid retenu pour le replay
//
// Permet de détecter les ambiguïtés (changement de gamertag, doublons).
func printAliasReport(db *sql.DB, gamertags []string, picked map[string]string) {
	fmt.Println("══ Résolution xuid_aliases ══")
	for _, gt := range gamertags {
		rows, err := db.Query(
			"SELECT xuid, COALESCE(last_seen::VARCHAR, '?') FROM xuid_aliases WHERE lower(gamertag) = ? ORDER BY last_seen DESC NULLS LAST",
			strings.ToLower(gt))
		if err != nil {
			fmt.Printf("  %s : erreur SQL %v\n", gt, err)
			continue
		}
		var aliases []string
		for rows.Next() {
			var x, ls string
			if rows.Scan(&x, &ls) == nil {
				aliases = append(aliases, x+" (last_seen "+ls+")")
			}
		}
		rows.Close()
		fmt.Printf("  %-12s : %d alias(es), retenu=%s\n", gt, len(aliases), picked[strings.ToLower(gt)])
		for _, a := range aliases {
			fmt.Printf("                 → %s\n", a)
		}
	}
}

// openShared ouvre la DB partagée en lecture seule.
func openShared() *sql.DB {
	connector, err := duckdb.NewConnector(sharedDBPath+"?access_mode=READ_ONLY",
		func(execer driver.ExecerContext) error {
			_, e := execer.ExecContext(context.Background(), "SET TimeZone='UTC'", nil)
			return e
		})
	if err != nil {
		log.Fatalf("open shared %s: %v", sharedDBPath, err)
	}
	db := sql.OpenDB(connector)
	if err := db.Ping(); err != nil {
		log.Fatalf("ping shared: %v", err)
	}
	return db
}

// resolveXUIDs interroge xuid_aliases (case-insensitive) pour mapper gamertag → xuid.
func resolveXUIDs(db *sql.DB, gamertags []string) map[string]string {
	if len(gamertags) == 0 {
		return map[string]string{}
	}
	q := "SELECT lower(gamertag), xuid FROM xuid_aliases WHERE lower(gamertag) IN ("
	args := make([]interface{}, len(gamertags))
	for i, gt := range gamertags {
		if i > 0 {
			q += ","
		}
		q += "?"
		args[i] = strings.ToLower(gt)
	}
	q += ")"
	rows, err := db.Query(q, args...)
	if err != nil {
		log.Fatalf("resolveXUIDs query: %v", err)
	}
	defer rows.Close()
	out := make(map[string]string, len(gamertags))
	for rows.Next() {
		var gt, xuid string
		if err := rows.Scan(&gt, &xuid); err == nil {
			if _, dup := out[gt]; !dup {
				out[gt] = xuid
			}
		}
	}
	return out
}

// finalMUByChain extrait la dernière valeur mu par chaîne d'un replay.
func finalMUByChain(rows []replayRow) map[string]float64 {
	out := make(map[string]float64)
	for _, r := range rows {
		out[r.chain] = r.muAfter
	}
	return out
}

// commonMatch décrit un match présent dans le replay des >=3 joueurs.
type commonMatch struct {
	matchID   string
	startTime string
	pairName  string
	chain     string
}

// intersectRecentCommon prend les N matchs récents communs aux joueurs fournis.
func intersectRecentCommon(players []playerData, n int) []commonMatch {
	if len(players) == 0 {
		return nil
	}
	// Set des match_ids pour chaque joueur, et map matchID→meta pour le premier
	sets := make([]map[string]bool, len(players))
	meta := make(map[string]commonMatch)
	for i, p := range players {
		set := make(map[string]bool, len(p.replayOld))
		for _, r := range p.replayOld {
			set[r.matchID] = true
			if _, ok := meta[r.matchID]; !ok {
				meta[r.matchID] = commonMatch{
					matchID:   r.matchID,
					startTime: r.startTime.UTC().Format("2006-01-02 15:04"),
					pairName:  r.pairName,
					chain:     r.chain,
				}
			}
		}
		sets[i] = set
	}
	// Intersection
	var common []commonMatch
	for mid, m := range meta {
		all := true
		for _, s := range sets {
			if !s[mid] {
				all = false
				break
			}
		}
		if all {
			common = append(common, m)
		}
	}
	sort.Slice(common, func(i, j int) bool { return common[i].startTime > common[j].startTime })
	if len(common) > n {
		common = common[:n]
	}
	// On veut chrono ASC dans le tableau
	sort.Slice(common, func(i, j int) bool { return common[i].startTime < common[j].startTime })
	return common
}

// playerData regroupe l'état de replay d'un joueur (deux modes en parallèle).
type playerData struct {
	gamertag      string
	xuid          string
	replayOld     []replayRow
	replayNew     []replayRow
	finalMUOldByC map[string]float64
	finalMUNewByC map[string]float64
}

// matchSlot groupe les deux replays (old/new) d'un joueur pour un match donné.
type matchSlot struct {
	old replayRow
	neu replayRow
	ok  bool
}

// printMatchTable affiche le tableau comparatif sur les matchs communs.
// verbose=true ajoute le breakdown des 8 composantes du composite par joueur.
func printMatchTable(matches []commonMatch, players []playerData, verbose bool) {
	idx := make(map[string]map[string]matchSlot)
	for _, p := range players {
		oldByID := make(map[string]replayRow, len(p.replayOld))
		for _, r := range p.replayOld {
			oldByID[r.matchID] = r
		}
		neuByID := make(map[string]replayRow, len(p.replayNew))
		for _, r := range p.replayNew {
			neuByID[r.matchID] = r
		}
		for mid, o := range oldByID {
			if _, ok := idx[mid]; !ok {
				idx[mid] = map[string]matchSlot{}
			}
			idx[mid][p.gamertag] = matchSlot{old: o, neu: neuByID[mid], ok: true}
		}
	}

	for i, m := range matches {
		fmt.Printf("\n[%2d] %s  chain=%s  mode=%s\n      match_id=%s\n",
			i+1, m.startTime, m.chain, truncate(m.pairName, 40), m.matchID)
		fmt.Println("      ┌─────────────┬────┬─────┬──────┬──────┬─────────┬─────────┬───────────────┬──────────────┬────────────────┐")
		fmt.Println("      │ joueur      │ T  │ Out │ K    │ D    │ KE / TM │ DE      │ score_kve_o/n │ composite o/n│ Δmu_old / new  │")
		fmt.Println("      ├─────────────┼────┼─────┼──────┼──────┼─────────┼─────────┼───────────────┼──────────────┼────────────────┤")
		for _, p := range players {
			s, ok := idx[m.matchID][p.gamertag]
			if !ok || !s.ok {
				continue
			}
			team := "?"
			if s.old.teamIDValid {
				team = fmt.Sprintf("%d", s.old.teamID)
			}
			outc := "?"
			if s.old.outcome >= 0 {
				outc = map[int]string{1: "tie", 2: "win", 3: "lose", 4: "dnf"}[s.old.outcome]
			}
			tmAvg := "-"
			if s.old.teammateAvgKE > 0 {
				tmAvg = fmt.Sprintf("%.1f", s.old.teammateAvgKE)
			}
			fmt.Printf("      │ %-11s │ %-2s │ %-3s │ %4.0f │ %4.0f │ %4.1f/%-3s │ %4.1f    │ %.2f / %.2f   │ %.3f / %.3f │ %+5.1f / %+5.1f  │\n",
				truncate(p.gamertag, 11), team, outc, s.old.kills, s.old.deaths,
				s.old.killsExpected, tmAvg, s.old.deathsExp,
				s.old.scoreKVE, s.neu.scoreKVE,
				s.old.composite, s.neu.composite,
				s.old.deltaMU, s.neu.deltaMU)
		}
		fmt.Println("      └─────────────┴────┴─────┴──────┴──────┴─────────┴─────────┴───────────────┴──────────────┴────────────────┘")

		if verbose {
			printBreakdown(m.matchID, idx, players)
		}
	}
}

// printBreakdown détaille les 8 composantes du composite pour chaque joueur.
func printBreakdown(matchID string, idx map[string]map[string]matchSlot, players []playerData) {
	keys := []string{
		lusync.MetricKeyKillsVsExpected,
		lusync.MetricKeyDeathsVsExpected,
		lusync.MetricKeyWinFactor,
		lusync.MetricKeyDamageEfficiency,
		lusync.MetricKeyAccuracyDelta,
		lusync.MetricKeyOffensiveConv,
		lusync.MetricKeyDefensiveResist,
	}
	weights := lusync.CompositeWeights
	fmt.Println("      breakdown (composantes du composite, mode old) :")
	fmt.Printf("      %-13s │", "composante")
	for _, p := range players {
		fmt.Printf(" %-11s │", truncate(p.gamertag, 11))
	}
	fmt.Println(" poids")
	for _, k := range keys {
		fmt.Printf("      %-13s │", k)
		for _, p := range players {
			s := idx[matchID][p.gamertag]
			v, has := s.old.breakdown[k]
			if !has {
				fmt.Printf(" %-11s │", "–")
			} else {
				fmt.Printf(" %-11s │", fmt.Sprintf("%.3f", v))
			}
		}
		fmt.Printf(" %.2f\n", weights[k])
	}
}

// printFinalSummary affiche le tier final old vs new par joueur, par chaîne.
func printFinalSummary(players []playerData) {
	fmt.Println("\n══ Bilan final par chaîne LUSR (μ après tout l'historique) ══")
	fmt.Println("┌─────────────┬──────────────────┬──────────┬───────────────┬──────────┬───────────────┐")
	fmt.Println("│ joueur      │ chaîne           │ μ_old    │ tier_old      │ μ_new    │ tier_new      │")
	fmt.Println("├─────────────┼──────────────────┼──────────┼───────────────┼──────────┼───────────────┤")
	for _, p := range players {
		chains := mergeChainKeys(p.finalMUOldByC, p.finalMUNewByC)
		for _, c := range chains {
			muOld := p.finalMUOldByC[c]
			muNew := p.finalMUNewByC[c]
			fmt.Printf("│ %-11s │ %-16s │ %8.1f │ %-13s │ %8.1f │ %-13s │\n",
				truncate(p.gamertag, 11), c,
				muOld, lusync.FormatTierLabel(muOld),
				muNew, lusync.FormatTierLabel(muNew))
		}
	}
	fmt.Println("└─────────────┴──────────────────┴──────────┴───────────────┴──────────┴───────────────┘")
}

func mergeChainKeys(a, b map[string]float64) []string {
	set := make(map[string]bool)
	for k := range a {
		set[k] = true
	}
	for k := range b {
		set[k] = true
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
