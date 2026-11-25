package replayview

// convert_reloads_eliminations.go — les rechargements et les éliminations du document, leurs couvertures, et la
// couverture de la commande de déplacement, les gestes et leur couverture. Jumeaux de `domain/replaydoc/reloads.go`,
// `domain/replaydoc/eliminations.go`, `domain/replaydoc/gestures.go` et `domain/replaydoc/coverage_stances.go`.

import (
	"levelup/go-api/internal/domain/replaydoc"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

func toReload(v replay.Reload) replaydoc.Reload {
	return replaydoc.Reload{T: v.T, Place: v.Place}
}

func toReloadCoverage(v replay.ReloadCoverage) replaydoc.ReloadCoverage {
	return replaydoc.ReloadCoverage{StoppedFrames: v.StoppedFrames, WithoutPlayer: v.WithoutPlayer}
}

func toElimination(v replay.Elimination) replaydoc.Elimination {
	return replaydoc.Elimination{
		T: v.T, XUID: v.XUID, Place: v.Place, Victim: v.Victim, VictimPlace: v.VictimPlace,
		Headshot: v.Headshot, Perfect: v.Perfect,
	}
}

func toEliminationCoverage(replay.EliminationCoverage) replaydoc.EliminationCoverage {
	return replaydoc.EliminationCoverage{}
}

func toGesture(v replay.Gesture) replaydoc.Gesture {
	return replaydoc.Gesture{T: v.T, Place: v.Place, Kind: v.Kind}
}

func toChatter(v replay.Chatter) replaydoc.Chatter {
	return replaydoc.Chatter{T: v.T, Player: v.Player, Sound: v.Sound, Name: v.Name}
}

func toGestureCoverage(v replay.GestureCoverage) replaydoc.GestureCoverage {
	return replaydoc.GestureCoverage{Gated: v.Gated, StoppedFrames: v.StoppedFrames, WithoutPlayer: v.WithoutPlayer}
}

func toStanceControlCoverage(v replay.StanceControlCoverage) replaydoc.StanceControlCoverage {
	return replaydoc.StanceControlCoverage{WithHole: v.WithHole, NoTrack: v.NoTrack, NoPlayer: v.NoPlayer}
}
