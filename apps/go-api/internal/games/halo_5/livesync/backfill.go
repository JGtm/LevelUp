package livesync

// backfill.go — backfill HISTORIQUE complet, incrémental et résumable, pour Halo 5.
//
// Contraste avec le sync live (Runner.RunDelta) :
//   - Live   : delta-stop au 1er match connu (on ne veut QUE les nouveaux), capture
//     TOUT en mémoire puis persiste à la fin (volume borné par MaxMatches).
//   - Backfill : paginne TOUT l'historique (start croissant jusqu'à page vide), SAUTE
//     les matchs connus mais NE s'arrête PAS dessus (skip-known + paginate-deeper), et
//     PERSISTE CHAQUE PAGE IMMÉDIATEMENT (lease RW court par page) → incrémental +
//     résumable. INSERT-only idempotent par match_id : relancer après interruption
//     reprend sans doublon (les pages déjà persistées sont skip-known, les nouvelles
//     pages sont ingérées).
//
// RÉUTILISE les briques du runner SANS rien réinventer :
//   - capture page-par-page : halo5.CapturePageAt (cœur partagé avec CollectRecentMatches),
//   - known-set            : loadKnownMatchIDs (même requête match_registry, lease court),
//   - persistance          : persistBatches (SharedPersister, INSERT-only, ART-safe),
//   - résolution xuid       : halo5ResolverFactory (CachingResolver PeopleHub + graine aliases).

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/canonical"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/persist"
)

// BackfillStats résume une passe de backfill (logs + sortie CLI).
type BackfillStats struct {
	Pages         int // pages parcourues (y compris la page vide finale)
	MatchesSeen   int // résumés parcourus (toutes pages)
	Inserted      int // batches réellement persistés (nouveaux matchs)
	Skipped       int // matchs déjà connus sautés (résume sans doublon)
	EventsFailed  int // timelines indisponibles (batch registry-only)
	CarnageFailed int // carnages indisponibles (batch sans participants)
	Warzone       int // matchs Warzone écartés
	Campaign      int // matchs Campagne écartés (non jouables côté app)
	PersistErrors int // erreurs de persistance (page-level)
}

// CapturePageFunc : signature de halo5.CapturePageAt (injectable → fake en test, la
// vraie source h5 expose un type events non exporté, donc on mocke ce niveau plutôt
// que la source elle-même).
type CapturePageFunc func(
	ctx context.Context,
	viewer canonical.PlayerIdentity,
	resolveXUID func(gamertag string) string,
	isKnown func(matchID string) bool,
	start, pageSize int,
	seen map[string]struct{},
	stats *halo5.CaptureStats,
) (batches []*persist.MatchBatch, hasMore bool, err error)

// BackfillDeps regroupe les dépendances injectées du backfill (toutes mockables →
// boucle unit-testable sans réseau ni DuckDB).
//
//   - CapturePage  : capture UNE page à un offset (halo5.CapturePageAt câblé sur la
//     source live). Skip-known SANS delta-stop. La source live n'est PAS dans Deps :
//     elle est capturée dans la closure CapturePage (le type events h5 est non exporté).
//   - Viewer       : joueur backfillé (self), owner des batches. XUID = xuid Xbox RÉSOLU.
//   - ResolveXUID  : gamertag → xuid Xbox ("" toléré, identité conservée au gamertag).
//   - LoadKnown    : match_ids déjà persistés (lease RW court). Lu UNE fois au début
//     (le backfill ne re-lit pas le known-set à chaque page : il est amorcé une fois,
//     puis les nouveaux match_ids persistés y sont ajoutés en mémoire — un re-collect
//     d'un match déjà inséré dans CE run est de toute façon no-opé par le persister).
//   - PersistPage  : persiste les batches d'UNE page sous un lease RW court ; retourne
//     les batches réellement écrits + les erreurs.
type BackfillDeps struct {
	CapturePage CapturePageFunc
	Viewer      canonical.PlayerIdentity
	ResolveXUID func(gamertag string) string
	LoadKnown   func(ctx context.Context) (map[string]bool, error)
	PersistPage func(ctx context.Context, batches []*persist.MatchBatch) (done []*persist.MatchBatch, errs []string)
}

// backfillMaxPages : borne de sécurité anti-boucle-infinie (la condition d'arrêt
// normale est la page vide). ~2000 pages × 25 = 50000 matchs : très au-delà de tout
// historique h5 réaliste, mais protège d'une API qui n'honorerait jamais `start`.
const backfillMaxPages = 2000

