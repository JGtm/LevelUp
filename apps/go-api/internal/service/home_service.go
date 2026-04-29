// Package service — home_service.go : service de la page d'accueil Mission Control.
package service

import (
	"context"
	"log/slog"
	"time"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/platform/halo"
	"levelup/go-api/internal/port"

	"levelup/go-api/internal/platform/duckdb"
)

// Durées de vie du cache BP/Challenges.
const (
	battlePassCacheTTLFallback = 24 * time.Hour // fallback si live indisponible — accepte même des données vieilles
)

// HomeService orchestre les données de la page d'accueil.
type HomeService struct {
	repo         port.HomeRepository
	cacheRepo    port.BattlePassCacheRepository
	provider     *halo.HaloProvider
	sink         *duckdb.PersistSink // nil → pas de persistance (tests, joueurs sans auth)
	socialRepo   port.SocialRepository
	playerSlug   string
	semantic     games.TitleSemanticAdapter // nil → libellés rangs construits via fallbacks (RankName)
	matchesCache *HomeMatchesCache          // nil → pas de cache (tests, appels HomeCtx sans auth)
	xuid         string                     // clé de cache ; vide si matchesCache est nil
	// dataAdapter (optionnel, Phase C+ multi-titres) : point d'extension pour
	// router LoadPlayerStats via la couche canonique. À ce jour, le service
	// utilise le repo direct car canonical.PlayerStats ne couvre pas encore
	// la totalité du payload home KPIs (favorite_playlist, avg_kda, etc.).
	// Le hook est en place pour permettre une bascule incrémentale.
	dataAdapter games.TitleDataAdapter
	// playerMatchesRepo (P4.1, ADR 0011) : loader canonical-aware optionnel.
	// Quand fourni avec titleSlug + gamertag, fetchMatchesAndSessions charge
	// canonical et convertit via homeMatchRowFromCanonical / homeSessionsFromCanonical.
	// SkillTierLabel et SkillRankImageURL sont laissés vides côté converter
	// (TODO P4.3 : enrichir via TitleSemanticAdapter.Ranks() et
	// TitleAssetURLAdapter.CSRRankImageURL une fois les adapters CSR câblés
	// dans le service).
	playerMatchesRepo port.PlayerMatchesRepository
	titleSlug         string
	gamertag          string
}

// NewHomeService crée un HomeService avec le repository et le provider Halo.
func NewHomeService(repo port.HomeRepository) *HomeService {
	return &HomeService{
		repo:     repo,
		provider: halo.DefaultHaloProvider,
	}
}

// WithHaloProvider remplace le provider Halo utilisé par le service.
// Utile pour injecter un provider configuré par joueur (cache local, tests).
func (s *HomeService) WithHaloProvider(provider *halo.HaloProvider) *HomeService {
	if provider != nil {
		s.provider = provider
	}
	return s
}

// WithPersistSink configure le sink de persistance fire-and-forget.
// Retourne le service pour permettre le chaînage.
func (s *HomeService) WithPersistSink(sink *duckdb.PersistSink) *HomeService {
	s.sink = sink
	return s
}

// WithCacheRepo configure le repository de cache BP/Challenges.
// Retourne le service pour permettre le chaînage.
func (s *HomeService) WithCacheRepo(r port.BattlePassCacheRepository) *HomeService {
	s.cacheRepo = r
	return s
}

// WithSocial configure le repository social (favoris) et le slug joueur.
// Retourne le service pour permettre le chaînage.
func (s *HomeService) WithSocial(repo port.SocialRepository, playerSlug string) *HomeService {
	s.socialRepo = repo
	s.playerSlug = playerSlug
	return s
}

// WithSemanticAdapter injecte le SemanticAdapter du titre courant pour résoudre
// les libellés des rangs de carrière (Ranks() expose un *mappings.RankCatalog).
// Si nil, les libellés tombent sur le fallback RankName de la player DB.
func (s *HomeService) WithSemanticAdapter(semantic games.TitleSemanticAdapter) *HomeService {
	s.semantic = semantic
	return s
}

