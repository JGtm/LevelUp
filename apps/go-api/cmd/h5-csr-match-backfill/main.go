// Outil ops : backfill du CSR PAR MATCH (matchs classés → match_skill_rank) + du rang
// SR par match (career_progression) Halo 5, depuis le carnage arena (CurrentCsr +
// XpInfo). Le carnage est déjà fetché à l'ingest mais ces champs étaient droppés
// (DTO corrigé). Délègue à livesync.PersistPerMatchRatings (helper partagé avec le
// hook live PostScore — un seul fetch carnage par match pour CSR + SR).
//
// Token EMPRUNTÉ possible (LEVELUP_H5_AUTH_AS).
//
//	Usage : LEVELUP_REPO_ROOT=<repo principal> [LEVELUP_H5_AUTH_AS=<sain>] \
//	        go run ./cmd/h5-csr-match-backfill [Gamertag] [maxMatches]
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/games/halo_5/livesync"
	"levelup/go-api/internal/platform/auth"
	syncpkg "levelup/go-api/internal/sync"
)

func main() {
	gt := "JGtm"
	if len(os.Args) > 1 {
		gt = os.Args[1]
	}
	maxMatches := 0
	if len(os.Args) > 2 {
		if n, err := strconv.Atoi(os.Args[2]); err == nil {
			maxMatches = n
		}
	}
	authGT := gt
	if v := os.Getenv("LEVELUP_H5_AUTH_AS"); v != "" {
		authGT = v
	}

	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		fatal("config.Load: %v", err)
	}
	findXUID := func(who string) string {
		for _, slug := range []string{halo5.TitleSlug, ""} {
			ps, e := cfg.LoadPlayers(slug)
			if e != nil {
				continue
			}
			for i := range ps {
				if ps[i].Gamertag == who {
					return ps[i].XUID
				}
			}
		}
		return ""
	}
	xuid := findXUID(gt)
	authXUID := findXUID(authGT)
	if xuid == "" || authXUID == "" {
		fatal("xuid introuvable (cible=%q auth=%q)", gt, authGT)
	}

	store := auth.NewMultiUserTokenStore(titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())
	res, err := auth.RefreshHaloTokensViaStoreFirst(ctx, store, auth.NewMSALProvider(), authXUID, authGT, auth.LegacyAuthInputs{})
	if err != nil || res == nil || res.Tokens == nil {
		fatal("refresh tokens (auth_as=%s): %v", authGT, err)
	}
	ctx = ctxkeys.WithHaloAuth(ctx, res.Tokens, authXUID)
	src, err := halo5.NewCaptureSource(ctx)
	if err != nil {
		fatal("NewCaptureSource: %v", err)
	}

	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	shared, err := sql.Open("duckdb", pr.SharedDBPath(halo5.TitleSlug)+"?access_mode=read_only")
	if err != nil {
		fatal("open shared RO: %v", err)
	}
	defer shared.Close()
	playerDB, err := syncpkg.OpenPlayerDB(pr.PlayerDBPath(halo5.TitleSlug, gt))
	if err != nil {
		fatal("open player DB: %v", err)
	}
	defer playerDB.Close()

	// TOUS les matchs du joueur (le SR progresse à chaque match ; le CSR n'est écrit
	// que pour les classés par le helper). Plus récent → ancien.
	matchIDs, err := loadAllMatchIDs(ctx, shared, xuid, maxMatches)
	if err != nil {
		fatal("load matches: %v", err)
	}
	fmt.Printf("ratings par match h5 %s (auth_as=%s) : %d matchs à traiter\n", gt, authGT, len(matchIDs))

	csrN, srN := livesync.PersistPerMatchRatings(ctx, src, playerDB.SQLDb(), shared, gt, xuid, matchIDs)
	fmt.Printf("ratings par match h5 %s : matchs=%d csr=%d sr=%d\n", gt, len(matchIDs), csrN, srN)
	verify(ctx, playerDB.SQLDb())
}

func loadAllMatchIDs(ctx context.Context, shared *sql.DB, xuid string, maxMatches int) ([]string, error) {
	q := `SELECT mp.match_id
	      FROM match_participants mp JOIN match_registry mr ON mr.match_id = mp.match_id
	      WHERE mp.xuid || '' = ?
	      ORDER BY COALESCE(mr.start_time_utc, mr.start_time) DESC`
	if maxMatches > 0 {
		q += fmt.Sprintf(" LIMIT %d", maxMatches)
	}
	rows, err := shared.QueryContext(ctx, q, xuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func verify(ctx context.Context, playerDB *sql.DB) {
	count := func(q string) int {
		var n int
		_ = playerDB.QueryRowContext(ctx, q).Scan(&n)
		return n
	}
	fmt.Printf("match_skill_rank CSR=%d ; career_progression=%d snapshots\n",
		count(`SELECT COUNT(*) FROM match_skill_rank WHERE rating_type='CSR'`),
		count(`SELECT COUNT(*) FROM career_progression`))
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
