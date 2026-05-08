// cmd/replay_highlight_events — Rejoue le parsing des highlight events pour
// les matchs où l'ancien parser byte-aligné a silencieusement échoué OU dont
// les killer_victim_pairs sont vides à cause de l'ancien INSERT OR IGNORE.
//
// Idempotent — peut être ré-exécuté sans dommage.
//
// **Note** : ce binaire sera remplacé par la sous-commande
// `levelup replay-events` (Phase 3 de PLAN_HIGHLIGHT_EVENTS_BACKFILL.md). Il
// reste ici à titre transitionnel et délègue toute la logique aux helpers
// `internal/sync/events_replay.go`.
//
// Usage :
//
//	go run ./cmd/replay_highlight_events --gamertag JGtm [--limit 200] [--dry-run]
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	auth_platform "levelup/go-api/internal/platform/auth"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	go_sync "levelup/go-api/internal/sync"

	"database/sql"
)

func main() {
	gamertag := flag.String("gamertag", "", "Gamertag pour résoudre le refresh token via .env.local (obligatoire)")
	limit := flag.Int("limit", 200, "Nombre maximum de matchs à rejouer (les plus récents en premier)")
	dryRun := flag.Bool("dry-run", false, "Lister les matchs cassés sans les rejouer")
	rps := flag.Int("rps", 1, "Requêtes API par seconde (rate limiting)")
	flag.Parse()

	if strings.TrimSpace(*gamertag) == "" {
		fmt.Fprintln(os.Stderr, "usage: replay_highlight_events --gamertag <GT> [--limit N] [--dry-run] [--rps N]")
		os.Exit(2)
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})))

	if err := run(*gamertag, *limit, *dryRun, *rps); err != nil {
		fmt.Fprintf(os.Stderr, "erreur: %v\n", err)
		os.Exit(1)
	}
}

func run(gamertag string, limit int, dryRun bool, rps int) error {
	loadEnvLocal()

	envKey := "SPNKR_OAUTH_REFRESH_TOKEN_" + strings.ToUpper(gamertag)
	refreshToken := os.Getenv(envKey)
	if refreshToken == "" {
		return fmt.Errorf("refresh token absent : %s n'est pas défini dans .env.local", envKey)
	}

	ctx := context.Background()

	repoRoot, err := findRepoRoot()
	if err != nil {
		return fmt.Errorf("repo root: %w", err)
	}
	resolver := titlePkg.NewPathResolver(repoRoot)
	sharedPath := resolver.SharedDBPath(titlePkg.DefaultSlug)
	if _, err := os.Stat(sharedPath); err != nil {
		return fmt.Errorf("shared DB introuvable : %s", sharedPath)
	}

	sharedHandle, err := duckdbpkg.OpenReadWrite(sharedPath)
	if err != nil {
		return fmt.Errorf("open shared RW (server LevelUp tourne ?): %w", err)
	}
	defer sharedHandle.Close()
	sharedDB := sharedHandle.SQLDb()

	globalDB, globalCloser := openGlobalDBOptional(repoRoot)
	if globalCloser != nil {
		defer globalCloser()
	}

	broken, err := go_sync.FindBrokenHighlightEventMatches(ctx, sharedDB, limit)
	if err != nil {
		return fmt.Errorf("FindBrokenHighlightEventMatches: %w", err)
	}
	fmt.Printf("Matchs cassés détectés : %d (limit=%d)\n", len(broken), limit)
	if len(broken) == 0 {
		fmt.Println("Rien à rejouer.")
		return nil
	}

	if dryRun {
		fmt.Println("--- dry-run, liste seulement ---")
		for i, id := range broken {
			if i < 20 {
				fmt.Printf("  %s\n", id)
			}
		}
		if len(broken) > 20 {
			fmt.Printf("  ... (+ %d autres)\n", len(broken)-20)
		}
		return nil
	}

	provider := auth_platform.NewMSALProvider()
	tok, err := provider.TryOAuthRefresh(ctx, refreshToken)
	if err != nil {
		return fmt.Errorf("oauth refresh: %w", err)
	}
	exch, err := auth_platform.ExchangeAccessToken(ctx, tok)
	if err != nil {
		return fmt.Errorf("exchange: %w", err)
	}

	client := go_sync.NewHaloAPIClient(exch.Tokens.SpartanToken, exch.Tokens.ClearanceToken, rps)

	beforeAnomaly := observability.LoadCounter("highlight_events_parse_anomaly_total")

	progress := func(done, total int, matchID, status string) {
		fmt.Printf("[%d/%d] %s %s\n", done, total, matchID, status)
	}

	res, err := go_sync.ReplayHighlightEventsForMatches(ctx, client, sharedDB, globalDB, broken, progress)
	if err != nil {
		fmt.Fprintf(os.Stderr, "replay interrompu: %v\n", err)
	}

	afterAnomaly := observability.LoadCounter("highlight_events_parse_anomaly_total")
	fmt.Println("\n=== Rapport replay highlight events ===")
	fmt.Printf("  total processed   : %d\n", res.Total)
	fmt.Printf("  healed (events>0) : %d  (events insérés : %d)\n", res.Healed, res.EventsInserted)
	fmt.Printf("  no_film (404)     : %d\n", res.NoFilm)
	fmt.Printf("  parse_anomaly     : %d  (compteur expvar : %d → %d, delta=%d)\n",
		res.ParseAnomaly, beforeAnomaly, afterAnomaly, afterAnomaly-beforeAnomaly)
	fmt.Printf("  errors            : %d\n", res.Errors)

	if res.Errors > 0 {
		return fmt.Errorf("%d erreur(s) lors du replay", res.Errors)
	}
	return nil
}

// openGlobalDBOptional ouvre data/global/xbox_aliases.duckdb en RW si la DB
// existe et n'est pas lockée. Le replay fonctionne même si ce DB est absent
// (les xuid_aliases sont également peuplés dans la shared DB).
func openGlobalDBOptional(repoRoot string) (*sql.DB, func()) {
	path := filepath.Join(repoRoot, "data", "global", "xbox_aliases.duckdb")
	if _, err := os.Stat(path); err != nil {
		return nil, nil
	}
	handle, err := duckdbpkg.OpenReadWrite(path)
	if err != nil {
		fmt.Printf("[INFO] global DB lockée — alias upserts globaux désactivés (%v)\n", err)
		return nil, nil
	}
	return handle.SQLDb(), func() { _ = handle.Close() }
}

// loadEnvLocal lit `.env.local` et expose chaque clé non encore présente dans
// l'environnement (pattern repris de cmd/get-token).
func loadEnvLocal() {
	for _, path := range []string{".env.local", "../.env.local", "../../.env.local"} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
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
			if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
				val = val[1 : len(val)-1]
			}
			if os.Getenv(key) == "" {
				_ = os.Setenv(key, val)
			}
		}
		return
	}
}

// findRepoRoot remonte les parents jusqu'à trouver le dossier `.ai/`.
func findRepoRoot() (string, error) {
	if env := strings.TrimSpace(os.Getenv("LEVELUP_REPO_ROOT")); env != "" {
		return env, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, ".ai")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("repo root introuvable depuis %s", cwd)
}
