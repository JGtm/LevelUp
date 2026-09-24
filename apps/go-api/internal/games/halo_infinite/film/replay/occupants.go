package replay

// occupants.go — QUI OCCUPE LE MATCH, QUAND, ET DANS QUELLE EQUIPE : LU DANS LE FILM (lot M2.3 de
// la campagne « retours rejeu », 2026-09-23 ; decision Q24 : option A).
//
// # CE QUE CE FICHIER LIT, ET D'OU
//
// Trois lectures du film, aucune de la base :
//
//	les ENTITES `ti=9`   un occupant par entite, jamais reutilisee ; sa presence au pas des
//	                     images-cles et son designateur d'equipe (grammar/player_entities.go) ;
//	BOT_METADATA         l'instant ou chaque bot est declare puis retire, a la frame pres
//	                     (`BotIdentity.Declarations`) ;
//	les VIES nommees     un corps qui bouge prouve la presence de son occupant.
//
// # LE LIEN ENTREE -> ENTITE (sonde P4)
//
// Un index de joueur n'est PAS un occupant : sur `b1ad85eb` trois bots de deux equipes se relaient
// sur l'index 8. Chaque entree du roster se lie donc aux entites de SON index par le temps :
//
//	un BOT    a l'entite dont la fenetre STRICTE (premiere .. derniere image-cle) croise une de
//	          ses declarations BOT_METADATA ;
//	un HUMAIN aux entites restantes de son index dont la fenetre LARGE (image-cle voisine exclue
//	          de chaque cote) contient le debut d'une de ses vies ; sans vie, a l'unique entite
//	          restante de son index quand il est le seul humain a le porter.
//
// Une entite que deux entrees revendiquent n'est liee a personne (comptee, `entitesContestees`),
// une entite qu'aucune ne revendique non plus (`entitesNonLiees`) : rien ne se devine.
//
// # LA PRESENCE, ET SES DEUX BORNES DE FIN
//
// Une presence est `[de, a]` CERTAINE, prolongee jusqu'a `aMax` pour l'AFFICHAGE : l'entite ne
// se voit qu'aux images-cles (~20 s), donc un depart n'est borne que par la premiere image-cle
// ou l'entite n'est plus la (decision Q22 : la tuile sort la). Un bot dont BOT_METADATA date le
// retrait a `aMax` exact. Un occupant lu a la premiere image-cle porteuse est la depuis la
// frame 0 ; lu a la derniere, jusqu'a la fin. Un trou d'entite n'est pas un depart.
//
// SANS ENTITE LUE (registre sans ti=9, faits anterieurs, chemin `BuildFromPositions`), la
// presence retombe sur l'enveloppe des vies — c'est un REPLI, nomme et compte par la pose des
// places (cf. sieges.go), qui y ajoute la regle d'avant « le dernier occupant d'une place reste
// jusqu'a la fin » (mourir n'est pas partir, et sans entite on ne sait pas qui est parti).
//
// SUR UN FILM BALAYE, UNE ENTREE QU'AUCUNE ENTITE NE PORTE (entite contestee, instable, occupant
// present moins d'une image-cle) a le meme repli, PAR ENTREE, et il se compte a part (revue
// M2-R5, `coverage.seats.presencesParLesVies`) : son affichage court jusqu'a la veille de la
// premiere image-cle porteuse qui suit sa derniere vie — l'instant ou son entite, s'il en avait
// une, aurait dit qu'il n'est plus la (Q22) —, jusqu'au bout s'il n'y en a plus. Mourir n'y est
// donc pas partir : un mort qui attend sa reapparition garde sa fiche jusqu'a l'image-cle.

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// intervalleDePresence est une presence en FRAMES, bornes incluses : `[de, a]` est certain,
// `[de, aMax]` est ce qui s'affiche (aMax >= a).
type intervalleDePresence struct {
	de, a, aMax int
}

// occupantDuRoster est ce que la publication sait d'UNE entree de roster.
type occupantDuRoster struct {
	// entites : indices, dans le balayage, des entites liees a cette entree.
	entites []int
	// equipe : l'equipe PAR ENTREE (designateur des entites liees, a l'unanimite) ; nil = silence.
	equipe *int
	// vies : les vies nommees de l'entree, en frames, dans l'ordre du temps.
	vies [][2]int
	// presence : ses intervalles de presence, dans l'ordre du temps.
	presence []intervalleDePresence
	// lue : la presence vient d'une entite ou de BOT_METADATA, pas des seules vies.
	lue bool
}

