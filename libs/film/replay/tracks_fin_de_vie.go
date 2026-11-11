package replay

// tracks_fin_de_vie.go — OU FINIT UNE VIE PUBLIEE : A LA MORT QUE LE FILM ECRIT ; SINON, POUR UN JOUEUR
// QUE L API DECLARE PRESENT A LA FIN, AU BOUT DU FILM ; SINON AU DERNIER POINT.
//
// Le film cesse de repliquer la position d un bipede avant que sa mort soit ecrite : l ecart
// va de zero a plusieurs secondes, pendant lesquelles le joueur est vivant et sa position n est
// plus ecrite. Une vie qui finit au dernier point ecrit passe donc a « elimine » trop tot, et le fil des eliminations
// porte une ligne sans icone la ou le client ne trouve aucune vie qui se termine a l instant du
// kill. La fin d une vie est l instant de sa mort quand le film l ecrit.
//
// LA REGLE, SANS FENETRE DE TOLERANCE. La mort d une vie est la PREMIERE mort ecrite de son
// joueur posterieure a l ouverture de la vie. Elle ne la prolonge que si elle tombe APRES le
// dernier point publie, dans une frame ANTERIEURE a l ouverture de la vie suivante du joueur ou du
// slot (deux vies d un slot ne se recouvrent jamais), et dans le film. Rien n est pose entre le dernier point et la mort : aucune position n est inventee, le
// client garde la derniere et la montre vivante a cet endroit. Une vie qu aucune mort ne clot
// (fin de manche, fin du film, joueur parti) garde son dernier point, sauf la derniere vie d un
// joueur que l API declare present a la fin (tracks_fin_de_vie_presence.go).
//
// L HORLOGE EST CELLE DU CALAGE DEJA RESOLU (`IdentityRegistry.DeathOffsetMS`) : le fil des morts
// est date sur l horloge du match, les frames sur celle du film. Sans calage connu, aucune mort ne
// se place sur l axe : la fin reste le dernier point, et le repli est compte
// ([fallback.NomFinDeVieAuDernierPoint]).

