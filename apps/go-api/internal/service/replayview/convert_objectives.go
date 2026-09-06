package replayview

// convert_objectives.go — score dans le temps, drapeaux, zones, couronne, crane, bombe.
// Jumeau de `domain/replaydoc/objectives.go`.

import (
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/replaydoc"
)

func toObjectiveAction(v replay.ObjectiveAction) replaydoc.ObjectiveAction {
	return replaydoc.ObjectiveAction{
		T:      v.T,
		XUID:   v.XUID,
		Stat:   v.Stat,
		TimeMS: v.TimeMS,
	}
}

func toScoreTimeline(v replay.ScoreTimeline) replaydoc.ScoreTimeline {
	return replaydoc.ScoreTimeline{
		Teams:             sliceOf(v.Teams, toTeamScore),
		Players:           sliceOf(v.Players, toPlayerScore),
		TargetScore:       v.TargetScore,
		HoldTicks:         sliceOf(v.HoldTicks, toTeamHold),
		HoldTicksPerPoint: v.HoldTicksPerPoint,
	}
}

func toTeamScore(v replay.TeamScore) replaydoc.TeamScore {
	return replaydoc.TeamScore{
		TeamID: v.TeamID,
		Rounds: sliceOf(v.Rounds, toScoreRound),
		Total:  sliceOf(v.Total, toScoreTick),
	}
}

func toScoreRound(v replay.ScoreRound) replaydoc.ScoreRound {
	return replaydoc.ScoreRound{
		Round:  v.Round,
		Points: sliceOf(v.Points, toScoreTick),
	}
}

func toScoreTick(v replay.ScoreTick) replaydoc.ScoreTick {
	return replaydoc.ScoreTick{
		T: v.T,
		V: v.V,
	}
}

func toPlayerScore(v replay.PlayerScore) replaydoc.PlayerScore {
	return replaydoc.PlayerScore{
		XUID:    v.XUID,
		Score:   toScoreSeries(v.Score),
		Kills:   toScoreSeries(v.Kills),
		Deaths:  toScoreSeries(v.Deaths),
		Assists: toScoreSeries(v.Assists),
	}
}

func toScoreSeries(v replay.ScoreSeries) replaydoc.ScoreSeries {
	return replaydoc.ScoreSeries{
		Rounds: sliceOf(v.Rounds, toScoreRound),
		Total:  sliceOf(v.Total, toScoreTick),
	}
}

func toTeamHold(v replay.TeamHold) replaydoc.TeamHold {
	return replaydoc.TeamHold{
		TeamID: v.TeamID,
		Ticks:  sliceOf(v.Ticks, toScoreTick),
	}
}

func toFlagCarry(v replay.FlagCarry) replaydoc.FlagCarry {
	return replaydoc.FlagCarry{
		Team:  v.Team,
		Spans: sliceOf(v.Spans, toFlagSpan),
	}
}

func toFlagSpan(v replay.FlagSpan) replaydoc.FlagSpan {
	return replaydoc.FlagSpan{
		State: v.State,
		T0:    v.T0,
		T1:    v.T1,
		XUID:  v.XUID,
		X:     v.X,
		Y:     v.Y,
	}
}

func toFlagReturnZone(v replay.FlagReturnZone) replaydoc.FlagReturnZone {
	return replaydoc.FlagReturnZone{
		RadiusM:      v.RadiusM,
		ResetSeconds: v.ResetSeconds,
		SoloSeconds:  v.SoloSeconds,
	}
}

func toObjectiveObjectLife(v replay.ObjectiveObjectLife) replaydoc.ObjectiveObjectLife {
	return replaydoc.ObjectiveObjectLife{
		Family: v.Family,
		En:     v.En,
		Fr:     v.Fr,
		T0:     v.T0,
		T1:     v.T1,
		Pts:    sliceOf(v.Pts, toObjectiveObjectPoint),
	}
}

func toObjectiveObjectPoint(v replay.ObjectiveObjectPoint) replaydoc.ObjectiveObjectPoint {
	return replaydoc.ObjectiveObjectPoint{
		T: v.T,
		X: v.X,
		Y: v.Y,
	}
}

func toZoneState(v replay.ZoneState) replaydoc.ZoneState {
	return replaydoc.ZoneState{
		ZoneRef:    v.ZoneRef,
		LetterRank: v.LetterRank,
		Key:        v.Key,
		Spans:      sliceOf(v.Spans, toZoneSpan),
		Gauge:      sliceOf(v.Gauge, toGaugePoint),
	}
}

func toZoneSpan(v replay.ZoneSpan) replaydoc.ZoneSpan {
	return replaydoc.ZoneSpan{
		T0:       v.T0,
		T1:       v.T1,
		Owner:    v.Owner,
		Progress: v.Progress,
		Active:   v.Active,
	}
}

func toGaugePoint(v replay.GaugePoint) replaydoc.GaugePoint {
	return replaydoc.GaugePoint{
		T: v.T,
		V: v.V,
	}
}

func toVipPeriod(v replay.VipPeriod) replaydoc.VipPeriod {
	return replaydoc.VipPeriod{
		XUID:   v.XUID,
		T0:     v.T0,
		T1:     v.T1,
		Closed: v.Closed,
	}
}

func toSkullCarry(v replay.SkullCarry) replaydoc.SkullCarry {
	return replaydoc.SkullCarry{
		XUID:   v.XUID,
		T0:     v.T0,
		T1:     v.T1,
		Closed: v.Closed,
	}
}

func toBombArming(v replay.BombArming) replaydoc.BombArming {
	return replaydoc.BombArming{
		T:       v.T,
		TimeMS:  v.TimeMS,
		StartT:  v.StartT,
		StartMS: v.StartMS,
		FuseMS:  v.FuseMS,
	}
}

func toBombCarry(v replay.BombCarry) replaydoc.BombCarry {
	return replaydoc.BombCarry{
		XUID:   v.XUID,
		T0:     v.T0,
		T1:     v.T1,
		Closed: v.Closed,
	}
}

func toBombMatchStats(v replay.BombMatchStats) replaydoc.BombMatchStats {
	return replaydoc.BombMatchStats{
		Players:  sliceOf(v.Players, toBombPlayerStats),
		Coverage: toBombStatsCoverage(v.Coverage),
	}
}

func toBombPlayerStats(v replay.BombPlayerStats) replaydoc.BombPlayerStats {
	return replaydoc.BombPlayerStats{
		XUID:                 v.XUID,
		Detonations:          v.Detonations,
		Arms:                 v.Arms,
		Grabs:                v.Grabs,
		TimeAsCarrierSeconds: v.TimeAsCarrierSeconds,
		CarriersKilled:       v.CarriersKilled,
	}
}

func toBombEvent(v replay.BombEvent) replaydoc.BombEvent {
	return replaydoc.BombEvent{
		Type:        v.Type,
		TimeMS:      v.TimeMS,
		XUID:        v.XUID,
		ActorSource: v.ActorSource,
	}
}
