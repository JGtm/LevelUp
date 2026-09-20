//go:build research

package main

// verdict_geometrie_research_test.go — LA GEOMETRIE DU VERDICT (etape 2, 2026-09-20).
//
// Le verdict contre l'oracle se prononce sur des RECOUVREMENTS DE SURFACE : « la position
// calculee couvre-t-elle 30 % de la zone attendue ? », « quelle zone nommee domine cette
// position ? ». Ces deux questions demandent une aire d'intersection entre un polygone
// CONVEXE (l'enveloppe d'une position, cf. powerpos.Enveloppe) et une zone nommee qui, elle,
// peut etre CONCAVE, en plusieurs morceaux et trouee (provenance « decoupe » du catalogue de
// callouts : `Polygon` + `Parts` + `Holes`, remplissage pair-impair).
//
// POURQUOI UN ECHANTILLONNAGE ET PAS UNE INTERSECTION EXACTE. Une intersection de polygones
// generale (Greiner-Hormann, Vatti) est plusieurs centaines de lignes et un nid a cas
// degeneres (sommets colineaires, contours auto-intersectants — le catalogue en porte). Un
// echantillonnage sur pas fixe rend la MEME reponse a la tolerance pres, se relit en trente
// lignes, et se verifie a la main. Le pas est de 0,2 m, soit 2,5 echantillons par cote de
// cellule de la grille tactique : une zone de 10 m2 porte 250 points, et le seuil de 30 % s'y
// lit a mieux qu'un point de pourcentage. Ce n'est pas la precision qui decide ici, c'est
// l'ordre de grandeur.
//
// Tout est en metres monde, le repere des polygones du catalogue comme celui des enveloppes.

import (
	"math"
	"strings"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// pasEchantillonM est le pas de la grille d'echantillonnage des surfaces.
const pasEchantillonM = 0.2

// surface est un polygone eventuellement troue et en plusieurs morceaux, avec sa boite.
//
// LE REMPLISSAGE EST PAIR-IMPAIR SUR TOUS LES ANNEAUX (contours et trous confondus) — c'est
// la regle du rendu du depot (`calloutsLayer.ts`, provenance « decoupe »), et elle traite les
// trous sans code special : un point dans un trou traverse un nombre PAIR de cotes.
type surface struct {
	anneaux                [][][2]float64
	minX, minY, maxX, maxY float64
}

// nouvelleSurface assemble une surface et mesure sa boite englobante.
func nouvelleSurface(anneaux ...[][2]float64) surface {
	s := surface{minX: math.Inf(1), minY: math.Inf(1), maxX: math.Inf(-1), maxY: math.Inf(-1)}
	for _, a := range anneaux {
		if len(a) < 3 {
			continue
		}
		s.anneaux = append(s.anneaux, a)
		for _, p := range a {
			s.minX, s.maxX = math.Min(s.minX, p[0]), math.Max(s.maxX, p[0])
			s.minY, s.maxY = math.Min(s.minY, p[1]), math.Max(s.maxY, p[1])
		}
	}
	return s
}

// contient dit si un point est dans la surface (pair-impair, lancer de rayon vers +X).
func (s surface) contient(x, y float64) bool {
	if len(s.anneaux) == 0 || x < s.minX || x > s.maxX || y < s.minY || y > s.maxY {
		return false
	}
	croisements := 0
	for _, a := range s.anneaux {
		for i := range a {
			p, q := a[i], a[(i+1)%len(a)]
			if (p[1] > y) == (q[1] > y) {
				continue
			}
			if x < p[0]+(y-p[1])/(q[1]-p[1])*(q[0]-p[0]) {
				croisements++
			}
		}
	}
	return croisements%2 == 1
}

// echantillons rend les points de la grille d'echantillonnage contenus dans la surface.
// La grille est ancree sur l'ORIGINE DU MONDE (et non sur la boite de la surface) : deux
// surfaces echantillonnees separement partagent alors exactement les memes points, ce qui
// rend le recouvrement additif et independant de l'ordre des appels.
func (s surface) echantillons() [][2]float64 {
	if len(s.anneaux) == 0 {
		return nil
	}
	var out [][2]float64
	i0 := int(math.Floor(s.minX / pasEchantillonM))
	i1 := int(math.Ceil(s.maxX / pasEchantillonM))
	j0 := int(math.Floor(s.minY / pasEchantillonM))
	j1 := int(math.Ceil(s.maxY / pasEchantillonM))
	for i := i0; i <= i1; i++ {
		for j := j0; j <= j1; j++ {
			x, y := float64(i)*pasEchantillonM, float64(j)*pasEchantillonM
			if s.contient(x, y) {
				out = append(out, [2]float64{x, y})
			}
		}
	}
	return out
}

// partCouverte rend la fraction des echantillons de `pts` qui tombent dans `s`. Zero quand
// `pts` est vide — une zone sans surface echantillonnable n'est couverte par rien.
func partCouverte(pts [][2]float64, s surface) float64 {
	if len(pts) == 0 {
		return 0
	}
	dedans := 0
	for _, p := range pts {
		if s.contient(p[0], p[1]) {
			dedans++
		}
	}
	return float64(dedans) / float64(len(pts))
}

// aireEchantillonnee rend l'aire mesuree par le nombre d'echantillons, en m2. C'est CETTE
// aire qui est rapportee dans le verdict, et non l'aire analytique : le rapport de
// recouvrement et l'aire doivent venir de la meme mesure, sinon ils ne se composent pas.
func aireEchantillonnee(pts [][2]float64) float64 {
	return float64(len(pts)) * pasEchantillonM * pasEchantillonM
}

// zoneIndexee est l'union des zones nommees qui portent le meme libelle EN (les cartes en
// miroir en portent deux, une par moitie — les separer diviserait le rappel par deux).
type zoneIndexee struct {
	surfaces []surface
	points   [][2]float64
	aireM2   float64
}

// indexeZones rassemble les zones par libelle EN minuscule et echantillonne leur surface.
func indexeZones(zones []replay.CalloutZone) map[string]*zoneIndexee {
	out := map[string]*zoneIndexee{}
	for _, z := range zones {
		nom := strings.ToLower(strings.TrimSpace(z.EN))
		if nom == "" {
			continue
		}
		s := surfaceDeZone(z)
		if len(s.anneaux) == 0 {
			continue
		}
		e := out[nom]
		if e == nil {
			e = &zoneIndexee{}
			out[nom] = e
		}
		e.surfaces = append(e.surfaces, s)
		e.points = append(e.points, s.echantillons()...)
	}
	for _, e := range out {
		e.aireM2 = aireEchantillonnee(e.points)
	}
	return out
}

// contient dit si un point tombe dans l'une des zones du libelle.
func (e *zoneIndexee) contient(x, y float64) bool {
	for _, s := range e.surfaces {
		if s.contient(x, y) {
			return true
		}
	}
	return false
}
