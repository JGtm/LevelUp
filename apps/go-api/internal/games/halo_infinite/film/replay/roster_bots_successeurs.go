package replay

// roster_bots_successeurs.go — LE BOT QUI SUCCEDE A UN HUMAIN SUR SON INDEX A SON ENTREE DE ROSTER
// (revue adverse du lot M2, constat M2-R1, 2026-09-24).
//
// # LE DEFAUT
//
// `buildRoster` ecarte le bot dont l'index est tenu par un humain : « les deux declarations se
// contredisent ». Or le film ecrit le cas contraire : un bot tient un index au coup d'envoi, un
// humain le reprend (`4f77afc1`, index 26 : `343 Doomfruit` puis Narotlcs — Q23), ou un humain
// part et un bot prend son index. Le bot ecarte n'avait donc PAS d'entree, mais ses vies etaient
// nommees (par l'entite de son corps, ou par le pont) : le web lui dessinait une place a lui
// (`joueur:bot:<nom>`), vide tout le match hors de ses vies — une fiche de plus que de places, et
// le remplacant hors de la place du partant.
//
// UNE AUTRE CAUSE DONNE LE MEME SYMPTOME, ET ELLE N'EST PAS ICI : un bot que le kill-feed n'epingle
// pas (`killsource` le declare « non epingle » quand son index est inferieur au nombre d'humains —
// `c75f33b8`, `343 Robot Hoida` sur l'index 8 avec dix humains) n'atteint pas le rejeu
// (`replayidentity.BotIdentities` l'ecarte) ; nomme par le relais de la base, il reste hors roster.
// Il se COMPTE (`identitesHorsRoster`) et le web ne lui rend aucune tuile ; sa correction est
// celle de l'epinglage, hors de ce lot.
//
// # LA LECTURE
//
// Les deux occupants ne se contredisent que s'ils sont SIMULTANES. Le film dit s'ils le sont : un
// occupant a SON entite `ti=9`, jamais reutilisee (lot 1.7). Le bot entre au roster quand, sur son
// index, les entites stables se partagent en deux groupes DISJOINTS dans le temps :
//
//	les SIENNES   celles que ses declarations BOT_METADATA croisent (fenetre STRICTE) ;
//	les AUTRES    au moins une, que ses declarations ne croisent pas — l'occupant humain ;
//
// et aucune des siennes ne partage une image-cle porteuse avec une des autres. Deux entites d'un
// meme index a la MEME image-cle sont deux occupants simultanes : c'est la contradiction d'origine,
// et le bot reste dehors. Sans balayage des entites (faits anterieurs, chemin sans ti=9), ou sans
// declarations datees, rien ne se lit : la regle d'avant tient.
//
// Ce qu'une entree de bot admise devient ensuite n'a rien de particulier : la liaison
// (occupants.go) lui donne ses entites, son equipe et sa presence ; la pose des places (sieges.go)
// l'assoit — sur la place du partant quand son index est un siege de la table (lecture `lu`, borne
// au successeur), par chainage sinon. Un bot que cette lecture laisse dehors mais dont une vie est
// nommee se COMPTE (`coverage.seats.identitesHorsRoster`).

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// rosterDesOccupants est le roster publie : celui de [buildRoster], plus les bots successeurs
// d'un humain que le film montre (cf. l'en-tete). Rend aussi le nombre de ces bots.
func rosterDesOccupants(idx PlayerIndexTable, names map[uint64]string, bots []BotIdentity,
	scan grammar.PlayerEntityScan, equipes teamPublication) ([]RosterEntry, int) {
	return admettreLesBotsSuccesseurs(buildRoster(idx, names, bots, equipes), bots, scan, equipes)
}

// admettreLesBotsSuccesseurs ajoute au roster les bots que [buildRoster] a ecartes parce que leur
// index est tenu par un humain, quand le film les montre SUCCESSIFS de cet humain (cf. l'en-tete).
// Rend le roster, trie comme [buildRoster] le trie, et le nombre de bots admis.
func admettreLesBotsSuccesseurs(roster []RosterEntry, bots []BotIdentity, scan grammar.PlayerEntityScan,
	equipes teamPublication) ([]RosterEntry, int) {
	if !scan.Scanned || len(bots) == 0 {
		return roster, 0
	}
	humains, presents := map[int]bool{}, map[string]bool{}
	for _, e := range roster {
		if !e.Bot {
			humains[e.FilmIndex] = true
		} else {
			presents[e.Name] = true
		}
	}
	admis := 0
	for _, b := range bots {
		if !humains[b.FilmIndex] || b.Name == "" || presents[b.Name] || !botSuccesseurLu(scan, b) {
			continue
		}
		presents[b.Name] = true
		e := RosterEntry{FilmIndex: b.FilmIndex, Name: b.Name, Bot: true, Bid: b.Bid()}
		e.Team = equipes.equipeDuRoster(e)
		roster = append(roster, e)
		admis++
	}
	if admis > 0 {
		sort.SliceStable(roster, func(i, j int) bool {
			if roster[i].FilmIndex != roster[j].FilmIndex {
				return roster[i].FilmIndex < roster[j].FilmIndex
			}
			if roster[i].XUID != roster[j].XUID {
				return roster[i].XUID < roster[j].XUID
			}
			return roster[i].Name < roster[j].Name
		})
	}
	return roster, admis
}

// botSuccesseurLu dit que, sur l'index du bot, ses entites et celles d'un autre occupant sont
// DISJOINTES dans le temps (cf. l'en-tete). Faux sans declaration datee.
func botSuccesseurLu(scan grammar.PlayerEntityScan, b BotIdentity) bool {
	if len(b.Declarations) == 0 {
		return false
	}
	var siennes, autres []grammar.PlayerEntity
	for _, e := range scan.Entities {
		if e.Index != b.FilmIndex || e.Unstable {
			continue
		}
		if entiteDeclareeParLeBot(scan, e, b.Declarations) {
			siennes = append(siennes, e)
		} else {
			autres = append(autres, e)
		}
	}
	if len(siennes) == 0 || len(autres) == 0 {
		return false
	}
	for _, s := range siennes {
		for _, a := range autres {
			if s.FirstKF <= a.LastKF && a.FirstKF <= s.LastKF {
				return false // deux occupants a la meme image-cle : la contradiction d'origine
			}
		}
	}
	return true
}
