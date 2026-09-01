package filmdec

// navpoint_radial_rises.go — LES MONTEES CONTIGUES DE L'ANNEAU (ti=12 i14), en production.
//
// # LA DEFINITION, ET D'OU ELLE VIENT
//
// Une MONTEE CONTIGUE est une suite maximale d'echantillons d'un meme slot, ordonnee dans le
// temps, NON DECROISSANTE, dont deux echantillons consecutifs sont separes d'au plus
// [NavpointRiseMaxGapMS], d'au moins [NavpointRiseMinSamples] echantillons et d'amplitude au
// moins [NavpointRiseMinQuanta] quanta. C'est la definition EXACTE du protocole du 2026-09-01
// (`navpoint_ti12_plancher_test.go`, fixee AVANT la mesure) : sous elle, la fin de la derniere
// montee precede chacune des 13 explosions Neutral Bomb de 4,93 s (CV 0,016) et AUCUN des
// 1 000 tirages nuls ne fait aussi bien. La contiguite est ce qui separe les rampes reelles
// des ramassages de queue.
//
// # CE QUE LES BORNES VEULENT DIRE
//
//	DEBUT de la montee   un joueur commence a tenir l'interaction d'armement (le hold :
//	                     ~5 s en Neutral Bomb, ~0,9 s en Husky Raid — un reglage de mode) ;
//	FIN de la montee     l'anneau se fige : la bombe est ARMEE (si la montee va au bout) ;
//	fin + la meche       l'explosion (4,93 s, constante moteur — la meche vit chez
//	                     l'interpretation, `analysis/replay`).
//
// Ce fichier ne decide PAS ce qu'est une montee « qui va au bout » ni ne deduplique les paires
// de navpoints : c'est le travail de l'interpretation. Il rend TOUTES les montees au sens du
// protocole, dans un ordre deterministe — l'instrument de mesure et la production appellent la
// MEME fonction, pour qu'ils ne divergent jamais.

import "sort"

// Les seuils de la definition d'une MONTEE CONTIGUE — protocole du 2026-09-01, 0/1000.
const (
	// NavpointRiseMaxGapMS : deux echantillons separes de plus que ce trou CASSENT la montee.
	NavpointRiseMaxGapMS = 500
	// NavpointRiseMinSamples : une montee compte au moins ce nombre d'echantillons.
	NavpointRiseMinSamples = 3
	// NavpointRiseMinQuanta : amplitude minimale, en quanta de la plage R(8) — 16 quanta =
	// 1/16 de la course du disque.
	NavpointRiseMinQuanta = 16
)

// NavpointRise est une montee contigue de l'anneau d'un slot.
type NavpointRise struct {
	// Slot est le point de navigation porteur (les navpoints vont par paires, +12).
	Slot uint32
	// StartMS / EndMS : les instants du premier et du dernier echantillon de la montee,
	// sur l'horloge du manifeste (la meme que les lectures).
	StartMS, EndMS int32
	// QStart / QEnd : les quanta du premier et du dernier echantillon (plage 0..255).
	QStart, QEnd uint8
	// Samples est le nombre d'echantillons de la montee.
	Samples int
}

// NavpointContiguousRises decoupe les lectures en montees contigues, tous slots, et les rend
// triees par (EndMS, Slot) — deterministe : deux montees peuvent finir a la MEME milliseconde
// (les navpoints vont par paires et portent le meme anneau).
func NavpointContiguousRises(reads []NavpointRadialRead) []NavpointRise {
	series := map[uint32][]NavpointRadialRead{}
	for _, r := range reads {
		series[r.Slot] = append(series[r.Slot], r)
	}
	out := make([]NavpointRise, 0, len(series))
	for slot, s := range series {
		sort.Slice(s, func(i, j int) bool { return s[i].TMS < s[j].TMS })
		out = append(out, navpointRisesOfSeries(slot, s)...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].EndMS != out[j].EndMS {
			return out[i].EndMS < out[j].EndMS
		}
		return out[i].Slot < out[j].Slot
	})
	return out
}

// navpointRisesOfSeries decoupe UNE serie triee en montees contigues au sens du protocole.
// La marche est celle de l'instrument (`tpMonteesContigues`), a l'identique : un trou de plus
// de NavpointRiseMaxGapMS entre deux echantillons casse la montee, et la borne de fin d'une
// montee ouvre la suivante.
func navpointRisesOfSeries(slot uint32, s []NavpointRadialRead) []NavpointRise {
	var out []NavpointRise
	for i := 0; i < len(s); {
		j := i
		for j+1 < len(s) && s[j+1].Q >= s[j].Q && s[j+1].TMS-s[j].TMS <= NavpointRiseMaxGapMS {
			j++
		}
		if n := j - i + 1; n >= NavpointRiseMinSamples &&
			int(s[j].Q)-int(s[i].Q) >= NavpointRiseMinQuanta {
			out = append(out, NavpointRise{
				Slot: slot, StartMS: s[i].TMS, EndMS: s[j].TMS,
				QStart: s[i].Q, QEnd: s[j].Q, Samples: n,
			})
		}
		if j == i {
			i++
			continue
		}
		i = j
	}
	return out
}
