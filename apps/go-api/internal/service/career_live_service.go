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
// Directive produit apparence : bannière/emblème/backdrop sont des champs
// INDÉPENDANTS ; chacun affiche toujours une valeur (l'actuelle si résoluble,
// sinon la dernière connue), jamais vide, sans couplage entre eux. Cas des
// emblèmes nouvelle génération sans nameplate upstream : la dernière bannière
// connue reste servie (cf. career_live_merge.go).
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
	"levelup/go-api/internal/games"
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
	GetCareerProgress(ctx context.Context, xuid string) (*domain.CareerRankSnapshot, error)
	GetSpartanCustomization(ctx context.Context, xuid string) (*domain.SpartanCustomizationData, error)
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
	LoadLastCareerRank(ctx context.Context, xuid string) (*domain.CareerRankRow, error)
	EnrichFromMetadata(ctx context.Context, row *domain.CareerRankRow) error
	// InsertCareerProgressionIfChanged écrit une copie complète (live + carry-forward).
	// Conservé pour compat tests legacy. Le chemin V2 utilise
	// InsertCareerProgressionPartial qui n'écrit que les champs frais.
	InsertCareerProgressionIfChanged(ctx context.Context, xuid string, data *domain.CareerRankRow) (bool, error)
	// InsertCareerProgressionPartial écrit UNIQUEMENT les champs set du partial,
	// les autres restent NULL (Phase 2/3 PLAN_V2 §5).
	InsertCareerProgressionPartial(ctx context.Context, xuid string, partial *domain.CareerProgressionPartial) (bool, error)
}

