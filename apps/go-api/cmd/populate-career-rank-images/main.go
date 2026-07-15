// cmd/populate-career-rank-images — One-shot CLI pour télécharger toutes les
// images de rang carrière depuis Halo GameCMS et les persister sous
// data/cache/career-rank-image/{titleID}/... pour bundling repo.
//
// Le but : remplir le cache local en une passe (avec auth Spartan), puis
// commit le dossier dans le repo pour que les collègues n'aient pas besoin
// de tokens Halo valides — git pull suffit, le serveur sert depuis le FS.
//
// Usage :
//
//	populate-career-rank-images --player JGtm
//	populate-career-rank-images --player JGtm --variants large,icon,adornment
//	SPARTAN_TOKEN=xxx populate-career-rank-images
//
// Variants :
//
//	large     → career_ranks.large_icon_path  (CelebrationMoment, ~120px)
//	icon      → career_ranks.icon_path        (ProgressWidget, ~64px)
//	adornment → career_ranks.adornment_icon_path (NameplateAdornment, ~32px)
//
// Par défaut, fetch uniquement "large" (utilisé par la page Carrière).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	authpkg "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/duckdb"
)

type rankRow struct {
	rankID    int
	pathLarge string
	pathIcon  string
	pathAdorn string
}

func main() {
	fs := flag.NewFlagSet("populate-career-rank-images", flag.ExitOnError)
	titleID := fs.String("title-id", titlePkg.DefaultSlug, "Title ID (ex: halo_infinite)")
	player := fs.String("player", "", "Slug joueur pour résoudre les tokens (ignoré si SPARTAN_TOKEN env est défini)")
	variants := fs.String("variants", "large", "Variantes à fetcher (CSV : large,icon,adornment)")
	maxFlag := fs.Int("max", 0, "Nombre max de rangs à traiter (0 = tous)")
	timeoutFlag := fs.Int("timeout", 30, "Timeout par fetch en secondes")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(1)
	}

	if err := run(*titleID, *player, parseVariants(*variants), *maxFlag, *timeoutFlag); err != nil {
		fmt.Fprintf(os.Stderr, "erreur : %v\n", err)
		os.Exit(1)
	}
}

func run(titleID, playerSlug string, variants []string, maxRanks, timeoutSec int) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config.Load: %w", err)
	}
	if cfg.RepoRoot == "" {
		return fmt.Errorf("LEVELUP_REPO_ROOT non défini")
	}

	ctx := context.Background()

	tokens, err := resolveTokens(ctx, cfg, playerSlug)
	if err != nil {
		return fmt.Errorf("résolution tokens: %w", err)
	}

	rows, err := loadCareerRanks(ctx, cfg, titleID)
	if err != nil {
		return fmt.Errorf("load career_ranks: %w", err)
	}
	if len(rows) == 0 {
		return fmt.Errorf("aucun rang trouvé dans career_ranks")
	}
	if maxRanks > 0 && len(rows) > maxRanks {
		rows = rows[:maxRanks]
	}
	fmt.Printf("→ %d rangs à traiter, variantes %v, timeout %ds/fetch\n", len(rows), variants, timeoutSec)

	df := &directFetcher{tokens: tokens}
	fetchTimeout := time.Duration(timeoutSec) * time.Second

	cacheRoot := filepath.Join(cfg.RepoRoot, "data", "cache", "career-rank-image", titleID)
	stats := struct{ skipped, fetched, failed int }{}

	for _, row := range rows {
		for _, v := range variants {
			path := pathForVariant(row, v)
			if path == "" {
				continue
			}
			target := filepath.Join(cacheRoot, filepath.FromSlash(path))
			if _, err := os.Stat(target); err == nil {
				stats.skipped++
				continue
			}
			start := time.Now()
			if err := fetchAndPersist(ctx, df, titleID, path, target, fetchTimeout); err != nil {
				fmt.Fprintf(os.Stderr, "  ✗ rank %d (%s) [%dms]: %v\n", row.rankID, v, time.Since(start).Milliseconds(), err)
				stats.failed++
				continue
			}
			fmt.Printf("  ✓ rank %d %s [%dms] → %s\n", row.rankID, v, time.Since(start).Milliseconds(), path)
			stats.fetched++
		}
	}

	fmt.Printf("\n✅ Terminé. Fetched=%d, Skipped(déjà en cache)=%d, Failed=%d\n",
		stats.fetched, stats.skipped, stats.failed)
	if stats.failed > 0 {
		return fmt.Errorf("%d échecs (probablement upstream Halo down ou token invalide)", stats.failed)
	}
	return nil
}

