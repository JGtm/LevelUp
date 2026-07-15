// cmd/backfill_all — backfill rétroactif weapons + PSA pour tous les players.
//
// Pour chaque player dans data/titles/<title>/players/, charge ses tokens Halo
// (msal_token_cache ou oauth_refresh_token), liste les match_ids manquants
// (weapon_kills < 28j, PSA tous matchs), et lance les pipelines en série.
//
// Lance ce CLI quand l'audit (cmd/audit_coverage) montre des trous, et que tu
// veux rattraper sans passer par /api/v1/backfill/start (pas besoin de session).
//
// Usage :
//
//	go run ./cmd/backfill_all/                         # tous les players
//	go run ./cmd/backfill_all/ -player JGtm            # un seul
//	go run ./cmd/backfill_all/ -only weapons           # weapons uniquement
//	go run ./cmd/backfill_all/ -only psa               # psa uniquement
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
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
	gosync "levelup/go-api/internal/sync"
)

const (
	filmExpiryDays        = 28.0
	mBitWeaponKillsNoFilm = 1 << 22
)

var (
	dataRoot     = flag.String("data", "data", "Racine du dossier data/")
	titleSlug    = flag.String("title", "halo_infinite", "Title slug")
	envFile      = flag.String("env-file", ".env.local", "Chemin .env.local")
	authFile     = flag.String("auth-file", "data/auth/watcher_tokens.json", "Chemin tokens.json")
	playerFilter = flag.String("player", "", "Limiter à un player (vide = tous)")
	onlyType     = flag.String("only", "", "weapons | psa | both (par défaut)")
	forceWeapons = flag.Bool("force-weapons", false, "Effacer et re-backfiller weapon_kills même si déjà présents")
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
	tokens, err := loadTokens(ctx, playerDBPath)
	if err != nil {
		slog.Warn("tokens introuvables, skip", "player", gamertag, "err", err)
		return
	}

	fmt.Printf("\n=== %s (xuid=%s) ===\n", gamertag, xuid)

	// Match list pour weapons : <28j sans wk et sans noFilm bit.
	// En mode -force-weapons : efface d'abord les rows existantes puis recharge.
	weaponMatches, err := loadMissingWeaponMatches(sharedDBPath, xuid, *forceWeapons)
	if err != nil {
		slog.Error("loadMissingWeaponMatches", "player", gamertag, "err", err)
	}
	// Match list pour PSA : tous les matchs participés sans entry dans personal_score_awards
	psaMatches, err := loadMissingPSAMatches(playerDBPath, sharedDBPath, xuid)
	if err != nil {
		slog.Error("loadMissingPSAMatches", "player", gamertag, "err", err)
	}

	fmt.Printf("  weapons à backfill : %d match(s) <28j\n", len(weaponMatches))
	fmt.Printf("  psa à backfill     : %d match(s)\n", len(psaMatches))

	engine := gosync.NewSyncEngine(repoRootForCWD(), gamertag, xuid, tokens, nil)

	if *onlyType != "psa" && len(weaponMatches) > 0 {
		fmt.Printf("  ▶ BackfillWeaponKillsForMatches (%d)...\n", len(weaponMatches))
		done, noFilm, err := engine.BackfillWeaponKillsForMatches(ctx, weaponMatches)
		if err != nil {
			slog.Error("BackfillWeaponKillsForMatches", "player", gamertag, "err", err)
		}
		fmt.Printf("    → %d done, %d film expiré\n", done, noFilm)
	}
	if *onlyType != "weapons" && len(psaMatches) > 0 {
		fmt.Printf("  ▶ BackfillPersonalScoreAwardsForMatches (%d)...\n", len(psaMatches))
		matches, rows, err := engine.BackfillPersonalScoreAwardsForMatches(ctx, psaMatches)
		if err != nil {
			slog.Error("BackfillPersonalScoreAwardsForMatches", "player", gamertag, "err", err)
		}
		fmt.Printf("    → %d match(s), %d rows insérés\n", matches, rows)
	}
}

