package filmdec

// navpoint_radial_segments.go — LES SEGMENTS DE L'ANNEAU (ti=12 i14), en production : la
// lecture « MECHE PAUSABLE » du 2026-09-01, portee ici depuis l'instrument de mesure.
//
// # POURQUOI UN SEGMENT PLUTOT QU'UNE MONTEE
//
// `NavpointContiguousRises` (navpoint_radial_rises.go) decoupe la serie d'un slot en suites
// NON DECROISSANTES : chacune finit donc a son maximum PAR CONSTRUCTION, et le cycle de
// RECHARGE du marqueur (130 -> 253 -> 127, ~5,0 s, joue apres chaque evenement terminal du
// site) y entre par sa moitie montante. Le filtre du quantum plein suffisait sur Neutral Bomb,
// pas ailleurs.
//
// Le SEGMENT est le fait brut : la suite maximale d'echantillons d'un meme slot separes d'au
// plus [NavpointRiseMaxGapMS], SANS exigence de monotonie. La forme du segment ENTIER dit
// alors ce qui s'est passe, et l'inspection du 2026-09-01
// (`navpoint_ti12_onebomb_test.go`, journal) en a MESURE trois :
//
//	ARMEMENT     un segment qui monte (131 -> 254) et FINIT A SON SOMMET : l'anneau reste
//	             plein, la bombe est armee. Dix occurrences identiques sur One Bomb (n=30,
//	             ~2,9 s), les memes sur Neutral Bomb et Husky Raid.
//	DESARMEMENT  une TENUE de defenseur : segment strictement DESCENDANT depuis ~251, pente
//	             MESUREE a 14-26 quanta/s, interrompu (fins a 185..246) ou complet (fin a
//	             127). La chute de l'anneau A L'EXPLOSION, elle, vaut ~138 quanta/s : les deux
//	             ne se confondent pas.
//	RECHARGE     le cycle complet 130 -> 253 -> 127 : il finit A SON MINIMUM, donc il n'est ni
//	             l'un ni l'autre. C'est ce qu'un decoupage en montees ne pouvait pas voir.
//
// # CE QUE CE FICHIER NE DECIDE PAS
//
// Il rend les segments et dit de chacun s'il a la FORME d'un armement ou d'une tenue de
// desarmement. La MECHE (le delai entre l'armement et l'explosion), la deduplication des
// paires de navpoints et la confrontation aux explosions vivent chez l'interpretation
// (`analysis/replay/bomb_armings.go`) : ce paquet ne connait ni les explosions ni le mode.
//
// L'instrument de mesure et la production appellent les MEMES fonctions — c'est la condition
// pour que le gate juge le code livre et non une copie qui derivera.

import "sort"

const (
	// NavpointSummitToleranceQ est la tolerance de fin de segment, en quanta : un armement
	// finit a son SOMMET a ce bruit pres, une tenue de desarmement ne remonte JAMAIS
	// au-dessus de son depart de plus que cela. 4 quanta = 1/64 de la course, tres en
	// dessous de l'amplitude minimale d'une montee (NavpointRiseMinQuanta = 16).
	NavpointSummitToleranceQ = 4
	// NavpointPauseMaxSlopeQS est la pente maximale (quanta par seconde) d'une TENUE DE
	// DESARMEMENT. Elle est posee au milieu du vide MESURE entre les tenues observees
	// (14 a 26 quanta/s) et la chute de l'anneau a l'explosion (138 quanta/s).
	NavpointPauseMaxSlopeQS = 60.0
)

// NavpointSegment est une suite maximale d'echantillons contigus de l'anneau d'un slot.
type NavpointSegment struct {
	// Slot est le point de navigation porteur (les navpoints vont par paires, +12).
	Slot uint32
	// StartMS / EndMS : instants du premier et du dernier echantillon, sur l'horloge du
	// manifeste (la meme que les lectures).
	StartMS, EndMS int32
	// QStart / QEnd : quanta du premier et du dernier echantillon (plage 0..255).
	QStart, QEnd uint8
	// QMin / QMax : les extremes ATTEINTS dans le segment — c'est leur position par rapport
	// aux bornes qui donne sa forme au segment.
	QMin, QMax uint8
	// Samples est le nombre d'echantillons du segment.
	Samples int
}

