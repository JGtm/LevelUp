package replay

import "sort"

// score_team_identity.go — QUEL CAMP EST DERRIERE UN SLOT D'ENTITE D'EQUIPE.
//
// # LE PROBLEME, ET IL EST LE MEME QUE POUR LES JOUEURS
//
// Le statborg replique deux entites d'equipe, aux slots 6 et 8. Rien dans le film ne dit
// laquelle est `team_0` et laquelle est `team_1` du registre : l'ordre des slots n'est pas
// l'ordre des camps, et le supposer colorerait la courbe du mauvais camp — une erreur que
// personne ne verrait a l'ecran, puisque les deux courbes existent bien.
//
// # LES TROIS PREUVES, DANS L'ORDRE DE LEUR FORCE (D3 du plan registre-film, (a0) au lot M5.1)
//
//	(a) LE SCORE FINAL. Quand `team_0_score` et `team_1_score` DIFFERENT, le score final de
//	    chaque slot les designe sans ambiguite. C'est la preuve la plus forte, et la seule qui
//	    n'emprunte rien au pont d'identite des joueurs.
//	(a0) LE MATCH A SENS UNIQUE (schema 69, lot M5.1 des retours rejeu du 2026-09-23). Un
//	    SEUL slot d'equipe porte une serie de score : le camp muet n'a jamais quitte zero (le
//	    statborg n'emet un composant qu'a son changement). Le score absent vaut donc 0, et
//	    quand le registre dit X-0 (X > 0) avec une serie qui finit EXACTEMENT a X, cette serie
//	    est le camp X. Garde-fou : une serie a 2 contre un registre a 3 (film tronque avant la
//	    derniere capture) ne prouve rien et reste `unresolved`.
//	(b) LA SOMME DES FRAGS. A egalite de scores (ou sans scores du tout), le slot d'equipe
//	    porte en `comp 2 A` le total de frags de son camp — acquis mesure a la phase 0-bis du
//	    lot A. On le compare a la somme des frags des joueurs IDENTIFIES de chaque camp.
//	(c) AUCUNE DES TROIS : refus explicite. Les courbes sortent sans `teamId`.
//
// # LA REGLE DE PRUDENCE EST LA MEME QU'AILLEURS
//
// Chaque preuve exige une correspondance UNIQUE et des valeurs DISTINCTES. Deux camps a
// egalite de frags ne prouvent rien : les apparier revient a tirer a pile ou face, et se
// taire vaut mieux que colorer au hasard.

// resolveTeamIdentity applique D3 et rend (slot d'entite -> camp, methode retenue).
//
// Une carte vide avec la methode `unresolved` n'est PAS un echec du calque : les courbes sont
// publiees, seulement sans camp.
func resolveTeamIdentity(in *ScoreInput, slots []int, score, teamFrags, playerFrags scoreSeriesSet,
	identity map[int]string) (map[int]int, string) {
	if in == nil || len(slots) == 0 || len(slots) > 2 {
		return nil, ScoreIdentityUnresolved
	}
	// UN SEUL SLOT VU : le camp muet n'a rien emis du tout, ni score ni frags. Seule (a0) peut
	// trancher — (a) et (b) exigent les deux slots.
	if len(slots) == 1 {
		if m := identityByOneSidedScore(in.TeamScores, slots, score); m != nil {
			return m, ScoreIdentityFinalOneSided
		}
		return nil, ScoreIdentityUnresolved
	}
	if m := identityByFinalScore(in.TeamScores, slots, score); m != nil {
		return m, ScoreIdentityFinal
	}
	if m := identityByOneSidedScore(in.TeamScores, slots, score); m != nil {
		return m, ScoreIdentityFinalOneSided
	}
	if m := identityByFrags(in.TeamByXUID, slots, teamFrags, playerFrags, identity); m != nil {
		return m, ScoreIdentityFrags
	}
	return nil, ScoreIdentityUnresolved
}

// identityByFinalScore est la preuve (a) : le score final de chaque slot contre les scores du
// registre, quand ceux-ci different.
//
// L'EGALITE EST EXACTE, jamais approchee. Un ecart d'un point signifie que le film et le
// registre ne mesurent pas la meme grandeur (c'est le cas de Strongholds et de KOTH, ou l'API
// compte des ticks et des secondes de colline) : dans ce cas la preuve ne s'applique pas, et
// c'est (b) qui doit trancher.
func identityByFinalScore(scores *[2]int, slots []int, score scoreSeriesSet) map[int]int {
	if scores == nil || scores[0] == scores[1] {
		return nil
	}
	a, okA := score.final(slots[0])
	b, okB := score.final(slots[1])
	if !okA || !okB {
		return nil
	}
	switch {
	case a == int64(scores[0]) && b == int64(scores[1]):
		return map[int]int{slots[0]: 0, slots[1]: 1}
	case a == int64(scores[1]) && b == int64(scores[0]):
		return map[int]int{slots[0]: 1, slots[1]: 0}
	}
	return nil
}

