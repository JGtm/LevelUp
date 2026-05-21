package pool

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
)

// Labels Source canoniques d'une CredentialSource.
// Identiques aux valeurs attendues par les tests resolver_test.go.
const (
	credSourceDuckDBMSAL  = "duckdb_msal"
	credSourceDuckDBOAuth = "duckdb_oauth"
)

// discoveryImpl scanne les sources de credentials (env + DuckDB) pour construire une liste de CredentialSource.
type discoveryImpl struct {
	cfg       *config.AppConfig
	resolver  *titlePkg.PathResolver
	titleSlug string
}

// NewDiscovery crée une nouvelle instance Discovery.
// cfg : configuration (contient db_profiles.json et RepoRoot)
// resolver : PathResolver pour accéder aux chemins DuckDB via titleSlug (évite filepath.Join direct)
// titleSlug : titre courant ("halo_infinite" par défaut)
func NewDiscovery(cfg *config.AppConfig, resolver *titlePkg.PathResolver, titleSlug string) Discovery {
	return &discoveryImpl{
		cfg:       cfg,
		resolver:  resolver,
		titleSlug: titleSlug,
	}
}

// Scan implémente Discovery.Scan() — scanne env + DuckDB pour chaque joueur.
func (d *discoveryImpl) Scan(ctx context.Context) ([]CredentialSource, error) {
	players, err := d.cfg.LoadPlayers()
	if err != nil {
		slog.ErrorContext(ctx, "pool: LoadPlayers échoué", "err", err)
		return nil, err
	}

	sources := make([]CredentialSource, 0, len(players))

	for _, player := range players {
		playerDBPath := d.resolver.PlayerDBPath(d.titleSlug, player.Gamertag)

		// Ouvrir la DB player en read-only pour lire sync_meta.
		playerDB, err := duckdb.OpenReadOnly(playerDBPath)
		if err != nil {
			// DB inexistante pour ce joueur — skip silencieux (normal, joueur peut ne pas être synced).
			slog.DebugContext(ctx, "pool: PlayerDB introuvable — skip joueur",
				"gamertag", player.Gamertag, "db", playerDBPath)
			continue
		}

		// Lire MSAL cache et OAuth refresh token depuis sync_meta.
		msal, _ := duckdb.ReadMSALCacheJSON(ctx, playerDB)
		oauth, _ := duckdb.ReadOAuthRefreshToken(ctx, playerDB)
		_ = playerDB.Close() // best-effort fermeture

		// Fallback env var SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG> si pas de token en DuckDB.
		envToken := ""
		if oauth == "" {
			envToken = readOAuthRefreshTokenFromEnv(player.Gamertag)
			if envToken != "" {
				oauth = envToken
			}
		}

		// Exclure les joueurs sans aucun token.
		if msal == "" && oauth == "" {
			slog.DebugContext(ctx, "pool: aucun token pour joueur — exclut",
				"gamertag", player.Gamertag)
			continue
		}

		// Construire CredentialSource.
		source := CredentialSource{
			Gamertag:     player.Gamertag,
			XUID:         player.XUID,
			PlayerDBPath: playerDBPath,
			MSALCache:    msal,
			RefreshToken: oauth,
		}

		// Déterminer la source exacte pour logs.
		if msal != "" && oauth != "" {
			source.Source = credSourceDuckDBMSAL + "+" + credSourceDuckDBOAuth
		} else if msal != "" {
			source.Source = credSourceDuckDBMSAL
		} else if envToken != "" {
			source.Source = "env_oauth"
		} else {
			source.Source = credSourceDuckDBOAuth
		}

		sources = append(sources, source)
		slog.DebugContext(ctx, "pool: credential source découverte",
			"gamertag", player.Gamertag, "source", source.Source, "has_msal", msal != "", "has_oauth", oauth != "")
	}

	slog.InfoContext(ctx, "pool: scan terminé",
		"total_players_scanned", len(players),
		"players_with_token", len(sources))

	return sources, nil
}

// readOAuthRefreshTokenFromEnv retourne la valeur de SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG>.
// Transforme le gamertag en clé uppercase avec espaces/tirets/points → underscores.
func readOAuthRefreshTokenFromEnv(gamertag string) string {
	key := strings.ToUpper(strings.TrimSpace(gamertag))
	key = strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' || r == '.' {
			return '_'
		}
		return r
	}, key)
	return os.Getenv("SPNKR_OAUTH_REFRESH_TOKEN_" + key)
}
