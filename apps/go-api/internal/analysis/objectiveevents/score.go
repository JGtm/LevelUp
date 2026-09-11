package objectiveevents

import "sort"

// score.go — les COURBES tirees des enregistrements d'entite : score de mode et score
// personnel, a la milliseconde.
//
// Complement direct des events d'objectif : ceux-ci disent QUI prend une base ou capture un
// drapeau, celles-ci disent OU EN EST le score a cet instant. Meme horloge, superposables
// sans recalage. Le decodage des enregistrements lui-meme vit dans statborg.go.
//
// # Ou vit quoi, et c'est mesure et non suppose
//
//	composant 0, valeur A : score de MODE      (200-94 en Strongholds, 3-0 en CTF)
//	composant 1, valeur B : score PERSONNEL    (valeur A constamment nulle)
//	composant 2           : frags et morts
//
// # Rendement, confronte position de bit par position de bit a une capture Cheat Engine
//
//	Strongholds 696a9d7c, composant 0 : 283/284 (99,6 %), 2 faux positifs ;
//	CTF         530820e5, composant 0 : 6/6 (100 %), 0 faux positif ;
//	Strongholds, composant 1 : 374/381 · composant 2 : 385/397 ;
//	zero valeur fausse a bonne position dans tous les cas.

// modeScoreComp / personalScoreComp : les index de composant nommes par la mesure.
const (
	modeScoreComp     = 0
	personalScoreComp = 1
)

// ScorePoint est une emission de score par une entite du match.
type ScorePoint struct {
	// TimeMS est l'instant de l'emission sur l'horloge du film.
	TimeMS int
	// Slot identifie l'entite : 6 et 8 sont les deux equipes, 10..24 les huit joueurs.
	Slot int
	// Value est le score a cet instant.
	Value int64
}

// StatComponent designe un emplacement de statistique repliquee : l'index du composant, le
// cote (A ou B) de sa paire de valeurs, et la stricte croissance attendue de sa suite.
//
// C'est la meme adresse que `statSlotKey`, EXPORTEE pour que l'assembleur du rejeu demande
// une serie sans avoir a connaitre la grammaire du statborg. Aucun de ces emplacements n'est
// suppose : chacun est mesure (cf. l'en-tete de ce fichier et celui de slotidentity.go).
type StatComponent struct {
	// Comp est l'index du composant dans l'archetype.
	Comp int
	// SideB choisit la valeur B de la paire plutot que la valeur A.
	SideB bool
	// Strict dit si la suite retenue doit etre STRICTEMENT croissante. La distinction n'est
	// pas cosmetique (cf. [longestRun]) : le score de MODE ne repete jamais une valeur (une
	// egalite serait un doublon a jeter), alors qu'un compteur de recompense est seulement
	// NON DECROISSANT — un composant porte deux valeurs et il est reemis des que l'UNE des
	// deux bouge, donc la meme valeur revient legitimement.
	Strict bool
	// Unitary dit que CHAQUE UNITE de ce compteur est une ACTION du joueur — un frag, une
	// mort, une assistance — donc que la borne par pas de [boundSteps] s'y applique.
	//
	// # LA BORNE NE VAUT QUE POUR EUX, ET C'EST MESURE (correctif 6.R, 2026-09-11)
	//
	// [maxUnrollPerStep] = 16 est calibre sur l'oracle API des compteurs d'ACTION : pire pas
	// sain mesure 3, plus petit pas aberrant 64. Les autres emplacements publies sont des
	// compteurs de CADENCE, dont un pas legitime depasse tres largement 16 :
	//
	//	score PERSONNEL (comp 1 B)     +100 et plus par frag ;
	//	tics de GARDE   (comp 23 A)    35 tics valent UN point de colline (hill_hold_ticks.go) ;
	//	score de MODE   (comp 0 A)     Strongholds compte des tics, Oddball des secondes.
	//
	// Les borner effacerait des courbes vraies. D'ou l'opt-in explicite plutot qu'un filtre
	// pose sur toutes les series : le domaine de validite de la borne est SA calibration.
	Unitary bool
}