import (
	"slices"
	"sort"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// clotureParMort porte ce dont la fin d une vie a besoin : les morts ecrites sur l horloge du film,
// les ouvertures des vies (pour borner la fin de chaque vie avant la vie suivante) et ce que le
// tableau de l API dit de chaque joueur (cf. tracks_fin_de_vie_presence.go).
type clotureParMort struct {
	origin, step uint64
	// frames : le nombre de frames du document ; une fin hors du film ne clot rien.
	frames int
	// morts : les instants des morts ecrites par joueur, en microsecondes sur l horloge du film,
	// croissants. Vide quand le calage n est pas connu.
	morts map[uint64][]int64
	// mortsDuFilm : le NOMBRE de morts que le film ecrit par joueur ; il ne depend d aucune horloge.
	mortsDuFilm map[uint64]int
	// debutsParJoueur / debutsParSlot : les ouvertures de vies, en microsecondes, croissantes.
	debutsParJoueur map[uint64][]int64
	debutsParSlot   map[uint32][]int64
	// tableau : la presence a la fin et le nombre de morts que l API declare, par xuid.
	tableau map[uint64]presenceAPI
	// calageInconnu : le fil porte des morts mais le calage n est pas resolu, aucune ne se place.
	calageInconnu bool
}

// nouvelleClotureParMort pose les morts du fil sur l'horloge du film. `calage` est celui du
// registre (cf. [IdentityRegistry.CalageSiConnu]) : nil = non resolu, aucune mort ne se place.
// `tableau` est le tableau de l API (cf. [Options.Participants]) : vide = aucune presence declaree.
func nouvelleClotureParMort(deaths []types.Death, calage *int64, vies []lifeSpan,
	tableau []Participant, h replayClock) clotureParMort {
	c := clotureParMort{origin: h.origin, step: h.step, frames: h.frames, tableau: presencesDuTableau(tableau)}
	c.debutsParJoueur = map[uint64][]int64{}
	c.debutsParSlot = map[uint32][]int64{}
	for _, l := range vies {
		if l.xuid != 0 {
			c.debutsParJoueur[l.xuid] = append(c.debutsParJoueur[l.xuid], l.from)
		}
		c.debutsParSlot[l.slot] = append(c.debutsParSlot[l.slot], l.from)
	}
	for x := range c.debutsParJoueur {
		slices.Sort(c.debutsParJoueur[x])
	}
	for s := range c.debutsParSlot {
		slices.Sort(c.debutsParSlot[s])
	}
	if len(deaths) == 0 {
		return c
	}
	c.mortsDuFilm = make(map[uint64]int, len(deaths))
	for _, d := range deaths {
		c.mortsDuFilm[d.XUID]++
	}
	if calage == nil {
		c.calageInconnu = true
		return c
	}
	c.morts = mortsParJoueur(deaths, *calage)
	for x := range c.morts {
		slices.Sort(c.morts[x])
	}
	return c
}

// fin rend la frame de fin de la vie `l`, dont le dernier point publie est a la frame `derniere`,
// et le motif de ce choix : la frame de sa mort ecrite quand elle suit ce point ; sinon, pour la
// derniere vie d un joueur que l API declare present a la fin, la derniere frame du film ; sinon
// `derniere`.
func (c clotureParMort) fin(l lifeSpan, derniere int) (int, motifFin) {
	if f, ok := c.finParMort(l, derniere); ok {
		return f, finParMort
	}
	return c.finParPresence(l, derniere)
}

// finParMort rend la frame de la mort ecrite qui clot la vie `l` apres son dernier point `derniere`,
// et vrai ; sinon `derniere` et faux.
func (c clotureParMort) finParMort(l lifeSpan, derniere int) (int, bool) {
	morts := c.morts[l.xuid]
	if l.xuid == 0 || len(morts) == 0 {
		return derniere, false
	}
	i := sort.Search(len(morts), func(i int) bool { return morts[i] > l.from })
	if i == len(morts) {
		return derniere, false
	}
	t := morts[i]
	if t < int64(c.origin) {
		return derniere, false
	}
	f := int((uint64(t) - c.origin) / c.step)
	if f <= derniere || f >= c.frames || f >= c.frameDe(c.ouvertureSuivante(l)) {
		return derniere, false
	}
	return f, true
}

// frameDe rend la frame qui contient l instant `t` (microsecondes sur l horloge du film) ; negative
// avant l origine. Une mort doit tomber dans une frame STRICTEMENT anterieure a celle de la vie
// suivante : la fin d une piste couvre sa frame entiere, et deux vies d un slot ne se recouvrent
// jamais, meme d un instant.
func (c clotureParMort) frameDe(t int64) int {
	d := t - int64(c.origin)
	if d < 0 {
		return -1
	}
	return int(d / int64(c.step))
}

// ouvertureSuivante rend l ouverture de la vie qui suit `l` pour son joueur ou son slot — la plus
// proche des deux — ou le plus grand instant quand aucune ne suit.
func (c clotureParMort) ouvertureSuivante(l lifeSpan) int64 {
	suivante := int64(1<<63 - 1)
	for _, debuts := range [][]int64{c.debutsParJoueur[l.xuid], c.debutsParSlot[l.slot]} {
		j := sort.Search(len(debuts), func(j int) bool { return debuts[j] > l.from })
		if j < len(debuts) {
			suivante = min(suivante, debuts[j])
		}
	}
	return suivante
}

// compterLeSilenceDeCalage compte les vies qui auraient pu finir a leur mort et restent au dernier point
// faute de calage : une par vie publiee que le registre nomme, quand le fil porte des morts.
func (c clotureParMort) compterLeSilenceDeCalage(fb *fallback.Compteur) {
	if c.calageInconnu {
		fb.Declenche(fallback.NomFinDeVieAuDernierPoint)
	}
}
