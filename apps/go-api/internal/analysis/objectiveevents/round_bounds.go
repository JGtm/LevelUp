package objectiveevents

import "sort"

// round_bounds.go — LES BORNES DE MANCHE, ET LA QUESTION QU'ELLES TRANCHENT : « cet
// enregistrement peut-il appartenir a la manche qu'il DECLARE ? »
//
// # LE DEFAUT, MESURE
//
// La manche d'un enregistrement est lue dans deux en-tetes de 5 bits du premier composant
// ([decodeComponents]). L'assertion d'en-tete etant relachee, un residu de faux positifs
// franchit la porte : un enregistrement mal aligne porte une manche quelconque et des valeurs
// arbitraires. Jusqu'ici la decoupe par manche prenait ce numero pour argent comptant, et
// [longestRun] ne pouvait pas l'ecarter — une valeur PLUS GRANDE prolonge la suite non
// decroissante au lieu de la rompre.
//
// Mesure du 2026-09-06 sur `51ebbc0f` (Oddball, 2 manches) : un enregistrement date de
// 316 777 ms — soit 57 s APRES le debut de la manche 1 — declare la manche 0 et porte
// `comp 3 A = 60` (assistances) a cote d'un `comp 5 A = 4 164 778 782` qui le denonce. La
// manche 0 du slot 12 s'achevait donc sur 60 assistances, ce 60 devenait le decalage de la
// manche 1, et le document publiait 63 assistances pour un joueur qui en a 5 a la feuille.
// Le meme enregistrement portait `comp 2 A = 0`, ce qui coutait au passage son unique frag de
// manche 0 (la sous-suite non decroissante prefere la paire de zeros au 1 reel).
//
// # LA REGLE, ET POURQUOI ELLE NE TIENT A AUCUN SEUIL
//
// LES MANCHES SE JOUENT DANS L'ORDRE. Une manche occupe un intervalle de temps, et le suivant
// commence quand la suivante commence : un enregistrement date hors de l'intervalle de la
// manche qu'il declare ne peut pas en etre. Trois questions de mesure, toutes resolues par des
// MEDIANES et des MAJORITES — aucune constante ajustee :
//
//	DEBUT D'UNE MANCHE   la mediane, sur les slots, du premier instant ou le slot declare cette
//	                     manche. Le minimum ne convient pas : sur `24dbb67d`, DEUX slots
//	                     declarent la manche 1 des 85 193 ms alors que les huit autres la
//	                     commencent a 298 909 ms — le minimum aurait jete 3 612 enregistrements
//	                     legitimes de la manche 0, la mediane en jette 16.
//	MANCHE UTILISABLE    une manche dont MOINS DE LA MOITIE des slots parlent ne fixe aucune
//	                     borne : un debut de manche est un CONSENSUS, et un slot n'en est pas un.
//	                     Elle est retiree de la chaine, comme une manche sans enregistrement.
//	                     La separation mesuree est franche : sur les douze films multi-manche du
//	                     parc, une manche est declaree par les DIX slots ou par un seul
//	                     (`a4083bd2` manche 1 : 1 enregistrement, 1 slot) voire aucun.
//	BORNE CREDIBLE       une borne n'est appliquee que si elle SEPARE VRAIMENT les deux
//	                     populations : mediane des instants de la manche qui precede STRICTEMENT
//	                     avant la borne, mediane de celle qui suit a la borne ou apres.
//
// Et une garde d'ensemble : LES DEBUTS DOIVENT CROITRE avec le numero de manche. S'ils ne
// croissent pas, le numero de manche ne suit pas l'horloge du tout — aucune borne n'est alors
// posee, et le film sort exactement comme avant.
//
// Trois films du parc sont ainsi laisses INTACTS, et ce sont les trois dont l'etiquetage de
// manche est faux : `fb1a1a72` (CTF, « manche 2 » repandue de 66 s a 814 s), `72b0a25e` et
// `a4083bd2` (Slayer — un mode SANS manche). Le compte de leurs enregistrements ecartes est
// zero, verifie a l'octet sur l'artefact.
//
// # NEUTRALITE MONO-MANCHE, PAR CONSTRUCTION
//
// Moins de deux manches utilisables : aucune borne, donc aucun enregistrement ecarte. Un film
// mono-manche rend l'octet qu'il rendait avant.
//
// Instrument du releve : `analysis/replay/manches_bornes_research_test.go` (garde `MANCHES_CACHE`
// + `MANCHES_FILMS`). Mutations qui prouvent chaque garde :
// `analysis/replay/manches_compteurs_test.go`.

// roundSpan est l'intervalle de temps d'une manche, demi-ouvert : `[fromMS, toMS)`. Les
// bords non contraints valent [roundSpanOpenFrom] / [roundSpanOpenTo].
type roundSpan struct {
	fromMS int
	toMS   int
}

// Les bords ouverts d'une fenetre : la premiere manche n'a pas de borne gauche, la derniere
// pas de borne droite, et une borne jugee non credible laisse son bord ouvert.
const (
	roundSpanOpenFrom = -1 << 62
	roundSpanOpenTo   = 1<<62 - 1
)

// roundMark est ce qu'on mesure d'une manche pour en tirer une borne : son DEBUT (consensus des
// slots) et son MILIEU (le repere qui juge une borne credible).
type roundMark struct {
	round    int
	startMS  int
	middleMS int
}

