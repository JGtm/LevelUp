package replay

// ground_weapon_pickup_rule_test.go — LES TESTS DE LA REGLE DE RAMASSAGE, isolee de tout film
// pour qu'elle soit TESTEE et non seulement executee (meme parti que
// `ground_weapon_pads_cluster_test.go`).
//
// LA REGLE ELLE-MEME EST EN PRODUCTION depuis la phase 3 (`ground_weapon_rules.go` :
// `gwPickupBounds`, `gwPickupBoundsFrom`, `gwPickupNearestPass`, `gwPickupSeenWithin`,
// `gwPickupRefPos`) : c'est elle qui borne les intervalles publies par l'artefact. Ce fichier
// garde les TESTS et le seul temoin qui n'a pas de role en production (`gwPickupNearestAt`).
//
// CE QUE LA REGLE DIT, ET D'OU ELLE VIENT. Le film ne porte AUCUN evenement type pick-up
// (mesure du 2026-08-12) et le record DEL n'est pas isolable (78 090 candidats pour 477 vies,
// correctif de revue des poses). Le ramassage se lit donc en DEUX temps, et les deux sont des
// mesures distinctes qu'il ne faut pas confondre :
//
//	BORNER — le recensement des images-cles (walker durci, 249/250 entites) PROUVE la survie
//	d'un objet : tant qu'un record `ti=42` de la vie (slot, gen) est recense, l'objet est la.
//	La derniere image-cle qui le recense et la premiere qui ne le recense plus encadrent la
//	disparition. Ce recensement est espace de ~20 s : il BORNE, il ne DATE pas.
//
//	DATER — dans cet intervalle, le passage d'un joueur a moins de `originDropMaxDist` de la
//	position de l'objet. C'est le MIROIR de la regle `dropped` du 2026-08-18 (un objet nait la
//	ou son porteur meurt ; un objet disparait la ou un joueur passe), aux MEMES constantes de
//	production — pas des jumelles.
//
// LES SEUILS SONT CEUX DU PLAN, ECRITS AVANT LA MESURE (decision 4, phase 2.1 amendee) : 1,5 m
// pour le passage, « le premier » quand plusieurs joueurs passent, `unknown` et date = borne
// haute quand aucun ne passe.