// occupants est le resultat de la liaison : une entree par entree de roster, dans le meme ordre.
type occupants struct {
	parEntree []occupantDuRoster
	// balaye : les entites ont ete lues (`PlayerEntityScan.Scanned`). Faux = repli sur les vies.
	balaye bool
	// entitesNonLiees / entitesContestees / equipesDivergentes : ce que la liaison n'a pas su
	// poser, compte.
	entitesNonLiees, entitesContestees, equipesDivergentes int
	// trous : les trous d'entite (images-cles manquees entre la premiere et la derniere).
	trous int
	// simultanees : par designateur, le plus grand nombre d'entites stables lues a une meme
	// image-cle porteuse (cf. [entitesSimultanees]). Nil sans balayage.
	simultanees map[int]int
	// horloge : la grille du document — la liaison compare des debuts de vie (frames) a des
	// fenetres d'entite (microsecondes de film).
	horloge replayClock
	// horsRoster : les identites qui nomment au moins une vie publiee et qu'aucune entree du roster
	// ne porte (revue M2-R1). Chacune est un occupant sans place ni presence : le web ne lui rend
	// aucune tuile (la regle des places prime), et ce compteur la rend visible. 0 attendu.
	horsRoster int
}

// entreesDesOccupants porte ce que la liaison consomme hors du roster et des pistes. Une
// structure plutot que quatre parametres de plus : le depot borne a cinq.
type entreesDesOccupants struct {
	scan grammar.PlayerEntityScan
	bots []BotIdentity
	// horloge : la grille du document (origine, pas, nombre de frames).
	horloge replayClock
	// parIndex : la table de CONTROLE `index -> designateur`. Elle ne sert qu'aux entrees qu'aucune
	// entite ne porte, et seulement sur un index que toutes ses lectures accordent.
	parIndex map[int]int
}

// lierLesOccupants lie chaque entree du roster a ses entites, et en tire son equipe et sa
// presence. PURE : ni octet de film, ni base.
func lierLesOccupants(roster []RosterEntry, tracks []Track, in entreesDesOccupants) occupants {
	out := occupants{parEntree: make([]occupantDuRoster, len(roster)), balaye: in.scan.Scanned,
		horloge: in.horloge}
	vies := viesParIdentite(tracks)
	portees := make(map[string]bool, len(roster))
	for i := range roster {
		out.parEntree[i].vies = vies[cleDeRoster(roster[i])]
		portees[cleDeRoster(roster[i])] = true
	}
	for cle := range vies {
		if !portees[cle] {
			out.horsRoster++
		}
	}
	if out.balaye {
		out.trous = in.scan.Holes()
		out.simultanees = entitesSimultanees(in.scan)
		out.lierLesEntites(roster, in)
	}
	for i := range roster {
		out.parEntree[i].equipe = out.equipeDe(roster[i], i, in)
		out.parEntree[i].presence, out.parEntree[i].lue = presenceDe(roster[i], out.parEntree[i], in)
	}
	return out
}

// imagesClesDUnOccupantDurable : une entite ne compte dans [entitesSimultanees] que lue a au moins
// DEUX images-cles porteuses (revue M2-R6, 2026-09-24). Une entite d'une seule image-cle peut etre
// un RELAIS TRANSITOIRE : sur `43e96765`, le bot `343 PardonMy`, declare 21 s, est lu a la meme
// image-cle que l'humain qui le remplace — cinq entites pour une equipe de quatre, et une capacite
// estimee a 5 qui aurait laisse ouvrir une cinquieme place. Ce n'est pas un seuil mesure : c'est le
// plus petit nombre de lectures qui distingue un occupant d'un relais.
const imagesClesDUnOccupantDurable = 2

// entitesSimultanees rend, par designateur, le plus grand nombre d'entites STABLES et DURABLES
// (cf. [imagesClesDUnOccupantDurable]) lues a une meme image-cle porteuse, trous compris (un trou
// n'est pas un depart) : le nombre d'occupants que le JEU a tenus ensemble dans cette equipe — une
// borne basse LUE de sa taille, qui ne doit rien a la liaison des entrees (une liaison fausse ne la
// fait pas monter).
func entitesSimultanees(scan grammar.PlayerEntityScan) map[int]int {
	out := map[int]int{}
	for r := range scan.KeyframesUS {
		ici := map[int]int{}
		for _, e := range scan.Entities {
			if !e.Unstable && e.Seen >= imagesClesDUnOccupantDurable && e.FirstKF <= r && r <= e.LastKF {
				ici[e.Team]++
			}
		}
		for t, n := range ici {
			out[t] = max(out[t], n)
		}
	}
	return out
}

