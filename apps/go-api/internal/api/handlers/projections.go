// Package handlers — projections partagées entre handlers (refactor de
// l'inline qui existait dans ExplorerHandler.QueryMatches).
package handlers

import "levelup/go-api/internal/domain"

// BuildExplorerRowFromMatchHistory projette une MatchHistoryRow (issue de
// MatchHistoryService.GetPage) vers une ExplorerMatchesRow. Utilisée par
// l'Explorer (table principale) et la page Carrière (Matchs marquants).
func BuildExplorerRowFromMatchHistory(item domain.MatchHistoryRow) domain.ExplorerMatchesRow {
	var deltaPerf *int
	if item.PerformanceScoreRelative != nil {
		v := *item.PerformanceScoreRelative - 50
		deltaPerf = &v
	}
	return domain.ExplorerMatchesRow{
		MatchID:             item.MatchID,
		StartTime:           item.StartTime,
		StartTimeLabel:      item.StartTimeLabel,
		MapUI:               item.MapUI,
		ModeUI:              item.ModeUI,
		PlaylistLabel:       item.PlaylistLabel,
		OutcomeCode:         item.OutcomeCode,
		OutcomeLabel:        item.OutcomeLabel,
		ScoreLabel:          item.ScoreLabel,
		IsWithFriends:       item.IsWithFriends,
		ExperienceTypeLabel: item.ExperienceTypeLabel,
		MatchURL:            item.MatchURL,
		Kills:               item.Kills,
		Deaths:              item.Deaths,
		Assists:             item.Assists,
		PerfScore:           item.PerformanceScoreRelative,
		PerfTier:            item.PerfTier,
		DeltaPerf:           deltaPerf,
		SkillTierLabel:      item.SkillTierLabel,
		RatingType:          item.SkillRatingType,
		ExpectedWinProb:     item.ExpectedWinProb,
		PlacementDone:       item.PlacementDone,
		PlacementTotal:      item.PlacementTotal,
		PerfPlacementDone:   item.PerfPlacementDone,
		PerfPlacementTotal:  item.PerfPlacementTotal,
		DeltaMMR:            item.DeltaMMR,
		TeamMMR:             item.TeamMMR,
		EnemyMMR:            item.EnemyMMR,
		KDA:                 item.KDA,
		DurationSeconds:     item.DurationSeconds,
		IsOvertime:          item.IsOvertime,
		OvertimeSeconds:     item.OvertimeSeconds,
		DominanceFlag:       item.DominanceFlag,
	}
}
