package main

// fusion_png.go — LA PLANCHE DE FUSION : le score fusionne par cellule sur le fond publie,
// les contours des positions retenues, les zones nommees en filigrane. Meme peintre que les
// autres planches (png.go) : une cellule est peinte a son emprise, les couleurs sont des
// `color.NRGBA`, l'echelle est etalee entre le p10 et le p99 de la carte (choix
// d'affichage, la selection n'en depend pas).

import (
	"image/color"

	"levelup/go-api/internal/analysis/powerpos"
	"levelup/go-api/internal/domain/title"
)

// EcrisPNGFusion ecrit la planche de fusion d'une carte.
func EcrisPNGFusion(res *title.PathResolver, titleSlug string, c *carteFusionnee, chemin string) error {
	fond, err := chargeFond(res, titleSlug, c.Identite.Carte)
	if err != nil {
		return err
	}
	zones := chargeZones(res, titleSlug, c.Identite.Carte, c.Identite.MapIDDominant)
	cellules := make([]powerpos.Cellule, 0, len(c.Cellules))
	scores := make(map[[2]int]float64, len(c.Cellules))
	tous := make([]float64, 0, len(c.Cellules))
	for _, f := range c.Cellules {
		cellules = append(cellules, f.Cellule)
		scores[[2]int{f.Col, f.Lig}] = f.Score
		tous = append(tous, f.Score)
	}
	bas, haut := powerpos.Quantile(tous, 0.10), powerpos.Quantile(tous, 0.99)
	toile := peins(fond, zones, cellules, func(cel powerpos.Cellule) (color.NRGBA, bool) {
		v := scores[[2]int{cel.Col, cel.Lig}]
		if haut > bas {
			v = (v - bas) / (haut - bas)
		}
		return sequentiel(v), true
	})
	trait := color.NRGBA{R: 60, G: 255, B: 140, A: 230}
	for _, p := range c.Positions {
		tracePolygone(toile, fond.cal, p.Polygone, trait)
	}
	return encode(chemin, toile)
}
