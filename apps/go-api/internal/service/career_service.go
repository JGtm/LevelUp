// Package service — CareerService : page carrière, top matches, encounters.
//
// Port Go de career_service.py (apps/api/app/services/career_service.py)
// et career_logic.py (src/ui/pages/career_logic.py).
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"
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
	repo             port.CareerRepository
	metaRepo         port.MetadataRepository // optionnel — nil = fallback synthétique
	titleSlug        string                  // titre courant, ex: "halo_infinite"
	friendsXPLoader  FriendsXPLoader         // optionnel — nil = pas de courbes amis
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
	friendGamertags FriendGamertagsResolver
	// friendXUIDResolver : optionnel — résout un gamertag en XUID via xuid_aliases.
	// Si nil, GetTopEncounters dégrade gracieusement (pas d'exclusion d'amis).
	friendXUIDResolver func(ctx context.Context, gamertag string) (string, error)
	// seasonsCatalog : optionnel — résolveur saisons (TOML + DB + lazy fetch).
	// Utilisé par GetHighlightMatchIDs pour traduire les SeasonIDs sélectionnés
	// en fenêtres temporelles SQL et pour calculer les cascade counts. Quand
	// nil, le filtre saisons est inopérant et available_seasons reste vide.
	seasonsCatalog *SeasonsCatalog
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
func (s *CareerService) WithFriendGamertagsResolver(r FriendGamertagsResolver) *CareerService {
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
	s.seasonsCatalog = catalog
	return s
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
	xpHistory, err := s.repo.GetXPHistory(ctx)
	if err != nil {
		return domain.CareerPageResponse{}, fmt.Errorf("CareerService.GetCareerPage: %w", err)
	}
	lusrHistory, err := s.repo.GetLUSRHistory(ctx)
	if err != nil {
		return domain.CareerPageResponse{}, fmt.Errorf("CareerService.GetCareerPage: %w", err)
	}

	summary := s.buildCareerSummaryEnriched(rank)
	xpTotal := summaryXPTotal(rank)
	hero := buildHeroProgress(xpTotal, rankIDFromData(rank))
	projs := buildProjections(xpHistory, xpTotal)
	lusr := buildLUSRSummary(lusrHistory)

	currentSeason := s.resolveCurrentSeason(ctx)

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

// GetTopMatches retourne les 10 meilleurs et 10 moins bons matchs.
func (s *CareerService) GetTopMatches(ctx context.Context) (domain.CareerTopMatchesResponse, error) {
	rows, err := s.repo.GetTopMatches(ctx)
	if err != nil {
		return domain.CareerTopMatchesResponse{}, fmt.Errorf("CareerService.GetTopMatches: %w", err)
	}

	// Q9 retourne jusqu'à 20 matchs classés DESC par performance_score.
	// Les 10 premiers = meilleurs, les 10 derniers = moins bons.
	bestRows, worstRows := splitTopRows(rows)
	best := convertTopMatches(bestRows)
	// Inverser l'affichage des pires (les moins bons en premier)
	reverseTopMatches(worstRows)
	worst := convertTopMatches(worstRows)

	return domain.CareerTopMatchesResponse{
		BestMatches:  best,
		WorstMatches: worst,
	}, nil
}

// GetEncounters retourne les joueurs les plus fréquemment croisés.
//
// Phase C+ multi-titres : si un dataAdapter est injecté et que sa capability
// career.progression est supportée, la lecture passe par LoadEncounters avec
// projection canonical.EncounterRow → domain.EncounterDTO. Sinon fallback
// gracieux sur s.repo.GetEncounters (parité comportementale par construction).
func (s *CareerService) GetEncounters(ctx context.Context) (domain.CareerEncountersResponse, error) {
	rows, err := s.loadEncounterRows(ctx)
	if err != nil {
		return domain.CareerEncountersResponse{}, fmt.Errorf("CareerService.GetEncounters: %w", err)
	}

	var teammates, enemies []domain.EncounterDTO
	for _, r := range rows {
		if r.AsTeammate >= r.AsEnemy {
			teammates = append(teammates, r)
		} else {
			enemies = append(enemies, r)
		}
	}

	return domain.CareerEncountersResponse{
		Teammates: teammates,
		Enemies:   enemies,
		Total:     len(rows),
	}, nil
}

// GetHighlightMatchIDs retourne les match_ids triés (best d'abord, worst
// ensuite) des matchs marquants — 15 + 15 — avec les cascade counts pour
// les dropdowns Expérience / Saisons. Le handler enrichit ensuite les IDs
// via MatchHistoryService pour produire des ExplorerMatchesRow complets.
func (s *CareerService) GetHighlightMatchIDs(ctx context.Context, input domain.HighlightFilterInput) (domain.HighlightMatchesData, error) {
	// Résout les SeasonIDs sélectionnés en fenêtres temporelles via le catalog.
	catalog := s.loadSeasonCatalog(ctx)
	selectedRanges, _ := resolveSeasonRanges(catalog, input.SeasonIDs)

	filters := domain.CareerHighlightFilters{
		Experience:   normalizeExperience(input.Experience),
		SeasonRanges: selectedRanges,
	}

	rows, err := s.repo.GetHighlightMatchIDs(ctx, filters)
	if err != nil {
		return domain.HighlightMatchesData{}, fmt.Errorf("CareerService.GetHighlightMatchIDs: %w", err)
	}

	// Pool complet pour cascade counts (silently dégradé si erreur).
	pool, perr := s.repo.GetHighlightPool(ctx)
	if perr != nil {
		slog.WarnContext(ctx, "career.highlight.pool_load_failed", "err", perr)
	}

	return domain.HighlightMatchesData{
		Rows:                rows,
		AvailableExperience: computeHighlightAvailableExperience(pool, selectedRanges),
		AvailableSeasons:    computeHighlightAvailableSeasons(pool, filters.Experience, catalog),
	}, nil
}

// loadSeasonCatalog charge le catalog via SeasonsCatalog injecté. Retourne nil
// si non câblé (dégradation gracieuse — pas de cascade saisons).
func (s *CareerService) loadSeasonCatalog(ctx context.Context) []SeasonCatalogEntry {
	if s.seasonsCatalog == nil || s.titleSlug == "" {
		return nil
	}
	return s.seasonsCatalog.Load(ctx, s.titleSlug)
}

// normalizeExperience clamp la valeur d'entrée sur les 3 valeurs autorisées.
// Toute autre valeur (vide, "tous", etc.) → "all" (= pas de filtre).
func normalizeExperience(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "ranked":
		return "ranked"
	case "unranked":
		return "unranked"
	default:
		return "all"
	}
}

