// Package service — home_service.go : service de la page d'accueil Mission Control.
package service

import (
	"context"
	"log/slog"
	"strconv"
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
type homePageData struct {
	matches        []domain.HomeMatchRow
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
// Retourne également un booléen indiquant si les données viennent du cache.
//
// P4.1 (ADR 0011) : si playerMatchesRepo + titleSlug + gamertag sont fournis,
// charge canonical.PlayerMatchRow et dérive les deux listes via les
// converters home*FromCanonical. Sinon fallback repo direct (LoadHomeMatches
// + LoadHomeSessions en parallèle). Le cache TTL fonctionne dans les deux
// cas — il est indexé par xuid et stocke les types domain.* finaux.
func (s *HomeService) fetchMatchesAndSessions(ctx context.Context) (
	matches []domain.HomeMatchRow, sessions []domain.HomeSessionRow, fromCache bool, err error,
) {
	if s.matchesCache != nil && s.xuid != "" {
		if m, sess, hit := s.matchesCache.Get(s.xuid); hit {
			slog.DebugContext(ctx, "home_cache: hit", "xuid", s.xuid, "matches", len(m))
			return m, sess, true, nil
		}
	}

	if s.playerMatchesRepo != nil && s.titleSlug != "" && s.gamertag != "" {
		canonicalRows, e := s.playerMatchesRepo.LoadPlayerMatches(
			ctx, s.titleSlug, s.gamertag, port.PlayerMatchFilters{},
		)
		if e != nil {
			return nil, nil, false, e
		}
		matches = homeMatchRowsFromCanonical(canonicalRows)
		sessions = homeSessionsFromCanonical(canonicalRows)
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
			return nil, nil, false, err
		}
	}

	if s.matchesCache != nil && s.xuid != "" {
		s.matchesCache.Set(s.xuid, matches, sessions)
	}
	slog.DebugContext(ctx, "home_cache: miss — données rechargées", "xuid", s.xuid, "matches", len(matches))
	return matches, sessions, false, nil
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
		d.matches, d.sessions, cacheHit, err = s.fetchMatchesAndSessions(gctx)
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
func (s *HomeService) GetHomePage(ctx context.Context, gamertag, locale string) (*domain.HomePageResponse, error) {
	d, err := s.fetchHomePageData(ctx, locale)
	if err != nil {
		return nil, err
	}

	hasRankedHistory, hasUnrankedHistory := inferHomeSkillHistory(d.matches)

	hero := analysis.BuildHeroCard(d.matches, gamertag, d.totalMatches)
	if d.favWeaponName != "" {
		hero.KPIs.FavoriteWeaponName = d.favWeaponName
		hero.KPIs.FavoriteWeaponKills = d.favWeaponKills
	}

	highlights := analysis.BuildHighlights(d.matches)
	recentMatches := analysis.BuildRecentMatchesWithFavoritesForLocale(d.matches, len(d.matches), d.favoriteIDs, locale)
	favoriteMatches := buildFavoriteMatchList(d.matches, d.favoriteIDs, locale)

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
		SoloSession:         analysis.BuildSessionSummary(d.matches, d.sessions, false),
		SquadSession:        analysis.BuildSessionSummary(d.matches, d.sessions, true),
		SoloSessions:        analysis.BuildSessionSummaries(d.matches, d.sessions, false, 20),
		SquadSessions:       analysis.BuildSessionSummaries(d.matches, d.sessions, true, 20),
		HasRankedHistory:    hasRankedHistory,
		HasUnrankedHistory:  hasUnrankedHistory,
		RecentPlaylistRanks: d.playlistRanks,
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
// P4.1 (ADR 0011) : converters canonical → home types
// =============================================================================

// homeMatchRowFromCanonical convertit une canonical.PlayerMatchRow vers le format
// domain.HomeMatchRow consommé par les fonctions analysis.Build* (HeroCard,
// Highlights, RecentMatches, SessionSummary).
//
// Mapping selon ADR 0011 :
//   - Données brutes (kills, deaths, MMR, …) : depuis canonical.
//   - Libellés map/playlist/game_variant FR : depuis AssetReference.Labels["fr"]
//     avec fallback DefaultLabel (le canonical les transporte déjà localisés
//     pour les locales chargées).
//   - SkillTierLabel et SkillRankImageURL : LAISSÉS VIDES — TODO P4.3 :
//     enrichir via TitleSemanticAdapter.Ranks() (label) et
//     TitleAssetURLAdapter.CSRRankImageURL (URL) une fois ces adapters câblés
//     en CSR-tier-aware sur le service.
//   - PairID/PairName/PairNameFR : composite Halo-only — laissés vides (P4.3).
//
// TODO P4.3 : retirer cette conversion quand BuildHeroCard/BuildHighlights/etc.
// consommeront canonical directement.
func homeMatchRowFromCanonical(r canonical.PlayerMatchRow) domain.HomeMatchRow {
	out := domain.HomeMatchRow{
		MatchID:          r.Summary.MatchID,
		StartTime:        r.Summary.StartedAtUTC,
		SessionLabel:     r.Enrichment.SessionLabel,
		IsWithFriends:    r.Enrichment.IsWithFriends,
		Kills:            homeDerefInt(r.Self.Kills),
		Deaths:           homeDerefInt(r.Self.Deaths),
		Assists:          homeDerefInt(r.Self.Assists),
		KDA:              r.Self.KDA,
		Accuracy:         r.Self.Accuracy,
		AvgLifeSeconds:   r.Self.AvgLifeSeconds,
		TimePlayedSecs:   r.Self.TimePlayed,
		TeamMMR:          r.Enrichment.TeamMMR,
		EnemyMMR:         r.Enrichment.EnemyMMR,
		PerformanceScore: r.Enrichment.PerformanceScore,
		HeadshotKills:    homeDerefInt(r.Self.HeadshotKills),
		PerfectKills:     homeDerefInt(r.Self.PerfectKills),
		MaxKillingSpree:  r.Self.MaxKillingSpree,
		DominanceFlag:    int(r.Enrichment.DominanceFlag),
	}
	if r.Self.TeamID != nil {
		out.TeamID = *r.Self.TeamID
	}
	if r.Self.RankInMatch != nil {
		v := *r.Self.RankInMatch
		out.RankInTeam = &v
	}
	// Damage : canonical *int → home *float64
	if r.Self.DamageDealt != nil {
		v := float64(*r.Self.DamageDealt)
		out.DamageDealt = &v
	}
	if r.Self.DamageTaken != nil {
		v := float64(*r.Self.DamageTaken)
		out.DamageTaken = &v
	}
	// Map / Playlist / GameVariant : depuis AssetReference si présent.
	if r.Summary.Map != nil {
		out.MapID = r.Summary.Map.ID
		out.MapName = r.Summary.Map.DefaultLabel
		if v, ok := r.Summary.Map.Labels["fr"]; ok && v != "" {
			out.MapNameFR = v
		} else {
			out.MapNameFR = r.Summary.Map.DefaultLabel
		}
	}
	if r.Summary.Playlist != nil {
		out.PlaylistID = r.Summary.Playlist.ID
		out.PlaylistName = r.Summary.Playlist.DefaultLabel
		if v, ok := r.Summary.Playlist.Labels["fr"]; ok && v != "" {
			out.PlaylistNameFR = v
		} else {
			out.PlaylistNameFR = r.Summary.Playlist.DefaultLabel
		}
	}
	if r.Summary.GameVariant != nil {
		out.GameVariantID = r.Summary.GameVariant.ID
		out.GameVariantName = r.Summary.GameVariant.DefaultLabel
		if v, ok := r.Summary.GameVariant.Labels["fr"]; ok && v != "" {
			out.GameVariantNameFR = v
		} else {
			out.GameVariantNameFR = r.Summary.GameVariant.DefaultLabel
		}
	}
	// IsRanked / IsFirefight : depuis Summary.
	if r.Summary.IsRanked != nil {
		out.IsRanked = *r.Summary.IsRanked
	}
	if r.Summary.IsPvE != nil {
		out.IsFirefight = *r.Summary.IsPvE
	}
	// Outcome : canonical → int Halo (1=Tie, 2=Win, 3=Loss, 4=DNF).
	switch r.Self.Outcome {
	case canonical.OutcomeWin:
		out.Outcome = domain.OutcomeWin
	case canonical.OutcomeLoss:
		out.Outcome = domain.OutcomeLoss
	case canonical.OutcomeTie:
		out.Outcome = domain.OutcomeDraw
	case canonical.OutcomeDNF:
		out.Outcome = domain.OutcomeDNF
	}
	// Team scores : depuis Summary.Teams.
	for _, t := range r.Summary.Teams {
		if t.Score == nil {
			continue
		}
		switch t.TeamID {
		case 0:
			out.Team0Score = *t.Score
		case 1:
			out.Team1Score = *t.Score
		}
	}
	// SkillSnapshot : data fields uniquement. Tier label + image URL = TODO P4.3.
	if r.Enrichment.SkillSnapshot != nil {
		out.SkillRatingValue = r.Enrichment.SkillSnapshot.RatingValue
		out.SkillRatingType = string(r.Enrichment.SkillSnapshot.RatingType)
		out.SkillTier = r.Enrichment.SkillSnapshot.TierCode
		if r.Enrichment.SkillSnapshot.SubTier != nil {
			out.SkillSubTier = *r.Enrichment.SkillSnapshot.SubTier
		}
		out.SkillRatingDelta = r.Enrichment.SkillSnapshot.Delta
		out.SkillPlaylistGroup = r.Enrichment.SkillSnapshot.PlaylistGroup
		// SkillTierLabel : TODO P4.3 — TitleSemanticAdapter pour CSR/LUSR tier.
		// SkillRankImageURL : TODO P4.3 — TitleAssetURLAdapter.CSRRankImageURL.
	}
	// Ratio : KDR canonique calculé (analysis.KDR), distinct de KDA.
	if r.Self.Deaths != nil && *r.Self.Deaths > 0 && r.Self.Kills != nil {
		v := float64(*r.Self.Kills) / float64(*r.Self.Deaths)
		out.Ratio = &v
	}
	return out
}

// homeMatchRowsFromCanonical : version slice. TODO P4.3.
func homeMatchRowsFromCanonical(rows []canonical.PlayerMatchRow) []domain.HomeMatchRow {
	out := make([]domain.HomeMatchRow, len(rows))
	for i, r := range rows {
		out[i] = homeMatchRowFromCanonical(r)
	}
	return out
}

// homeSessionsFromCanonical dérive la liste HomeSessionRow depuis les rows
// canonical (1 entrée par match avec Enrichment.SessionID/SessionLabel).
//
// Conversion SessionID : canonical *string → home *int via strconv. Si parse
// échoue, SessionID reste nil (le sync legacy garantit des IDs numériques).
//
// TODO P4.3 : retirer ce converter quand domain.HomeSessionRow sera supprimé
// au profit de canonical.PlayerMatchEnrichment direct.
func homeSessionsFromCanonical(rows []canonical.PlayerMatchRow) []domain.HomeSessionRow {
	out := make([]domain.HomeSessionRow, 0, len(rows))
	for _, r := range rows {
		entry := domain.HomeSessionRow{
			MatchID:       r.Summary.MatchID,
			SessionLabel:  r.Enrichment.SessionLabel,
			IsWithFriends: r.Enrichment.IsWithFriends,
		}
		if r.Enrichment.SessionID != nil {
			if id, err := strconv.Atoi(*r.Enrichment.SessionID); err == nil {
				entry.SessionID = &id
			}
		}
		t := r.Summary.StartedAtUTC
		entry.StartTime = &t
		out = append(out, entry)
	}
	return out
}

// homeDerefInt retourne *p ou 0 si p est nil. Utilitaire converter local.
func homeDerefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
