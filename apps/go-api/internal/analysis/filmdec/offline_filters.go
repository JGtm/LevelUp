package filmdec

import (
	"math"
	"sort"
)

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
	return DropTeleportsExcept(pos, maxSpeed, nil)
}

// translocExemptToleranceUS : demi-fenêtre de l'exemption autour d'un événement 117 —
// mesurée, pas choisie : les 51 rejets à tort des 18 téléportations du corpus tombent TOUS
// à ±200 ms de leur événement (rapport R3 §3), et les fenêtres de ±200 ms n'attrapent
// aucune fausse exemption (0/51 hors corroboration, R3 §5 option A).
const translocExemptToleranceUS = 200_000

// TeleportExemptions porte, par slot, les instants (horloge des paquets) où le filtre de
// vitesse est LEVÉ : les événements 117 du même slot, à ±translocExemptToleranceUS.
//
// C'est la décision D2 du PLAN_LECTURE_FIABLE_EQUIPEMENT_2026-09-03 (option A du rapport
// R3) : une téléportation réelle part à 193-1540 m/s et le filtre à 100 m/s la rejetait à
// tort — 1 à 3 échantillons bruts par saut, jamais l'arrivée (le réancrage borne la
// cascade). L'exemption est déclenchée par un ENREGISTREMENT du film, jamais par un seuil
// spatial : sur un film sans tête 117, elle n'existe pas et le filtre rend bit à bit la
// sémantique du schéma 37 — invariance prouvée contre une implémentation de RÉFÉRENCE
// FIGÉE dans le test (copie verbatim de l'ancien DropTeleports :
// TestDropTeleportsInvarianceSansEvenement, et TestP1InvarianceSansTete117 sur film
// témoin avec le même oracle — comparer à DropTeleports ne prouverait rien, il délègue).
type TeleportExemptions map[uint32][]uint64

// TeleportExemptionsOf construit les fenêtres d'exemption depuis les événements du scan
// (ScanFilmTranslocatorTeleports rend les instants triés ; la construction re-trie par slot
// pour ne pas dépendre de ce contrat).
func TeleportExemptionsOf(evts []TranslocatorTeleport) TeleportExemptions {
	if len(evts) == 0 {
		return nil
	}
	out := TeleportExemptions{}
	for _, e := range evts {
		out[e.Slot] = append(out[e.Slot], e.TimestampUS)
	}
	for _, ts := range out {
		sort.Slice(ts, func(i, j int) bool { return ts[i] < ts[j] })
	}
	return out
}

// covers dit si l'instant tombe à ±translocExemptToleranceUS d'un événement du slot.
func (x TeleportExemptions) covers(slot uint32, tsUS uint64) bool {
	ts := x[slot]
	i := sort.Search(len(ts), func(i int) bool { return ts[i]+translocExemptToleranceUS >= tsUS })
	return i < len(ts) && ts[i] <= tsUS+translocExemptToleranceUS
}

// DropTeleportsExcept est DropTeleports avec l'exemption D2 : une position couverte par une
// fenêtre d'exemption de SON slot est acceptée sans condition de vitesse — et devient
// l'ancre des décisions suivantes, comme toute position acceptée. exempt nil ou vide rend
// le filtre STRICTEMENT identique à DropTeleports. PUR.
func DropTeleportsExcept(pos []BipedPosition, maxSpeed float64, exempt TeleportExemptions) []BipedPosition {
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
		if a.ok && a.streak < maxRejectStreak && speedFrom(a.p, p) > maxSpeed &&
			(len(exempt) == 0 || !exempt.covers(p.Slot, p.TimestampUS)) {
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
