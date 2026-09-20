package geo

// scene_test.go — UNE SCENE SYNTHETIQUE POUR LES TEMOINS : un sol, une dalle haute
// atteinte par une rampe, un mur, un plafond bas, une caisse. Tout ce que les cinq
// variables doivent savoir lire, en quelques dizaines de triangles.

import (
	"context"
	"testing"

	"levelup/go-api/internal/analysis/tactical"
)

// quad rend les deux triangles d'un rectangle horizontal a l'altitude z, face vers le haut.
func quad(x0, y0, x1, y1, z float64) []Triangle {
	a, b, c, d := [3]float64{x0, y0, z}, [3]float64{x1, y0, z}, [3]float64{x1, y1, z}, [3]float64{x0, y1, z}
	return []Triangle{{a, b, c}, {a, c, d}}
}

// boite rend les douze triangles d'un pave.
func boite(x0, y0, z0, x1, y1, z1 float64) []Triangle {
	p := func(x, y, z float64) [3]float64 { return [3]float64{x, y, z} }
	var out []Triangle
	face := func(a, b, c, d [3]float64) { out = append(out, Triangle{a, b, c}, Triangle{a, c, d}) }
	face(p(x0, y0, z0), p(x1, y0, z0), p(x1, y1, z0), p(x0, y1, z0)) // dessous
	face(p(x0, y0, z1), p(x1, y0, z1), p(x1, y1, z1), p(x0, y1, z1)) // dessus
	face(p(x0, y0, z0), p(x1, y0, z0), p(x1, y0, z1), p(x0, y0, z1)) // sud
	face(p(x0, y1, z0), p(x1, y1, z0), p(x1, y1, z1), p(x0, y1, z1)) // nord
	face(p(x0, y0, z0), p(x0, y1, z0), p(x0, y1, z1), p(x0, y0, z1)) // ouest
	face(p(x1, y0, z0), p(x1, y1, z0), p(x1, y1, z1), p(x1, y0, z1)) // est
	return out
}

// rampe rend un plan incline le long de X, de z0 en x0 a z1 en x1.
func rampe(x0, y0, x1, y1, z0, z1 float64) []Triangle {
	a, b, c, d := [3]float64{x0, y0, z0}, [3]float64{x1, y0, z1}, [3]float64{x1, y1, z1}, [3]float64{x0, y1, z0}
	return []Triangle{{a, b, c}, {a, c, d}}
}

// sceneTemoin : sol 20 x 20 m a z = 0 ; dalle 6 x 6 m a z = 2 dans le coin nord-est,
// rejointe par une rampe de 4 m ; mur de 3 m de haut au milieu (x = 10, y de 2 a 10) ;
// plafond bas (z = 1,2) sur 3 x 3 m au sud-ouest ; caisse de 1 m de haut (on y saute) ;
// pilier de 2,5 m (on n'y monte pas).
func sceneTemoin() []Triangle {
	var tris []Triangle
	tris = append(tris, quad(0, 0, 20, 20, 0)...)
	tris = append(tris, boite(14, 14, 0, 20, 20, 2)...)   // dalle haute (pleine)
	tris = append(tris, rampe(10, 15, 14, 19, 0, 2)...)   // rampe vers la dalle
	tris = append(tris, boite(9.8, 2, 0, 10.2, 10, 3)...) // mur
	tris = append(tris, quad(1, 1, 4, 4, 1.2)...)         // plafond bas
	tris = append(tris, boite(5, 15, 0, 6, 16, 1)...)     // caisse
	tris = append(tris, boite(2, 17, 0, 3, 18, 2.5)...)   // pilier
	return tris
}

// entreesTemoin assemble les entrees de la scene : cadre sur [0, 20]^2, ancres au sol et
// sur la dalle, une arme forte au pied de la dalle, un objectif au sud.
func entreesTemoin(t *testing.T) Entrees {
	t.Helper()
	cadre, err := NouveauCadre(tactical.GrilleParDefaut(), 0, 0, 20, 20)
	if err != nil {
		t.Fatal(err)
	}
	return Entrees{
		Cadre: cadre, Triangles: sceneTemoin(), ZMin: -1, ZMax: 6,
		Ancres: [][3]float64{{3, 10, 0}, {17, 17, 2}, {15, 5, 0}},
		Ressources: []Ressource{
			{X: 12, Y: 12, Z: 0, Nature: NatureArmeForte},
			{X: 15, Y: 3, Z: 0, Nature: NatureObjectif},
		},
	}
}

// mesureTemoin execute la mesure complete sur la scene.
func mesureTemoin(t *testing.T) (*Resultat, Bilan) {
	t.Helper()
	res, b, err := Mesure(context.Background(), entreesTemoin(t), ParametresParDefaut())
	if err != nil {
		t.Fatal(err)
	}
	return res, b
}

// noeudEn rend le noeud le plus proche d'un point monde, ou echoue.
func noeudEn(t *testing.T, g *Graphe, x, y, z float64) int {
	t.Helper()
	n, ok := g.NoeudLePlusProche(x, y, z, ParametresParDefaut())
	if !ok {
		t.Fatalf("aucun noeud en (%v, %v, %v)", x, y, z)
	}
	return n
}
