package killsource

// match.go — L APPARIEMENT AU KILL-FEED, ET LA CONSTRUCTION DES LIGNES PUBLIEES.
//
// QUATRE CRITERES, de plus en plus faibles, et JAMAIS melanges :
//
//	COUPLE EXACT     le couple (tueur, victime) du dead-state est celui du feed dans la fenetre.
//	                 C est le critere FORT ; c est lui qui fonde le gate (b).
//	VICTIME SEULE    une contrainte de moins. Reserve aux morts dont la SOURCE APPARTIENT A LA
//	                 VICTIME : le feed credite alors un AUTRE joueur, donc le couple exact NE
//	                 PEUT PAS s apparier — la victime, elle, est juste. Il n est employe que
//	                 sur les instants que le critere fort n a pas couverts.
//	MORT DE BOT      le kill du feed existe, la mort n existe pas. Le candidat doit porter
//	                 l indice EPINGLE d un bot en victime, et rendre le tueur du feed.
//	MORT PAR UN BOT  la mort du feed existe, le kill n existe pas. Le candidat doit porter
//	                 l indice EPINGLE d un bot en TUEUR, et rendre la victime du feed. C est le
//	                 symetrique exact du precedent, et le kill-feed y garde la moitie qu il porte.
//
// Le controle de hasard de la fenetre a ete mesure pour chacun (horloge des candidats decalee de
// +-20 s et +-60 s) : c est la baseline triviale du 3e temps, sans laquelle un taux
// d appariement ne prouve rien.

import "levelup/go-api/internal/games/halo_infinite/film/damagetag"

// matchExact : le critere FORT.
func (c *decodeCtx) matchExact(cd candidate) *feedEvent {
	for i := range c.feed.pairs {
		e := &c.feed.pairs[i]
		dt := e.timeMS - cd.ms
		if dt < -tolMS || dt > tolMS {
			continue
		}
		if e.victim == c.roster.nameOf(cd.victim) && e.killer == c.roster.nameOf(cd.killer) {
			return e
		}
	}
	return nil
}

// matchVictim : la victime du candidat est-elle celle d une mort du feed dans la fenetre ?
func (c *decodeCtx) matchVictim(cd candidate) *feedEvent {
	for i := range c.feed.pairs {
		e := &c.feed.pairs[i]
		dt := e.timeMS - cd.ms
		if dt < -tolMS || dt > tolMS {
			continue
		}
		if e.victim == c.roster.nameOf(cd.victim) {
			return e
		}
	}
	return nil
}

// selfSourceOK : LES DEUX DISCRIMINANTS. Ils portent sur la STRUCTURE du candidat — ou il tombe,
// quelle est la magnitude de son tag — et sur aucune heuristique temporelle.
//
// LE TEST DE MAGNITUDE ACHETE DE LA JUSTESSE, PAS DE LA COUVERTURE — ET SA PORTEE COMPTE. Sous
// une architecture SCAN-D-ABORD, sans lui, une ligne publie un tag << degat global >> a la place
// du lance-roquettes confirme en Theater, les deux candidats etant dans le meme paquet, et la
// couverture ne bouge pas. Sous l HYBRIDE — ce que ce paquet fait — la MARCHE sert cette ligne et
// le filtre ne change AUCUNE etiquette sur les quatre films de reference : il est le FILET du
// regime ou le scan reprendrait la main, pas un correcteur en service. Voir [Options].
//
// PORTEE : il ne vaut QUE dans la population `victime == tueur`. Applique a toute la population
// il couterait 47 vrais positifs sur 82 sur un film — trois tags REELS vivent sous 0x10000.
func (c *decodeCtx) selfSourceOK(cd candidate, mult map[multKey]int) bool {
	if multOf(mult, cd) >= c.opts.MultiplicityMax {
		return false
	}
	return !c.opts.StrongTagRequired || damagetag.Strong(cd.tag)
}

// isBotSide : le candidat implique-t-il un indice epingle sur un bot ?
func (c *decodeCtx) isBotSide(cd candidate) bool {
	return c.roster.isBotIndex(cd.victim) || c.roster.isBotIndex(cd.killer)
}

// botMatch : un kill du feed sans victime humaine, et le candidat du scan qui le decode.
type botMatch struct {
	event feedEvent
	fab   bool // le kill avait ete RECOLLE en un couple fabrique (REPLI)
	// victimeLue : l indice de replication que le KILL-EVENT 85 nomme en victime, -1 quand le
	// film s est tu. Quand il vaut >= 0, l appariement exige CET indice et non « un bot
	// quelconque » : la victime est nommee a la source (lot 1.9.3).
	victimeLue int
	found      bool
	cand       candidate
}

