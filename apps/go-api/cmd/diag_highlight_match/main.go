// Outil de diagnostic : explique pourquoi un match donné apparait ou non dans
// la section "Meilleures performances" de la page carrière d'un joueur.
//
// Usage:
//
//	go run apps/go-api/cmd/diag_highlight_match/main.go <gamertag> <match_id>
package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
)

const repoRoot = `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration`

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: diag_highlight_match <gamertag> <match_id>")
		os.Exit(1)
	}
	gamertag := os.Args[1]
	matchID := os.Args[2]

	playerPath := filepath.Join(repoRoot, "data", "titles", "halo_infinite", "players", gamertag, "stats.duckdb")
	sharedPath := filepath.Join(repoRoot, "data", "titles", "halo_infinite", "warehouse", "shared_matches_v2.duckdb")

	playerHandle, err := duckdbpkg.OpenReadOnly(playerPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open player: %v\n", err)
		os.Exit(1)
	}
	defer playerHandle.Close()
	playerDB := playerHandle.SQLDb()

	sharedHandle, err := duckdbpkg.OpenReadOnly(sharedPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open shared: %v\n", err)
		os.Exit(1)
	}
	defer sharedHandle.Close()
	sharedDB := sharedHandle.SQLDb()

	fmt.Printf("Gamertag: %s\nMatchID:  %s\n\n", gamertag, matchID)

	// --- Phase A : player_match_enrichment ---
	var perfScore sql.NullFloat64
	var dominanceFlag sql.NullInt64
	var hadBotTeammate sql.NullBool
	errA := playerDB.QueryRow(`
		SELECT performance_score, dominance_flag, had_bot_teammate
		FROM player_match_enrichment
		WHERE match_id = ?
	`, matchID).Scan(&perfScore, &dominanceFlag, &hadBotTeammate)

	fmt.Println("=== Phase A — player_match_enrichment ===")
	switch {
	case errA == sql.ErrNoRows:
		fmt.Println("  ! Match ABSENT de player_match_enrichment !")
	case errA != nil:
		fmt.Printf("  ! erreur: %v\n", errA)
	default:
		fmt.Printf("  performance_score:  %s\n", nullFloatStr(perfScore))
		fmt.Printf("  dominance_flag:     %s  (%s)\n", nullInt64Str(dominanceFlag), dominanceLabel(dominanceFlag))
		fmt.Printf("  had_bot_teammate:   %s\n", nullBoolStr(hadBotTeammate))
	}

	// --- xuid lookup ---
	var xuid string
	err = sharedDB.QueryRow(`
		SELECT xuid FROM v_gamertag_lookup
		WHERE gamertag ILIKE ? AND xuid NOT LIKE 'bid(%'
		ORDER BY LENGTH(xuid) DESC LIMIT 1
	`, gamertag).Scan(&xuid)
	if err != nil {
		fmt.Printf("\nxuid lookup error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\nXUID résolu: %s\n", xuid)

	// --- Phase B : shared (mp + r) ---
	var outcome sql.NullInt64
	var timePlayed sql.NullInt64
	var isFirefight sql.NullBool
	var kills, deaths sql.NullInt64
	var startTime sql.NullTime
	var pairName, playlistName sql.NullString
	errB := sharedDB.QueryRow(`
		SELECT mp.outcome, mp.time_played_seconds, r.is_firefight, mp.kills, mp.deaths,
			`+analysis.SQLStartTimeCanonical("r")+` AS start_time,
			r.pair_name, r.playlist_name
		FROM match_participants mp
		JOIN match_registry r ON mp.match_id = r.match_id
		WHERE mp.match_id = ? AND mp.xuid = ?
	`, matchID, xuid).Scan(&outcome, &timePlayed, &isFirefight, &kills, &deaths, &startTime, &pairName, &playlistName)

	fmt.Println("\n=== Phase B — shared (match_participants + match_registry) ===")
	switch {
	case errB == sql.ErrNoRows:
		fmt.Println("  ! Match ABSENT de shared !")
	case errB != nil:
		fmt.Printf("  ! erreur: %v\n", errB)
	default:
		fmt.Printf("  outcome:               %s  (%s)\n", nullInt64Str(outcome), outcomeLabel(outcome))
		fmt.Printf("  time_played_seconds:   %s\n", nullInt64Str(timePlayed))
		fmt.Printf("  is_firefight:          %s\n", nullBoolStr(isFirefight))
		fmt.Printf("  kills:                 %s\n", nullInt64Str(kills))
		fmt.Printf("  deaths:                %s\n", nullInt64Str(deaths))
		if startTime.Valid {
			fmt.Printf("  start_time:            %s\n", startTime.Time.Format("2006-01-02 15:04:05 MST"))
		}
		fmt.Printf("  pair_name:             %q\n", pairName.String)
		fmt.Printf("  playlist_name:         %q\n", playlistName.String)
	}

	// --- Présence cumulée des bots dans la team du joueur ---
	// Helpful pour comprendre pourquoi had_bot_teammate vaut TRUE/FALSE selon
	// le seuil hybride (botPresenceMinSeconds=30 ET botPresenceMinRatio=15%).
	var botSeconds sql.NullInt64
	var matchDuration sql.NullInt64
	errBot := sharedDB.QueryRow(`
		WITH self_team AS (
			SELECT team_id FROM match_participants WHERE match_id = ? AND xuid = ?
		)
		SELECT
			SUM(COALESCE(mp_bot.time_played_seconds, 0)) AS total_bot_seconds,
			MAX(COALESCE(r.duration_seconds, 0))         AS match_duration
		FROM match_participants mp_bot
		JOIN match_registry r ON r.match_id = mp_bot.match_id
		JOIN self_team st ON mp_bot.team_id = st.team_id
		WHERE mp_bot.match_id = ?
		  AND mp_bot.xuid <> ?
		  AND mp_bot.xuid LIKE 'bid(%'
	`, matchID, xuid, matchID, xuid).Scan(&botSeconds, &matchDuration)
	fmt.Println("\n=== Présence bots dans la team du joueur (seuil hybride) ===")
	switch {
	case errBot == sql.ErrNoRows || !botSeconds.Valid:
		fmt.Println("  Pas de bot dans la team du joueur.")
	case errBot != nil:
		fmt.Printf("  ! erreur: %v\n", errBot)
	default:
		fmt.Printf("  bot_seconds_cumulés:   %d\n", botSeconds.Int64)
		fmt.Printf("  match_duration_sec:    %d\n", matchDuration.Int64)
		if matchDuration.Int64 > 0 {
			ratio := float64(botSeconds.Int64) / float64(matchDuration.Int64)
			fmt.Printf("  ratio bot/match:       %.1f%%\n", ratio*100)
		}
		fmt.Printf("  seuils:                bot >= 30s ET ratio >= 15%%\n")
	}

	// --- ParticipationInfo des bots dans la team du joueur (colonnes ajoutées
	//     post mini-Phase 0.5, cf. steps_shared_add_participation_info.go). ---
	botRows, errBP := sharedDB.Query(`
		WITH self_team AS (
			SELECT team_id FROM match_participants WHERE match_id = ? AND xuid = ?
		)
		SELECT mp_bot.xuid,
		       COALESCE(mp_bot.time_played_seconds, 0),
		       mp_bot.present_at_beginning,
		       mp_bot.joined_in_progress,
		       mp_bot.left_in_progress,
		       mp_bot.present_at_completion
		FROM match_participants mp_bot
		JOIN self_team st ON mp_bot.team_id = st.team_id
		WHERE mp_bot.match_id = ?
		  AND mp_bot.xuid <> ?
		  AND mp_bot.xuid LIKE 'bid(%'
		ORDER BY mp_bot.xuid
	`, matchID, xuid, matchID, xuid)
	fmt.Println("\n=== ParticipationInfo des bots teammates ===")
	if errBP != nil {
		fmt.Printf("  ! erreur: %v\n", errBP)
	} else {
		defer botRows.Close()
		any := false
		for botRows.Next() {
			any = true
			var botXUID string
			var tp int64
			var atBegin, joined, left, atEnd sql.NullBool
			if err := botRows.Scan(&botXUID, &tp, &atBegin, &joined, &left, &atEnd); err != nil {
				fmt.Printf("  ! scan: %v\n", err)
				continue
			}
			fmt.Printf("  bot %s — time_played=%ds\n", botXUID, tp)
			fmt.Printf("    present_at_beginning  = %s\n", nullBoolStr(atBegin))
			fmt.Printf("    joined_in_progress    = %s\n", nullBoolStr(joined))
			fmt.Printf("    left_in_progress      = %s\n", nullBoolStr(left))
			fmt.Printf("    present_at_completion = %s\n", nullBoolStr(atEnd))
		}
		if !any {
			fmt.Println("  Aucun bot teammate.")
		}
	}

	// --- Verdict des filtres ---
	fmt.Println("\n=== Verdict des filtres ===")
	check := func(label string, ok bool, value string) {
		status := "[OK]    "
		if !ok {
			status = "[BLOQUÉ]"
		}
		fmt.Printf("  %s  %-35s  → %s\n", status, label, value)
	}
	perfOK := perfScore.Valid
	check("perf_score IS NOT NULL (Phase A)", perfOK, nullFloatStr(perfScore))

	timeOK := timePlayed.Valid && timePlayed.Int64 >= 180
	check("time_played >= 180 (Phase B)", timeOK, nullInt64Str(timePlayed))

	fireOK := !isFirefight.Valid || !isFirefight.Bool
	check("is_firefight = FALSE (Phase B)", fireOK, nullBoolStr(isFirefight))

	outcomeOK := outcome.Valid && (outcome.Int64 == 2 || outcome.Int64 == 3)
	check("outcome ∈ {2,3} (Phase B)", outcomeOK, nullInt64Str(outcome))

	// Asymétrie WIN/LOSS sur had_bot_teammate (politique A2 actuelle) :
	// WIN avec bot = conservé, LOSS avec bot = exclu de worst_matches.
	hadBot := hadBotTeammate.Valid && hadBotTeammate.Bool
	botOK := !(outcome.Valid && outcome.Int64 == 3 && hadBot)
	check("LOSS+bot non exclu (politique A2)", botOK,
		fmt.Sprintf("outcome=%s had_bot=%s", nullInt64Str(outcome), nullBoolStr(hadBotTeammate)))

	allOK := perfOK && botOK && timeOK && fireOK && outcomeOK
	if !allOK {
		fmt.Println("\n→ Le match est exclu par au moins un filtre (cf. ci-dessus).")
		return
	}

	// --- Phase C : rang dans le classement ---
	fmt.Println("\n→ Tous les filtres passent. Calcul du rang dans le tri WIN/LOSS…")

	pmeRows, err := playerDB.Query(`
		SELECT match_id, performance_score, COALESCE(dominance_flag, 0)
		FROM player_match_enrichment
		WHERE performance_score IS NOT NULL
	`)
	if err != nil {
		fmt.Printf("phase A query: %v\n", err)
		os.Exit(1)
	}
	type pmeEntry struct {
		Perf float64
		Flag int
	}
	pmes := make(map[string]pmeEntry)
	var ids []string
	for pmeRows.Next() {
		var mid string
		var p pmeEntry
		if err := pmeRows.Scan(&mid, &p.Perf, &p.Flag); err != nil {
			pmeRows.Close()
			fmt.Printf("scan: %v\n", err)
			os.Exit(1)
		}
		pmes[mid] = p
		ids = append(ids, mid)
	}
	pmeRows.Close()
	fmt.Printf("  Phase A : %d matchs candidats\n", len(ids))

	if len(ids) == 0 {
		return
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	q := fmt.Sprintf(`
		SELECT mp.match_id, COALESCE(mp.outcome, 0)
		FROM match_participants mp
		JOIN match_registry r ON mp.match_id = r.match_id
		WHERE mp.xuid = ?
		  AND COALESCE(mp.time_played_seconds, 0) >= 180
		  AND COALESCE(r.is_firefight, FALSE) = FALSE
		  AND COALESCE(mp.outcome, 0) IN (2, 3)
		  AND mp.match_id IN (%s)
	`, placeholders)
	args := make([]any, 0, 1+len(ids))
	args = append(args, xuid)
	for _, m := range ids {
		args = append(args, m)
	}
	sharedRows, err := sharedDB.Query(q, args...)
	if err != nil {
		fmt.Printf("phase B query: %v\n", err)
		os.Exit(1)
	}
	type sortable struct {
		MatchID string
		Outcome int
		Perf    float64
		Flag    int
	}
	var wins, losses []sortable
	for sharedRows.Next() {
		var mid string
		var oc int
		if err := sharedRows.Scan(&mid, &oc); err != nil {
			continue
		}
		p := pmes[mid]
		row := sortable{MatchID: mid, Outcome: oc, Perf: p.Perf, Flag: p.Flag}
		switch oc {
		case 2:
			wins = append(wins, row)
		case 3:
			losses = append(losses, row)
		}
	}
	sharedRows.Close()
	fmt.Printf("  Phase B : %d wins éligibles, %d losses éligibles\n", len(wins), len(losses))

	prio := func(flag int, ps []int) int {
		for _, p := range ps {
			if flag == p {
				return flag
			}
		}
		return 0
	}
	sort.SliceStable(wins, func(i, j int) bool {
		pi := prio(wins[i].Flag, []int{5, 3, 1})
		pj := prio(wins[j].Flag, []int{5, 3, 1})
		if pi != pj {
			return pi > pj
		}
		return wins[i].Perf > wins[j].Perf
	})

	rank := -1
	for i, w := range wins {
		if w.MatchID == matchID {
			rank = i + 1
			break
		}
	}
	fmt.Printf("\n  Rang du match cible dans wins triés : %d / %d\n", rank, len(wins))

	fmt.Println("\n  Classement détaillé (top 25 wins) :")
	fmt.Printf("  %-4s %-40s %-7s %-20s %s\n", "rk", "match_id", "perf", "dominance", "limite_15")
	fmt.Println("  " + strings.Repeat("-", 90))
	end := 25
	if end > len(wins) {
		end = len(wins)
	}
	for i := 0; i < end; i++ {
		marker := "  "
		if i+1 <= 15 {
			marker = "✓ "
		}
		highlight := ""
		if wins[i].MatchID == matchID {
			highlight = "  ← MATCH CIBLE"
		}
		fmt.Printf("  %-4d %-40s %-7.2f %-20s %s%s\n",
			i+1, wins[i].MatchID, wins[i].Perf, dominanceLabelInt(wins[i].Flag), marker, highlight)
	}

	switch {
	case rank == -1:
		fmt.Println("\n→ DIAGNOSTIC: Le match n'apparait PAS dans la liste finale alors qu'il passe les filtres. Bug logique improbable.")
	case rank > 15:
		fmt.Printf("\n→ DIAGNOSTIC: Le match est en position %d (HORS top 15).\n", rank)
		fmt.Println("  Cause : relégation par le tri 'dominance flag prioritaire d'abord'.")
		fmt.Println("  Les matchs aux positions 1-15 ont un dominance_flag ∈ {1,3,5} (DOMINATION/REMONTADA/CONTRE_REMONTADA).")
		fmt.Println("  Le match cible (flag absent ou non prioritaire) ne peut pas remonter, même avec un perf_score plus élevé.")
	default:
		fmt.Printf("\n→ DIAGNOSTIC: Le match est en position %d, DANS le top 15. Le bug est ailleurs (UI ? cache ? filtres saison/playlist ?)\n", rank)
	}
}

func nullFloatStr(n sql.NullFloat64) string {
	if !n.Valid {
		return "NULL"
	}
	return fmt.Sprintf("%.2f", n.Float64)
}

func nullInt64Str(n sql.NullInt64) string {
	if !n.Valid {
		return "NULL"
	}
	return fmt.Sprintf("%d", n.Int64)
}

func nullBoolStr(n sql.NullBool) string {
	if !n.Valid {
		return "NULL"
	}
	return fmt.Sprintf("%v", n.Bool)
}

func dominanceLabel(n sql.NullInt64) string {
	if !n.Valid {
		return "NULL"
	}
	return dominanceLabelInt(int(n.Int64))
}

func dominanceLabelInt(f int) string {
	switch f {
	case 0:
		return "none"
	case 1:
		return "1=DOMINATION"
	case 2:
		return "2=HUMILIATION"
	case 3:
		return "3=REMONTADA"
	case 4:
		return "4=DEBANDADE"
	case 5:
		return "5=CONTRE_REMONTADA"
	default:
		return fmt.Sprintf("%d=?", f)
	}
}

func outcomeLabel(n sql.NullInt64) string {
	if !n.Valid {
		return "NULL"
	}
	switch n.Int64 {
	case 1:
		return "TIE"
	case 2:
		return "WIN"
	case 3:
		return "LOSS"
	case 4:
		return "DNF"
	default:
		return fmt.Sprintf("%d=?", n.Int64)
	}
}
