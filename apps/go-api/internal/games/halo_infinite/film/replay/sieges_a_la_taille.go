package replay

// sieges_a_la_taille.go — AUCUNE EQUIPE NE TIENT PLUS DE PLACES QUE SA TAILLE (controle n° 8 des
// controles de la v8 ; decision utilisateur du 2026-10-21, sur la REGLE DES PLACES de sieges.go).
//
// # CE QUI FAISAIT UNE PLACE DE TROP
//
// Une place est un index de joueur du film, et l'index n'est pas une place d'EQUIPE : un partant
// d'avant le coup d'envoi laisse son index dans la table du debut, un joueur qui revient prend
// l'index qu'il avait, un arrivant prend l'index libre que le jeu lui donne — pas forcement celui
// du partant qu'il remplace. Une equipe peut donc voir passer plus d'index que sa taille, et le
// rejeu, qui rend une fiche par place, lui en montrait autant (`4f77afc1`, Eagle : 13 fiches en
// BTB, une toujours vide).
//
// # LA REGLE
//
// Une fiche est une PLACE D'EQUIPE, et une equipe en a autant que sa taille ([poseDesPlaces.capaciteDe] :
// la taille officielle de la playlist quand la table du titre la declare, sinon la capacite estimee).
// Les places d'une equipe en surnombre sont departagees par leur premiere occupation CERTAINE : les
// premieres tenues restent, les autres rendent leurs occupants, qui prennent une place gardee de leur
// equipe —
//
//	LIBRE pendant leurs presences certaines, celle qui s'est liberee le plus tard avant leur arrivee
//	  (un remplacant prend la place du partant) ;
//	sinon tenue par un BOT deja la a leur arrivee et dont les vies sont finies : l'humain le remplace
//	  (Q23), l'affichage du bot s'arrete la veille. Un bot qui joue encore n'est pas releve : le
//	  film montre alors l'equipe au-dela de sa taille (`3d58eb37` : cinq corps de l'equipe 1 en
//	  4v4 pendant plus de cinq minutes), et le compteur le dit ;
//	ou tenue par un occupant ARRIVE APRES eux, qui passe sur une autre place gardee libre pendant
//	  toute sa presence (`343 Darkstar`, deux images, sur la place qu'AJM002 vient de quitter).
//
// Un occupant sans presence certaine (parti avant le coup d'envoi) n'est plus affiche. Un occupant
// qu'aucune place gardee ne recoit reste ou il est : l'equipe depasse encore sa taille, et
// `coverage.seats.placesEnTrop` le dit. Repli nomme et compte
// (`repli_place_ramenee_a_la_taille_d_equipe`) : c'est la regle des places qui decide la fiche, pas
// une lecture du film.

import (
	"cmp"
	"math"
	"slices"
)

// SeatSourceTaille : la place a ete ramenee dans les places de son equipe (cf. l'en-tete), un REPLI.
const SeatSourceTaille = "taille"

