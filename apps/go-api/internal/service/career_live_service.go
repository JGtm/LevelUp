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
	InsertCareerProgressionIfChanged(ctx context.Context, xuid string, data *duckdb.CareerRankRow) (bool, error)
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
// Garantie de non-vidage : si la player DB porte une row historique, on la
// retourne en fallback même si tous les appels live échouent. Retourne nil
// uniquement quand DB et live sont tous deux vides (joueur jamais sync'd).
func (s *CareerLiveService) GetSpartanIdentity(ctx context.Context) (*domain.HomeSpartanIdentityRow, error) {
	xuid := ctxkeys.HaloXUID(ctx)
	if xuid == "" {
		// Pas de xuid en contexte : on tente la lecture DB seule pour ne pas
		// vider l'écran si un fallback existe (cas: handler sans middleware
		// session, tests, etc.).
		return s.serveDBFallback(ctx, ""), nil
	}

	merged, err := s.fetchAndMerge(ctx, xuid)
	if err != nil {
		slog.WarnContext(ctx, careerLiveLogModule+": fetch+merge failed → DB fallback",
			"xuid", xuid, "err", err)
		return s.serveDBFallback(ctx, xuid), nil
	}

	// Persist (INSERT-if-changed) avant de retourner — fire-and-forget mais
	// dans le même flow (cohérence inter-requêtes garantie).
	if merged != nil {
		s.persistIfChanged(ctx, xuid, merged)
	}

	identity := s.builder.BuildSpartanIdentityFromCareerRow(ctx, merged)
	if identity == nil {
		careerLiveIdentityMissing.Add(1)
		careerLiveEmptyResult.Add(1)
		return nil, nil
	}
	careerLiveIdentityServed.Add(1)
	return identity, nil
}

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
	if hasAuth && needRefresh {
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
		_ = s.fetchProgressCached(bgCtx, xuid)
		_ = s.fetchCustomizationCached(bgCtx, xuid)
		slog.DebugContext(bgCtx, careerLiveLogModule+": background refresh completed", "xuid", xuid)
	}()
}

// fetchProgressCached retourne la progression depuis le cache si frais, sinon
// fait l'appel live (avec singleflight). Erreurs live → log warn + nil.
func (s *CareerLiveService) fetchProgressCached(ctx context.Context, xuid string) *syncpkg.CareerRankData {
	if s.cache != nil {
		if cached, hit := s.cache.GetProgress(xuid); hit {
			careerLiveProgressCache.Add(1)
			slog.DebugContext(ctx, careerLiveLogModule+": progress cache hit", "xuid", xuid)
			return cached
		}
	}

	fetcher := s.makeFetcher(ctx)
	if fetcher == nil {
		return nil
	}

	fetch := func() (*syncpkg.CareerRankData, error) {
		return fetcher.GetCareerProgress(ctx, xuid)
	}
	var (
		data *syncpkg.CareerRankData
		err  error
	)
	if s.cache != nil {
		data, err = s.cache.DoProgress(xuid, fetch)
	} else {
		data, err = fetch()
	}
	if err != nil {
		careerLiveProgressFail.Add(1)
		slog.WarnContext(ctx, careerLiveLogModule+": progress fetch failed",
			"xuid", xuid, "err", err)
		return nil
	}
	careerLiveProgressLive.Add(1)
	if s.cache != nil {
		s.cache.PutProgress(xuid, data)
	}
	return data
}

// fetchCustomizationCached : pendant pour la customisation (TTL 6 h).
func (s *CareerLiveService) fetchCustomizationCached(ctx context.Context, xuid string) *syncpkg.SpartanCustomizationData {
	if s.cache != nil {
		if cached, hit := s.cache.GetCustomization(xuid); hit {
			careerLiveCustomCache.Add(1)
			slog.DebugContext(ctx, careerLiveLogModule+": customization cache hit", "xuid", xuid)
			return cached
		}
	}

	fetcher := s.makeFetcher(ctx)
	if fetcher == nil {
		return nil
	}
	fetch := func() (*syncpkg.SpartanCustomizationData, error) {
		return fetcher.GetSpartanCustomization(ctx, xuid)
	}
	var (
		data *syncpkg.SpartanCustomizationData
		err  error
	)
	if s.cache != nil {
		data, err = s.cache.DoCustomization(xuid, fetch)
	} else {
		data, err = fetch()
	}
	if err != nil {
		careerLiveCustomFail.Add(1)
		slog.WarnContext(ctx, careerLiveLogModule+": customization fetch failed",
			"xuid", xuid, "err", err)
		return nil
	}
	careerLiveCustomLive.Add(1)
	if s.cache != nil {
		s.cache.PutCustomization(xuid, data)
	}
	return data
}

// makeFetcher construit un fetcher depuis le contexte. Retourne nil si la
// factory n'est pas câblée ou si elle elle-même retourne nil (tokens absents).
func (s *CareerLiveService) makeFetcher(ctx context.Context) CareerFetcher {
	if s.fetcherFactory == nil {
		return nil
	}
	return s.fetcherFactory(ctx)
}

// persistIfChanged écrit le snapshot dans career_progression si différent
// de la dernière row. Best-effort — erreur loguée mais non propagée.
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

