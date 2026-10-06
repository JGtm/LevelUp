package replay

// sieges_places.go — LES PLACES ET LEUR OCCUPATION : l'etat de la pose, la capacite d'une equipe,
// le chainage, les places ouvertes et la borne au successeur (lot M2.3, 2026-09-23). La doctrine
// est ecrite en tete de sieges.go.

import (
	"cmp"
	"math"
	"slices"
	"sort"
	"strconv"
)

// occupation : une entree sur une place, pendant UN de ses intervalles de presence.
type occupation struct {
	entree, intervalle int
}

// placeDeLaTable : un siege de la table du debut, et ceux qui l'ont tenu.
type placeDeLaTable struct {
	index int
	// equipe : l'equipe de la place, celle de ses occupants. Nil tant que personne ne l'a tenue
	// avec une equipe lue (le siege de WNBA Fan A5 sur `b1ad85eb`).
	equipe      *int
	occupations []occupation
}

// poseDesPlaces porte l'etat de la pose : le roster (ecrit en place), les occupants, les places.
type poseDesPlaces struct {
	roster []RosterEntry
	occ    *occupants
	in     entreesDesPlaces
	// origines : les index de la table du debut ; nil sans table.
	origines map[int]bool
	// places : les sieges de la table, puis les places OUVERTES par des arrivants (cf.
	// [poseDesPlaces.ouvrirUnePlace]) ; ordre : leurs numeros, croissants.
	places map[int]*placeDeLaTable
	ordre  []int
	// capacite : la borne basse de la taille d'une equipe que la TABLE donne (cf.
	// [poseDesPlaces.estimerLaCapacite]) ; [poseDesPlaces.capaciteDe] y ajoute ce que les entites
	// lues disent de chaque equipe.
	capacite int
}

func nouvellePoseDesPlaces(roster []RosterEntry, occ *occupants, in entreesDesPlaces) *poseDesPlaces {
	pp := &poseDesPlaces{roster: roster, occ: occ, in: in, origines: siegesDuDebut(in.table)}
	if pp.origines == nil {
		return pp
	}
	pp.places = make(map[int]*placeDeLaTable, len(pp.origines))
	for idx := range pp.origines {
		pp.places[idx] = &placeDeLaTable{index: idx}
		pp.ordre = append(pp.ordre, idx)
	}
	sort.Ints(pp.ordre)
	return pp
}

// estimerLaCapacite pose la borne basse de la taille d'une equipe que la table donne, APRES la
// pose des origines : la plus haute de deux bornes, sous l'hypothese que les equipes d'un mode
// sont de meme taille (4 contre 4, 8 contre 8, 12 contre 12 ; « aucune equipe » en FFA est UNE
// equipe de tous les joueurs) —
//
//	les sieges de la table repartis entre les equipes lues (`ceil(sieges / equipes)`) : un siege
//	  jamais tenu (un partant d'avant le coup d'envoi) y compte ;
//	la plus grande equipe de la table : une table a qui manque un joueur au debut (`e5adf7b2`,
//	  23 sieges pour 12 contre 12) ne retire pas une place a son equipe.
//
// La capacite n'est pas ecrite dans le film (la taille d'equipe du mode n'y est pas lue) : ce qui
// en depend est un REPLI nomme, compte (cf. sieges.go).
func (pp *poseDesPlaces) estimerLaCapacite() {
	equipes := map[int]bool{}
	for i := range pp.roster {
		if t := pp.occ.parEntree[i].equipe; t != nil && len(pp.occ.parEntree[i].presence) > 0 {
			equipes[*t] = true
		}
	}
	n := max(1, len(equipes))
	pp.capacite = (len(pp.ordre) + n - 1) / n
	for t := range equipes {
		pp.capacite = max(pp.capacite, pp.placesDeLEquipe(t))
	}
}

// capaciteDe rend le nombre de places qu'une equipe peut tenir : la borne de la table, ou le plus
// grand nombre de ses entites que le film montre ensemble s'il est plus haut — une borne LUE, qui
// couvre la table a laquelle il manque un joueur dans chaque equipe.
func (pp *poseDesPlaces) capaciteDe(t int) int {
	return max(pp.capacite, pp.occ.simultanees[t])
}