func loadCareerRanks(ctx context.Context, cfg *config.AppConfig, titleID string) ([]rankRow, error) {
	metaPath := titlePkg.NewPathResolver(cfg.RepoRoot).MetadataDBPath(titleID)
	metaDB, err := duckdb.OpenReadWriteShared(metaPath)
	if err != nil {
		return nil, fmt.Errorf("ouverture metadata.duckdb: %w", err)
	}
	defer metaDB.Close()

	rows, err := metaDB.Query(ctx, `
		SELECT rank_id,
		       COALESCE(NULLIF(TRIM(large_icon_path), ''), '')        AS large_icon_path,
		       COALESCE(NULLIF(TRIM(icon_path), ''), '')              AS icon_path,
		       COALESCE(NULLIF(TRIM(adornment_icon_path), ''), '')    AS adornment_path
		FROM career_ranks
		ORDER BY rank_id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query career_ranks: %w", err)
	}
	defer rows.Close()

	var out []rankRow
	for rows.Next() {
		var r rankRow
		if err := rows.Scan(&r.rankID, &r.pathLarge, &r.pathIcon, &r.pathAdorn); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func pathForVariant(r rankRow, v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "large":
		return r.pathLarge
	case "icon":
		return r.pathIcon
	case "adornment", "adorn":
		return r.pathAdorn
	}
	return ""
}

func parseVariants(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// directFetcher fait l'appel via curl.exe Windows natif (Schannel + trust store
// Windows). Le client HTTP Go (net/http + crypto/tls) hang sur les requêtes
// authentifiées vers gamecms-hacs.svc.halowaypoint.com (cause inconnue, peut-
// être stack TLS/HTTP2 vs Spartan token volumineux). curl bypass le problème
// avec une latence de 500-900ms par image.
//
// Important : utilise C:\Windows\System32\curl.exe (pas le curl Git Bash /
// MSYS2 qui utilise OpenSSL sans la CA Microsoft → ERR 60).
type directFetcher struct {
	tokens *domain.HaloTokens
}

func (d *directFetcher) fetch(ctx context.Context, path string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	url := "https://gamecms-hacs.svc.halowaypoint.com/hi/images/file/" + path
	args := []string{
		"-sS", "--fail-with-body",
		"--max-time", fmt.Sprintf("%.0f", timeout.Seconds()),
		"-H", "x-343-authorization-spartan: " + d.tokens.SpartanToken,
		"-H", "343-clearance: " + d.tokens.ClearanceToken,
		"-H", "Accept: image/png,image/*;q=0.9,*/*;q=0.5",
		"-H", "User-Agent: LevelUp-populate-career-rank-images/1.0",
		"-o", "-",
		url,
	}
	// Forcer le curl.exe Windows natif (Schannel + trust store Windows)
	// au lieu du curl Git Bash / MSYS2 (OpenSSL sans CA Microsoft).
	curlBin := `C:\Windows\System32\curl.exe`
	if _, err := os.Stat(curlBin); err != nil {
		curlBin = "curl.exe" // fallback PATH
	}
	cmd := exec.CommandContext(ctx, curlBin, args...)
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return nil, fmt.Errorf("curl exit %d: %s", ee.ExitCode(), strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, fmt.Errorf("curl: %w", err)
	}
	if len(out) < 8 {
		return nil, fmt.Errorf("réponse trop courte (%d bytes)", len(out))
	}
	// Vérifier signature PNG (89 50 4E 47 0D 0A 1A 0A).
	if out[0] != 0x89 || out[1] != 0x50 || out[2] != 0x4E || out[3] != 0x47 {
		return nil, fmt.Errorf("réponse pas un PNG (commence par %x)", out[:8])
	}
	return out, nil
}

func fetchAndPersist(ctx context.Context, df *directFetcher, _, path, target string, timeout time.Duration) error {
	data, err := df.fetch(ctx, path, timeout)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// resolveTokens — pattern identique à refresh-career-ranks.
func resolveTokens(ctx context.Context, cfg *config.AppConfig, playerSlug string) (*domain.HaloTokens, error) {
	if envToken := os.Getenv("SPARTAN_TOKEN"); envToken != "" {
		return &domain.HaloTokens{
			SpartanToken:   envToken,
			ClearanceToken: os.Getenv("CLEARANCE_TOKEN"),
		}, nil
	}
	if playerSlug == "" {
		return nil, errors.New("SPARTAN_TOKEN absent ET --player non fourni — précisez l'un des deux")
	}

	pdb, err := config.ResolvePlayer(ctx, cfg, playerSlug, titlePkg.DefaultSlug)
	if err != nil {
		return nil, fmt.Errorf("résoudre player %q: %w", playerSlug, err)
	}

	provider := authpkg.NewSISUProvider()

	// ADR 0023 — pipeline canonique via MultiUserTokenStore puis legacy DuckDB/env.
	store := authpkg.NewMultiUserTokenStore(titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())
	legacy := authpkg.LegacyAuthInputs{Source: "duckdb_or_env"}
	legacy.MSALCache, _ = duckdb.ReadMSALCacheJSON(ctx, pdb.Player)
	legacy.OAuthRT, _ = duckdb.ReadOAuthRefreshToken(ctx, pdb.Player)
	if legacy.OAuthRT == "" {
		legacy.OAuthRT = envRefreshTokenForGamertag(pdb.Gamertag)
	}

	result, rerr := authpkg.RefreshHaloTokensViaStoreFirst(ctx, store, provider, pdb.XUID, pdb.Gamertag, legacy)
	if rerr != nil {
		return nil, rerr
	}
	if tokens := authpkg.HaloTokensFromExchange(result); tokens != nil {
		fmt.Fprintf(os.Stderr, "auth: tokens obtenus pour xuid=%s\n", pdb.XUID)
		return tokens, nil
	}
	return nil, fmt.Errorf("aucun token disponible pour player %q (ni MSAL cache, ni OAuth refresh DB, ni env SPNKR_OAUTH_REFRESH_TOKEN)", playerSlug)
}

// envRefreshTokenForGamertag lit SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG_UPPER>.
// Convention identique à internal/api/registry.go::oauthRefreshTokenForPlayer.
func envRefreshTokenForGamertag(gamertag string) string {
	if gamertag == "" {
		return ""
	}
	return os.Getenv("SPNKR_OAUTH_REFRESH_TOKEN_" + normalizeGamertagKey(gamertag))
}

func normalizeGamertagKey(gamertag string) string {
	key := strings.ToUpper(gamertag)
	return strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' || r == '.' {
			return '_'
		}
		return r
	}, key)
}
