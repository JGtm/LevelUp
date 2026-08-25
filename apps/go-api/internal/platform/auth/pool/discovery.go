package pool

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
)

// Labels Source canoniques d'une CredentialSource.
// Identiques aux valeurs attendues par les tests resolver_test.go.
//
// ADR 0023 Phase 5 (2026-08-25) : les labels legacy (duckdb_msal, duckdb_oauth,
// env_oauth, watcher_legacy) ont disparu avec leurs sources — le scan ne peut
// plus produire qu'une seule provenance.
const (
	credSourceWatcherOAuth = "watcher_oauth" // ADR 0023 : MultiUserTokenStore — OAuth RT
)

// discoveryImpl scanne le MultiUserTokenStore (source unique ADR 0023) pour
// construire une liste de CredentialSource.
type discoveryImpl struct {
	cfg       *config.AppConfig
	resolver  *titlePkg.PathResolver
	titleSlug string

	// multiUserStore — data/auth/watcher_tokens/{xuid}.json : SEULE source de
	// credentials du pool depuis ADR 0023 Phase 5. Nil → aucun joueur découvert.
	multiUserStore *auth.MultiUserTokenStore
}

// NewDiscoveryWithStore crée un Discovery qui lit le MultiUserTokenStore
// (data/auth/watcher_tokens/{xuid}.json) — source unique des refresh tokens
// (ADR 0023). SEUL constructeur : le variant sans store a été supprimé en
// Phase 5, car privé de fallback il ne découvrait plus rien et faisait échouer
// silencieusement les appelants (régression `sync-delta --all`).
//
// cfg : configuration (contient db_profiles.json et RepoRoot)
// resolver : PathResolver pour les chemins par titre (évite filepath.Join direct)
// titleSlug : titre courant ("halo_infinite" par défaut)
// multiUserStore : peut être nil — le scan dégrade alors gracieusement (aucun
// joueur découvert), utile pour un appelant qui n'a délibérément pas de store.
func NewDiscoveryWithStore(
	cfg *config.AppConfig,
	resolver *titlePkg.PathResolver,
	titleSlug string,
	multiUserStore *auth.MultiUserTokenStore,
) Discovery {
	return &discoveryImpl{
		cfg:            cfg,
		resolver:       resolver,
		titleSlug:      titleSlug,
		multiUserStore: multiUserStore,
	}
}

// Scan implémente Discovery.Scan() — scanne le MultiUserTokenStore (source unique
// ADR 0023) pour chaque joueur déclaré.
func (d *discoveryImpl) Scan(ctx context.Context) ([]CredentialSource, error) {
	players, err := d.cfg.LoadPlayers()
	if err != nil {
		slog.ErrorContext(ctx, "pool: LoadPlayers échoué", "err", err)
		return nil, err
	}

	sources := make([]CredentialSource, 0, len(players))
	for _, player := range players {
		src := d.scanPlayer(ctx, player)
		if src == nil {
			continue
		}
		sources = append(sources, *src)
		recordScanSource(d.titleSlug, src.Gamertag, src.Source)
		slog.DebugContext(ctx, "pool: credential source découverte",
			"gamertag", src.Gamertag, "source", src.Source)
	}

	slog.InfoContext(ctx, "pool: scan terminé",
		"total_players_scanned", len(players),
		"players_with_token", len(sources))
	return sources, nil
}

// scanPlayer lit le refresh_token du joueur dans le MultiUserTokenStore (source
// unique ADR 0023, Phase 5 : plus aucun fallback sync_meta / env var / mono-user).
//
// Retourne nil si le store ne couvre pas le joueur (joueur skipped).
func (d *discoveryImpl) scanPlayer(ctx context.Context, player domain.PlayerSummary) *CredentialSource {
	if d.multiUserStore == nil || player.XUID == "" {
		slog.DebugContext(ctx, "pool: pas de store ou xuid absent — joueur exclu",
			"gamertag", player.Gamertag)
		return nil
	}
	ut, err := d.multiUserStore.Load(player.XUID)
	if err != nil || ut == nil || ut.OAuthRefreshToken == "" {
		slog.DebugContext(ctx, "pool: aucun refresh token store pour joueur — exclut",
			"gamertag", player.Gamertag, "xuid", player.XUID)
		return nil
	}

	return &CredentialSource{
		Gamertag:     player.Gamertag,
		TitleSlug:    d.titleSlug,
		XUID:         player.XUID,
		PlayerDBPath: d.resolver.PlayerDBPath(d.titleSlug, player.Gamertag),
		RefreshToken: ut.OAuthRefreshToken,
		Source:       credSourceWatcherOAuth,
	}
}
