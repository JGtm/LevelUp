//go:build cgo

// cmd/probe-service-record — sonde live (LECTURE SEULE) pour valider B2 : l'endpoint
// service record matchmade accepte-t-il le filtre `playlistAssetId` et renvoie-t-il
// les CoreStats complets par (saison, playlist) ? Aucun INSERT. Réutilise le harnais
// d'auth de cmd/probe-world-stats.
//
// Usage (depuis apps/go-api/) :
//
//	go run ./cmd/probe-service-record \
//	  --shared-db  ../../data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb \
//	  --tokens-dir ../../data/auth/watcher_tokens \
//	  --env-file   ../../.env.local \
//	  --token-xuid 2533274823110022 --token-gamertag JGtm
package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
	syncpkg "levelup/go-api/internal/sync"
)

func main() {
	sharedDB := flag.String("shared-db", "", "shared_matches_v2.duckdb (RO) — lit saison+playlist courantes si absentes")
	tokensDir := flag.String("tokens-dir", "data/auth/watcher_tokens", "MultiUserTokenStore")
	envFile := flag.String("env-file", ".env.local", ".env.local (SPNKR_AZURE_CLIENT_ID)")
	tokenXUID := flag.String("token-xuid", "", "XUID porteur de tokens — requis")
	tokenGamertag := flag.String("token-gamertag", "", "gamertag porteur (logs)")
	season := flag.String("season", "", "seasonId (défaut = season_id du dernier snapshot)")
	playlist := flag.String("playlist", "", "playlistAssetId (défaut = 1re playlist du dernier snapshot)")
	target := flag.String("xuid", "", "xuid du joueur sondé (défaut = --token-xuid)")
	flag.Parse()

	loadEnv(*envFile)
	ctx := context.Background()
	if strings.TrimSpace(*tokenXUID) == "" {
		fatal("--token-xuid requis")
	}
	if strings.TrimSpace(*target) == "" {
		*target = *tokenXUID
	}
	if (*season == "" || *playlist == "") && strings.TrimSpace(*sharedDB) != "" {
		s, p := readCurrentSeasonPlaylist(*sharedDB)
		if *season == "" {
			*season = s
		}
		if *playlist == "" {
			*playlist = p
		}
	}
	fmt.Printf("[probe] xuid=%s season=%q playlist=%q\n", *target, *season, *playlist)

	client := buildClient(ctx, *tokensDir, *tokenXUID, *tokenGamertag)

	recAll, errAll := client.GetSeasonPlaylistServiceRecord(ctx, *target, *season, "")
	fmt.Printf("[probe] saison SEULE    -> %s\n", describe(recAll, errAll))

	recPl, errPl := client.GetSeasonPlaylistServiceRecord(ctx, *target, *season, *playlist)
	fmt.Printf("[probe] saison+PLAYLIST -> %s\n", describe(recPl, errPl))

	if recPl != nil {
		b, _ := json.MarshalIndent(recPl, "", "  ")
		fmt.Printf("[probe] CoreStats par playlist (B2 VALIDÉ) :\n%s\n", b)
	}
}

func describe(r *domain.WorldServiceRecord, err error) string {
	if err != nil {
		return "ERREUR: " + err.Error()
	}
	if r == nil {
		return "(nil) — 404/403 ou 0 match"
	}
	return fmt.Sprintf("matches=%d wins=%d kills=%d deaths=%d assists=%d shots=%.0f/%.0f medals=%d",
		r.MatchesCompleted, r.Wins, r.Kills, r.Deaths, r.Assists, r.ShotsHit, r.ShotsFired, r.MedalCount)
}

func readCurrentSeasonPlaylist(path string) (string, string) {
	db, err := sql.Open("duckdb", path+"?access_mode=read_only")
	if err != nil {
		fmt.Printf("[probe] open shared (RO) échoué: %v\n", err)
		return "", ""
	}
	defer db.Close()
	var season, playlist string
	_ = db.QueryRow(`SELECT season_id, playlist_id FROM world_csr_leaderboard_snapshots
		WHERE title_slug='halo_infinite' AND playlist_id<>''
		ORDER BY fetched_at DESC LIMIT 1`).Scan(&season, &playlist)
	return season, playlist
}

func buildClient(ctx context.Context, tokensDir, xuid, gt string) *syncpkg.HaloAPIClient {
	store := auth.NewMultiUserTokenStore(tokensDir)
	bearer, err := store.Load(xuid)
	if err != nil {
		fatal("chargement tokens %s : %v", xuid, err)
	}
	var accessToken string
	if bearer.MSALCacheJSON != "" {
		accessor := auth.NewInMemoryCacheAccessorFromJSON(bearer.MSALCacheJSON)
		if at, _ := auth.AcquireTokenSilent(ctx, accessor); at != "" {
			accessToken = at
		}
	}
	if accessToken == "" && bearer.OAuthRefreshToken != "" {
		at, rotatedRT, rerr := auth.ExchangeRefreshTokenWithRotation(ctx, bearer.OAuthRefreshToken)
		if rerr != nil {
			fatal("refresh RT (%s) : %v", xuid, rerr)
		}
		accessToken = at
		// ANTI-CHURN CRITIQUE : le RT est à usage unique — persister le RT roté, sinon
		// le prochain run réutilise un RT consommé → invalid_grant (incident probe
		// 2026-06-10 ; ADR 0023). Ne JAMAIS omettre cette persistance dans une sonde.
		if rotatedRT != "" && rotatedRT != bearer.OAuthRefreshToken {
			bearer.OAuthRefreshToken = rotatedRT
			if uerr := store.Upsert(bearer); uerr != nil {
				fmt.Printf("[auth] WARN: persistance RT roté échouée: %v\n", uerr)
			}
		}
	}
	if strings.TrimSpace(accessToken) == "" {
		fatal("aucun access_token frais pour %s (ADR 0023 : ne PAS re-capturer sans diagnostic)", xuid)
	}
	res, err := auth.ExchangeAccessToken(ctx, accessToken)
	if err != nil {
		fatal("ExchangeAccessToken : %v", err)
	}
	if res.Tokens == nil || strings.TrimSpace(res.Tokens.SpartanToken) == "" {
		fatal("aucun Spartan token pour %s", xuid)
	}
	fmt.Printf("[auth] tokens Halo OK pour %s (xuid %s)\n", gt, xuid)
	return syncpkg.NewHaloAPIClient(res.Tokens.SpartanToken, res.Tokens.ClearanceToken, 5)
}

func loadEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		_ = os.Setenv(strings.TrimSpace(k), strings.Trim(strings.TrimSpace(v), `"'`))
	}
}

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", a...)
	os.Exit(1)
}
