//go:build cgo

// diag_backfill_dryrun — audit read-only "what would CSR/LUSR backfill do?".
//
// Pour chaque joueur de db_profiles.json :
//   - Compte les matchs ranked dans shared.match_registry (CSR fetch scope)
//   - Compte les rows match_skill_rank existantes (CSR + LUSR déjà persistées)
//   - Compte les match_participants pour ce xuid (LUSR scope)
//   - Vérifie la présence d'un refresh token dans le MultiUserTokenStore
//     (data/auth/watcher_tokens/{xuid}.json — source unique ADR 0023)
//
// N'OUVRE QUE EN READ-ONLY, ne fait AUCUNE écriture. Aucun appel API Halo.
//
// Usage : go run -tags cgo ./cmd/diag_backfill_dryrun [data-root]
//
//	data-root par défaut : ../../data (relatif à apps/go-api/)
package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	duckdb "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/platform/auth"
)

type playerProfile struct {
	Gamertag string `json:"-"`
	XUID     string `json:"xuid"`
	DBPath   string `json:"db_path"`
}

type profilesFile struct {
	Profiles map[string]map[string]playerProfile `json:"profiles"`
}

func main() {
	dataRoot := "../../data"
	if len(os.Args) > 1 {
		dataRoot = os.Args[1]
	}
	profilesPath := filepath.Join(dataRoot, "..", "db_profiles.json")
	if _, err := os.Stat(profilesPath); err != nil {
		profilesPath = "db_profiles.json"
	}

	fmt.Println("================================================================")
	fmt.Println(" DRY-RUN AUDIT : backfill --csr --force --all + --lusr --force --all")
	fmt.Println("================================================================")
	fmt.Printf("data-root     : %s\n", dataRoot)
	fmt.Printf("profiles-file : %s\n", profilesPath)
	fmt.Println()

	players, err := loadPlayers(profilesPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: %v\n", err)
		os.Exit(1)
	}
	if len(players) == 0 {
		fmt.Println("Aucun joueur configuré.")
		return
	}

	sharedPath := filepath.Join(dataRoot, "titles", "halo_infinite", "warehouse", "shared_matches_v2.duckdb")
	shared, err := openRO(sharedPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: open shared: %v\n", err)
		os.Exit(1)
	}
	defer shared.Close()

	totalRankedAll := countSharedRanked(shared)
	fmt.Printf("shared.match_registry : %d matchs ranked au total (toutes lignes confondues)\n", totalRankedAll)
	fmt.Println()

	dumpRankedAndPlaylistDistribution(shared)
	fmt.Println()

	fmt.Printf("%-20s %-22s %-8s %-12s %-12s %-12s %-12s\n",
		"gamertag", "xuid", "token", "matches", "ranked", "msr_rows", "lusr_scope")
	fmt.Println(strings.Repeat("-", 100))

	tokenStore := auth.NewMultiUserTokenStore(filepath.Join(dataRoot, "auth", "watcher_tokens"))

	var grandRanked, grandMSR, grandLUSRScope, withToken, withoutToken int
	for _, p := range players {
		user, terr := tokenStore.Load(p.XUID)
		hasToken := terr == nil && user != nil && user.OAuthRefreshToken != ""
		tokenStr := "MISSING"
		if hasToken {
			tokenStr = "OK"
			withToken++
		} else {
			withoutToken++
		}

		matches := countMatchParticipants(shared, p.XUID)
		ranked := countRankedForPlayer(shared, p.XUID)
		grandRanked += ranked

		playerDB := filepath.Join(dataRoot, "titles", "halo_infinite", "players", p.Gamertag, "stats.duckdb")
		msr, lusrScope := 0, matches
		if _, err := os.Stat(playerDB); err == nil {
			pdb, perr := openRO(playerDB)
			if perr == nil {
				msr = countMatchSkillRank(pdb)
				pdb.Close()
			}
		}
		grandMSR += msr
		grandLUSRScope += lusrScope

		fmt.Printf("%-20s %-22s %-8s %-12d %-12d %-12d %-12d\n",
			truncate(p.Gamertag, 20), p.XUID, tokenStr, matches, ranked, msr, lusrScope)
	}

	fmt.Println(strings.Repeat("-", 100))
	fmt.Println()
	fmt.Println("RESUME DRY-RUN :")
	fmt.Printf("  Joueurs              : %d (avec token: %d, sans token: %d)\n", len(players), withToken, withoutToken)
	fmt.Printf("  CSR fetch candidats  : ~%d (matchs ranked tous joueurs avec token)\n", grandRanked)
	fmt.Printf("  CSR rows deja en DB  : %d (match_skill_rank total)\n", grandMSR)
	fmt.Printf("  LUSR scope (matches) : %d (sera recalcule from scratch avec --force)\n", grandLUSRScope)
	fmt.Println()
	fmt.Println("ESTIMATIONS TEMPS :")
	fmt.Printf("  CSR  : ~%d sec (matchs %d * 200ms en moyenne, rate-limited)\n", grandRanked*200/1000, grandRanked)
	fmt.Printf("  LUSR : ~%d sec (calcul local pur Go, ~5ms/match)\n", grandLUSRScope*5/1000)
	fmt.Println()
	if withoutToken > 0 {
		fmt.Println("ATTENTION : Joueurs SANS token OAuth seront SKIP par CSR backfill.")
		fmt.Println("            Authentifier le joueur (SSO Xbox) ou `go run ./cmd/token-capture/ <GT>`.")
	}
}