// resolveSeasonRanges projette les seasonIDs sélectionnés en SeasonTimeRange
// via le catalog. Retourne aussi la liste des IDs qui ont matché (pour debug).
// IDs inconnus du catalog silencieusement ignorés.
func resolveSeasonRanges(catalog []SeasonCatalogEntry, seasonIDs []string) ([]domain.SeasonTimeRange, []string) {
	if len(seasonIDs) == 0 || len(catalog) == 0 {
		return nil, nil
	}
	wanted := make(map[string]struct{}, len(seasonIDs))
	for _, id := range seasonIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			wanted[id] = struct{}{}
		}
	}
	ranges := make([]domain.SeasonTimeRange, 0, len(wanted))
	matched := make([]string, 0, len(wanted))
	for _, e := range catalog {
		if _, ok := wanted[e.ID]; !ok {
			continue
		}
		ranges = append(ranges, domain.SeasonTimeRange{Start: e.Start, End: e.End})
		matched = append(matched, e.ID)
	}
	return ranges, matched
}

// computeHighlightAvailableExperience calcule les counts cascade-aware pour
// la dropdown Expérience : on respecte le filtre Saisons mais on ignore le
// filtre Expérience courant (sinon on aurait juste le count de l'option active).
func computeHighlightAvailableExperience(pool []domain.HighlightMatchPoolRow, seasonRanges []domain.SeasonTimeRange) []domain.HighlightExperienceCount {
	counts := struct {
		all, ranked, unranked int
	}{}
	for _, m := range pool {
		if !matchInSeasonRanges(m.StartTime, seasonRanges) {
			continue
		}
		counts.all++
		if m.IsRanked {
			counts.ranked++
		} else {
			counts.unranked++
		}
	}
	return []domain.HighlightExperienceCount{
		{Value: "all", Count: counts.all},
		{Value: "ranked", Count: counts.ranked},
		{Value: "unranked", Count: counts.unranked},
	}
}

