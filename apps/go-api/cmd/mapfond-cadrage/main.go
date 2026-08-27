// cmd/mapfond-cadrage — MESURE LE CADRAGE DES FONDS DE CARTE PUBLIES.
//
// CE QU'ELLE REPOND. Le cadre d'un fond n'est pas derive de l'aire de jeu : c'est la boite des
// ancres d'objectifs elargie d'une CONSTANTE (`himap.MargeCadre` = 2 x `PorteeAncre` = 50 m,
// `internal/himap/cuisson.go`). La coquille de mort, elle, efface ensuite tout ce qui tombe
// hors de la frontiere — sans que le cadre soit recalcule. Resultat possible : une image dont
// la matiere n'occupe qu'une fraction de la largeur, le reste etant transparent.
//
// Cet outil chiffre cette fraction, fond par fond, pour que le defaut cesse d'etre une
// impression et devienne une colonne du registre de revue.
//
// CE QU'ELLE NE FAIT PAS. Elle mesure la matiere DESSINEE, pas la zone JOUEE. Une carte dont
// le decor remplit le cadre sera a 100 % ici tout en cadrant mal a l'oeil. La mesure contre
// les positions jouees est un second instrument, distinct.
//
// Lecture seule, hors ligne : n'ouvre que les PNG deja publies.
//
// Usage :
//
//	go run ./cmd/mapfond-cadrage [--title slug] [--dir <chemin>]
//
// Sortie : TSV sur stdout (cle, largeur, hauteur, matiere l/h, occupation l/h/aire, %).
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/domain/title"
)

// seuilAlpha : en deca, un pixel est tenu pour vide. Les fonds sont produits en RGBA avec un
// fond strictement transparent (alpha 0) ; le seuil laisse passer un eventuel liseré
// d'antialiasing sans le compter comme matiere.
const seuilAlpha = 8

type mesure struct {
	cle                    string
	largeur, hauteur       int
	matiereL, matiereH     int
	pixels                 int
	occupL, occupH, occupA float64
}

func main() {
	titleSlug := flag.String("title", "halo_infinite", "slug du titre")
	dir := flag.String("dir", "", "dossier des fonds (par defaut : celui du PathResolver)")
	flag.Parse()

	racine, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cwd:", err)
		os.Exit(1)
	}
	dossier := *dir
	if dossier == "" {
		// Le PathResolver rend le chemin d'un fond ; son dossier est celui qu'on balaie.
		dossier = filepath.Dir(title.NewPathResolver(racine).MapBackgroundPath(*titleSlug, "x"))
	}

	fichiers, err := filepath.Glob(filepath.Join(dossier, "*.png"))
	if err != nil || len(fichiers) == 0 {
		fmt.Fprintf(os.Stderr, "aucun fond dans %s (err=%v)\n", dossier, err)
		os.Exit(1)
	}

	mesures := make([]mesure, 0, len(fichiers))
	for _, f := range fichiers {
		m, err := mesureFond(f)
		if err != nil {
			// Jamais avalee : un fond illisible est une donnee manquante, pas un zero.
			fmt.Fprintf(os.Stderr, "ILLISIBLE %s: %v\n", filepath.Base(f), err)
			continue
		}
		mesures = append(mesures, m)
	}
	sort.Slice(mesures, func(i, j int) bool { return mesures[i].occupL < mesures[j].occupL })

	fmt.Println(strings.Join([]string{
		"cle", "largeurPx", "hauteurPx", "matiereLPx", "matiereHPx",
		"occupLargeurPct", "occupHauteurPct", "occupAirePct",
	}, "\t"))
	for _, m := range mesures {
		fmt.Printf("%s\t%d\t%d\t%d\t%d\t%.1f\t%.1f\t%.1f\n",
			m.cle, m.largeur, m.hauteur, m.matiereL, m.matiereH,
			100*m.occupL, 100*m.occupH, 100*m.occupA)
	}
}

// mesureFond rend la boite de la matiere non transparente d'un PNG, rapportee a son cadre.
func mesureFond(chemin string) (mesure, error) {
	f, err := os.Open(chemin)
	if err != nil {
		return mesure{}, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return mesure{}, err
	}
	b := img.Bounds()
	m := mesure{
		cle:     strings.TrimSuffix(filepath.Base(chemin), ".png"),
		largeur: b.Dx(),
		hauteur: b.Dy(),
	}
	minX, minY, maxX, maxY := b.Max.X, b.Max.Y, b.Min.X-1, b.Min.Y-1
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if !opaque(img, x, y) {
				continue
			}
			m.pixels++
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	if m.pixels == 0 {
		return m, nil
	}
	m.matiereL, m.matiereH = maxX-minX+1, maxY-minY+1
	m.occupL = float64(m.matiereL) / float64(m.largeur)
	m.occupH = float64(m.matiereH) / float64(m.hauteur)
	m.occupA = float64(m.pixels) / float64(m.largeur*m.hauteur)
	return m, nil
}

// opaque dit si le pixel porte de la matiere. `At` suffit : les fonds sont des RGBA, et la
// conversion en RGBA64 preserve l'alpha.
func opaque(img image.Image, x, y int) bool {
	_, _, _, a := img.At(x, y).RGBA()
	return a>>8 >= seuilAlpha
}
