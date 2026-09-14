package replay

// roster_xuids.go — LA REGLE QUI FAIT UN ROSTER D APPOINT A PARTIR D UNE FEUILLE DE MATCH.
//
// # POURQUOI ELLE VIT ICI (lot 1.0, revue R1, constat R1-1)
//
// Elle vivait dans `replaybuild.rosterXUIDs`, avec le seul appelant de production. Le fixture
// d entrees, lui, passait `RosterXUIDs` NUL : `rosterOf(deaths, nil)` au fixture contre
// `rosterOf(deaths, feuille)` en production, donc un joueur a ZERO MORT manquait a la table
// d index du golden — et la fidelite ne pouvait rien en dire, puisqu elle compare deux
// assemblages batis sur les MEMES options.
//
// La regle descend donc dans le paquet qui PORTE le champ (`Options.RosterXUIDs`), ou les deux
// chemins peuvent l appeler : `replaybuild` depuis `port.MatchFacts`, le fixture depuis le
// `<short8>.facts.json` du corpus d equivalence. Une regle, un site. Le sens de lecture est le
// bon : `replaybuild` importe `replay`, jamais l inverse (`port` depend de `replay`).

import "strconv"

// RosterXUIDsOf rend les joueurs d une feuille de match, en decimal, pour COMPLETER le roster
// que le fil des morts donne au rejeu (cf. [Options.RosterXUIDs]).
//
// UN JOUEUR QUI NE MEURT JAMAIS N EST DANS AUCUNE MORT, donc dans aucun roster deduit du fil —
// et il disparait de toute la chaine : pas d index de joueur, pas de pont, pas d entree au
// roster publie. Mesure du 2026-09-07 sur `3372e7eb` : 6 joueurs publies pour 8 a la feuille,
// les deux manquants a 0 mort.
//
// Un xuid que la feuille ne donne pas en decimal (un bot, `bid(N.0)`) est IGNORE : le pont des
// bots passe par BOT_METADATA et les relais, pas par l index de joueur.
//
// AUCUN joueur exploitable rend nil, et non une tranche vide : `nil` est ce que les appelants
// hors ligne passent deja, et c est ce qui garde le rejeu publiable sans base.
func RosterXUIDsOf(xuids []string) []uint64 {
	out := make([]uint64, 0, len(xuids))
	for _, s := range xuids {
		if x, err := strconv.ParseUint(s, 10, 64); err == nil && x != 0 {
			out = append(out, x)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
