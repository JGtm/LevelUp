// Package worldenrich assemble les dépendances de l'enrichissement du classement
// mondial (source de fetch Halo + résolveur xuid PeopleHub) à partir de la config,
// selon le chemin d'auth canonique store-first (ADR 0023) — exactement celui des
// backfills qui marchent (cf. cmd/backfill-csr-history). Mutualisé entre le CLI
// (cmd/backfill-world-player-stats) et le cron in-process (cmd/server, Phase C) :
// une seule implémentation de la résolution token → zéro divergence.
package worldenrich

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/config"
	title "levelup/go-api/internal/domain/title"
	auth "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/service"
	syncpkg "levelup/go-api/internal/sync"
)

// loadLegacyInputs lit le couple (refresh_token, msal_cache) du sync_meta de la
// player DB — fallback canonique ADR 0023 quand le watcher store ne couvre pas le
// joueur (cf. cmd/backfill-csr-history). Best-effort (vide si DB/clé absente).
func loadLegacyInputs(cfg *config.AppConfig, gamertag string) auth.LegacyAuthInputs {
	path := title.NewPathResolver(cfg.RepoRoot).PlayerDBPath(title.DefaultSlug, gamertag)
	db, err := sql.Open("duckdb", path+"?access_mode=read_only")
	if err != nil {
		return auth.LegacyAuthInputs{}
	}
	defer db.Close()
	var rt, msal string
	_ = db.QueryRowContext(context.Background(), `SELECT value FROM sync_meta WHERE key='oauth_refresh_token'`).Scan(&rt)
	_ = db.QueryRowContext(context.Background(), `SELECT value FROM sync_meta WHERE key='msal_token_cache'`).Scan(&msal)
	return auth.LegacyAuthInputs{OAuthRT: rt, MSALCache: msal, Source: "player_db.sync_meta"}
}

// resolveAccessToken obtient un access_token Microsoft frais pour xuid : store
// watcher_tokens d'abord (canonique ADR 0023), puis legacy sync_meta. Pour chaque
// source, MSAL silent puis refresh token (rotation persistée si store). C'est
// exactement la résolution des backfills qui marchent — pas de pool reconstruit.
func resolveAccessToken(ctx context.Context, provider auth.TokenProvider, store *auth.MultiUserTokenStore, xuid string, legacy auth.LegacyAuthInputs) (string, error) {
	var lastErr error // dernière erreur OAuth sous-jacente — surfacée pour le diagnostic
	try := func(msal, rt string, persist bool) string {
		if msal != "" {
			at, e := provider.TrySilentRefresh(ctx, msal)
			if e == nil && at != "" {
				return at
			}
			if e != nil {
				lastErr = e
			}
		}
		if rt != "" {
			at, rot, e := provider.TryOAuthRefreshWithRotation(ctx, rt)
			if e == nil && at != "" {
				if persist && rot != "" && rot != rt {
					_ = store.UpdateOAuthRefreshToken(xuid, rot)
				}
				return at
			}
			if e != nil {
				lastErr = e
			}
		}
		return ""
	}
	if user, _ := store.Load(xuid); user != nil {
		if at := try(user.MSALCacheJSON, user.OAuthRefreshToken, true); at != "" {
			return at, nil
		}
	}
	if at := try(legacy.MSALCache, legacy.OAuthRT, false); at != "" {
		return at, nil
	}
	if lastErr != nil {
		// Surface la cause réelle (ex. invalid_grant = RT minté par un autre client
		// Azure ou révoqué) plutôt que le générique — sinon le skip est indiagnosticable.
		return "", fmt.Errorf("aucun access_token frais pour xuid(%s): %w", xuid, lastErr)
	}
	return "", fmt.Errorf("aucun access_token frais pour xuid(%s) (aucun refresh token exploitable)", xuid)
}

// refreshingHaloSource : client Halo single-token (Spartan/Clearance) ré-résolu
// quand le TTL expire (Spartan ~4h) — supporte les runs longs sans pool. Implémente
// service.WorldMatchSource (GetMatchHistory + GetMatchStats).
type refreshingHaloSource struct {
	mu      sync.Mutex
	resolve func(ctx context.Context) (*syncpkg.HaloAPIClient, error)
	ttl     time.Duration
	now     func() time.Time
	client  *syncpkg.HaloAPIClient
	exp     time.Time
}

func (s *refreshingHaloSource) get(ctx context.Context) (*syncpkg.HaloAPIClient, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client != nil && s.now().Before(s.exp) {
		return s.client, nil
	}
	c, err := s.resolve(ctx)
	if err != nil {
		return nil, err
	}
	s.client, s.exp = c, s.now().Add(s.ttl)
	return c, nil
}