// computeHighlightAvailableSeasons calcule les counts par saison du catalog
// en respectant le filtre Expérience courant mais pas le filtre Saisons
// (la dropdown affiche le count par saison si on coche cette saison).
func computeHighlightAvailableSeasons(pool []domain.HighlightMatchPoolRow, experience string, catalog []SeasonCatalogEntry) []domain.HighlightSeasonCount {
	if len(catalog) == 0 {
		return nil
	}
	out := make([]domain.HighlightSeasonCount, 0, len(catalog))
	for _, season := range catalog {
		count := 0
		for _, m := range pool {
			if !matchPassesExperience(m.IsRanked, experience) {
				continue
			}
			if m.StartTime == nil {
				continue
			}
			if m.StartTime.Before(season.Start) {
				continue
			}
			if season.End != nil && !m.StartTime.Before(*season.End) {
				continue
			}
			count++
		}
		out = append(out, domain.HighlightSeasonCount{Value: season.ID, Count: count})
	}
	return out
}

// matchPassesExperience : true si le match passe le filtre Expérience.
func matchPassesExperience(isRanked bool, experience string) bool {
	switch experience {
	case "ranked":
		return isRanked
	case "unranked":
		return !isRanked
	default: // "all" ou inconnu
		return true
	}
}

// matchInSeasonRanges : true si startTime tombe dans au moins une fenêtre
// de seasonRanges. Si seasonRanges est vide → true (pas de filtre).
func matchInSeasonRanges(startTime *time.Time, seasonRanges []domain.SeasonTimeRange) bool {
	if len(seasonRanges) == 0 {
		return true
	}
	if startTime == nil {
		return false
	}
	for _, w := range seasonRanges {
		if startTime.Before(w.Start) {
			continue
		}
		if w.End != nil && !startTime.Before(*w.End) {
			continue
		}
		return true
	}
	return false
}

// GetTopEncounters retourne les 10 joueurs les plus croisés au niveau carrière
// globale, hors amis configurés (FriendGamertags). Enrichit chaque encounter
// avec les badges narratifs (ally_plus / tough_enemy / ordinal) via le même
// algorithme que MatchView.
func (s *CareerService) GetTopEncounters(ctx context.Context) (domain.CareerTopEncountersResponse, error) {
	excludeXUIDs := s.resolveFriendXUIDs(ctx)
	encounters, stats, err := s.repo.GetTopEncountersGlobal(ctx, excludeXUIDs)
	if err != nil {
		return domain.CareerTopEncountersResponse{}, fmt.Errorf("CareerService.GetTopEncounters: %w", err)
	}
	// Index stats par xuid pour O(1) lookup lors de l'application des badges.
	statsByXUID := make(map[string]domain.EncounterStatsRaw, len(stats))
	for _, st := range stats {
		statsByXUID[st.XUID] = st
	}
	out := make([]domain.MatchEncounterRow, 0, len(encounters))
	for _, e := range encounters {
		st, ok := statsByXUID[e.XUID]
		if !ok {
			out = append(out, e)
			continue
		}
		e.Badges = computeCareerEncounterBadges(e, st)
		out = append(out, e)
	}
	return domain.CareerTopEncountersResponse{Items: out}, nil
}

