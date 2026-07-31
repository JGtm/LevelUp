package filmdec

import "math"

// Garde-fous contre les FAUX POSITIFS du balayage bit à bit : la grammaire du record biped
// est fortement contrainte, mais sur des millions de positions de bit un motif conforme
// finit par apparaître dans du bruit. Deux signatures les distinguent d'une vraie
// trajectoire, et elles se voient dans les données de 000d5950 :
//
//   - ISOLEMENT TEMPOREL — un biped vivant émet en continu (~60 Hz) ; les 7 aberrations
//     observées sont des points uniques séparés de 66 à 320 s du reste de leur slot.
//   - TÉLÉPORTATION — un saut instantané de plusieurs dizaines de mètres à l'intérieur
//     d'une vie.
//
// Sans eux, l'enveloppe des trajectoires double (X[-37,5;49,1] au lieu de X[-7,0;43,8]).

// DefaultIsolationGapMS : au-delà de cet écart avec le voisin le PLUS PROCHE du même slot
// (avant ET après), un échantillon n'appartient pas à une vie. Très large devant les
// silences légitimes d'un joueur immobile (la compression delta peut suspendre l'émission
// quelques secondes), très petit devant les 66 s minimum observées sur les aberrations.
const DefaultIsolationGapMS = 15000

// DropIsolated écarte les échantillons temporellement isolés au sein de leur slot : ceux
// dont l'écart au voisin le plus proche (précédent ou suivant) dépasse gapMS. Un slot
// réduit à un seul échantillon disparaît (ce n'est pas une trajectoire). gapMS <= 0
// désactive le filtre. PUR. L'entrée doit être chronologique par slot.
func DropIsolated(pos []BipedPosition, gapMS int) []BipedPosition {
	if gapMS <= 0 || len(pos) == 0 {
		return pos
	}
	gap := uint64(gapMS) * 1000
	bySlot := map[uint32][]int{}
	for i, p := range pos {
		bySlot[p.Slot] = append(bySlot[p.Slot], i)
	}
	keep := make([]bool, len(pos))
	for _, idx := range bySlot {
		for k, i := range idx {
			if nearestGap(pos, idx, k) <= gap {
				keep[i] = true
			}
		}
	}
	out := pos[:0:0]
	for i, p := range pos {
		if keep[i] {
			out = append(out, p)
		}
	}
	return out
}

// nearestGap renvoie l'écart temporel au voisin le plus proche dans la séquence du slot
// (math.MaxUint64 si l'échantillon n'a aucun voisin).
func nearestGap(pos []BipedPosition, idx []int, k int) uint64 {
	best := uint64(math.MaxUint64)
	ts := pos[idx[k]].TimestampUS
	if k > 0 {
		best = ts - pos[idx[k-1]].TimestampUS
	}
	if k+1 < len(idx) {
		if d := pos[idx[k+1]].TimestampUS - ts; d < best {
			best = d
		}
	}
	return best
}

// DropTeleports écarte, slot par slot et dans l'ordre chronologique, les positions dont la
// vitesse depuis la dernière position acceptée dépasse maxSpeed (m/s). maxSpeed <= 0
// désactive le filtre. PUR.
func DropTeleports(pos []BipedPosition, maxSpeed float64) []BipedPosition {
	if maxSpeed <= 0 || len(pos) == 0 {
		return pos
	}
	type anchor struct {
		p      BipedPosition
		ok     bool
		streak int
	}
	anchors := map[uint32]*anchor{}
	out := pos[:0:0]
	for _, p := range pos {
		a := anchors[p.Slot]
		if a == nil {
			a = &anchor{}
			anchors[p.Slot] = a
		}
		if a.ok && a.streak < maxRejectStreak && speedFrom(a.p, p) > maxSpeed {
			a.streak++
			continue
		}
		a.p, a.ok, a.streak = p, true, 0
		out = append(out, p)
	}
	return out
}

// speedFrom renvoie la vitesse (m/s) entre deux positions ; +Inf si elles partagent le même
// horodatage avec des coordonnées différentes (donc rejetable), 0 si elles sont identiques.
func speedFrom(a, b BipedPosition) float64 {
	dx, dy, dz := float64(b.X-a.X), float64(b.Y-a.Y), float64(b.Z-a.Z)
	dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if b.TimestampUS <= a.TimestampUS {
		if dist == 0 {
			return 0
		}
		return math.Inf(1)
	}
	return dist / (float64(b.TimestampUS-a.TimestampUS) / 1e6)
}
