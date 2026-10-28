package replay

// document_equipement_porte_rangs.go — LE RANG DE CHAQUE OBJET PORTE : l annonce i48 du porteur dans le paquet de
// la creation, et les index par paquet que l assemblage de l equipement porte lit
// (`document_equipement_porte.go`). DEPLACEMENT PUR (plan des durees d effets, B.2) pour tenir le seuil de
// 500 lignes : aucune ligne de logique n a change.

import "levelup/go-api/internal/games/halo_infinite/film/types"

// indexDesRangs : les annonces i48 par paquet et porteur, les créations par paquet et porteur, et les
// pistes publiées.
type indexDesRangs struct {
	rangs     map[paquetDuPorteur]int
	nouveaux  map[paquetDuPorteur]int
	published map[uint32]bool
}

// rangDe rend le rang de l'objet créé par `creation` : celui de l'annonce i48 de son porteur dans le
// même paquet, quand l'objet y est seul et que la palette du film le nomme. Un écart se compte.
func (x indexDesRangs) rangDe(creation types.CarriedEquipmentRead, origin uint64, palette *AbilityPalette,
	cov *CarriedEquipmentCoverage) (int, bool) {
	cle := paquetDuPorteur{creation.TimestampUS, creation.Carrier}
	rang, annonce := x.rangs[cle]
	switch {
	case creation.TimestampUS < origin:
		cov.BeforeOrigin++
	case !x.published[creation.Carrier]:
		cov.NoIdentity++
	case !annonce:
		cov.WithoutRank++
	case x.nouveaux[cle] > 1:
		cov.AmbiguousRank++
	case palette.FamilyOf(rang) == "":
		cov.Unnamed++
	default:
		return rang, true
	}
	return 0, false
}

// frame rend la frame d'un instant du film.
func (in entreeDEquipementPorte) frame(ts uint64) int {
	return int((ts - in.origin) / in.step) //nolint:gosec // instants posterieurs a l origine
}

// paquetDuPorteur : un paquet (son instant) et un porteur.
type paquetDuPorteur struct {
	ts   uint64
	slot uint32
}

// rangsParPaquet indexe le rang des annonces i48 qui donnent un équipement (prise ou réapparition).
func rangsParPaquet(changes []types.EquipmentChange) map[paquetDuPorteur]int {
	out := map[paquetDuPorteur]int{}
	for _, c := range changes {
		if c.Kind == types.EquipmentTaken || c.Kind == types.EquipmentSpawned {
			out[paquetDuPorteur{c.TimestampUS, c.Slot}] = c.Rank
		}
	}
	return out
}

// nouveauxParPaquet compte les objets créés attachés par paquet et par porteur.
func nouveauxParPaquet(reads []types.CarriedEquipmentRead) map[paquetDuPorteur]int {
	out := map[paquetDuPorteur]int{}
	for _, l := range reads {
		if l.Kind == types.CarriedNew {
			out[paquetDuPorteur{l.TimestampUS, l.Carrier}]++
		}
	}
	return out
}