// identityByOneSidedScore est la preuve (a0) : UNE SEULE serie de score parmi les slots
// d'equipe, un registre a X-0 (X > 0), et une serie qui finit EXACTEMENT a X.
//
// POURQUOI « ABSENT VAUT ZERO » EST UNE LECTURE ET PAS UN PARI. Le statborg n'emet un composant
// qu'a son CHANGEMENT : un camp qui ne marque jamais ne publie jamais de score de mode, alors
// que son slot existe (il porte ses frags). C'est la regle que le client applique deja au
// bandeau (en-tete de `lib/replay/scoreTimeline.ts`, temoin CTF 3-0 `530820e5`) ; elle est
// ecrite ici pour que le DOCUMENT la porte, au lieu de laisser chaque lecteur la re-deduire.
//
// LE REGISTRE DOIT DIRE ZERO POUR LE CAMP MUET. Un registre 1-3 avec une seule serie a 3 ne
// prouve rien : le camp du registre a 1 a marque, et son absence du film est un TROU de lecture,
// pas un zero — on se tait. Et l'egalite reste EXACTE, comme en (a) : une serie a 2 contre un
// registre a 3 (film tronque avant sa derniere capture) ne designe aucun camp.
func identityByOneSidedScore(scores *[2]int, slots []int, score scoreSeriesSet) map[int]int {
	if scores == nil || scores[0] == scores[1] {
		return nil
	}
	porteur, n := -1, 0
	for _, slot := range slots {
		if _, ok := score.final(slot); ok {
			porteur, n = slot, n+1
		}
	}
	if n != 1 {
		return nil
	}
	v, _ := score.final(porteur)
	var marque int
	switch {
	case scores[1] == 0 && scores[0] > 0 && v == int64(scores[0]):
		marque = 0
	case scores[0] == 0 && scores[1] > 0 && v == int64(scores[1]):
		marque = 1
	default:
		return nil
	}
	// LE SLOT MUET N EST PAS DANS LA CARTE (revue adverse M5, constat R4, 2026-09-24) : son camp
	// s en deduirait (1 - marque), mais aucun lecteur ne le lirait — [buildTeamScores] ne publie
	// que les slots qui portent une serie, et le slot muet n en a pas par definition.
	return map[int]int{porteur: marque}
}

// identityByFrags est la preuve (b) : `comp 2 A` du slot d'equipe contre la somme des frags
// des joueurs identifies de chaque camp.
//
// Elle EMPRUNTE au pont d'identite des joueurs (le triplet frags/morts/assistances), donc elle
// ne vaut que si ce pont a apparie des joueurs des DEUX camps. Sans cela, une seule somme
// existe et rien ne la distingue.
func identityByFrags(teamByXUID map[string]int, slots []int, teamFrags, playerFrags scoreSeriesSet,
	identity map[int]string) map[int]int {
	camp := fragsByTeam(teamByXUID, playerFrags, identity)
	if len(camp) != 2 {
		return nil
	}
	ids := make([]int, 0, len(camp))
	for t := range camp {
		ids = append(ids, t)
	}
	sort.Ints(ids)
	if camp[ids[0]] == camp[ids[1]] {
		return nil // deux camps a egalite de frags : l'appariement ne prouve rien
	}
	out := make(map[int]int, len(slots))
	for _, slot := range slots {
		v, ok := teamFrags.final(slot)
		if !ok {
			return nil
		}
		found, n := 0, 0
		for _, t := range ids {
			if camp[t] == v {
				found, n = t, n+1
			}
		}
		if n != 1 {
			return nil
		}
		out[slot] = found
	}
	if out[slots[0]] == out[slots[1]] {
		return nil // les deux slots designent le meme camp : lecture incoherente, on se tait
	}
	return out
}

// fragsByTeam somme les frags des joueurs IDENTIFIES, camp par camp.
func fragsByTeam(teamByXUID map[string]int, playerFrags scoreSeriesSet, identity map[int]string) map[int]int64 {
	out := map[int]int64{}
	for slot, xuid := range identity {
		team, ok := teamByXUID[xuid]
		if !ok {
			continue
		}
		v, ok := playerFrags.final(slot)
		if !ok {
			continue
		}
		out[team] += v
	}
	return out
}
