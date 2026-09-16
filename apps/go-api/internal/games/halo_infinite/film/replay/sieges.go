package replay

// sieges.go — LE SIEGE D'UN JOUEUR, ET CE QUI LE DECIDE (lot 1.9.14).
//
// # LE CONSTAT QUI OUVRE CE FICHIER (utilisateur, 2026-09-15)
//
// « Le rejeu affiche tout le roster du match tout le temps ; on n'a pas de raison d'afficher
// les joueurs qui ne jouent pas a l'instant T. » Mesure a l'appui, sur `bcb6d393` (CTF 4v4) :
// le document publie ONZE entrees de roster alors que le film ne porte JAMAIS plus de HUIT
// occupants simultanes. Les trois de trop sont des partants et des remplacants dont les
// presences ne se recouvrent pas — ils tiennent chacun une fiche a l'ecran du debut a la fin.
//
// # CE QUE LE FILM ECRIT, ET CE QU'IL N'ECRIT PAS (mesure du 2026-09-15, 4 temoins)
//
// IL ECRIT L'INDEX DE CHAQUE ENTITE `ti=9`, remplacants compris : un arrivant en cours de
// partie porte son index de joueur comme les autres (lot 1.7). L'index EST le siege.
//
// IL N'ECRIT PAS « ce remplacant prend le siege de ce partant » — sauf quand il REUTILISE
// l'index du partant, ce qui arrive 2 fois sur 35 arrivees des 18 films du lot 1.7, et 1 fois
// sur les 11 arrivees couvertes par les bobines des quatre temoins (`11de8353`, index 23 :
// l'entite de slot 1343 s'arrete au paquet 6, celle de slot 1789 demarre au paquet 7).
// Les 33 autres arrivees prennent un index NEUF, au-dessus du dernier siege de la table de
// depart.
//
// # LES DEUX DECISIONS, DANS L'ORDRE (D13, D14)
//
//	LECTURE  le siege d'une entree EST son index de film. Deux entrees qui partagent un index
//	         partagent un siege : le film a ECRIT la reprise, il n'y a rien a deviner.
//	         [SeatSourceLu].
//	REPLI    un arrivant dont le film ecrit un index NEUF n'est rattache a AUCUN partant par le
//	         film. Le chainer sur le siege d'un partant du meme camp est un APPARIEMENT ORDINAL
//	         — le k-ieme arrivant continue le k-ieme siege libere — c'est-a-dire le modele des
//	         sieges valide le 2026-09-02, ramene a son rang de repli NOMME et COMPTE
//	         (`repli_siege_du_remplacant_par_appariement_ordinal`). [SeatSourceApparie].
//
// L'ORDRE EST FIXE ET IL COMPTE : on lit d'abord (index, reprise ecrite), on se replie ensuite,
// et JAMAIS sur une entree dont le film a ecrit la reprise.
//
// # QUI EST UN ARRIVANT SE LIT, IL NE SE SEUILLE PAS
//
// La table des joueurs de `chunk_00` est le roster du DEBUT du film (D2 du lot 1.6) : une
// entree dont l'index y a un siege est une ORIGINE, une entree dont l'index n'y est pas est une
// ARRIVEE. Aucune fenetre de temps, aucune marge — ce que d'autres surfaces du produit ont du
// seuiller faute de cette table (10 s / 20 s dans `presenceFeed`), la table le donne
// exactement. UN FILM SANS TABLE ne permet donc pas de distinguer les deux : AUCUN chainage
// n'est alors decide, et la couverture le dit ([SeatCoverage.SansTableDuFilm]) — s'abstenir
// n'est pas se replier.
//
// # CE FICHIER NE DESSINE RIEN
//
// Il publie un siege et une provenance. La regle d'affichage — « a l'instant T, les seules
// fiches des occupants presents » — vit cote web, et elle se sert des INTERVALLES DE PRESENCE
// que les vies portent deja (lot 1.9.13).

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/replay/fallback"
)

// SeatSourceLu / SeatSourceApparie : les deux provenances d'un siege publie.
//
// ELLES NE SONT PAS DECORATIVES. Un client qui dessine un relais de siege affiche, dans la
// meme fiche, deux joueurs differents ; savoir si le film l'a ECRIT ou si un appariement l'a
// DEDUIT est ce qui permet de ne pas presenter une deduction comme une lecture.
const (
	// SeatSourceLu : le siege est l'index que le film ecrit pour cette entree.
	SeatSourceLu = "lu"
	// SeatSourceApparie : le film a ecrit un index NEUF ; le rattachement au siege d'un partant
	// vient de l'appariement ordinal par camp, un REPLI (cf. l'en-tete).
	SeatSourceApparie = "apparie"
)

