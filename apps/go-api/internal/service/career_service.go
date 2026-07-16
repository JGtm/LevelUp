// Package service — CareerService : page Carrière, top matches, encounters.
//
// Port Go de career_service.py (apps/api/app/services/career_service.py)
// et career_logic.py (src/ui/pages/career_logic.py).
//
// Le code est découpé en fichiers thématiques pour respecter la limite des
// 500 lignes par fichier (CLAUDE.md). Ce fichier contient le constructeur,
// les Withers de configuration, GetCareerCSRs, GetCareerPage et les helpers
// directement liés (loadLatestRank, resolveCurrentSeason). Les autres
// responsabilités vivent dans :
//
//   - career_service_highlights.go : section "Matchs marquants" (filtres + cascade)
//   - career_service_encounters.go : top matches, encounters, rivals
//   - career_service_summary.go    : builders summary / hero / projections / LUSR
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/teammates"
)

// trendLabelStable est le label retourné quand une tendance est stable
// (delta nul, fenêtre identique). Partagé entre career_service et timeseries_service.
const trendLabelStable = "stable"

// Constantes domaine Halo Infinite.
const (
	xpHeroTotal       = 9_319_350
	rankMax           = 272
	inactivityGapDays = 14.0
)

// FriendsXPLoader charge l'historique XP de tous les amis d'un joueur.
// Retourne une liste vide (sans erreur) si aucun ami n'a de données.
type FriendsXPLoader func(ctx context.Context, titleSlug string) ([]domain.FriendXPHistory, error)

// CareerService construit les réponses pour la page Carrière.
type CareerService struct {
	repo            port.CareerRepository
	metaRepo        port.MetadataRepository // optionnel — nil = fallback synthétique
	titleSlug       string                  // titre courant, ex: "halo_infinite"
	friendsXPLoader FriendsXPLoader         // optionnel — nil = pas de courbes amis
	// dataAdapter (optionnel) — Phase C+ multi-titres. Quand fourni, GetEncounters
	// passe par games.TitleDataAdapter.LoadEncounters au lieu d'appeler le repo
	// directement. Préserve une parité comportementale stricte : projection
	// canonical.EncounterRow → domain.EncounterDTO identique à la version repo.
	// Activé via WithDataAdapter.
	dataAdapter   games.TitleDataAdapter
	rankCatalog   *mappings.RankCatalog // optionnel — nil = pas de nom prochain rang
	rankImageURLs map[int]*string       // optionnel — nil = pas d'images de rang
	// friendGamertags : resolver des gamertags amis (cf. settings.FriendGamertags).
	// Utilisé par GetTopEncounters pour exclure les amis du tableau "joueurs les
	// plus croisés (hors amis)". Si nil, aucune exclusion (équivalent à 0 ami).
	friendGamertags teammates.FriendGamertagsResolver
	// friendXUIDResolver : optionnel — résout un gamertag en XUID via xuid_aliases.
	// Si nil, GetTopEncounters dégrade gracieusement (pas d'exclusion d'amis).
	friendXUIDResolver func(ctx context.Context, gamertag string) (string, error)
	// seasonsCatalog : optionnel — résolveur saisons (TOML + DB + lazy fetch).
	// Utilisé par GetHighlightMatchIDs pour traduire les SeasonIDs sélectionnés
	// en fenêtres temporelles SQL et pour calculer les cascade counts. Quand
	// nil, le filtre saisons est inopérant et available_seasons reste vide.
	seasonsCatalog seasonsCatalogLoader
	// csrSeasonID : identifiant de la saison CSR courante (ex. "CsrSeason8").
	// Utilisé comme metadata dans GetCareerCSRs. Vide = non configuré.
	csrSeasonID string
}

// NewCareerService crée un CareerService.
func NewCareerService(repo port.CareerRepository) *CareerService {
	return &CareerService{repo: repo}
}

// WithMetadataRepo injecte le repository de métadonnées (saisons, etc.).
func (s *CareerService) WithMetadataRepo(r port.MetadataRepository) *CareerService {
	s.metaRepo = r
	return s
}

// WithTitleSlug configure le slug du titre (ex: "halo_infinite").
func (s *CareerService) WithTitleSlug(slug string) *CareerService {
	s.titleSlug = slug
	return s
}

// WithDataAdapter injecte un games.TitleDataAdapter optionnel pour faire
// passer GetEncounters par la couche multi-titres (Phase C+).
//
// Si nil ou si LoadEncounters retourne ErrCapabilityNotSupported, le service
// retombe sur l'appel direct repo.GetEncounters. Cette dégradation gracieuse
// permet d'activer/désactiver la bascule sans casser le service.
func (s *CareerService) WithDataAdapter(a games.TitleDataAdapter) *CareerService {
	s.dataAdapter = a
	return s
}

