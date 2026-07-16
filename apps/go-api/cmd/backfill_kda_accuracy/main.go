// cmd/backfill_kda_accuracy — réécrit match_participants.kda et .accuracy avec
// les valeurs NATIVES de l'API Halo (CoreStats.KDA, CoreStats.Accuracy).
//
// Contexte : avant le fix 2026-06-05, le sync CALCULAIT le KDA comme (k+a)/d et
// l'accuracy comme shots_hit/shots_fired*100, au lieu de lire les champs fournis
// par l'API. Le KDA stocké était donc faux (mauvaise formule — Halo utilise
// Kills + Assists/3 − Deaths). Ce backfill re-fetch chaque match et réécrit les
// deux colonnes telles que le jeu les expose. Soit la valeur vient de l'API,
// soit elle est NULL (aucun calcul de repli).
//
// Usage (serveur LevelUp ARRÊTÉ — accès RW exclusif à la shared DB) :
//
//	go run ./cmd/backfill_kda_accuracy --gamertag JGtm [--dry-run] [--limit N] [--rps 5]
//
// Auth : MultiUserTokenStore (ADR 0023) en priorité, via le xuid du --gamertag.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	auth_platform "levelup/go-api/internal/platform/auth"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	go_sync "levelup/go-api/internal/sync"
)

// (helper de progression : compteurs gérés inline dans main)

func main() {
	gamertag := flag.String("gamertag", "JGtm", "Gamertag dont les tokens servent à l'auth API")
	dryRun := flag.Bool("dry-run", false, "Lister sans écrire")
	limit := flag.Int("limit", 0, "Limiter au N matchs les plus récents (0 = tous)")
	rps := flag.Int("rps", 5, "Requêtes API par seconde")
	flag.Parse()

	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	sharedPath := pr.SharedDBPath(titlePkg.DefaultSlug)

	// 1. Ouvrir la shared DB en RW (échoue si le serveur tient le lock).
	handle, err := duckdbpkg.OpenReadWrite(sharedPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open shared RW (serveur LevelUp arrêté ?): %v\n", err)
		os.Exit(1)
	}
	defer handle.Close()
	db := handle.SQLDb()

	// 2. Résoudre le xuid du gamertag pour l'auth store-first.
	xuid, err := resolveXUID(cfg, *gamertag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve xuid %s: %v\n", *gamertag, err)
		os.Exit(1)
	}

	// 3. Auth via MultiUserTokenStore (ADR 0023).
	store := auth_platform.NewMultiUserTokenStore(pr.WatcherTokensDir())
	provider := auth_platform.NewSISUProvider()
	exch, err := auth_platform.RefreshHaloTokensViaStoreFirst(ctx, store, provider, xuid, *gamertag, auth_platform.LegacyAuthInputs{})
	if err != nil || exch == nil {
		fmt.Fprintf(os.Stderr, "auth %s: %v\n", *gamertag, err)
		os.Exit(1)
	}
	client := go_sync.NewHaloAPIClient(exch.Tokens.SpartanToken, exch.Tokens.ClearanceToken, *rps)

	// 4. Lister les match_ids (du plus récent au plus ancien).
	matchIDs, err := listMatchIDs(ctx, db, *limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "list matchs: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Matchs à retraiter : %d (dry-run=%v)\n", len(matchIDs), *dryRun)

	// 5. Re-fetch + UPDATE.
	var fetched, updated, apiNil int
	for i, mid := range matchIDs {
		mj, ferr := client.GetMatchStats(ctx, mid)
		if ferr != nil {
			slog.WarnContext(ctx, "backfill_kda: GetMatchStats échoué", "match_id", mid, "err", ferr)
			continue
		}
		fetched++
		rows := go_sync.ExtractParticipants(mj)
		for _, r := range rows {
			if r.KDA == nil && r.Accuracy == nil {
				apiNil++
			}
			if *dryRun {
				continue
			}
			if _, err := db.ExecContext(ctx,
				`UPDATE match_participants SET kda = ?, accuracy = ? WHERE match_id = ? AND xuid = ?`,
				r.KDA, r.Accuracy, mid, r.XUID); err != nil {
				slog.WarnContext(ctx, "backfill_kda: UPDATE échoué", "match_id", mid, "xuid", r.XUID, "err", err)
				continue
			}
			updated++
		}
		if (i+1)%25 == 0 {
			fmt.Printf("  [%d/%d] fetched=%d updated=%d api_nil=%d\n", i+1, len(matchIDs), fetched, updated, apiNil)
		}
	}
	fmt.Printf("\n=== Terminé ===\n  matchs fetchés : %d\n  rows mises à jour : %d\n  participants sans KDA/Accuracy API : %d\n",
		fetched, updated, apiNil)
}

// resolveXUID lit db_profiles.json pour mapper gamertag → xuid.
func resolveXUID(cfg *config.AppConfig, gamertag string) (string, error) {
	players, err := cfg.LoadPlayers(titlePkg.DefaultSlug)
	if err != nil {
		return "", err
	}
	for i := range players {
		if players[i].Gamertag == gamertag {
			return players[i].XUID, nil
		}
	}
	return "", fmt.Errorf("gamertag %q absent de db_profiles.json", gamertag)
}

// listMatchIDs retourne les match_id distincts triés du plus récent au plus ancien.
func listMatchIDs(ctx context.Context, db *sql.DB, limit int) ([]string, error) {
	q := `SELECT DISTINCT mp.match_id
	      FROM match_participants mp
	      JOIN match_registry mr ON mr.match_id = mp.match_id
	      ORDER BY ` + analysis.SQLStartTimeCanonical("mr") + ` DESC`
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
