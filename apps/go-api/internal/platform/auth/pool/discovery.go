package pool

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/duckdb"
)

// Labels Source canoniques d'une CredentialSource.
// Identiques aux valeurs attendues par les tests resolver_test.go.
const (
	credSourceDuckDBMSAL    = "duckdb_msal"
	credSourceDuckDBOAuth   = "duckdb_oauth"
	credSourceWatcherMSAL   = "watcher_msal"   // E.v1 : MultiUserTokenStore (PR 2.5a)
	credSourceWatcherLegacy = "watcher_legacy" // E.v1 : mono-user TokenStore (legacy)
)

// discoveryImpl scanne les sources de credentials (env + DuckDB + watcher stores)
// pour construire une liste de CredentialSource.
type discoveryImpl struct {
	cfg       *config.AppConfig
	resolver  *titlePkg.PathResolver
	titleSlug string

	// multiUserStore (E.v1, optionnel) — si défini, Scan lit aussi
	// data/auth/watcher_tokens/{xuid}.json pour récupérer le MSAL cache
	// frais maintenu par le watcher daemon. Permet de peupler le pool
	// au 1er boot sans attendre un sync manuel qui aurait écrit dans
	// sync_meta DuckDB.
	multiUserStore *auth.MultiUserTokenStore

	// legacyStore (E.v1, optionnel) — pareil pour le store mono-user
	// data/auth/watcher_tokens.json. Lit le refresh_token OAuth v2 si
	// présent. Pertinent tant que la migration PR 2.5b n'est pas livrée.
	legacyStore *auth.TokenStore
}

// NewDiscovery crée une nouvelle instance Discovery.
// cfg : configuration (contient db_profiles.json et RepoRoot)
// resolver : PathResolver pour accéder aux chemins DuckDB via titleSlug (évite filepath.Join direct)
// titleSlug : titre courant ("halo_infinite" par défaut)
//
// Pour activer la lecture des watcher token stores (E.v1), utiliser
// NewDiscoveryWithStores à la place.
func NewDiscovery(cfg *config.AppConfig, resolver *titlePkg.PathResolver, titleSlug string) Discovery {
	return &discoveryImpl{
		cfg:       cfg,
		resolver:  resolver,
		titleSlug: titleSlug,
	}
}

// NewDiscoveryWithStores crée un Discovery qui lit aussi les watcher token
// stores (E.v1 du refactor auth unification). Permet de peupler le pool au
// 1er boot avec le MSAL frais maintenu par le watcher daemon, sans dépendre
// d'un sync manuel ayant écrit dans sync_meta DuckDB.
//
//   - multiUserStore : data/auth/watcher_tokens/{xuid}.json (PR 2.5a)
//   - legacyStore    : data/auth/watcher_tokens.json (mono-user historique)
//
// Soit l'un OU l'autre OU les deux peuvent être nil — degrade gracieusement.
func NewDiscoveryWithStores(
	cfg *config.AppConfig,
	resolver *titlePkg.PathResolver,
	titleSlug string,
	multiUserStore *auth.MultiUserTokenStore,
	legacyStore *auth.TokenStore,
) Discovery {
	return &discoveryImpl{
		cfg:            cfg,
		resolver:       resolver,
		titleSlug:      titleSlug,
		multiUserStore: multiUserStore,
		legacyStore:    legacyStore,
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

		// Tenter d'ouvrir la DB player en read-only pour lire sync_meta.
		// Échec d'ouverture = DB inexistante (normal pour un joueur jamais
		// synced) — on continue sans erreur et on tentera les autres sources.
		var msal, oauth string
		if playerDB, dbErr := duckdb.OpenReadOnly(playerDBPath); dbErr == nil {
			msal, _ = duckdb.ReadMSALCacheJSON(ctx, playerDB)
			oauth, _ = duckdb.ReadOAuthRefreshToken(ctx, playerDB)
			_ = playerDB.Close() // best-effort fermeture
		} else {
			slog.DebugContext(ctx, "pool: PlayerDB introuvable — fallback sources externes",
				"gamertag", player.Gamertag, "db", playerDBPath)
		}

		// Fallback env var SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG> si pas de token en DuckDB.
		envToken := ""
		if oauth == "" {
			envToken = readOAuthRefreshTokenFromEnv(player.Gamertag)
			if envToken != "" {
				oauth = envToken
			}
		}

		// E.v1 — Fallback watcher stores si toujours rien.
		// Le watcher daemon entretient un MSAL cache frais (refresh proactif
		// chaque ~5min). On le lit ici pour peupler le pool au 1er boot sans
		// dépendre d'un sync manuel ayant écrit dans sync_meta DuckDB.
		watcherSource := ""
		if msal == "" && oauth == "" {
			// MultiUserTokenStore (data/auth/watcher_tokens/{xuid}.json)
			if d.multiUserStore != nil && player.XUID != "" {
				if ut, _ := d.multiUserStore.Load(player.XUID); ut != nil && ut.MSALCacheJSON != "" {
					msal = ut.MSALCacheJSON
					watcherSource = credSourceWatcherMSAL
				}
			}
			// Mono-user legacy (data/auth/watcher_tokens.json) — n'a qu'un
			// seul user. Match approximatif via le RefreshToken disponible
			// dans le store ; on l'attribue au premier joueur sans token.
			if msal == "" && oauth == "" && d.legacyStore != nil {
				if st, _ := d.legacyStore.Load(); st != nil && st.RefreshToken != "" {
					oauth = st.RefreshToken
					watcherSource = credSourceWatcherLegacy
				}
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
		switch {
		case watcherSource != "":
			source.Source = watcherSource
		case msal != "" && oauth != "":
			source.Source = credSourceDuckDBMSAL + "+" + credSourceDuckDBOAuth
		case msal != "":
			source.Source = credSourceDuckDBMSAL
		case envToken != "":
			source.Source = "env_oauth"
		default:
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
