// Package service — career_live_service.go : orchestration du flow live
// carrière (XP/rang + Spartan ID) découplé du post-sync matchs.
//
// Pourquoi un service dédié : le post-sync matchs n'est pas le bon endroit
// pour rafraîchir l'XP carrière. Quand le joueur ne joue pas, le watcher
// matchs n'a aucune raison de tourner ; pourtant l'utilisateur peut ouvrir
// la home et s'attendre à voir une XP fraîche (typiquement après avoir reçu
// du XP via un défi terminé hors-match).
//
// Cadences :
//
//   - GetCareerProgress (XP + rang)            → cache TTL 5 min + singleflight
//   - GetSpartanCustomization (ServiceTag, …)  → cache TTL 6 h + singleflight
//
// Budget de latence STRICT sur la home (CareerLiveBudget, défaut 2.5 s) :
// la home ne doit jamais bloquer plus que ce budget sur le live carrière.
// Si dépassé, on tombe immédiatement sur la dernière row DB et on laisse
// le fetch terminer en arrière-plan (singleflight + cache détaché) pour
// que la requête suivante bénéficie de données fraîches.
//
// Le fetch customization peut faire jusqu'à 4 HTTP calls en série (1 endpoint
// + 3 image resolves vers GameCMS), donc on parallélise progress et
// customization et on cap toute la phase live au budget.
//
// Fallback à 4 niveaux (la home ne doit jamais montrer un bloc Spartan vide
// si la player DB porte des données historiques) :
//
//  1. live OK, champs complets         → utilise les valeurs live
//  2. live OK, certains champs vides   → per-field merge depuis la dernière
//     row connue en DB (carry-forward)
//  3. live KO/timeout complet          → fallback total sur la dernière row
//  4. live KO + DB vide                → nil (front affiche placeholder)
//
// INSERT-if-changed dans `career_progression` : une nouvelle row n'est
// écrite que si au moins un champ d'identité diffère de la dernière (cf.
// duckdb.CareerRankRowEqualForInsert). Évite de saturer la table à 288
// rows/jour (cadence 5 min) tout en gardant un historique propre pour le
// graphe d'évolution XP de la page Carrière.
package service

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
	syncpkg "levelup/go-api/internal/sync"
)

// CareerLiveBudget cap la durée totale du fetch live dans le chemin
// synchrone de la home. Au-delà, on retourne la dernière row DB connue et
// on laisse le fetch terminer en background pour la prochaine requête.
//
// 2.5 s est un compromis : assez long pour absorber un cold-start DNS +
// 1 round-trip Halo + 3 resolves GameCMS en // ; assez court pour ne pas
// dégrader visiblement la home si le réseau Halo se met à ramer.
const CareerLiveBudget = 2500 * time.Millisecond

// careerLiveBgTimeout cap la durée des refresh background détachés du
// contexte de la requête. Plus généreux que le budget synchrone parce que
// le caller ne nous attend plus.
const careerLiveBgTimeout = 30 * time.Second

const careerLiveLogModule = "career_live"

// CareerFetcher abstrait les appels live Halo nécessaires au flow.
// Implémenté par sync.HaloAPIClient (production) et par les mocks de tests.
type CareerFetcher interface {
	GetCareerProgress(ctx context.Context, xuid string) (*syncpkg.CareerRankData, error)
	GetSpartanCustomization(ctx context.Context, xuid string) (*syncpkg.SpartanCustomizationData, error)
}

// CareerFetcherFactory instancie un fetcher live depuis le contexte de la
// requête (lecture des tokens via ctxkeys.HaloTokens). Retourne (nil, nil)
// si les tokens sont absents — auquel cas le service tombe directement en
// fallback DB.
type CareerFetcherFactory func(ctx context.Context) CareerFetcher

// CareerLiveRepo regroupe les opérations DB nécessaires au service.
// Interface volontairement étroite pour faciliter le mocking en tests
// unitaires (cf. career_live_service_test.go).
type CareerLiveRepo interface {
	LoadLastCareerRank(ctx context.Context, xuid string) (*duckdb.CareerRankRow, error)
	EnrichFromMetadata(ctx context.Context, row *duckdb.CareerRankRow) error
	// InsertCareerProgressionIfChanged écrit une copie complète (live + carry-forward).
	// Conservé pour compat tests legacy. Le chemin V2 utilise
	// InsertCareerProgressionPartial qui n'écrit que les champs frais.
	InsertCareerProgressionIfChanged(ctx context.Context, xuid string, data *duckdb.CareerRankRow) (bool, error)
	// InsertCareerProgressionPartial écrit UNIQUEMENT les champs set du partial,
	// les autres restent NULL (Phase 2/3 PLAN_V2 §5).
	InsertCareerProgressionPartial(ctx context.Context, xuid string, partial *duckdb.CareerProgressionPartial) (bool, error)
}

