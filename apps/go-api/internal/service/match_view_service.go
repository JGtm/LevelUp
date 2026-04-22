// Package service — MatchViewService : vue complète d'un match.
//
// Port Go de apps/api/app/services/match_view_service.py.
// Assemble les données des 4 onglets + header à partir du repo.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// outcomeColors : couleur hex par code d'outcome Halo Infinite.
// (outcomeLabels est défini dans match_history_service.go)
var outcomeColors = map[int]string{
	1: "#8b5cf6", // Égalité
	2: "#22c55e", // Victoire
	3: "#ef4444", // Défaite
	4: "#8b5cf6", // Non terminé
}

// MatchViewService assemble la réponse Match View.
type MatchViewService struct {
	repo port.MatchViewRepository
	xuid string
}

// NewMatchViewService crée un MatchViewService.
func NewMatchViewService(repo port.MatchViewRepository, xuid string) *MatchViewService {
	return &MatchViewService{repo: repo, xuid: xuid}
}

// GetMatchView retourne la réponse complète pour un match.
func (s *MatchViewService) GetMatchView(ctx context.Context, matchID string) (domain.MatchViewResponse, error) {
	meta, err := s.repo.GetMatchMeta(ctx, matchID)
	if err != nil {
		return domain.MatchViewResponse{}, fmt.Errorf("MatchViewService: meta: %w", err)
	}

	stats, err := s.repo.GetPlayerMatchStats(ctx, s.xuid, matchID)
	if err != nil {
		slog.Warn("match_view: stats indisponibles", "match_id", matchID, "err", err)
	}
	enrich, err := s.repo.GetMatchEnrichment(ctx, matchID)
	if err != nil {
		slog.Warn("match_view: enrichment indisponible", "match_id", matchID, "err", err)
	}
	scoreboard, err := s.repo.GetMatchScoreboard(ctx, matchID)
	if err != nil {
		slog.Warn("match_view: scoreboard indisponible", "match_id", matchID, "err", err)
	}
	medals, err := s.repo.GetMatchMedals(ctx, s.xuid, matchID)
	if err != nil {
		slog.Warn("match_view: medals indisponibles", "match_id", matchID, "err", err)
	}
	events, err := s.repo.GetMatchEvents(ctx, matchID)
	if err != nil {
		slog.Warn("match_view: events indisponibles", "match_id", matchID, "err", err)
	}
	weapons, err := s.repo.GetMatchWeaponKills(ctx, s.xuid, matchID)
	if err != nil {
		slog.Warn("match_view: weapons indisponibles", "match_id", matchID, "err", err)
	}
	kvPairs, err := s.repo.GetMatchKVPairs(ctx, matchID)
	if err != nil {
		slog.Warn("match_view: kv_pairs indisponibles", "match_id", matchID, "err", err)
	}

	header := buildMatchHeader(matchID, meta, stats, enrich, scoreboard)
	rank := domain.MatchViewRank{RatingType: "none"}
	summary := buildSummaryTab(stats, medals)
	combat := buildCombatTab(weapons, events)
	team := buildTeamTab(scoreboard, kvPairs, s.xuid)

	return domain.MatchViewResponse{
		Header:     header,
		Rank:       rank,
		SummaryTab: summary,
		CombatTab:  combat,
		TeamTab:    team,
		MediaTab:   domain.MatchMediaTab{MediaItems: []domain.MatchAssociatedMedia{}}, CitationsTab: domain.MatchCitationsTab{Commendations: []domain.MatchCitation{}, Medals: []domain.MatchMedal{}},
	}, nil
}

// GetMatchNeighbors retourne les matchs adjacents pour la navigation prev/next.
func (s *MatchViewService) GetMatchNeighbors(ctx context.Context, matchID string) (domain.MatchNeighbors, error) {
	slog.Info("match_view: GetMatchNeighbors", "match_id", matchID, "xuid", s.xuid)
	n, err := s.repo.GetMatchNeighbors(ctx, s.xuid, matchID)
	if err != nil {
		slog.Warn("match_view: neighbors indisponibles", "match_id", matchID, "err", err)
		return domain.MatchNeighbors{}, nil
	}
	if n == nil {
		return domain.MatchNeighbors{}, nil
	}
	slog.Info("match_view: neighbors chargés",
		"match_id", matchID,
		"current_index", n.CurrentIndex,
		"total", n.TotalMatches,
		"has_prev", n.PreviousMatchID != nil,
		"has_next", n.NextMatchID != nil,
	)
	return *n, nil
}

// ---------------------------------------------------------------------------
// Header
// ---------------------------------------------------------------------------

