package main

// planche.go — OU LA COUPE MORD, en image.
//
// UN POURCENTAGE NE DIT PAS OU. « 23 % de l'image changerait » se lit de deux facons opposees :
// un toit continu qui recouvre l'arene, ou des rochers epars sur tout le pourtour. La planche
// tranche d'un coup d'oeil : elle reprend l'habillage de production (`himap.FondPNG`) et TEINTE
// les pixels dont la surface affichee est au-dessus du plafond propose.
//
// CE QU'ELLE N'EST PAS : un « apres ». Montrer ce que la carte afficherait A LA PLACE demande
// de rejouer la cuisson sous un autre plafond — c'est le changement du lot C1, pas une mesure.
// La planche dit OU la coupe mord ; elle ne promet rien de ce qui apparaitrait dessous.

import (
	"fmt"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"levelup/go-api/internal/himap"
)

// teinteCoupe : le rouge des pixels que la coupe emporterait. Opaque — une transparence les
// rendrait indiscernables du relief clair de l'habillage `jeu`.
var teinteCoupe = color.RGBA{R: 204, G: 51, B: 51, A: 255}

// ecritPlanche dessine la carte cuite et teinte ce que le plafond emporterait.
func ecritPlanche(dir, module string, rendu *himap.Rendu, seuil float64) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("repertoire des planches : %w", err)
	}
	img := himap.FondPNG(rendu, rendu.NX, rendu.NY, nil, himap.StyleFondParDefaut)
	for j := 0; j < rendu.NY; j++ {
		for i := 0; i < rendu.NX; i++ {
			z, ok := rendu.Altitude(i, j)
			if !ok || z <= seuil {
				continue
			}
			// `FondPNG` retourne l'image : sa ligne `py` lit la ligne `NY-1-py` du rendu
			// (convention de calage, `himap.ConventionCalage`). On teinte donc en `NY-1-j`.
			img.Set(i, rendu.NY-1-j, teinteCoupe)
		}
	}
	chemin := filepath.Join(dir, module+"_coupe.png")
	f, err := os.Create(chemin) //nolint:gosec // planche de mesure, chemin donne en ligne de commande
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