func (s *refreshingHaloSource) GetMatchHistory(ctx context.Context, gamertag, matchType string, start, count int) ([]syncpkg.MatchHistoryEntry, error) {
	c, err := s.get(ctx)
	if err != nil {
		return nil, err
	}
	return c.GetMatchHistory(ctx, gamertag, matchType, start, count)
}

func (s *refreshingHaloSource) GetMatchStats(ctx context.Context, matchID string) (map[string]any, error) {
	c, err := s.get(ctx)
	if err != nil {
		return nil, err
	}
	return c.GetMatchStats(ctx, matchID)
}

// BuildHaloSource résout le compte gamertag (store-first) et retourne un client Halo
// single-token auto-rafraîchi pour le fetch des matchs. Si eager, échoue tôt si le
// token est KO (avant d'ouvrir la base) ; sinon la résolution est différée au 1er
// usage (boot lazy du serveur — pas d'I/O token au démarrage).
func BuildHaloSource(cfg *config.AppConfig, gamertag string, rps int, eager bool) (service.WorldMatchSource, error) {
	xuid, err := xuidForGamertag(cfg, gamertag)
	if err != nil {
		return nil, err
	}
	provider := auth.NewMSALProvider()
	store := auth.NewMultiUserTokenStore(title.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())
	legacy := loadLegacyInputs(cfg, gamertag)
	src := &refreshingHaloSource{
		ttl: 3 * time.Hour, // Spartan ~4h, marge
		now: time.Now,
		resolve: func(ctx context.Context) (*syncpkg.HaloAPIClient, error) {
			at, e := resolveAccessToken(ctx, provider, store, xuid, legacy)
			if e != nil {
				return nil, e
			}
			exch, e := auth.ExchangeAccessToken(ctx, at)
			if e != nil {
				return nil, fmt.Errorf("exchange Halo: %w", e)
			}
			if exch.Tokens == nil || strings.TrimSpace(exch.Tokens.SpartanToken) == "" {
				return nil, fmt.Errorf("aucun Spartan token pour %s", gamertag)
			}
			return syncpkg.NewHaloAPIClient(exch.Tokens.SpartanToken, exch.Tokens.ClearanceToken, rps), nil
		},
	}
	if eager {
		if _, err := src.get(context.Background()); err != nil {
			return nil, fmt.Errorf("résolution token %s: %w", gamertag, err)
		}
	}
	return src, nil
}

// multiHaloSource round-robine le fetch sur N clients single-token (un par compte
// résolu). Chaque sous-source est résolue par le chemin prouvé (store-first + legacy)
// et s'auto-rafraîchit. NB : Halo limitant ~par IP, le gain est borné par le plafond
// IP, pas multiplié par N.
type multiHaloSource struct {
	sources []service.WorldMatchSource
	idx     uint64
}

func (m *multiHaloSource) next() service.WorldMatchSource {
	i := atomic.AddUint64(&m.idx, 1)
	return m.sources[int(i)%len(m.sources)]
}

func (m *multiHaloSource) GetMatchHistory(ctx context.Context, gamertag, matchType string, start, count int) ([]syncpkg.MatchHistoryEntry, error) {
	return m.next().GetMatchHistory(ctx, gamertag, matchType, start, count)
}

func (m *multiHaloSource) GetMatchStats(ctx context.Context, matchID string) (map[string]any, error) {
	return m.next().GetMatchStats(ctx, matchID)
}

// BuildMultiHaloSource résout TOUS les comptes db_profiles via le chemin prouvé et
// round-robine sur ceux qui réussissent (les autres — RT mintés par un autre client,
// ex. env-var .env.local — sont skippés avec un warn). Retourne les gamertags actifs.
func BuildMultiHaloSource(cfg *config.AppConfig, rps int, eager bool) (service.WorldMatchSource, []string, error) {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return nil, nil, fmt.Errorf("chargement db_profiles.json: %w", err)
	}
	var sources []service.WorldMatchSource
	var ok []string
	for _, p := range players {
		s, e := BuildHaloSource(cfg, p.Gamertag, rps, eager)
		if e != nil {
			slog.WarnContext(context.Background(), "world-enrich: compte skippé (token non résolu)",
				"gamertag", p.Gamertag, "err", e)
			continue
		}
		sources = append(sources, s)
		ok = append(ok, p.Gamertag)
	}
	if len(sources) == 0 {
		return nil, nil, fmt.Errorf("aucun compte db_profiles résolu")
	}
	return &multiHaloSource{sources: sources}, ok, nil
}