// Les emplacements que le rejeu publie.
var (
	// ModeScoreComponent : le score de MODE (comp 0, valeur A) — celui que l'ECRAN affiche,
	// et qui n'est pas toujours celui de l'API (phase 0-ter du lot A : Strongholds compte des
	// ticks, KOTH des secondes de colline).
	ModeScoreComponent = StatComponent{Comp: modeScoreComp, Strict: true}
	// PersonalScoreComponent : le score PERSONNEL (comp 1, valeur B).
	PersonalScoreComponent = StatComponent{Comp: personalScoreComp, SideB: true}
	// KillsComponent, DeathsComponent, AssistsComponent : les trois compteurs de base,
	// confirmes nominativement contre `match_participants` (cf. slotidentity.go).
	//
	// CE SONT EXACTEMENT LES TROIS EMPLACEMENTS QUE LIT LA CLE D'APPARIEMENT
	// ([SlotIdentityFrom] -> [countsOf]), d'ou leur `Unitary` : la serie publiee et la cle
	// derivent alors le meme compteur par le meme verdict de [boundSteps], et ne peuvent plus
	// annoncer deux totaux differents (correctif 6.R, 2026-09-11).
	KillsComponent   = StatComponent{Comp: coreKillsComp, Unitary: true}
	DeathsComponent  = StatComponent{Comp: coreKillsComp, SideB: true, Unitary: true}
	AssistsComponent = StatComponent{Comp: coreAssistsComp, Unitary: true}

	// SkullTicksComponent : le canal du PORTEUR d'Oddball. C'est le score de MODE par joueur
	// (`comp 0 A`), qui en Oddball compte les TICS DE POSSESSION du crane (`skull_scoring_ticks`,
	// identifie par l'oracle films confondus : 47 accords non-nuls sur 7 films). C'est le
	// ModeScoreComponent SANS la stricte croissance : le composant est reemis a valeur A EGALE
	// quand sa valeur B bouge, et [longestRun] non strict garde ces repetitions ; c'est la
	// deduplication par valeur du consommateur (`skullTickInstants`, paquet replay) qui date
	// ensuite chaque tic a sa PREMIERE emission (la vraie), la ou la stricte croissance le
	// daterait a la derniere. C'est
	// l'instrument valide au gate du porteur (principal 7/7 films) — a ne pas changer sans
	// remesurer.
	SkullTicksComponent = StatComponent{Comp: modeScoreComp}
	// SkullGrabsComponent : les PRISES du crane d'Oddball (`comp 21 B` = `skull_grabs`, identifie
	// par l'oracle films confondus : 21 accords non-nuls, 56 accords totaux sur 7 films). Un
	// compteur non decroissant, donc non strict.
	SkullGrabsComponent = StatComponent{Comp: skullGrabsComp, SideB: true}
)

// skullGrabsComp est l'index du composant des prises du crane (`skull_grabs`), en valeur B.
const skullGrabsComp = 21

// key traduit l'emplacement exporte en cle interne.
func (c StatComponent) key() statSlotKey {
	side := sideA
	if c.SideB {
		side = sideB
	}
	return statSlotKey{Comp: c.Comp, Side: side}
}

// SeriesByRound rend, pour un emplacement, les emissions de chaque slot MANCHE PAR MANCHE :
// valeurs propres a la manche, NON cumulees. C'est la forme que publie l'artefact de rejeu,
// ou chaque manche est une courbe distincte.
//
// teams choisit les slots d'equipe (6 et 8) plutot que ceux de joueur (10..24 pairs). Les
// manches FANTOMES — celles que [RealRounds] refuse — sont ecartees : les cumuler ferait
// exploser les compteurs (mesure : le score d'equipe d'un CTF passait de 1 a 2 104).
//
// UN COMPTEUR `Unitary` PASSE PAR [boundedSeries] (correctif 6.R) : la borne par pas s'applique
// MANCHE PAR MANCHE, `prev` repartant de zero a chaque manche — c'est exactement le decoupage
// des pas que voit [countsOf] sur la suite cumulee, puisque le decalage d'une manche vaut le
// total de la precedente (le pas de sa premiere emission y vaut donc sa valeur brute). Les deux
// lectures jugent le meme pas.
func SeriesByRound(recs []StatRecord, c StatComponent, teams bool) map[int]map[int][]ScorePoint {
	real := RealRounds(recs)
	out := map[int]map[int][]ScorePoint{}
	for slot, byRound := range rawSeriesByRound(recs, c.key(), teams) {
		for round, pts := range byRound {
			if !real[round] {
				continue
			}
			sort.SliceStable(pts, func(i, j int) bool { return pts[i].TimeMS < pts[j].TimeMS })
			kept := longestRun(pts, c.Strict)
			if c.Unitary {
				kept = boundedSeries(kept)
			}
			if len(kept) == 0 {
				continue
			}
			if out[slot] == nil {
				out[slot] = map[int][]ScorePoint{}
			}
			out[slot][round] = kept
		}
	}
	return out
}

