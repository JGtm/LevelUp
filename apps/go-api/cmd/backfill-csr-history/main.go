//go:build cgo

// cmd/backfill-csr-history — backfille player_csr_snapshots pour les saisons CSR
// PASSÉES d'un joueur, alimentant le menu déroulant saison de la page Carrière
// (section "Classements"). Mécanisme Grunt : GetPlaylistCsr par playlist+saison,
// qui fonctionne pour n'importe quelle saison (l'endpoint skill est public —
// le xuid est un paramètre, un token de service suffit).
//
// Ne persiste que les snapshots à tier réel (une saison jamais jouée en classé
// n'apparaît donc pas dans le menu).
//
// IMPORTANT : stopper le serveur API avant de lancer (la player DB est ouverte
// en RW ; DuckDB interdit deux writers sur Windows).
//
// Usage :
//
//	go run ./cmd/backfill-csr-history -xuid 2533274823110022 \
//	    -player-db data/titles/halo_infinite/players/Madina97294/stats.duckdb
//	go run ./cmd/backfill-csr-history -xuid <XUID> -player-db <path> -season CsrSeason12-1 -dry-run
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/platform/auth"
	syncpkg "levelup/go-api/internal/sync"
)

func main() {
	xuid := flag.String("xuid", "", "XUID numérique du joueur cible (requis)")
	playerDBPath := flag.String("player-db", "", "chemin stats.duckdb du joueur (RW requis — stopper le serveur)")
	metadataDBPath := flag.String("metadata-db", "data/titles/halo_infinite/warehouse/metadata.duckdb", "chemin metadata.duckdb (RO, lecture csr_season_calendars)")
	watcherTokensDir := flag.String("watcher-tokens-dir", "data/auth/watcher_tokens", "répertoire MultiUserTokenStore (data/auth/watcher_tokens)")
	gamertag := flag.String("gamertag", "", "gamertag (logs uniquement)")
	titleID := flag.String("title", "halo_infinite", "title_id pour csr_season_calendars")
	season := flag.String("season", "", "limiter à une seule saison CSR (ex: CsrSeason12-1) ; vide = toutes les saisons")
	envFile := flag.String("env-file", ".env.local", "chemin .env.local (secret client Azure + LEVELUP_OAUTH_CLIENT_ID, refresh OAuth via SISUProvider + MultiUserTokenStore ADR 0023)")
	rateLimit := flag.Int("rate-limit", 60, "requêtes max/minute vers l'API skill")
	dryRun := flag.Bool("dry-run", false, "liste les saisons ciblées sans appeler l'API ni écrire")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	loadEnvLocal(*envFile)
	ctx := context.Background()

	if strings.TrimSpace(*xuid) == "" || strings.TrimSpace(*playerDBPath) == "" {
		fatal("-xuid et -player-db sont requis")
	}

	// Saisons cibles depuis csr_season_calendars (ou la seule saison demandée).
	seasons, err := resolveSeasons(ctx, *metadataDBPath, *titleID, *season)
	if err != nil {
		fatal("résolution saisons: %v", err)
	}
	if len(seasons) == 0 {
		fmt.Println("Aucune saison CSR trouvée dans csr_season_calendars.")
		return
	}
	fmt.Printf("Saisons ciblées (%d) pour xuid %s : %s\n", len(seasons), *xuid, strings.Join(seasons, ", "))

	if *dryRun {
		fmt.Println("[dry-run] aucun appel API, aucune écriture.")
		return
	}

	// Player DB en RW (cible des snapshots).
	playerDB, err := sql.Open("duckdb", *playerDBPath)
	if err != nil {
		fatal("open player DB: %v", err)
	}
	defer playerDB.Close()
	playerDB.SetMaxOpenConns(1)

	// Tokens Halo (canonique ADR 0023 : MultiUserTokenStore, source unique).
	tokens, err := loadHaloTokens(ctx, *watcherTokensDir, *xuid, *gamertag)
	if err != nil {
		fatal("chargement tokens Halo: %v", err)
	}
	client := syncpkg.NewHaloAPIClient(tokens.SpartanToken, tokens.ClearanceToken, ratePerSecond(*rateLimit))

	t0 := time.Now()
	total, err := syncpkg.BackfillCSRHistory(ctx, client, playerDB, *xuid, seasons)
	if err != nil {
		fatal("backfill: %v", err)
	}
	slog.InfoContext(ctx, "backfill CSR history terminé",
		"xuid", *xuid, "gamertag", *gamertag, "seasons", len(seasons),
		"snapshots", total, "duration", time.Since(t0).String())
	fmt.Printf("\nTerminé : %d snapshots CSR écrits sur %d saisons.\n", total, len(seasons))
}

// ratePerSecond convertit un débit/minute en requêtes/seconde (min 1).
func ratePerSecond(perMinute int) int {
	rps := perMinute / 60
	if rps < 1 {
		rps = 1
	}
	return rps
}

// resolveSeasons retourne la liste des season_id de csr_season_calendars (triés
// récent d'abord), ou [only] si only est non vide.
func resolveSeasons(ctx context.Context, metadataDBPath, titleID, only string) ([]string, error) {
	if s := strings.TrimSpace(only); s != "" {
		return []string{s}, nil
	}
	meta, err := sql.Open("duckdb", metadataDBPath+"?access_mode=read_only")
	if err != nil {
		return nil, fmt.Errorf("open metadata RO: %w", err)
	}
	defer meta.Close()
	rows, err := meta.QueryContext(ctx,
		`SELECT season_id FROM csr_season_calendars WHERE title_id = ? AND season_id IS NOT NULL AND season_id != '' ORDER BY start_date DESC`,
		titleID)
	if err != nil {
		return nil, fmt.Errorf("query csr_season_calendars: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out, rows.Err()
}

// loadHaloTokens suit le pipeline canonique ADR 0023 : MultiUserTokenStore,
// source unique des refresh tokens. La rotation RT est persistée par le helper.
func loadHaloTokens(ctx context.Context, watcherTokensDir, xuid, gamertag string) (*authTokens, error) {
	store := auth.NewMultiUserTokenStore(watcherTokensDir)
	provider := auth.NewSISUProvider()

	result, err := auth.RefreshHaloTokensViaStoreFirst(ctx, store, provider, xuid, gamertag)
	if err != nil {
		return nil, err
	}
	tokens := auth.HaloTokensFromExchange(result)
	if tokens == nil || strings.TrimSpace(tokens.SpartanToken) == "" {
		return nil, fmt.Errorf("aucun token Spartan obtenu (store=%s xuid=%s)", watcherTokensDir, xuid)
	}
	return &authTokens{SpartanToken: tokens.SpartanToken, ClearanceToken: tokens.ClearanceToken}, nil
}

type authTokens struct {
	SpartanToken   string
	ClearanceToken string
}

// loadEnvLocal injecte les variables de .env.local dans l'environnement (sans
// écraser celles déjà définies). Requis pour le secret client Azure et
// LEVELUP_OAUTH_CLIENT_ID, lus par ResolveAzureOAuthClient lors du refresh OAuth
// (pipeline SISUProvider + MultiUserTokenStore, ADR 0023 — MSALProvider a été
// supprimé le 2026-07-15 ; le nom exact de la variable du secret vit dans
// azure_credentials.go, la sentinelle du package auth interdit son littéral ici).
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

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", args...)
	os.Exit(1)
}
