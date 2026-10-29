package replay

// sieges_a_la_taille.go — AUCUNE EQUIPE NE TIENT PLUS DE PLACES QUE SA TAILLE (controle n° 8 des
// controles de la v8 ; decisions utilisateur du 2026-10-21, sur la REGLE DES PLACES de sieges.go).
//
// # CE QUI FAISAIT UNE PLACE DE TROP
//
// Une place est un index de joueur du film, et l'index n'est pas une place d'EQUIPE : un partant
// d'avant le coup d'envoi laisse son index dans la table du debut, un joueur qui revient prend
// l'index qu'il avait (ou un autre), un arrivant prend l'index libre que le jeu lui donne — pas
// forcement celui du partant qu'il remplace. Une equipe peut donc voir passer plus d'index que sa
// taille, et le rejeu, qui rend une fiche par place, lui en montrait autant (`4f77afc1`, Eagle : 13
// fiches en BTB).
//
// # LA REGLE (utilisateur, 2026-10-21 : « impossible d'avoir 5 joueurs en meme temps, meme s'il y a
// un bot » ; et elle est SYMETRIQUE)
//
// Une fiche est une PLACE D'EQUIPE ; une equipe en a autant que sa taille ([poseDesPlaces.capaciteDe] :
// la taille officielle de la playlist quand la table du titre la declare, sinon la capacite estimee).
// Les places gardees sont les premieres tenues (premiere occupation CERTAINE). Chaque intervalle de
// presence de l'equipe, dans l'ordre de son debut, prend une place gardee —
//
//	la SIENNE si elle est gardee et libre ;
//	sinon une place gardee LIBRE pendant sa presence certaine, celle que son dernier occupant a quittee
//	  le plus tard : un remplacant, humain ou bot, prend la fiche du partant ;
//	sinon, pour un HUMAIN, la place de BOTS deja la : l'humain qui joue pendant leur presence les
//	  releve, leur affichage s'arrete la veille de son arrivee (quand le film donne encore une vie au
//	  bot apres l'arrivee, son pion reste sans fiche et `coverage.seats.botsRelevesEnJeu` le compte) ;
//	  celui qui ne joue qu'apres eux se charge pendant qu'ils jouent, son affichage commence au
//	  lendemain du dernier (Q23) ; celui qui ne joue pas et part avant eux n'est plus affiche ;
//	sinon, pour un BOT, la place d'un HUMAIN qui a joue avant lui et qu'aucune vie ne montre pendant
//	  la presence du bot : le bot le remplace le temps de son absence (`f235e2f1` : Extinet Hunter sans
//	  corps de 275 a 816, `343 Brew Dog` de 439 a 481), et l'humain retrouve sa fiche apres lui.
//
// Les arrivants que le chainage laissait sans place, leur equipe etant pleine, entrent dans la meme
// affectation : c'est elle qui leur donne la fiche du partant, du bot ou de l'absent.
//
// Comme l'affectation suit l'ordre des debuts, un intervalle ne dispute sa place qu'a des intervalles
// arrives avant lui : la place se decide sans deplacer personne (coloriage glouton d'intervalles, qui
// tient la taille des que les presences certaines la tiennent). Un joueur qui revient peut donc tenir
// une autre fiche que la premiere (`400bb628` : Mayoman8270) — la place se publie alors PAR
// INTERVALLE (`roster[].presence[].seat`). Un intervalle sans presence certaine (parti avant le coup
// d'envoi) sur une place en trop n'est plus affiche. Un intervalle qu'aucune place ne recoit reste sur
// la sienne, et `coverage.seats.placesEnTrop` le dit. Repli nomme et compte
// (`repli_place_ramenee_a_la_taille_d_equipe`) : c'est la regle des places qui decide la fiche.

import (
	"cmp"
	"math"
	"slices"
)

// SeatSourceTaille : la place a ete ramenee dans les places de son equipe (cf. l'en-tete), un REPLI.
const SeatSourceTaille = "taille"

// bilanALaTaille : ce que la regle a fait — les intervalles deplaces ou dont l'affichage est retire,
// les bots releves par un humain (dont ceux a qui le film donne encore une vie apres l'arrivee), les
// absences d'humain qu'un bot a tenues, et les arrivants que le chainage laissait sans place et que la
// regle assoit (comptes par ENTREE : un arrivant a plusieurs intervalles est assis une fois).
type bilanALaTaille struct {
	ramenees, releves, relevesEnJeu, absencesTenues, placees int
	entreesPlacees                                           map[int]bool
}