// SeatCoverage est ce que la pose des sieges a lu, appariee, et laisse sans presence.
//
// SON CHIFFRE CENTRAL EST LE COUPLE `Entrees` / `OccupantsMax` : leur ECART est exactement le
// nombre de fiches que la regle de lecture retire de l'ecran. C'est le constat utilisateur,
// rendu mesurable artefact par artefact.
type SeatCoverage struct {
	// Entrees est le denominateur : les entrees de roster publiees.
	Entrees int `json:"entrees"`
	// Sieges est le nombre de sieges DISTINCTS apres la pose — donc le nombre de fiches qu'un
	// client dessinerait s'il les affichait toutes.
	Sieges int `json:"sieges"`
	// Lus : les entrees dont le siege est l'index ecrit par le film.
	Lus int `json:"lus"`
	// Apparies : les entrees dont le siege vient du repli ordinal.
	Apparies int `json:"apparies"`
	// ReprisesEcrites : les sieges que le film donne a PLUSIEURS entrees — la reprise que le
	// film ecrit lui-meme. C'est le compteur qui doit MONTER a mesure que la chaine
	// `index -> xuid` d'un arrivant se ferme, et faire tomber `Apparies`.
	ReprisesEcrites int `json:"reprisesEcrites"`
	// Arrivants : les entrees dont l'index n'a pas de siege dans la table du DEBUT du film.
	Arrivants int `json:"arrivants"`
	// PresencesCloses : les entrees dont la presence publiee s'acheve AVANT la derniere frame.
	//
	// CE COMPTEUR NE DIT PAS « PARTI », ET C'EST VOULU : le film ne distingue pas un joueur qui
	// quitte la partie d'un joueur qui meurt sans reapparaitre avant la fin. Les deux liberent
	// leur siege au sens de la presence, et c'est cette imprecision — nommee ici plutot que tue —
	// que l'appariement ordinal porte.
	PresencesCloses int `json:"presencesCloses"`
	// SansPresence : les entrees qu'AUCUNE vie publiee ne couvre, a aucun instant. Elles n'ont
	// de fiche a aucun T — ni maintenant ni jamais — et le dire est la seule facon de ne pas
	// confondre « le joueur n'a pas joue » avec « la regle de lecture l'a perdu ».
	SansPresence int `json:"sansPresence"`
	// OccupantsMax : le plus grand nombre d'entrees dont une vie couvre une meme frame.
	OccupantsMax int `json:"occupantsMax"`
	// SansTableDuFilm : le film ne porte pas sa table de depart, donc arrivants et origines ne
	// se distinguent pas et AUCUN chainage n'est decide. Ce n'est pas un repli : c'est une
	// abstention, et elle se lit ici.
	SansTableDuFilm bool `json:"sansTableDuFilm,omitempty"`
}

// siegeEntree : une entree de roster et ce que ses vies en disent, pendant la pose.
type siegeEntree struct {
	i          int
	debut, fin int
	vies       int
	origine    bool
}

// poserLesSieges ECRIT le siege de chaque entree du roster, EN PLACE, et rend sa couverture.
//
// EN PLACE, COMME LES AUTRES POSES DE CE PAQUET (`nameTracksByLives`, `nameBotTracks`) : le
// roster est une tranche, l'assemblage en tient deja la reference, et la rendre pour se la
// faire reassigner n'ajouterait qu'une occasion d'oublier l'affectation.
//
// PURE AU SENS DU DECODAGE : elle ne lit aucun octet de film et n'ouvre aucune base. Son
// entree est ce que l'assemblage a deja produit — le roster, les pistes publiees, la table du
// debut.
func poserLesSieges(roster []RosterEntry, tracks []Track, table FilmPlayerTable,
	frames int, fb *fallback.Compteur,
) SeatCoverage {
	cov := SeatCoverage{Entrees: len(roster)}
	if len(roster) == 0 {
		return cov
	}
	origines := siegesDuDebut(table)
	cov.SansTableDuFilm = origines == nil
	presences := presencesParIdentite(tracks)
	etat := make([]siegeEntree, len(roster))
	partages := map[int]int{}
	for i := range roster {
		roster[i].Seat, roster[i].SeatSource = roster[i].FilmIndex, SeatSourceLu
		partages[roster[i].FilmIndex]++
		etat[i] = etatDeLEntree(i, roster[i], presences, origines)
		cov.compterUneEntree(etat[i], frames)
	}
	for _, n := range partages {
		if n > 1 {
			cov.ReprisesEcrites++
		}
	}
	if origines != nil {
		cov.Apparies = apparierLesArrivants(roster, etat, partages, frames)
		fb.DeclencheN(fallback.NomSiegeDuRemplacantParAppariementOrdinal, cov.Apparies)
	}
	cov.Lus = cov.Entrees - cov.Apparies
	cov.Sieges = compterLesSieges(roster)
	cov.OccupantsMax = occupantsSimultanes(etat)
	return cov
}