// estUnePlace dit si un index est un siege de la table du debut.
func (pp *poseDesPlaces) estUnePlace(idx int) bool { return pp.places[idx] != nil }

// asseoir pose une entree sur une place, pour tous ses intervalles de presence.
func (pp *poseDesPlaces) asseoir(i int, p *placeDeLaTable, source string) {
	pp.roster[i].Seat, pp.roster[i].SeatSource = p.index, source
	for k := range pp.occ.parEntree[i].presence {
		p.occupations = append(p.occupations, occupation{entree: i, intervalle: k})
	}
	if p.equipe == nil {
		p.equipe = pp.occ.parEntree[i].equipe
	}
}

// poserLesOrigines : LECTURE `lu` — chaque entree presente dont l'index est un siege de la table.
func (pp *poseDesPlaces) poserLesOrigines() {
	for i, e := range pp.roster {
		if p := pp.places[e.FilmIndex]; p != nil && len(pp.occ.parEntree[i].presence) > 0 {
			pp.asseoir(i, p, SeatSourceLu)
		}
	}
}

// ouvrirAuCoupDEnvoi : l'occupant qu'un siege de la table NOMME (son xuid y est ecrit), quand sa
// presence ne vient que de ses vies (repli : aucune entite ne le porte), la tient depuis la frame 0.
// C'est une LECTURE, pas un prolongement : la table de `chunk_00` l'assoit au coup d'envoi (revue
// M2-R7 — sans elle, sa place s'affichait VIDE pendant le preambule, alors que le joueur est la et
// n'a seulement pas encore apparu). Le xuid, et pas l'index : un arrivant qui reprend l'index d'un
// partant n'etait pas a la table. Une presence lue (entite, BOT_METADATA) n'y passe pas : l'entite
// dit deja s'il etait la au debut.
func (pp *poseDesPlaces) ouvrirAuCoupDEnvoi() {
	for _, s := range pp.in.table.Seats {
		x := strconv.FormatUint(s.XUID, 10)
		for i, e := range pp.roster {
			o := &pp.occ.parEntree[i]
			if e.XUID == x && e.FilmIndex == s.FilmIndex && !o.lue && len(o.presence) > 0 {
				o.presence[0].de = 0
			}
		}
	}
}

// arrivants rend les entrees presentes dont l'index n'est pas un siege et qui n'ont pas encore de
// place, dans l'ordre de leur arrivee.
func (pp *poseDesPlaces) arrivants() []int {
	var out []int
	for i, e := range pp.roster {
		if pp.places[e.FilmIndex] == nil && len(pp.occ.parEntree[i].presence) > 0 &&
			pp.roster[i].SeatSource == SeatSourceLu {
			out = append(out, i)
		}
	}
	ordreDesArrivants(pp.roster, pp.occ, out)
	return out
}

// marquerLesArrivantsSansPresence : une entree hors de la table qu'aucune presence ne couvre n'a
// de place a aucun instant — son siege reste son index, et sa provenance le dit (`index`), au lieu
// de se lire comme un siege de la table.
func (pp *poseDesPlaces) marquerLesArrivantsSansPresence() {
	for i, e := range pp.roster {
		if !pp.origines[e.FilmIndex] && len(pp.occ.parEntree[i].presence) == 0 {
			pp.roster[i].SeatSource = SeatSourceIndex
		}
	}
}

// libre dit si une place est libre pendant TOUTES les presences CERTAINES d'une entree. Deux
// presences qui se recouvrent ne se contredisent pas quand le recouvrement est une SUCCESSION que
// le film date ([poseDesPlaces.successionAdmise]).
func (pp *poseDesPlaces) libre(p *placeDeLaTable, i int) bool {
	for _, iv := range pp.occ.parEntree[i].presence {
		for _, o := range p.occupations {
			autre := pp.occ.parEntree[o.entree].presence[o.intervalle]
			if iv.de <= autre.a && autre.de <= iv.a && !pp.successionAdmise(i, iv, o.entree, autre) {
				return false
			}
		}
	}
	return true
}

