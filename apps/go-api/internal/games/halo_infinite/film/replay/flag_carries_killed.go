package replay

// flag_carries_killed.go — `flag_carriers_killed` : QUEL PORTAGE UNE CHUTE CREDITEE PEUT FERMER
// (constat RB1-1 de l audit du 2026-09-24, lot J9.1 du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25).
//
// # CE QUE L EVENEMENT DIT, ET CE QU IL NE DIT PAS
//
// Il est credite au TUEUR, pas a la victime : il ne nomme pas le porteur qui tombe. Il ne sert
// qu a fermer un portage que le fil des morts aurait manque, et SEULEMENT quand exactement UN
// portage peut etre celui de la victime — sinon rien n indique lequel, et l evenement se compte en
// incoherence (`ambiguousCarrierKills`) plutot que de fermer au hasard.
//
// # QUI PEUT ETRE LA VICTIME : UN ADVERSAIRE DU TUEUR, ET PERSONNE D AUTRE
//
// Jusqu au 2026-09-26 le seul porteur exclu etait le tueur lui-meme. Deux consequences, toutes
// deux fausses : un COEQUIPIER du tueur, seul porteur ouvert a cet instant, voyait son portage
// ferme et sa capture effacee (le drapeau publie AU SOL pendant qu il courait avec) ; et un tueur
// NON NOMME par le pont n excluait personne. La regle lit desormais l equipe LUE DANS LE FILM
// (`FlagCarryScan.TeamOf`, lot 1.7) :
//
//	tueur non nomme, ou equipe du tueur non lue   aucune fermeture ; compte `unjudgedCarrierKills`
//	porteur de l equipe du tueur                  jamais candidat (on ne credite pas la chute
//	                                              d un coequipier comme celle d un porteur adverse)
//	porteur d equipe non lue                      candidat DOUTEUX : il empeche de trancher
//	un seul candidat adverse, aucun douteux       fermeture (`flagCloserCarrierKill`)
//	plusieurs candidats, ou un et un douteux      aucune fermeture ; compte `ambiguousCarrierKills`
//	aucun candidat, au moins un douteux           aucune fermeture ; compte `unjudgedCarrierKills`
//
// Le fait ne CLOT toujours pas le portage (cf. [flagCloser.ferme]) : seule sa borne change.

import "levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"

// carrierKillVerdict est ce que la regle statue pour UN evenement.
type carrierKillVerdict uint8

const (
	// carrierKillNone : aucun portage ouvert d un autre joueur a cet instant — rien a statuer.
	carrierKillNone carrierKillVerdict = iota
	// carrierKillClose : un seul candidat adverse, aucun doute — il est ferme.
	carrierKillClose
	// carrierKillAmbiguous : plusieurs candidats, ou un candidat et un porteur d equipe non lue.
	carrierKillAmbiguous
	// carrierKillUnjudged : le tueur n est pas nomme, son equipe n est pas lue, ou seuls des
	// porteurs d equipe non lue etaient ouverts.
	carrierKillUnjudged
)

// closeByCarrierKills raccourcit un portage quand `flag_carriers_killed` date une chute que le fil
// des morts n a pas vue. Rend le nombre d evenements AMBIGUS et celui des evenements NON JUGES
// (cf. l en-tete).
func closeByCarrierKills(raws []flagCarryRaw, scan FlagCarryScan) (ambiguous, unjudged int) {
	for _, e := range scan.Events {
		if e.Stat != objectives.StatFlagCarriersKilled {
			continue
		}
		at := int64(e.TimeMS)
		victim, verdict := carrierKillVictim(raws, scan, at, scan.Identity.At(e.Slot, e.TimeMS))
		switch verdict {
		case carrierKillClose:
			flagCloseAt(&raws[victim], at, flagCloserCarrierKill)
		case carrierKillAmbiguous:
			ambiguous++
		case carrierKillUnjudged:
			unjudged++
		}
	}
	return ambiguous, unjudged
}

// carrierKillVictim rend l index du SEUL portage qui peut etre celui de la victime a l instant
// `at`, et le verdict qui le fonde.
func carrierKillVictim(raws []flagCarryRaw, scan FlagCarryScan, at int64, killer string) (int, carrierKillVerdict) {
	killerTeam, known := scan.TeamOf[killer]
	if killer == "" || !known || killerTeam == TeamNeutral {
		return -1, carrierKillUnjudged
	}
	victim, candidates, doubtful := -1, 0, false
	for i := range raws {
		if raws[i].t0 >= at || at >= raws[i].t1 || raws[i].xuid == killer {
			continue
		}
		team, lue := scan.TeamOf[raws[i].xuid]
		switch {
		case !lue || team == TeamNeutral:
			doubtful = true
		case team != killerTeam:
			victim, candidates = i, candidates+1
		}
	}
	return verdictDuPorteurTue(victim, candidates, doubtful)
}

// verdictDuPorteurTue statue selon la table de l en-tete, a partir des portages candidats comptes
// (ceux des adversaires du tueur ouverts a l instant, `victim` etant l index du dernier vu) et du
// doute (un porteur d equipe non lue etait ouvert) : un seul candidat sans doute se ferme ;
// plusieurs, ou un et un doute, sont ambigus ; un doute sans candidat n est pas juge ; sinon, rien
// a statuer.
func verdictDuPorteurTue(victim, candidates int, doubtful bool) (int, carrierKillVerdict) {
	switch {
	case candidates > 1 || (candidates == 1 && doubtful):
		return -1, carrierKillAmbiguous
	case candidates == 1:
		return victim, carrierKillClose
	case doubtful:
		return -1, carrierKillUnjudged
	}
	return -1, carrierKillNone
}