// RunBackfill exécute le backfill historique complet d'un joueur Halo 5 : paginne
// l'historique par offset croissant, capture+persiste CHAQUE page immédiatement,
// saute les matchs connus et continue à paginer en profondeur, s'arrête à la page
// vide (fin d'historique) ou à la borne de sécurité backfillMaxPages.
//
// pageSize <= 0 → défaut halo5 (25). logger nil → slog.Default().
func RunBackfill(ctx context.Context, deps BackfillDeps, pageSize int, logger *slog.Logger) (BackfillStats, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if pageSize <= 0 {
		pageSize = 25
	}

	// Known-set amorcé UNE fois (lease court relâché avant le fetch réseau). Best-effort :
	// sans known-set on collecte tout (idempotence garantie en aval par match_registry).
	known := map[string]bool{}
	if deps.LoadKnown != nil {
		if k, err := deps.LoadKnown(ctx); err != nil {
			logger.WarnContext(ctx, "h5 backfill: known-set indisponible (collecte sans skip)", "err", err)
		} else {
			known = k
		}
	}
	isKnown := func(id string) bool { return known[id] }

	var (
		stats BackfillStats
		cum   halo5.CaptureStats              // cumulé sur toutes les pages
		seen  = make(map[string]struct{}, 64) // anti-boucle partagé entre pages
	)

	for page := 0; page < backfillMaxPages; page++ {
		start := page * pageSize
		before := cum
		batches, hasMore, err := deps.CapturePage(ctx, deps.Viewer,
			deps.ResolveXUID, isKnown, start, pageSize, seen, &cum)
		if err != nil {
			logger.ErrorContext(ctx, "h5 backfill: capture page échouée (arrêt)",
				"page", page, "start", start, "err", err)
			return stats, err
		}
		stats.Pages = page + 1

		inserted := persistAndTrack(ctx, deps, batches, known, &stats, logger)

		oldest := oldestStart(batches)
		logger.InfoContext(ctx, "h5 backfill: page",
			"page", page, "start", start,
			"page_seen", cum.MatchesSeen-before.MatchesSeen,
			"inserted", inserted, "cumulated", stats.Inserted,
			"skipped_cum", cum.MatchesSkipped, "oldest_match_date", oldest)

		if !hasMore {
			break // fin de l'historique (page vide ou incomplète)
		}
	}

	finalizeStats(&stats, cum)
	logger.InfoContext(ctx, "h5 backfill: terminé",
		"gamertag", deps.Viewer.Gamertag, "pages", stats.Pages,
		"seen", stats.MatchesSeen, "inserted", stats.Inserted, "skipped", stats.Skipped,
		"events_failed", stats.EventsFailed, "carnage_failed", stats.CarnageFailed,
		"warzone", stats.Warzone, "campaign", stats.Campaign, "persist_errors", stats.PersistErrors)
	return stats, nil
}

// persistAndTrack persiste les batches d'une page (lease court) et met à jour le
// known-set en mémoire avec les match_ids réellement écrits (anti re-collect dans le
// même run). Retourne le nombre de matchs insérés sur la page.
func persistAndTrack(ctx context.Context, deps BackfillDeps, batches []*persist.MatchBatch,
	known map[string]bool, stats *BackfillStats, logger *slog.Logger) int {
	if len(batches) == 0 || deps.PersistPage == nil {
		return 0
	}
	done, errs := deps.PersistPage(ctx, batches)
	stats.Inserted += len(done)
	stats.PersistErrors += len(errs)
	for _, e := range errs {
		logger.WarnContext(ctx, "h5 backfill: persist erreur (match sauté)", "err", e)
	}
	for _, b := range done {
		if b.Shared.Match != nil {
			known[b.Shared.Match.MatchID] = true
		}
	}
	return len(done)
}

// finalizeStats projette les compteurs cumulés de capture dans BackfillStats.
func finalizeStats(stats *BackfillStats, cum halo5.CaptureStats) {
	stats.MatchesSeen = cum.MatchesSeen
	stats.Skipped = cum.MatchesSkipped
	stats.EventsFailed = cum.EventsFailed
	stats.CarnageFailed = cum.CarnageFailed
	stats.Warzone = cum.ExcludedWarzone
	stats.Campaign = cum.ExcludedCampaign
}

// oldestStart retourne la date de début du match le plus ancien de la page (le dernier
// du lot, l'historique étant trié du plus récent au plus ancien). "" si page vide ou
// dates absentes — sert au log de progression.
func oldestStart(batches []*persist.MatchBatch) string {
	for i := len(batches) - 1; i >= 0; i-- {
		if b := batches[i]; b != nil && b.Shared.Match != nil && !b.Shared.Match.StartTime.IsZero() {
			return b.Shared.Match.StartTime.Format("2006-01-02")
		}
	}
	return ""
}

// BuildBackfillDeps câble les BackfillDeps de PRODUCTION pour un joueur : known-set +
// persist sur le shared h5 (même provider que la lecture, B-swap), resolver PeopleHub
// (graine xuid_aliases). La Source DOIT être construite par le caller (réseau différé,
// token déjà dans le ctx). RÉUTILISE intégralement les helpers du runner.
func BuildBackfillDeps(ctx context.Context, cfg *config.AppConfig, src halo5.CaptureSource, gamertag, xuid string) BackfillDeps {
	logger := slog.Default()
	sharedPath := titlePkg.NewPathResolver(cfg.RepoRoot).SharedDBPath(halo5.TitleSlug)
	provider := sharedProviderForPath(cfg, sharedPath)

	var resolveXUID func(string) string = func(string) string { return "" }
	if factory := halo5ResolverFactory(cfg, gamertag, xuid, logger); factory != nil {
		resolveXUID = ResolveXUIDClosure(ctx, factory(ctx), logger)
	}

	return BackfillDeps{
		// Capture page-par-page câblée sur la source live : la source est CAPTURÉE dans
		// la closure (son type events est non exporté → pas dans Deps). Réutilise le
		// cœur de capture partagé avec le sync live (halo5.CapturePageAt).
		CapturePage: func(ctx context.Context, viewer canonical.PlayerIdentity,
			resolveXUID func(string) string, isKnown func(string) bool,
			start, pageSize int, seen map[string]struct{}, stats *halo5.CaptureStats,
		) ([]*persist.MatchBatch, bool, error) {
			return halo5.CapturePageAt(ctx, src, viewer, resolveXUID, isKnown,
				halo5.CaptureOptions{}, start, pageSize, seen, stats)
		},
		Viewer:      canonical.PlayerIdentity{Gamertag: gamertag, XUID: xuid},
		ResolveXUID: resolveXUID,
		LoadKnown: func(ctx context.Context) (map[string]bool, error) {
			return loadKnownMatchIDs(ctx, provider, sharedPath)
		},
		PersistPage: func(ctx context.Context, batches []*persist.MatchBatch) ([]*persist.MatchBatch, []string) {
			return persistBatches(ctx, provider, sharedPath, batches)
		},
	}
}