// successionAdmise dit que deux presences certaines d'une meme place ne se recouvrent que par une
// succession : un relais a la frame ([poseDesPlaces.relaisALaFrame]), ou l'humain qui succede a un
// bot ([poseDesPlaces.succedeAuBot]), dans un sens ou dans l'autre.
func (pp *poseDesPlaces) successionAdmise(i int, iv intervalleDePresence, j int, autre intervalleDePresence) bool {
	return pp.relaisALaFrame(i, iv, j, autre) || pp.succedeAuBot(i, iv, j, autre) || pp.succedeAuBot(j, autre, i, iv)
}

// relaisALaFrame dit que deux presences ne partagent que la frame d'un RELAIS — l'une commence a la
// frame ou l'autre finit, apres elle — et que l'une des deux est celle d'un bot que BOT_METADATA
// date a la frame pres : le depart et l'arrivee tombent dans la meme frame du document
// (`572e236b` : la derniere vie de biOly goLab1054 finit a la frame 582, ou `343 Bachici` est
// declare). Deux bornes lues aux images-cles qui se touchent sont deux occupants a la meme
// image-cle : la contradiction reste.
func (pp *poseDesPlaces) relaisALaFrame(i int, iv intervalleDePresence, j int, autre intervalleDePresence) bool {
	if !pp.occ.parEntree[i].declaree && !pp.occ.parEntree[j].declaree {
		return false
	}
	return (iv.de == autre.a && autre.de < iv.de) || (autre.de == iv.a && iv.de < autre.de)
}

// succedeAuBot dit que l'entree `h` est l'HUMAIN qui remplace le bot `b` sur sa place (Q23 : un
// humain qui arrive remplace le bot) : ARRIVE — lu par son entite, a l'image-cle pres — APRES le
// debut de la declaration du bot et avant son retrait EXACT, sans aucune vie avant ce retrait (`43e96765` :
// Cmillward21 lu a l'image-cle de la frame 1145, `343 PardonMy` declare jusqu'a la frame 1344,
// premier corps de Cmillward21 a la frame 1367). Le jeu tient les deux pendant que l'humain se
// charge ; la place est au bot jusqu'a son retrait.
func (pp *poseDesPlaces) succedeAuBot(b int, ivb intervalleDePresence, h int, ivh intervalleDePresence) bool {
	if !pp.roster[b].Bot || !pp.occ.parEntree[b].declaree || pp.roster[h].Bot {
		return false
	}
	// STRICTEMENT apres le debut du bot : un occupant deja la quand le bot arrive (au coup d'envoi,
	// avant sa premiere vie) ne le remplace pas, il le cotoie.
	if ivh.de <= ivb.de || ivh.de > ivb.a {
		return false
	}
	for _, v := range pp.occ.parEntree[h].vies {
		if v[0] <= ivb.a {
			return false
		}
	}
	return true
}

// placeDuBotRemplace rend la place de son equipe ou l'arrivant `i` succede a un bot
// ([poseDesPlaces.succedeAuBot]) : c'est la place du bot qu'il remplace, avant tout chainage — celle
// du bot qui se retire LE PREMIER apres son arrivee quand plusieurs le pourraient (le jeu retire un
// bot quand l'humain rejoint). Nil : aucune.
func (pp *poseDesPlaces) placeDuBotRemplace(i, t int) *placeDeLaTable {
	iv := pp.occ.parEntree[i].presence[0]
	var elue *placeDeLaTable
	retrait := math.MaxInt
	for _, idx := range pp.ordre {
		p := pp.places[idx]
		if p.equipe == nil || *p.equipe != t || !pp.libre(p, i) {
			continue
		}
		for _, o := range p.occupations {
			if b := pp.intervalle(o); pp.succedeAuBot(o.entree, *b, i, iv) && b.a < retrait {
				elue, retrait = p, b.a
			}
		}
	}
	return elue
}

// memeEquipe dit si une entree peut tenir une place d'apres les equipes : une equipe inconnue,
// d'un cote ou de l'autre, ne contredit rien.
func memeEquipe(p *placeDeLaTable, t *int) bool {
	return p.equipe == nil || t == nil || *p.equipe == *t
}