// placer compte l'entree `i`, arrivant sans place que la regle assoit — une seule fois.
func (b *bilanALaTaille) placer(i int) {
	if b.entreesPlacees[i] {
		return
	}
	if b.entreesPlacees == nil {
		b.entreesPlacees = map[int]bool{}
	}
	b.entreesPlacees[i] = true
	b.placees++
}

// occupationDEquipe : une occupation et la place qui la portait avant la regle (nil : un arrivant que
// le chainage a laisse sans place, son equipe etant pleine).
type occupationDEquipe struct {
	occupation
	origine *placeDeLaTable
}

// ramenerALaTaille ramene chaque equipe a sa capacite.
func (pp *poseDesPlaces) ramenerALaTaille() bilanALaTaille {
	var b bilanALaTaille
	if pp.places == nil {
		return b
	}
	coupees := map[int]bool{}
	for _, t := range pp.equipesDesPlaces() {
		pp.ramenerLEquipe(t, pp.capaciteDe(t), &b, coupees)
	}
	pp.renumeroterLesIntervalles(coupees)
	pp.reposerLesSieges()
	return b
}

// equipesDesPlaces rend les equipes qui tiennent une place, triees.
func (pp *poseDesPlaces) equipesDesPlaces() []int {
	vues := map[int]bool{}
	for _, p := range pp.places {
		if p.equipe != nil {
			vues[*p.equipe] = true
		}
	}
	out := make([]int, 0, len(vues))
	for t := range vues {
		out = append(out, t)
	}
	slices.Sort(out)
	return out
}

// ramenerLEquipe reaffecte les intervalles de l'equipe `t` a ses `c` places gardees (cf. l'en-tete).
// `coupees` recoit les entrees dont un intervalle a ete coupe par l'absence qu'un bot a tenue.
func (pp *poseDesPlaces) ramenerLEquipe(t, c int, b *bilanALaTaille, coupees map[int]bool) {
	places := pp.placesTenuesPar(t)
	sansPlace := pp.arrivantsSansPlace(t)
	if c <= 0 || (len(places) <= c && len(sansPlace) == 0) {
		return
	}
	slices.SortStableFunc(places, func(x, y *placeDeLaTable) int {
		return cmp.Or(cmp.Compare(pp.premiereOccupationCertaine(x), pp.premiereOccupationCertaine(y)),
			cmp.Compare(x.index, y.index))
	})
	gardees := places[:min(c, len(places))]
	occs := pp.occupationsDe(places, sansPlace)
	for _, p := range places {
		p.occupations = nil
	}
	for _, oc := range occs {
		iv := pp.intervalle(oc.occupation)
		if iv.a < iv.de { // aucune presence certaine
			if oc.origine == nil {
				continue
			}
			if !slices.Contains(gardees, oc.origine) {
				iv.aMax = iv.a
				b.ramenees++
			}
			oc.origine.occupations = append(oc.origine.occupations, oc.occupation)
			continue
		}
		q := pp.placeGardeePour(oc, gardees, b, coupees)
		if q == nil {
			q = oc.origine // aucune place gardee ne le recoit : l'equipe depasse encore sa taille
		}
		if q == nil {
			continue // l'arrivant reste sans place (`sansPlace`)
		}
		if oc.origine == nil && pp.roster[oc.entree].SeatSource == SeatSourceIndex {
			pp.asseoirParLaTaille(oc.entree, b)
		}
		q.occupations = append(q.occupations, oc.occupation)
		if q != oc.origine {
			b.ramenees++
			pp.roster[oc.entree].SeatSource = SeatSourceTaille
		}
	}
}