// GetRivals retourne le top 10 des némésis (deaths DESC) et top 10 des
// souffre-douleur (frags DESC). Le ratio est calculé côté service (frags/deaths
// avec garde div-par-zéro : 0 morts → ratio = float64(Frags)).
func (s *CareerService) GetRivals(ctx context.Context) (domain.CareerRivalsResponse, error) {
	nemesesRaw, victimsRaw, err := s.repo.GetRivals(ctx)
	if err != nil {
		return domain.CareerRivalsResponse{}, fmt.Errorf("CareerService.GetRivals: %w", err)
	}
	return domain.CareerRivalsResponse{
		Nemeses: convertRivals(nemesesRaw),
		Victims: convertRivals(victimsRaw),
	}, nil
}

// resolveFriendXUIDs résout la liste des amis configurés (gamertags) en XUIDs.
// Dégrade gracieusement : skip silencieux pour chaque gamertag non résolvable.
// En cas d'amis non résolus, log Warn pour signaler une dérive de config (un
// gamertag dans settings n'existe ni dans xuid_aliases ni dans match_participants).
func (s *CareerService) resolveFriendXUIDs(ctx context.Context) []string {
	if s.friendGamertags == nil || s.friendXUIDResolver == nil {
		return nil
	}
	gts := s.friendGamertags(ctx)
	if len(gts) == 0 {
		return nil
	}
	out := make([]string, 0, len(gts))
	var unresolved []string
	for _, gt := range gts {
		gt = strings.TrimSpace(gt)
		if gt == "" {
			continue
		}
		xuid, err := s.friendXUIDResolver(ctx, gt)
		if err != nil || xuid == "" {
			unresolved = append(unresolved, gt)
			continue
		}
		out = append(out, xuid)
	}
	if len(unresolved) > 0 {
		slog.WarnContext(ctx, "career.top_encounters.friends_unresolved",
			"unresolved", unresolved,
			"resolved", len(out),
		)
	}
	return out
}

// computeCareerEncounterBadges applique narrative.ComputeEncounterBadges
// (ordinal + ally_plus + tough_enemy) avec le même protocole que MatchView.
func computeCareerEncounterBadges(e domain.MatchEncounterRow, st domain.EncounterStatsRaw) []domain.MatchEncounterBadge {
	winrateAsAlly := encounterBadgeWinrate(st.WinsAsAlly, st.LossesAsAlly)
	winrateVsEnemy := encounterBadgeWinrate(st.WinsVsEnemy, st.LossesVsEnemy)
	stats := narrative.EncounterStats{
		XUID:            e.XUID,
		Gamertag:        e.Gamertag,
		TotalEncounters: e.CountTogether,
		AllyCount:       st.AllyCount,
		EnemyCount:      st.EnemyCount,
		WinrateAsAlly:   winrateAsAlly,
		WinrateVsEnemy:  winrateVsEnemy,
		KillsDealt:      st.KillsDealt,
		DeathsSuffered:  st.DeathsSuffered,
	}
	ordinal := e.CountTogether - 1
	if ordinal < 0 {
		ordinal = 0
	}
	raw := narrative.ComputeEncounterBadges(stats, ordinal)
	if len(raw) == 0 {
		return nil
	}
	out := make([]domain.MatchEncounterBadge, 0, len(raw))
	for _, b := range raw {
		out = append(out, domain.MatchEncounterBadge{
			Kind:       string(b.Kind),
			LabelKey:   b.LabelKey,
			ColorToken: b.ColorToken,
			Detail:     b.Detail,
		})
	}
	return out
}

// encounterBadgeWinrate : retourne nil si W+L == 0, sinon le ratio (0..1).
// Mirror de service.encounterWinrate (match_view_service.go) — duplication
// volontaire pour éviter le couplage entre les deux services.
func encounterBadgeWinrate(wins, losses int) *float64 {
	total := wins + losses
	if total == 0 {
		return nil
	}
	rate := analysis.WinRate(wins, total)
	return &rate
}