// compterUneEntree met a jour les compteurs qu'une seule entree renseigne.
func (c *SeatCoverage) compterUneEntree(e siegeEntree, frames int) {
	switch {
	case e.vies == 0:
		c.SansPresence++
	case frames > 0 && e.fin < frames-1:
		c.PresencesCloses++
	}
	if !e.origine {
		c.Arrivants++
	}
}

// siegesDuDebut rend les index que la table de `chunk_00` occupe, ou NIL quand le film ne porte
// pas sa table — nil et une table vide ne disent pas la meme chose, et tout ce fichier en
// depend.
func siegesDuDebut(table FilmPlayerTable) map[int]bool {
	if !table.Lue() {
		return nil
	}
	out := make(map[int]bool, len(table.Seats))
	for _, s := range table.Seats {
		out[s.FilmIndex] = true
	}
	return out
}

// etatDeLEntree resume ce que les vies disent d'une entree : combien, et de quand a quand.
func etatDeLEntree(i int, e RosterEntry, presences map[string][2]int,
	origines map[int]bool,
) siegeEntree {
	out := siegeEntree{i: i, origine: origines == nil || origines[e.FilmIndex]}
	if w, ok := presences[cleDeRoster(e)]; ok {
		out.debut, out.fin, out.vies = w[0], w[1], 1
	}
	return out
}

// presencesParIdentite rend, par identite, l'ENVELOPPE des vies publiees : premiere frame de la
// premiere vie, derniere frame de la derniere.
//
// LA CLE EST CELLE DU ROSTER — le xuid quand il existe, `bot:<nom>` sinon — parce que c'est
// celle sur laquelle le client joint ses fiches. Une piste sans identite n'entre nulle part :
// elle n'appartient a personne, donc elle ne prouve la presence de personne.
func presencesParIdentite(tracks []Track) map[string][2]int {
	out := map[string][2]int{}
	for _, tr := range tracks {
		cle := cleDePiste(tr)
		if cle == "" {
			continue
		}
		w, vu := out[cle]
		if !vu {
			out[cle] = [2]int{tr.StartFrame, tr.EndFrame}
			continue
		}
		if tr.StartFrame < w[0] {
			w[0] = tr.StartFrame
		}
		if tr.EndFrame > w[1] {
			w[1] = tr.EndFrame
		}
		out[cle] = w
	}
	return out
}

// cleDeRoster / cleDePiste : LA MEME CLE DES DEUX COTES. Un bot n'a pas de xuid (schema 36) :
// son identite est son nom. Les deux fonctions existent parce que les deux types portent cette
// identite sous des champs differents ; elles ne divergent pas.
func cleDeRoster(e RosterEntry) string {
	if e.XUID != "" {
		return e.XUID
	}
	if e.Bot && e.Name != "" {
		return botIdentityKey(e.Name)
	}
	return ""
}

func cleDePiste(tr Track) string {
	if tr.XUID != "" {
		return tr.XUID
	}
	if tr.Bot != "" {
		return botIdentityKey(tr.Bot)
	}
	return ""
}

// botIdentityKey est la forme de l'identite d'un bot, la meme que le client emploie.
func botIdentityKey(nom string) string { return "bot:" + nom }

