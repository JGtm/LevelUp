package replay

// ground_weapon_pickup_rule_test.go — LA REGLE DE RAMASSAGE, isolee de tout film pour qu'elle
// soit TESTEE et non seulement executee (meme parti que `ground_weapon_pads_cluster_test.go`).
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

// gwPickupBounds encadre la disparition d'un objet au sol : ce que le recensement des
// images-cles permet d'affirmer, et rien de plus.
type gwPickupBounds struct {
	// LowUS : dernier instant ou l'objet est PROUVE present (derniere image-cle qui le
	// recense, a defaut l'instant de sa creation).
	LowUS uint64
	// HighUS : premier instant ou son absence est PROUVEE (premiere image-cle qui ne le
	// recense plus), a defaut la fin de sa vie de cle ou la fin du film.
	HighUS uint64
	// NeverPicked : l'objet est encore recense a la DERNIERE image-cle du film. Il n'a pas
	// ete ramasse — ou il l'a ete apres la derniere image-cle, ce que rien ne dit.
	NeverPicked bool
	// NoLaterKF : aucune image-cle ne suit l'instant de reference. La borne haute est alors la
	// fin du film, et l'intervalle n'est pas une mesure de recensement.
	NoLaterKF bool
	// SeenKF : nombre d'images-cles qui recensent l'objet. Zero = ne PAS conclure qu'il n'a
	// jamais existe : il a pu naitre et disparaitre entre deux images-cles.
	SeenKF int
}

// WidthS rend la largeur de l'intervalle de bornage, en secondes.
func (b gwPickupBounds) WidthS() float64 {
	if b.HighUS <= b.LowUS {
		return 0
	}
	return float64(b.HighUS-b.LowUS) / 1e6
}

// gwPickupBoundsFrom encadre la disparition d'un objet ne a `t0`, dont la cle (slot, gen) est
// reprise par une autre creation a `lifeEnd` (ou `filmEnd` si elle ne l'est pas).
//
// `kfTimes` sont les instants de TOUTES les images-cles du film, tries ; `seen` ceux qui
// RECENSENT l'objet, tries, deja restreints a [t0, lifeEnd).
//
// LA CLE EST REPRISE, ET C'EST LA DONNEE : la generation ne fait que 2 bits, donc un meme
// couple (slot, gen) porte plusieurs objets successifs au cours d'un match. Sans la borne
// `lifeEnd`, le recensement du SUIVANT prouverait la survie du PRECEDENT.
func gwPickupBoundsFrom(t0, lifeEnd, filmEnd uint64, kfTimes, seen []uint64) gwPickupBounds {
	b := gwPickupBounds{LowUS: t0, HighUS: filmEnd, SeenKF: len(seen)}
	if len(seen) > 0 {
		b.LowUS = seen[len(seen)-1]
	}
	if len(kfTimes) > 0 && len(seen) > 0 && seen[len(seen)-1] == kfTimes[len(kfTimes)-1] {
		b.NeverPicked = true
		return b
	}
	next, ok := gwPickupNextAfter(kfTimes, b.LowUS)
	switch {
	case ok:
		b.HighUS = next
	default:
		b.NoLaterKF = true
	}
	if lifeEnd < b.HighUS {
		b.HighUS = lifeEnd
	}
	if b.HighUS < b.LowUS {
		b.HighUS = b.LowUS
	}
	return b
}

// gwPickupNextAfter rend le premier instant de `sorted` strictement superieur a `at`.
func gwPickupNextAfter(sorted []uint64, at uint64) (uint64, bool) {
	i := sort.Search(len(sorted), func(i int) bool { return sorted[i] > at })
	if i >= len(sorted) {
		return 0, false
	}
	return sorted[i], true
}

// gwPickupHit est un passage de joueur pres d'un objet : qui, quand, a quelle distance.
type gwPickupHit struct {
	Slot  uint32
	TUS   uint64
	DistM float64
	Found bool
}

// gwPickupNearestPass rend le PREMIER passage d'un bipede a moins de `maxDistM` de `pos` dans
// la fenetre [lowUS, highUS]. `samples` doit etre TRIE par instant.
//
// « LE PREMIER », PAS « LE PLUS PROCHE » : le plan tranche l'ambiguite dans ce sens (phase 2.1
// amendee — « si plusieurs : le premier »). A instant egal, le plus proche l'emporte, puis le
// slot le plus bas : sans ces deux departages, deux executions rendraient deux ramasseurs.
func gwPickupNearestPass(
	pos [3]float32, lowUS, highUS uint64, samples []filmdec.BipedPosition, maxDistM float64,
) gwPickupHit {
	var out gwPickupHit
	i := sort.Search(len(samples), func(i int) bool { return samples[i].TimestampUS >= lowUS })
	for ; i < len(samples); i++ {
		p := samples[i]
		if p.TimestampUS > highUS {
			break
		}
		if !p.HasWorld {
			continue
		}
		d := gwPadsDist(pos[0], pos[1], pos[2], p.X, p.Y, p.Z)
		if d >= maxDistM {
			continue
		}
		if !out.Found || p.TimestampUS < out.TUS ||
			(p.TimestampUS == out.TUS && (d < out.DistM || (d == out.DistM && p.Slot < out.Slot))) {
			out = gwPickupHit{Slot: p.Slot, TUS: p.TimestampUS, DistM: d, Found: true}
		}
		if out.Found && p.TimestampUS > out.TUS {
			break
		}
	}
	return out
}

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
			if gwPickupTimeGap(pts[j].TimestampUS, atUS) > gwPickupWitnessTolUS {
				continue
			}
			if d := gwPadsDist(pos[0], pos[1], pos[2], pts[j].X, pts[j].Y, pts[j].Z); d < best {
				best, found = d, true
			}
		}
	}
	return best, found
}

// gwPickupTimeGap rend l'ecart absolu entre deux instants (les uint64 ne se soustraient pas
// naivement).
func gwPickupTimeGap(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
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
	got := gwPickupNearestPass(pos, 0, 400, s, originDropMaxDist)
	if !got.Found || got.Slot != 3 || got.TUS != 200 {
		t.Fatalf("le PREMIER passage sous 1,5 m (slot 3 a t=200) attendu : %+v", got)
	}
	if hors := gwPickupNearestPass(pos, 250, 400, s, originDropMaxDist); hors.Slot != 5 {
		t.Fatalf("hors fenetre, le passage suivant devient le premier : %+v", hors)
	}
	if vide := gwPickupNearestPass(pos, 0, 150, s, originDropMaxDist); vide.Found {
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