import (
	"math"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// gwPickupWitnessTolUS est la tolerance temporelle du TEMOIN d'instant : de combien on accepte
// de s'ecarter de l'instant tire au sort pour trouver la position d'un joueur. 100 ms, soit la
// moitie de `originDropWindowUS` et environ six pas de replication (~16 ms) : assez large pour
// que tout joueur vivant ait un echantillon, assez etroit pour qu'il soit encore la ou il etait.
const gwPickupWitnessTolUS = 100_000

// gwPickupWitnessSeed fige le tirage des temoins. Un temoin qui change d'une execution a
// l'autre n'est pas un temoin : c'est un bruit qu'on relance jusqu'a ce qu'il arrange.
const gwPickupWitnessSeed = 20260817

// gwPickupNearestAt rend la distance du joueur le PLUS PROCHE de `pos` a l'instant `atUS` —
// le TEMOIN de l'item 2.1. Chaque slot est represente par son echantillon le plus proche dans
// le temps, a `gwPickupWitnessTolUS` pres ; un slot sans echantillon dans cette fenetre est un
// joueur mort ou non replique, et il ne compte pas.
func gwPickupNearestAt(
	pos [3]float32, atUS uint64, bySlot map[uint32][]filmdec.BipedPosition,
) (float64, bool) {
	best, found := math.Inf(1), false
	for _, pts := range bySlot {
		i := sort.Search(len(pts), func(i int) bool { return pts[i].TimestampUS >= atUS })
		for _, j := range []int{i - 1, i} {
			if j < 0 || j >= len(pts) || !pts[j].HasWorld {
				continue
			}
			if equipTimeGap(pts[j].TimestampUS, atUS) > gwPickupWitnessTolUS {
				continue
			}
			if d := gwPadsDist(pos[0], pos[1], pos[2], pts[j].X, pts[j].Y, pts[j].Z); d < best {
				best, found = d, true
			}
		}
	}
	return best, found
}

// --- TESTS DE LA REGLE (sans garde : ils tournent avec le paquet) ------------------------

// TestGwPickupBornesParLeRecensement : les trois issues du bornage — recense jusqu'a la fin
// (jamais ramasse), recense puis absent (intervalle), jamais recense (borne des la creation).
func TestGwPickupBornesParLeRecensement(t *testing.T) {
	kf := []uint64{10, 30, 50, 70}
	const fin = 80

	jamais := gwPickupBoundsFrom(5, fin, fin, kf, []uint64{10, 30, 50, 70})
	if !jamais.NeverPicked || jamais.LowUS != 70 {
		t.Fatalf("objet recense a la derniere image-cle = jamais ramasse : %+v", jamais)
	}
	borne := gwPickupBoundsFrom(5, fin, fin, kf, []uint64{10, 30})
	if borne.NeverPicked || borne.LowUS != 30 || borne.HighUS != 50 {
		t.Fatalf("disparition entre 30 et 50 attendue : %+v", borne)
	}
	aucun := gwPickupBoundsFrom(35, fin, fin, kf, nil)
	if aucun.LowUS != 35 || aucun.HighUS != 50 || aucun.SeenKF != 0 {
		t.Fatalf("ne et disparu entre deux images-cles : borne [35,50] attendue : %+v", aucun)
	}
	tard := gwPickupBoundsFrom(75, fin, fin, kf, nil)
	if !tard.NoLaterKF || tard.HighUS != fin {
		t.Fatalf("creation apres la derniere image-cle : borne haute = fin du film : %+v", tard)
	}
}

// TestGwPickupBorneHauteSuitLaRepriseDeCle : quand la cle (slot, gen) est reprise par une autre
// creation avant l'image-cle suivante, c'est la REPRISE qui borne — sinon le recensement du
// suivant prouverait la survie du precedent.
func TestGwPickupBorneHauteSuitLaRepriseDeCle(t *testing.T) {
	got := gwPickupBoundsFrom(12, 40, 80, []uint64{10, 30, 50, 70}, []uint64{30})
	if got.HighUS != 40 {
		t.Fatalf("borne haute = reprise de cle (40), pas l'image-cle suivante (50) : %+v", got)
	}
}

// TestGwPickupPremierPassageEtPasLePlusProche : la regle tranchee est « le premier », et le
// departage a instant egal est la distance puis le slot.
func TestGwPickupPremierPassageEtPasLePlusProche(t *testing.T) {
	pos := [3]float32{0, 0, 0}
	s := []filmdec.BipedPosition{
		{Slot: 7, TimestampUS: 100, X: 9, HasWorld: true},
		{Slot: 3, TimestampUS: 200, X: 1.2, HasWorld: true},
		{Slot: 5, TimestampUS: 300, X: 0.1, HasWorld: true},
	}
	got := gwPickupNearestPass(pos, 0, 400, s)
	if !got.Found || got.Slot != 3 || got.TUS != 200 {
		t.Fatalf("le PREMIER passage sous 1,5 m (slot 3 a t=200) attendu : %+v", got)
	}
	if hors := gwPickupNearestPass(pos, 250, 400, s); hors.Slot != 5 {
		t.Fatalf("hors fenetre, le passage suivant devient le premier : %+v", hors)
	}
	if vide := gwPickupNearestPass(pos, 0, 150, s); vide.Found {
		t.Fatalf("aucun passage sous 1,5 m dans la fenetre : %+v", vide)
	}
}

// TestGwPickupTemoinIgnoreLesEchantillonsTropLoinDansLeTemps : le temoin d'instant ne doit pas
// aller chercher une position vieille de plusieurs secondes pour la faire passer pour actuelle.
func TestGwPickupTemoinIgnoreLesEchantillonsTropLoinDansLeTemps(t *testing.T) {
	bySlot := map[uint32][]filmdec.BipedPosition{
		1: {{Slot: 1, TimestampUS: 1_000_000, X: 0.5, HasWorld: true}},
		2: {{Slot: 2, TimestampUS: 5_000_000, X: 0.1, HasWorld: true}},
	}
	d, ok := gwPickupNearestAt([3]float32{0, 0, 0}, 1_050_000, bySlot)
	if !ok || math.Abs(d-0.5) > 1e-6 {
		t.Fatalf("seul le slot 1 est dans la tolerance : d=%.3f ok=%t", d, ok)
	}
	if _, ok := gwPickupNearestAt([3]float32{0, 0, 0}, 3_000_000, bySlot); ok {
		t.Fatalf("aucun echantillon a +-100 ms de 3 s : le temoin doit se taire")
	}
}