// chainerLesArrivants : les REPLIS `apparie` et `ouverte` (cf. sieges.go), sur les arrivants que
// la lecture n'a pas places. Rend les chainages poses, les places ouvertes et les entrees restees
// sans place.
func (pp *poseDesPlaces) chainerLesArrivants() (apparies, ouvertes, sansPlace int) {
	for _, i := range pp.arrivants() {
		t := pp.occ.parEntree[i].equipe
		if t == nil {
			pp.roster[i].SeatSource = SeatSourceIndex
			sansPlace++
			continue
		}
		if p := pp.placeParChainage(i, *t); p != nil {
			pp.asseoir(i, p, SeatSourceApparie)
			apparies++
			continue
		}
		if pp.placesDeLEquipe(*t) >= pp.capaciteDe(*t) {
			pp.roster[i].SeatSource = SeatSourceIndex
			sansPlace++
			continue
		}
		pp.asseoir(i, pp.ouvrirUnePlace(i), SeatSourceOuverte)
		ouvertes++
	}
	return apparies, ouvertes, sansPlace
}

// placeParChainage choisit la place d'un arrivant dans son equipe, dans l'ordre du repli : la place
// du bot auquel il succede ([poseDesPlaces.placeDuBotRemplace]), la place liberee le plus tot avant
// son arrivee (un remplacant prend la place du partant), puis une place de son equipe sans occupant
// anterieur, puis — si son equipe n'a pas sa capacite — un siege de la table jamais tenu. Nil :
// aucune.
func (pp *poseDesPlaces) placeParChainage(i, t int) *placeDeLaTable {
	if p := pp.placeDuBotRemplace(i, t); p != nil {
		return p
	}
	arrivee := pp.occ.parEntree[i].presence[0].de
	var liberee, sansAnterieur *placeDeLaTable
	liberation := math.MaxInt
	for _, idx := range pp.ordre {
		p := pp.places[idx]
		if p.equipe == nil || *p.equipe != t || !pp.libre(p, i) {
			continue
		}
		fin, anterieur := pp.derniereFinAvant(p, arrivee)
		switch {
		case anterieur && fin < liberation:
			liberee, liberation = p, fin
		case !anterieur && sansAnterieur == nil:
			sansAnterieur = p
		}
	}
	switch {
	case liberee != nil:
		return liberee
	case sansAnterieur != nil:
		return sansAnterieur
	case pp.placesDeLEquipe(t) >= pp.capaciteDe(t):
		return nil
	}
	for _, idx := range pp.ordre {
		if p := pp.places[idx]; p.equipe == nil && len(p.occupations) == 0 && pp.libre(p, i) {
			return p
		}
	}
	return nil
}

// premierePlaceHorsIndex : le premier numero de place qu'aucun index de joueur ne peut porter —
// l'index est lu sur 6 bits (`readManagedPlayerDefaultState`, 0..63). Ce n'est pas un seuil : une
// place ouverte qui ne peut pas prendre l'index de son arrivant (deja une place) prend un numero
// au-dela, qui ne se confond avec aucun index.
const premierePlaceHorsIndex = 1 << 6

// ouvrirUnePlace ajoute a l'equipe d'un arrivant la place que la table du debut ne portait pas :
// son propre index quand aucune place ne l'a, sinon le premier numero libre au-dela des index.
func (pp *poseDesPlaces) ouvrirUnePlace(i int) *placeDeLaTable {
	num := pp.roster[i].FilmIndex
	if pp.places[num] != nil {
		num = premierePlaceHorsIndex
		for pp.places[num] != nil {
			num++
		}
	}
	p := &placeDeLaTable{index: num}
	pp.places[num] = p
	pp.ordre = append(pp.ordre, num)
	sort.Ints(pp.ordre)
	return p
}

// derniereFinAvant rend la fin CERTAINE la plus tardive des occupations d'une place qui finissent
// avant un instant — ou A cet instant : sur une place que [poseDesPlaces.libre] a admise, une fin a
// la frame d'arrivee est un relais a la frame —, et dit s'il y en a une.
func (pp *poseDesPlaces) derniereFinAvant(p *placeDeLaTable, t int) (int, bool) {
	fin, vu := -1, false
	for _, o := range p.occupations {
		if a := pp.occ.parEntree[o.entree].presence[o.intervalle].a; a <= t && a > fin {
			fin, vu = a, true
		}
	}
	return fin, vu
}

// placesDeLEquipe compte les places tenues par une equipe.
func (pp *poseDesPlaces) placesDeLEquipe(t int) int {
	n := 0
	for _, p := range pp.places {
		if p.equipe != nil && *p.equipe == t {
			n++
		}
	}
	return n
}

