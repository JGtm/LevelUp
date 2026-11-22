package replayview

// convert_medals.go — medailles, score par action, horloges et regles de la variante. Jumeau de
// `domain/replaydoc/{medals,score_actions,clocks}.go`.

import (
	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/replaytext"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

func toMedal(v replay.Medal) replaydoc.Medal {
	return replaydoc.Medal{T: v.T, MS: v.MS, XUID: v.XUID, ID: v.ID, Name: v.Name}
}

// toMedalLabel recopie un libelle de medaille. Le calque est resolu a la requete (`service/replay_medal_labels.go`)
// et l'artefact ne l'ecrit jamais : la projection d'un artefact rend donc une table absente.
func toMedalLabel(v replay.MedalLabel) replaydoc.MedalLabel {
	out := replaydoc.MedalLabel{
		Text:       replaytext.TexteDepuisFilm(v.Text),
		Img:        v.Img,
		Difficulty: v.Difficulty,
	}
	if v.Description != nil {
		out.Description = &replaydoc.MedalDescription{Text: replaytext.TexteDepuisFilm(v.Description.Text)}
	}
	if v.Sprite != nil {
		out.Sprite = &replaydoc.MedalSprite{
			Sheet: v.Sprite.Sheet, Left: v.Sprite.Left, Top: v.Sprite.Top, Width: v.Sprite.Width, Height: v.Sprite.Height,
		}
	}
	return out
}

func toMedalCoverage(v replay.MedalCoverage) replaydoc.MedalCoverage {
	return replaydoc.MedalCoverage{
		Decoded:      v.Decoded,
		Published:    v.Published,
		NoOrigin:     v.NoOrigin,
		BeforeOrigin: v.BeforeOrigin,
		AfterEnd:     v.AfterEnd,
		UnknownName:  v.UnknownName,
	}
}

func toScoreAction(v replay.ScoreAction) replaydoc.ScoreAction {
	return replaydoc.ScoreAction{T: v.T, XUID: v.XUID, Name: v.Name, Points: v.Points, Target: v.Target}
}

func toScoreActionCoverage(v replay.ScoreActionCoverage) replaydoc.ScoreActionCoverage {
	return replaydoc.ScoreActionCoverage{
		Complete:           v.Complete,
		Group:              v.Group,
		GroupMissing:       v.GroupMissing,
		Frames:             v.Frames,
		ClosedFrames:       v.ClosedFrames,
		StoppedFrames:      v.StoppedFrames,
		ViewAUnreadable:    v.ViewAUnreadable,
		Messages:           v.Messages,
		Copies:             v.Copies,
		ScoreMessages:      v.ScoreMessages,
		Unrelated:          v.Unrelated,
		NamedWithoutPoints: v.NamedWithoutPoints,
		OutsideTree:        v.OutsideTree,
		UnconfirmedRank:    v.UnconfirmedRank,
		Scored:             v.Scored,
		Published:          v.Published,
		NoRecipient:        v.NoRecipient,
		NotPresent:         v.NotPresent,
		NoOrigin:           v.NoOrigin,
		BeforeOrigin:       v.BeforeOrigin,
		AfterEnd:           v.AfterEnd,
		NoTarget:           v.NoTarget,
	}
}

func toClockSample(v replay.ClockSample) replaydoc.ClockSample {
	return replaydoc.ClockSample{T: v.T, Left: v.Left, Limit: v.Limit, State: v.State}
}

func toRespawnCountdown(v replay.RespawnCountdown) replaydoc.RespawnCountdown {
	return replaydoc.RespawnCountdown{
		FilmIndex: v.FilmIndex,
		From:      v.From,
		To:        v.To,
		Seconds:   v.Seconds,
		StartRead: v.StartRead,
		EndRead:   v.EndRead,
	}
}

func toMatchRules(v replay.MatchRules) replaydoc.MatchRules {
	return replaydoc.MatchRules{
		MinRespawnS:      v.MinRespawnS,
		SuicidePenaltyS:  v.SuicidePenaltyS,
		BetrayalPenaltyS: v.BetrayalPenaltyS,
		FlagReturnS:      v.FlagReturnS,
		FriendlyFire:     v.FriendlyFire,
	}
}

func toClocksCoverage(v replay.ClocksCoverage) replaydoc.ClocksCoverage {
	return replaydoc.ClocksCoverage{
		Frames:                  v.Frames,
		FramesUnread:            v.FramesUnread,
		ClockReads:              v.ClockReads,
		ClockPublished:          v.ClockPublished,
		ClockImpossible:         v.ClockImpossible,
		ClockUnmeasuredState:    v.ClockUnmeasuredState,
		CountdownReads:          v.CountdownReads,
		CountdownImpossible:     v.CountdownImpossible,
		CountdownNoPlayer:       v.CountdownNoPlayer,
		CountdownIsolated:       v.CountdownIsolated,
		WithheldUnknownIndex:    v.WithheldUnknownIndex,
		WithheldReusedIndex:     v.WithheldReusedIndex,
		WithheldOutsidePresence: v.WithheldOutsidePresence,
		Countdowns:              v.Countdowns,
		StartDeduced:            v.StartDeduced,
		EndDeduced:              v.EndDeduced,
	}
}
