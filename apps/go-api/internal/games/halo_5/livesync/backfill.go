package livesync

// backfill.go — backfill HISTORIQUE complet, incrémental et résumable, pour Halo 5.
//
// Contraste avec le sync live (Runner.RunDelta) :
//   - Live   : delta-stop après une suite de matchs connus (domain.DeltaKnownRun) (on ne veut QUE les nouveaux), capture
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
	"levelup/go-api/internal/games/canonical"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/sync/historystate"
)

// backfillDefaultPageSize est la taille de page de la liste de l'API (celle de halo5.CapturePageAt).
const backfillDefaultPageSize = 25

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

	// HistoryEnd : la liste de l'API a rendu une page vide (fin d'historique vue,
	// voir halo5.CaptureStats.HistoryEnd). Faux quand la passe s'arrête avant : budget atteint,
	// erreur de page, anti-boucle, profondeur historystate.MaxDepth.
	HistoryEnd bool
	// BudgetReached : la passe s'est arrêtée parce que BackfillDeps.Budget matchs étaient insérés.
	BudgetReached bool
	// Capture : compteurs de capture cumulés sur toutes les pages (les champs ci-dessus en sont la projection).
	Capture halo5.CaptureStats

	// ResumeMissed : le marque-page n'était pas sur la page de recouvrement, la passe est repartie du début.
	ResumeMissed bool
	// AbandonedPages : pages en échec durable sautées (BackfillResume.MaxAttempts atteint), dont
	// AbandonedErrors erreurs de persistance, qui ne comptent plus comme erreurs bloquantes.
	AbandonedPages  int
	AbandonedErrors int
	// FrontierAttempts et FrontierSkipped : à la sortie, les échecs consécutifs de la première page non
	// abandonnée après le marque-page et le nombre de pages abandonnées juste après lui.
	FrontierAttempts int
	FrontierSkipped  int
	// LastError : le texte de la dernière erreur de page ou de persistance.
	LastError string
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
//   - Budget       : nombre de matchs INSÉRÉS au-delà duquel la passe s'arrête (0 = sans borne,
//     comportement de la commande h5-backfill). Contrôlé entre deux pages, après leur persistance :
//     une page entière est toujours rangée, la passe insère donc au plus Budget + pageSize - 1
//     matchs. La passe suivante reprend en sautant les matchs connus.
type BackfillDeps struct {
	CapturePage CapturePageFunc
	Viewer      canonical.PlayerIdentity
	ResolveXUID func(gamertag string) string
	LoadKnown   func(ctx context.Context) (map[string]bool, error)
	PersistPage func(ctx context.Context, batches []*persist.MatchBatch) (done []*persist.MatchBatch, errs []string)
	Budget      int
	// Resume : reprise au marque-page du rattrapage automatique (nil pour la commande h5-backfill).
	Resume *BackfillResume
}

// BackfillResume décrit la reprise d'une passe de rattrapage au marque-page (historystate.Cursor).
//
// La passe lit d'abord la page de recouvrement à Start : si ExpectID (l'identifiant du dernier match
// traité par la passe précédente) n'y figure pas, le décalage rangé ne vaut plus et la passe repart du
// début (ResumeMissed). Sinon elle continue au-delà, en sautant Skip pages abandonnées. Sans ExpectID
// (pas de marque-page), les Skip pages sautées sont celles du début de la liste.
//
// Un marque-page est rangé (Save) après chaque page pleine rangée sans erreur qui a inséré au moins un
// match, et une dernière fois à la sortie pour la dernière page propre. Une page en échec le laisse avant
// elle. Quand la première page non abandonnée échoue pour la MaxAttempts-ième passe de suite (Attempts
// échecs déjà comptés), elle est abandonnée : journalisée, comptée (h5_history_abandoned_pages_total),
// sautée ; la passe continue. MaxAttempts 0 : jamais d'abandon.
//
// HeadID est l'identifiant de l'entrée en position 0 de la liste lue par cette passe (l'ancre des
// marque-pages qu'elle range, historystate.Cursor.HeadID). Vide quand la passe n'a pas encore lu la
// tête : les marque-pages prennent alors la tête vue par la capture de la première page.
//
// Un échec qui ne compte pas (panne de l'API : historyfail) ne fait pas avancer le compte vers l'abandon. Une
// page abandonnée est un trou connu : OnAbandon le note (historystate.NoteHole), pour que le rattrapage
// relise la liste après historystate.AbandonRetryAfter.
type BackfillResume struct {
	Start       int
	ExpectID    string
	HeadID      string
	Skip        int
	Attempts    int
	MaxAttempts int
	Save        func(ctx context.Context, c historystate.Cursor)
	OnAbandon   func(ctx context.Context)
}

