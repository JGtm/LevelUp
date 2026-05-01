// Package service â€” home_service.go : service de la page d'accueil Mission Control.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/halo"
	"levelup/go-api/internal/port"

	"levelup/go-api/internal/platform/duckdb"
)

// DurÃ©es de vie du cache BP/Challenges.
const (
	battlePassCacheTTLFallback = 24 * time.Hour // fallback si live indisponible â€” accepte mÃªme des donnÃ©es vieilles
)

// HomeService orchestre les donnÃ©es de la page d'accueil.
type HomeService struct {
	repo         port.HomeRepository
	cacheRepo    port.BattlePassCacheRepository
	provider     *halo.HaloProvider
	sink         *duckdb.PersistSink // nil â†’ pas de persistance (tests, joueurs sans auth)
	socialRepo   port.SocialRepository
	playerSlug   string
	semantic     games.TitleSemanticAdapter // nil â†’ libellÃ©s rangs construits via fallbacks (RankName)
	matchesCache *HomeMatchesCache          // nil â†’ pas de cache (tests, appels HomeCtx sans auth)
	xuid         string                     // clÃ© de cache ; vide si matchesCache est nil
	// dataAdapter (optionnel, Phase C+ multi-titres) : point d'extension pour
	// router LoadPlayerStats via la couche canonique. Ã€ ce jour, le service
	// utilise le repo direct car canonical.PlayerStats ne couvre pas encore
	// la totalitÃ© du payload home KPIs (favorite_playlist, avg_kda, etc.).
	// Le hook est en place pour permettre une bascule incrÃ©mentale.
	dataAdapter games.TitleDataAdapter
	// playerMatchesRepo (P4.1, ADR 0011) : loader canonical-aware optionnel.
	// Quand fourni avec titleSlug + gamertag, fetchMatchesAndSessions charge
	// canonical et convertit via homeMatchRowFromCanonical / homeSessionsFromCanonical.
	// SkillTierLabel et SkillRankImageURL sont laissÃ©s vides cÃ´tÃ© converter
	// (TODO P4.3 : enrichir via TitleSemanticAdapter.Ranks() et
	// TitleAssetURLAdapter.CSRRankImageURL une fois les adapters CSR cÃ¢blÃ©s
	// dans le service).
	playerMatchesRepo port.PlayerMatchesRepository
	titleSlug         string
	gamertag          string
}

// NewHomeService crÃ©e un HomeService avec le repository et le provider Halo.
func NewHomeService(repo port.HomeRepository) *HomeService {
	return &HomeService{
		repo:     repo,
		provider: halo.DefaultHaloProvider,
	}
}

// WithHaloProvider remplace le provider Halo utilisÃ© par le service.
// Utile pour injecter un provider configurÃ© par joueur (cache local, tests).
func (s *HomeService) WithHaloProvider(provider *halo.HaloProvider) *HomeService {
	if provider != nil {
		s.provider = provider
	}
	return s
}

// WithPersistSink configure le sink de persistance fire-and-forget.
// Retourne le service pour permettre le chaÃ®nage.
func (s *HomeService) WithPersistSink(sink *duckdb.PersistSink) *HomeService {
	s.sink = sink
	return s
}

// WithCacheRepo configure le repository de cache BP/Challenges.
// Retourne le service pour permettre le chaÃ®nage.
func (s *HomeService) WithCacheRepo(r port.BattlePassCacheRepository) *HomeService {
	s.cacheRepo = r
	return s
}

// WithSocial configure le repository social (favoris) et le slug joueur.
// Retourne le service pour permettre le chaÃ®nage.
func (s *HomeService) WithSocial(repo port.SocialRepository, playerSlug string) *HomeService {
	s.socialRepo = repo
	s.playerSlug = playerSlug
	return s
}