// occupationsDe rend les occupations de ces places et les intervalles des arrivants sans place, dans
// l'ordre de leur debut (puis de leur place et de leur entree : l'ordre ne depend pas de celui des maps).
func (pp *poseDesPlaces) occupationsDe(places []*placeDeLaTable, sansPlace []int) []occupationDEquipe {
	var out []occupationDEquipe
	for _, i := range sansPlace {
		for k := range pp.occ.parEntree[i].presence {
			out = append(out, occupationDEquipe{occupation: occupation{entree: i, intervalle: k}})
		}
	}
	for _, p := range places {
		for _, o := range p.occupations {
			out = append(out, occupationDEquipe{occupation: o, origine: p})
		}
	}
	slices.SortStableFunc(out, func(x, y occupationDEquipe) int {
		return cmp.Or(cmp.Compare(pp.intervalle(x.occupation).de, pp.intervalle(y.occupation).de),
			cmp.Compare(indexDOrigine(x), indexDOrigine(y)), cmp.Compare(x.entree, y.entree),
			cmp.Compare(x.intervalle, y.intervalle))
	})
	return out
}

// indexDOrigine rend l'index de la place d'origine d'une occupation ; un arrivant sans place passe apres
// les occupations deja placees qui commencent avec lui.
func indexDOrigine(oc occupationDEquipe) int {
	if oc.origine == nil {
		return math.MaxInt
	}
	return oc.origine.index
}

// arrivantsSansPlace rend les entrees presentes de l'equipe `t` que le chainage a laissees sans place
// (`index`, l'equipe etait pleine) : la regle les assoit quand une place gardee les recoit — un bot sur
// la fiche d'un humain absent, un humain sur celle d'un bot qu'il releve.
func (pp *poseDesPlaces) arrivantsSansPlace(t int) []int {
	var out []int
	for i, e := range pp.roster {
		o := pp.occ.parEntree[i]
		if e.SeatSource == SeatSourceIndex && len(o.presence) > 0 && o.equipe != nil && *o.equipe == t &&
			pp.places[e.FilmIndex] == nil && !pp.ecarte(i) {
			out = append(out, i)
		}
	}
	return out
}

// asseoirParLaTaille marque l'arrivant sans place `i` que la regle assoit : il n'est plus compte sans
// place (ni, pour un bot qui a une vie, parmi les bots sans place).
func (pp *poseDesPlaces) asseoirParLaTaille(i int, b *bilanALaTaille) {
	pp.roster[i].SeatSource = SeatSourceTaille
	b.placer(i)
	if pp.roster[i].Bot && len(pp.occ.parEntree[i].vies) > 0 {
		pp.sansPlace.avecVie--
	}
}

// placeGardeePour choisit la place gardee d'une occupation, dans l'ordre de la regle (cf. l'en-tete),
// et pose ce que ce choix borne (bot releve, absence coupee). Nil : aucune.
func (pp *poseDesPlaces) placeGardeePour(oc occupationDEquipe, gardees []*placeDeLaTable,
	b *bilanALaTaille, coupees map[int]bool) *placeDeLaTable {
	iv := *pp.intervalle(oc.occupation)
	if slices.Contains(gardees, oc.origine) && len(pp.recouvrements(oc.origine, oc.entree, iv)) == 0 {
		return oc.origine
	}
	var liberee *placeDeLaTable
	fin := math.MinInt
	for _, q := range gardees {
		if len(pp.recouvrements(q, oc.entree, iv)) > 0 {
			continue
		}
		if f, vu := pp.derniereFinAvant(q, iv.de); liberee == nil || (vu && f > fin) {
			liberee, fin = q, f
		}
	}
	if liberee != nil {
		return liberee
	}
	if pp.roster[oc.entree].Bot {
		return pp.tenirUneAbsence(oc, gardees, b, coupees)
	}
	return pp.releverUnBot(oc, gardees, b)
}

// recouvrements rend les occupations de `q` dont la presence certaine recouvre `iv` (presence
// certaine de l'entree `i`), hors succession que le film date ([poseDesPlaces.successionAdmise]).
func (pp *poseDesPlaces) recouvrements(q *placeDeLaTable, i int, iv intervalleDePresence) []occupation {
	var out []occupation
	for _, o := range q.occupations {
		autre := pp.intervalle(o)
		if o.entree != i && autre.a >= autre.de && iv.de <= autre.a && autre.de <= iv.a &&
			!pp.successionAdmise(i, iv, o.entree, *autre) {
			out = append(out, o)
		}
	}
	return out
}

