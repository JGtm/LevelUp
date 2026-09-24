package replayview

// convert_coverage_armes.go — LA PROJECTION DES COUVERTURES DES ARMES A L INSTANT (schema 69,
// campagne « retours rejeu », lot M3) : la sante de la marche d image-cle et les dotations de
// naissance. Meme regle que les autres convertisseurs : champ par champ, rien de derive.

import (
	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

func toKeyframeCoverage(v replay.KeyframeCoverage) replaydoc.KeyframeCoverage {
	return replaydoc.KeyframeCoverage{
		Keyframes:          v.Keyframes,
		Records:            v.Records,
		Bipeds:             v.Bipeds,
		Neighbors:          v.Neighbors,
		Jumps:              v.Jumps,
		Resyncs:            v.Resyncs,
		Elections:          v.Elections,
		Slides:             v.Slides,
		FramedAbsentBipeds: v.FramedAbsentBipeds,
	}
}

func toBirthLoadoutCoverage(v replay.BirthLoadoutCoverage) replaydoc.BirthLoadoutCoverage {
	return replaydoc.BirthLoadoutCoverage{
		Creations:         v.Creations,
		Closed:            v.Closed,
		Read:              v.Read,
		Desync:            v.Desync,
		Overflow:          v.Overflow,
		Unconfirmed:       v.Unconfirmed,
		NoWeaponComponent: v.NoWeaponComponent,
		Published:         v.Published,
		Snapped:           v.Snapped,
		NoLife:            v.NoLife,
		BeforeOrigin:      v.BeforeOrigin,
		NonWeapon:         v.NonWeapon,
		NoDisplayable:     v.NoDisplayable,
	}
}
