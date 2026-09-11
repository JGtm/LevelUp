package replay

// flag_carries_home.go — LE DRAPEAU QUI RENTRE CHEZ LUI FERME LE PORTAGE (lot 6.11, item 2).
//
// # LE DEFAUT QUE CE FICHIER FERME, ET SA MESURE
//
// Un joueur qui LACHE puis REPREND le meme drapeau produit deux prises successives du meme slot.
// La seconde borne la premiere (`nextOpeningOfSlot`), mais le lacher qui les separe n'est date
// par rien quand aucune vie libre ne nait aux pieds du porteur : tout le sejour AU SOL entre les
// deux est alors publie comme du portage. Cas mesure au lot 6.7-B1 et reste ouvert :
// `16ea3668` / 2535417044536883, un unique span de 55,1 s pour 0,9 s a l'oracle — il vole le
// drapeau, le lache aussitot, et le RE-VOLE 55 s plus tard.
//
// # LE CANAL QUI LES DATE : LE DRAPEAU EST RENTRE ENTRE-TEMPS
//
// Un VOL (`flag_steals`) se fait AU SOCLE (cf. flag_assign.go) : pour que ce joueur re-vole le
// drapeau, il a fallu que celui-ci RENTRE CHEZ LUI. Et un drapeau chez lui n'est dans la main de
// personne. Deux chaines DATENT cette rentree, et toutes deux NOMMENT leur drapeau :
//
//	le RETOUR CREDITE      `flag_returns` — credite au joueur qui rend SON drapeau, jamais celui
//	                       de l'adversaire : l'EQUIPE du rendeur nomme le drapeau rendu ;
//	la RENTREE DE L'OBJET  le drapeau est RE-CREE a son socle, et le socle le nomme
//	                       (`flagObjectHomecomings`) — le retour AUTOMATIQUE, que personne ne
//	                       provoque et qu'aucun compteur ne credite.
//
// Les deux etaient deja lues — elles servent l'etat du sol dans `flag_carries_lives.go` — et
// AUCUNE ne fermait un portage : le drapeau rentrait chez lui pendant qu'un joueur etait cense
// courir avec. Comme les autres chaines du calque, celle-ci ne retient qu'un instant STRICTEMENT
// interieur a `]t0, t1[` : elle ne peut que RACCOURCIR.
//
// # L'ABSTENTION DE LA RENTREE, ET POURQUOI ELLE EST PLUS STRICTE ICI
//
// Une naissance au socle du drapeau F peut aussi etre le drapeau ADVERSE tombe au pied de ce
// socle — l'ambiguite que `applyFlagHomecoming` ecarte par la POSITION des laches. Ici les
// positions n'existent pas encore (le bornage precede `attachFlagCarryPositions`), donc le refus
// se fait sur le TEMPS : une rentree dont un portage d'un AUTRE drapeau s'acheve dans la meme
// seconde n'est pas retenue. On se tait plutot que de fermer sur la chute d'un autre drapeau.
//
// # CE QUI RESTE NON BORNE APRES CE FICHIER
//
// Le lacher suivi d'une reprise PAR LE MEME JOUEUR sur un drapeau qui n'est jamais rentre entre
// les deux — repris AU SOL (`flag_grabs`) — et dont l'objet n'a pas replique sa naissance aux
// pieds du porteur. Aucune chaine du film ne le date : le marqueur d'image-cle le pourrait, mais
// il est le CONTROLE INDEPENDANT du calque (`flag_carries_marker.go`), et s'en servir comme
// source le rendrait tautologique. Le residu est chiffre au rapport du lot.

