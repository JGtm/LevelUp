package replayview

// convert_reloads_eliminations.go — les rechargements et les éliminations du document, leurs couvertures, et la
// couverture de la commande de déplacement. Jumeaux de `domain/replaydoc/reloads.go`,
// `domain/replaydoc/eliminations.go` et `domain/replaydoc/coverage_stances.go`.

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
	return replaydoc.Elimination{T: v.T, XUID: v.XUID, Victim: v.Victim, Headshot: v.Headshot, Perfect: v.Perfect}
}

func toEliminationCoverage(replay.EliminationCoverage) replaydoc.EliminationCoverage {
	return replaydoc.EliminationCoverage{}
}

func toStanceControlCoverage(v replay.StanceControlCoverage) replaydoc.StanceControlCoverage {
	return replaydoc.StanceControlCoverage{WithHole: v.WithHole, NoTrack: v.NoTrack, NoPlayer: v.NoPlayer}
}