// WithFriendsXPLoader injecte un loader d'historique XP des amis.
// Quand nil, aucune courbe ami n'est incluse dans GetCareerPage.
func (s *CareerService) WithFriendsXPLoader(loader FriendsXPLoader) *CareerService {
	s.friendsXPLoader = loader
	return s
}

// WithRankCatalog injecte le catalog des rangs (noms localisés).
// Quand nil, next_rank_name_fr/en restent vides.
func (s *CareerService) WithRankCatalog(c *mappings.RankCatalog) *CareerService {
	s.rankCatalog = c
	return s
}

// WithRankImageURLs injecte la map rank_id → imageURL (chargée au démarrage).
// Quand nil, rank_image_url et next_rank_image_url restent absents.
func (s *CareerService) WithRankImageURLs(imgs map[int]*string) *CareerService {
	s.rankImageURLs = imgs
	return s
}

// WithFriendGamertagsResolver injecte le resolver d'amis configurés (lit
// app_settings.friend_gamertags). Quand nil, GetTopEncounters n'exclut aucun
// joueur (le tableau "hors amis" affichera tous les plus croisés).
func (s *CareerService) WithFriendGamertagsResolver(r teammates.FriendGamertagsResolver) *CareerService {
	s.friendGamertags = r
	return s
}

// WithFriendXUIDResolver injecte un résolveur gamertag → XUID (typiquement
// délégué à ExplorerRepo.ResolveXUIDByGamertag). Requis pour exclure les amis
// dans GetTopEncounters (la query travaille en XUIDs).
func (s *CareerService) WithFriendXUIDResolver(fn func(ctx context.Context, gamertag string) (string, error)) *CareerService {
	s.friendXUIDResolver = fn
	return s
}

// WithSeasonsCatalog injecte le résolveur unifié des saisons (mêmes pattern
// que FiltersService). Sert au filtre Saisons + cascade counts dans la
// section "Matchs marquants". Quand nil, le filtre saisons est inopérant.
func (s *CareerService) WithSeasonsCatalog(catalog *SeasonsCatalog) *CareerService {
	if catalog != nil { // garde concret fiable — évite le piège interface typed-nil
		s.seasonsCatalog = catalog
	}
	return s
}

// WithCSRSeasonID injecte l'identifiant de la saison CSR courante (ex. "CsrSeason8").
func (s *CareerService) WithCSRSeasonID(id string) *CareerService {
	s.csrSeasonID = id
	return s
}

// GetCareerCSRs retourne les classements CSR du joueur par playlist pour la
// saison demandée (seasonID vide → saison courante). Inclut la liste des saisons
// proposables (menu déroulant). Réponse vide (pas d'erreur) si aucun snapshot.
func (s *CareerService) GetCareerCSRs(ctx context.Context, seasonID string) (domain.CareerCSRResponse, error) {
	playlists, err := s.repo.GetCSRSnapshots(ctx, seasonID)
	if err != nil {
		return domain.CareerCSRResponse{}, fmt.Errorf("GetCareerCSRs: %w", err)
	}
	if playlists == nil {
		playlists = []domain.CareerPlaylistCSR{}
	}
	seasons, err := s.repo.AvailableCSRSeasons(ctx)
	if err != nil {
		return domain.CareerCSRResponse{}, fmt.Errorf("GetCareerCSRs seasons: %w", err)
	}
	if seasons == nil {
		seasons = []domain.CSRSeasonOption{}
	}
	// SeasonID effectif = demandé si fourni, sinon la saison courante configurée.
	effective := strings.TrimSpace(seasonID)
	if effective == "" {
		effective = s.csrSeasonID
	}
	return domain.CareerCSRResponse{
		Playlists:        playlists,
		SeasonID:         effective,
		AvailableSeasons: seasons,
	}, nil
}

