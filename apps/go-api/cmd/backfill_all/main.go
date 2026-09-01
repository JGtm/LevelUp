// cmd/backfill_all — backfill rétroactif PSA pour tous les players.
//
// Pour chaque player dans data/titles/<title>/players/, charge ses tokens Halo
// depuis le MultiUserTokenStore (ADR 0023), liste les match_ids sans
// personal_score_awards, et lance le pipeline en série.
//
// Lance ce CLI quand l'audit (cmd/audit_coverage) montre des trous, et que tu
// veux rattraper sans passer par /api/v1/backfill/start (pas besoin de session).
//
// LA MOITIÉ « WEAPONS » A ÉTÉ RETIRÉE le 2026-09-01 : son exécuteur (étape 1.55,
// corrélation tirs ↔ instant du kill) est supprimé. Le rattrapage du détail par arme
// passe désormais par `levelup backfill-killsource --online`, qui décode la source
// du dégât du même film.
//
// Usage :
//
//	go run ./cmd/backfill_all/                         # tous les players
//	go run ./cmd/backfill_all/ -player JGtm            # un seul
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
	gosync "levelup/go-api/internal/sync"
)

var (
	dataRoot     = flag.String("data", "data", "Racine du dossier data/")
	titleSlug    = flag.String("title", "halo_infinite", "Title slug")
	envFile      = flag.String("env-file", ".env.local", "Chemin .env.local")
	playerFilter = flag.String("player", "", "Limiter à un player (vide = tous)")
)

func main() {
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	loadEnvLocal(*envFile)

	ctx := context.Background()

	playersDir := filepath.Join(*dataRoot, "titles", *titleSlug, "players")
	entries, err := os.ReadDir(playersDir)
	if err != nil {
		fatal("readdir %s: %v", playersDir, err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		gamertag := e.Name()
		if *playerFilter != "" && gamertag != *playerFilter {
			continue
		}
		processPlayer(ctx, gamertag)
	}
}

func processPlayer(ctx context.Context, gamertag string) {
	playerDBPath := filepath.Join(*dataRoot, "titles", *titleSlug, "players", gamertag, "stats.duckdb")
	sharedDBPath := filepath.Join(*dataRoot, "titles", *titleSlug, "warehouse", "shared_matches_v2.duckdb")

	xuid, err := readPlayerXUID(playerDBPath)
	if err != nil {
		slog.Warn("xuid lookup failed", "player", gamertag, "err", err)
	}
	if xuid == "" {
		// Fallback via xuid_aliases
		shared, err := sql.Open("duckdb", sharedDBPath+"?access_mode=read_only")
		if err == nil {
			_ = shared.QueryRow(`SELECT xuid FROM xuid_aliases WHERE gamertag = ? LIMIT 1`, gamertag).Scan(&xuid)
			shared.Close()
		}
	}
	if xuid == "" {
		slog.Warn("xuid introuvable, skip", "player", gamertag)
		return
	}

	// Tokens
	tokens, err := loadTokens(ctx, gamertag, xuid)
	if err != nil {
		slog.Warn("tokens introuvables, skip", "player", gamertag, "err", err)
		return
	}

	fmt.Printf("\n=== %s (xuid=%s) ===\n", gamertag, xuid)

	// Match list pour PSA : tous les matchs participés sans entry dans personal_score_awards
	psaMatches, err := loadMissingPSAMatches(playerDBPath, sharedDBPath, xuid)
	if err != nil {
		slog.Error("loadMissingPSAMatches", "player", gamertag, "err", err)
	}

	fmt.Printf("  psa à backfill     : %d match(s)\n", len(psaMatches))

	engine := gosync.NewSyncEngine(repoRootForCWD(), gamertag, xuid, tokens, nil)

	if len(psaMatches) > 0 {
		fmt.Printf("  ▶ BackfillPersonalScoreAwardsForMatches (%d)...\n", len(psaMatches))
		matches, rows, err := engine.BackfillPersonalScoreAwardsForMatches(ctx, psaMatches)
		if err != nil {
			slog.Error("BackfillPersonalScoreAwardsForMatches", "player", gamertag, "err", err)
		}
		fmt.Printf("    → %d match(s), %d rows insérés\n", matches, rows)
	}
}

func loadMissingPSAMatches(playerDBPath, sharedDBPath, xuid string) ([]string, error) {
	pdb, err := sql.Open("duckdb", playerDBPath+"?access_mode=read_only")
	if err != nil {
		return nil, err
	}
	defer pdb.Close()
	shared, err := sql.Open("duckdb", sharedDBPath+"?access_mode=read_only")
	if err != nil {
		return nil, err
	}
	defer shared.Close()

	// Set des match_ids déjà avec PSA
	have := map[string]struct{}{}
	rows, err := pdb.Query(`SELECT DISTINCT match_id FROM personal_score_awards`)
	if err != nil {
		return nil, fmt.Errorf("psa lookup: %w", err)
	}
	for rows.Next() {
		var id string
		_ = rows.Scan(&id)
		have[id] = struct{}{}
	}
	rows.Close()

	// Match_ids participés dans match_participants - have
	rows, err = shared.Query(`
		SELECT DISTINCT mp.match_id
		FROM match_participants mp
		JOIN match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?
		ORDER BY mr.start_time DESC NULLS LAST`, xuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		_ = rows.Scan(&id)
		if _, ok := have[id]; !ok {
			out = append(out, id)
		}
	}
	return out, rows.Err()
}

func readPlayerXUID(dbPath string) (string, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	if err != nil {
		return "", err
	}
	defer db.Close()
	var xuid string
	_ = db.QueryRow(`SELECT value FROM sync_meta WHERE key = 'player_xuid'`).Scan(&xuid)
	return xuid, nil
}

// loadTokens résout les tokens Halo du joueur via le MultiUserTokenStore
// (source unique ADR 0023) — plus aucun fallback sync_meta / env var / store
// mono-user depuis la Phase 5 (2026-08-25).
func loadTokens(ctx context.Context, gamertag, xuid string) (*domain.HaloTokens, error) {
	store := auth.NewMultiUserTokenStore(filepath.Join(*dataRoot, "auth", "watcher_tokens"))
	result, err := auth.RefreshHaloTokensViaStoreFirst(ctx, store, auth.NewSISUProvider(), xuid, gamertag)
	if err != nil {
		return nil, err
	}
	tokens := auth.HaloTokensFromExchange(result)
	if tokens == nil {
		return nil, fmt.Errorf("aucun token Halo pour %s (xuid=%s) — store watcher_tokens vide", gamertag, xuid)
	}
	return tokens, nil
}

func loadEnvLocal(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}

func repoRootForCWD() string {
	wd, _ := os.Getwd()
	// Le CLI s'exécute typiquement depuis apps/go-api/. SyncEngine attend la
	// racine du repo (parent de apps/).
	if strings.HasSuffix(strings.ReplaceAll(wd, "\\", "/"), "/apps/go-api") {
		return filepath.Dir(filepath.Dir(wd))
	}
	return wd
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", args...)
	os.Exit(1)
}