// SeriesTotal rend, pour un emplacement, la serie CUMULEE sur le match de chaque slot :
// chaque manche est decalee du total des precedentes, la suite est donc croissante de bout en
// bout et son dernier point vaut le total du match.
//
// C'est le pendant de [SeriesByRound], et les deux sont necessaires : la manche est ce que
// l'ecran affiche pendant qu'elle se joue, le total est ce que le match retient.
//
// UN COMPTEUR `Unitary` PASSE PAR [boundedSeries] (correctif 6.R), sur la suite CUMULEE et donc
// sur les memes pas que [countsOf] : le total publie vaut exactement le compte que la cle
// d'appariement a lu.
func SeriesTotal(recs []StatRecord, c StatComponent, teams bool) map[int][]ScorePoint {
	out := cumulateRounds(rawSeriesByRound(recs, c.key(), teams), RealRounds(recs))
	for slot, pts := range out {
		if c.Unitary {
			pts = boundedSeries(pts)
		}
		if c.Strict {
			pts = longestRun(pts, true)
		}
		out[slot] = pts
	}
	return out
}

// longestRun rend la plus longue sous-suite croissante en valeur, l'ordre d'entree etant
// deja chronologique (patience sorting, O(n log n)).
//
// strict choisit entre les deux usages, et la difference n'est pas cosmetique :
//   - le score de MODE est strictement croissant (une egalite serait un doublon a jeter) ;
//   - un compteur de recompense est NON DECROISSANT — un composant porte deux valeurs et
//     il est reemis des que l'UNE des deux bouge, donc la meme valeur revient
//     legitimement. Exiger la stricte croissance y jetterait des emissions reelles.
func longestRun(pts []ScorePoint, strict bool) []ScorePoint {
	if len(pts) == 0 {
		return nil
	}
	var tailIdx []int             // tailIdx[k] = index de la fin de la suite de longueur k+1
	prev := make([]int, len(pts)) // chainage arriere pour la reconstruction
	for i, p := range pts {
		lo, hi := 0, len(tailIdx)
		for lo < hi {
			mid := (lo + hi) / 2
			fits := pts[tailIdx[mid]].Value <= p.Value
			if strict {
				fits = pts[tailIdx[mid]].Value < p.Value
			}
			if fits {
				lo = mid + 1
			} else {
				hi = mid
			}
		}
		prev[i] = -1
		if lo > 0 {
			prev[i] = tailIdx[lo-1]
		}
		if lo == len(tailIdx) {
			tailIdx = append(tailIdx, i)
		} else {
			tailIdx[lo] = i
		}
	}
	out := make([]ScorePoint, len(tailIdx))
	for i, k := len(tailIdx)-1, tailIdx[len(tailIdx)-1]; i >= 0; i, k = i-1, prev[k] {
		out[i] = pts[k]
	}
	return out
}

// modeScoreInDomain dit si un composant de SCORE DE MODE est plausible, SUR SES DEUX CANAUX.
//
// Un compteur de recompense est positif, et aucun mode Halo Infinite ne fait marquer plus de
// [statMaxModeScore] points a une entite dans une manche. Les deux canaux d'un composant sont
// decodes de la MEME emission : si l'un est hors domaine, l'emission etait mal alignee et
// l'autre ne vaut rien non plus. C'est la GARDE DE DOMAINE (garde anti-manche-fantome n 1),
// etendue au canal frere le 2026-08-31.
func modeScoreInDomain(v StatValue) bool {
	return v.A >= 0 && v.A <= statMaxModeScore && v.B >= 0 && v.B <= statMaxModeScore
}
