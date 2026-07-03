package pool

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/duckdb"
)

// Labels Source canoniques d'une CredentialSource.
// Identiques aux valeurs attendues par les tests resolver_test.go.
const (
	credSourceDuckDBMSAL    = "duckdb_msal"
	credSourceDuckDBOAuth   = "duckdb_oauth"
	credSourceWatcherMSAL   = "watcher_msal"   // E.v1 : MultiUserTokenStore — MSAL cache (PR 2.5a)
	credSourceWatcherOAuth  = "watcher_oauth"  // ADR 0023 : MultiUserTokenStore — OAuth RT
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

// Scan implémente Discovery.Scan() — scanne MultiUserTokenStore (canonique post-ADR 0023)
// puis fallbacks DuckDB sync_meta + env var (DEPRECATED) + legacy mono-user store.
func (d *discoveryImpl) Scan(ctx context.Context) ([]CredentialSource, error) {
	players, err := d.cfg.LoadPlayers()
	if err != nil {
		slog.ErrorContext(ctx, "pool: LoadPlayers échoué", "err", err)
		return nil, err
	}

	// Le legacy mono-user TokenStore (data/auth/watcher_tokens.json) contient
	// UN SEUL RT qui appartient à UN seul joueur (le current_user du watcher
	// daemon historique). Attribué AU PLUS UNE FOIS au premier joueur sans
	// autre source — sinon mismatch xuid/token.
	legacyConsumed := false

	sources := make([]CredentialSource, 0, len(players))
	for _, player := range players {
		src := d.scanPlayer(ctx, player, &legacyConsumed)
		if src == nil {
			continue
		}
		sources = append(sources, *src)
		recordScanSource(d.titleSlug, src.Gamertag, src.Source)
		slog.DebugContext(ctx, "pool: credential source découverte",
			"gamertag", src.Gamertag, "source", src.Source,
			"has_msal", src.MSALCache != "", "has_oauth", src.RefreshToken != "")
	}

	slog.InfoContext(ctx, "pool: scan terminé",
		"total_players_scanned", len(players),
		"players_with_token", len(sources))
	return sources, nil
}

// scanPlayer applique la priorité ADR 0023 :
//  1. MultiUserTokenStore (RT + MSAL) — source canonique
//  2. sync_meta DuckDB (RT + MSAL) — DEPRECATED, warn log
//  3. env var (RT seulement) — DEPRECATED, warn log
//  4. mono-user legacy TokenStore (RT) — fallback final, attribuable une seule fois
//
// Retourne nil si aucune source ne donne rien (joueur skipped).
func (d *discoveryImpl) scanPlayer(ctx context.Context, player domain.PlayerSummary, legacyConsumed *bool) *CredentialSource {
	playerDBPath := d.resolver.PlayerDBPath(d.titleSlug, player.Gamertag)

	// --- Priorité 1 : MultiUserTokenStore (canonique post-ADR 0023) ---
	var msal, oauth, sourceLabel string
	if d.multiUserStore != nil && player.XUID != "" {
		if ut, _ := d.multiUserStore.Load(player.XUID); ut != nil {
			msal = ut.MSALCacheJSON
			oauth = ut.OAuthRefreshToken
			switch {
			case msal != "" && oauth != "":
				sourceLabel = credSourceWatcherMSAL + "+" + credSourceWatcherOAuth
			case msal != "":
				sourceLabel = credSourceWatcherMSAL
			case oauth != "":
				sourceLabel = credSourceWatcherOAuth
			}
		}
	}

	// --- Priorité 2 : sync_meta DuckDB (DEPRECATED) ---
	if msal == "" || oauth == "" {
		msal, oauth, sourceLabel = d.adoptLegacySyncMeta(ctx, player, playerDBPath, msal, oauth, sourceLabel)
	}

	// --- Priorité 3 : env var (DEPRECATED) ---
	if oauth == "" {
		if envToken := readOAuthRefreshTokenFromEnv(player.Gamertag); envToken != "" {
			slog.WarnContext(ctx, "pool: legacy env var utilisée — à migrer",
				"gamertag", player.Gamertag, "deprecated_since", "ADR-0023")
			observability.RecordLegacySourceUsed(observability.LegacySourceEnvOAuth)
			oauth = envToken
			sourceLabel = appendSource(sourceLabel, "env_oauth")
		}
	}

	// --- Priorité 4 : mono-user legacy store (fallback final) ---
	if msal == "" && oauth == "" && d.legacyStore != nil {
		if !*legacyConsumed {
			if st, _ := d.legacyStore.Load(); st != nil && st.RefreshToken != "" {
				oauth = st.RefreshToken
				sourceLabel = credSourceWatcherLegacy
				*legacyConsumed = true
				slog.WarnContext(ctx, "pool: legacy mono-user store attribué (approximation)",
					"gamertag", player.Gamertag,
					"hint", "configurer le store via token-capture pour éviter cette ambiguïté")
				observability.RecordLegacySourceUsed(observability.LegacySourceMonoUser)
			}
		}
	}

	if msal == "" && oauth == "" {
		slog.DebugContext(ctx, "pool: aucun token pour joueur — exclut",
			"gamertag", player.Gamertag)
		return nil
	}

	return &CredentialSource{
		Gamertag:     player.Gamertag,
		TitleSlug:    d.titleSlug,
		XUID:         player.XUID,
		PlayerDBPath: playerDBPath,
		MSALCache:    msal,
		RefreshToken: oauth,
		Source:       sourceLabel,
	}
}

// adoptLegacySyncMeta complète msal/oauth depuis sync_meta pour les champs que
// le store n'a pas fournis. Le warn « à migrer » n'est émis QUE si une valeur
// legacy est réellement ADOPTÉE (fix 2026-06-11 : l'ancien warn se déclenchait
// à la simple lecture, même quand le store couvrait déjà — bruit mensonger à
// chaque boot).
func (d *discoveryImpl) adoptLegacySyncMeta(
	ctx context.Context,
	player domain.PlayerSummary,
	playerDBPath, msal, oauth, sourceLabel string,
) (string, string, string) {
	dbMsal, dbOauth, ok := d.readLegacyDuckDB(ctx, player, playerDBPath)
	if !ok {
		return msal, oauth, sourceLabel
	}
	var adopted []string
	if msal == "" && dbMsal != "" {
		msal = dbMsal
		sourceLabel = appendSource(sourceLabel, credSourceDuckDBMSAL)
		adopted = append(adopted, "msal")
	}
	if oauth == "" && dbOauth != "" {
		oauth = dbOauth
		sourceLabel = appendSource(sourceLabel, credSourceDuckDBOAuth)
		adopted = append(adopted, "oauth")
	}
	if len(adopted) > 0 {
		slog.WarnContext(ctx, "pool: legacy sync_meta DuckDB utilisée — à migrer",
			"gamertag", player.Gamertag, "fields", strings.Join(adopted, "+"),
			"deprecated_since", "ADR-0023")
		for _, f := range adopted {
			if f == "msal" {
				observability.RecordLegacySourceUsed(observability.LegacySourceDuckDBMSAL)
			} else {
				observability.RecordLegacySourceUsed(observability.LegacySourceDuckDBOAuth)
			}
		}
	}
	return msal, oauth, sourceLabel
}

// readLegacyDuckDB lit msal+oauth depuis sync_meta. Ne logue PAS : c'est le
// caller (adoptLegacySyncMeta) qui warn, et uniquement si une valeur est adoptée.
func (d *discoveryImpl) readLegacyDuckDB(ctx context.Context, player domain.PlayerSummary, playerDBPath string) (msal, oauth string, ok bool) {
	// OpenReadForQuery (jamais OpenReadOnly nu) : réutilise un handle en cache si la
	// player DB est déjà tenue RW dans le process, au lieu de doubler l'ouverture RO
	// (erreur « different configuration », incident 2026-06-01) — E6, ADR 0016.
	playerDB, release, dbErr := duckdb.OpenReadForQuery(playerDBPath)
	if dbErr != nil {
		slog.DebugContext(ctx, "pool: PlayerDB introuvable — fallback sources externes",
			"gamertag", player.Gamertag, "db", playerDBPath)
		return "", "", false
	}
	defer release()

	msal, _ = duckdb.ReadMSALCacheJSONFromSQL(ctx, playerDB)
	oauth, _ = duckdb.ReadOAuthRefreshTokenFromSQL(ctx, playerDB)
	return msal, oauth, true
}

// appendSource concatène un label de source à la liste séparée par '+'.
func appendSource(current, add string) string {
	if current == "" {
		return add
	}
	return current + "+" + add
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