import (
	"sort"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// flagHomeGuardMS — l'ecart, en millisecondes, sous lequel la fin d'un portage d'un AUTRE
// drapeau rend une RENTREE ambigue. C'est [flagFreeDropWindowMS], la fenetre deja utilisee pour
// apparier un lacher a la vie libre qui l'explique : aucun seuil neuf n'est introduit ici.
const flagHomeGuardMS = flagFreeDropWindowMS

// flagOfOwner rend l'index du drapeau dont l'equipe `team` est PROPRIETAIRE — celui qu'elle
// defend et qu'elle seule peut renvoyer. Un seul drapeau en jeu : c'est celui-la. Faux quand
// l'equipe n'en possede pas exactement un.
func flagOfOwner(spawns []FlagSpawn, team int) (int, bool) {
	if flagSingleInPlay(spawns) {
		return 0, true
	}
	found := -1
	for i := range spawns {
		if spawns[i].Team != team {
			continue
		}
		if found >= 0 {
			return -1, false
		}
		found = i
	}
	return found, found >= 0
}

// flagReturns rend, TRIES, TOUS les retours credites du film, chacun avec le drapeau que
// l'EQUIPE de son auteur nomme — ou -1 quand elle ne le nomme pas (equipe non lue, carte a plus
// d'un socle par camp). Un retour non nomme ne ferme rien : fermer au hasard serait pire que se
// taire, et l'etat du sol garde pour lui sa regle historique (`applyFlagReturn`).
func flagReturns(scan FlagCarryScan) []flagHomecoming {
	var out []flagHomecoming
	for _, e := range scan.Events {
		if e.Stat != objectiveevents.StatFlagReturns {
			continue
		}
		f := -1
		if team, known := scan.TeamOf[scan.Identity.At(e.Slot, e.TimeMS)]; known {
			if owned, named := flagOfOwner(scan.Spawns, team); named {
				f = owned
			}
		}
		out = append(out, flagHomecoming{flag: f, at: int64(e.TimeMS)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].at < out[j].at })
	return out
}

// closeByHomecoming ferme chaque portage au PREMIER instant ou SON drapeau est rentre chez lui.
// Rend les deux comptes, PAR CHAINE — le retour credite et la rentree de l'objet — parce que
// leur provenance n'a pas la meme force : l'un est un fait credite a un joueur, l'autre une
// lecture de l'objet.
func closeByHomecoming(raws []flagCarryRaw, scan FlagCarryScan,
	ctx flagCarryCtx) (byReturn, byObject int) {
	retours := flagReturns(scan)
	rentrees := flagObjectHomecomings(scan, ctx)
	if len(retours) == 0 && len(rentrees) == 0 {
		return 0, 0
	}
	drapeaux := flagIndexByTeam(raws, scan)
	for i := range raws {
		f := drapeaux[i]
		if f < 0 {
			continue
		}
		atR, okR := flagFirstHomeInside(retours, f, raws[i], nil, nil)
		atO, okO := flagFirstHomeInside(rentrees, f, raws[i], raws, drapeaux)
		switch {
		case okR && (!okO || atR <= atO):
			raws[i].t1, raws[i].closed, raws[i].captured, raws[i].homed = atR, true, false, true
			byReturn++
		case okO:
			raws[i].t1, raws[i].closed, raws[i].captured, raws[i].homed = atO, true, false, true
			byObject++
		}
	}
	return byReturn, byObject
}

// flagFirstHomeInside rend le PREMIER instant, strictement interieur au portage, ou le drapeau
// `f` rentre chez lui. `autres` non nil arme le refus d'ambiguite de la RENTREE (cf. l'en-tete) ;
// nil le desarme, ce qui est le cas du retour credite — lui nomme son drapeau sans equivoque.
func flagFirstHomeInside(homes []flagHomecoming, f int, r flagCarryRaw,
	autres []flagCarryRaw, drapeaux []int) (int64, bool) {
	for _, h := range homes {
		if h.flag != f || h.at <= r.t0 || h.at >= r.t1 {
			continue
		}
		if autres != nil && flagAutreLacherProche(autres, drapeaux, f, h.at) {
			continue
		}
		return h.at, true
	}
	return 0, false
}

// flagAutreLacherProche dit qu'un portage d'un AUTRE drapeau s'acheve dans la seconde qui entoure
// l'instant : sa chute a pu produire la naissance qu'on lit comme une rentree.
func flagAutreLacherProche(raws []flagCarryRaw, drapeaux []int, f int, at int64) bool {
	for i := range raws {
		if raws[i].captured || drapeaux[i] == f {
			continue
		}
		if ecart := raws[i].t1 - at; ecart <= flagHomeGuardMS && ecart >= -flagHomeGuardMS {
			return true
		}
	}
	return false
}
