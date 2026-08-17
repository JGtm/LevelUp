package replay

// ground_weapon_bounds.go — BORNER puis DATER la disparition d'une arme au sol.
//
// CE QUE LA REGLE DIT, ET D'OU ELLE VIENT. Le film ne porte AUCUN evenement type pick-up
// (mesure du 2026-08-12) et le record DEL n'est pas isolable (78 090 candidats pour 477 vies,
// correctif de revue des poses). La disparition se lit donc en DEUX temps, et les deux sont des
// mesures distinctes qu'il ne faut pas confondre :
//
//	BORNER — le recensement des images-cles (walker durci, 249/250 entites) PROUVE la survie
//	d'un objet : tant qu'un record `ti=42` de la vie (slot, gen) est recense, l'objet est la. La
//	derniere image-cle qui le recense et la premiere qui ne le recense plus encadrent la
//	disparition. Ce recensement est espace de ~20 s : il BORNE, il ne DATE pas. C'est CE couple
//	de bornes que l'artefact publie.
//
//	DATER — dans cet intervalle, le passage d'un joueur a moins de `originDropMaxDist` de la
//	position de l'objet. C'est le MIROIR de la regle du lacher (un objet nait la ou son porteur
//	meurt ; un objet disparait la ou un joueur passe), aux MEMES constantes. Cette date n'est PAS
//	publiee : elle sert au CYCLE, dont l'horloge repart au ramassage.
//
// Extrait de `ground_weapon_rules.go` a la phase 3 pour tenir le seuil de 500 lignes du depot ;
// meme lot, meme regle, memes seuils.
//
// PUR (aucune I/O).

import (
	"math"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// gwPickupBounds encadre la disparition d'un objet au sol : ce que le recensement des images-clés
// permet d'affirmer, et rien de plus.
type gwPickupBounds struct {
	// LowUS : dernier instant où l'objet est PROUVÉ présent (dernière image-clé qui le
	// recense, à défaut l'instant de sa création).
	LowUS uint64
	// HighUS : premier instant où son absence est PROUVÉE (première image-clé qui ne le
	// recense plus), à défaut la fin de sa vie de clé ou la fin du film.
	HighUS uint64
	// NeverPicked : l'objet est encore recensé à la DERNIÈRE image-clé du film. Il n'a pas été
	// ramassé — ou il l'a été après la dernière image-clé, ce que rien ne dit.
	NeverPicked bool
	// NoLaterKF : aucune image-clé ne suit l'instant de référence. La borne haute est alors la
	// fin du film, et l'intervalle n'est pas une mesure de recensement.
	NoLaterKF bool
	// SeenKF : nombre d'images-clés qui recensent l'objet. Zéro = ne PAS conclure qu'il n'a
	// jamais existé : il a pu naître et disparaître entre deux images-clés (24,0 % des
	// apparitions mesurées).
	SeenKF int
}

// WidthS rend la largeur de l'intervalle de bornage, en secondes.
func (b gwPickupBounds) WidthS() float64 {
	if b.HighUS <= b.LowUS {
		return 0
	}
	return float64(b.HighUS-b.LowUS) / 1e6
}

// gwPickupBoundsFrom encadre la disparition d'un objet né à `t0`, dont la clé (slot, gen) est
// reprise par une autre création à `lifeEnd` (ou `filmEnd` si elle ne l'est pas).
//
// `kfTimes` sont les instants de TOUTES les images-clés du film, triés ; `seen` ceux qui
// RECENSENT l'objet, triés, déjà restreints à [t0, lifeEnd).
//
// LA CLÉ EST REPRISE, ET C'EST LA DONNÉE : la génération ne fait que 2 bits, donc un même couple
// (slot, gen) porte plusieurs objets successifs au cours d'un match. Sans la borne `lifeEnd`, le
// recensement du SUIVANT prouverait la survie du PRÉCÉDENT.
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

// gwPickupNextAfter rend le premier instant de `sorted` strictement supérieur à `at`.
func gwPickupNextAfter(sorted []uint64, at uint64) (uint64, bool) {
	i := sort.Search(len(sorted), func(i int) bool { return sorted[i] > at })
	if i >= len(sorted) {
		return 0, false
	}
	return sorted[i], true
}

// gwPickupSeenWithin restreint le recensement d'une clé à la vie [t0, lifeEnd).
func gwPickupSeenWithin(seen []uint64, t0, lifeEnd uint64) []uint64 {
	lo := sort.Search(len(seen), func(i int) bool { return seen[i] >= t0 })
	hi := sort.Search(len(seen), func(i int) bool { return seen[i] >= lifeEnd })
	if hi < lo {
		hi = lo
	}
	return seen[lo:hi]
}

// gwPickupHit est un passage de joueur près d'un objet : qui, quand, à quelle distance.
type gwPickupHit struct {
	Slot  uint32
	TUS   uint64
	DistM float64
	Found bool
}

// gwPickupNearestPass rend le PREMIER passage d'un bipède à moins d'`originDropMaxDist` de `pos`
// dans la fenêtre [lowUS, highUS]. `samples` doit être TRIÉ par instant.
//
// « LE PREMIER », PAS « LE PLUS PROCHE » : le plan tranche l'ambiguïté dans ce sens (phase 2.1
// amendée — « si plusieurs : le premier »). À instant égal, le plus proche l'emporte, puis le
// slot le plus bas : sans ces deux départages, deux exécutions rendraient deux ramasseurs.
//
// LE SEUIL N'EST PAS UN PARAMÈTRE, même raison que le rayon de grappe : 1,5 m est la constante
// de production de la règle du lâcher (`originDropMaxDist`), et cette règle-ci en est le MIROIR
// — un objet naît là où son porteur meurt, il disparaît là où un joueur passe. Les deux doivent
// bouger ensemble ou pas du tout.
//
// LA DISTANCE RENDUE N'EST PAS UNE MESURE (découverte 9 du plan) : c'est celle à laquelle le
// seuil est franchi, et elle vaut mécaniquement 1,46 à 1,50 m sur tous les sous-ensembles. Elle
// ne se publie pas.
func gwPickupNearestPass(
	pos [3]float32, lowUS, highUS uint64, samples []filmdec.BipedPosition,
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
		if d >= originDropMaxDist {
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

// gwPickupRefPos rend la position où l'objet SE TROUVE quand on le ramasse : le dernier point de
// sa piste delta s'il a bougé dans sa vie, sa position de création sinon.
func gwPickupRefPos(
	c filmdec.EquipmentCreation, lifeEnd uint64, tracks []filmdec.ProjectileTrack,
) ([3]float32, bool) {
	best, bestGap := -1, uint64(math.MaxUint64)
	for i, tr := range tracks {
		if len(tr.Pts) == 0 {
			continue
		}
		t0 := tr.Pts[0].TimestampUS
		if t0 >= lifeEnd || t0+gwPickupTrackTolUS < c.TimestampUS {
			continue
		}
		if g := equipTimeGap(t0, c.TimestampUS); g < bestGap {
			best, bestGap = i, g
		}
	}
	if best < 0 {
		return [3]float32{c.X, c.Y, c.Z}, false
	}
	p := tracks[best].Pts[len(tracks[best].Pts)-1]
	return [3]float32{p.X, p.Y, p.Z}, true
}
