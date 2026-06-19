// Package api — registry_catalog_drain.go : runner de l'action « drain
// DiscoveryUGC » (job asynchrone). Recense les asset IDs vus en jeu
// (catalog_fetch_queue) puis les résout via l'API DiscoveryUGC (réseau,
// rate-limité). Complète le mode from_match_registry (zéro réseau) : ici on
// hydrate les assets dont le nom n'est PAS dans match_registry.
//
// In-process, fidèle au CLI populate-playlists-catalog (chaîne provider →
// fetcher → adapter → resolver → CatalogFetcherService.Drain). Spécifique Halo
// Infinite — comme ops/catalog_refresh.go. Single-flight + tokens chargés via
// le MultiUserTokenStore (ADR 0023, n'importe quel joueur authentifié).
package api

import (
	"context"
	"fmt"
	"path/filepath"
	gosync "sync"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	halo_games "levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/halo"
	"levelup/go-api/internal/service"
)

// catalogDrainMu sérialise le drain UGC (single-flight).
var catalogDrainMu gosync.Mutex

const (
	// catalogDrainRateLimit borne les requêtes vers Discovery UGC (req/min).
	catalogDrainRateLimit = 60
	// catalogDrainMaxPasses : garde-fou de boucle (chaque passe vide ce qu'elle
	// peut ; on s'arrête dès qu'une passe ne progresse plus).
	catalogDrainMaxPasses = 10
)

// RunCatalogUGCDrain seed la file depuis match_registry puis la draine via
// l'API DiscoveryUGC. Bloquant (réseau) — appelé dans la goroutine d'un job.
func (r *ServiceRegistry) RunCatalogUGCDrain(ctx context.Context, titleSlug string) (domain.CatalogUGCDrainResult, error) {
	var res domain.CatalogUGCDrainResult
	if !catalogDrainMu.TryLock() {
		return res, ErrActionBusy
	}
	defer catalogDrainMu.Unlock()

	sharedSQL, metaSQL, closeAll, err := r.dataQualityHandles(titleSlug)
	if err != nil {
		return res, err
	}
	defer closeAll()
	if metaSQL == nil {
		return res, fmt.Errorf("metadata indisponible pour %s", titleSlug)
	}

	// 1. Recensement (zéro réseau).
	seeded, err := ops.SeedCatalogQueueFromRegistry(ctx, metaSQL, sharedSQL, titleSlug)
	if err != nil {
		return res, fmt.Errorf("seed queue: %w", err)
	}
	res.Seeded = seeded.Total()

	// 2. Tokens Halo (DiscoveryUGC nécessite un Spartan token valide).
	tokens, err := r.haloTokensForDrain(ctx, titleSlug)
	if err != nil {
		return res, err
	}

	// 3. Chaîne de résolution (fidèle au CLI populate-playlists-catalog).
	provider := halo.NewHaloProvider().WithRateLimit(catalogDrainRateLimit).WithTokens(tokens)
	fetcher := halo.NewCatalogFetcher(provider)
	rulesPath := filepath.Join(r.cfg.RepoRoot, "config", "titles", titleSlug, "catalog", "experience_rules.toml")
	adapter, err := halo_games.NewCatalogAdapter(fetcher, rulesPath)
	if err != nil {
		return res, fmt.Errorf("init catalog adapter: %w", err)
	}
	resolver := games.NewStaticResolver(titleSlug)
	resolver.RegisterCatalog(adapter)

	// 4. Drain loop (s'arrête dès qu'une passe ne progresse plus, ou ctx annulé).
	svc := service.NewCatalogFetcherService(metaSQL, resolver)
	for pass := 1; pass <= catalogDrainMaxPasses; pass++ {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		d, err := svc.Drain(ctx, titleSlug)
		if err != nil {
			return res, fmt.Errorf("drain pass %d: %w", pass, err)
		}
		if d.Playlists+d.Pairs+d.Maps+d.GameVariants == 0 {
			// Aucune NOUVELLE résolution cette passe : file résolue, ou le reste est
			// non résolvable (404 Discovery). Inutile de re-tenter dans ce run —
			// append-only, les non-résolvables restent "pending" pour un drain futur.
			break
		}
		res.Playlists += d.Playlists
		res.Pairs += d.Pairs
		res.Maps += d.Maps
		res.GameVariants += d.GameVariants
		res.Errors += d.Errors
	}

	monitoringLog.InfoContext(ctx, "admin_actions: drain UGC terminé",
		"title", titleSlug, "seeded", res.Seeded, "playlists", res.Playlists,
		"pairs", res.Pairs, "maps", res.Maps, "game_variants", res.GameVariants, "errors", res.Errors)
	observability.IncCounter("admin_action_catalog_ugc_drain_total")
	return res, nil
}

// haloTokensForDrain charge des tokens Halo via le MultiUserTokenStore (premier
// joueur authentifié). L'API DiscoveryUGC est par-asset, pas par-joueur — tout
// joueur valide convient.
func (r *ServiceRegistry) haloTokensForDrain(ctx context.Context, titleSlug string) (*domain.HaloTokens, error) {
	pr := titlePkg.NewPathResolver(r.cfg.RepoRoot)
	store := auth.NewMultiUserTokenStore(pr.WatcherTokensDir())
	provider := auth.NewMSALProvider()

	players, err := r.cfg.LoadPlayers(titleSlug)
	if err != nil {
		return nil, fmt.Errorf("load players: %w", err)
	}
	for _, p := range players {
		if p.XUID == "" {
			continue
		}
		result, refreshErr := auth.RefreshHaloTokensViaStoreFirst(ctx, store, provider, p.XUID, p.Gamertag, auth.LegacyAuthInputs{})
		if refreshErr == nil && result != nil && result.Tokens != nil {
			return result.Tokens, nil
		}
	}
	return nil, fmt.Errorf("aucun token Halo valide (le drain UGC nécessite un joueur authentifié — re-capturer un token)")
}