func loadMissingWeaponMatches(sharedDBPath, xuid string, force bool) ([]string, error) {
	accessMode := "?access_mode=read_only"
	if force {
		accessMode = ""
	}
	db, err := sql.Open("duckdb", sharedDBPath+accessMode)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	cutoff := time.Now().Add(-time.Duration(filmExpiryDays*24) * time.Hour)

	if force {
		// Supprimer les weapon_kills existants pour ce joueur dans la fenêtre 28j
		// afin de permettre une re-attribution correcte (player_index).
		_, err := db.Exec(`
			DELETE FROM weapon_kills
			WHERE match_id IN (
				SELECT DISTINCT mp.match_id
				FROM match_participants mp
				JOIN match_registry mr ON mr.match_id = mp.match_id
				WHERE mp.xuid = ?
				  AND COALESCE(mr.is_firefight, FALSE) = FALSE
				  AND `+analysis.SQLStartTimeCanonical("mr")+` >= ?
				  AND (COALESCE(mr.backfill_completed, 0) & ?) = 0
			)
			AND xuid = ?`,
			xuid, cutoff, mBitWeaponKillsNoFilm, xuid)
		if err != nil {
			return nil, fmt.Errorf("force-weapons delete: %w", err)
		}
		// Effacer aussi le bit MBitWeaponKills (bit 21) pour que BackfillWeaponKillsForMatch
		// ne skipe pas les matchs déjà marqués done.
		const mBitWeaponKills = 1 << 21
		_, err = db.Exec(`
			UPDATE match_registry
			SET backfill_completed = backfill_completed & ~?
			WHERE match_id IN (
				SELECT DISTINCT mp.match_id
				FROM match_participants mp
				JOIN match_registry mr ON mr.match_id = mp.match_id
				WHERE mp.xuid = ?
				  AND COALESCE(mr.is_firefight, FALSE) = FALSE
				  AND `+analysis.SQLStartTimeCanonical("mr")+` >= ?
				  AND (COALESCE(mr.backfill_completed, 0) & ?) = 0
			)`,
			mBitWeaponKills, xuid, cutoff, mBitWeaponKillsNoFilm)
		if err != nil {
			return nil, fmt.Errorf("force-weapons clear bit: %w", err)
		}
	}

	rows, err := db.Query(`
		SELECT DISTINCT mp.match_id
		FROM match_participants mp
		JOIN match_registry mr ON mr.match_id = mp.match_id
		LEFT JOIN weapon_kills wk ON wk.match_id = mp.match_id AND wk.xuid = mp.xuid
		WHERE mp.xuid = ?
		  AND wk.match_id IS NULL
		  AND COALESCE(mr.is_firefight, FALSE) = FALSE
		  AND `+analysis.SQLStartTimeCanonical("mr")+` >= ?
		  AND (COALESCE(mr.backfill_completed, 0) & ?) = 0
		ORDER BY mr.start_time DESC`,
		xuid, cutoff, mBitWeaponKillsNoFilm)
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

func loadTokens(ctx context.Context, playerDBPath string) (*domain.HaloTokens, error) {
	const margin = 5 * time.Minute
	store := auth.NewTokenStore(*authFile)
	stored, err := store.Load()
	if err != nil {
		slog.Warn("watcher_tokens load failed", "path", *authFile, "err", err)
	}
	if stored != nil {
		slog.Info("watcher_tokens loaded",
			"xsts_valid", stored.IsXSTSValid(margin),
			"oauth_valid", stored.IsOAuthValid(margin),
			"has_refresh", stored.HasRefreshToken(),
		)
		if stored.IsXSTSValid(margin) {
			tokens, err := auth.ExchangeXSTSForHaloTokens(ctx, stored.XSTSToken)
			if err == nil {
				return tokens, nil
			}
			slog.Warn("ExchangeXSTSForHaloTokens failed", "err", err)
		}
		if stored.IsOAuthValid(margin) {
			result, err := auth.ExchangeAccessToken(ctx, stored.AccessToken)
			if err == nil {
				return result.Tokens, nil
			}
			slog.Warn("ExchangeAccessToken (watcher) failed", "err", err)
		}
	}
	return loadTokensFromPlayerDB(ctx, playerDBPath)
}

func loadTokensFromPlayerDB(ctx context.Context, dbPath string) (*domain.HaloTokens, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	var cacheJSON, refreshToken string
	_ = db.QueryRowContext(ctx, `SELECT value FROM sync_meta WHERE key = 'msal_token_cache'`).Scan(&cacheJSON)
	_ = db.QueryRowContext(ctx, `SELECT value FROM sync_meta WHERE key = 'oauth_refresh_token'`).Scan(&refreshToken)

	provider := auth.NewSISUProvider()
	gamertag := extractGamertag(dbPath)

	var accessToken string
	if cacheJSON != "" {
		if tok, err := provider.TrySilentRefresh(ctx, cacheJSON); err == nil && tok != "" {
			accessToken = tok
		}
	}
	if accessToken == "" && refreshToken != "" {
		if tok, err := provider.TryOAuthRefresh(ctx, refreshToken); err == nil && tok != "" {
			accessToken = tok
		}
	}
	if accessToken == "" && gamertag != "" {
		envKey := "SPNKR_OAUTH_REFRESH_TOKEN_" + strings.ToUpper(gamertag)
		if envRT := os.Getenv(envKey); envRT != "" {
			if tok, err := provider.TryOAuthRefresh(ctx, envRT); err == nil && tok != "" {
				accessToken = tok
			}
		}
	}
	if accessToken == "" {
		return nil, fmt.Errorf("aucun access token (msal_cache=%v oauth_rt=%v)", cacheJSON != "", refreshToken != "")
	}
	result, err := auth.ExchangeAccessToken(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	return result.Tokens, nil
}

func extractGamertag(dbPath string) string {
	parts := strings.Split(strings.ReplaceAll(dbPath, "\\", "/"), "/")
	for i, part := range parts {
		if part == "players" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
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
