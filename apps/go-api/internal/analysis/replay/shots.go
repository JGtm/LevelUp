package replay

// Rattachement des ÉVÉNEMENTS DE TIR (record film type 105) aux trajectoires du rejeu.
//
// L'ÉVÉNEMENT PORTE DÉJÀ SON AUTEUR. Le champ FilmIndex est écrit dans le film ; il n'est ni
// deviné ni voté. Ce fichier ne cherche donc pas QUI a tiré — il cherche OÙ ce joueur se
// trouvait, pour poser le tir sur la carte.
//
// POURQUOI CE N'EST PAS IMMÉDIAT : les positions sont indexées par SLOT de biped, et un slot
// change à chaque réapparition. Rien dans l'événement de tir ne donne le slot. Le pont
// slot -> joueur est construit ailleurs (owners.go, à partir du fil des morts) ; ici on ne fait
// que l'appliquer, en comptant tout ce qu'on écarte.
//
// LE VOTE PAR ÉGALITÉ DES CAPS DE VISÉE A ÉTÉ SUPPRIMÉ le 2026-07-28. Il désignait le tireur en
// comparant la direction du tir aux directions de visée des bipeds — une inférence, lisible dans
// ~44 % des records seulement, et qui contredisait la lecture sur 4 slots. Mesure du retrait :
// 496 -> 475 tirs publiés, et 4 -> 0 désaccords.

import (
	"math"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// shotPosToleranceUS : écart temporel maximal entre l'événement et l'échantillon de position
// utilisé (les records de position arrivent à ~60 Hz).
const shotPosToleranceUS = 120_000

// slotTrack est l'index temporel des positions d'un slot, pour la recherche par instant.
type slotTrack struct {
	pts []filmdec.BipedPosition // triées par TimestampUS
}

// at renvoie l'échantillon le plus proche de tUS et son écart.
func (s slotTrack) at(tUS uint64) (filmdec.BipedPosition, uint64) {
	i := sort.Search(len(s.pts), func(i int) bool { return s.pts[i].TimestampUS >= tUS })
	best, bestD := filmdec.BipedPosition{}, uint64(math.MaxUint64)
	for _, j := range []int{i - 1, i} {
		if j < 0 || j >= len(s.pts) {
			continue
		}
		d := absDiffU64(s.pts[j].TimestampUS, tUS)
		if d < bestD {
			bestD, best = d, s.pts[j]
		}
	}
	return best, bestD
}

// orphanShot est un événement de tir que la porte du BIPÈDE a écarté, avec la cause du rejet.
//
// IL EXISTE PARCE QUE LE REJET N'EST PAS LA FIN DE L'HISTOIRE. Un joueur EMBARQUÉ cesse de
// répliquer la position de son bipède (primitive V1a.4) : ses tirs sortent d'ici sous
// `reasonNoSlot`, non parce que le film ignorerait qui a tiré — l'événement porte son tireur —
// mais parce qu'il n'y a AUCUNE position de bipède à poser sur la carte. La seconde porte, celle
// du VÉHICULE (`vehicle_shots.go`), reprend exactement cette liste et lui donne la position du
// véhicule occupé. LA CAUSE VOYAGE AVEC L'ÉVÉNEMENT : sans elle, la couverture ne saurait pas
// quel compteur décrémenter quand la seconde porte rattache.
type orphanShot struct {
	ev     filmdec.FireEvent
	reason rejectReason
}

// buildShots rattache les events de tir aux slots et produit les tirs publiables, AVEC la
// couverture : combien rattachés sur combien disponibles, et pourquoi le reste est écarté.
//
// AUCUN CHEMIN DE REJET N'EST MUET ICI. C'est l'invariant que le test vérifie :
// rattachés + rejetés = disponibles, exactement.
//
// LE SECOND RETOUR est la liste des ORPHELINS reprenables par la porte du véhicule : les rejets
// « sans slot » et « hors fenêtre », ceux dont la signature est celle d'un tireur embarqué. Les
// AMBIGUS n'y sont PAS, et c'est une mesure : l'ambiguïté naît de DEUX slots du même joueur qui
// répliquent tous deux une position à cet instant — donc d'un joueur qui n'est PAS embarqué.
func buildShots(pos []filmdec.BipedPosition, events []filmdec.FireEvent, origin, step uint64,
	owner map[uint32]int) ([]Shot, []orphanShot, LayerCoverage) {
	cov := LayerCoverage{Available: len(events)}
	if len(pos) == 0 || len(events) == 0 || len(owner) == 0 {
		cov.NoSlot = len(events) // rien pour rattacher : tout est écarté, et c'est dit
		return nil, nil, cov
	}
	tracks := indexBySlot(pos)
	var out []Shot
	var orphans []orphanShot
	for _, e := range events {
		slot, reason := slotFor(tracks, owner, e.FilmIndex, e.TimestampUS)
		if reason != reasonAttached {
			cov.count(reason)
			if reason == reasonNoSlot {
				orphans = append(orphans, orphanShot{ev: e, reason: reason})
			}
			continue
		}
		p, d := tracks[slot].at(e.TimestampUS)
		if d > shotPosToleranceUS || !p.HasWorld {
			cov.count(reasonOutOfWindow)
			orphans = append(orphans, orphanShot{ev: e, reason: reasonOutOfWindow})
			continue
		}
		cov.count(reasonAttached)
		s := Shot{
			T:    int((e.TimestampUS - origin) / step),
			Slot: slot,
			X:    round2(p.X),
			Y:    round2(p.Y),
		}
		if h, ok := e.AimHeadingDeg(); ok {
			s.H = headingForJSON(float32(h))
		}
		if e.WeaponID != 0 {
			s.Weapon = formatWeaponID(e.WeaponID)
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].T < out[j].T })
	return out, orphans, cov
}

// indexBySlot regroupe les positions par slot, triées par instant.
func indexBySlot(pos []filmdec.BipedPosition) map[uint32]slotTrack {
	m := map[uint32][]filmdec.BipedPosition{}
	for _, p := range pos {
		m[p.Slot] = append(m[p.Slot], p)
	}
	out := make(map[uint32]slotTrack, len(m))
	for s, ps := range m {
		sort.Slice(ps, func(i, j int) bool { return ps[i].TimestampUS < ps[j].TimestampUS })
		out[s] = slotTrack{pts: ps}
	}
	return out
}

// La porte de rattachement vit désormais dans coverage.go sous le nom `slotFor`, et elle
// REND SA CAUSE au lieu d'un booléen. L'ancienne `uniqueSlotFor` est supprimée : elle ne
// distinguait pas « slot introuvable » de « slot ambigu », qui désignent pourtant deux
// chantiers opposés, et ses appelants faisaient `continue` sans rien compter.

func absDiffU64(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

// formatWeaponID rend l'identifiant 64 bits en hexadécimal : un entier 64 bits ne survit
// pas au `number` de JavaScript (53 bits de mantisse).
func formatWeaponID(id uint64) string {
	const hexDigits = "0123456789ABCDEF"
	buf := make([]byte, 16)
	for i := 15; i >= 0; i-- {
		buf[i] = hexDigits[id&0xF]
		id >>= 4
	}
	return "0x" + string(buf)
}
