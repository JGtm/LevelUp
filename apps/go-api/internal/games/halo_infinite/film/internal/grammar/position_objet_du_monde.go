package grammar

// position_objet_du_monde.go — LE LECTEUR UNIQUE DE LA POSITION D UN OBJET DU MONDE
// (`object-position-component`, chemin dominant) : celui des pistes et celui des objets que
// l equipement porte engendre.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// decodeWorldObjectPos lit et déquantifie `object-position-component` sur le chemin dominant.
//
// REJET DES QUANTA SATURÉS : un axe à 0 ou à 2^w-1 est écarté. Sans cette règle, une vie sur
// soixante-dix finit au plancher du BSP (z = -84 m) — un quantum saturé n'est pas une position,
// c'est une valeur de garde.
//
// REJET D'UNE AUTRE RÉGION : l'index de région lu doit valoir celui que la carte joue
// (`WorldObjectPrecision.Region`, installé depuis le catalogue). Un record d'une autre région
// exprime ses quanta dans une AUTRE AABB : le déquantifier avec ces bornes rendrait une
// position fausse SILENCIEUSE, ce qui est pire qu'un refus. Même règle que le jumeau bipède
// `decodeBipedI0Pos`.
func decodeWorldObjectPos(pay []byte, at int, wr *profile.Vec3Range, lg profile.PrecisionDescriptor) ([3]float32, bool) {
	q, ok := lireQuantaObjetDuMonde(pay, at, lg)
	if !ok {
		return [3]float32{}, false
	}
	return dequantObjetDuMonde(q, wr, lg), true
}

// lireQuantaObjetDuMonde lit les trois quanta d `object-position-component` sur le chemin dominant,
// avec les deux rejets de [decodeWorldObjectPos] (autre region, quantum sature).
func lireQuantaObjetDuMonde(pay []byte, at int, lg profile.PrecisionDescriptor) ([3]uint64, bool) {
	var q [3]uint64
	if source.BitsTolerants(pay, at, 2) != 0 { // precHigh et index-sel nuls = chemin dominant
		return q, false
	}
	idxW := int(lg.IndexW)
	if uint32(source.BitsTolerants(pay, at+2, idxW)) != lg.Region {
		return q, false
	}
	off := at + 2 + idxW
	for a := range 3 {
		w := lg.AxisW[a]
		q[a] = source.BitsTolerants(pay, off, int(w))
		if q[a] == 0 || q[a] == (uint64(1)<<w)-1 {
			return q, false
		}
		off += int(w)
	}
	return q, true
}

// dequantObjetDuMonde dequantifie trois quanta lus par [lireQuantaObjetDuMonde] aux bornes `wr`.
func dequantObjetDuMonde(q [3]uint64, wr *profile.Vec3Range, lg profile.PrecisionDescriptor) [3]float32 {
	var v [3]float32
	for a := range 3 {
		w := lg.AxisW[a]
		lo, hi := wr[a].Min, wr[a].Max
		v[a] = lo + (float32(q[a])+0.5)*(hi-lo)/float32(uint64(1)<<w)
	}
	return v
}
