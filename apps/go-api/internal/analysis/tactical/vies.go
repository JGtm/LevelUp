package tactical

// vies.go — CE QU'UNE VIE PRODUIT EN PLUS DE SON OCCUPATION : sa MORT et sa ROUTE.
//
// # UNE VIE NOMMEE EST CLOSE PAR UNE MORT, ET C'EST TOUTE LA MESURE
//
// Le document de rejeu ne publie AUCUNE liste de morts datees par joueur : `neutralDeaths`
// ne couvre que les morts que personne ne revendique (chute, suicide). La seule source de
// « ce joueur est mort a cet instant » disponible HORS LIGNE est donc la fin d'une vie
// NOMMEE — et elle est fiable par construction : c'est le fil des morts du film qui pose
// l'identite de la victime sur la vie que sa mort termine (`replay.nameLivesByDeaths`,
// appariement glouton dans une fenetre de 150 ms). Une vie sans nom est une vie que rien
// n'a close : sur le film de reference, les 15 vies anonymes sont 4 vies anterieures au
// debut reel du match et 6 SURVIVANTS de fin de partie.
//
// Consequence assumee : un joueur qui SURVIT a la fin du match ne produit pas de mort, et
// c'est juste. Un joueur dont la derniere vie n'a pas ete appariee non plus, et c'est une
// LACUNE — comptee nulle part, parce qu'elle est indistinguable de la premiere.
//
// # ON NE COMPTE MORT QUE CE QU'ON SAIT MORT (correction P0-1, revue du 2026-09-06)
//
// Un voisin dont on n'observe aucune position a l'instant d'une mort N'EST PAS un voisin
// mort. Deux situations le produisent en permanence, et les confondre avec un deces rendait
// des morts « isolees » alors qu'un coequipier etait a trois metres :
//
//	EN VEHICULE      un occupant ne replique plus son bipede, et la primitive d'episodes
//	                 n'attribue que 15,6 a 21,1 % des vies de vehicule ;
//	SURVIVANT        la derniere vie d'un joueur qui finit le match n'est close par aucune
//	                 mort, donc le fil des morts ne la nomme pas.
//
// Chaque voisin porte donc un STATUT a trois valeurs — `vivant` (position connue, avec sa
// distance), `mort` (une vie nommee de ce joueur a ete close a d <= t, et rien n'a ete
// observe depuis), `inconnu` (aucune position observable, aucune preuve de mort). C'est le
// service qui en tire un verdict d'isolement.
//
// # LA DISTANCE EST HORIZONTALE, ET DEUX ETAGES LISENT ZERO
//
// `math.Hypot(dx, dy)` ignore Z : un coequipier situe juste au-dessus ou au-dessous, separe
// par une dalle, est mesure a 0 m et compte comme « a portee ». C'est une LIMITE CONNUE de
// la mesure, pas un defaut a chercher — toutes les lectures de l'onglet sont des vues du
// dessus, et corriger cela demanderait une notion d'etage que la grille n'a pas.
//
// # LES VOISINS SONT MESURES ICI, L'ISOLEMENT SE DECIDE AILLEURS
//
// Le film NE PORTE PAS LES EQUIPES (`Track.Team` vaut -1 pour tout le monde : le camp vit
// dans la base). Une cuisson hors ligne ne peut donc pas dire « il etait seul » — elle ne
// sait pas qui est un coequipier. Ce fichier mesure donc la seule chose qu'il connaisse :
// LA DISTANCE A CHAQUE AUTRE JOUEUR NOMME VIVANT a l'instant de la mort. Le service joint
// les equipes, applique le rayon du match et tranche (cf. isolation.go).
//
// « VIVANT A CET INSTANT » = une vie de ce joueur couvre cet instant. Sa position est la
// derniere connue, exactement comme pour l'occupation — embarquement compris : un
// coequipier en Warthog est a la position du Warthog.
//
// # LA ROUTE EST LA SORTIE DE SPAWN
//
// Les 15 premieres secondes de CHAQUE vie (decision produit du plan), en cellules
// ORDONNEES, doublons consecutifs fusionnes : ce qui compte est le CHEMIN, pas le temps
// passe dans chaque case — celui-la est deja mesure par l'occupation. L'instant
// contributeur est le debut de la vie.

import (
	"math"
	"sort"
)

// FenetreRouteMs est la duree, en millisecondes, pendant laquelle on suit une vie apres sa
// reapparition : 15 s (plan tactique, decision produit du 2026-09-05).
//
// POURQUOI 15 s. C'est la duree qui separe une SORTIE de spawn d'un deplacement ordinaire :
// au-dela, le joueur a rejoint le jeu et sa trajectoire ne dit plus par ou l'on sort. La
// borne est un choix produit, pas une mesure — et elle est ici, nommee, pour qu'on sache
// quoi changer si elle se revele fausse.
const FenetreRouteMs = 15_000