// CareerIdentityBuilder construit le HomeSpartanIdentityRow final à partir
// d'une CareerRankRow (déjà mergée) + skill peaks. Implémenté par
// duckdb.HomeRepo.BuildSpartanIdentityFromCareerRow.
type CareerIdentityBuilder interface {
	BuildSpartanIdentityFromCareerRow(ctx context.Context, careerRow *domain.CareerRankRow, includePeaks bool) *domain.HomeSpartanIdentityRow
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

// GetSpartanIdentityFor retourne le bloc Spartan ID complet pour le xuid SUJET
// passé EXPLICITEMENT — jamais lu du contexte ambiant (finding ID4, revue 2026-07 :
// le sujet d'identité est un paramètre, pas une valeur ambiante ; le forçage
// point-par-point de ctxkeys.HaloXUID avait récidivé 4 fois). Les appelants — home
// (pdb.XUID de la page), cron customization (xuid du joueur rafraîchi), Explorer
// (xuid cible) — fournissent le sujet ; ctxkeys.HaloXUID reste réservé à l'ownership
// (subjectIsOwner ci-dessous), sans jouer le rôle de sujet.
//
// **Contrat UI-first** : si la player DB porte une row historique avec une
// bannière (ou emblem/backdrop/spartan_id), on la retourne TOUJOURS. La
// fraîcheur du live ne doit JAMAIS dégrader la visibilité (cf. revue
// 2026-05-20 « les bannières vont et viennent »). Retourne nil uniquement
// quand DB ET live sont tous deux vides (joueur jamais sync'd).
// Chaque asset (bannière/emblème/backdrop) : l'actuel si résoluble, sinon le
// dernier connu — jamais vide tant qu'une valeur a existé ; indépendants.
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
//
// Garde-fou critique : la persistance dans `career_progression` (déclenchée
// par kickoffBackgroundRefresh) n'est activée QUE si le xuid passé est égal
// au xuid du user connecté (subjectIsOwner). Sinon on évite de polluer la player
// DB du user avec les rangs/customisations d'un joueur tiers (cas Explorer).
func (s *CareerLiveService) GetSpartanIdentityFor(ctx context.Context, xuid string) (*domain.HomeSpartanIdentityRow, error) {
	// includePeaks : les skill peaks sont lus sur la player DB du propriétaire
	// de la page. Ils ne sont valides que si le sujet de l'identité EST ce
	// propriétaire (dans le contexte enrichi, HaloXUID == pdb.XUID). Pour un
	// xuid tiers (cas Explorer : joueur cible), on les omet — sinon on afficherait
	// les peaks du propriétaire sur la carte d'un autre joueur et on paierait
	// 2 scans match_skill_rank inutiles. Même condition que allowPersist.
	subjectIsOwner := xuid == ctxkeys.HaloXUID(ctx)

	// Étape 1-2 : filet DB systématique. serveDBFallback est tolérant à
	// xuid="" / repo nil (retourne nil), donc on peut toujours l'appeler.
	dbFallback := s.serveDBFallback(ctx, xuid, subjectIsOwner)
	if xuid == "" {
		return dbFallback, nil
	}

	// allowPersist : la persistance asynchrone n'est ouverte que pour le xuid
	// du user connecté. Pour un xuid tiers (cas Explorer), on lit le cache et
	// on rend le résultat, mais on ne déclenche pas le kickoff background qui
	// écrirait dans career_progression de la player DB courante (qui est
	// celle du user, pas du target).
	allowPersist := subjectIsOwner

	// Étape 3 : tentative live (peut échouer transitoirement).
	merged, err := s.fetchAndMerge(ctx, xuid, allowPersist)
	if err != nil {
		slog.WarnContext(ctx, careerLiveLogModule+": fetch+merge failed → DB fallback",
			"xuid", xuid, "err", err)
		return dbFallback, nil
	}

	// Phase 4 PLAN_V2 : la persistance est déléguée à kickoffBackgroundRefresh
	// (path background) qui a accès au progress+custom bruts. Le path sync
	// se contente de servir ce qu'on a — pas de persist depuis ici, pas de
	// risque d'écrire des champs carry-forward dans une nouvelle ligne.

	identity := s.builder.BuildSpartanIdentityFromCareerRow(ctx, merged, subjectIsOwner)

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
func (s *CareerLiveService) fetchAndMerge(ctx context.Context, xuid string, allowPersist bool) (*domain.CareerRankRow, error) {
	tokens := ctxkeys.HaloTokens(ctx)
	hasAuth := tokens != nil && tokens.SpartanToken != ""

	// Gating title-aware (cause racine S1) : le live careerranks (endpoint
	// economy) n'existe QUE pour les titres exposant le catalogue de rangs de
	// carrière (CapCareerRankCatalog). Un titre comme Halo 5 dérive son SR de la
	// carnage (persisté direct dans career_progression au sync) et n'a pas ce
	// catalogue. Comme son token Spartan reste un token Infinite valide, appeler
	// careerranks renverrait la carrière INFINITE → contamination cross-titre.
	// On court-circuite donc TOUT le chemin live (cache + kickoff/persistPartial)
	// et on sert uniquement le DB fallback (déjà mergé + EnrichFromMetadata).
	// Title-agnostic : capability via resolver, jamais comparaison de slug.
	slug := ctxkeys.TitleSlug(ctx)
	liveCareerProvided := games.ProvidesLiveCareerProgression(slug)

	dbLast, dbErr := s.repo.LoadLastCareerRank(ctx, xuid)
	if dbErr != nil {
		slog.WarnContext(ctx, careerLiveLogModule+": LoadLastCareerRank failed",
			"xuid", xuid, "err", dbErr)
		dbLast = nil
	}

	if !liveCareerProvided {
		slog.DebugContext(ctx, careerLiveLogModule+": live career fetch skipped (title sans careerranks)",
			"xuid", xuid, "title_slug", slug, "db_has_row", dbLast != nil)
		merged := mergeCareerRow(nil, nil, dbLast)
		if merged != nil {
			if err := s.repo.EnrichFromMetadata(ctx, merged); err != nil {
				slog.WarnContext(ctx, careerLiveLogModule+": EnrichFromMetadata failed",
					"xuid", xuid, "err", err)
			}
		}
		return merged, nil
	}

	var (
		cachedProgress *domain.CareerRankSnapshot
		cachedCustom   *domain.SpartanCustomizationData
		needRefresh    bool
	)
	if hasAuth && s.cache != nil {
		if p, hit := s.cache.GetProgress(xuid, slug); hit {
			cachedProgress = p
			careerLiveProgressCache.Add(1)
		} else {
			needRefresh = true
		}
		if c, hit := s.cache.GetCustomization(xuid, slug); hit {
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
	//
	// allowPersist=false (cas xuid tiers depuis Explorer) court-circuite le
	// kickoff parce qu'il écrirait dans career_progression du user connecté
	// (la player DB courante), pas dans celle du target. On garde quand même
	// le bénéfice de la lecture cache + DB fallback.
	switch {
	case !hasAuth:
		slog.InfoContext(ctx, careerLiveLogModule+": kickoff skipped",
			"xuid", xuid, "reason", "no_auth_tokens",
			"db_has_row", dbLast != nil)
	case !allowPersist:
		slog.DebugContext(ctx, careerLiveLogModule+": kickoff skipped",
			"xuid", xuid, "reason", "persist_disabled_third_party_xuid")
	case !needRefresh:
		slog.DebugContext(ctx, careerLiveLogModule+": kickoff skipped",
			"xuid", xuid, "reason", "cache_warm")
	default:
		slog.InfoContext(ctx, careerLiveLogModule+": kickoff fired",
			"xuid", xuid,
			"cache_miss_progress", cachedProgress == nil,
			"cache_miss_custom", cachedCustom == nil)
		// Porteur réel des tokens (finding ID3) : capturé AVANT le détachement du
		// contexte requête pour que le refresh background impute le budget API au
		// compte connecté, pas au sujet de la page. Peut différer de `xuid` quand la
		// page consultée n'est pas celle du porteur (admin/membre de groupe).
		ownerXUID := ctxkeys.TokensOwnerXUID(ctx)
		s.kickoffBackgroundRefresh(xuid, ownerXUID, tokens, slug)
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
//
// `ownerXUID` (porteur réel des tokens, capturé depuis le ctx requête AVANT le
// détachement) est ré-injecté dans le bgCtx pour que le budget API de ce refresh
// s'impute au compte connecté, même quand `xuid` (le sujet) est une autre page
// (finding ID3). Vide → on garde le sujet comme porteur (dégradation best-effort).
//
// `slug` (title slug capturé depuis le ctx de la requête AVANT le détachement)
// est RE-PROPAGÉ dans le bgCtx détaché : sans ça, ctxkeys.TitleSlug(bgCtx)
// retomberait sur "halo_infinite" par défaut et le fetch live careerranks
// ciblerait l'endpoint economy Infinite (cause racine S1 : contamination
// cross-titre). Le gating en amont (fetchAndMerge) n'appelle déjà plus le
// kickoff pour les titres sans careerranks, mais propager le slug rend le
// chemin background correct par construction (defense en profondeur + futurs
// titres qui exposeraient careerranks sur un host distinct via le resolver).
func (s *CareerLiveService) kickoffBackgroundRefresh(xuid, ownerXUID string, tokens *domain.HaloTokens, slug string) {
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
		// WithHaloAuth pose tokensOwnerXUID = xuid (le sujet). Le corriger vers le
		// PORTEUR réel (finding ID3) : le refresh dépense le quota du compte
		// connecté, à imputer à son bucket ratebudget, pas à celui de la page.
		// ownerXUID vide (porteur inconnu) → on garde le sujet (dégradation).
		if ownerXUID != "" {
			bgCtx = ctxkeys.WithTokensOwnerXUID(bgCtx, ownerXUID)
		}
		if slug != "" {
			bgCtx = ctxkeys.WithTitleSlug(bgCtx, slug)
		}

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
	progress *domain.CareerRankSnapshot,
	custom *domain.SpartanCustomizationData,
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
func computeFetchStatus(progress *domain.CareerRankSnapshot, custom *domain.SpartanCustomizationData) FetchStatus {
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
func (s *CareerLiveService) serveDBFallback(ctx context.Context, xuid string, includePeaks bool) *domain.HomeSpartanIdentityRow {
	var dbRow *domain.CareerRankRow
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
	identity := s.builder.BuildSpartanIdentityFromCareerRow(ctx, dbRow, includePeaks)
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