// resolveBotDeaths : confronte au scan chaque kill dont la victime n est pas au feed.
//
// TROIS POPULATIONS CANDIDATES, ET LA PREMIERE EST NEUVE AU LOT 1.9.3 :
//
//	LUE       le kill-event 85 NOMME un bot en victime. Le couple est contraint des deux cotes
//	          par la LECTURE, pas par la structure du feed.
//	PERDUE    le kill n avait aucun voisin a consommer (`orphK`).
//	FABRIQUEE le REPLI a recolle un voisin, peut-etre a tort (`fab`) — le cas que la lecture
//	          fait disparaitre partout ou elle se prononce.
//
// C est le DEAD-STATE qui tranche, jamais la structure du feed.
func (c *decodeCtx) resolveBotDeaths() []botMatch {
	ms := make([]botMatch, 0, len(c.feed.orphK)+len(c.feed.fab)+len(c.feed.botLus))
	for _, b := range c.feed.botLus {
		ms = append(ms, botMatch{event: b.ev, victimeLue: b.victime})
	}
	for _, e := range c.feed.orphK {
		ms = append(ms, botMatch{event: e, victimeLue: -1})
	}
	for _, e := range c.feed.fab {
		ms = append(ms, botMatch{event: e, fab: true, victimeLue: -1})
	}
	for i := range ms {
		c.apparierMortDeBot(&ms[i])
	}
	return ms
}

// apparierMortDeBot : le premier candidat de la fenetre dont la victime est LE bot nomme (ou, a
// defaut de nom lu, un bot epingle quelconque) et dont le tueur est celui du feed.
func (c *decodeCtx) apparierMortDeBot(m *botMatch) {
	for _, cd := range c.scanCands {
		dt := m.event.timeMS - cd.ms
		if dt < -tolMS || dt > tolMS {
			continue
		}
		if m.victimeLue >= 0 {
			if cd.victim != m.victimeLue {
				continue
			}
		} else if !c.roster.isBotIndex(cd.victim) {
			continue
		}
		if c.roster.nameOf(cd.killer) == m.event.killer {
			m.found, m.cand = true, cd
			return
		}
	}
}

// botKillerMatch : une mort du feed que personne n a consommee, et le candidat qui la decode.
type botKillerMatch struct {
	event feedEvent
	found bool
	cand  sourcedCandidate
}

// resolveBotKillerDeaths : confronte aux candidats chaque mort du feed dont le TUEUR n est pas au
// feed. Le candidat retenu doit porter un indice EPINGLE sur un bot en TUEUR et rendre la VICTIME
// du feed — le couple est donc contraint des deux cotes, comme partout ailleurs, la moitie
// << tueur >> venant du roster de replication au lieu du kill-feed.
//
// LA POPULATION EST DESIGNEE PAR LE FEED, PAS PAR LE DEAD-STATE, et c est ce qui la rend
// falsifiable : sur les deux films SANS bot elle est VIDE (0 mort orpheline sur 93 et 99), et sur
// les deux films AVEC bot son cardinal est exactement le nombre de kills que l API attribue au bot
// (3 et 1). Une population qui se remplit sur les films a bot et reste vide sur les autres n est
// pas un artefact de lecteur.
//
// `all` arrive DANS L ORDRE DE PRIORITE de l hybride (marche puis scan) : le premier candidat qui
// satisfait le couple gagne, donc la marche garde la priorite ici comme partout.
func (c *decodeCtx) resolveBotKillerDeaths(all []sourcedCandidate) []botKillerMatch {
	ms := make([]botKillerMatch, 0, len(c.feed.orphD))
	for _, e := range c.feed.orphD {
		ms = append(ms, botKillerMatch{event: e})
	}
	for i := range ms {
		for _, cd := range all {
			dt := ms[i].event.timeMS - cd.ms
			if dt < -tolMS || dt > tolMS {
				continue
			}
			if c.roster.isBotIndex(cd.killer) && c.roster.nameOf(cd.victim) == ms[i].event.victim {
				ms[i].found, ms[i].cand = true, cd
				break
			}
		}
	}
	return ms
}

// ghostPairs : les couples FABRIQUES dont le kill est en realite une mort de BOT.
//
// ILS DOIVENT SORTIR DU DENOMINATEUR. Ce ne sont pas des morts manquees, ce sont des morts qui
// N EXISTENT PAS : la vraie victime est un bot, et la reconstruction a pris la victime du voisin.
// Le decodeur a ainsi corrige NOTRE PROPRE APPARIEMENT, verifie en Theater. Compter une mort
// inexistante comme manquee fausse le denominateur ; l alerter comme anomalie serait pire.
func (c *decodeCtx) ghostPairs() map[int]bool {
	g := map[int]bool{}
	for _, m := range c.resolveBotDeaths() {
		if m.found && m.fab {
			g[m.event.timeMS] = true
		}
	}
	return g
}

// killDraft : ce que l APPARIEMENT sait d une mort, avant que la source ne s y ajoute. Il existe
// pour que [decodeCtx.buildKill] garde deux parametres : la verite du feed d un cote, la verite
// de la source de l autre — la separation des deux verites jusque dans la signature.
type killDraft struct {
	timeMS   int
	victim   string
	killer   string
	inFeed   bool
	origin   Origin
	diverges bool
}

// buildKill : LA CONSTRUCTION DES DEUX VERITES. Elles sont remplies COTE A COTE, a partir de deux
// sources differentes, et aucune n est derivee de l autre.
func (c *decodeCtx) buildKill(d killDraft, cd sourcedCandidate) Kill {
	return Kill{
		TimeMS:   d.timeMS,
		Victim:   d.victim,
		Feed:     FeedTruth{Killer: d.killer, Present: d.inFeed},
		Source:   sourceTruthOf(cd.tag, cd.cat),
		Diverges: d.diverges,
		Read:     Provenance{Path: cd.path, Origin: d.origin, Multiplicity: multOf(c.mult, cd.candidate)},
	}
}