// WithDataAdapter injecte le DataAdapter multi-titres pour activer une
// future bascule LoadPlayerStats. Dégradation gracieuse si nil.
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
// xuid est la clé de cache ; sans lui le cache ne peut pas fonctionner.
func (s *HomeService) WithMatchesCache(cache *HomeMatchesCache, xuid string) *HomeService {
	s.matchesCache = cache
	s.xuid = xuid
	return s
}

// SetSessionActive implémente port.SessionNotifier.
// Conservé pour compatibilité avec le watcher — aucun effet sur le handler HTTP
// qui appelle toujours le live directement.
func (s *HomeService) SetSessionActive(_ bool) {
}

// homePageData regroupe toutes les données brutes chargées en parallèle par fetchHomePageData.
//
// P4.3b (ADR 0011) : `canonicalRows` est renseigné quand le path canonical est
// actif (playerMatchesRepo + titleSlug + gamertag). `matches`/`sessions`
// restent renseignés pour la rétrocompatibilité (legacy fallback path).
type homePageData struct {
	matches        []domain.HomeMatchRow
	canonicalRows  []canonical.PlayerMatchRow // nil = legacy path
	spartanIdent   *domain.HomeSpartanIdentityRow
	totalMatches   int
	sessions       []domain.HomeSessionRow
	media          []domain.HomeMediaRow
	playlistRanks  []domain.HomePlaylistRank
	favoriteIDs    map[string]bool
	favWeaponName  string
	favWeaponKills int
}

// fetchMatchesAndSessions charge matches + sessions depuis le cache TTL ou le repo.
// Retourne également les canonicalRows (nil si legacy path) et un booléen
// indiquant si les données viennent du cache.
//
// P4.1 + P4.3b (ADR 0011) : si playerMatchesRepo + titleSlug + gamertag sont
// fournis, charge canonical.PlayerMatchRow et expose les rows canonical
// directement (pas de conversion service-level). Le cache TTL stocke les
// matches/sessions legacy pour rétrocompat ; la conversion canonical → legacy
// est encapsulée par les wrappers `analysis.*FromCanonical` quand requise.
func (s *HomeService) fetchMatchesAndSessions(ctx context.Context) (
	matches []domain.HomeMatchRow, sessions []domain.HomeSessionRow, canonicalRows []canonical.PlayerMatchRow, fromCache bool, err error,
) {
	if s.matchesCache != nil && s.xuid != "" {
		if m, sess, hit := s.matchesCache.Get(s.xuid); hit {
			slog.DebugContext(ctx, "home_cache: hit", "xuid", s.xuid, "matches", len(m))
			return m, sess, nil, true, nil
		}
	}

	if s.playerMatchesRepo != nil && s.titleSlug != "" && s.gamertag != "" {
		rows, e := s.playerMatchesRepo.LoadPlayerMatches(
			ctx, s.titleSlug, s.gamertag, port.PlayerMatchFilters{},
		)
		if e != nil {
			return nil, nil, nil, false, e
		}
		canonicalRows = rows
		// Conversion seulement pour le cache TTL (qui stocke encore le format legacy).
		// Les analyses passent ensuite par les wrappers *FromCanonical.
		matches = analysis.HomeMatchRowsFromCanonical(canonicalRows)
		sessions = analysis.HomeSessionsFromCanonical(canonicalRows)
		slog.DebugContext(ctx, "home: loaded canonical",
			"rows", len(canonicalRows), "title_slug", s.titleSlug)
	} else {
		g, gctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			var e error
			matches, e = s.repo.LoadHomeMatches(gctx)
			return e
		})
		g.Go(func() error {
			var e error
			sessions, e = s.repo.LoadHomeSessions(gctx)
			return e
		})
		if err = g.Wait(); err != nil {
			return nil, nil, nil, false, err
		}
	}

	if s.matchesCache != nil && s.xuid != "" {
		s.matchesCache.Set(s.xuid, matches, sessions)
	}
	slog.DebugContext(ctx, "home_cache: miss — données rechargées", "xuid", s.xuid, "matches", len(matches))
	return matches, sessions, canonicalRows, false, nil
}