// Les trois STATUTS d'un voisin a l'instant d'une mort.
//
// ILS NE SE REPLIENT PAS L'UN SUR L'AUTRE : `inconnu` n'est pas un `mort` prudent, c'est une
// absence de mesure — et c'est precisement la distinction qui manquait (revue P0-1).
const (
	// StatutVivant : position CONNUE a cet instant (vie nommee couvrant l'instant, ou
	// episode d'embarquement dont le vehicule a un point). `DistanceM` vaut alors.
	StatutVivant = "vivant"
	// StatutMort : une vie NOMMEE de ce joueur a ete close par une mort a d <= t, et rien
	// n'a ete observe de lui depuis. `DistanceM` est nulle et sans objet.
	StatutMort = "mort"
	// StatutInconnu : aucune position observable a cet instant, ET aucune preuve de mort.
	// C'est le cas d'un occupant de vehicule non attribue et d'un survivant de fin de
	// partie. `DistanceM` est nulle et sans objet.
	StatutInconnu = "inconnu"
)

// Voisin est un autre joueur NOMME, avec son statut a l'instant d'une mort. Ni equipe ni
// camp : le film ne les porte pas.
type Voisin struct {
	XUID   string `json:"xuid"`
	Statut string `json:"statut"`
	// DistanceM n'a de sens que sous StatutVivant.
	DistanceM float64 `json:"distance_m,omitempty"`
}

// MortMesuree est la fin d'une vie nommee : ou, quand, et qui etait encore debout autour.
type MortMesuree struct {
	// Frame est l'instant de la mort sur l'axe du rejeu (la fin de la vie).
	Frame int `json:"frame"`
	// X, Y : la position de la victime, en metres monde.
	X float64 `json:"x"`
	Y float64 `json:"y"`
	// PositionInconnue : la mort a eu lieu, mais le film ne dit pas OU (embarquement sans
	// point de vehicule). Elle n'est alors ni peinte ni examinee — et elle se compte. Lui
	// preter le point de montee serait l'invention que ce fichier interdit partout ailleurs.
	PositionInconnue bool `json:"position_inconnue,omitempty"`

	// Voisins : TOUS les autres joueurs nommes, avec leur STATUT a cet instant, tries par
	// xuid. Vide quand la position de la mort est inconnue — sans elle, aucune distance
	// n'est mesurable.
	Voisins []Voisin `json:"voisins"`
}

// Route est la sortie de spawn d'une vie : les cellules traversees pendant FenetreRouteMs.
type Route struct {
	// DebutFrame est l'instant de la reapparition — l'instant contributeur de la lecture.
	DebutFrame int `json:"debut_frame"`
	// Cellules est le CHEMIN, dans l'ordre, doublons consecutifs fusionnes. Une cellule
	// peut revenir plus loin (un aller-retour) : seule la repetition IMMEDIATE est reduite.
	Cellules []Cellule `json:"cellules"`
}

// enrichirVies remplit, pour chaque joueur nomme, ses morts et ses routes, et marque sa
// PREMIERE vie.
//
// Elle repasse sur les memes pistes que `Occupation` plutot que de partager sa boucle : les
// deux mesures n'ont pas la meme maille (l'une echantillonne toute la vie, l'autre ne
// regarde que deux instants et une fenetre de 15 s), et les melanger aurait fait une boucle
// que personne ne relit.
func (e echantillonneur) enrichirVies(etat *etatOccupation) {
	pos := e.indexerPositions()
	for _, p := range e.entree.Pistes {
		if p.XUID == "" {
			continue
		}
		points := pointsDansLaFenetre(p)
		if len(points) == 0 {
			continue
		}
		j, _ := etat.joueur(p.XUID)
		j.Routes = append(j.Routes, e.routeDeLaVie(p.XUID, points, pos))
		// TOUTE PISTE NOMMEE PRODUIT UNE MORT, sans condition : c'est le NOMMAGE qui le
		// garantit (cf. la doc de mortDeLaVie), et le filtre `XUID != ""` ci-dessus l'a
		// deja applique. Un second retour « est-ce bien une mort » serait toujours vrai.
		j.Morts = append(j.Morts, e.mortDeLaVie(p, points, pos))
	}
	for _, j := range etat.parJoueur {
		sort.Slice(j.Morts, func(a, b int) bool { return j.Morts[a].Frame < j.Morts[b].Frame })
		sort.Slice(j.Routes, func(a, b int) bool { return j.Routes[a].DebutFrame < j.Routes[b].DebutFrame })
	}
}

