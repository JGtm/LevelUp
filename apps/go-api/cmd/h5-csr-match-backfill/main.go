// Outil ops : backfill du CSR PAR MATCH Halo 5 (matchs classés) dans match_skill_rank
// (rating_type='CSR', lu en priorité par l'UI). Le carnage ARENA (PGCR) porte
// PlayerStats[].CurrentCsr — déjà fetché à l'ingest mais droppé (DTO non modélisé,
// corrigé). Cet outil re-fetch le carnage des matchs classés du joueur et persiste
// le CSR post-match du joueur. La vue match_skill_rank_latest priorise CSR > LUSR →
// les matchs classés affichent le CSR, les sociaux gardent le LUSR (pas de re-run LUSR).
//
// Token EMPRUNTÉ possible (LEVELUP_H5_AUTH_AS) — le carnage est servi pour n'importe
// quel match avec n'importe quel SpartanToken v4 valide.
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
	"levelup/go-api/internal/platform/auth"
	syncpkg "levelup/go-api/internal/sync"
)

// h5Tiers — DesignationId (0..5) → libellé EN. Onyx (5) = palier unique sans sous-palier.
var h5Tiers = []string{"Bronze", "Silver", "Gold", "Platinum", "Diamond", "Onyx"}

func tierName(d int) string {
	if d >= 0 && d < len(h5Tiers) {
		return h5Tiers[d]
	}
	return ""
}

func main() {
	gt := "JGtm"
	if len(os.Args) > 1 {
		gt = os.Args[1]
	}
	maxMatches := 0 // 0 = tous
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
	sharedPath := pr.SharedDBPath(halo5.TitleSlug)
	playerPath := pr.PlayerDBPath(halo5.TitleSlug, gt)

	// Shared en lecture seule (liste des matchs classés). Player DB en RW (CSR).
	shared, err := sql.Open("duckdb", sharedPath+"?access_mode=read_only")
	if err != nil {
		fatal("open shared RO: %v", err)
	}
	defer shared.Close()
	playerDB, err := syncpkg.OpenPlayerDB(playerPath)
	if err != nil {
		fatal("open player DB: %v", err)
	}
	defer playerDB.Close()

	matches, err := loadRankedMatches(ctx, shared, xuid, maxMatches)
	if err != nil {
		fatal("load ranked matches: %v", err)
	}
	fmt.Printf("CSR par match h5 %s (auth_as=%s) : %d matchs classés à traiter\n", gt, authGT, len(matches))

	var written, withCsr, placement, fetchErr int
	for i, m := range matches {
		carnage, cerr := src.GetMatchCarnage(ctx, m.id, "arena")
		if cerr != nil {
			fetchErr++
			continue
		}
		cur, mml, found := ownerCsr(carnage, gt)
		if !found {
			continue
		}
		if cur == nil {
			placement++ // en placement / pas de CSR ce match
			continue
		}
		tier := tierName(cur.DesignationId)
		subTier := cur.Tier
		label := tier
		if tier != "" && !equalFoldOnyx(tier) && subTier > 0 {
			label = fmt.Sprintf("%s %d", tier, subTier)
		}
		if equalFoldOnyx(tier) {
			subTier = 0
		}
		if err := writeCSRRow(ctx, playerDB.SQLDb(), m.id, float64(cur.Csr), tier, subTier, label, m.start); err != nil {
			fmt.Printf("  warn: write CSR %s: %v\n", m.id, err)
			continue
		}
		written++
		withCsr++
		_ = mml
		if (i+1)%100 == 0 {
			fmt.Printf("  ... %d/%d (écrits=%d)\n", i+1, len(matches), written)
		}
	}
	fmt.Printf("CSR par match h5 %s : matchs=%d écrits=%d avec_csr=%d placement=%d fetch_err=%d\n",
		gt, len(matches), written, withCsr, placement, fetchErr)
	verify(ctx, playerDB.SQLDb())
}

type rankedMatch struct {
	id    string
	start sql.NullTime
}

// loadRankedMatches : match_ids classés du joueur (is_ranked=TRUE), du plus récent au
// plus ancien, + start_time. maxMatches <= 0 → tous.
func loadRankedMatches(ctx context.Context, shared *sql.DB, xuid string, maxMatches int) ([]rankedMatch, error) {
	q := `
		SELECT mr.match_id, COALESCE(mr.start_time_utc, mr.start_time)
		FROM match_registry mr
		JOIN match_participants mp ON mr.match_id = mp.match_id
		WHERE mp.xuid || '' = ?
		  AND COALESCE(mr.is_ranked, FALSE) = TRUE
		ORDER BY COALESCE(mr.start_time_utc, mr.start_time) DESC`
	if maxMatches > 0 {
		q += fmt.Sprintf(" LIMIT %d", maxMatches)
	}
	rows, err := shared.QueryContext(ctx, q, xuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []rankedMatch
	for rows.Next() {
		var m rankedMatch
		if err := rows.Scan(&m.id, &m.start); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ownerCsr trouve l'entrée du joueur (par gamertag) dans le carnage → CurrentCsr +
// MeasurementMatchesLeft. found=false si le joueur n'est pas dans le carnage.
func ownerCsr(c *halo5.H5CarnageResponse, gamertag string) (*halo5.H5Csr, int, bool) {
	if c == nil {
		return nil, 0, false
	}
	for i := range c.PlayerStats {
		p := &c.PlayerStats[i]
		if p.Player.Gamertag == gamertag {
			return p.CurrentCsr, p.MeasurementMatchesLeft, true
		}
	}
	return nil, 0, false
}

// writeCSRRow insère une ligne CSR append-only dans match_skill_rank (player DB).
func writeCSRRow(ctx context.Context, db *sql.DB, matchID string, value float64, tier string, subTier int, label string, start sql.NullTime) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO match_skill_rank
			(match_id, rating_type, rating_value, tier, sub_tier, tier_label, playlist_group, start_time)
		VALUES (?, 'CSR', ?, ?, ?, ?, 'h5_arena', ?)`,
		matchID, value, tier, subTier, label, start)
	return err
}

func equalFoldOnyx(s string) bool { return s == "Onyx" || s == "onyx" || s == "ONYX" }

func verify(ctx context.Context, playerDB *sql.DB) {
	var csr, lusr int
	_ = playerDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM match_skill_rank WHERE rating_type='CSR'`).Scan(&csr)
	_ = playerDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM match_skill_rank_latest WHERE rating_type='CSR'`).Scan(&lusr)
	fmt.Printf("match_skill_rank : CSR=%d lignes ; visibles dans _latest (priorité CSR>LUSR)=%d\n", csr, lusr)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
