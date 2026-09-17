package objectives

import "sort"

// rounds_decision.go — LE VERDICT COMPLET DE LA LECTURE DES DESIGNATEURS DE MANCHE.
//
// # POURQUOI CE TYPE EXISTE (lot 1.9.11, 2026-09-16)
//
// [RealRounds] ne rendait qu'un ensemble : les manches retenues. Tout le reste — ce que le film
// a ECRIT, ce que l'admission a refuse, et le cas ou aucune manche n'est admise et ou la
// manche 0 est DECRETEE — se perdait dans la fonction. Or c'est exactement ce que la decision
// D14 (c) du PLAN_DECODEUR_FILM exige de publier : « chaque fait publie
// `coverage.<fait>.{grammaire, repli, contradiction}` ; l'artefact dit quelle part de lui vient
// d'un repli ».
//
// Le fait, ici, est « QUELLES MANCHES D'UN MATCH SONT REELLES », et il se decompose en trois :
//
//	GRAMMAIRE      les designateurs que le film ECRIT sur ses enregistrements ([Written]).
//	CONTRADICTION  ceux qui sont MATERIELS — assez d'emissions pour etre une manche — et que
//	               l'ordre refuse parce que le film ne declare pas les manches qui les precedent
//	               ([Contradicted]). Ce n'est PAS un repli au sens de D14 (b) : un repli se
//	               declenche sur un silence du film, celui-ci se declenche sur un DESACCORD avec
//	               une lecture disponible. Il se compte et se publie, il ne se tait pas.
//	REPLI          le decret de la manche 0 quand AUCUNE manche n'est admise ([Decreed]) : la,
//	               le film est muet et la chaine pose une manche pour rester lisible.
//	               C'est `repli_manche_zero_decretee` au registre.
//
// # LA MESURE QUI FONDE LA SEPARATION (1 351 films du cache, 2026-09-16)
//
// Cf. l'en-tete de [contiguousRounds] : le motif « designateur materiel, manche precedente
// absente » existe sur 24 films, dont 23 ont fini DANS leur temps reglementaire sur des modes
// sans manche. Ce sont donc des contradictions a compter, pas des manches a publier.
//
// # NEUTRALITE
//
// [RealRounds] rend exactement ce qu'elle rendait : `ResolveRounds(recs).RealSet()` passe par la
// meme chaine, dans le meme ordre. Aucun octet cuit ne change de ce fait — seuls les champs de
// couverture neufs s'ajoutent.

// RoundsDecision porte le verdict complet de la lecture des designateurs de manche d'un film.
type RoundsDecision struct {
	// Written : les designateurs que le film ECRIT, tries. C'est la GRAMMAIRE, avant tout
	// jugement — un designateur porte par un seul enregistrement fortuit y figure comme un
	// autre, et c'est voulu : le denominateur doit etre lisible.
	Written []int
	// Real : les manches RETENUES, triees. C'est ce que tout le reste du depot consomme.
	Real []int
	// Contradicted : les designateurs MATERIELS (au sens des deux criteres d'admission) que
	// l'ordre refuse. Triees.
	Contradicted []int
	// ContradictedRecords : le nombre d'enregistrements que ces designateurs portent — la
	// grandeur qui dit si la contradiction est marginale ou massive.
	ContradictedRecords int
	// Decreed : aucune manche n'a ete admise, la manche 0 a ete DECRETEE (repli).
	Decreed bool
}

// RealSet rend les manches retenues sous la forme d'ensemble attendue par les lecteurs.
func (d RoundsDecision) RealSet() map[int]bool {
	out := make(map[int]bool, len(d.Real))
	for _, r := range d.Real {
		out[r] = true
	}
	return out
}

// ResolveRounds rend le verdict complet. Fonction PURE, sans etat de paquet.
func ResolveRounds(recs []StatRecord) RoundsDecision {
	runs := modeScoreRunsByRound(recs)
	material, present := materialRounds(recs), presentRounds(recs)
	real, decreed := contiguousRounds(runs, material, present)

	d := RoundsDecision{
		Written: sortedTrueKeys(present),
		Real:    sortedTrueKeys(real),
		Decreed: decreed,
	}
	for _, round := range d.Written {
		if real[round] || !admissibleRound(runs, material, round) {
			continue
		}
		d.Contradicted = append(d.Contradicted, round)
	}
	d.ContradictedRecords = recordsOfRounds(recs, d.Contradicted)
	return d
}

// admissibleRound dit qu'une manche passe l'un des DEUX criteres d'admission — suite coherente
// du score de mode, ou matiere. C'est la meme condition que la boucle de [contiguousRounds], et
// elle est ecrite ICI une seule fois pour que les deux lectures ne puissent pas diverger.
func admissibleRound(runs map[int]int, material map[int]bool, round int) bool {
	return runs[round] >= statMinRoundRun || material[round]
}

// recordsOfRounds compte les enregistrements portant l'un des designateurs nommes.
func recordsOfRounds(recs []StatRecord, rounds []int) int {
	if len(rounds) == 0 {
		return 0
	}
	in := make(map[int]bool, len(rounds))
	for _, r := range rounds {
		in[r] = true
	}
	n := 0
	for _, r := range recs {
		if in[r.Round] {
			n++
		}
	}
	return n
}

// sortedTrueKeys rend, triees, les cles vraies d'un ensemble de manches.
func sortedTrueKeys(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for r, ok := range m {
		if ok {
			out = append(out, r)
		}
	}
	sort.Ints(out)
	return out
}