// routeDeLaVie suit la vie pendant FenetreRouteMs a pas fixe et rend les cellules
// traversees, doublons CONSECUTIFS fusionnes.
//
// La position tenue et les embarquements s'appliquent comme pour l'occupation : une sortie
// de spawn en vehicule est une route comme une autre.
func (e echantillonneur) routeDeLaVie(xuid string, points []PointPiste, pos positionsParJoueur) Route {
	intervalle := e.entree.IntervalleFrameMs
	debut := points[0].T
	finMs := debut*intervalle + FenetreRouteMs
	if dernier := points[len(points)-1].T * intervalle; dernier < finMs {
		// LA VIE PEUT ETRE PLUS COURTE QUE LA FENETRE : on ne suit pas au-dela de ce que
		// le film montre — tenir la derniere position jusqu'a 15 s peindrait un
		// stationnement posthume.
		finMs = dernier
	}
	r := Route{DebutFrame: debut, Cellules: []Cellule{}}
	for tMs := debut * intervalle; tMs < finMs; tMs += e.pasMs {
		// LA POSITION VIENT DE L'INDEX, donc l'embarquement gagne sur le bipede : une
		// sortie de spawn en vehicule est une route comme une autre.
		p, st := pos.statutA(xuid, tMs/intervalle)
		if st != StatutVivant {
			continue
		}
		c, ok := e.grille.Cellule(p.X, p.Y)
		if !ok {
			continue
		}
		if n := len(r.Cellules); n > 0 && r.Cellules[n-1] == c {
			continue
		}
		r.Cellules = append(r.Cellules, c)
	}
	return r
}

// mortDeLaVie rend la mort qui clot cette vie, et le statut de chaque autre joueur nomme.
//
// # CE QUI GARANTIT QU'UNE VIE NOMMEE EST UNE MORT, ET CE N'EST PAS `EndFrame`
//
// Une version precedente exigeait `EndFrame > StartFrame` « pour ne pas confondre une mort
// avec la fin du film ». CE GARDE ETAIT INERTE (revue P1-2) : le producteur ecrit TOUJOURS
// `StartFrame`/`EndFrame` = premier/dernier point de la piste (`replay/build.go`), si bien
// que la condition est vraie de toute piste de deux frames ou plus.
//
// LE VRAI MECANISME EST LE NOMMAGE, verifie sur pieces : `replay/lives.go:191` est la SEULE
// assignation d'un xuid a une vie, et elle vient de `nameLivesByDeaths` — la vie prend
// l'identite de la victime dont la mort la termine. Les traces heritent ensuite de ce nom
// par RECOUVREMENT DE VIE (`identity.go:nameTracksByLives`), jamais par leur slot. Une piste
// NOMMEE est donc close par une mort, et un survivant de fin de partie reste anonyme : c'est
// le filtre `XUID != ""` de `Occupation` qui fait tout le travail.
func (e echantillonneur) mortDeLaVie(p Piste, points []PointPiste,
	pos positionsParJoueur) MortMesuree {
	// L'INSTANT DE LA MORT EST LA FIN DE LA VIE. `EndFrame` la porte pour tout artefact
	// produit par ce depot (`build.go` l'ecrit toujours) ; un artefact qui ne la porterait
	// pas retombe sur le dernier point, qui est la MEME valeur — ce n'est pas un garde,
	// c'est une resolution d'instant.
	frame := p.EndFrame
	if frame <= p.StartFrame && len(points) > 0 {
		frame = points[len(points)-1].T
	}
	m := MortMesuree{Frame: frame, Voisins: []Voisin{}}
	moi, statut := pos.statutA(p.XUID, frame)
	if statut != StatutVivant || !finie(moi.X, moi.Y) {
		// AUCUN REPLI : une mort dont on ignore la position ne se peint pas au dernier point
		// de bipede connu — ce serait la peindre au point de MONTEE dans le vehicule,
		// c'est-a-dire inventer un stationnement la ou il y a eu un trajet (revue P0-3).
		m.PositionInconnue = true
		return m
	}
	m.X, m.Y = moi.X, moi.Y
	for _, autre := range pos.xuids {
		if autre == p.XUID {
			continue
		}
		q, st := pos.statutA(autre, frame)
		v := Voisin{XUID: autre, Statut: st}
		switch {
		case st == StatutVivant && finie(q.X, q.Y):
			v.DistanceM = math.Hypot(q.X-moi.X, q.Y-moi.Y)
		case st == StatutVivant:
			// On le sait vivant, on ne sait pas ou : c'est une absence de mesure.
			v.Statut = StatutInconnu
		}
		m.Voisins = append(m.Voisins, v)
	}
	return m
}

// finie dit si une position est exploitable (ni NaN ni Inf).
func finie(x, y float64) bool {
	return !math.IsNaN(x) && !math.IsNaN(y) && !math.IsInf(x, 0) && !math.IsInf(y, 0)
}