// mergeCareerRow fusionne progress (live) + custom (live) + dbLast (carry-
// forward) en une seule CareerRankRow. Ordre de priorité par champ :
//
//	live (si non-vide) → dbLast → zéro-valeur
//
// Si live retourne quelque chose pour un champ mais que la valeur est zéro
// ou chaîne vide, le DB l'écrase. C'est exactement le comportement « per-
// field fallback » attendu : on ne remplace jamais une valeur connue par
// une valeur vide remontée d'un fetch partiellement réussi.
//
// Retourne nil si toutes les sources sont vides.
func mergeCareerRow(
	progress *syncpkg.CareerRankData,
	custom *syncpkg.SpartanCustomizationData,
	dbLast *duckdb.CareerRankRow,
) *duckdb.CareerRankRow {
	if progress == nil && custom == nil && dbLast == nil {
		return nil
	}

	merged := &duckdb.CareerRankRow{}
	mergedPerField := false

	// Progress : rank, current_xp, is_max_rank.
	if progress != nil {
		merged.Rank = progress.CurrentRank
		merged.CurrentXP = progress.CurrentXP
		merged.IsMaxRank = progress.IsMaxRank
	}
	if dbLast != nil {
		if merged.Rank <= 0 && dbLast.Rank > 0 {
			merged.Rank = dbLast.Rank
			mergedPerField = true
		}
		// current_xp : on ne réécrit JAMAIS depuis dbLast quand progress live
		// a réussi — même un current_xp=0 live est l'état réel du joueur
		// (palier juste franchi). Le carry-forward ne sert qu'en l'absence
		// totale de live.
		if progress == nil && merged.CurrentXP == 0 {
			merged.CurrentXP = dbLast.CurrentXP
		}
		if progress == nil {
			merged.IsMaxRank = dbLast.IsMaxRank
		}
	}

	// Customization : spartan_id, banner, emblem, backdrop.
	if custom != nil {
		merged.SpartanID = custom.SpartanID
		merged.BannerImageURL = custom.BannerImageURL
		merged.EmblemImageURL = custom.EmblemImageURL
		merged.BackdropImageURL = custom.BackdropImageURL
	}
	if dbLast != nil {
		if merged.SpartanID == "" && dbLast.SpartanID != "" {
			merged.SpartanID = dbLast.SpartanID
			mergedPerField = true
		}
		if merged.BannerImageURL == "" && dbLast.BannerImageURL != "" {
			merged.BannerImageURL = dbLast.BannerImageURL
			mergedPerField = true
		}
		if merged.EmblemImageURL == "" && dbLast.EmblemImageURL != "" {
			merged.EmblemImageURL = dbLast.EmblemImageURL
			mergedPerField = true
		}
		if merged.BackdropImageURL == "" && dbLast.BackdropImageURL != "" {
			merged.BackdropImageURL = dbLast.BackdropImageURL
			mergedPerField = true
		}
		// Champs purement dérivés : rank_name, rank_tier, xp_for_next_rank,
		// xp_total, adornment_path. EnrichFromMetadata les recalculera depuis
		// merged.Rank ; le carry-forward depuis dbLast n'est utile que si
		// metadata est indisponible.
		if merged.RankName == "" {
			merged.RankName = dbLast.RankName
		}
		if merged.RankTier == "" {
			merged.RankTier = dbLast.RankTier
		}
		if merged.XPForNextRank == 0 {
			merged.XPForNextRank = dbLast.XPForNextRank
		}
		if merged.XPTotal == 0 {
			merged.XPTotal = dbLast.XPTotal
		}
		if merged.AdornmentPath == "" {
			merged.AdornmentPath = dbLast.AdornmentPath
		}
	}

	if mergedPerField {
		careerLivePerFieldMerge.Add(1)
	}
	if merged.IsEmpty() {
		return nil
	}
	return merged
}

// CareerFetcherFactoryFromTokens retourne une factory qui instancie un
// HaloAPIClient depuis les tokens du contexte. requestsPerSecond contrôle le
// rate limiting du client (defaults à 10 si <= 0).
//
// Le client est jetable : un nouvel objet par requête. Le coût d'allocation
// est négligeable comparé au HTTP call lui-même, et permet de bénéficier
// systématiquement des tokens à jour (refresh rotation handled in middleware).
func CareerFetcherFactoryFromTokens(requestsPerSecond int) CareerFetcherFactory {
	return func(ctx context.Context) CareerFetcher {
		tokens := ctxkeys.HaloTokens(ctx)
		if tokens == nil || tokens.SpartanToken == "" {
			return nil
		}
		return syncpkg.NewHaloAPIClient(tokens.SpartanToken, tokens.ClearanceToken, requestsPerSecond)
	}
}

// Compile-time check : sync.HaloAPIClient implémente bien CareerFetcher.
var _ CareerFetcher = (*syncpkg.HaloAPIClient)(nil)

// errNoFallback est conservé pour signaler explicitement le cas "rien à
// servir" dans les futurs chemins (CLI manuel, healthcheck). Inutilisé pour
// l'instant côté HTTP.
//
//nolint:unused // export api defensive — supprimer si non câblé d'ici 2026-Q3
var errNoFallback = sql.ErrNoRows