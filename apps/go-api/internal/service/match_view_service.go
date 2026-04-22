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

	"golang.org/x/sync/errgroup"
	"levelup/go-api/internal/analysis"
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
func (s *MatchViewService) GetMatchView(ctx context.Context, matchID string) (domain.MatchViewResponse, error) { //nolint:cyclop
	// --- Appels séquentiels bloquants (meta est nécessaire pour la suite) ---
	meta, err := s.repo.GetMatchMeta(ctx, matchID)
	if err != nil {
		return domain.MatchViewResponse{}, fmt.Errorf("MatchViewService: meta: %w", err)
	}

	// --- Appels parallèles via errgroup ---
	var (
		stats      *domain.PlayerMatchStatsRaw
		enrich     *domain.MatchEnrichmentRaw
		scoreboard []domain.ScoreboardRaw
		medals     []domain.MedalRaw
		events     []domain.EventRaw
		weapons    []domain.WeaponKillRaw
		kvPairs    []domain.KVPairRaw
		skillRank  *domain.SkillRankRaw
		encounters []domain.EncounterRaw
		media      []domain.MediaAssocRaw
		expected   *domain.ExpectedStatsRaw
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var e error
		stats, e = s.repo.GetPlayerMatchStats(gctx, s.xuid, matchID)
		if e != nil {
			slog.Warn("match_view: stats indisponibles", "match_id", matchID, "err", e)
		}
		return nil
	})
	g.Go(func() error {
		var e error
		enrich, e = s.repo.GetMatchEnrichment(gctx, matchID)
		if e != nil {
			slog.Warn("match_view: enrichment indisponible", "match_id", matchID, "err", e)
		}
		return nil
	})
	g.Go(func() error {
		var e error
		scoreboard, e = s.repo.GetMatchScoreboard(gctx, matchID)
		if e != nil {
			slog.Warn("match_view: scoreboard indisponible", "match_id", matchID, "err", e)
		}
		return nil
	})
	g.Go(func() error {
		var e error
		medals, e = s.repo.GetMatchMedals(gctx, s.xuid, matchID)
		if e != nil {
			slog.Warn("match_view: medals indisponibles", "match_id", matchID, "err", e)
		}
		return nil
	})
	g.Go(func() error {
		var e error
		events, e = s.repo.GetMatchEvents(gctx, matchID)
		if e != nil {
			slog.Warn("match_view: events indisponibles", "match_id", matchID, "err", e)
		}
		return nil
	})
	g.Go(func() error {
		var e error
		weapons, e = s.repo.GetMatchWeaponKills(gctx, s.xuid, matchID)
		if e != nil {
			slog.Warn("match_view: weapons indisponibles", "match_id", matchID, "err", e)
		}
		return nil
	})
	g.Go(func() error {
		var e error
		kvPairs, e = s.repo.GetMatchKVPairs(gctx, matchID)
		if e != nil {
			slog.Warn("match_view: kv_pairs indisponibles", "match_id", matchID, "err", e)
		}
		return nil
	})
	g.Go(func() error {
		var e error
		skillRank, e = s.repo.GetMatchSkillRank(gctx, matchID)
		if e != nil {
			slog.Warn("match_view: skill_rank indisponible", "match_id", matchID, "err", e)
		}
		return nil
	})
	g.Go(func() error {
		var e error
		encounters, e = s.repo.GetMatchEncounters(gctx, matchID, s.xuid)
		if e != nil {
			slog.Warn("match_view: encounters indisponibles", "match_id", matchID, "err", e)
		}
		return nil
	})
	g.Go(func() error {
		var e error
		// playerSlug = xuid (identifiant de stockage dans shared_social)
		media, e = s.repo.GetMatchMedia(gctx, matchID, s.xuid)
		if e != nil {
			slog.Warn("match_view: media indisponibles", "match_id", matchID, "err", e)
		}
		return nil
	})
	g.Go(func() error {
		var e error
		expected, e = s.repo.GetMatchExpectedStats(gctx, matchID, s.xuid)
		if e != nil {
			slog.Warn("match_view: expected_stats indisponibles", "match_id", matchID, "err", e)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return domain.MatchViewResponse{}, err
	}

	// Durée pour les bins tug-of-war
	var durationMS int64
	if meta.PlayableDurationSeconds != nil {
		durationMS = *meta.PlayableDurationSeconds * 1000
	}

	header := buildMatchHeader(matchID, meta, stats, enrich, scoreboard)
	rank := buildRankBlock(skillRank)
	summary := buildSummaryTabFull(stats, medals, expected)
	combat := buildCombatTabFull(weapons, events, kvPairs, s.xuid, durationMS)
	team := buildTeamTabFull(scoreboard, kvPairs, encounters, s.xuid)
	mediaTab := buildMediaTab(media)

	return domain.MatchViewResponse{
		Header:       header,
		Rank:         rank,
		SummaryTab:   summary,
		CombatTab:    combat,
		TeamTab:      team,
		MediaTab:     mediaTab,
		CitationsTab: domain.MatchCitationsTab{Commendations: []domain.MatchCitation{}, Medals: []domain.MatchMedal{}},
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

// buildRankBlock construit le bloc rank depuis SkillRankRaw.
func buildRankBlock(sr *domain.SkillRankRaw) domain.MatchViewRank {
	if sr == nil {
		return domain.MatchViewRank{RatingType: "none"}
	}
	rank := domain.MatchViewRank{RatingType: sr.RatingType}
	if sr.TierLabel != nil {
		rank.TierLabel = sr.TierLabel
	}
	rank.NumericVal = sr.RatingValue
	rank.DeltaValue = sr.RatingDelta
	return rank
}

// ---------------------------------------------------------------------------
// Summary Tab
// ---------------------------------------------------------------------------

func buildSummaryTabFull(stats *domain.PlayerMatchStatsRaw, medals []domain.MedalRaw, expected *domain.ExpectedStatsRaw) domain.MatchSummaryTab {
	tab := domain.MatchSummaryTab{
		KPIs:           domain.MatchSummaryKpis{},
		PersonalResult: domain.MatchPersonalResult{OutcomeLabel: "-", OutcomeColor: "#94a3b8"},
		Medals:         convertMedals(medals),
		Citations:      []domain.MatchCitation{},
		ExpectedStats:  buildExpectedStats(expected),
	}

	if stats == nil {
		return tab
	}

	tab.KPIs = domain.MatchSummaryKpis{
		Kills:         &stats.Kills,
		Deaths:        &stats.Deaths,
		Assists:       &stats.Assists,
		KDA:           stats.KDA,
		DamageDealt:   stats.DamageDealt,
		AverageLife:   formatLifeSeconds(stats.AvgLifeSeconds),
		Accuracy:      stats.Accuracy,
		PersonalScore: toIntPtr(stats.PersonalScore),
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

// buildExpectedStats construit le bloc de stats attendues.
func buildExpectedStats(e *domain.ExpectedStatsRaw) domain.MatchExpectedStats {
	if e == nil {
		return domain.MatchExpectedStats{}
	}
	return domain.MatchExpectedStats{
		HasExpectedData: e.KillsExpected != nil || e.DeathsExpected != nil,
		ExpectedKills:   e.KillsExpected,
		ExpectedDeaths:  e.DeathsExpected,
		ExpectedAssists: e.AssistsExpected,
		HasHistAvg:      false, // calculé séparément via ComputeModeCategoryAverages si historique disponible
	}
}

func toIntPtr(f *float64) *int {
	if f == nil {
		return nil
	}
	v := int(math.Round(*f))
	return &v
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

func buildCombatTabFull(
	weapons []domain.WeaponKillRaw,
	events []domain.EventRaw,
	kvPairs []domain.KVPairRaw,
	myXUID string,
	durationMS int64,
) domain.MatchCombatTab {
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

	// Tug-of-war
	tugEvents := buildTugEvents(kvPairs, myXUID)
	tugBins := analysis.ComputeTugOfWar(tugEvents, durationMS, 0)
	tugDomain := make([]domain.MatchTugOfWarBin, 0, len(tugBins))
	for _, b := range tugBins {
		allyKills := 0
		enemyKills := 0
		if b.Delta > 0 {
			allyKills = b.Delta
		} else {
			enemyKills = -b.Delta
		}
		tugDomain = append(tugDomain, domain.MatchTugOfWarBin{
			BinStart:   int(b.BinStartMS / 1000),
			BinEnd:     int(b.BinEndMS / 1000),
			TeamKills:  allyKills,
			EnemyKills: enemyKills,
			NetKills:   b.CumDelta,
		})
	}

	// Impact badges
	impactInput := buildImpactInput(kvPairs, myXUID)
	badges := analysis.ComputeSingleMatchImpact(impactInput)
	badgesDomain := make([]domain.MatchImpactBadge, 0, len(badges))
	for _, b := range badges {
		badgesDomain = append(badgesDomain, domain.MatchImpactBadge{
			Key:   b.BadgeKey,
			Label: b.BadgeFR,
		})
	}

	// KD timeline
	kdEvents := buildKDEvents(kvPairs, myXUID)
	kdPoints := analysis.ComputeKDTimeline(kdEvents, myXUID)
	kdDomain := make([]domain.MatchKDTimelinePoint, 0, len(kdPoints))
	for _, p := range kdPoints {
		kdDomain = append(kdDomain, domain.MatchKDTimelinePoint{
			TimeSeconds: int(p.TimeMS / 1000),
			Kills:       p.CumKills,
			Deaths:      p.CumDeaths,
		})
	}

	return domain.MatchCombatTab{
		WeaponKills:     wkList,
		HighlightEvents: evtList,
		TugOfWar:        tugDomain,
		ImpactBadges:    badgesDomain,
		KDTimeline:      kdDomain,
		NemesisDuels:    []domain.MatchNemesisRow{},
	}
}

func buildTugEvents(kvPairs []domain.KVPairRaw, myXUID string) []analysis.TugOfWarEvent {
	events := make([]analysis.TugOfWarEvent, 0, len(kvPairs))
	for _, kv := range kvPairs {
		isAlly := kv.KillerXUID == myXUID
		events = append(events, analysis.TugOfWarEvent{
			TimeMS:    kv.TimeMS,
			IsAlly:    isAlly,
			EventType: "kill",
		})
	}
	return events
}

func buildImpactInput(kvPairs []domain.KVPairRaw, myXUID string) analysis.MatchImpactInput {
	impactEvents := make([]analysis.ImpactEvent, 0, len(kvPairs))
	myKills := 0
	for _, kv := range kvPairs {
		impactEvents = append(impactEvents, analysis.ImpactEvent{
			TimeMS:     kv.TimeMS,
			KillerXUID: kv.KillerXUID,
			VictimXUID: kv.VictimXUID,
		})
		if kv.KillerXUID == myXUID {
			myKills += kv.KillCount
		}
	}
	return analysis.MatchImpactInput{
		KillEvents: impactEvents,
		MyXUID:     myXUID,
		MyKills:    myKills,
	}
}

func buildKDEvents(kvPairs []domain.KVPairRaw, myXUID string) []analysis.KDEvent {
	events := make([]analysis.KDEvent, 0, len(kvPairs)*2)
	for _, kv := range kvPairs {
		if kv.KillerXUID == myXUID {
			events = append(events, analysis.KDEvent{
				TimeMS:    kv.TimeMS,
				IsKill:    true,
				ActorXUID: myXUID,
			})
		}
		if kv.VictimXUID == myXUID {
			events = append(events, analysis.KDEvent{
				TimeMS:    kv.TimeMS,
				IsKill:    false,
				ActorXUID: myXUID,
			})
		}
	}
	return events
}

// ---------------------------------------------------------------------------
// Team Tab
// ---------------------------------------------------------------------------

func buildTeamTabFull(
	scoreboard []domain.ScoreboardRaw,
	kvPairs []domain.KVPairRaw,
	encounters []domain.EncounterRaw,
	myXUID string,
) domain.MatchTeamTab {
	rows := make([]domain.MatchScoreboardRow, 0, len(scoreboard))
	for _, s := range scoreboard {
		// Combat Yield calculé pour ce joueur
		var oc, dr, dpk, dpd *float64
		if s.DamageDealt != nil && s.DamageTaken != nil {
			cy := analysis.ComputeCombatYield(s.Kills, s.Assists, *s.DamageDealt, *s.DamageTaken, s.Deaths)
			oc = &cy.OffensiveConversion
			dr = &cy.DefensiveResistance
		}
		if s.DamageDealt != nil && s.Kills > 0 {
			v := *s.DamageDealt / float64(s.Kills)
			dpk = &v
		}
		if s.DamageTaken != nil && s.Deaths > 0 {
			v := *s.DamageTaken / float64(s.Deaths)
			dpd = &v
		}

		row := domain.MatchScoreboardRow{
			XUID:                s.XUID,
			Gamertag:            s.Gamertag,
			IsMe:                s.XUID == myXUID,
			Rank:                s.RankInTeam,
			Kills:               &s.Kills,
			Deaths:              &s.Deaths,
			Assists:             &s.Assists,
			KDA:                 s.KDA,
			Accuracy:            s.Accuracy,
			DamageDealt:         s.DamageDealt,
			DamageTaken:         s.DamageTaken,
			ShotsFired:          s.ShotsFired,
			ShotsHit:            s.ShotsHit,
			AvgLifeSeconds:      s.AvgLifeSeconds,
			HeadshotKills:       s.HeadshotKills,
			MaxKillingSpree:     s.MaxKillingSpree,
			GrenadeKills:        s.GrenadeKills,
			MeleeKills:          s.MeleeKills,
			PowerWeaponKills:    s.PowerWeaponKills,
			OutcomeLabel:        outcomeLabel(s.OutcomeCode),
			Score:               toIntPtr(s.PersonalScore),
			PerfectKills:        &s.PerfectKills,
			TopWeaponID:         s.TopWeaponID,
			OffensiveConversion: oc,
			DefensiveResistance: dr,
			DamagePerKill:       dpk,
			DamagePerDeath:      dpd,
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
		Encounters: convertEncounters(encounters),
	}
}

func convertEncounters(raw []domain.EncounterRaw) []domain.MatchEncounterRow {
	if len(raw) == 0 {
		return []domain.MatchEncounterRow{}
	}
	result := make([]domain.MatchEncounterRow, 0, len(raw))
	for _, e := range raw {
		result = append(result, domain.MatchEncounterRow{
			XUID:          e.XUID,
			Gamertag:      e.Gamertag,
			CountTogether: e.CountTogether,
			IsAlly:        e.IsAlly,
		})
	}
	return result
}

// buildMediaTab construit l'onglet médias.
func buildMediaTab(media []domain.MediaAssocRaw) domain.MatchMediaTab {
	if len(media) == 0 {
		return domain.MatchMediaTab{MediaItems: []domain.MatchAssociatedMedia{}}
	}
	items := make([]domain.MatchAssociatedMedia, 0, len(media))
	for _, m := range media {
		items = append(items, domain.MatchAssociatedMedia{
			FileID:       m.FileID,
			FileName:     m.FileName,
			FilePath:     m.FilePath,
			ThumbnailURL: m.ThumbnailPath,
			CaptureTime:  m.CaptureTime,
			Liked:        m.Liked,
		})
	}
	return domain.MatchMediaTab{MediaItems: items}
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