// RoundBounds porte l'intervalle de temps de chaque manche REELLE d'un film. Une manche
// absente de la table n'est jamais contredite : la structure ne sait rien d'elle.
type RoundBounds struct {
	byRound map[int]roundSpan
}

// ResolveRoundBounds mesure l'intervalle de chaque manche du film (cf. l'en-tete de fichier
// pour la regle et les mesures qui la fondent).
func ResolveRoundBounds(recs []StatRecord) RoundBounds {
	marks := chainedRounds(recs)
	if len(marks) < 2 || !increasingStarts(marks) {
		return RoundBounds{}
	}
	out := make(map[int]roundSpan, len(marks))
	for _, m := range marks {
		out[m.round] = roundSpan{fromMS: roundSpanOpenFrom, toMS: roundSpanOpenTo}
	}
	for i := 0; i+1 < len(marks); i++ {
		prev, next := marks[i], marks[i+1]
		// LA BORNE DOIT SEPARER LES DEUX POPULATIONS, sans quoi elle n'est pas appliquee :
		// un etiquetage de manche qui ne suit pas le temps ne se corrige pas par une coupe.
		if prev.middleMS >= next.startMS || next.middleMS < next.startMS {
			continue
		}
		w := out[prev.round]
		w.toMS = next.startMS
		out[prev.round] = w
		w = out[next.round]
		w.fromMS = next.startMS
		out[next.round] = w
	}
	return RoundBounds{byRound: out}
}

// increasingStarts dit que les debuts croissent avec le numero de manche — l'invariant « les
// manches se jouent dans l'ordre ». Faux : le numero de manche ne suit pas l'horloge, et rien
// ne doit etre coupe sur sa foi.
func increasingStarts(marks []roundMark) bool {
	for i := 1; i < len(marks); i++ {
		if marks[i].startMS <= marks[i-1].startMS {
			return false
		}
	}
	return true
}

// chainedRounds rend, dans l'ordre, les manches REELLES qui peuvent fixer une borne : celles
// dont une MAJORITE des slots parlent (cf. l'en-tete, MANCHE UTILISABLE). Les autres — sans
// enregistrement, ou declarees par une poignee de slots — sont retirees de la chaine : leurs
// voisines s'enchainent alors directement.
func chainedRounds(recs []StatRecord) []roundMark {
	real := RealRounds(recs)
	parSlot := map[int]map[int]int{}
	instants := map[int][]int{}
	slotsMax := 0
	for _, r := range recs {
		if !real[r.Round] {
			continue
		}
		instants[r.Round] = append(instants[r.Round], r.TimeMS)
		if parSlot[r.Round] == nil {
			parSlot[r.Round] = map[int]int{}
		}
		if v, seen := parSlot[r.Round][r.Slot]; !seen || r.TimeMS < v {
			parSlot[r.Round][r.Slot] = r.TimeMS
		}
		if n := len(parSlot[r.Round]); n > slotsMax {
			slotsMax = n
		}
	}
	out := make([]roundMark, 0, len(parSlot))
	for round, debuts := range parSlot {
		if 2*len(debuts) <= slotsMax {
			continue
		}
		out = append(out, roundMark{
			round:    round,
			startMS:  medianOfMap(debuts),
			middleMS: medianOfSlice(instants[round]),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].round < out[j].round })
	return out
}

// medianOfMap rend la mediane des valeurs d'une table (les premiers instants par slot).
func medianOfMap(m map[int]int) int {
	out := make([]int, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return medianOfSlice(out)
}

// medianOfSlice rend la mediane BASSE d'une liste, qu'elle trie sur place.
//
// BASSE, ET C'EST UN CHOIX DE SURETE, pas un detail : sur une longueur paire, la mediane haute
// place le debut d'une manche APRES le premier slot qui l'a declaree, et ce slot perdrait sa
// premiere emission de la manche. Un corpus a deux slots l'a montre tout de suite
// (`named_onepass_test.go` : debuts 20 000 et 20 050, mediane haute = 20 050, l'emission de
// 20 000 tombait hors fenetre). Vers le BAS, la borne ne coupe jamais l'ouverture d'une manche ;
// elle peut au pire laisser passer la toute fin de la precedente, ce que la suite non
// decroissante encaisse. Sur les films multi-manche mesures (8 a 10 slots), les deux medianes
// donnent la MEME valeur : le choix ne se voit que sur les petits effectifs.
func medianOfSlice(v []int) int {
	if len(v) == 0 {
		return 0
	}
	sort.Ints(v)
	return v[(len(v)-1)/2]
}

// Excludes dit qu'un enregistrement est date HORS de l'intervalle de la manche qu'il declare —
// donc que sa manche est mal lue et qu'il ne doit alimenter aucune serie.
//
// Une manche inconnue de la table (manche fantome, manche sans consensus, film mono-manche)
// n'est jamais contredite.
func (w RoundBounds) Excludes(r StatRecord) bool {
	span, ok := w.byRound[r.Round]
	if !ok {
		return false
	}
	return r.TimeMS < span.fromMS || r.TimeMS >= span.toMS
}

// Outliers compte les enregistrements qu'[Excludes] ecarte. Sert au journal de cuisson : le
// compte est le NOMINAL d'un film multi-manche (5 a 24 sur les temoins mesures), et son
// explosion signalerait un etiquetage de manche qui ne tient plus.
func (w RoundBounds) Outliers(recs []StatRecord) int {
	if len(w.byRound) == 0 {
		return 0
	}
	n := 0
	for _, r := range recs {
		if w.Excludes(r) {
			n++
		}
	}
	return n
}
