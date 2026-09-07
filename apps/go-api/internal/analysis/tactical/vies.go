package tactical

// vies.go — CE QU'UNE VIE PRODUIT EN PLUS DE SON OCCUPATION : sa ROUTE de sortie de spawn,
// et la CHRONOLOGIE DE POSITIONS du joueur.
//
// # LE SIDECAR NE JUGE RIEN (decision utilisateur du 2026-09-07)
//
// Une version precedente faisait dire au film qui etait mort a l'instant d'une mort, en
// s'appuyant sur « une vie nommee est close par une mort ». CETTE PREMISSE EST FAUSSE :
// `replay/owners.go:166` (`nameClosedLives`) nomme aussi une vie par FERMETURE DE SLOT,
// c'est-a-dire par le pont des tirs — un joueur qui SURVIT en ayant tire recevait donc une
// vie nommee, dont ce fichier fabriquait une mort qui n'a jamais eu lieu.
//
// Le film ne porte pas la liste des morts ; la BASE la porte (journal des morts, departs).
// Ce fichier ne mesure donc plus que ce que le film sait vraiment dire : OU ETAIT CHACUN,
// ET QUAND. Le verdict d'isolement se prend entierement a la lecture
// (cf. analysis/coordination/isolation.go). Ce que « vivant » veut dire quand le film se
// tait est une DECISION PRODUIT EN COURS (2026-09-07) : la lecture tient en attendant sur
// « vivant = une position connue a cet instant », et le point d'attente est commente sur
// pieces dans service/tactical_service_lectures.go (`PROVISOIRE 2026-09-07`).
//
// # LA CHRONOLOGIE
//
// Par joueur NOMME, sa position tenue au pas de PasChronologieMs sur chaque fenetre ou une
// position est CONNUE — vies nommees et episodes d'embarquement rattaches. Entre deux
// fenetres : RIEN. Une absence n'est pas une mort : c'est une absence, et c'est la lecture
// qui, journal en main, saura laquelle des deux elle est.
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

// PasChronologieMs est le pas de la chronologie de positions : 500 ms.
//
// POURQUOI DEUX FOIS PLUS GROS QUE L'OCCUPATION (250 ms). La chronologie ne mesure pas une
// duree, elle repond a « ou etait-il a cet instant » pour une comparaison de DISTANCE a
// 18 ou 24 m. A 500 ms, un joueur au sprint (~5 m/s en Halo) parcourt 2,5 m entre deux
// echantillons : l'incertitude reste tres inferieure au rayon, et le volume est divise par
// deux. Un pas plus fin n'ajouterait pas de verdict juste, il ajouterait des octets.
const PasChronologieMs = 500

// Route est la sortie de spawn d'une vie : les cellules traversees pendant FenetreRouteMs.
type Route struct {
	// DebutFrame est l'instant de la reapparition — l'instant contributeur de la lecture.
	DebutFrame int `json:"debut_frame"`
	// Cellules est le CHEMIN, dans l'ordre, doublons consecutifs fusionnes. Une cellule
	// peut revenir plus loin (un aller-retour) : seule la repetition IMMEDIATE est reduite.
	Cellules []Cellule `json:"cellules"`
}

// SegmentChrono est une fenetre CONTINUE ou la position du joueur est connue.
//
// POURQUOI DES SEGMENTS ET NON UNE SUITE UNIQUE : entre deux vies, on ne sait rien. Une
// suite unique devrait combler ces trous par une valeur, et toute valeur serait une
// invention — un joueur mort n'est pas « a sa derniere position », il n'est nulle part.
type SegmentChrono struct {
	// DebutFrame est l'instant du PREMIER echantillon du segment.
	DebutFrame int `json:"debut_frame"`
	// XY porte les positions APLATIES (x0, y0, x1, y1, ...), un couple par pas de
	// PasChronologieMs a partir de DebutFrame.
	XY []float64 `json:"xy"`
}

// enrichirVies remplit, pour chaque joueur nomme, ses routes et sa chronologie, et marque
// sa PREMIERE vie.
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
	for xuid, j := range etat.parJoueur {
		sort.Slice(j.Routes, func(a, b int) bool { return j.Routes[a].DebutFrame < j.Routes[b].DebutFrame })
		j.Chronologie = e.chronologieDe(xuid, pos)
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

// chronologieDe echantillonne les fenetres OBSERVABLES d'un joueur au pas de la
// chronologie, et rend un segment par fenetre continue.
//
// LES FENETRES SONT L'UNION DES VIES NOMMEES ET DES EMBARQUEMENTS, fusionnee quand elles se
// touchent : un joueur qui monte en vehicule a la fin de sa vie ne doit pas produire deux
// segments accoles, qui se liraient comme une interruption.
func (e echantillonneur) chronologieDe(xuid string, pos positionsParJoueur) []SegmentChrono {
	fenetres := pos.fenetresObservables(xuid)
	if len(fenetres) == 0 {
		return []SegmentChrono{}
	}
	intervalle := e.entree.IntervalleFrameMs
	pasFrames := PasChronologieMs / intervalle
	if pasFrames < 1 {
		pasFrames = 1
	}
	out := make([]SegmentChrono, 0, len(fenetres))
	for _, f := range fenetres {
		seg := SegmentChrono{DebutFrame: f.debut, XY: []float64{}}
		for frame := f.debut; frame <= f.fin; frame += pasFrames {
			p, ok := pos.positionA(xuid, frame)
			if !ok || !finie(p.X, p.Y) {
				// Trou DANS une fenetre observable (embarquement sans point de vehicule) :
				// on n'invente rien. Le segment garde son pas — l'echantillon manquant
				// serait une position fausse, l'absence est une absence.
				seg.XY = append(seg.XY, math.NaN(), math.NaN())
				continue
			}
			seg.XY = append(seg.XY, arrondi2(p.X), arrondi2(p.Y))
		}
		if len(seg.XY) > 0 {
			out = append(out, seg)
		}
	}
	return out
}

// arrondi2 arrondit a deux decimales — MEME convention que les coordonnees de l'artefact
// (`replay.round2`) et que le reste du sidecar.
func arrondi2(v float64) float64 {
	return math.Round(v*100) / 100
}

// finie dit si une position est exploitable (ni NaN ni Inf).
func finie(x, y float64) bool {
	return !math.IsNaN(x) && !math.IsNaN(y) && !math.IsInf(x, 0) && !math.IsInf(y, 0)
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

// fenetre est un intervalle de frames.
type fenetre struct{ debut, fin int }

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

// fenetresObservables rend les intervalles ou une position de ce joueur est connue, tries
// et FUSIONNES quand ils se touchent ou se chevauchent.
func (p positionsParJoueur) fenetresObservables(xuid string) []fenetre {
	brutes := make([]fenetre, 0, len(p.vies[xuid])+len(p.embs[xuid]))
	for _, v := range p.vies[xuid] {
		brutes = append(brutes, fenetre{debut: v.debut, fin: v.fin})
	}
	for _, em := range p.embs[xuid] {
		brutes = append(brutes, fenetre{debut: em.T0, fin: em.T1})
	}
	if len(brutes) == 0 {
		return nil
	}
	sort.Slice(brutes, func(a, b int) bool { return brutes[a].debut < brutes[b].debut })
	out := []fenetre{brutes[0]}
	for _, f := range brutes[1:] {
		dernier := &out[len(out)-1]
		if f.debut <= dernier.fin+1 {
			if f.fin > dernier.fin {
				dernier.fin = f.fin
			}
			continue
		}
		out = append(out, f)
	}
	return out
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