// GetCareerPage retourne la réponse complète de la page Carrière.
//
// Phase C+ multi-titres : GetLatestRank passe par DataAdapter.LoadCareerSnapshot
// quand le service a un dataAdapter ; sinon fallback repo direct. Parité de
// payload garantie par projectionLatestRankFromCanonical.
func (s *CareerService) GetCareerPage(ctx context.Context) (domain.CareerPageResponse, error) {
	// P8.3 (revue 2026-04-29) : observabilité service_duration_ms.
	defer func(start time.Time) {
		observability.RecordDurationMS("career_get_page", time.Since(start).Milliseconds())
	}(time.Now())
	rank, err := s.loadLatestRank(ctx)
	if err != nil {
		return domain.CareerPageResponse{}, fmt.Errorf("CareerService.GetCareerPage: %w", err)
	}
	xpHistory, err := s.loadXPHistory(ctx)
	if err != nil {
		return domain.CareerPageResponse{}, fmt.Errorf("CareerService.GetCareerPage: %w", err)
	}
	lusrHistory, err := s.loadLUSRHistory(ctx)
	if err != nil {
		return domain.CareerPageResponse{}, fmt.Errorf("CareerService.GetCareerPage: %w", err)
	}

	summary := s.buildCareerSummaryEnriched(ctx, rank)
	xpTotal := summaryXPTotal(rank)
	xpHeroMax := heroXPTotal(rank)
	hero := buildHeroProgress(xpTotal, rankIDFromData(rank), xpHeroMax, heroRankMax(rank))
	projs := buildProjections(xpHistory, xpTotal, xpHeroMax)
	lusr := buildLUSRSummary(lusrHistory)

	currentSeason := s.resolveCurrentSeason(ctx)

	// Init [] plutôt que nil : un slice nil sérialise en JSON `null` et crashe le
	// front. Cf. testutil.RequireNoNilSlicesWithoutOmitempty.
	if xpHistory == nil {
		xpHistory = []domain.XPHistoryPoint{}
	}
	resp := domain.CareerPageResponse{
		Summary:       summary,
		HeroProgress:  hero,
		Projections:   projs,
		XPHistory:     xpHistory,
		LUSR:          lusr,
		CurrentSeason: currentSeason,
	}

	if s.friendsXPLoader != nil {
		friends, err := s.friendsXPLoader(ctx, s.titleSlug)
		if err != nil {
			slog.WarnContext(ctx, "career_friends_xp_load_failed", "err", err)
		} else if len(friends) > 0 {
			resp.FriendsXPHistory = friends
		}
	}

	return resp, nil
}

// loadLatestRank centralise la résolution repo/adapter pour GetLatestRank.
// Si dataAdapter est fourni et supporte career.progression, passe par
// LoadCareerSnapshot et reconstitue un *domain.CareerRankData strictement
// identique à celui que repo.GetLatestRank aurait retourné.
func (s *CareerService) loadLatestRank(ctx context.Context) (*domain.CareerRankData, error) {
	if s.dataAdapter != nil {
		snap, err := s.dataAdapter.LoadCareerSnapshot(ctx, "", canonical.CareerOptions{})
		if err == nil {
			return rankDataFromCanonical(snap), nil
		}
		if !errors.Is(err, games.ErrCapabilityNotSupported) {
			return nil, err
		}
	}
	return s.repo.GetLatestRank(ctx)
}

// rankDataFromCanonical projette canonical.CareerSnapshot → *domain.CareerRankData.
// Préserve la forme exacte attendue par buildCareerSummary / summaryXPTotal.
func rankDataFromCanonical(snap *canonical.CareerSnapshot) *domain.CareerRankData {
	if snap == nil {
		return nil
	}
	row := &domain.CareerRankData{
		RankNumber: snap.RankNumber,
		IsMaxRank:  snap.IsMaxRank,
	}
	if snap.RecordedAt != nil {
		row.RecordedAt = *snap.RecordedAt
	}
	if snap.CurrentXP != nil {
		row.CurrentXP = *snap.CurrentXP
	}
	if snap.XPForNextRank != nil {
		v := *snap.XPForNextRank
		row.XPForNextRank = &v
	}
	if snap.XPTotal != nil {
		v := *snap.XPTotal
		row.XPTotal = &v
	}
	if snap.RankTier != nil {
		v := *snap.RankTier
		row.RankTier = &v
	}
	if snap.RankName != nil {
		v := *snap.RankName
		row.RankName = &v
	}
	if snap.CurrentRank != nil && snap.CurrentRank.ID != "" {
		v := snap.CurrentRank.ID
		row.RankLabel = &v
	}
	if snap.XPMax != nil {
		v := *snap.XPMax
		row.XPHeroTotal = &v
	}
	if snap.RankMax != nil {
		v := *snap.RankMax
		row.RankMax = &v
	}
	return row
}

// loadXPHistory centralise la résolution repo/adapter pour l'historique XP (HIGH-C).
// Si dataAdapter est fourni et supporte career.progression, passe par
// LoadCareerSnapshot(IncludeHistory) et reconstitue []domain.XPHistoryPoint
// strictement identique à repo.GetXPHistory. Fallback repo sur capability absente.
//
// Note : GetCareerPage appelle loadLatestRank AVANT loadXPHistory ; une erreur
// GetLatestRank avorte donc la page en amont — le double appel via LoadCareerSnapshot
// est sans effet sur le payload (mêmes données, mêmes erreurs gérées en amont).
func (s *CareerService) loadXPHistory(ctx context.Context) ([]domain.XPHistoryPoint, error) {
	if s.dataAdapter != nil {
		snap, err := s.dataAdapter.LoadCareerSnapshot(ctx, "", canonical.CareerOptions{IncludeHistory: true})
		if err == nil {
			return xpHistoryFromCanonical(snap.History), nil
		}
		if !errors.Is(err, games.ErrCapabilityNotSupported) {
			return nil, err
		}
	}
	return s.repo.GetXPHistory(ctx)
}