// fetchHomePageData charge toutes les données de la page d'accueil en parallèle.
// Les erreurs non-critiques sont absorbées (dégradation silencieuse).
func (s *HomeService) fetchHomePageData(ctx context.Context, locale string) (homePageData, error) {
	var d homePageData

	// Groupe 1 : matches+sessions (cache TTL) en parallèle avec les autres appels légers.
	var cacheHit bool
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		d.matches, d.sessions, d.canonicalRows, cacheHit, err = s.fetchMatchesAndSessions(gctx)
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
		// Fallback sur len(matches) après le Wait si la query échoue (totalMatches reste 0).
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
		d.totalMatches = len(d.matches)
	}
	_ = cacheHit // exploitable pour des métriques futures
	return d, nil
}

// GetHomePage retourne la page d'accueil agrégée (hero card, highlights, matchs récents,
// médias récents, résumés de sessions solo et escouade).
//
// P4.3b (ADR 0011) : si d.canonicalRows est renseigné (path canonical), les
// analyses passent par les wrappers `analysis.*FromCanonical`. Sinon, fallback
// sur les analyses legacy consommant `[]domain.HomeMatchRow` directement.
func (s *HomeService) GetHomePage(ctx context.Context, gamertag, locale string) (*domain.HomePageResponse, error) {
	d, err := s.fetchHomePageData(ctx, locale)
	if err != nil {
		return nil, err
	}

	useCanonical := d.canonicalRows != nil

	var hasRankedHistory, hasUnrankedHistory bool
	var hero domain.HomeHeroCard
	var highlights []domain.HighlightItem
	var recentMatches, favoriteMatches []domain.RecentMatchItem
	var soloSession, squadSession *domain.SessionSummaryItem
	var soloSessions, squadSessions []domain.SessionSummaryItem

	if useCanonical {
		hasRankedHistory, hasUnrankedHistory = analysis.InferHomeSkillHistoryFromCanonical(d.canonicalRows)
		hero = analysis.BuildHeroCardFromCanonical(d.canonicalRows, gamertag, d.totalMatches)
		highlights = analysis.BuildHighlightsFromCanonical(d.canonicalRows)
		recentMatches = analysis.BuildRecentMatchesWithFavoritesFromCanonical(d.canonicalRows, len(d.canonicalRows), d.favoriteIDs, locale)
		favoriteMatches = buildFavoriteMatchListCanonical(d.canonicalRows, d.favoriteIDs, locale)
		soloSession = analysis.BuildSessionSummaryFromCanonical(d.canonicalRows, false)
		squadSession = analysis.BuildSessionSummaryFromCanonical(d.canonicalRows, true)
		soloSessions = analysis.BuildSessionSummariesFromCanonical(d.canonicalRows, false, 20)
		squadSessions = analysis.BuildSessionSummariesFromCanonical(d.canonicalRows, true, 20)
	} else {
		hasRankedHistory, hasUnrankedHistory = inferHomeSkillHistory(d.matches)
		hero = analysis.BuildHeroCard(d.matches, gamertag, d.totalMatches)
		highlights = analysis.BuildHighlights(d.matches)
		recentMatches = analysis.BuildRecentMatchesWithFavoritesForLocale(d.matches, len(d.matches), d.favoriteIDs, locale)
		favoriteMatches = buildFavoriteMatchList(d.matches, d.favoriteIDs, locale)
		soloSession = analysis.BuildSessionSummary(d.matches, d.sessions, false)
		squadSession = analysis.BuildSessionSummary(d.matches, d.sessions, true)
		soloSessions = analysis.BuildSessionSummaries(d.matches, d.sessions, false, 20)
		squadSessions = analysis.BuildSessionSummaries(d.matches, d.sessions, true, 20)
	}

	if d.favWeaponName != "" {
		hero.KPIs.FavoriteWeaponName = d.favWeaponName
		hero.KPIs.FavoriteWeaponKills = d.favWeaponKills
	}

	// Enrichissement médailles + citations par liste en parallèle.
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
// buildFavoriteMatchList. Délègue à la version legacy via le wrapper analysis.
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

func inferHomeSkillHistory(matches []domain.HomeMatchRow) (bool, bool) {
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

// enrichMatchesWithMedals injecte les TopMedals (max 4, sélection par rareté/count)
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

// selectTopMedals sélectionne au plus n médailles parmi la liste, en privilégiant
// les médailles avec le plus grand count (déjà triées count DESC par Q26h).
func selectTopMedals(medals []domain.RecentMatchMedal, n int) []domain.RecentMatchMedal {
	if len(medals) <= n {
		return medals
	}
	return medals[:n]
}

// maxCitationSnippets est le nombre maximum de citations affichées par MatchCard.
const maxCitationSnippets = 3

// enrichMatchesWithCitations injecte les TopCitations (max 3, filtre citations déjà masterisées)
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

// buildFavoriteMatchList construit la liste des matchs favoris à partir de tous les matchs
// chargés (pas limités à 6), en appliquant le flag IsFavorite.
func buildFavoriteMatchList(
	all []domain.HomeMatchRow,
	favoriteIDs map[string]bool,
	locale string,
) []domain.RecentMatchItem {
	if len(favoriteIDs) == 0 {
		return nil
	}
	// Construire la liste complète des matchs favoris (pas limités aux 6 récents).
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
// Appel live systématique pour garantir des données fraîches au rechargement de page.
// Si le live échoue (tokens absents, API indisponible), le cache DB est retourné.
// Si un PersistSink est configuré et que le live réussit, les données sont persistées
// de manière synchrone avant le retour (garantit que loadTrackSnapshots lit un rang à jour).
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
	// Live indisponible (pas de tokens, erreur réseau) → fallback cache DB.
	if s.cacheRepo != nil {
		if cached, hit, err := s.cacheRepo.LoadCachedBattlePass(ctx, battlePassCacheTTLFallback); err == nil && hit {
			slog.DebugContext(ctx, "home: BattlePass live indisponible — fallback cache DB")
			return *cached
		}
	}
	slog.DebugContext(ctx, "home: BattlePass live indisponible, aucun cache disponible")
	return resp
}

// GetChallenges retourne les défis actifs (live d'abord, cache DB en fallback).
// Appel live systématique pour garantir des données fraîches au rechargement de page.
// Si le live échoue (tokens absents, API indisponible), le cache DB est retourné.
func (s *HomeService) GetChallenges(ctx context.Context) domain.ChallengesResponse {
	resp, raw := s.provider.GetChallengesWithRaw(ctx)
	if resp.Available {
		slog.DebugContext(ctx, "home: Challenges obtenus depuis API live")
		if s.sink != nil {
			s.sink.PersistChallenges(raw)
		}
		return resp
	}
	// Live indisponible → fallback cache DB.
	if s.cacheRepo != nil {
		if cached, hit, err := s.cacheRepo.LoadCachedChallenges(ctx, battlePassCacheTTLFallback); err == nil && hit {
			if cacheChallengesAreRenderable(cached) {
				slog.DebugContext(ctx, "home: Challenges live indisponibles — fallback cache DB")
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
// P4.3b (ADR 0011) : les converters canonical → home types ont été déplacés
// dans `analysis/home_canonical.go` (encapsulés derrière les wrappers
// `analysis.*FromCanonical`). Le service ne porte plus de logique de
// conversion : il consomme les wrappers directement.
// =============================================================================