// convertRivals projette CareerRivalRawRow → CareerRival (calcule le ratio).
func convertRivals(raw []domain.CareerRivalRawRow) []domain.CareerRival {
	out := make([]domain.CareerRival, 0, len(raw))
	for _, r := range raw {
		var ratio float64
		if r.Deaths > 0 {
			ratio = float64(r.Frags) / float64(r.Deaths)
		} else {
			ratio = float64(r.Frags) // 0 morts → ratio = nb de frags (semantically "infini" approximé)
		}
		out = append(out, domain.CareerRival{
			Gamertag:   r.Gamertag,
			Frags:      r.Frags,
			Deaths:     r.Deaths,
			Ratio:      ratio,
			MatchCount: r.MatchCount,
		})
	}
	return out
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
	return row
}

// loadEncounterRows centralise la résolution repo/adapter et garantit la
// même forme de sortie []domain.EncounterDTO quel que soit le chemin.
func (s *CareerService) loadEncounterRows(ctx context.Context) ([]domain.EncounterDTO, error) {
	if s.dataAdapter != nil {
		canonicalRows, err := s.dataAdapter.LoadEncounters(ctx, "")
		if err == nil {
			out := make([]domain.EncounterDTO, 0, len(canonicalRows))
			for _, r := range canonicalRows {
				out = append(out, encounterDTOFromCanonical(r))
			}
			return out, nil
		}
		if !errors.Is(err, games.ErrCapabilityNotSupported) {
			return nil, err
		}
		// capability not supported → fallback repo
	}

	rows, err := s.repo.GetEncounters(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.EncounterDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.EncounterDTO{
			Gamertag:   r.Gamertag,
			XUID:       r.XUID,
			MatchCount: r.MatchCount,
			AsTeammate: r.AsTeammate,
			AsEnemy:    r.AsEnemy,
			AvgKDA:     r.AvgKDA,
		})
	}
	return out, nil
}

// encounterDTOFromCanonical projette canonical.EncounterRow → domain.EncounterDTO
// avec strictement la même forme JSON que la projection legacy depuis
// domain.EncounterRawRow. Garantit la parité de payload pour la golden parity.
func encounterDTOFromCanonical(r canonical.EncounterRow) domain.EncounterDTO {
	return domain.EncounterDTO{
		Gamertag:   r.Identity.Gamertag,
		XUID:       r.Identity.XUID,
		MatchCount: r.MatchCount,
		AsTeammate: r.AsTeammate,
		AsEnemy:    r.AsEnemy,
		AvgKDA:     r.AvgKDA,
	}
}

// ---------------------------------------------------------------------------
// Builders internes
// ---------------------------------------------------------------------------

func buildCareerSummary(rank *domain.CareerRankData) domain.CareerRankSummary {
	if rank == nil {
		return domain.CareerRankSummary{}
	}
	label := formatRankLabel(rank)
	xpForNext := 0
	if rank.XPForNextRank != nil {
		xpForNext = *rank.XPForNextRank
	}
	xpTotal := 0
	if rank.XPTotal != nil {
		xpTotal = *rank.XPTotal
	}
	tier := ""
	if rank.RankTier != nil {
		tier = *rank.RankTier
	}
	nameRaw := ""
	if rank.RankName != nil {
		nameRaw = *rank.RankName
	}
	pct := computeProgressPct(rank.CurrentXP, xpForNext, rank.IsMaxRank)
	rec := rank.RecordedAt
	return domain.CareerRankSummary{
		RankNumber:    rank.RankNumber,
		RankLabel:     label,
		RankNameRaw:   nameRaw,
		RankTier:      tier,
		CurrentXP:     rank.CurrentXP,
		XPForNextRank: xpForNext,
		XPTotal:       xpTotal,
		ProgressPct:   pct,
		IsMaxRank:     rank.IsMaxRank,
		RecordedAt:    &rec,
	}
}