// xpHistoryFromCanonical projette canonical.CareerSnapshot.History →
// []domain.XPHistoryPoint. Retourne nil pour une entrée vide (préserve le
// short-circuit projections len<2 + la ré-init nil→[] de GetCareerPage).
func xpHistoryFromCanonical(entries []canonical.CareerHistoryEntry) []domain.XPHistoryPoint {
	if len(entries) == 0 {
		return nil
	}
	out := make([]domain.XPHistoryPoint, 0, len(entries))
	for _, e := range entries {
		p := domain.XPHistoryPoint{RecordedAt: e.RecordedAt, Rank: e.RankNumber}
		if e.CurrentXP != nil {
			p.CurrentXP = *e.CurrentXP
		}
		if e.XPTotal != nil {
			p.XPTotal = *e.XPTotal
		}
		out = append(out, p)
	}
	return out
}

// loadLUSRHistory centralise la résolution repo/adapter pour l'historique LUSR
// (HIGH-C). Adapter-first via LoadLUSRHistory + reconstitution byte-identique ;
// fallback repo.GetLUSRHistory sur capability absente.
func (s *CareerService) loadLUSRHistory(ctx context.Context) ([]domain.LUSRCheckpointDTO, error) {
	if s.dataAdapter != nil {
		checkpoints, err := s.dataAdapter.LoadLUSRHistory(ctx, "")
		if err == nil {
			return lusrCheckpointsFromCanonical(checkpoints), nil
		}
		if !errors.Is(err, games.ErrCapabilityNotSupported) {
			return nil, err
		}
	}
	return s.repo.GetLUSRHistory(ctx)
}

// lusrCheckpointsFromCanonical projette []canonical.LUSRCheckpoint →
// []domain.LUSRCheckpointDTO (copie profonde des pointeurs). Retourne nil pour
// une entrée vide (buildLUSRSummary émet alors Checkpoints: [] comme le legacy).
func lusrCheckpointsFromCanonical(cs []canonical.LUSRCheckpoint) []domain.LUSRCheckpointDTO {
	if len(cs) == 0 {
		return nil
	}
	out := make([]domain.LUSRCheckpointDTO, 0, len(cs))
	for _, c := range cs {
		d := domain.LUSRCheckpointDTO{
			MatchID:      c.MatchID,
			RatingType:   c.RatingType,
			RatingValue:  c.RatingValue,
			PlaylistName: c.PlaylistName,
			PlaylistID:   c.PlaylistID,
		}
		if c.TierLabel != nil {
			v := *c.TierLabel
			d.TierLabel = &v
		}
		if c.PlaylistGroup != nil {
			v := *c.PlaylistGroup
			d.PlaylistGroup = &v
		}
		if c.RecordedAt != nil {
			v := *c.RecordedAt
			d.RecordedAt = &v
		}
		if c.RatingDelta != nil {
			v := *c.RatingDelta
			d.RatingDelta = &v
		}
		if c.BadgeImageURL != nil {
			v := *c.BadgeImageURL
			d.BadgeImageURL = &v
		}
		out = append(out, d)
	}
	return out
}

// ── Sprint 54-A7/A8 : résolution saison courante avec fallback synthétique ──

// resolveCurrentSeason retourne la saison courante depuis le MetadataRepo.
// Si le repo est absent ou ne contient aucune saison, retourne un CurrentSeasonResult
// avec Synthetic non-nil (fallback S54-A8).
func (s *CareerService) resolveCurrentSeason(ctx context.Context) *domain.CurrentSeasonResult {
	if s.metaRepo == nil {
		return syntheticSeasonResult()
	}
	titleID := s.titleSlug
	if titleID == "" {
		titleID = title.DefaultSlug
	}
	slog.DebugContext(ctx, "resolveCurrentSeason", "titleSlug", titleID)
	season, err := s.metaRepo.GetCurrentSeason(ctx, titleID)
	if err != nil || season == nil {
		return syntheticSeasonResult()
	}
	return &domain.CurrentSeasonResult{Season: season}
}

// syntheticSeasonResult retourne un fallback synthétique (0 date hardcodée).
func syntheticSeasonResult() *domain.CurrentSeasonResult {
	return &domain.CurrentSeasonResult{
		Synthetic: &domain.SeasonSynthetic{
			SeasonID:   "unknown",
			Name:       "Saison inconnue",
			StartDate:  time.Time{},
			IsFallback: true,
		},
	}
}
