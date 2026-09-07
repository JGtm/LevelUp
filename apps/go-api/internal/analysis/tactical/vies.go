package tactical

// vies.go — CE QU'UNE VIE PRODUIT EN PLUS DE SON OCCUPATION : sa ROUTE de sortie de spawn,
// et la CHRONOLOGIE DE POSITIONS du joueur.
//
// # LE SIDECAR NE JUGE RIEN, ET IL NE PORTE PLUS DE CHRONOLOGIE
//
// Une version precedente faisait dire au film qui etait mort a l'instant d'une mort, en
// s'appuyant sur « une vie nommee est close par une mort ». CETTE PREMISSE EST FAUSSE :
// `replay/owners.go:166` (`nameClosedLives`) nomme aussi une vie par FERMETURE DE SLOT,
// c'est-a-dire par le pont des tirs — un joueur qui SURVIT en ayant tire recevait donc une
// vie nommee, dont ce fichier fabriquait une mort qui n'a jamais eu lieu.
//
// Ce fichier a alors produit une CHRONOLOGIE DE POSITIONS, pour que la lecture tranche
// elle-meme. Elle n'a plus de consommateur : les faits d'isolement se produisent AU SYNC,
// par le collecteur de kills, dans les tables `match_lives` et `match_death_context`
// (decision utilisateur du 2026-09-07, lot 7C) — « les donnees d'un match en base sont
// completes au sync ; seul le rejeu peut attendre la cuisson ».
//
// # LA ROUTE EST LA SORTIE DE SPAWN
//
// Les 15 premieres secondes de CHAQUE vie (decision produit du plan), en cellules
// ORDONNEES, doublons consecutifs fusionnes : ce qui compte est le CHEMIN, pas le temps
// passe dans chaque case — celui-la est deja mesure par l'occupation. L'instant
// contributeur est le debut de la vie.

import "sort"

// FenetreRouteMs est la duree, en millisecondes, pendant laquelle on suit une vie apres sa
// reapparition : 15 s (plan tactique, decision produit du 2026-09-05).
//
// POURQUOI 15 s. C'est la duree qui separe une SORTIE de spawn d'un deplacement ordinaire :
// au-dela, le joueur a rejoint le jeu et sa trajectoire ne dit plus par ou l'on sort. La
// borne est un choix produit, pas une mesure — et elle est ici, nommee, pour qu'on sache
// quoi changer si elle se revele fausse.
const FenetreRouteMs = 15_000

// Route est la sortie de spawn d'une vie : les cellules traversees pendant FenetreRouteMs.
type Route struct {
	// DebutFrame est l'instant de la reapparition — l'instant contributeur de la lecture.
	DebutFrame int `json:"debut_frame"`
	// Cellules est le CHEMIN, dans l'ordre, doublons consecutifs fusionnes. Une cellule
	// peut revenir plus loin (un aller-retour) : seule la repetition IMMEDIATE est reduite.
	Cellules []Cellule `json:"cellules"`
}

// enrichirVies remplit, pour chaque joueur nomme, ses routes, et marque sa PREMIERE vie.
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
	}
	for _, j := range etat.parJoueur {
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

// positionsParJoueur repond a « ou etait ce joueur a cette frame ».
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
			// LA FENETRE DECLAREE FAIT FOI : un joueur est observable jusqu'a la fin de sa
			// vie, meme si le film a cesse de repliquer sa position avant (embarquement).
			debut, fin = p.StartFrame, p.EndFrame
		}
		idx.vies[p.XUID] = append(idx.vies[p.XUID], vieIndexee{debut: debut, fin: fin, points: points})
	}
	for xuid := range idx.vies {
		idx.xuids = append(idx.xuids, xuid)
	}
	for xuid := range idx.embs {
		if _, deja := idx.vies[xuid]; !deja {
			// Un occupant NOMME dont aucune vie de bipede n'a ete nommee existe quand meme.
			idx.xuids = append(idx.xuids, xuid)
		}
	}
	sort.Strings(idx.xuids)
	return idx
}

// positionA rend la position d'un joueur a une frame, et si elle est connue.
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
		// Embarque, mais sans point de vehicule : on ne sait pas ou. On ne retombe pas sur
		// le bipede — il ne replique plus.
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
