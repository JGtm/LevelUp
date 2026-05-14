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
// Fallback à 4 niveaux (la home ne doit jamais montrer un bloc Spartan vide
// si la player DB porte des données historiques) :
//
//  1. live OK, champs complets         → utilise les valeurs live
//  2. live OK, certains champs vides   → per-field merge depuis la dernière
//     row connue en DB (carry-forward)
//  3. live KO complet                  → fallback total sur la dernière row
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

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
	syncpkg "levelup/go-api/internal/sync"
)

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
	repo            CareerLiveRepo
	builder         CareerIdentityBuilder
	fetcherFactory  CareerFetcherFactory
	cache           *CareerLiveCache
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

// fetchAndMerge réalise les fetches live (cache-aware + singleflight) puis
// merge per-field avec la dernière row DB pour combler les champs vides.
//
// Retourne (nil, nil) si aucune donnée n'a pu être obtenue ni live ni DB
// (cas joueur jamais sync'd). Une erreur n'est retournée que pour des bugs
// inattendus côté DB ; les erreurs live sont absorbées (fallback silencieux).
func (s *CareerLiveService) fetchAndMerge(ctx context.Context, xuid string) (*duckdb.CareerRankRow, error) {
	tokens := ctxkeys.HaloTokens(ctx)
	hasAuth := tokens != nil && tokens.SpartanToken != ""

	var (
		progress *syncpkg.CareerRankData
		custom   *syncpkg.SpartanCustomizationData
	)

	if hasAuth {
		progress = s.fetchProgressCached(ctx, xuid)
		custom = s.fetchCustomizationCached(ctx, xuid)
	}

	// Lecture DB systématique : utilisée pour
	//   - per-field merge si live partiel
	//   - fallback complet si live KO ou pas d'auth
	//   - source du compare avant INSERT-if-changed
	dbLast, dbErr := s.repo.LoadLastCareerRank(ctx, xuid)
	if dbErr != nil {
		slog.WarnContext(ctx, careerLiveLogModule+": LoadLastCareerRank failed",
			"xuid", xuid, "err", dbErr)
		dbLast = nil
	}

	merged := mergeCareerRow(progress, custom, dbLast)
	if merged == nil {
		return nil, nil
	}

	// Enrichissement metadata (rank_name, rank_tier, xp_for_next_rank, xp_total).
	// Best-effort : si la table metadata est absente, on conserve les valeurs
	// portées par la DB row (toujours mieux que rien).
	if err := s.repo.EnrichFromMetadata(ctx, merged); err != nil {
		slog.WarnContext(ctx, careerLiveLogModule+": EnrichFromMetadata failed",
			"xuid", xuid, "err", err)
	}
	return merged, nil
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