// viesParIdentite rend, par identite de roster, les fenetres de ses vies publiees, triees.
func viesParIdentite(tracks []Track) map[string][][2]int {
	out := map[string][][2]int{}
	for _, tr := range tracks {
		if cle := cleDePiste(tr); cle != "" {
			out[cle] = append(out[cle], [2]int{tr.StartFrame, tr.EndFrame})
		}
	}
	for cle, vs := range out {
		sort.Slice(vs, func(a, b int) bool { return vs[a][0] < vs[b][0] })
		out[cle] = vs
	}
	return out
}

// lierLesEntites pose le lien entree -> entites (cf. l'en-tete) et compte ce qui reste.
func (o *occupants) lierLesEntites(roster []RosterEntry, in entreesDesOccupants) {
	revendications := make([][]int, len(in.scan.Entities)) // entite -> entrees qui la revendiquent
	for i, e := range roster {
		if e.Bot {
			for _, k := range entitesDuBot(in.scan, e.FilmIndex, declarationsDe(e, in.bots)) {
				revendications[k] = append(revendications[k], i)
			}
		}
	}
	// LES HUMAINS NE VOIENT QUE CE QU'AUCUN BOT NE REVENDIQUE, et ils revendiquent TOUS avant
	// qu'on tranche : deux humains qui designent la meme entite la rendent contestee, jamais
	// « au premier arrive ».
	prisParUnBot := make([]bool, len(revendications))
	for k, qui := range revendications {
		prisParUnBot[k] = len(qui) > 0
	}
	for i, e := range roster {
		if e.Bot {
			continue
		}
		for _, k := range o.entitesDeLHumain(roster, i, in.scan, prisParUnBot) {
			revendications[k] = append(revendications[k], i)
		}
	}
	for k, qui := range revendications {
		switch {
		case in.scan.Entities[k].Unstable || len(qui) == 0:
			o.entitesNonLiees++
		case len(qui) > 1:
			o.entitesContestees++
		default:
			o.parEntree[qui[0]].entites = append(o.parEntree[qui[0]].entites, k)
		}
	}
}

// entitesDeLHumain rend les entites de l'index d'un humain que ses vies designent (fenetre
// LARGE contenant le debut d'une vie). Sans vie : l'unique entite encore libre de son index,
// s'il est le seul humain a le porter.
func (o *occupants) entitesDeLHumain(roster []RosterEntry, i int, scan grammar.PlayerEntityScan,
	prisParUnBot []bool) []int {
	idx := roster[i].FilmIndex
	libres := []int{}
	for k, ent := range scan.Entities {
		if ent.Index == idx && !ent.Unstable && !prisParUnBot[k] {
			libres = append(libres, k)
		}
	}
	vies := o.parEntree[i].vies
	if len(vies) == 0 {
		if len(libres) == 1 && humainsDeLIndex(roster, idx) == 1 {
			return libres
		}
		return nil
	}
	var out []int
	for _, k := range libres {
		f := fenetreLargeDe(scan, scan.Entities[k])
		for _, v := range vies {
			if f.contient(instantDeFrame(o.horloge, v[0])) {
				out = append(out, k)
				break
			}
		}
	}
	return out
}

// humainsDeLIndex compte les entrees humaines qui portent cet index.
func humainsDeLIndex(roster []RosterEntry, idx int) int {
	n := 0
	for _, e := range roster {
		if !e.Bot && e.FilmIndex == idx {
			n++
		}
	}
	return n
}

// equipeDe rend l'equipe PAR ENTREE : le designateur de ses entites a l'unanimite ; sans entite
// liee, la table de CONTROLE sur un index non divergent (la meme valeur que l'ancienne equipe
// par index, donc aucune entree ne perd l'equipe qu'elle avait).
func (o *occupants) equipeDe(e RosterEntry, i int, in entreesDesOccupants) *int {
	ents := o.parEntree[i].entites
	if len(ents) == 0 {
		if t, ok := in.parIndex[e.FilmIndex]; ok && !o.balaye {
			return &t
		}
		if t, ok := in.parIndex[e.FilmIndex]; ok && !indexPorteParUneEntiteLiee(o, in.scan, e.FilmIndex) {
			return &t
		}
		return nil
	}
	t := in.scan.Entities[ents[0]].Team
	for _, k := range ents[1:] {
		if in.scan.Entities[k].Team != t {
			o.equipesDivergentes++
			return nil
		}
	}
	return &t
}

// indexPorteParUneEntiteLiee dit qu'une entite de cet index est deja liee a une AUTRE entree :
// sa couleur est alors celle d'un autre occupant, et la table de controle ne la preterait pas.
func indexPorteParUneEntiteLiee(o *occupants, scan grammar.PlayerEntityScan, idx int) bool {
	for _, occ := range o.parEntree {
		for _, k := range occ.entites {
			if scan.Entities[k].Index == idx {
				return true
			}
		}
	}
	return false
}
