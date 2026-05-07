// cmd/replay_highlight_events — Rejoue le parsing des highlight events pour
// les matchs où l'ancien parser byte-aligné a silencieusement échoué :
// events_loaded=TRUE dans match_registry mais aucune ligne dans
// highlight_events. Ces matchs étaient marqués "déjà traités" mais leur
// historique d'événements (kills/deaths/medals/mode) avait été perdu.
//
// Le tool :
//  1. ouvre la shared DB en read-write (échoue si server LevelUp tourne)
//  2. liste les matchs cassés (events_loaded=TRUE, 0 row, participants présents)
//  3. pour chaque match : clear events_loaded, re-télécharge le chunk highlight,
//     parse, insère, re-set events_loaded
//
// Idempotent — peut être ré-exécuté sans dommage.
//
// Usage :
//
//	go run ./cmd/replay_highlight_events --gamertag JGtm [--limit 200] [--dry-run]
//
// `--gamertag` sert uniquement à résoudre le refresh token via .env.local
// (`SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG>`). Le token est utilisé pour les appels
// API ; il n'a pas besoin d'être lié au joueur des matchs en question.
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

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	auth_platform "levelup/go-api/internal/platform/auth"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	go_sync "levelup/go-api/internal/sync"
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

	// Réduit le bruit slog du package sync (Warnings restent visibles via le compteur expvar et les warnings dans result).
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

	broken, err := selectBrokenMatches(ctx, sharedDB, limit)
	if err != nil {
		return fmt.Errorf("select broken matches: %w", err)
	}
	fmt.Printf("Matchs cassés détectés : %d (limit=%d)\n", len(broken), limit)
	if len(broken) == 0 {
		fmt.Println("Rien à rejouer.")
		return nil
	}

	if dryRun {
		fmt.Println("--- dry-run, liste seulement ---")
		for i, m := range broken {
			if i < 20 {
				fmt.Printf("  %s  start=%s\n", m.MatchID, m.StartTime)
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

	stats := replayStats{}
	for i, m := range broken {
		if ctx.Err() != nil {
			break
		}
		fmt.Printf("[%d/%d] %s ", i+1, len(broken), m.MatchID)

		if err := clearEventsLoaded(sharedDB, m.MatchID); err != nil {
			fmt.Printf("FAIL clear: %v\n", err)
			stats.errors++
			continue
		}

		result := &domain.SyncResult{}
		err := go_sync.ProcessHighlightEvents(ctx, client, sharedDB, globalDB, m.MatchID, result)
		switch {
		case err != nil:
			fmt.Printf("FAIL: %v\n", err)
			stats.errors++
		case result.EventsInserted > 0:
			fmt.Printf("OK +%d events\n", result.EventsInserted)
			stats.healed++
			stats.eventsTotal += result.EventsInserted
		case len(result.Warnings) > 0:
			fmt.Printf("ANOMALY (chunk présent mais 0 events parsés)\n")
			stats.anomaly++
		default:
			fmt.Printf("no-film\n")
			stats.noFilm++
		}
	}

	afterAnomaly := observability.LoadCounter("highlight_events_parse_anomaly_total")
	fmt.Println("\n=== Rapport replay highlight events ===")
	fmt.Printf("  total processed   : %d\n", len(broken))
	fmt.Printf("  healed (events>0) : %d  (events insérés : %d)\n", stats.healed, stats.eventsTotal)
	fmt.Printf("  no_film (404)     : %d\n", stats.noFilm)
	fmt.Printf("  parse_anomaly     : %d  (compteur expvar : %d → %d, delta=%d)\n",
		stats.anomaly, beforeAnomaly, afterAnomaly, afterAnomaly-beforeAnomaly)
	fmt.Printf("  errors            : %d\n", stats.errors)

	if stats.errors > 0 {
		return fmt.Errorf("%d erreur(s) lors du replay", stats.errors)
	}
	return nil
}

type brokenMatch struct {
	MatchID   string
	StartTime string
}

type replayStats struct {
	healed      int
	noFilm      int
	anomaly     int
	errors      int
	eventsTotal int
}

// selectBrokenMatches retourne les matchs où le pipeline highlight events a
// échoué silencieusement, dans l'un des deux cas :
//
//  1. events_loaded=TRUE mais aucune ligne dans highlight_events (cassure
//     primaire = ancien parser byte-aligné) ;
//  2. highlight_events présents (kill events détectés) mais killer_victim_pairs
//     vides (cassure secondaire = ancien InsertKillerVictimPairs avec OR IGNORE).
//
// Filtré sur les matchs réels (présence de match_participants). Triés par
// start_time décroissant — les plus récents d'abord, dont le film est le plus
// susceptible d'être encore disponible côté CDN Halo.
func selectBrokenMatches(ctx context.Context, db *sql.DB, limit int) ([]brokenMatch, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := db.QueryContext(ctx, `
		SELECT mr.match_id,
		       COALESCE(mr.start_time_utc, mr.start_time AT TIME ZONE 'UTC')::VARCHAR
		FROM match_registry mr
		WHERE COALESCE(mr.events_loaded, FALSE) = TRUE
		  AND EXISTS (SELECT 1 FROM match_participants mp WHERE mp.match_id = mr.match_id)
		  AND (
		    NOT EXISTS (SELECT 1 FROM highlight_events he WHERE he.match_id = mr.match_id)
		    OR (
		      EXISTS (SELECT 1 FROM highlight_events he WHERE he.match_id = mr.match_id AND he.event_type = 'kill')
		      AND NOT EXISTS (SELECT 1 FROM killer_victim_pairs kvp WHERE kvp.match_id = mr.match_id)
		    )
		  )
		ORDER BY COALESCE(mr.start_time_utc, mr.start_time AT TIME ZONE 'UTC') DESC NULLS LAST
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []brokenMatch
	for rows.Next() {
		var m brokenMatch
		if err := rows.Scan(&m.MatchID, &m.StartTime); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// clearEventsLoaded remet events_loaded=FALSE et clear le bit MBitEvents dans
// match_registry.backfill_completed afin que le pipeline de re-parse traite
// le match comme "à faire".
func clearEventsLoaded(db *sql.DB, matchID string) error {
	_, err := db.Exec(`
		UPDATE match_registry
		SET events_loaded      = FALSE,
		    backfill_completed = COALESCE(backfill_completed, 0) & ~?
		WHERE match_id = ?`, go_sync.MBitEvents, matchID)
	return err
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
