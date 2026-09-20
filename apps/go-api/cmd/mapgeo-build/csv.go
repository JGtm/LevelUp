package main

// csv.go — UNE LIGNE PAR NOEUD : tout ce qui a ete mesure, brut et normalise. C'est la
// matiere du calibrage (distributions par carte) et de la fusion de l'etape 2bis.D.

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"strconv"

	"levelup/go-api/internal/analysis/powerpos/geo"
)

// enteteCSV est l'ordre des colonnes.
var enteteCSV = []string{
	"carte", "col", "lig", "niveau", "composante", "x", "y", "z",
	"h_m", "v", "e", "nb_visibles",
	"d_arme_forte_m", "d_arme_m", "d_objectif_m", "d_couvert_m",
	"r_prox", "m_prox",
	"h_n", "v_n", "e_n", "r_n", "m_n", "score",
}

// EcrisCSV ecrit les noeuds d'une carte.
func EcrisCSV(chemin string, c *Cuite) error {
	f, err := os.Create(chemin)
	if err != nil {
		return fmt.Errorf("creation du CSV (%s) : %w", chemin, err)
	}
	w := csv.NewWriter(f)
	if err := w.Write(enteteCSV); err != nil {
		_ = f.Close()
		return err
	}
	for i, n := range c.Resultat.Noeuds {
		ligne := []string{
			c.Cible.Carte, strconv.Itoa(n.Cellule.Col), strconv.Itoa(n.Cellule.Lig), strconv.Itoa(n.Niveau),
			strconv.Itoa(c.Resultat.Graphe.Composante[i]), f2(n.X), f2(n.Y), f2(n.Z),
			f3(n.Brut.H), f3(n.Brut.V), f3(n.Brut.E), strconv.Itoa(n.NbVisibles),
			f2(n.DArmeForte), f2(n.DArme), f2(n.DObjectif), f2(n.DCouvert),
			f3(n.Brut.R), f3(n.Brut.M),
			f3(n.Norm.H), f3(n.Norm.V), f3(n.Norm.E), f3(n.Norm.R), f3(n.Norm.M), f3(n.Score),
		}
		if err := w.Write(ligne); err != nil {
			_ = f.Close()
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// EcrisRejetsCSV ecrit les candidats de sol ecartes et leur motif : le diagnostic d'un
// sol troue.
func EcrisRejetsCSV(chemin string, c *Cuite) error {
	f, err := os.Create(chemin)
	if err != nil {
		return fmt.Errorf("creation du CSV (%s) : %w", chemin, err)
	}
	w := csv.NewWriter(f)
	if err := w.Write([]string{"x", "y", "z", "motif"}); err != nil {
		_ = f.Close()
		return err
	}
	for _, r := range c.Bilan.Sol.Rejets {
		if err := w.Write([]string{f2(r.X), f2(r.Y), f2(r.Z), r.Motif}); err != nil {
			_ = f.Close()
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// EcrisCouturesCSV ecrit les coutures du graphe (paires de composantes voisines coupees par
// un rayon) avec le voxel qui bloque : le diagnostic d'un sol en morceaux.
func EcrisCouturesCSV(chemin string, c *Cuite, p geo.Parametres) error {
	f, err := os.Create(chemin)
	if err != nil {
		return fmt.Errorf("creation du CSV (%s) : %w", chemin, err)
	}
	w := csv.NewWriter(f)
	entete := []string{"comp_de", "comp_vers", "x1", "y1", "z1", "x2", "y2", "z2",
		"poitrine_libre", "obst_poitrine_x", "obst_poitrine_y", "obst_poitrine_z",
		"saut_libre", "obst_saut_x", "obst_saut_y", "obst_saut_z"}
	if err := w.Write(entete); err != nil {
		_ = f.Close()
		return err
	}
	g := c.Resultat.Graphe
	for _, k := range geo.Coutures(g, c.Resultat.Volume, p) {
		n, m := g.Noeuds[k.De], g.Noeuds[k.Vers]
		ligne := []string{strconv.Itoa(g.Composante[k.De]), strconv.Itoa(g.Composante[k.Vers]),
			f2(n.X), f2(n.Y), f2(n.Z), f2(m.X), f2(m.Y), f2(m.Z),
			strconv.FormatBool(k.PoitrineLibre), f2(k.ObstaclePoitrine[0]), f2(k.ObstaclePoitrine[1]), f2(k.ObstaclePoitrine[2]),
			strconv.FormatBool(k.SautLibre), f2(k.ObstacleSaut[0]), f2(k.ObstacleSaut[1]), f2(k.ObstacleSaut[2])}
		if err := w.Write(ligne); err != nil {
			_ = f.Close()
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func f2(v float64) string { return formate(v, 2) }
func f3(v float64) string { return formate(v, 3) }

// formate ecrit un nombre ; l'infini (aucun chemin) s'ecrit `inf`, jamais un zero.
func formate(v float64, dec int) string {
	if math.IsInf(v, 1) {
		return "inf"
	}
	if math.IsNaN(v) {
		return "nan"
	}
	return strconv.FormatFloat(v, 'f', dec, 64)
}