// buildCareerSummaryEnriched enrichit le résumé avec les images et noms de rang
// depuis rankCatalog et rankImageURLs injectés dans le service.
func (s *CareerService) buildCareerSummaryEnriched(rank *domain.CareerRankData) domain.CareerRankSummary {
	summary := buildCareerSummary(rank)
	if rank == nil {
		return summary
	}
	if img, ok := s.rankImageURLs[rank.RankNumber]; ok {
		summary.RankImageURL = img
	}
	if !rank.IsMaxRank && s.rankCatalog != nil {
		if next, ok := s.rankCatalog.Next(rank.RankNumber); ok {
			fr, _ := next.FullLabel("fr")
			en, _ := next.FullLabel("en")
			summary.NextRankNameFR = strings.TrimSpace(fr)
			summary.NextRankNameEN = strings.TrimSpace(en)
		}
		if img, ok := s.rankImageURLs[rank.RankNumber+1]; ok {
			summary.NextRankImageURL = img
		}
	}
	return summary
}

// rankIDFromData retourne RankNumber depuis CareerRankData, ou 0 si nil.
func rankIDFromData(rank *domain.CareerRankData) int {
	if rank == nil {
		return 0
	}
	return rank.RankNumber
}

func formatRankLabel(rank *domain.CareerRankData) string {
	if rank.RankLabel != nil && *rank.RankLabel != "" {
		return *rank.RankLabel
	}
	var parts []string
	if rank.RankName != nil && *rank.RankName != "" {
		parts = append(parts, *rank.RankName)
	}
	if rank.RankTier != nil && *rank.RankTier != "" {
		parts = append(parts, *rank.RankTier)
	}
	if len(parts) > 0 {
		result := parts[0]
		for _, p := range parts[1:] {
			result += " - " + p
		}
		return result
	}
	return fmt.Sprintf("Rang %d", rank.RankNumber)
}

func computeProgressPct(currentXP, xpForNext int, isMaxRank bool) float64 {
	if isMaxRank {
		return 100.0
	}
	if xpForNext <= 0 {
		return 0.0
	}
	pct := float64(currentXP) / float64(xpForNext) * 100.0
	if pct > 100.0 {
		pct = 100.0
	}
	return math.Round(pct*100) / 100
}

func summaryXPTotal(rank *domain.CareerRankData) int {
	if rank == nil || rank.XPTotal == nil {
		return 0
	}
	return *rank.XPTotal
}

func buildHeroProgress(xpTotal, currentRank int) domain.HeroProgress {
	remaining := xpHeroTotal - xpTotal
	if remaining < 0 {
		remaining = 0
	}
	pct := float64(xpTotal) / float64(xpHeroTotal) * 100.0
	pct = math.Round(pct*100) / 100
	if pct > 100.0 {
		pct = 100.0
	}
	return domain.HeroProgress{
		XPTotalRequired: xpHeroTotal,
		XPRemaining:     remaining,
		Percentage:      pct,
		CurrentRank:     currentRank,
		TotalRanks:      rankMax,
	}
}

func buildProjections(history []domain.XPHistoryPoint, xpTotal int) domain.CareerProjections {
	if len(history) < 2 {
		return domain.CareerProjections{}
	}
	xpPerActive := computeActiveXPPerDay(history)
	firstDate := history[0].RecordedAt
	xpPerFallback := computeFallbackXPPerDay(xpTotal, firstDate)

	var heroDateStr *string
	if xpTotal < xpHeroTotal && xpPerActive > 0 {
		lastDate := history[len(history)-1].RecordedAt
		daysNeeded := float64(xpHeroTotal-xpTotal) / xpPerActive
		heroTime := lastDate.Add(time.Duration(daysNeeded * float64(24*time.Hour)))
		s := heroTime.Format("2006-01-02")
		heroDateStr = &s
	}

	return domain.CareerProjections{
		XPPerDayActive:    math.Round(xpPerActive*100) / 100,
		XPPerDayFallback:  math.Round(xpPerFallback*100) / 100,
		EstimatedHeroDate: heroDateStr,
	}
}