// NavpointSegments decoupe les lectures en segments contigus, tous slots, et les rend triees
// par (EndMS, Slot) — deterministe, et dans l'ordre qu'attend la confrontation (« le dernier
// armement avant l'explosion »).
func NavpointSegments(reads []NavpointRadialRead) []NavpointSegment {
	series := map[uint32][]NavpointRadialRead{}
	for _, r := range reads {
		series[r.Slot] = append(series[r.Slot], r)
	}
	out := make([]NavpointSegment, 0, len(series))
	for slot, s := range series {
		sort.Slice(s, func(i, j int) bool { return s[i].TMS < s[j].TMS })
		out = append(out, navpointSegmentsOfSeries(slot, s)...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].EndMS != out[j].EndMS {
			return out[i].EndMS < out[j].EndMS
		}
		return out[i].Slot < out[j].Slot
	})
	return out
}

// navpointSegmentsOfSeries decoupe UNE serie triee : un trou de plus de NavpointRiseMaxGapMS
// entre deux echantillons ferme le segment courant et ouvre le suivant.
func navpointSegmentsOfSeries(slot uint32, s []NavpointRadialRead) []NavpointSegment {
	var out []NavpointSegment
	for i := 0; i < len(s); {
		j := i
		for j+1 < len(s) && s[j+1].TMS-s[j].TMS <= NavpointRiseMaxGapMS {
			j++
		}
		g := NavpointSegment{
			Slot: slot, StartMS: s[i].TMS, EndMS: s[j].TMS,
			QStart: s[i].Q, QEnd: s[j].Q, QMin: s[i].Q, QMax: s[i].Q, Samples: j - i + 1,
		}
		for k := i; k <= j; k++ {
			if s[k].Q < g.QMin {
				g.QMin = s[k].Q
			}
			if s[k].Q > g.QMax {
				g.QMax = s[k].Q
			}
		}
		out = append(out, g)
		i = j + 1
	}
	return out
}

// EndsAtSummit dit que le segment a la FORME d'un ARMEMENT : assez d'echantillons, une
// amplitude montante d'au moins NavpointRiseMinQuanta, et un dernier echantillon a son sommet
// (a NavpointSummitToleranceQ pres) — l'anneau est reste plein.
//
// Ce que la forme ne dit PAS : que l'anneau a atteint le QUANTUM PLEIN. Un hold relache haut
// finit aussi a son sommet ; c'est l'interpretation qui exige le plein (`analysis/replay`).
func (g NavpointSegment) EndsAtSummit() bool {
	return g.Samples >= NavpointRiseMinSamples &&
		int(g.QEnd) >= int(g.QMax)-NavpointSummitToleranceQ &&
		int(g.QEnd)-int(g.QMin) >= NavpointRiseMinQuanta
}

// IsDisarmHold dit que le segment a la FORME d'une TENUE DE DESARMEMENT : il descend, il ne
// remonte jamais notablement au-dessus de son depart (le cycle de recharge, lui, remonte), et
// sa pente moyenne reste sous NavpointPauseMaxSlopeQS — la chute d'explosion la depasse.
//
// C'est la seule figure qui SUSPEND la meche : le compte a rebours reprend la ou il en etait
// quand la tenue s'interrompt (mesure du 2026-09-01, One Bomb 9/9, CV 0,017).
func (g NavpointSegment) IsDisarmHold() bool {
	if g.Samples < 2 || g.QEnd >= g.QStart || int(g.QMax) > int(g.QStart)+NavpointSummitToleranceQ {
		return false
	}
	durS := float64(g.EndMS-g.StartMS) / 1000
	if durS <= 0 {
		return false
	}
	return float64(int(g.QStart)-int(g.QEnd))/durS < NavpointPauseMaxSlopeQS
}