// releverUnBot rend la place gardee dont TOUTES les occupations que l'humain recouvre sont des bots
// arrives avant lui ou avec lui. Nil : aucune. Les deux ne se tiennent jamais ensemble :
//
//	l'humain JOUE pendant leur presence (une vie) : les bots quittent la partie a son arrivee, leur
//	  affichage s'arrete la veille ;
//	il ne joue qu'apres eux : il se charge pendant que les bots jouent, son affichage commence au
//	  lendemain du dernier (Q23, comme [poseDesPlaces.succedeAuBot]) ;
//	il ne joue pas et part avant eux : il ne tient la place a aucune frame, son affichage est retire
//	  (rend nil).
func (pp *poseDesPlaces) releverUnBot(oc occupationDEquipe, gardees []*placeDeLaTable, b *bilanALaTaille) *placeDeLaTable {
	iv := pp.intervalle(oc.occupation)
	for _, q := range gardees {
		bots := pp.recouvrements(q, oc.entree, *iv)
		if len(bots) == 0 || slices.ContainsFunc(bots, func(o occupation) bool {
			return !pp.roster[o.entree].Bot || pp.intervalle(o).de > iv.de
		}) {
			continue
		}
		finDesBots := math.MinInt
		for _, o := range bots {
			finDesBots = max(finDesBots, pp.intervalle(o).a)
		}
		switch {
		case !pp.sansVieEntre(oc.entree, iv.de, min(iv.a, finDesBots)):
			pp.releverLesBots(bots, iv.de, b)
		case iv.a > finDesBots:
			iv.de = finDesBots + 1
		default:
			iv.a, iv.aMax = iv.de-1, iv.de-1
			b.ramenees++
			if oc.origine == nil {
				b.placer(oc.entree)
			}
			return nil
		}
		return q
	}
	return nil
}

// releverLesBots borne l'affichage des bots releves a la veille de l'arrivee `t` de l'humain, et compte
// ceux a qui le film donne encore une vie apres elle (`coverage.seats.botsRelevesEnJeu`).
func (pp *poseDesPlaces) releverLesBots(bots []occupation, t int, b *bilanALaTaille) {
	for _, o := range bots {
		b.releves++
		if !pp.viesFiniesAvant(o.entree, t) {
			b.relevesEnJeu++
		}
		bot := pp.intervalle(o)
		bot.a, bot.aMax = min(bot.a, t-1), min(bot.aMax, t-1)
	}
}

// tenirUneAbsence rend la place gardee dont l'unique occupation que le bot recouvre est celle d'un
// HUMAIN qui a joue avant l'arrivee du bot, qu'aucune vie ne montre pendant sa presence, et qui l'encadre :
// l'humain est absent, le bot le remplace le temps de cette absence. Sa presence est coupee autour de
// celle du bot (la fin devient un nouvel intervalle, sur la meme place). Nil : aucune.
func (pp *poseDesPlaces) tenirUneAbsence(oc occupationDEquipe, gardees []*placeDeLaTable,
	b *bilanALaTaille, coupees map[int]bool) *placeDeLaTable {
	iv := *pp.intervalle(oc.occupation)
	for _, q := range gardees {
		rec := pp.recouvrements(q, oc.entree, iv)
		if len(rec) != 1 || pp.roster[rec[0].entree].Bot {
			continue
		}
		h := rec[0]
		hv := pp.intervalle(h)
		if hv.de >= iv.de || !pp.aJoueAvant(h.entree, iv.de) || !pp.sansVieEntre(h.entree, iv.de, iv.a) {
			continue
		}
		suite := intervalleDePresence{de: iv.a + 1, a: hv.a, aMax: hv.aMax}
		hv.a, hv.aMax = iv.de-1, iv.de-1
		if suite.a >= suite.de {
			pp.occ.parEntree[h.entree].presence = append(pp.occ.parEntree[h.entree].presence, suite)
			q.occupations = append(q.occupations,
				occupation{entree: h.entree, intervalle: len(pp.occ.parEntree[h.entree].presence) - 1})
			coupees[h.entree] = true
		}
		b.absencesTenues++
		return q
	}
	return nil
}

// sansVieEntre dit qu'aucune vie de l'entree `j` ne couvre une frame de [de, a].
func (pp *poseDesPlaces) sansVieEntre(j, de, a int) bool {
	for _, v := range pp.occ.parEntree[j].vies {
		if v[0] <= a && de <= v[1] {
			return false
		}
	}
	return true
}