// computeActiveXPPerDay calcule le rythme XP par jour actif en excluant
// les gaps d'inactivité supérieurs à inactivityGapDays.
func computeActiveXPPerDay(history []domain.XPHistoryPoint) float64 {
	if len(history) < 2 {
		return 0.0
	}
	firstXP := history[0].XPTotal
	lastXP := history[len(history)-1].XPTotal
	xpDelta := lastXP - firstXP
	if xpDelta <= 0 {
		return 0.0
	}
	var totalActiveDays float64
	for i := 1; i < len(history); i++ {
		prev := history[i-1].RecordedAt
		curr := history[i].RecordedAt
		gapDays := curr.Sub(prev).Hours() / 24.0
		if gapDays <= inactivityGapDays {
			totalActiveDays += gapDays
		} else {
			totalActiveDays += inactivityGapDays / 2
		}
	}
	if totalActiveDays <= 0 {
		return 0.0
	}
	return float64(xpDelta) / totalActiveDays
}

// computeFallbackXPPerDay calcule le rythme XP moyen global depuis la première date.
func computeFallbackXPPerDay(xpTotal int, firstDate time.Time) float64 {
	now := time.Now()
	days := now.Sub(firstDate).Hours() / 24.0
	if days <= 0 || xpTotal <= 0 {
		return 0.0
	}
	return float64(xpTotal) / days
}

func buildLUSRSummary(history []domain.LUSRCheckpointDTO) domain.LUSRSummary {
	if len(history) == 0 {
		return domain.LUSRSummary{}
	}
	// Snapshot actif = checkpoint le plus récent avec la valeur la plus élevée
	best := history[0]
	for _, cp := range history {
		if cp.RatingValue > best.RatingValue {
			best = cp
		}
	}
	var trendLabel *string
	if len(history) >= 2 {
		delta := history[len(history)-1].RatingValue - history[len(history)-2].RatingValue
		var s string
		switch {
		case delta > 0:
			s = fmt.Sprintf("+%.0f", delta)
		case delta < 0:
			s = fmt.Sprintf("%.0f", delta)
		default:
			s = trendLabelStable
		}
		trendLabel = &s
	}
	return domain.LUSRSummary{
		CurrentRating:        &best.RatingValue,
		CurrentTierLabel:     best.TierLabel,
		CurrentPlaylistGroup: best.PlaylistGroup,
		TrendLabel:           trendLabel,
		Checkpoints:          history,
	}
}

// ---------------------------------------------------------------------------
// Top matches helpers
// ---------------------------------------------------------------------------

func splitTopRows(rows []domain.TopMatchRawRow) (best, worst []domain.TopMatchRawRow) {
	for _, r := range rows {
		if r.Outcome == 2 { // WIN
			best = append(best, r)
		} else {
			worst = append(worst, r)
		}
	}
	return
}

func reverseTopMatches(rows []domain.TopMatchRawRow) {
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
}

func convertTopMatches(rows []domain.TopMatchRawRow) []domain.TopMatchDTO {
	out := make([]domain.TopMatchDTO, 0, len(rows))
	for _, r := range rows {
		mapUI := derefStr(r.MapName)
		modeUI := derefStr(r.PairName)
		var mapPtr, modePtr *string
		if mapUI != "" {
			mapPtr = &mapUI
		}
		if modeUI != "" {
			modePtr = &modeUI
		}
		out = append(out, domain.TopMatchDTO{
			MatchID:          r.MatchID,
			PerformanceScore: r.PerformanceScore,
			MapUI:            mapPtr,
			ModeUI:           modePtr,
			OutcomeCode:      r.Outcome,
			OutcomeLabel:     outcomeLabel(r.Outcome),
			Kills:            r.Kills,
			Deaths:           r.Deaths,
			KDA:              r.KDA,
		})
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