// apparierLesArrivants chaine, PAR CAMP, le k-ieme arrivant sur le k-ieme siege libere, et rend
// le nombre de chainages poses.
//
// C'EST LE REPLI, ET IL NE S'APPLIQUE QU'A CE QUE LA LECTURE N'A PAS TRANCHE : une entree dont
// le film a REUTILISE l'index (donc dont le siege est partage) est ecartee — le film a ecrit sa
// reprise, la deduire par-dessus serait exactement l'ordre que D14 (b) interdit.
func apparierLesArrivants(roster []RosterEntry, etat []siegeEntree, partages map[int]int,
	frames int,
) int {
	poses := 0
	for _, camp := range campsDuRoster(roster) {
		partants, arrivants := partantsEtArrivants(roster, etat, partages, campEtFrames{camp, frames})
		k := 0
		for _, a := range arrivants {
			for k < len(partants) && etat[partants[k]].fin >= etat[a].debut {
				k++ // un siege encore occupe a l'arrivee ne peut pas etre celui qu'on reprend
			}
			if k >= len(partants) {
				break
			}
			roster[a].Seat, roster[a].SeatSource = roster[partants[k]].Seat, SeatSourceApparie
			k++
			poses++
		}
	}
	return poses
}

// campEtFrames regroupe les deux bornes du tri — le depot borne a cinq parametres, et la paire
// voyage toujours ensemble.
type campEtFrames struct {
	camp   int
	frames int
}

// campsDuRoster rend les designateurs d'equipe presents, TRIES : l'ordre d'iteration d'une map
// Go est aleatoire, et un artefact qui change d'octets sans changer de contenu est indiffable.
//
// UNE ENTREE SANS EQUIPE LUE N'A PAS DE CAMP et n'entre dans aucun appariement : apparier deux
// joueurs dont on ignore le camp reviendrait a inventer qu'ils sont du meme.
func campsDuRoster(roster []RosterEntry) []int {
	vus := map[int]bool{}
	for _, e := range roster {
		if e.Team != nil {
			vus[*e.Team] = true
		}
	}
	out := make([]int, 0, len(vus))
	for c := range vus {
		out = append(out, c)
	}
	sort.Ints(out)
	return out
}

// partantsEtArrivants rend, pour un camp, les indices des entrees qui LIBERENT un siege (par
// fin de presence croissante) et celles qui en CHERCHENT un (par debut de presence croissant).
func partantsEtArrivants(roster []RosterEntry, etat []siegeEntree, partages map[int]int,
	cf campEtFrames,
) (partants, arrivants []int) {
	for i := range roster {
		if roster[i].Team == nil || *roster[i].Team != cf.camp || etat[i].vies == 0 {
			continue
		}
		switch {
		case etat[i].origine && cf.frames > 0 && etat[i].fin < cf.frames-1:
			partants = append(partants, i)
		case !etat[i].origine && partages[roster[i].FilmIndex] == 1:
			arrivants = append(arrivants, i)
		}
	}
	sort.Slice(partants, func(a, b int) bool { return etat[partants[a]].fin < etat[partants[b]].fin })
	sort.Slice(arrivants, func(a, b int) bool {
		return etat[arrivants[a]].debut < etat[arrivants[b]].debut
	})
	return partants, arrivants
}

// compterLesSieges rend le nombre de sieges distincts apres la pose.
func compterLesSieges(roster []RosterEntry) int {
	vus := map[int]bool{}
	for _, e := range roster {
		vus[e.Seat] = true
	}
	return len(vus)
}

// occupantsSimultanes rend le plus grand nombre d'entrees dont la presence couvre une meme
// frame, par balayage des bornes — la mesure qui chiffre l'ecart avec `Entrees`.
func occupantsSimultanes(etat []siegeEntree) int {
	type borne struct{ f, d int }
	bornes := make([]borne, 0, 2*len(etat))
	for _, e := range etat {
		if e.vies == 0 {
			continue
		}
		bornes = append(bornes, borne{e.debut, +1}, borne{e.fin + 1, -1})
	}
	// LES FERMETURES AVANT LES OUVERTURES A EGALITE DE FRAME : deux presences qui se touchent
	// sans se recouvrir ne doivent pas compter pour deux — c'est exactement le cas d'un relais.
	sort.Slice(bornes, func(a, b int) bool {
		if bornes[a].f != bornes[b].f {
			return bornes[a].f < bornes[b].f
		}
		return bornes[a].d < bornes[b].d
	})
	n, maxi := 0, 0
	for _, b := range bornes {
		n += b.d
		if n > maxi {
			maxi = n
		}
	}
	return maxi
}
