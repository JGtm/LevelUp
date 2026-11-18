// Package api — registry_catalog_drain.go : runner de l'action « drain
// DiscoveryUGC » (job asynchrone). Recense les asset IDs vus en jeu
// (catalog_fetch_queue) puis les résout via l'API DiscoveryUGC (réseau,
// rate-limité). Complète le mode from_match_registry (zéro réseau) : ici on
// hydrate les assets dont le nom n'est PAS dans match_registry.
//
// In-process, fidèle au CLI populate-playlists-catalog (chaîne provider →
// fetcher → adapter → resolver → CatalogFetcherService.Drain). Spécifique Halo
// Infinite — comme ops/catalog_refresh.go. Single-flight ; le jeton vient de la
// source de lecture publique de la mise en commun (ADR 0038), jamais d'un profil.
package wire

import (
	"context"
	"errors"
	"fmt"
	gosync "sync"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	halo_games "levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/games/halo_infinite/haloadapter"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/adminstate"
	"levelup/go-api/internal/platform/duckdb"
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
func (r *ServiceRegistry) RunCatalogUGCDrain(ctx context.Context, titleSlug string) (res domain.CatalogUGCDrainResult, err error) {
	if !catalogDrainMu.TryLock() {
		return res, ErrActionBusy
	}
	defer catalogDrainMu.Unlock()
	// C2 : journal de l'exécution réelle (single-flight non journalisé — placé
	// avant ce defer). err = erreur finale nommée.
	defer func() { r.journalAction(ctx, adminstate.ActionCatalogUGCDrain, err) }()

	metaSQL, closeMeta := r.titleBases().Metadata(ctx, titleSlug)
	defer closeMeta()
	if metaSQL == nil {
		return res, fmt.Errorf("metadata indisponible pour %s", titleSlug)
	}

	// 1. Recensement (zéro réseau) : seule étape qui lit la base partagée, rendue avant le
	// réseau (un lecteur en vol fait échouer toute prise d'écriture de la base, ADR 0016).
	seeded, err := ops.SeedCatalogQueueForTitle(ctx, r.titleBases(), metaSQL, titleSlug)
	if err != nil {
		return res, err
	}
	res.Seeded = seeded.Total()

	// 2. Tokens Halo (DiscoveryUGC nécessite un Spartan token valide).
	tokens, err := jetonsCatalogue(r, ctx)
	if err != nil {
		return res, err
	}

	// 3. Chaîne de résolution (fidèle au CLI populate-playlists-catalog).
	provider := haloadapter.NewHaloProvider().WithRateLimit(catalogDrainRateLimit).WithTokens(tokens)
	fetcher := haloadapter.NewCatalogFetcher(provider)
	rulesPath := r.catalogExperienceRulesPath(titleSlug)
	adapter, err := halo_games.NewCatalogAdapter(fetcher, rulesPath)
	if err != nil {
		return res, fmt.Errorf("init catalog adapter: %w", err)
	}
	resolver := games.NewStaticResolver(titleSlug)
	resolver.RegisterCatalog(adapter)

	// 4. Drain loop (s'arrête dès qu'une passe ne progresse plus, ou ctx annulé).
	svc := service.NewCatalogFetcherService(duckdb.NewCatalogWriter(metaSQL), resolver)
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

	// Auto-réparation (défense en profondeur) : si une écriture du drain a malgré
	// tout FATAL-invalidé le handle metadata partagé (bug ART résiduel — les
	// erreurs per-row de markError sont loggées, pas remontées par Drain), on le
	// détecte par un ping et on Reopen IN-PLACE. Comme metadata est un handle
	// process-wide (OpenReadWriteShared), ce Reopen ressuscite la base pour TOUS
	// les lecteurs (season pass, modes, playlists, home…) sans redémarrage — au
	// lieu de laisser l'app cassée jusqu'au prochain restart (qui re-cassait au
	// boot). Cf. drop_metadata_art_surface_indexes_v1 qui supprime la surface ART
	// en amont ; ceci est le filet si une nouvelle surface réapparaît.
	r.reopenMetadataIfInvalidated(ctx, titleSlug)
	return res, nil
}

// reopenMetadataIfInvalidated ping le handle metadata partagé ; s'il est invalidé
// (FATAL ART), le Reopen in-place pour rétablir toute l'app sans restart.
// Best-effort : emprunt non-possédant du cache (pas de Close).
func (r *ServiceRegistry) reopenMetadataIfInvalidated(ctx context.Context, titleSlug string) {
	pr := r.cfg.PathResolver()
	metaPath := pr.MetadataDBPath(titleSlug)
	wrapper, ok := duckdb.LookupCachedDB(metaPath)
	if !ok || wrapper == nil {
		return
	}
	if err := wrapper.SQLDb().PingContext(ctx); err == nil {
		return // sain
	} else if !duckdb.IsInvalidatedError(err) {
		return // erreur transitoire non liée à l'invalidation ART
	}
	if rerr := wrapper.Reopen(); rerr != nil {
		monitoringLog.ErrorContext(ctx, "catalog drain: metadata invalidée, Reopen échoué (restart requis)",
			"title", titleSlug, "err", rerr)
		return
	}
	monitoringLog.WarnContext(ctx, "catalog drain: metadata invalidée → Reopen in-place réussi (app rétablie sans restart)",
		"title", titleSlug)
}

// ErrNoPublicReadToken : aucun jeton Halo ne peut servir un téléchargement de fond (catalogue,
// configuration des playlists, manifeste de film). Deux causes, toutes deux un refus : la source de
// lecture publique de la mise en commun n'est pas câblée, ou elle ne rend aucun jeton parce qu'aucun
// compte n'a accepté la mise en commun (ADR 0038 : aucune décision = refus).
var ErrNoPublicReadToken = errors.New("wire: aucun jeton de lecture publique disponible")

// haloTokensForDrain rend un jeton Halo pour un téléchargement de fond dont le contenu ne dépend
// d'aucun joueur (DiscoveryUGC, configuration d'une playlist, manifeste d'un film). Le jeton vient
// UNIQUEMENT de la source de lecture publique de la mise en commun, qui exclut les comptes sans
// accord : le jeton d'un joueur ne sert jamais le contenu d'un autre sans son consentement.
// Source non câblée ou aucun compte consentant : ErrNoPublicReadToken, journalisée ; le jeton d'un
// profil n'est jamais pris en repli.
func (r *ServiceRegistry) haloTokensForDrain(ctx context.Context) (*domain.HaloTokens, error) {
	if r.publicReadTokenSrc == nil {
		monitoringLog.WarnContext(ctx, "jeton de lecture publique : source de la mise en commun non câblée, rien n'est téléchargé")
		return nil, fmt.Errorf("%w: source non câblée", ErrNoPublicReadToken)
	}
	acquireCtx, cancel := context.WithTimeout(ctx, publicReadPoolAcquireTimeout)
	defer cancel()
	tokens, _, err := r.publicReadTokenSrc.ResolveAny(acquireCtx)
	if err != nil {
		monitoringLog.WarnContext(ctx, "jeton de lecture publique : aucun compte consentant ou sain dans la mise en commun, rien n'est téléchargé",
			"err", err)
		return nil, fmt.Errorf("%w: %w", ErrNoPublicReadToken, err)
	}
	if tokens == nil {
		monitoringLog.WarnContext(ctx, "jeton de lecture publique : la source n'a rendu aucun jeton, rien n'est téléchargé")
		return nil, fmt.Errorf("%w: jeton vide", ErrNoPublicReadToken)
	}
	return tokens, nil
}

// jetonsCatalogue : prise des jetons Halo du catalogue (expansion des playlists, drain UGC), le
// premier accès réseau de ces deux tâches. Variable de paquet : les tests y substituent une prise
// factice, sans réseau ni source de jetons.
var jetonsCatalogue = (*ServiceRegistry).haloTokensForDrain
