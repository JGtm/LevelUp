package killsource

// assist_fil.go — L ASSISTANT D UNE MORT SANS KILL-EVENT, LU AU FIL DES EVENEMENTS DE LA PARTIE.
//
// # POURQUOI UNE SECONDE LECTURE
//
// Le kill-event (genre 85, `PlayerKilledEvent`) ne s ecrit pas quand la victime est un bot : sur
// `0a08d2f2`, la recherche bit a bit de toute tete de genre 85 dans toutes les trames ne trouve, au
// couple des 23 morts de bot, qu une coincidence sans chaine d evenements (les 3 morts infligees PAR
// un bot ont chacune la leur, en chaine de 12), et la marche du film n en lit aucun, alors que les
// trames de ces morts sont lues jusqu a leur terminateur. Ces morts restaient donc `Known = false`
// quel que soit le film.
//
// Le jeu ecrit pourtant, pour chaque kill et dans la trame du dead-state, un message
// `PlayerGameEventSmall` (genre 82) par role — au tueur, a la victime, a leurs allies, a chaque
// assistant — dont le sac texte nomme le couple (tueur, victime) et dont le masque designe les
// destinataires ([grammar.FilLu]). Les morts de bot en portent comme les autres.
//
// # LE TYPE DE L EVENEMENT D ASSISTANCE SE LIT DANS LE FILM LUI-MEME
//
// Le `Type` d un message est son rang dans la table des evenements de la partie, et ce rang change
// d un build a l autre (70 sur les films de 2026, 61 sur un film d avril 2025). Il se lit donc par
// film, contre les kill-events du meme film : le type que le jeu adresse, au couple d un kill-event,
// a l assistant que ce kill-event nomme ([apprendreLeLexique]). Le lexique n est retenu que s il est
// UNIQUE et que le film ne le contredit pas : aucun kill-event a assistant dont le fil du couple ne
// s adresse pas a lui, aucun dont la trame lue entiere porte le fil du couple sans ce type.
//
// # CE QUE LA LECTURE PUBLIE, ET SEULEMENT SUR LES MORTS SANS KILL-EVENT
//
// Une mort a laquelle un kill-event est attache garde ce qu il dit ([decodeCtx.attachAssists]). Sur
// une autre, le fil du couple se cherche dans la trame de son dead-state ([Kill.paquet]) :
//
//	aucun message au couple                 Known reste faux
//	un destinataire du type appris          l assistant, nomme comme un assistant de kill-event
//	aucun, trame lue entiere                PAS d assistant, mesure
//	aucun, trame arretee                    Known reste faux (le message pouvait suivre l arret)
//	plusieurs destinataires                 tous ont assiste : le premier ecrit est nomme, les
//	                                        autres se comptent dans `Extra` (aucune part de degats
//	                                        ne dit lequel un kill-event aurait nomme)
//
// Les parts de degats ne s y lisent pas : elles restent non mesurees.