// ramenerALaTaille ramene chaque equipe a sa capacite. Rend les entrees deplacees ou dont
// l'affichage est retire, et les bots releves par un humain.
func (pp *poseDesPlaces) ramenerALaTaille() (ramenees, releves int) {
	if pp.places == nil {
		return 0, 0
	}
	for _, t := range pp.equipesDesPlaces() {
		r, b := pp.ramenerLEquipe(t, pp.capaciteDe(t))
		ramenees += r
		releves += b
	}
	return ramenees, releves
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

// ramenerLEquipe ramene les places de l'equipe `t` a `c` (cf. l'en-tete).
func (pp *poseDesPlaces) ramenerLEquipe(t, c int) (ramenees, releves int) {
	places := pp.placesTenuesPar(t)
	if c <= 0 || len(places) <= c {
		return 0, 0
	}
	slices.SortStableFunc(places, func(a, b *placeDeLaTable) int {
		return cmp.Or(cmp.Compare(pp.premiereOccupationCertaine(a), pp.premiereOccupationCertaine(b)),
			cmp.Compare(a.index, b.index))
	})
	gardees := places[:c]
	for _, i := range pp.occupantsDe(places[c:]) {
		if !pp.aUnePresenceCertaine(i) {
			pp.retirerLAffichage(i)
			ramenees++
			continue
		}
		if b, ok := pp.placerDansLesGardees(i, gardees); ok {
			ramenees++
			releves += b
		}
	}
	return ramenees, releves
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

// occupantsDe rend les entrees assises sur ces places, sans doublon, dans l'ordre de leur arrivee.
func (pp *poseDesPlaces) occupantsDe(places []*placeDeLaTable) []int {
	vues := map[int]bool{}
	var out []int
	for _, p := range places {
		for _, o := range p.occupations {
			if !vues[o.entree] {
				vues[o.entree] = true
				out = append(out, o.entree)
			}
		}
	}
	slices.SortStableFunc(out, func(a, b int) int {
		return cmp.Or(cmp.Compare(pp.premierDebutCertain(a), pp.premierDebutCertain(b)), cmp.Compare(a, b))
	})
	return out
}

// premierDebutCertain rend le debut de la premiere presence certaine d'une entree, ou MaxInt.
func (pp *poseDesPlaces) premierDebutCertain(i int) int {
	for _, iv := range pp.occ.parEntree[i].presence {
		if iv.a >= iv.de {
			return iv.de
		}
	}
	return math.MaxInt
}

// aUnePresenceCertaine dit si une entree est certainement la a une frame au moins.
func (pp *poseDesPlaces) aUnePresenceCertaine(i int) bool {
	return pp.premierDebutCertain(i) != math.MaxInt
}

// retirerLAffichage borne l'affichage d'une entree a sa presence certaine, vide : elle ne rend
// aucune fiche ([poseDesPlaces.retirerLesAffichagesVides] retire ensuite ses intervalles).
func (pp *poseDesPlaces) retirerLAffichage(i int) {
	for k := range pp.occ.parEntree[i].presence {
		iv := &pp.occ.parEntree[i].presence[k]
		iv.aMax = iv.a
	}
}

// accueil : ce qu'il faut faire d'une place gardee pour y asseoir une entree — les bots a relever
// (occupation et debut de l'arrivee qui les releve), les entrees a deplacer et leur place.
type accueil struct {
	place    *placeDeLaTable
	releves  []releve
	deplaces []deplacement
	libere   int
}

// deplacement : une entree deja assise sur la place qui accueille, et la place gardee qui la recoit.
type deplacement struct {
	entree int
	place  *placeDeLaTable
}

// releve : l'occupation d'un bot que l'arrivee d'un humain borne a la veille de `debut`.
type releve struct {
	occ   occupation
	debut int
}

// placerDansLesGardees assoit l'entree `i` sur une place gardee qui l'accueille (cf. l'en-tete) :
// une place libre d'abord, puis la moins chargee en releves et deplacements, la place liberee le
// plus tard avant son arrivee a egalite. Rend le nombre de bots releves, et faux quand aucune place
// ne l'accueille.
func (pp *poseDesPlaces) placerDansLesGardees(i int, gardees []*placeDeLaTable) (int, bool) {
	var elu *accueil
	arrivee := pp.premierDebutCertain(i)
	for _, q := range gardees {
		a, ok := pp.accueillir(q, i, gardees)
		if !ok {
			continue
		}
		a.libere, _ = pp.derniereFinAvant(q, arrivee)
		if elu == nil || meilleurAccueil(a, *elu) {
			elu = &a
		}
	}
	if elu == nil {
		return 0, false
	}
	for _, r := range elu.releves {
		iv := pp.intervalle(r.occ)
		iv.a, iv.aMax = min(iv.a, r.debut-1), min(iv.aMax, r.debut-1)
	}
	for _, d := range elu.deplaces {
		pp.deplacer(d.entree, d.place)
	}
	pp.deplacer(i, elu.place)
	return len(elu.releves), true
}

// meilleurAccueil dit si `a` vaut mieux que `b` : moins de releves et de deplacements, puis la place
// liberee le plus tard.
func meilleurAccueil(a, b accueil) bool {
	ca, cb := len(a.releves)+len(a.deplaces), len(b.releves)+len(b.deplaces)
	if ca != cb {
		return ca < cb
	}
	return a.libere > b.libere
}

// accueillir dit comment la place `q` recevrait l'entree `i`, et si elle le peut : chaque occupation
// que les presences certaines de `i` recouvrent est un bot deja la a l'arrivee (releve), ou une
// entree arrivee apres `i` qu'une AUTRE place gardee recoit libre (deplacee).
func (pp *poseDesPlaces) accueillir(q *placeDeLaTable, i int, gardees []*placeDeLaTable) (accueil, bool) {
	a := accueil{place: q}
	for _, iv := range pp.occ.parEntree[i].presence {
		if iv.a < iv.de {
			continue
		}
		for _, o := range q.occupations {
			if pp.recouvre(i, iv, o) && !pp.resoudre(&a, i, iv, o, gardees) {
				return accueil{}, false
			}
		}
	}
	return a, true
}

// recouvre dit que l'occupation `o` d'une AUTRE entree recouvre la presence certaine `iv` de `i`,
// hors succession que le film date ([poseDesPlaces.successionAdmise]).
func (pp *poseDesPlaces) recouvre(i int, iv intervalleDePresence, o occupation) bool {
	autre := pp.intervalle(o)
	return o.entree != i && autre.a >= autre.de && iv.de <= autre.a && autre.de <= iv.a &&
		!pp.successionAdmise(i, iv, o.entree, *autre)
}

// resoudre range dans l'accueil `a` l'occupation `o` que la presence `iv` de `i` recouvre : un bot
// aux vies finies deja la (releve), ou une entree arrivee apres `i` qu'une autre place gardee recoit
// (deplacee). Faux : la place ne peut pas recevoir `i`.
func (pp *poseDesPlaces) resoudre(a *accueil, i int, iv intervalleDePresence, o occupation,
	gardees []*placeDeLaTable) bool {
	autre := pp.intervalle(o)
	switch {
	case pp.roster[o.entree].Bot && !pp.roster[i].Bot && autre.de < iv.de && pp.viesFiniesAvant(o.entree, iv.de):
		a.releves = append(a.releves, releve{occ: o, debut: iv.de})
		return true
	case autre.de > iv.de:
		if a.deplace(o.entree) {
			return true
		}
		p := pp.placeLibrePour(o.entree, gardees, a.place, a.deplaces)
		if p == nil {
			return false
		}
		a.deplaces = append(a.deplaces, deplacement{entree: o.entree, place: p})
		return true
	}
	return false
}

// placeLibrePour rend une place gardee, autre que `sauf`, que la sienne et que celles deja promises
// (`deja`), libre pendant toutes les presences certaines de l'entree `j` ([poseDesPlaces.libre]) ; nil : aucune.
func (pp *poseDesPlaces) placeLibrePour(j int, gardees []*placeDeLaTable, sauf *placeDeLaTable,
	deja []deplacement) *placeDeLaTable {
	for _, p := range gardees {
		prise := slices.ContainsFunc(deja, func(d deplacement) bool { return d.place == p })
		if p != sauf && !prise && p.index != pp.roster[j].Seat && pp.libre(p, j) {
			return p
		}
	}
	return nil
}

// deplacer retire l'entree `i` de sa place et l'assoit sur `p` (provenance `taille`).
func (pp *poseDesPlaces) deplacer(i int, p *placeDeLaTable) {
	if ancienne := pp.places[pp.roster[i].Seat]; ancienne != nil {
		ancienne.occupations = slices.DeleteFunc(ancienne.occupations, func(o occupation) bool { return o.entree == i })
	}
	pp.asseoir(i, p, SeatSourceTaille)
}

// deplace dit si l'entree `j` est deja a deplacer dans cet accueil.
func (a accueil) deplace(j int) bool {
	return slices.ContainsFunc(a.deplaces, func(d deplacement) bool { return d.entree == j })
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

// viesFiniesAvant dit que toutes les vies de l'entree `j` finissent avant la frame `t` : un bot qui
// joue encore quand l'humain arrive n'est pas releve — le film montre alors une equipe au-dela de sa
// taille, et `coverage.seats.placesEnTrop` le dit plutot que de cacher un joueur qui joue.
func (pp *poseDesPlaces) viesFiniesAvant(j, t int) bool {
	for _, v := range pp.occ.parEntree[j].vies {
		if v[1] >= t {
			return false
		}
	}
	return true
}