// WithSemanticAdapter injecte le SemanticAdapter du titre courant pour rÃ©soudre
// les libellÃ©s des rangs de carriÃ¨re (Ranks() expose un *mappings.RankCatalog).
// Si nil, les libellÃ©s tombent sur le fallback RankName de la player DB.
func (s *HomeService) WithSemanticAdapter(semantic games.TitleSemanticAdapter) *HomeService {
	s.semantic = semantic
	return s
}

// WithDataAdapter injecte le DataAdapter multi-titres pour activer une
// future bascule LoadPlayerStats. DÃ©gradation gracieuse si nil.
func (s *HomeService) WithDataAdapter(a games.TitleDataAdapter) *HomeService {
	s.dataAdapter = a
	return s
}

// WithPlayerMatchesRepo (P4.1, ADR 0011) injecte le loader canonical-aware
// + titleSlug + gamertag. Quand les 3 sont fournis, fetchMatchesAndSessions
// charge canonical et convertit. Sinon fallback repo legacy.
func (s *HomeService) WithPlayerMatchesRepo(repo port.PlayerMatchesRepository, titleSlug, gamertag string) *HomeService {
	s.playerMatchesRepo = repo
	s.titleSlug = titleSlug
	s.gamertag = gamertag
	return s
}

// WithMatchesCache active le cache TTL process-level pour LoadHomeMatches + LoadHomeSessions.
// xuid est la clÃ© de cache ; sans lui le cache ne peut pas fonctionner.
func (s *HomeService) WithMatchesCache(cache *HomeMatchesCache, xuid string) *HomeService {
	s.matchesCache = cache
	s.xuid = xuid
	return s
}

// SetSessionActive implÃ©mente port.SessionNotifier.
// ConservÃ© pour compatibilitÃ© avec le watcher â€” aucun effet sur le handler HTTP
// qui appelle toujours le live directement.
func (s *HomeService) SetSessionActive(_ bool) {
}

// homePageData regroupe toutes les donnÃ©es brutes chargÃ©es en parallÃ¨le par fetchHomePageData.
//
// P4.3b (ADR 0011) : `canonicalRows` est renseignÃ© quand le path canonical est
// actif (playerMatchesRepo + titleSlug + gamertag). `matches`/`sessions`
// restent renseignÃ©s pour la rÃ©trocompatibilitÃ© (legacy fallback path).
type homePageData struct {
	matches        []legacymatch.HomeMatchRow
	canonicalRows  []canonical.PlayerMatchRow // nil = legacy path
	spartanIdent   *domain.HomeSpartanIdentityRow
	totalMatches   int
	sessions       []legacymatch.HomeSessionRow
	media          []domain.HomeMediaRow
	playlistRanks  []domain.HomePlaylistRank
	favoriteIDs    map[string]bool
	favWeaponName  string
	favWeaponKills int
}

