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

// VoisinVivant est un autre joueur NOMME, vivant a l'instant d'une mort, et sa distance a
// elle. Ni equipe ni camp : le film ne les porte pas.
type VoisinVivant struct {
	XUID      string  `json:"xuid"`
	DistanceM float64 `json:"distance_m"`
}

// MortMesuree est la fin d'une vie nommee : ou, quand, et qui etait encore debout autour.
type MortMesuree struct {
	// Frame est l'instant de la mort sur l'axe du rejeu (la fin de la vie).
	Frame int `json:"frame"`
	// X, Y : la position de la victime, en metres monde.
	X float64 `json:"x"`
	Y float64 `json:"y"`
	// Voisins : TOUS les autres joueurs nommes vivants a cet instant, tries par xuid.
	// Vide = le film ne montre personne d'autre debout — ce qui n'est PAS « il etait seul
	// avec ses adversaires » : c'est le service qui, connaissant les equipes, tranchera.
	Voisins []VoisinVivant `json:"voisins"`
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
		if m, ok := e.mortDeLaVie(p, points, pos); ok {
			j.Morts = append(j.Morts, m)
		}
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
		p, ok := pos.positionA(xuid, tMs/intervalle)
		if !ok {
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

// mortDeLaVie rend la mort qui clot cette vie, et les voisins vivants a cet instant.
//
// SANS FENETRE DECLAREE, PAS DE MORT : `EndFrame` est optionnel dans l'artefact, et un
// artefact ancien ne le porte pas. Deduire la mort du dernier point confondrait une mort
// avec la fin du film — donc avec un survivant.
//
//nolint:unparam // `points` sert de repli quand l'index ne rend rien (vie hors fenetre).
func (e echantillonneur) mortDeLaVie(p Piste, points []PointPiste, pos positionsParJoueur) (MortMesuree, bool) {
	if p.EndFrame <= p.StartFrame {
		return MortMesuree{}, false
	}
	frame := p.EndFrame
	dernier, ok := pos.positionA(p.XUID, frame)
	if !ok {
		dernier = points[len(points)-1]
	}
	m := MortMesuree{Frame: frame, X: dernier.X, Y: dernier.Y, Voisins: []VoisinVivant{}}
	if !finie(dernier.X, dernier.Y) {
		// Position non finie : la mort existe, mais elle n'est nulle part. On ne mesure
		// aucune distance depuis un point qui n'en est pas un.
		return m, true
	}
	for _, autre := range pos.xuids {
		if autre == p.XUID {
			continue
		}
		q, vivant := pos.positionA(autre, frame)
		if !vivant || !finie(q.X, q.Y) {
			continue
		}
		m.Voisins = append(m.Voisins, VoisinVivant{
			XUID:      autre,
			DistanceM: math.Hypot(q.X-dernier.X, q.Y-dernier.Y),
		})
	}
	return m, true
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

// positionA rend la position d'un joueur a une frame, et s'il etait vivant.
//
// L'EMBARQUEMENT GAGNE SUR LE BIPEDE, comme partout : pendant un episode, la derniere
// position de bipede connue ne dit plus ou est le joueur.
func (p positionsParJoueur) positionA(xuid string, frame int) (PointPiste, bool) {
	for _, em := range p.embs[xuid] {
		if frame < em.T0 || frame > em.T1 {
			continue
		}
		if pt, ok := dernierPointAvant(em.Points, frame); ok {
			return pt, true
		}
		// Embarque mais sans point de vehicule : VIVANT, position inconnue. On ne retombe
		// pas sur le bipede — il ne replique plus.
		return PointPiste{}, false
	}
	for _, v := range p.vies[xuid] {
		if frame < v.debut || frame > v.fin {
			continue
		}
		return dernierPointAvantOuPremier(v.points, frame)
	}
	return PointPiste{}, false
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