func loadPlayers(path string) ([]playerProfile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var pf profilesFile
	if err := json.Unmarshal(b, &pf); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	titleProfiles, ok := pf.Profiles["halo_infinite"]
	if !ok {
		return nil, fmt.Errorf("no profiles.halo_infinite in %s", path)
	}
	out := make([]playerProfile, 0, len(titleProfiles))
	for gt, p := range titleProfiles {
		p.Gamertag = gt
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Gamertag < out[j].Gamertag })
	return out, nil
}

func openRO(path string) (*sql.DB, error) {
	connector, err := duckdb.NewConnector(path+"?access_mode=READ_ONLY", func(execer driver.ExecerContext) error {
		_, e := execer.ExecContext(context.Background(), "SET TimeZone='UTC'", nil)
		return e
	})
	if err != nil {
		return nil, err
	}
	return sql.OpenDB(connector), nil
}

func countSharedRanked(db *sql.DB) int {
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_registry WHERE COALESCE(is_ranked, FALSE) = TRUE`).Scan(&n)
	return n
}

func dumpRankedAndPlaylistDistribution(db *sql.DB) {
	fmt.Println("== is_ranked distribution in match_registry ==")
	rows, err := db.Query(`SELECT CAST(is_ranked AS VARCHAR) AS v, COUNT(*) AS n FROM match_registry GROUP BY 1 ORDER BY 2 DESC`)
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
	} else {
		for rows.Next() {
			var v sql.NullString
			var n int
			_ = rows.Scan(&v, &n)
			label := "NULL"
			if v.Valid {
				label = v.String
			}
			fmt.Printf("  is_ranked=%-8s count=%d\n", label, n)
		}
		rows.Close()
	}

	fmt.Println("\n== playlist_id presence in match_registry ==")
	var total, nonNull, nonEmpty int
	_ = db.QueryRow(`SELECT COUNT(*), COUNT(playlist_id), COUNT(CASE WHEN NULLIF(TRIM(COALESCE(playlist_id,'')),'') IS NOT NULL THEN 1 END) FROM match_registry`).Scan(&total, &nonNull, &nonEmpty)
	fmt.Printf("  total=%d  non_null=%d  non_empty=%d\n", total, nonNull, nonEmpty)

	fmt.Println("\n== Top 10 playlist_name (par count) ==")
	rows, _ = db.Query(`SELECT COALESCE(playlist_name,'') AS pn, COUNT(*) n FROM match_registry GROUP BY 1 ORDER BY 2 DESC LIMIT 10`)
	for rows.Next() {
		var pn string
		var n int
		_ = rows.Scan(&pn, &n)
		fmt.Printf("  %-60s %d\n", truncate(pn, 60), n)
	}
	rows.Close()

	fmt.Println("\n== Top 10 pair_name (par count) ==")
	rows, _ = db.Query(`SELECT COALESCE(pair_name,'') AS pn, COUNT(*) n FROM match_registry GROUP BY 1 ORDER BY 2 DESC LIMIT 10`)
	for rows.Next() {
		var pn string
		var n int
		_ = rows.Scan(&pn, &n)
		fmt.Printf("  %-60s %d\n", truncate(pn, 60), n)
	}
	rows.Close()
}

// countRankedForPlayer compte les matchs ranked du joueur en utilisant le
// même filtre is_ranked que loadRankedMatchesForCSRBackfill. Reflète le
// scope réel du CSR backfill.
func countRankedForPlayer(db *sql.DB, xuid string) int {
	var n int
	_ = db.QueryRow(`
		SELECT COUNT(DISTINCT mp.match_id)
		FROM match_participants mp
		JOIN match_registry r ON r.match_id = mp.match_id
		WHERE mp.xuid = ?
		  AND COALESCE(r.is_ranked, FALSE) = TRUE`, xuid).Scan(&n)
	return n
}

func countMatchParticipants(db *sql.DB, xuid string) int {
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_participants WHERE xuid = ?`, xuid).Scan(&n)
	return n
}

func countMatchSkillRank(db *sql.DB) int {
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank`).Scan(&n)
	return n
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "."
}