import (
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// filRec : un message du fil, avec l identite de sa trame.
type filRec struct {
	chunk, pidx int
	complet     bool
	typ         uint32
	tueur       int
	victime     int
	// evenement : le message, dont [filEvenement.Destine] lit le masque des destinataires.
	evenement filEvenement
}

// filEvenement : le message du fil vu par ses destinataires ([grammar.FilLu.Destine]).
type filEvenement interface {
	Destine(i int) bool
}

// filScan : les messages du fil d un film, ranges par trame.
type filScan struct {
	parPaquet map[[2]int][]filRec
}

// filDeLaMarche range les messages du fil de la marche des trames par trame.
func filDeLaMarche(l grammar.LectureDeKillsource) *filScan {
	s := &filScan{parPaquet: map[[2]int][]filRec{}}
	for _, e := range l.Fil {
		k := [2]int{e.PositionDuChunk, e.Index}
		s.parPaquet[k] = append(s.parPaquet[k], filRec{chunk: e.PositionDuChunk, pidx: e.Index,
			complet: e.Complet, typ: e.Evenement.Type, tueur: int(e.Evenement.Tueur),
			victime: int(e.Evenement.Victime), evenement: e})
	}
	return s
}

// auCouple rend les messages de la trame `(chunk, pidx)` dont le couple satisfait `couple`.
func (s *filScan) auCouple(chunk, pidx int, couple func(tueur, victime int) bool) []filRec {
	var out []filRec
	for _, r := range s.parPaquet[[2]int{chunk, pidx}] {
		if couple(r.tueur, r.victime) {
			out = append(out, r)
		}
	}
	return out
}

// destinatairesDuType rend les participants que les messages `rs` de type `typ` designent.
func destinatairesDuType(rs []filRec, typ uint32) []int {
	vu := map[int]bool{}
	var out []int
	for _, r := range rs {
		if r.typ != typ {
			continue
		}
		for i := range grammar.ParticipantsDuFil {
			if r.evenement.Destine(i) && !vu[i] {
				vu[i] = true
				out = append(out, i)
			}
		}
	}
	return out
}

// lexiqueDuFil : le type de l evenement d assistance d un film, et ce qui le confronte.
type lexiqueDuFil struct {
	// typ : le type appris ; utilisable dit que le film l etablit sans le contredire.
	typ        uint32
	utilisable bool
	// Les comptes de la confrontation sont ceux de [AssistFilStats].
	stats AssistFilStats
}

// apprendreLeLexique : le type que le fil adresse, au couple d un kill-event, a l assistant qu il
// nomme — UN seul sur le film —, puis la confrontation de ce type a tous les kill-events.
func apprendreLeLexique(recs []killEventRec, fil *filScan) lexiqueDuFil {
	var lx lexiqueDuFil
	appris := map[uint32]int{}
	for _, r := range recs {
		a := r.fields.assist
		if a < 0 || a == r.fields.killer || a == r.fields.victim {
			continue
		}
		for _, e := range fil.auCouple(r.chunk, r.pidx, coupleDe(r.fields)) {
			if e.evenement.Destine(a) {
				appris[e.typ]++
			}
		}
	}
	lx.stats.TypesAppris = len(appris)
	if len(appris) != 1 {
		return lx
	}
	for t, n := range appris {
		lx.typ, lx.stats.Apprentissages = t, n
	}
	for _, r := range recs {
		lx.confronter(r, fil.auCouple(r.chunk, r.pidx, coupleDe(r.fields)))
	}
	lx.utilisable = lx.stats.Desaccords == 0 && lx.stats.AssistantsSansEvenement == 0
	return lx
}

// coupleDe : le predicat du couple d un kill-event.
func coupleDe(f killEventFields) func(tueur, victime int) bool {
	return func(tueur, victime int) bool { return tueur == f.killer && victime == f.victim }
}

// confronter : un kill-event contre le fil de son couple, sous le type appris.
func (lx *lexiqueDuFil) confronter(r killEventRec, auCouple []filRec) {
	if len(auCouple) == 0 {
		return
	}
	dest := destinatairesDuType(auCouple, lx.typ)
	a := r.fields.assist
	switch {
	case a >= 0 && len(dest) > 0:
		if slices.Contains(dest, a) {
			lx.stats.Accords++
		} else {
			lx.stats.Desaccords++
		}
	case a >= 0 && tousComplets(auCouple):
		lx.stats.AssistantsSansEvenement++
	case a < 0 && len(dest) > 0:
		lx.stats.EvenementsSansAssistant++
	}
}

// tousComplets : chaque trame des messages `rs` est lue jusqu a son terminateur.
func tousComplets(rs []filRec) bool {
	for _, r := range rs {
		if !r.complet {
			return false
		}
	}
	return true
}

// attachAssistsDuFil : l assistant des morts publiees sans kill-event, lu au fil de leur trame
// (cf. l en-tete du fichier). Rend les comptes de la passe.
func (c *decodeCtx) attachAssistsDuFil(kills []Kill) AssistFilStats {
	if c.fil == nil {
		return AssistFilStats{}
	}
	lx := apprendreLeLexique(c.killEvents.recs, c.fil)
	st := lx.stats
	for i := range kills {
		k := &kills[i]
		if k.Assist.Known || !k.paquet.ok {
			continue
		}
		auCouple := c.fil.auCouple(k.paquet.chunk, k.paquet.pidx, func(tueur, victime int) bool {
			return c.nomA(victime, k.TimeMS) == k.Victim && c.nomA(tueur, k.TimeMS) == k.Feed.Killer
		})
		if len(auCouple) == 0 {
			continue
		}
		st.MortsAuFil++
		if !lx.utilisable {
			continue
		}
		c.lireLAssistantDuFil(k, destinatairesDuType(auCouple, lx.typ), tousComplets(auCouple), &st)
	}
	return st
}

// lireLAssistantDuFil : la decision d une mort dont le fil du couple est present (cf. l en-tete).
func (c *decodeCtx) lireLAssistantDuFil(k *Kill, dest []int, complet bool, st *AssistFilStats) {
	switch {
	case len(dest) == 0 && !complet:
		st.TramesArretees++
	case len(dest) == 0:
		k.Assist.Known, k.AssistLuAuFil, k.Assist.Index = true, true, -1
		st.SansAssistant++
	default:
		i := dest[0]
		k.Assist.Known, k.AssistLuAuFil, k.Assist.Index = true, true, i
		if len(dest) > 1 {
			k.Assist.Extra = len(dest) - 1
			st.AssistantsMultiples++
		}
		nom := c.nomA(i, k.TimeMS)
		switch {
		case nom == k.Feed.Killer:
			k.Assist.Rejected = AssistRejectSelf
			st.Rejetes++
		case nom == k.Victim:
			k.Assist.Rejected = AssistRejectVictim
			st.Rejetes++
		case estNomDeRemplissage(nom):
			k.Assist.Rejected = AssistRejectRoster
			st.Rejetes++
		default:
			k.Assist.Name = nom
			st.Nommes++
		}
	}
}

// nomA : le nom que porte l indice `i` a l instant `ms` d une ligne — celui du bot que
// BOT_METADATA declare a cet instant ([decodeCtx.nomDuBotA]), sinon celui du roster.
func (c *decodeCtx) nomA(i, ms int) string {
	if nom, ok := c.nomDuBotA(i, ms); ok {
		return nom
	}
	return c.roster.nameOf(i)
}

// AssistFilStats : les comptes de la lecture de l assistant au fil des evenements. AUCUN RATIO.
type AssistFilStats struct {
	// TypesAppris : types distincts que le fil adresse, au couple d un kill-event, a l assistant
	// qu il nomme. Le lexique n existe que s il vaut 1. Apprentissages : les messages qui l etablissent.
	TypesAppris, Apprentissages int
	// Accords, Desaccords : kill-events a assistant dont le fil du couple porte le type appris, qui
	// s adresse (ou non) a cet assistant. AssistantsSansEvenement : kill-events a assistant dont la
	// trame, lue entiere, porte le fil du couple sans le type appris. Un desaccord ou un assistant
	// sans evenement ferme la lecture du film.
	Accords, Desaccords, AssistantsSansEvenement int
	// EvenementsSansAssistant : kill-events SANS assistant dont le fil du couple adresse le type
	// appris a un joueur. Ne ferme pas la lecture : le jeu credite une assistance que le champ du
	// kill-event ne nomme pas (constate sur un film d avril 2025, trois morts).
	EvenementsSansAssistant int
	// MortsAuFil : morts publiees sans kill-event dont la trame du dead-state porte le fil du couple.
	MortsAuFil int
	// Nommes, SansAssistant, Rejetes : ce que la lecture a publie (`Known = true`).
	Nommes, SansAssistant, Rejetes int
	// AssistantsMultiples : morts publiees avec plusieurs assistants (le premier nomme, les autres dans
	// `Extra`). TramesArretees : morts au fil laissees `Known = false`, la trame arretee avant son
	// terminateur ne portant aucun evenement d assistance.
	AssistantsMultiples, TramesArretees int
}