// CareerIdentityBuilder construit le HomeSpartanIdentityRow final à partir
// d'une CareerRankRow (déjà mergée) + skill peaks. Implémenté par
// duckdb.HomeRepo.BuildSpartanIdentityFromCareerRow.
type CareerIdentityBuilder interface {
	BuildSpartanIdentityFromCareerRow(ctx context.Context, careerRow *duckdb.CareerRankRow) *domain.HomeSpartanIdentityRow
}

// CareerLiveService orchestre live + cache + fallback + INSERT-if-changed.
//
// Le service est process-level et thread-safe (le cache l'est, et les autres
// dépendances sont immuables après construction).
type CareerLiveService struct {
	repo           CareerLiveRepo
	builder        CareerIdentityBuilder
	fetcherFactory CareerFetcherFactory
	cache          *CareerLiveCache

	// bgInflight déduplique les refresh background : un seul refresh actif
	// par xuid, peu importe combien de requêtes timeoutent en parallèle.
	bgInflightMu sync.Mutex
	bgInflight   map[string]bool
}

// NewCareerLiveService construit le service. `cache` peut être nil (auquel cas
// chaque appel ira au live sans throttle — utile pour les tests qui n'ont pas
// besoin du caching).
func NewCareerLiveService(
	repo CareerLiveRepo,
	builder CareerIdentityBuilder,
	fetcherFactory CareerFetcherFactory,
	cache *CareerLiveCache,
) *CareerLiveService {
	return &CareerLiveService{
		repo:           repo,
		builder:        builder,
		fetcherFactory: fetcherFactory,
		cache:          cache,
		bgInflight:     make(map[string]bool),
	}
}

// GetSpartanIdentity retourne le bloc Spartan ID complet pour la home.
//
// **Contrat UI-first** : si la player DB porte une row historique avec une
// bannière (ou emblem/backdrop/spartan_id), on la retourne TOUJOURS. La
// fraîcheur du live ne doit JAMAIS dégrader la visibilité (cf. revue
// 2026-05-20 « les bannières vont et viennent »). Retourne nil uniquement
// quand DB ET live sont tous deux vides (joueur jamais sync'd).
//
// Stratégie défense en profondeur :
//
//  1. Lecture DB last-known-good **systématique** au début (synchronous,
//     <50 ms). Cette row sert de filet de sécurité pour le reste du flow.
//  2. Si xuid absent ou repo DB indisponible → on retourne directement la
//     row DB telle quelle (peut être nil si la DB est vide).
//  3. fetch+merge live (cache + dbLast per-field merge interne). Si la
//     merge layer rate un fallback (erreur transitoire LoadLastCareerRank
//     pendant un lock B-swap), notre snapshot étape 1 reste.
//  4. Overlay final : tout champ d'identité (banner/emblem/backdrop/
//     spartan_id) absent du résultat live est patché depuis la row étape 1.
//     C'est ce qui transforme « le live a rendu null cette fois » en
//     « on continue de servir la dernière valeur connue ».
func (s *CareerLiveService) GetSpartanIdentity(ctx context.Context) (*domain.HomeSpartanIdentityRow, error) {
	xuid := ctxkeys.HaloXUID(ctx)

	// Étape 1-2 : filet DB systématique. serveDBFallback est tolérant à
	// xuid="" / repo nil (retourne nil), donc on peut toujours l'appeler.
	dbFallback := s.serveDBFallback(ctx, xuid)
	if xuid == "" {
		return dbFallback, nil
	}

	// Étape 3 : tentative live (peut échouer transitoirement).
	merged, err := s.fetchAndMerge(ctx, xuid)
	if err != nil {
		slog.WarnContext(ctx, careerLiveLogModule+": fetch+merge failed → DB fallback",
			"xuid", xuid, "err", err)
		return dbFallback, nil
	}

	// Phase 4 PLAN_V2 : la persistance est déléguée à kickoffBackgroundRefresh
	// (path background) qui a accès au progress+custom bruts. Le path sync
	// se contente de servir ce qu'on a — pas de persist depuis ici, pas de
	// risque d'écrire des champs carry-forward dans une nouvelle ligne.

	identity := s.builder.BuildSpartanIdentityFromCareerRow(ctx, merged)

	// Étape 4 : overlay final. Si live a produit identity == nil ou identity
	// avec des champs assets vides, on patche depuis dbFallback. Le résultat
	// est garanti aussi complet que ce que la DB historique sait offrir.
	identity = overlayIdentityFromFallback(identity, dbFallback)

	if identity == nil {
		careerLiveIdentityMissing.Add(1)
		careerLiveEmptyResult.Add(1)
		return nil, nil
	}
	careerLiveIdentityServed.Add(1)
	return identity, nil
}

