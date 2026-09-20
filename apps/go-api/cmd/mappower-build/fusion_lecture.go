package main

// fusion_lecture.go — CE QUE LA FUSION LIT : le CSV par noeud de `cmd/mapgeo-build` (une
// ligne par noeud du sol derive) et le CSV par cellule de la passe empirique (csv_lecture.go).
// Aucune base : les deux voies ont deja ecrit tout ce qu'il faut sur disque.
//
// LE CSV GEOMETRIQUE EST LU PAR NOM DE COLONNE, pas par rang : son en-tete appartient a un
// autre programme (`mapgeo-build/csv.go`) et le recopier ici ferait une deuxieme copie a
// tenir en phase. Seules quatre colonnes servent a la fusion — `col`, `lig`, `z`, `score` —
// plus `x` et `y` pour le CONTROLE D'ALIGNEMENT : chaque noeud doit etre au centre de la
// cellule qu'il declare, sur la grille tactique. Un ecart trahirait un ancrage different
// (une grille decalee d'une demi-cellule), et la fusion s'arrete plutot que de sommer deux
// grilles qui ne se superposent pas.

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/powerpos/fusion"
	"levelup/go-api/internal/analysis/tactical"
)

// toleranceAlignementM : ecart tolere entre le noeud et le centre de sa cellule (le CSV
// ecrit deux decimales).
const toleranceAlignementM = 0.011

// colonnesGeoRequises : les colonnes du CSV geometrique que la fusion lit.
var colonnesGeoRequises = []string{"col", "lig", "x", "y", "z", "score"}

// litCSVNoeudsGeo relit le CSV par noeud et verifie l'alignement de chaque noeud sur la
// grille. Rend les noeuds et le nombre de noeuds a score non fini (ecartes, comptes).
func litCSVNoeudsGeo(chemin string, g tactical.Grille) ([]fusion.NoeudGeo, int, error) {
	f, err := os.Open(chemin)
	if err != nil {
		return nil, 0, fmt.Errorf("CSV geometrique illisible (%s) : %w", chemin, err)
	}
	defer func() { _ = f.Close() }()
	lignes, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, 0, fmt.Errorf("CSV geometrique invalide (%s) : %w", chemin, err)
	}
	if len(lignes) == 0 {
		return nil, 0, fmt.Errorf("CSV geometrique vide (%s)", chemin)
	}
	rang, err := rangsDesColonnes(lignes[0], colonnesGeoRequises)
	if err != nil {
		return nil, 0, fmt.Errorf("CSV geometrique (%s) : %w", chemin, err)
	}
	out := make([]fusion.NoeudGeo, 0, len(lignes)-1)
	nonFinis := 0
	for i, l := range lignes[1:] {
		n := fusion.NoeudGeo{Col: entier(l[rang["col"]]), Lig: entier(l[rang["lig"]]),
			Z: flottant(l[rang["z"]]), Score: flottantOuNaN(l[rang["score"]])}
		if math.IsNaN(n.Score) || math.IsInf(n.Score, 0) {
			nonFinis++
			continue
		}
		if err := verifieAlignement(g, n, flottant(l[rang["x"]]), flottant(l[rang["y"]])); err != nil {
			return nil, 0, fmt.Errorf("CSV geometrique (%s), ligne %d : %w", chemin, i+2, err)
		}
		out = append(out, n)
	}
	return out, nonFinis, nil
}

// rangsDesColonnes rend le rang de chaque colonne requise dans l'en-tete.
func rangsDesColonnes(entete, requises []string) (map[string]int, error) {
	rang := map[string]int{}
	for i, nom := range entete {
		rang[strings.TrimSpace(nom)] = i
	}
	for _, nom := range requises {
		if _, ok := rang[nom]; !ok {
			return nil, fmt.Errorf("colonne %q absente de l'en-tete", nom)
		}
	}
	return rang, nil
}

// verifieAlignement compare la position du noeud au centre de sa cellule sur la grille.
func verifieAlignement(g tactical.Grille, n fusion.NoeudGeo, x, y float64) error {
	cx, cy := g.Centre(tactical.Cellule{Col: n.Col, Lig: n.Lig})
	if math.Abs(cx-x) > toleranceAlignementM || math.Abs(cy-y) > toleranceAlignementM {
		return fmt.Errorf("noeud (%.2f, %.2f) hors du centre (%.2f, %.2f) de la cellule (%d, %d) :"+
			" grilles non alignees", x, y, cx, cy, n.Col, n.Lig)
	}
	return nil
}

// flottantOuNaN lit un flottant ; `inf` et `nan` (ecrits par mapgeo-build) rendent NaN.
func flottantOuNaN(s string) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return math.NaN()
	}
	return v
}
