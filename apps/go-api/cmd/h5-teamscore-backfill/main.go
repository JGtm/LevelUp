// Outil ops : backfill des SCORES D'ÉQUIPE par match Halo 5 (team_0_score /
// team_1_score du registry) depuis le carnage (TeamStats[].Score = score objectif,
// captures de drapeau / zones incluses). Déjà fetché à l'ingest mais droppé par le
// mapper → registry NULL. Cet outil re-fetch le carnage et UPDATE le registry.
//
// Rend la dominance EXACTE tous modes (loadTeamScoresOrKillSums préfère le registry
// aux sommes de kills) + permet l'affichage du score d'équipe en match-view.
//
// Token EMPRUNTÉ possible (LEVELUP_H5_AUTH_AS). UPDATE offline single-writer (pas de
// serveur concurrent). Idempotent (re-UPDATE la même valeur).
//
//	Usage : LEVELUP_REPO_ROOT=<repo principal> [LEVELUP_H5_AUTH_AS=<sain>] \
//	        go run ./cmd/h5-teamscore-backfill [Gamertag-auth] [maxMatches]
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
	"levelup/go-api/internal/platform/auth"
)

func main() {
	authGT := "JGtm"
	if len(os.Args) > 1 {
		authGT = os.Args[1]
	}
	if v := os.Getenv("LEVELUP_H5_AUTH_AS"); v != "" {
		authGT = v
	}
	maxMatches := 0
	if len(os.Args) > 2 {
		if n, err := strconv.Atoi(os.Args[2]); err == nil {
			maxMatches = n
		}
	}

	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		fatal("config.Load: %v", err)
	}
	var authXUID string
	for _, slug := range []string{halo5.TitleSlug, ""} {
		ps, e := cfg.LoadPlayers(slug)
		if e != nil {
			continue
		}
		for i := range ps {
			if ps[i].Gamertag == authGT {
				authXUID = ps[i].XUID
			}
		}
		if authXUID != "" {
			break
		}
	}
	if authXUID == "" {
		fatal("xuid auth introuvable pour %q", authGT)
	}

	store := auth.NewMultiUserTokenStore(titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())
	res, err := auth.RefreshHaloTokensViaStoreFirst(ctx, store, auth.NewSISUProvider(), authXUID, authGT)
	if err != nil || res == nil || res.Tokens == nil {
		fatal("refresh tokens (auth_as=%s): %v", authGT, err)
	}
	ctx = ctxkeys.WithHaloAuth(ctx, res.Tokens, authXUID)
	src, err := halo5.NewCaptureSource(ctx)
	if err != nil {
		fatal("NewCaptureSource: %v", err)
	}

	sharedPath := titlePkg.NewPathResolver(cfg.RepoRoot).SharedDBPath(halo5.TitleSlug)
	shared, err := sql.Open("duckdb", sharedPath)
	if err != nil {
		fatal("open shared RW: %v", err)
	}
	defer shared.Close()
	shared.SetMaxOpenConns(1)

	matchIDs, err := loadMatchesMissingTeamScore(ctx, shared, maxMatches)
	if err != nil {
		fatal("load matches: %v", err)
	}
	fmt.Printf("team-score backfill h5 (auth_as=%s) : %d matchs à traiter\n", authGT, len(matchIDs))

	var updated, withScore, fetchErr int
	for i, id := range matchIDs {
		carnage, cerr := src.GetMatchCarnage(ctx, id, "arena")
		if cerr != nil {
			fetchErr++
			continue
		}
		t0, t1, ok := teamScores(carnage)
		if !ok {
			continue
		}
		if _, err := shared.ExecContext(ctx,
			`UPDATE match_registry SET team_0_score = ?, team_1_score = ? WHERE match_id = ?`,
			t0, t1, id); err != nil {
			fmt.Printf("  warn: update %s: %v\n", id, err)
			continue
		}
		updated++
		withScore++
		if (i+1)%200 == 0 {
			fmt.Printf("  ... %d/%d (maj=%d)\n", i+1, len(matchIDs), updated)
		}
	}
	fmt.Printf("team-score backfill h5 : matchs=%d maj=%d avec_score=%d fetch_err=%d\n",
		len(matchIDs), updated, withScore, fetchErr)
}

// loadMatchesMissingTeamScore : match_ids dont le score d'équipe n'est pas renseigné.
func loadMatchesMissingTeamScore(ctx context.Context, shared *sql.DB, maxMatches int) ([]string, error) {
	q := `SELECT match_id FROM match_registry
	      WHERE team_0_score IS NULL OR team_1_score IS NULL
	      ORDER BY COALESCE(start_time_utc, start_time) DESC`
	if maxMatches > 0 {
		q += fmt.Sprintf(" LIMIT %d", maxMatches)
	}
	rows, err := shared.QueryContext(ctx, q)
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

// teamScores extrait (team0, team1) depuis le carnage. ok=false si < 2 équipes
// (FFA / carnage vide) — la dominance 2-équipes ne s'applique pas.
func teamScores(c *halo5.H5CarnageResponse) (int, int, bool) {
	if c == nil {
		return 0, 0, false
	}
	t0, t1 := -1, -1
	for i := range c.TeamStats {
		switch c.TeamStats[i].TeamId {
		case 0:
			t0 = c.TeamStats[i].Score
		case 1:
			t1 = c.TeamStats[i].Score
		}
	}
	if t0 < 0 || t1 < 0 {
		return 0, 0, false
	}
	return t0, t1, true
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