// overlayIdentityFromFallback → extrait dans `career_live_merge.go`.

// fetchAndMerge construit la CareerRankRow servie à la home selon le pattern
// **stale-while-revalidate** :
//
//  1. Lecture DB synchrone (toujours <50 ms)
//  2. Lecture cache mémoire synchrone (TTL court 5 min / long 6 h)
//  3. Merge per-field : cache → DB carry-forward → zéro
//  4. Si le cache était stale (ou miss), spawn une goroutine background
//     détachée qui rafraîchit le cache pour la prochaine requête
//
// Aucun appel HTTP n'est fait dans le chemin synchrone. Latence garantie
// proche du temps DB pur. Le user voit toujours la donnée la plus fraîche
// disponible localement, et le cache se remplit asynchronement quand Halo
// répond. C'est exactement le tradeoff "home fast + données live"
// recherché — plus solide qu'un budget de fetch sync (qui pénalisait
// chaque chargement de home quand Halo ne répond pas dans les temps).
//
// du service ; aujourd'hui le merge est tout-en-mémoire (best-effort sur erreurs
// internes loggées), mais une future intégration LiveAPI pourrait remonter ici.
//
//nolint:unparam // err maintenu en signature pour cohérence avec autres fetchers
func (s *CareerLiveService) fetchAndMerge(ctx context.Context, xuid string) (*duckdb.CareerRankRow, error) {
	tokens := ctxkeys.HaloTokens(ctx)
	hasAuth := tokens != nil && tokens.SpartanToken != ""

	dbLast, dbErr := s.repo.LoadLastCareerRank(ctx, xuid)
	if dbErr != nil {
		slog.WarnContext(ctx, careerLiveLogModule+": LoadLastCareerRank failed",
			"xuid", xuid, "err", dbErr)
		dbLast = nil
	}

	var (
		cachedProgress *syncpkg.CareerRankData
		cachedCustom   *syncpkg.SpartanCustomizationData
		needRefresh    bool
	)
	if hasAuth && s.cache != nil {
		if p, hit := s.cache.GetProgress(xuid); hit {
			cachedProgress = p
			careerLiveProgressCache.Add(1)
		} else {
			needRefresh = true
		}
		if c, hit := s.cache.GetCustomization(xuid); hit {
			cachedCustom = c
			careerLiveCustomCache.Add(1)
		} else {
			needRefresh = true
		}
	}

	merged := mergeCareerRow(cachedProgress, cachedCustom, dbLast)
	if merged != nil {
		if err := s.repo.EnrichFromMetadata(ctx, merged); err != nil {
			slog.WarnContext(ctx, careerLiveLogModule+": EnrichFromMetadata failed",
				"xuid", xuid, "err", err)
		}
	}

	// Stale-while-revalidate : si une partie du cache est absente / expirée,
	// on déclenche un refresh background détaché. La home rend déjà avec ce
	// qu'on a (DB + parts cached). La requête suivante bénéficiera du cache
	// frais — sans avoir attendu Halo dans le chemin critique.
	switch {
	case !hasAuth:
		slog.InfoContext(ctx, careerLiveLogModule+": kickoff skipped",
			"xuid", xuid, "reason", "no_auth_tokens",
			"db_has_row", dbLast != nil)
	case !needRefresh:
		slog.DebugContext(ctx, careerLiveLogModule+": kickoff skipped",
			"xuid", xuid, "reason", "cache_warm")
	default:
		slog.InfoContext(ctx, careerLiveLogModule+": kickoff fired",
			"xuid", xuid,
			"cache_miss_progress", cachedProgress == nil,
			"cache_miss_custom", cachedCustom == nil)
		s.kickoffBackgroundRefresh(xuid, tokens)
	}

	return merged, nil
}