// renumeroterLesIntervalles remet dans l'ordre du temps les intervalles des entrees coupees, et les
// occupations qui les designent par leur rang.
func (pp *poseDesPlaces) renumeroterLesIntervalles(coupees map[int]bool) {
	for i := range coupees {
		ivs := pp.occ.parEntree[i].presence
		rang := make([]int, len(ivs))
		for k := range rang {
			rang[k] = k
		}
		slices.SortStableFunc(rang, func(x, y int) int { return cmp.Compare(ivs[x].de, ivs[y].de) })
		nouveau := make([]int, len(ivs))
		tries := make([]intervalleDePresence, len(ivs))
		for n, k := range rang {
			nouveau[k], tries[n] = n, ivs[k]
		}
		pp.occ.parEntree[i].presence = tries
		for _, p := range pp.places {
			for k, o := range p.occupations {
				if o.entree == i {
					p.occupations[k].intervalle = nouveau[o.intervalle]
				}
			}
		}
	}
}

// reposerLesSieges pose la place de chaque intervalle, et celle de chaque entree : la place de son
// premier intervalle. Un intervalle tenu ailleurs publie sa propre place (`presence[].seat`).
func (pp *poseDesPlaces) reposerLesSieges() {
	for o, idx := range pp.placesDesIntervalles() {
		iv := pp.intervalle(o)
		iv.place, iv.placeLue = idx, true
		if o.intervalle == 0 {
			pp.roster[o.entree].Seat = idx
		}
	}
}

// placesDesIntervalles rend, pour chaque occupation, le numero de la place qui la porte.
func (pp *poseDesPlaces) placesDesIntervalles() map[occupation]int {
	out := map[occupation]int{}
	for idx, p := range pp.places {
		for _, o := range p.occupations {
			out[o] = idx
		}
	}
	return out
}

// placesTenuesPar rend les places de l'equipe `t`.
func (pp *poseDesPlaces) placesTenuesPar(t int) []*placeDeLaTable {
	var out []*placeDeLaTable
	for _, idx := range pp.ordre {
		if p := pp.places[idx]; p.equipe != nil && *p.equipe == t {
			out = append(out, p)
		}
	}
	return out
}

// premiereOccupationCertaine rend le debut de la premiere presence certaine d'une place, ou
// MaxInt quand aucune ne l'est (un partant d'avant le coup d'envoi).
func (pp *poseDesPlaces) premiereOccupationCertaine(p *placeDeLaTable) int {
	debut := math.MaxInt
	for _, o := range p.occupations {
		if iv := pp.intervalle(o); iv.a >= iv.de {
			debut = min(debut, iv.de)
		}
	}
	return debut
}

// viesFiniesAvant dit que toutes les vies de l'entree `j` finissent avant la frame `t` : un bot releve
// dont une vie suit l'arrivee de l'humain est compte (`coverage.seats.botsRelevesEnJeu`).
func (pp *poseDesPlaces) viesFiniesAvant(j, t int) bool {
	for _, v := range pp.occ.parEntree[j].vies {
		if v[1] >= t {
			return false
		}
	}
	return true
}

// sourceDeLaCapacite dit d'ou vient la capacite des equipes (`coverage.seats.capaciteSource`).
func (pp *poseDesPlaces) sourceDeLaCapacite() string {
	if pp.tailleDuMode > 0 {
		return capaciteDuMode
	}
	return capaciteEstimee
}

// capaciteDuMode / capaciteEstimee : les deux valeurs de `coverage.seats.capaciteSource`.
const (
	capaciteDuMode  = "mode"
	capaciteEstimee = "estimee"
)

// placeAffichee rend la place qu'un intervalle de presence affiche : la sienne quand la regle de la
// taille l'a posee, sinon celle de l'entree.
func placeAffichee(e RosterEntry, iv intervalleDePresence) int {
	if iv.placeLue {
		return iv.place
	}
	return e.Seat
}

// aJoueAvant dit qu'une vie de l'entree `j` commence avant la frame `t` : un humain sans aucune vie
// avant l'arrivee du bot n'est pas « absent », rien ne le montre parti.
func (pp *poseDesPlaces) aJoueAvant(j, t int) bool {
	for _, v := range pp.occ.parEntree[j].vies {
		if v[0] < t {
			return true
		}
	}
	return false
}
