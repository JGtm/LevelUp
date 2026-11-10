package replay

// tracks_fin_de_vie_presence.go — LE JOUEUR QUI N A PAS QUITTE LA PARTIE RESTE AFFICHE JUSQU AU BOUT.
//
// Le film cesse parfois d ecrire la position d un joueur bien avant la fin, sans ecrire sa mort :
// il est vivant, immobile ou embarque, et rien ne le dit parti. Une vie qui finit alors a son
// dernier point fait disparaitre un joueur qui est encore dans la partie.
//
// LA REGLE. Pour la DERNIERE vie d un joueur (aucune vie de ce xuid ne s ouvre apres elle), quand
// aucune mort ecrite ne la clot et que l API le declare present a la fin
// (`present_at_completion` vrai, `left_in_progress` faux), la fin de la vie est la derniere frame du
// film, bornee STRICTEMENT avant l ouverture de la vie suivante du slot. Aucun point n est
// fabrique : le client garde la derniere position et montre le joueur vivant a cet endroit.
//
// « A QUITTE » SE LIT SUR L API, JAMAIS SUR L ARRET DES POSITIONS. Un joueur dont l API dit parti
// (`present_at_completion` faux ou `left_in_progress` vrai) garde son dernier point : c est la fin
// que le film montre.
//
// CE QUI N ETEND PAS, ET SE COMPTE (replis nommes, jamais une valeur par defaut qui affirme la
// presence) :
//
//   - le tableau ne dit pas : participant introuvable, drapeau ou nombre de morts non declare (matchs
//     anciens, colonnes NULL), vie sans xuid (bot, vie sans nom) que le film laisse courir jusqu au
//     bout ([fallback.NomFinDeVieSansTableauAPI]) ;
//   - l API compte plus de morts que le film n en ecrit : une mort manque, rien ne dit que cette vie
//     n est pas celle qu elle clot ([fallback.NomFinDeVieMortManquante]). Aucune tolerance.
//
// Une mort ecrite apres l ouverture de la vie decide seule : la regle de tracks_fin_de_vie.go la
// place ou la refuse, la presence n y change rien.

import (
	"sort"
	"strconv"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"
)

// presenceAPI est ce que le tableau de l API dit d un joueur pour la fin de sa derniere vie. Chaque
// champ est TRI-ETAT : nil = la base ne le declare pas, ce qui n est ni vrai ni faux.
type presenceAPI struct {
	present, left *bool
	deaths        *int
}

// motifFin dit pourquoi une vie finit ou elle finit.
type motifFin int

const (
	// finAuDernierPoint : rien a prolonger, ou le joueur a quitte la partie selon l API.
	finAuDernierPoint motifFin = iota
	// finParMort : la mort ecrite du joueur clot la vie.
	finParMort
	// finParPresence : l API declare le joueur present a la fin, la vie court jusqu au bout du film.
	finParPresence
	// finSansTableau : la vie aurait pu etre prolongee, le tableau de l API ne permet pas d en decider.
	finSansTableau
	// finMortManquante : l API compte plus de morts que le film n en ecrit.
	finMortManquante
)

// compter verse au compteur de replis le motif d une fin laissee au dernier point faute de pouvoir
// decider. nil-safe : un compteur absent ne compte rien.
func (m motifFin) compter(fb *fallback.Compteur) {
	switch m {
	case finSansTableau:
		fb.Declenche(fallback.NomFinDeVieSansTableauAPI)
	case finMortManquante:
		fb.Declenche(fallback.NomFinDeVieMortManquante)
	case finAuDernierPoint, finParMort, finParPresence:
	}
}

// presencesDuTableau indexe par xuid ce que le tableau de l API dit de chaque joueur. Une ligne
// dont l identifiant n est pas un xuid decimal (`bid(N.0)` d un bot) n entre pas : le fil des
// morts ne date aucune mort de bot, la garde du nombre de morts ne s y applique pas.
func presencesDuTableau(lignes []Participant) map[uint64]presenceAPI {
	if len(lignes) == 0 {
		return nil
	}
	out := make(map[uint64]presenceAPI, len(lignes))
	for _, p := range lignes {
		x, err := strconv.ParseUint(p.ID, 10, 64)
		if err != nil || x == 0 {
			continue
		}
		out[x] = presenceAPI{present: p.PresentAtCompletion, left: p.LeftInProgress, deaths: p.Deaths}
	}
	return out
}

// finParPresence decide la fin de la vie `l` quand aucune mort ecrite ne la prolonge : la derniere
// frame du film pour un joueur que l API declare present a la fin, sinon `derniere`. Cette etape ecarte
// les vies qui n ont rien a prolonger et celles que le tableau ne permet pas de decider ; la decision de
// prolongation est [clotureParMort.prolongerSiPresent].
func (c clotureParMort) finParPresence(l lifeSpan, derniere int) (int, motifFin) {
	if derniere >= c.frames-1 {
		return derniere, finAuDernierPoint // la vie touche deja le bout du film
	}
	if l.xuid == 0 {
		// Un bot ou une vie sans nom ne se joint pas au tableau par xuid : seule la vie que la
		// replication du slot laisse courir jusqu a la fin de ce que le film montre se compte.
		if l.cause == CauseVieFinFilm {
			return derniere, finSansTableau
		}
		return derniere, finAuDernierPoint
	}
	if !c.derniereVieDuJoueur(l) || c.uneMortSuitLOuverture(l) {
		return derniere, finAuDernierPoint
	}
	p, connu := c.tableau[l.xuid]
	if !connu || p.present == nil || p.left == nil || p.deaths == nil {
		return derniere, finSansTableau
	}
	return c.prolongerSiPresent(l, derniere, p)
}

// prolongerSiPresent decide la fin de la vie `l` quand le tableau declare ses trois champs : `derniere`
// si le joueur est parti ou si une mort manque au film, sinon la derniere frame du film bornee
// strictement avant l ouverture de la vie suivante du slot.
func (c clotureParMort) prolongerSiPresent(l lifeSpan, derniere int, p presenceAPI) (int, motifFin) {
	if !*p.present || *p.left {
		return derniere, finAuDernierPoint // parti : la fin du film n est pas la sienne
	}
	if *p.deaths > c.mortsDuFilm[l.xuid] {
		return derniere, finMortManquante
	}
	borne := min(c.frames-1, c.frameDe(c.ouvertureSuivante(l))-1)
	if borne <= derniere {
		return derniere, finAuDernierPoint
	}
	return borne, finParPresence
}

// derniereVieDuJoueur dit qu aucune vie du xuid de `l` ne s ouvre apres elle.
func (c clotureParMort) derniereVieDuJoueur(l lifeSpan) bool {
	debuts := c.debutsParJoueur[l.xuid]
	return len(debuts) > 0 && debuts[len(debuts)-1] == l.from
}

// uneMortSuitLOuverture dit que le film ecrit une mort du joueur posterieure a l ouverture de `l`.
// Sans calage, aucune mort ne se date : le joueur qui en a ecrit une ne se decide pas, celui qui n en
// a aucune n a rien a clore.
func (c clotureParMort) uneMortSuitLOuverture(l lifeSpan) bool {
	if c.calageInconnu {
		return c.mortsDuFilm[l.xuid] > 0
	}
	morts := c.morts[l.xuid]
	return sort.Search(len(morts), func(i int) bool { return morts[i] > l.from }) < len(morts)
}
