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
	return replaydoc.ReloadCoverage{
		Frames:        v.Frames,
		ClosedFrames:  v.ClosedFrames,
		StoppedFrames: v.StoppedFrames,
		Read:          v.Read,
		Copies:        v.Copies,
		WithoutPlayer: v.WithoutPlayer,
		BeforeOrigin:  v.BeforeOrigin,
		AfterEnd:      v.AfterEnd,
		Published:     v.Published,
	}
}

func toElimination(v replay.Elimination) replaydoc.Elimination {
	return replaydoc.Elimination{T: v.T, XUID: v.XUID, Victim: v.Victim, Headshot: v.Headshot, Perfect: v.Perfect}
}

func toEliminationCoverage(v replay.EliminationCoverage) replaydoc.EliminationCoverage {
	return replaydoc.EliminationCoverage{
		Eliminations:     v.Eliminations,
		WithVictim:       v.WithVictim,
		HeadYes:          v.HeadYes,
		HeadNo:           v.HeadNo,
		HeadUnknown:      v.HeadUnknown,
		HeadNoCounter:    v.HeadNoCounter,
		HeadAmbiguous:    v.HeadAmbiguous,
		HeadGuarded:      v.HeadGuarded,
		HeadOrphanRises:  v.HeadOrphanRises,
		PerfectYes:       v.PerfectYes,
		PerfectUnknown:   v.PerfectUnknown,
		PerfectAmbiguous: v.PerfectAmbiguous,
		PerfectOrphans:   v.PerfectOrphans,
	}
}

func toStanceControlCoverage(v replay.StanceControlCoverage) replaydoc.StanceControlCoverage {
	return replaydoc.StanceControlCoverage{
		Holds:      v.Holds,
		WithHole:   v.WithHole,
		InnerHoles: v.InnerHoles,
		Entries:    v.Entries,
		Published:  v.Published,
		Intervals:  v.Intervals,
		NoTrack:    v.NoTrack,
		NoPlayer:   v.NoPlayer,
	}
}
