package killsource

// roster_espace_humains.go — L ESPACE DES HUMAINS, LU DANS LE FILM (lot J7.2 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, constat FK-1).
//
// # LE DEFAUT QUE CE FICHIER FERME
//
// [roster.pinBots] n epingle un bot que si son slot tombe AU-DELA de l espace des humains. Cette
// borne valait le NOMBRE DE NOMS du kill-feed. Or le kill-feed nomme tous ceux qui ont tue ou sont
// morts, remplacants compris, alors qu un match a des PLACES FINIES — un partant libere la sienne,
// son remplacant ne cree pas de place. Chaque remplacant repoussait donc la borne, et le bot de
// relais arrive entre-temps (slot 8 sur `b1ad85eb`, apres les huit sieges de depart) tombait
// « dans l espace des humains » : desepingle en silence, ses morts et celles qu il inflige perdues,
// ses assistances publiees en `?N`.
//
// # LA REGLE
//
// La table de `chunk_00` est ecrite a l OUVERTURE du film : ses sieges sont les PLACES du match, et
// c est leur NOMBRE qui borne l espace des humains. Un bot n est DESEPINGLE que si DEUX conditions
// tiennent ([roster.contreditLaTable]) : son slot tombe sous cette borne, ET la table NOMME un humain
// a ce siege. Il contredit alors la table : il n est pas epingle, il est journalise par film et
// compte ([Coverage.BotsNonEpingles]).
//
//	SIEGE VACANT INTERCALE (revue adverse du lot J7 ; 13 films du cache, dont `111fa685`,
//	`a521164d`, `11de8353`) : sous la borne, mais aucun humain n y siege — rien ne contredit le
//	bot, il reste epingle. La borne seule le perdait.
//	AU-DELA DE LA BORNE : le bot reste epingle, meme sur un indice que la table nomme aussi — c est
//	alors la contradiction entre deux lectures que [roster.pinUnSiege] compte (`BotConflict`),
//	BOT_METADATA gardant la main. La borne restreint la regle du siege nomme, elle ne la remplace pas.
//
// SANS TABLE LUE, LE FILM NE DONNE AUCUNE BORNE, et le kill-feed n en tient pas lieu : aucune
// contradiction ne peut alors etre etablie, et BOT_METADATA — la lecture la plus ancienne et la plus
// eprouvee (RE_LOG 7ter.62) — epingle tous ses bots. Le refus de la table est deja nomme, journalise
// par [decodeCtx.prepare] et publie ([FilmTablePinning.Refusal]).

// borneDesHumainsDuFilm rend la borne de l espace des humains que la table du film donne, zero
// quand la table n est pas lue.
func borneDesHumainsDuFilm(t FilmTable) int {
	if !t.Lue() {
		return 0
	}
	return len(t.Seats)
}

// siegesNommes : les sieges que la table NOMME, vide quand elle n est pas lue.
func siegesNommes(t FilmTable) map[int]bool {
	out := make(map[int]bool, len(t.Seats))
	if !t.Lue() {
		return out
	}
	for idx := range t.Seats {
		out[idx] = true
	}
	return out
}

// contreditLaTable : le slot d un bot tombe dans l espace des humains ET sur un siege qu un humain
// tient d apres la table.
func (r *roster) contreditLaTable(slot int) bool {
	return slot < r.borneHumains && r.siegesHumains[slot]
}