// fetchMatchesAndSessions charge les rows canonical du joueur (P4.3 finale).
// Retourne aussi un cache du `bool fromCache` pour tÃ©lÃ©mÃ©trie.
//
// P4.3 finale (ADR 0011) : path canonical exclusif. playerMatchesRepo +
// titleSlug + gamertag sont REQUIS (wirÃ©s en DI universellement). Le legacy
// fallback (LoadHomeMatches + LoadHomeSessions parallel) a Ã©tÃ© supprimÃ©.
//
// Le cache TTL stocke encore les rows canonical pour rÃ©trocompat avec les
// signatures Get/Set existantes (qui prennent matches/sessions legacy) â€” la
// suppression du cache legacy est tracker dans une follow-up dÃ©diÃ©e.
func (s *HomeService) fetchMatchesAndSessions(ctx context.Context) (
	canonicalRows []canonical.PlayerMatchRow, fromCache bool, err error,
) {
	if s.matchesCache != nil && s.xuid != "" {
		if _, _, hit := s.matchesCache.Get(s.xuid); hit {
			// Cache hit : on doit reconstruire les canonical rows. Le cache n'est
			// pas encore canonical-aware ; pour P4.3 finale on bypass le cache hit
			// et recharge canonical. TODO P4.4 : adapter HomeMatchesCache Ã
			// canonical.
			slog.DebugContext(ctx, "home_cache: hit (bypass P4.3 finale)", "xuid", s.xuid)
		}
	}

	if s.playerMatchesRepo == nil || s.titleSlug == "" || s.gamertag == "" {
		return nil, false, fmt.Errorf("HomeService: PlayerMatchesRepo non cÃ¢blÃ© (P4.3 finale exige le wiring DI)")
	}
	rows, e := s.playerMatchesRepo.LoadPlayerMatches(
		ctx, s.titleSlug, s.gamertag, port.PlayerMatchFilters{},
	)
	if e != nil {
		return nil, false, e
	}
	canonicalRows = rows
	slog.DebugContext(ctx, "home: loaded canonical",
		"rows", len(canonicalRows), "title_slug", s.titleSlug)

	// Maintien du cache pour la mÃ©trique (set vide pour invalider stale).
	if s.matchesCache != nil && s.xuid != "" {
		s.matchesCache.Set(s.xuid, nil, nil)
	}
	return canonicalRows, false, nil
}