// bornerAuSuccesseur borne l'AFFICHAGE de chaque occupant a la veille de l'arrivee du suivant
// sur la meme place, et compte les presences certaines qui se recouvrent. L'humain qui succede a un
// bot ([poseDesPlaces.succedeAuBot]) commence au lendemain de son retrait exact, et se compte a part
// (journalise par [poserLesSieges]) ; un relais a la frame ([poseDesPlaces.relaisALaFrame]) borne
// le partant sans etre un chevauchement.
func (pp *poseDesPlaces) bornerAuSuccesseur() (bornes, chevauchements, successions int) {
	for _, idx := range pp.ordre {
		occ := pp.places[idx].occupations
		slices.SortStableFunc(occ, func(a, b occupation) int {
			return cmp.Compare(pp.intervalle(a).de, pp.intervalle(b).de)
		})
		for k := 0; k+1 < len(occ); k++ {
			cur, suiv := pp.intervalle(occ[k]), pp.intervalle(occ[k+1])
			if pp.succedeAuBot(occ[k].entree, *cur, occ[k+1].entree, *suiv) {
				suiv.de = cur.a + 1
				successions++
			}
			if cur.a >= suiv.de {
				if !pp.relaisALaFrame(occ[k].entree, *cur, occ[k+1].entree, *suiv) {
					chevauchements++
				}
				cur.a = max(cur.de, suiv.de-1)
			}
			if cur.aMax >= suiv.de {
				bornes++
				cur.aMax = max(cur.a, suiv.de-1)
			}
		}
	}
	return bornes, chevauchements, successions
}

// retirerLesAffichagesVides retire les intervalles que la borne au successeur a VIDES (affichage
// fini avant son debut) : un occupant releve des son arrivee n'est affiche a aucune frame de cet
// intervalle, et le publier dirait le contraire. Joue APRES la borne, et apres le dernier usage
// des occupations (qui designent les intervalles par leur rang).
func (pp *poseDesPlaces) retirerLesAffichagesVides() {
	for i := range pp.occ.parEntree {
		gardes := pp.occ.parEntree[i].presence[:0]
		for _, iv := range pp.occ.parEntree[i].presence {
			if iv.aMax >= iv.de {
				gardes = append(gardes, iv)
			}
		}
		pp.occ.parEntree[i].presence = gardes
	}
}

// intervalle rend l'intervalle de presence qu'une occupation designe, pour le modifier en place.
func (pp *poseDesPlaces) intervalle(o occupation) *intervalleDePresence {
	return &pp.occ.parEntree[o.entree].presence[o.intervalle]
}

// tenirLesDerniersJusquALaFin : le REPLI des presences SANS ENTITE (cf. sieges.go) — le dernier
// occupant de chaque place reste affiche jusqu'a la fin. Sans table, la place est le siege publie.
// Rend le nombre d'entrees ainsi prolongees.
func (pp *poseDesPlaces) tenirLesDerniersJusquALaFin() int {
	dernier := map[int]occupation{}
	vu := map[int]bool{}
	for i := range pp.roster {
		for k, iv := range pp.occ.parEntree[i].presence {
			s := pp.roster[i].Seat
			if o, deja := dernier[s]; !deja || iv.aMax > pp.intervalle(o).aMax {
				dernier[s], vu[s] = occupation{entree: i, intervalle: k}, true
			}
		}
	}
	fin, n := pp.in.horloge.frames-1, 0
	for s := range vu {
		if iv := pp.intervalle(dernier[s]); iv.aMax < fin {
			iv.aMax = fin
			n++
		}
	}
	return n
}

// publierLesPresences ecrit la presence de chaque entree dans le roster publie.
func (pp *poseDesPlaces) publierLesPresences() {
	for i := range pp.roster {
		ivs := pp.occ.parEntree[i].presence
		if len(ivs) == 0 {
			pp.roster[i].Presence = nil
			continue
		}
		out := make([]PresenceInterval, 0, len(ivs))
		for _, iv := range ivs {
			pi := PresenceInterval{From: iv.de, To: iv.a}
			if iv.aMax > iv.a {
				m := iv.aMax
				pi.ToMax = &m
			}
			out = append(out, pi)
		}
		pp.roster[i].Presence = out
	}
}