func buildMatchHeader(
	matchID string,
	meta *domain.MatchMetaRaw,
	stats *domain.PlayerMatchStatsRaw,
	enrich *domain.MatchEnrichmentRaw,
	scoreboard []domain.ScoreboardRaw,
) domain.MatchViewHeader {
	h := domain.MatchViewHeader{
		MatchID:      matchID,
		OutcomeLabel: "-",
		OutcomeColor: "#94a3b8",
		PerfDisplay:  "-",
	}

	if meta == nil {
		return h
	}

	h.StartTime = meta.StartTime
	if meta.StartTime != nil {
		h.StartTimeLabel = formatDateFRLong(*meta.StartTime)
	}
	if meta.MapName != nil {
		h.MapUI = *meta.MapName
	}
	if meta.MapAssetID != nil {
		h.MapID = *meta.MapAssetID
	}
	if meta.PairName != nil {
		h.ModeUI = *meta.PairName
	}
	if meta.PlaylistName != nil {
		h.PlaylistLabel = *meta.PlaylistName
	}
	h.PlayableDurationSeconds = meta.PlayableDurationSeconds
	if meta.MapAssetID != nil {
		h.WaypointURL = fmt.Sprintf("https://www.halowaypoint.com/halo-infinite/matches/%s", matchID)
	}

	if stats != nil && stats.OutcomeCode != 0 {
		code := stats.OutcomeCode
		h.OutcomeCode = &code
		h.OutcomeLabel = outcomeLabel(code)
		h.OutcomeColor = outcomeColor(code)
		h.ScoreLabel = buildScoreLabel(scoreboard)
	}

	if enrich != nil {
		if enrich.PerformanceScore != nil {
			perf := *enrich.PerformanceScore
			display := fmt.Sprintf("%.0f", perf)
			h.PerfDisplay = display
			color := perfColor(perf)
			h.PerfColor = &color
		}
		h.IsExcluded = enrich.IsExcluded
	}

	return h
}

// buildScoreLabel calcule "scoreEquipe1-scoreEquipe2" à partir du scoreboard.
func buildScoreLabel(scoreboard []domain.ScoreboardRaw) string {
	teamScores := make(map[int]float64)
	for _, row := range scoreboard {
		if row.TeamID == nil || row.PersonalScore == nil {
			continue
		}
		teamScores[*row.TeamID] += *row.PersonalScore
	}
	if len(teamScores) == 0 {
		return ""
	}
	teams := make([]int, 0, len(teamScores))
	for tid := range teamScores {
		teams = append(teams, tid)
	}
	// Trier pour affichage stable
	sortInts(teams)
	if len(teams) == 2 {
		return fmt.Sprintf("%.0f-%.0f", teamScores[teams[0]], teamScores[teams[1]])
	}
	return ""
}

// ---------------------------------------------------------------------------
// Summary Tab
// ---------------------------------------------------------------------------

func buildSummaryTab(stats *domain.PlayerMatchStatsRaw, medals []domain.MedalRaw) domain.MatchSummaryTab {
	tab := domain.MatchSummaryTab{
		KPIs:           domain.MatchSummaryKpis{},
		PersonalResult: domain.MatchPersonalResult{OutcomeLabel: "-", OutcomeColor: "#94a3b8"},
		Medals:         convertMedals(medals),
		Citations:      []domain.MatchCitation{},
		ExpectedStats:  domain.MatchExpectedStats{},
	}

	if stats == nil {
		return tab
	}

	tab.KPIs = domain.MatchSummaryKpis{
		Kills:       &stats.Kills,
		Deaths:      &stats.Deaths,
		Assists:     &stats.Assists,
		KDA:         stats.KDA,
		DamageDealt: stats.DamageDealt,
		AverageLife: formatLifeSeconds(stats.AvgLifeSeconds),
	}

	if stats.OutcomeCode != 0 {
		score := 0
		if stats.PersonalScore != nil {
			score = int(math.Round(*stats.PersonalScore))
		}
		tab.PersonalResult = domain.MatchPersonalResult{
			OutcomeLabel: outcomeLabel(stats.OutcomeCode),
			OutcomeColor: outcomeColor(stats.OutcomeCode),
			Score:        &score,
			RankInTeam:   stats.RankInTeam,
		}
	}

	return tab
}

func convertMedals(raw []domain.MedalRaw) []domain.MatchMedal {
	if len(raw) == 0 {
		return []domain.MatchMedal{}
	}
	medals := make([]domain.MatchMedal, 0, len(raw))
	for _, r := range raw {
		medals = append(medals, domain.MatchMedal{
			MedalNameID: r.MedalID,
			Name:        r.Label,
			Count:       r.Count,
		})
	}
	return medals
}

// ---------------------------------------------------------------------------
// Combat Tab
// ---------------------------------------------------------------------------

func buildCombatTab(weapons []domain.WeaponKillRaw, events []domain.EventRaw) domain.MatchCombatTab {
	wkList := make([]domain.MatchWeaponKill, 0, len(weapons))
	for _, w := range weapons {
		wkList = append(wkList, domain.MatchWeaponKill{
			WeaponID:    w.WeaponID,
			WeaponLabel: w.WeaponLabel,
			KillCount:   w.Kills,
		})
	}

	evtList := make([]domain.MatchHighlightEvent, 0, len(events))
	for _, e := range events {
		evtList = append(evtList, domain.MatchHighlightEvent{
			EventType:   e.EventType,
			EventTimeMS: e.TimeMS,
			ActorXUID:   e.XUID,
		})
	}

	return domain.MatchCombatTab{
		WeaponKills:     wkList,
		HighlightEvents: evtList,
		TugOfWar:        []domain.MatchTugOfWarBin{},
		ImpactBadges:    []domain.MatchImpactBadge{},
		KDTimeline:      []domain.MatchKDTimelinePoint{},
		NemesisDuels:    []domain.MatchNemesisRow{},
	}
}