// kickoffBackgroundRefresh lance un refresh asynchrone des deux caches pour
// préparer les prochaines requêtes. Garantit qu'au plus un refresh est actif
// par xuid (dédup via bgInflight + singleflight côté cache).
//
// Le contexte est totalement détaché de la requête caller (pas de WithCancel
// hérité) — le refresh continue même si la home retourne immédiatement.
// Plafonné à careerLiveBgTimeout (30 s).
//
// Les tokens sont capturés en argument (et non lus depuis un ctx) parce
// qu'on quitte le contexte de la requête HTTP : `*domain.HaloTokens` est
// une struct in-memory, sûre à partager tant qu'on ne mute pas. À noter :
// les tokens peuvent expirer pendant ce refresh ; un 401/403 est traité
// comme un fail silencieux par le HaloAPIClient (renvoie nil, nil).
func (s *CareerLiveService) kickoffBackgroundRefresh(xuid string, tokens *domain.HaloTokens) {
	if tokens == nil || tokens.SpartanToken == "" {
		return
	}
	s.bgInflightMu.Lock()
	if s.bgInflight[xuid] {
		s.bgInflightMu.Unlock()
		return
	}
	s.bgInflight[xuid] = true
	s.bgInflightMu.Unlock()

	careerLiveBgRefresh.Add(1)

	go func() {
		defer func() {
			s.bgInflightMu.Lock()
			delete(s.bgInflight, xuid)
			s.bgInflightMu.Unlock()
		}()

		bgCtx, cancel := context.WithTimeout(context.Background(), careerLiveBgTimeout)
		defer cancel()
		bgCtx = ctxkeys.WithHaloAuth(bgCtx, tokens, xuid)

		// On utilise les helpers cachés : ils écrivent dans la cache à la
		// fin du fetch, ce qui est exactement ce qu'on veut pour que la
		// prochaine requête synchrone hit la cache au lieu de retimeout.
		progress := s.fetchProgressCached(bgCtx, xuid)
		custom := s.fetchCustomizationCached(bgCtx, xuid)

		// Persist partial (Phase 2/3 PLAN_V2) : on n'écrit dans la nouvelle
		// ligne QUE les champs effectivement rendus non-vides par l'API live.
		// Les autres restent NULL et ARG_MAX FILTER WHERE NOT NULL côté
		// lecture conserve les valeurs historiques. Pas de pollution possible.
		// Le status (Phase 6) trace l'issue du fetch pour diag.
		status := computeFetchStatus(progress, custom)
		s.persistPartial(bgCtx, xuid, progress, custom, status)

		slog.DebugContext(bgCtx, careerLiveLogModule+": background refresh completed", "xuid", xuid)
	}()
}

// fetchProgressCached, fetchCustomizationCached, makeFetcher → extraits dans
// `career_live_fetcher.go` (refactor V2 dette technique 2026-05-26).

// persistIfChanged écrit le snapshot dans career_progression si différent
// de la dernière row. Best-effort — erreur loguée mais non propagée.
//
// Deprecated: utilise persistPartial pour les nouveaux chemins (V2 PLAN §5).
// Conservé pour compat avec les appels legacy non encore migrés.
func (s *CareerLiveService) persistIfChanged(ctx context.Context, xuid string, row *duckdb.CareerRankRow) {
	if s.repo == nil || row == nil {
		return
	}
	inserted, err := s.repo.InsertCareerProgressionIfChanged(ctx, xuid, row)
	if err != nil {
		slog.WarnContext(ctx, careerLiveLogModule+": persist failed",
			"xuid", xuid, "err", err)
		return
	}
	if inserted {
		careerLiveInsertChanged.Add(1)
		slog.InfoContext(ctx, careerLiveLogModule+": new snapshot inserted",
			"xuid", xuid,
			"rank", row.Rank,
			"current_xp", row.CurrentXP)
	} else {
		careerLiveInsertSkipped.Add(1)
	}
}

