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
// # GARDE PAR SLOT : UNE LECTURE VRAIE N'EST JAMAIS JETEE (revue MANCHES-R1, 2026-09-06)
//
// Les trois mesures ci-dessus sont des CONSENSUS GLOBAUX. Un slot minoritaire peut ouvrir ou
// fermer sa manche a cote du consensus sans que ses emissions soient fausses : la borne coupe
// alors au milieu, ou apres la fin, d'un bloc legitime. Fixture du relecteur (8 slots, 3 ouvrant
// la manche 1 a 40 s, 5 a 70 s) : 12 enregistrements legitimes jetes, `rounds[1]` VIDE pour trois
// joueurs, total d'assistances 8 -> 5.
//
// La regle est donc bornee : ON N'ECARTE QUE CE QUI EST CONTREDIT. Un bloc (slot, manche) dont
// une PARTIE tombe dans la fenetre voit ses emissions hors fenetre ecartees — elles sont
// contredites par celles du meme slot pour la meme manche (`51ebbc0f` slot 12 : un enregistrement
// a 316 777 ms contre 24 dans la fenetre). Un bloc qui tombe ENTIEREMENT hors de la fenetre n'est
// contredit par rien : il est GARDE dans sa manche declaree, et journalise
// ([RoundBounds.KeptSegments]). Sur les douze films multi-manche du parc, aucun bloc n'est dans
// ce cas : l'exemption est une surete, pas un cas nominal.
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

// slotRound designe le bloc d'enregistrements d'UN slot pour UNE manche declaree. C'est l'unite
// que la garde par slot protege : une borne globale ne doit jamais faire disparaitre un bloc
// entier.
type slotRound struct {
	slot  int
	round int
}

// KeptSegment decrit un bloc (slot, manche) GARDE dans sa manche declaree bien qu'il tombe
// ENTIEREMENT hors de la fenetre consensuelle de cette manche. C'est ce que l'appelant
// journalise : le bloc n'est pas jete, mais le consensus et ce slot ne s'accordent pas sur les
// bornes de la manche, et cela doit se voir.
type KeptSegment struct {
	// Slot et Round identifient le bloc.
	Slot, Round int
	// FromMS et ToMS bornent le bloc sur l'horloge du film.
	FromMS, ToMS int
	// GapMS est l'ecart, en ms, entre le bloc et le bord de fenetre le plus proche.
	GapMS int
	// Records est le nombre d'enregistrements du bloc.
	Records int
}

// RoundBounds porte l'intervalle de temps de chaque manche REELLE d'un film. Une manche
// absente de la table n'est jamais contredite : la structure ne sait rien d'elle.
//
// `kept` porte les blocs (slot, manche) que la fenetre exclurait ENTIEREMENT : ils sont EXEMPTES
// (cf. [RoundBounds.Excludes] et l'en-tete, GARDE PAR SLOT).
type RoundBounds struct {
	byRound map[int]roundSpan
	kept    map[slotRound]KeptSegment
}

// ResolveRoundBounds mesure l'intervalle de chaque manche du film (cf. l'en-tete de fichier
// pour la regle et les mesures qui la fondent), puis EXEMPTE les blocs par slot qu'elle
// exclurait entierement (cf. GARDE PAR SLOT).
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
	return RoundBounds{byRound: out, kept: keptSegmentsOf(recs, out)}
}

// keptSegmentsOf rend les blocs (slot, manche) que les fenetres excluraient ENTIEREMENT.
//
// # LA GARDE PAR SLOT, ET LA DOCTRINE QU'ELLE APPLIQUE
//
// Une lecture VRAIE n'est jamais jetee. Les bornes sont un CONSENSUS global ; un slot minoritaire
// peut ouvrir ou fermer sa manche a cote de ce consensus sans que ses emissions soient fausses
// pour autant. Si TOUT le bloc d'un slot pour une manche tombe hors de la fenetre, ce n'est plus
// un egare isole : c'est un desaccord sur les bornes, et jeter le bloc ferait disparaitre la
// contribution entiere d'un joueur a une manche. Mesure du relecteur (revue MANCHES-R1, fixture a
// 8 slots dont 3 ouvrent la manche 1 a 40 s et 5 a 70 s) : 12 enregistrements legitimes jetes,
// `rounds[1]` VIDE pour trois joueurs, total d'assistances 8 -> 5.
//
// Un bloc dont une PARTIE tombe dans la fenetre, lui, n'est pas exempte : les emissions hors
// fenetre y sont contredites par les emissions du meme slot pour la meme manche, et c'est le cas
// de l'egare (`51ebbc0f` slot 12 : un enregistrement a 316 777 ms contre 24 dans la fenetre).
func keptSegmentsOf(recs []StatRecord, spans map[int]roundSpan) map[slotRound]KeptSegment {
	type bilan struct {
		dedans, dehors int
		fromMS, toMS   int
	}
	par := map[slotRound]*bilan{}
	for _, r := range recs {
		span, ok := spans[r.Round]
		if !ok {
			continue
		}
		k := slotRound{slot: r.Slot, round: r.Round}
		b := par[k]
		if b == nil {
			b = &bilan{fromMS: r.TimeMS, toMS: r.TimeMS}
			par[k] = b
		}
		if r.TimeMS < b.fromMS {
			b.fromMS = r.TimeMS
		}
		if r.TimeMS > b.toMS {
			b.toMS = r.TimeMS
		}
		if r.TimeMS < span.fromMS || r.TimeMS >= span.toMS {
			b.dehors++
		} else {
			b.dedans++
		}
	}
	out := map[slotRound]KeptSegment{}
	for k, b := range par {
		if b.dedans > 0 || b.dehors == 0 {
			continue
		}
		out[k] = KeptSegment{
			Slot: k.slot, Round: k.round, FromMS: b.fromMS, ToMS: b.toMS,
			GapMS: gapToSpan(b.fromMS, b.toMS, spans[k.round]), Records: b.dehors,
		}
	}
	return out
}