// fetchHomePageData charge toutes les donnÃ©es de la page d'accueil en parallÃ¨le.
// Les erreurs non-critiques sont absorbÃ©es (dÃ©gradation silencieuse).
func (s *HomeService) fetchHomePageData(ctx context.Context, locale string) (homePageData, error) {
	var d homePageData

	// Groupe 1 : matches+sessions (cache TTL) en parallÃ¨le avec les autres appels lÃ©gers.
	var cacheHit bool
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		d.canonicalRows, cacheHit, err = s.fetchMatchesAndSessions(gctx)
		return err
	})
	g.Go(func() error {
		var err error
		d.spartanIdent, err = s.repo.LoadSpartanIdentity(gctx)
		if err != nil {
			slog.WarnContext(gctx, "home: LoadSpartanIdentity failed", "err", err)
		}
		return nil
	})
	g.Go(func() error {
		// Fallback sur len(matches) aprÃ¨s le Wait si la query Ã©choue (totalMatches reste 0).
		d.totalMatches, _ = s.repo.CountPlayerMatches(gctx)
		return nil
	})
	g.Go(func() error {
		var err error
		d.media, err = s.repo.LoadRecentMedia(gctx, 4)
		if err != nil {
			slog.WarnContext(gctx, "home: LoadRecentMedia failed", "err", err)
		}
		return nil
	})
	g.Go(func() error {
		var err error
		d.playlistRanks, err = s.repo.LoadRecentPlaylistRanks(gctx, locale)
		if err != nil {
			slog.WarnContext(gctx, "home: LoadRecentPlaylistRanks failed", "err", err)
		}
		return nil
	})
	g.Go(func() error {
		if wName, wKills, err := s.repo.LoadFavoriteWeapon(gctx, locale); err == nil && wName != "" {
			d.favWeaponName = wName
			d.favWeaponKills = wKills
		}
		return nil
	})
	if s.socialRepo != nil && s.playerSlug != "" {
		slug := s.playerSlug
		g.Go(func() error {
			if ids, err := s.socialRepo.GetFavoriteMatchIDs(gctx, slug); err == nil {
				d.favoriteIDs = ids
			} else {
				slog.WarnContext(gctx, "home: GetFavoriteMatchIDs failed", "err", err)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return homePageData{}, err
	}
	if d.totalMatches == 0 {
		d.totalMatches = len(d.canonicalRows)
	}
	_ = cacheHit // exploitable pour des mÃ©triques futures
	return d, nil
}

// GetHomePage retourne la page d'accueil agrÃ©gÃ©e (hero card, highlights, matchs rÃ©cents,
// mÃ©dias rÃ©cents, rÃ©sumÃ©s de sessions solo et escouade).
//
// P4.3 finale (ADR 0011) : path canonical exclusif. Toutes les analyses
// passent par les `analysis.*FromCanonical`. Le legacy fallback a Ã©tÃ© supprimÃ©.
func (s *HomeService) GetHomePage(ctx context.Context, gamertag, locale string) (*domain.HomePageResponse, error) {
	defer func(start time.Time) {
		observability.RecordDurationMS("home_get_page", time.Since(start).Milliseconds())
	}(time.Now())
	d, err := s.fetchHomePageData(ctx, locale)
	if err != nil {
		return nil, err
	}

	// Bug #2/#7 cascade : remplit Labels["fr"] des AssetReference depuis
	// metadata.asset_translations + mode_name_tr quand match_registry
	// .{...}_name_fr est NULL. Sans ça, modes/maps/playlists restent en EN.
	if err := s.repo.EnrichCanonicalAssetTranslations(ctx, d.canonicalRows); err != nil {
		slog.WarnContext(ctx, "home: EnrichCanonicalAssetTranslations failed", "err", err)
	}

	hasRankedHistory, hasUnrankedHistory := analysis.InferHomeSkillHistoryFromCanonical(d.canonicalRows)
	hero := analysis.BuildHeroCardFromCanonical(d.canonicalRows, gamertag, d.totalMatches, locale)
	highlights := analysis.BuildHighlightsFromCanonical(d.canonicalRows)
	recentMatches := analysis.BuildRecentMatchesWithFavoritesFromCanonical(d.canonicalRows, len(d.canonicalRows), d.favoriteIDs, locale)
	favoriteMatches := buildFavoriteMatchListCanonical(d.canonicalRows, d.favoriteIDs, locale)
	soloSession := analysis.BuildSessionSummaryFromCanonical(d.canonicalRows, false, locale)
	squadSession := analysis.BuildSessionSummaryFromCanonical(d.canonicalRows, true, locale)
	soloSessions := analysis.BuildSessionSummariesFromCanonical(d.canonicalRows, false, 20, locale)
	squadSessions := analysis.BuildSessionSummariesFromCanonical(d.canonicalRows, true, 20, locale)

	if d.favWeaponName != "" {
		hero.KPIs.FavoriteWeaponName = d.favWeaponName
		hero.KPIs.FavoriteWeaponKills = d.favWeaponKills
	}

	// Enrichissement mÃ©dailles + citations par liste en parallÃ¨le.
	enrichG, _ := errgroup.WithContext(ctx)
	enrichG.Go(func() error {
		enrichMatchesWithMedals(ctx, s.repo, recentMatches)
		enrichMatchesWithCitations(ctx, s.repo, recentMatches)
		return nil
	})
	enrichG.Go(func() error {
		enrichMatchesWithMedals(ctx, s.repo, favoriteMatches)
		enrichMatchesWithCitations(ctx, s.repo, favoriteMatches)
		return nil
	})
	_ = enrichG.Wait()

	var rankCatalog *mappings.RankCatalog
	if s.semantic != nil {
		rankCatalog = s.semantic.Ranks()
	}

	return &domain.HomePageResponse{
		Hero:                hero,
		SpartanIdentity:     analysis.BuildSpartanIdentity(d.spartanIdent, locale, rankCatalog),
		Highlights:          highlights,
		RecentMatches:       recentMatches,
		FavoriteMatches:     favoriteMatches,
		RecentMedia:         analysis.BuildRecentMedia(d.media, 4),
		SoloSession:         soloSession,
		SquadSession:        squadSession,
		SoloSessions:        soloSessions,
		SquadSessions:       squadSessions,
		HasRankedHistory:    hasRankedHistory,
		HasUnrankedHistory:  hasUnrankedHistory,
		RecentPlaylistRanks: d.playlistRanks,
	}, nil
}

// buildFavoriteMatchListCanonical est la variante canonical-aware de
// buildFavoriteMatchList. DÃ©lÃ¨gue Ã  la version legacy via le wrapper analysis.
func buildFavoriteMatchListCanonical(
	rows []canonical.PlayerMatchRow,
	favoriteIDs map[string]bool,
	locale string,
) []domain.RecentMatchItem {
	if len(favoriteIDs) == 0 {
		return nil
	}
	allItems := analysis.BuildRecentMatchesWithFavoritesFromCanonical(rows, len(rows), favoriteIDs, locale)
	var favorites []domain.RecentMatchItem
	for _, item := range allItems {
		if item.IsFavorite {
			favorites = append(favorites, item)
		}
	}
	return favorites
}

func inferHomeSkillHistory(matches []legacymatch.HomeMatchRow) (bool, bool) {
	hasRankedHistory := false
	hasUnrankedHistory := false
	for _, match := range matches {
		if match.IsFirefight {
			continue
		}
		if match.IsRanked {
			hasRankedHistory = true
		} else {
			hasUnrankedHistory = true
		}
		if hasRankedHistory && hasUnrankedHistory {
			break
		}
	}
	return hasRankedHistory, hasUnrankedHistory
}

// enrichMatchesWithMedals injecte les TopMedals (max 4, sÃ©lection par raretÃ©/count)
// dans chaque RecentMatchItem via un appel batch sur le repo.
func enrichMatchesWithMedals(ctx context.Context, repo port.HomeRepository, items []domain.RecentMatchItem) {
	if len(items) == 0 {
		return
	}
	matchIDs := make([]string, len(items))
	for i, item := range items {
		matchIDs[i] = item.MatchID
	}
	medalsMap, err := repo.LoadMatchMedals(ctx, matchIDs)
	if err != nil || len(medalsMap) == 0 {
		return
	}
	for i, item := range items {
		if all, ok := medalsMap[item.MatchID]; ok {
			items[i].TopMedals = selectTopMedals(all, 4)
		}
	}
}

// selectTopMedals sÃ©lectionne au plus n mÃ©dailles parmi la liste, en privilÃ©giant
// les mÃ©dailles avec le plus grand count (dÃ©jÃ  triÃ©es count DESC par Q26h).
func selectTopMedals(medals []domain.RecentMatchMedal, n int) []domain.RecentMatchMedal {
	if len(medals) <= n {
		return medals
	}
	return medals[:n]
}

// maxCitationSnippets est le nombre maximum de citations affichÃ©es par MatchCard.
const maxCitationSnippets = 3

// enrichMatchesWithCitations injecte les TopCitations (max 3, filtre citations dÃ©jÃ  masterisÃ©es)
// dans chaque RecentMatchItem via un appel batch sur le repo.
func enrichMatchesWithCitations(ctx context.Context, repo port.HomeRepository, items []domain.RecentMatchItem) {
	if len(items) == 0 {
		return
	}
	matchIDs := make([]string, len(items))
	for i, item := range items {
		matchIDs[i] = item.MatchID
	}
	citationsMap, err := repo.LoadMatchCitations(ctx, matchIDs)
	if err != nil || len(citationsMap) == 0 {
		return
	}
	for i, item := range items {
		if rows, ok := citationsMap[item.MatchID]; ok && len(rows) > 0 {
			items[i].TopCitations = analysis.BuildCitationSnippets(rows, maxCitationSnippets)
		}
	}
}

// buildFavoriteMatchList construit la liste des matchs favoris Ã  partir de tous les matchs
// chargÃ©s (pas limitÃ©s Ã  6), en appliquant le flag IsFavorite.
func buildFavoriteMatchList(
	all []legacymatch.HomeMatchRow,
	favoriteIDs map[string]bool,
	locale string,
) []domain.RecentMatchItem {
	if len(favoriteIDs) == 0 {
		return nil
	}
	// Construire la liste complÃ¨te des matchs favoris (pas limitÃ©s aux 6 rÃ©cents).
	allItems := analysis.BuildRecentMatchesWithFavoritesForLocale(all, len(all), favoriteIDs, locale)
	var favorites []domain.RecentMatchItem
	for _, item := range allItems {
		if item.IsFavorite {
			favorites = append(favorites, item)
		}
	}
	return favorites
}

// GetBattlePass retourne les infos Battle Pass (live d'abord, cache DB en fallback).
// Appel live systÃ©matique pour garantir des donnÃ©es fraÃ®ches au rechargement de page.
// Si le live Ã©choue (tokens absents, API indisponible), le cache DB est retournÃ©.
// Si un PersistSink est configurÃ© et que le live rÃ©ussit, les donnÃ©es sont persistÃ©es
// de maniÃ¨re synchrone avant le retour (garantit que loadTrackSnapshots lit un rang Ã  jour).
func (s *HomeService) GetBattlePass(ctx context.Context) domain.BattlePassResponse {
	resp, raw := s.provider.GetBattlePassWithRaw(ctx)
	if resp.Available && resp.RewardTrack != nil {
		slog.DebugContext(ctx, "home: BattlePass obtenu depuis API live")
		if s.sink != nil {
			if err := s.sink.PersistBattlePassSync(ctx, *resp.RewardTrack, raw); err != nil {
				slog.WarnContext(ctx, "home: BattlePass persist failed", "err", err)
			}
		}
		return resp
	}
	// Live indisponible (pas de tokens, erreur rÃ©seau) â†’ fallback cache DB.
	if s.cacheRepo != nil {
		if cached, hit, err := s.cacheRepo.LoadCachedBattlePass(ctx, battlePassCacheTTLFallback); err == nil && hit {
			slog.DebugContext(ctx, "home: BattlePass live indisponible â€” fallback cache DB")
			return *cached
		}
	}
	slog.DebugContext(ctx, "home: BattlePass live indisponible, aucun cache disponible")
	return resp
}

// GetChallenges retourne les dÃ©fis actifs (live d'abord, cache DB en fallback).
// Appel live systÃ©matique pour garantir des donnÃ©es fraÃ®ches au rechargement de page.
// Si le live Ã©choue (tokens absents, API indisponible), le cache DB est retournÃ©.
func (s *HomeService) GetChallenges(ctx context.Context) domain.ChallengesResponse {
	resp, raw := s.provider.GetChallengesWithRaw(ctx)
	if resp.Available {
		slog.DebugContext(ctx, "home: Challenges obtenus depuis API live")
		if s.sink != nil {
			s.sink.PersistChallenges(raw)
		}
		return resp
	}
	// Live indisponible â†’ fallback cache DB.
	if s.cacheRepo != nil {
		if cached, hit, err := s.cacheRepo.LoadCachedChallenges(ctx, battlePassCacheTTLFallback); err == nil && hit {
			if cacheChallengesAreRenderable(cached) {
				slog.DebugContext(ctx, "home: Challenges live indisponibles â€” fallback cache DB")
				return *cached
			}
		}
	}
	slog.DebugContext(ctx, "home: Challenges live indisponibles, aucun cache disponible")
	return resp
}

func cacheChallengesAreRenderable(resp *domain.ChallengesResponse) bool {
	if resp == nil {
		return false
	}
	if len(resp.Items) > 0 {
		return true
	}
	if resp.Total != nil && resp.Completed != nil && *resp.Total > *resp.Completed {
		return false
	}
	return true
}

// =============================================================================
// P4.3b (ADR 0011) : les converters canonical â†’ home types ont Ã©tÃ© dÃ©placÃ©s
// dans `analysis/home_canonical.go` (encapsulÃ©s derriÃ¨re les wrappers
// `analysis.*FromCanonical`). Le service ne porte plus de logique de
// conversion : il consomme les wrappers directement.
// =============================================================================