// BuildResolver construit le résolveur xuid single-compte : CHAÎNE PeopleHub→Profil
// Xbox (fix #10 — fallback universel hors graphe social), partageant le MÊME header
// XSTS dérivé du compte (access_token store-first → AcquireXSTSForRTA), mémoïsé (TTL).
func BuildResolver(cfg *config.AppConfig, tokenGamertag string) (XUIDResolver, error) {
	xuid, err := xuidForGamertag(cfg, tokenGamertag)
	if err != nil {
		return nil, err
	}
	provider := auth.NewMSALProvider()
	store := auth.NewMultiUserTokenStore(title.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())
	legacy := loadLegacyInputs(cfg, tokenGamertag)
	hp := auth.NewCachedHeaderProvider(0, func(ctx context.Context) (string, error) {
		at, e := resolveAccessToken(ctx, provider, store, xuid, legacy)
		if e != nil {
			return "", e
		}
		rta, e := auth.AcquireXSTSForRTA(ctx, at)
		if e != nil {
			return "", fmt.Errorf("AcquireXSTSForRTA: %w", e)
		}
		return fmt.Sprintf("XBL3.0 x=%s;%s", rta.UserHash, rta.Token), nil
	})
	return chainResolver{resolvers: []XUIDResolver{
		auth.NewPeopleHubResolver(nil, hp.Header),
		auth.NewXboxProfileResolver(nil, hp.Header),
	}}, nil
}

// BuildDirectoryResolver construit le résolveur live gamertag→xuid pour la
// recherche annuaire (Explorer + Face-à-face) : multi-comptes (round-robin
// anti-throttle, cf. BuildMultiResolver) enveloppé d'un cache process-wide
// (CachingResolver). LAZY : aucune I/O token au build (la résolution token est
// différée au 1er usage). Retourne une erreur si aucun compte db_profiles n'est
// exploitable (offline/démo) → le caller dégrade en recherche purement locale.
func BuildDirectoryResolver(cfg *config.AppConfig) (XUIDResolver, error) {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return nil, fmt.Errorf("BuildDirectoryResolver: chargement db_profiles.json: %w", err)
	}
	gts := make([]string, 0, len(players))
	for _, p := range players {
		gts = append(gts, p.Gamertag)
	}
	resolvers, ok, err := BuildMultiResolver(cfg, gts)
	if err != nil {
		return nil, err
	}
	slog.InfoContext(context.Background(), "directory live resolver construit (recherche joueur)",
		"comptes_resolus", len(ok))
	return NewCachingResolver(resolvers, nil, nil), nil
}

// xuidForGamertag résout l'xuid d'un gamertag depuis db_profiles.json.
func xuidForGamertag(cfg *config.AppConfig, gamertag string) (string, error) {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return "", fmt.Errorf("chargement db_profiles.json: %w", err)
	}
	for _, p := range players {
		if strings.EqualFold(p.Gamertag, gamertag) {
			if strings.TrimSpace(p.XUID) == "" {
				return "", fmt.Errorf("joueur %s sans xuid dans db_profiles.json", gamertag)
			}
			return p.XUID, nil
		}
	}
	return "", fmt.Errorf("gamertag %q introuvable dans db_profiles.json", gamertag)
}

// EnricherOptions paramètre BuildEnricher (convenience cron serveur).
type EnricherOptions struct {
	RPS           int    // requêtes/seconde par client (défaut 5)
	TokenGamertag string // compte token ; "" → multi (tous les comptes db_profiles)
	Eager         bool   // true : résout les tokens au build (fail-fast CLI) ; false : lazy (boot serveur)
}

// BuildEnricher compose source de fetch + résolveur xuid + enricher (ranked-only via
// RankedPlaylistSet). Convenience pour le cron in-process : en mode multi-token, le
// header PeopleHub est dérivé du 1er compte résolu (sauf TokenGamertag explicite).
// Retourne aussi la liste des gamertags de tokens actifs (pour le log de boot).
func BuildEnricher(cfg *config.AppConfig, opts EnricherOptions) (*service.WorldStatsEnricher, []string, error) {
	rps := opts.RPS
	if rps <= 0 {
		rps = 5
	}
	var (
		src service.WorldMatchSource
		gts []string
		err error
	)
	if strings.TrimSpace(opts.TokenGamertag) == "" {
		src, gts, err = BuildMultiHaloSource(cfg, rps, opts.Eager)
	} else {
		src, err = BuildHaloSource(cfg, opts.TokenGamertag, rps, opts.Eager)
		gts = []string{opts.TokenGamertag}
	}
	if err != nil {
		return nil, nil, err
	}
	resolverGT := strings.TrimSpace(opts.TokenGamertag)
	if resolverGT == "" && len(gts) > 0 {
		resolverGT = gts[0]
	}
	resolver, err := BuildResolver(cfg, resolverGT)
	if err != nil {
		return nil, nil, err
	}
	enr := service.NewWorldStatsEnricher(src, resolver, service.WorldStatsAggregatorConfig{
		RankedPlaylists: service.RankedPlaylistSet(),
	})
	return enr, gts, nil
}