// gapToSpan rend l'ecart, en ms, entre un bloc entierement hors fenetre et le bord de fenetre
// dont il est le plus proche. Zero quand la fenetre n'a pas de bord de ce cote.
func gapToSpan(fromMS, toMS int, span roundSpan) int {
	if toMS < span.fromMS && span.fromMS != roundSpanOpenFrom {
		return span.fromMS - toMS
	}
	if fromMS >= span.toMS && span.toMS != roundSpanOpenTo {
		return fromMS - span.toMS
	}
	return 0
}

// KeptSegments rend les blocs (slot, manche) exemptes, tries par slot puis par manche. Vide sur
// tous les films multi-manche du parc mesures le 2026-09-06 : l'exemption est une SURETE, pas un
// cas nominal.
func (w RoundBounds) KeptSegments() []KeptSegment {
	out := make([]KeptSegment, 0, len(w.kept))
	for _, s := range w.kept {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Slot != out[j].Slot {
			return out[i].Slot < out[j].Slot
		}
		return out[i].Round < out[j].Round
	})
	return out
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
// DEUX EXEMPTIONS, et les deux disent la meme chose : on n'ecarte que ce qui est CONTREDIT.
//
//  1. une manche inconnue de la table (manche fantome, manche sans consensus, film mono-manche)
//     n'est jamais contredite ;
//  2. un bloc (slot, manche) qui tombe ENTIEREMENT hors de la fenetre n'est pas un egare mais un
//     desaccord sur les bornes : il est garde dans sa manche declaree, et l'appelant le
//     journalise (cf. [keptSegmentsOf] et [RoundBounds.KeptSegments]).
func (w RoundBounds) Excludes(r StatRecord) bool {
	span, ok := w.byRound[r.Round]
	if !ok {
		return false
	}
	if _, exempte := w.kept[slotRound{slot: r.Slot, round: r.Round}]; exempte {
		return false
	}
	return r.TimeMS < span.fromMS || r.TimeMS >= span.toMS
}

// OutliersNominalMax est le plus grand nombre d'enregistrements ecartes observe sur un film SAIN
// du parc — LA SEULE ECRITURE de cette fourchette, referencee par le journal de cuisson
// (`analysis/replay/build_score.go`).
//
// Releve du 2026-09-06 sur les douze films multi-manche du parc de 119 :
//
//	c75f33b8 27 · 9f57c612 24 · d9781168 20 · 24dbb67d 19 · 43716616 19 · 51ebbc0f 13
//	7fce3219 9 · cde26226 6 · 64e8adfa 6 · fb1a1a72 0 · 72b0a25e 0 · a4083bd2 0
//
// Un compte AU-DELA n'est pas une erreur en soi : c'est le signal qu'un etiquetage de manche a
// cesse de tenir sur ce film, et qu'il faut le regarder. Les trois zeros sont les films dont le
// numero de manche ne suit pas l'horloge : aucune borne n'y est posable.
const OutliersNominalMax = 27

// Outliers compte les enregistrements qu'[Excludes] ecarte. Sert au journal de cuisson ; la
// fourchette nominale est [OutliersNominalMax].
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

// RoundStartsMS rend le DEBUT CONSENSUEL de chaque manche utilisable du film — LA SEULE
// DEFINITION du « debut d'une manche » dans ce paquet.
//
// # POURQUOI ELLE EST EXPORTEE (constat C2 de la revue MANCHES-R1, 2026-09-06)
//
// Deux definitions coexistaient : la decoupe par manche prenait la mediane des premiers instants
// par slot, et l'identite par manche ([roundStartsOf], slotidentity_rounds.go) prenait le
// MINIMUM des instants declares. Sur `24dbb67d`, la manche 1 commencait donc a 85 193 ms pour
// l'identite et a 298 909 ms pour la decoupe — 213 s d'ecart, pendant lesquelles
// [RoundIdentity.At] resolvait la manche SUIVANTE et attribuait toute action d'objectif au joueur
// de la mauvaise manche. Le minimum suit le premier faux positif venu ; la mediane est un
// consensus. Une seule des deux peut etre le debut d'une manche.
//
// Les manches que le consensus ne peut pas fixer (sans enregistrement, sans majorite de slots)
// sont ABSENTES du resultat : l'appelant decide quoi en faire, il n'y a rien a mesurer.
func RoundStartsMS(recs []StatRecord) map[int]int {
	marks := chainedRounds(recs)
	out := make(map[int]int, len(marks))
	for _, m := range marks {
		out[m.round] = m.startMS
	}
	return out
}