// persistPartial écrit dans career_progression UNIQUEMENT les champs
// effectivement rendus non-vides par l'API live (PartialFromLive). Les
// colonnes omises restent NULL dans la nouvelle ligne — la lecture via
// ARG_MAX FILTER WHERE NOT NULL conserve les valeurs historiques.
//
// Phase 6 PLAN_V2 : status trace l'issue du fetch (ok / api_empty /
// forbidden_403 / auth_missing / failed). Toujours écrit pour permettre
// le diag "pourquoi ce joueur n'a pas de bannière".
//
// Best-effort : une erreur est loggée mais non propagée à l'appelant.
func (s *CareerLiveService) persistPartial(
	ctx context.Context,
	xuid string,
	progress *syncpkg.CareerRankData,
	custom *syncpkg.SpartanCustomizationData,
	status FetchStatus,
) {
	if s.repo == nil {
		return
	}
	partial := PartialFromLive(progress, custom)
	statusStr := string(status)
	partial.LastFetchStatus = &statusStr

	inserted, err := s.repo.InsertCareerProgressionPartial(ctx, xuid, partial)
	if err != nil {
		slog.WarnContext(ctx, careerLiveLogModule+": persist partial failed",
			"xuid", xuid, "err", err)
		return
	}
	if inserted {
		careerLiveInsertChanged.Add(1)
		slog.InfoContext(ctx, careerLiveLogModule+": partial snapshot inserted",
			"xuid", xuid,
			"status", statusStr,
			"has_rank", partial.Rank != nil,
			"has_xp", partial.CurrentXP != nil,
			"has_banner", partial.BannerImageURL != nil,
			"has_emblem", partial.EmblemImageURL != nil,
			"has_spartan_id", partial.SpartanID != nil)
	} else {
		careerLiveInsertSkipped.Add(1)
	}
}

// computeFetchStatus dérive le FetchStatus depuis le résultat des 2 fetchs.
// Source de vérité unique pour la classification des outcomes.
func computeFetchStatus(progress *syncpkg.CareerRankData, custom *syncpkg.SpartanCustomizationData) FetchStatus {
	hasProgress := progress != nil && (progress.CurrentRank > 0 || progress.IsMaxRank)
	hasCustom := custom != nil && (custom.SpartanID != "" || custom.BannerImageURL != "" ||
		custom.EmblemImageURL != "" || custom.BackdropImageURL != "")
	if hasProgress || hasCustom {
		return FetchStatusOK
	}
	// Aucune data exploitable. Si les 2 sont nil → l'API a probablement échoué
	// silencieusement (cf. fetchProgressCached qui log "API silent skip" et
	// retourne nil sur data == nil + cache miss). Sinon, c'est un retour vide.
	if progress == nil && custom == nil {
		return FetchStatusAPIEmpty
	}
	return FetchStatusAPIEmpty
}

// serveDBFallback charge la dernière row DB et construit directement la
// HomeSpartanIdentityRow depuis ses valeurs. Retourne nil si DB vide.
// Utilisé quand le live est totalement indisponible (tokens absents, etc.)
// pour ne jamais montrer un bloc Spartan vide à l'utilisateur si une row
// historique existe.
func (s *CareerLiveService) serveDBFallback(ctx context.Context, xuid string) *domain.HomeSpartanIdentityRow {
	var dbRow *duckdb.CareerRankRow
	if xuid != "" && s.repo != nil {
		row, err := s.repo.LoadLastCareerRank(ctx, xuid)
		if err != nil {
			slog.WarnContext(ctx, careerLiveLogModule+": DB fallback load failed",
				"xuid", xuid, "err", err)
		} else if row != nil {
			careerLiveDBFallback.Add(1)
			if metaErr := s.repo.EnrichFromMetadata(ctx, row); metaErr != nil {
				slog.WarnContext(ctx, careerLiveLogModule+": DB fallback metadata failed",
					"xuid", xuid, "err", metaErr)
			}
			dbRow = row
		}
	}
	identity := s.builder.BuildSpartanIdentityFromCareerRow(ctx, dbRow)
	if identity == nil {
		careerLiveIdentityMissing.Add(1)
		careerLiveEmptyResult.Add(1)
		return nil
	}
	careerLiveIdentityServed.Add(1)
	return identity
}

// mergeCareerRow, overlayIdentityFromFallback → extraits dans
// `career_live_merge.go` (refactor V2 dette technique 2026-05-26).
//
// CareerFetcherFactoryFromTokens, compile-time check → extraits dans
// `career_live_fetcher.go`.

// errNoFallback est conservé pour signaler explicitement le cas "rien à
// servir" dans les futurs chemins (CLI manuel, healthcheck). Inutilisé pour
// l'instant côté HTTP.
//
//nolint:unused // export api defensive — supprimer si non câblé d'ici 2026-Q3
var errNoFallback = sql.ErrNoRows
