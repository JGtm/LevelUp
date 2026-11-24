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

// identified dit si une ligne du film nomme son joueur : depuis le schema 127 le decodeur publie aussi les lignes d'un bot,
// qui n'a pas d'identifiant et se designe par `place` (champ que l'API ne sert pas). Une ligne sans identifiant ne peut
// etre rangee sous aucun joueur : la servir avec un identifiant vide la ferait ranger sous la meme cle que toute autre
// ligne sans identifiant, donc sous le mauvais joueur. Elle est ecartee, comme les autres champs de bot sans lecteur
// (`testutil.ReplayChampsNonServis`).
func identified(xuid string) bool { return xuid != "" }

// sliceOfIdentified projette les lignes dont `xuid` nomme un joueur, la nullite de `in` preservee (cf. [sliceOf]).
func sliceOfIdentified[S, D any](in []S, xuid func(S) string, conv func(S) D) []D {
	if in == nil {
		return nil
	}
	out := make([]D, 0, len(in))
	for _, v := range in {
		if identified(xuid(v)) {
			out = append(out, conv(v))
		}
	}
	return out
}

func toElimination(v replay.Elimination) replaydoc.Elimination {
	return replaydoc.Elimination{T: v.T, XUID: v.XUID, Victim: v.Victim, Headshot: v.Headshot, Perfect: v.Perfect}
}

func toEliminationCoverage(replay.EliminationCoverage) replaydoc.EliminationCoverage {
	return replaydoc.EliminationCoverage{}
}

func toGesture(v replay.Gesture) replaydoc.Gesture {
	return replaydoc.Gesture{T: v.T, Place: v.Place, Kind: v.Kind}
}

func toGestureCoverage(v replay.GestureCoverage) replaydoc.GestureCoverage {
	return replaydoc.GestureCoverage{Gated: v.Gated, StoppedFrames: v.StoppedFrames, WithoutPlayer: v.WithoutPlayer}
}

func toStanceControlCoverage(v replay.StanceControlCoverage) replaydoc.StanceControlCoverage {
	return replaydoc.StanceControlCoverage{WithHole: v.WithHole, NoTrack: v.NoTrack, NoPlayer: v.NoPlayer}
}