// positionsParJoueur repond a « ou etait ce joueur a cette frame, et etait-il vivant ».
type positionsParJoueur struct {
	// xuids : les joueurs nommes, TRIES — la sortie doit etre deterministe.
	xuids []string
	// vies : xuid -> ses vies, chacune avec ses points bornes et tries.
	vies map[string][]vieIndexee
	// embs : xuid -> ses embarquements retenus (cf. embarquementsParJoueur).
	embs map[string][]Embarquement
}

// vieIndexee est une vie prete a etre interrogee.
type vieIndexee struct {
	debut, fin int
	points     []PointPiste
}

// indexerPositions construit l'index une seule fois par match.
func (e echantillonneur) indexerPositions() positionsParJoueur {
	idx := positionsParJoueur{
		vies: make(map[string][]vieIndexee),
		embs: embarquementsParJoueur(e.entree.Embarquements),
	}
	for _, p := range e.entree.Pistes {
		if p.XUID == "" {
			continue
		}
		points := pointsDansLaFenetre(p)
		if len(points) == 0 {
			continue
		}
		debut, fin := points[0].T, points[len(points)-1].T
		if p.EndFrame > p.StartFrame {
			// LA FENETRE DECLAREE FAIT FOI POUR LA VITALITE : un joueur est vivant jusqu'a
			// sa mort, meme si le film a cesse de repliquer sa position avant (c'est le cas
			// de tout embarquement).
			debut, fin = p.StartFrame, p.EndFrame
		}
		idx.vies[p.XUID] = append(idx.vies[p.XUID], vieIndexee{debut: debut, fin: fin, points: points})
	}
	for xuid := range idx.vies {
		idx.xuids = append(idx.xuids, xuid)
	}
	sort.Strings(idx.xuids)
	return idx
}

// statutA rend la position d'un joueur a une frame ET son statut (cf. les trois constantes).
//
// L'EMBARQUEMENT GAGNE SUR LE BIPEDE, comme partout : pendant un episode, la derniere
// position de bipede connue ne dit plus ou est le joueur.
//
// LES SEGMENTS NON NOMMES DU MEME SLOT NE SERVENT PAS, et c'est verifie sur pieces : le
// depot a SUPPRIME le nommage par slot parce qu'un slot RECYCLE donnait tout son intervalle
// a son premier porteur nomme — « le remplacant (arrivant, bot) n'existait pas et l'ancien
// vivait a sa place » (`identity.go`) —, et `ownersFromLives` compte les collisions plutot
// que de trancher. Un slot n'est donc pas une identite : s'en servir pour prouver qu'un
// joueur est vivant rattacherait a lui la position de quelqu'un d'autre. Le prix de ce refus
// est ecrit : un survivant de fin de partie sort en `inconnu`, jamais en `vivant`.
func (p positionsParJoueur) statutA(xuid string, frame int) (PointPiste, string) {
	for _, em := range p.embs[xuid] {
		if frame < em.T0 || frame > em.T1 {
			continue
		}
		if pt, ok := dernierPointAvant(em.Points, frame); ok {
			return pt, StatutVivant
		}
		// Embarque, donc VIVANT — mais sans point de vehicule, on ne sait pas ou. On ne
		// retombe pas sur le bipede : il ne replique plus.
		return PointPiste{}, StatutInconnu
	}
	for _, v := range p.vies[xuid] {
		if frame < v.debut || frame > v.fin {
			continue
		}
		if pt, ok := dernierPointAvantOuPremier(v.points, frame); ok {
			return pt, StatutVivant
		}
		return PointPiste{}, StatutInconnu
	}
	// AUCUNE OBSERVATION A CET INSTANT. Mort seulement si une vie NOMMEE de ce joueur a ete
	// close avant : la derniere fin retenue etant la PLUS TARDIVE, « rien observe depuis »
	// est acquis par construction (une vie posterieure couvrirait cet instant, ou finirait
	// plus tard et serait celle-la).
	derniereFin := -1
	for _, v := range p.vies[xuid] {
		if v.fin <= frame && v.fin > derniereFin {
			derniereFin = v.fin
		}
	}
	if derniereFin >= 0 {
		return PointPiste{}, StatutMort
	}
	return PointPiste{}, StatutInconnu
}

// dernierPointAvant rend le dernier point a ou avant `frame`.
func dernierPointAvant(points []PointPiste, frame int) (PointPiste, bool) {
	trouve := false
	var out PointPiste
	for _, pt := range points {
		if pt.T > frame {
			break
		}
		out, trouve = pt, true
	}
	return out, trouve
}

// dernierPointAvantOuPremier tient la derniere position connue ; avant le premier point
// replique d'une vie declaree, c'est ce premier point qui vaut.
func dernierPointAvantOuPremier(points []PointPiste, frame int) (PointPiste, bool) {
	if len(points) == 0 {
		return PointPiste{}, false
	}
	if pt, ok := dernierPointAvant(points, frame); ok {
		return pt, true
	}
	return points[0], true
}