// RunBackfill exécute le backfill historique complet d'un joueur Halo 5 : paginne
// l'historique par offset croissant, capture+persiste CHAQUE page immédiatement,
// saute les matchs connus et continue à paginer en profondeur, s'arrête à la page
// vide (fin d'historique), à la profondeur historystate.MaxDepth ou, quand deps.Budget > 0,
// dès que le budget de matchs insérés est atteint (la fin n'est alors pas vue : stats.HistoryEnd faux).
// La boucle vit dans backfill_walk.go ; deps.Resume ajoute le marque-page et l'abandon des pages
// en échec durable (rattrapage automatique), sans effet pour la commande h5-backfill.
//
// pageSize <= 0 → défaut halo5 (25). logger nil → slog.Default().
func RunBackfill(ctx context.Context, deps BackfillDeps, pageSize int, logger *slog.Logger) (BackfillStats, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if pageSize <= 0 {
		pageSize = backfillDefaultPageSize
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

	run := newBackfillRun(deps, pageSize, logger, known)
	err := run.walk(ctx)
	run.flushCursor(ctx)
	run.stats.FrontierAttempts, run.stats.FrontierSkipped = run.attempts, run.skipped
	finalizeStats(&run.stats, run.cum)
	stats := run.stats
	logger.InfoContext(ctx, "h5 backfill: terminé",
		"gamertag", deps.Viewer.Gamertag, "pages", stats.Pages,
		"seen", stats.MatchesSeen, "inserted", stats.Inserted, "skipped", stats.Skipped,
		"events_failed", stats.EventsFailed, "carnage_failed", stats.CarnageFailed,
		"warzone", stats.Warzone, "campaign", stats.Campaign, "persist_errors", stats.PersistErrors,
		"history_end", stats.HistoryEnd, "budget_reached", stats.BudgetReached,
		"abandoned_pages", stats.AbandonedPages, "resume_missed", stats.ResumeMissed)
	return stats, err
}

// persistAndTrack persiste les batches d'une page (lease court) et met à jour le
// known-set en mémoire avec les match_ids réellement écrits (anti re-collect dans le
// même run). Retourne le nombre de matchs insérés sur la page et ses erreurs de persistance.
func persistAndTrack(ctx context.Context, deps BackfillDeps, batches []*persist.MatchBatch,
	known map[string]bool, stats *BackfillStats, logger *slog.Logger) (int, []string) {
	if len(batches) == 0 || deps.PersistPage == nil {
		return 0, nil
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
	return len(done), errs
}

// finalizeStats projette les compteurs cumulés de capture dans BackfillStats.
func finalizeStats(stats *BackfillStats, cum halo5.CaptureStats) {
	stats.MatchesSeen = cum.MatchesSeen
	stats.Skipped = cum.MatchesSkipped
	stats.EventsFailed = cum.EventsFailed
	stats.CarnageFailed = cum.CarnageFailed
	stats.Warzone = cum.ExcludedWarzone
	stats.Campaign = cum.ExcludedCampaign
	stats.HistoryEnd = cum.HistoryEnd
	stats.Capture = cum
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
	sharedPath := cfg.PathResolver().SharedDBPath(halo5.TitleSlug)
	provider := sharedProviderForPath(cfg, sharedPath)

	var resolveXUID func(string) string = func(string) string { return "" }
	if factory := halo5ResolverFactory(cfg, gamertag, xuid, logger); factory != nil {
		resolveXUID = ResolveXUIDClosure(ctx, factory(ctx), logger)
	}

	return BackfillDeps{
		CapturePage: capturePageFrom(src),
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

// capturePageFrom câble la capture page-par-page sur la source live : la source est capturée
// dans la closure (son type events est non exporté, donc absent des Deps). Réutilise le cœur de
// capture partagé avec le sync live (halo5.CapturePageAt).
func capturePageFrom(src halo5.CaptureSource) CapturePageFunc {
	return func(ctx context.Context, viewer canonical.PlayerIdentity,
		resolveXUID func(string) string, isKnown func(string) bool,
		start, pageSize int, seen map[string]struct{}, stats *halo5.CaptureStats,
	) ([]*persist.MatchBatch, bool, error) {
		return halo5.CapturePageAt(ctx, src, viewer, resolveXUID, isKnown,
			halo5.CaptureOptions{}, start, pageSize, seen, stats)
	}
}
