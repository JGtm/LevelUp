// Package service — home_service.go : service de la page d'accueil Mission Control.
package service

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/halo"
	"levelup/go-api/internal/port"

	"levelup/go-api/internal/platform/duckdb"
)

// Durées de vie du cache BP/Challenges selon l'état de session.
const (
	battlePassCacheTTLDefault = 1 * time.Hour   // hors session
	battlePassCacheTTLActive  = 5 * time.Minute // session active (symétrie avec liveRefreshInterval)
)

// HomeService orchestre les données de la page d'accueil.
type HomeService struct {
	repo       port.HomeRepository
	cacheRepo  port.BattlePassCacheRepository
	provider   *halo.HaloProvider
	sink       *duckdb.PersistSink // nil → pas de persistance (tests, joueurs sans auth)
	socialRepo port.SocialRepository
	playerSlug string
	sessionTTL atomic.Int64 // TTL en nanosecondes ; 0 = défaut (1h)
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

// SetSessionActive bascule le TTL cache BP/Challenges selon la présence du joueur.
// true → 5 min (symétrie avec liveRefreshInterval), false → 1 h (défaut).
// Implémente port.SessionNotifier. Thread-safe via atomic.Int64.
func (s *HomeService) SetSessionActive(active bool) {
	if active {
		s.sessionTTL.Store(battlePassCacheTTLActive.Nanoseconds())
		slog.Info("home_service: session active — TTL cache réduit",
			"ttl", battlePassCacheTTLActive,
			"player", s.playerSlug,
		)
	} else {
		s.sessionTTL.Store(0)
		slog.Info("home_service: session inactive — TTL cache restauré",
			"ttl", battlePassCacheTTLDefault,
			"player", s.playerSlug,
		)
	}
}

// currentTTL retourne le TTL effectif selon l'état de session courant.
func (s *HomeService) currentTTL() time.Duration {
	if ns := s.sessionTTL.Load(); ns > 0 {
		return time.Duration(ns)
	}
	return battlePassCacheTTLDefault
}

// GetHomePage retourne la page d'accueil agrégée (hero card, highlights, matchs récents,
// médias récents, résumés de sessions solo et escouade).
func (s *HomeService) GetHomePage(ctx context.Context, gamertag, locale string) (*domain.HomePageResponse, error) {
	matches, err := s.repo.LoadHomeMatches(ctx)
	if err != nil {
		return nil, err
	}

	spartanIdentity, err := s.repo.LoadSpartanIdentity(ctx)
	if err != nil {
		slog.WarnContext(ctx, "home: LoadSpartanIdentity failed", "err", err)
		spartanIdentity = nil
	}

	totalMatches, err := s.repo.CountPlayerMatches(ctx)
	if err != nil {
		// Fallback sur len(matches) si la query échoue.
		totalMatches = len(matches)
	}

	sessions, err := s.repo.LoadHomeSessions(ctx)
	if err != nil {
		return nil, err
	}

	media, err := s.repo.LoadRecentMedia(ctx, 4)
	if err != nil {
		// Médias non critiques — on continue sans eux.
		media = nil
	}

	playlistRanks, err := s.repo.LoadRecentPlaylistRanks(ctx, locale)
	if err != nil {
		slog.WarnContext(ctx, "home: LoadRecentPlaylistRanks failed", "err", err)
		playlistRanks = nil
	}
	hasRankedHistory, hasUnrankedHistory := inferHomeSkillHistory(matches)

	// Charger les favoris si le social repo est disponible (dégradation silencieuse sinon).
	var favoriteIDs map[string]bool
	if s.socialRepo != nil && s.playerSlug != "" {
		if ids, err := s.socialRepo.GetFavoriteMatchIDs(ctx, s.playerSlug); err == nil {
			favoriteIDs = ids
		} else {
			slog.WarnContext(ctx, "home: GetFavoriteMatchIDs failed", "err", err)
		}
	}

	hero := analysis.BuildHeroCard(matches, gamertag, totalMatches)

	// Arme favorite — dégradation silencieuse si la table weapon_kills est absente.
	if wName, wKills, wErr := s.repo.LoadFavoriteWeapon(ctx, locale); wErr == nil && wName != "" {
		hero.KPIs.FavoriteWeaponName = wName
		hero.KPIs.FavoriteWeaponKills = wKills
	}

	highlights := analysis.BuildHighlights(matches)
	recentMatches := analysis.BuildRecentMatchesWithFavoritesForLocale(matches, len(matches), favoriteIDs, locale)
	favoriteMatches := buildFavoriteMatchList(recentMatches, matches, favoriteIDs, locale)

	// Enrichissement médailles : batch sur tous les match_id récents + favoris.
	enrichMatchesWithMedals(ctx, s.repo, recentMatches)
	enrichMatchesWithMedals(ctx, s.repo, favoriteMatches)

	// Enrichissement citations : batch sur les mêmes lots.
	enrichMatchesWithCitations(ctx, s.repo, recentMatches)
	enrichMatchesWithCitations(ctx, s.repo, favoriteMatches)

	recentMedia := analysis.BuildRecentMedia(media, 4)
	soloSession := analysis.BuildSessionSummary(matches, sessions, false)
	squadSession := analysis.BuildSessionSummary(matches, sessions, true)
	soloSessions := analysis.BuildSessionSummaries(matches, sessions, false, 20)
	squadSessions := analysis.BuildSessionSummaries(matches, sessions, true, 20)

	return &domain.HomePageResponse{
		Hero:                hero,
		SpartanIdentity:     analysis.BuildSpartanIdentity(spartanIdentity, locale),
		Highlights:          highlights,
		RecentMatches:       recentMatches,
		FavoriteMatches:     favoriteMatches,
		RecentMedia:         recentMedia,
		SoloSession:         soloSession,
		SquadSession:        squadSession,
		SoloSessions:        soloSessions,
		SquadSessions:       squadSessions,
		HasRankedHistory:    hasRankedHistory,
		HasUnrankedHistory:  hasUnrankedHistory,
		RecentPlaylistRanks: playlistRanks,
	}, nil
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
func buildFavoriteMatchList(recent []domain.RecentMatchItem, all []domain.HomeMatchRow, favoriteIDs map[string]bool, locale string) []domain.RecentMatchItem {
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

// GetBattlePass retourne les infos Battle Pass (cache DB d'abord, live en fallback).
// Le TTL cache est dynamique : 5 min pendant une session active, 1 h sinon.
// Si un PersistSink est configuré et que le live est appelé, les données sont persistées
// de manière synchrone avant le retour (garantit que loadTrackSnapshots lit un rang à jour).
func (s *HomeService) GetBattlePass(ctx context.Context) domain.BattlePassResponse {
	ttl := s.currentTTL()
	if s.cacheRepo != nil {
		if cached, hit, err := s.cacheRepo.LoadCachedBattlePass(ctx, ttl); err == nil && hit {
			slog.DebugContext(ctx, "home: BattlePass servi depuis cache DB", "ttl_used", ttl)
			return *cached
		}
	}
	slog.DebugContext(ctx, "home: BattlePass cache miss → appel live", "ttl_used", ttl)
	resp, raw := s.provider.GetBattlePassWithRaw(ctx)
	if s.sink != nil && resp.Available && resp.RewardTrack != nil {
		if err := s.sink.PersistBattlePassSync(ctx, *resp.RewardTrack, raw); err != nil {
			slog.WarnContext(ctx, "home: BattlePass persist failed", "err", err)
		}
	}
	return resp
}

// GetChallenges retourne les défis actifs (cache DB d'abord, live en fallback).
// Le TTL cache est dynamique : 5 min pendant une session active, 1 h sinon (symétrie GetBattlePass).
// Si un PersistSink est configuré et que le live est appelé, les snapshots sont persistés
// en arrière-plan (fire-and-forget).
func (s *HomeService) GetChallenges(ctx context.Context) domain.ChallengesResponse {
	ttl := s.currentTTL()
	if s.cacheRepo != nil {
		if cached, hit, err := s.cacheRepo.LoadCachedChallenges(ctx, ttl); err == nil && hit {
			if cacheChallengesAreRenderable(cached) {
				slog.DebugContext(ctx, "home: Challenges servis depuis cache DB", "ttl_used", ttl)
				return *cached
			}
			slog.DebugContext(ctx, "home: Challenges cache incomplet → fallback live", "ttl_used", ttl)
		}
	}
	slog.DebugContext(ctx, "home: Challenges cache miss → appel live", "ttl_used", ttl)
	resp, raw := s.provider.GetChallengesWithRaw(ctx)
	if s.sink != nil && resp.Available {
		s.sink.PersistChallenges(raw)
	}
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