// ---------------------------------------------------------------------------
// Team Tab
// ---------------------------------------------------------------------------

func buildTeamTab(scoreboard []domain.ScoreboardRaw, kvPairs []domain.KVPairRaw, myXUID string) domain.MatchTeamTab {
	rows := make([]domain.MatchScoreboardRow, 0, len(scoreboard))
	for _, s := range scoreboard {
		row := domain.MatchScoreboardRow{
			XUID:             s.XUID,
			Gamertag:         s.Gamertag,
			IsMe:             s.XUID == myXUID,
			Rank:             s.RankInTeam,
			Kills:            &s.Kills,
			Deaths:           &s.Deaths,
			Assists:          &s.Assists,
			KDA:              s.KDA,
			Accuracy:         s.Accuracy,
			DamageDealt:      s.DamageDealt,
			DamageTaken:      s.DamageTaken,
			ShotsFired:       s.ShotsFired,
			ShotsHit:         s.ShotsHit,
			AvgLifeSeconds:   s.AvgLifeSeconds,
			HeadshotKills:    s.HeadshotKills,
			MaxKillingSpree:  s.MaxKillingSpree,
			GrenadeKills:     s.GrenadeKills,
			MeleeKills:       s.MeleeKills,
			PowerWeaponKills: s.PowerWeaponKills,
			OutcomeLabel:     outcomeLabel(s.OutcomeCode),
		}
		if s.TeamID != nil {
			team := fmt.Sprintf("t%d", *s.TeamID)
			row.TeamSide = &team
		}
		rows = append(rows, row)
	}

	// Nemesis depuis KV pairs
	nemesisByXUID := buildNemesisMap(kvPairs, myXUID, scoreboard)
	nemesisList := make([]domain.MatchNemesisRow, 0, len(nemesisByXUID))
	for xuid, n := range nemesisByXUID {
		nemesisList = append(nemesisList, domain.MatchNemesisRow{
			XUID:     xuid,
			Gamertag: n.Gamertag,
			KilledMe: n.KilledMe,
			IKilled:  n.IKilled,
		})
	}
	sortNemesisByKilledMe(nemesisList)

	return domain.MatchTeamTab{
		Roster:     []domain.MatchRosterRow{},
		Scoreboard: rows,
		Nemesis:    nemesisList,
		Encounters: []domain.MatchEncounterRow{},
	}
}

type nemesisEntry struct {
	Gamertag string
	KilledMe int
	IKilled  int
}

func buildNemesisMap(
	kvPairs []domain.KVPairRaw,
	myXUID string,
	scoreboard []domain.ScoreboardRaw,
) map[string]*nemesisEntry {
	gtMap := make(map[string]string, len(scoreboard))
	for _, s := range scoreboard {
		gtMap[s.XUID] = s.Gamertag
	}

	result := make(map[string]*nemesisEntry)
	for _, kv := range kvPairs {
		if kv.VictimXUID == myXUID {
			if _, ok := result[kv.KillerXUID]; !ok {
				gt := gtMap[kv.KillerXUID]
				if gt == "" {
					gt = kv.KillerGT
				}
				result[kv.KillerXUID] = &nemesisEntry{Gamertag: gt}
			}
			result[kv.KillerXUID].KilledMe += kv.KillCount
		}
		if kv.KillerXUID == myXUID {
			if _, ok := result[kv.VictimXUID]; !ok {
				gt := gtMap[kv.VictimXUID]
				if gt == "" {
					gt = kv.VictimGT
				}
				result[kv.VictimXUID] = &nemesisEntry{Gamertag: gt}
			}
			result[kv.VictimXUID].IKilled += kv.KillCount
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
// outcomeLabel est défini dans match_history_service.go (même package).

func outcomeColor(code int) string {
	if c, ok := outcomeColors[code]; ok {
		return c
	}
	return "#94a3b8"
}

func perfColor(score float64) string {
	switch {
	case score >= 80:
		return "#22c55e"
	case score >= 60:
		return "#3b82f6"
	case score >= 40:
		return "#f59e0b"
	default:
		return "#ef4444"
	}
}

// formatDateFRLong formate une date en "JJ mois AAAA, HH:MM" (FR long).
// Distinct de formatDateFR (match_history) qui utilise le format court.
func formatDateFRLong(t time.Time) string {
	months := [...]string{
		"janv.", "févr.", "mars", "avr.", "mai", "juin",
		"juil.", "août", "sept.", "oct.", "nov.", "déc.",
	}
	local := t.Local()
	return fmt.Sprintf("%02d %s %d, %02d:%02d",
		local.Day(), months[local.Month()-1], local.Year(),
		local.Hour(), local.Minute())
}

func sortInts(s []int) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func sortNemesisByKilledMe(s []domain.MatchNemesisRow) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j].KilledMe > s[j-1].KilledMe; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// formatLifeSeconds est défini dans match_history_service.go (même package).
