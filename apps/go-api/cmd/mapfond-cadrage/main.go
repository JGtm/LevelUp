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
	// bruit : part des pixels de matiere dont la LUMINANCE saute fortement par rapport a leurs
	// voisins de droite et du bas. C est la mesure du « gribouillis » : une carte faite de
	// dalles planes rend de grands aplats et un bruit faible ; une carte faite de coques
	// organiques qui se chevauchent rend un enchevetrement de traits et un bruit eleve.
	//
	// Pourquoi cette mesure existe : l utilisateur demande, le 2026-08-27, ce qui distingue
	// les cartes Forge propres des cartes en bouillie. Tant que le defaut restait une
	// impression, on ne pouvait ni le classer ni verifier qu on l avait corrige.
	bruit float64
	// alignement : part des CONTOURS qui suivent un axe de la grille, a 15 degres pres.
	//
	// C est la mesure qui separe ce que le bruit ne separait pas. Une carte Forge batie de
	// pieces prismatiques posees a la grille rend des contours horizontaux et verticaux ; une
	// carte batie de rochers rend des contours qui pointent dans toutes les directions. Le
	// bruit, lui, compte les ruptures sans regarder leur SENS : il monte aussi bien sur une
	// carte tres detaillee mais lisible (Cliffhanger, 43 pour cent) que sur une bouillie.
	alignement float64
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
	// Tri par BRUIT decroissant : la question du jour est « quelles cartes sont en bouillie »,
	// et la reponse doit se lire sur la premiere ligne.
	sort.Slice(mesures, func(i, j int) bool { return mesures[i].alignement < mesures[j].alignement })

	fmt.Println(strings.Join([]string{
		"cle", "largeurPx", "hauteurPx", "matiereLPx", "matiereHPx",
		"occupLargeurPct", "occupHauteurPct", "occupAirePct", "bruitPct", "alignementPct",
	}, "\t"))
	for _, m := range mesures {
		fmt.Printf("%s\t%d\t%d\t%d\t%d\t%.1f\t%.1f\t%.1f\t%.1f\t%.1f\n",
			m.cle, m.largeur, m.hauteur, m.matiereL, m.matiereH,
			100*m.occupL, 100*m.occupH, 100*m.occupA, 100*m.bruit, 100*m.alignement)
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
	m.bruit = mesureBruit(img, b)
	m.alignement = mesureAlignement(img, b)
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

// seuilSautLuminance : ecart de luminance, sur 255, au-dela duquel deux pixels voisins comptent
// comme une rupture. 24 separe une arete tracee (l'arete divise la teinte par 3) d'un simple
// degrade d'eclairement.
const seuilSautLuminance = 24

// mesureBruit rend la part des pixels de matiere qui rompent avec leur voisin de droite ou du
// bas. Deux voisins suffisent : le bruit d'un enchevetrement de traits est isotrope, et le
// compter dans quatre directions doublerait le cout sans rien separer de plus.
func mesureBruit(img image.Image, b image.Rectangle) float64 {
	matiere, ruptures := 0, 0
	for y := b.Min.Y; y < b.Max.Y-1; y++ {
		for x := b.Min.X; x < b.Max.X-1; x++ {
			if !opaque(img, x, y) {
				continue
			}
			matiere++
			l := luminance(img, x, y)
			if (opaque(img, x+1, y) && abs(l-luminance(img, x+1, y)) > seuilSautLuminance) ||
				(opaque(img, x, y+1) && abs(l-luminance(img, x, y+1)) > seuilSautLuminance) {
				ruptures++
			}
		}
	}
	if matiere == 0 {
		return 0
	}
	return float64(ruptures) / float64(matiere)
}

// luminance rend la clarte d'un pixel sur 0..255, ponderee comme l'oeil la percoit.
func luminance(img image.Image, x, y int) int {
	r, g, bl, _ := img.At(x, y).RGBA()
	return (299*int(r>>8) + 587*int(g>>8) + 114*int(bl>>8)) / 1000
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// seuilContour : norme minimale du gradient pour qu'un pixel compte comme contour. En deca on
// ne mesure que le bruit d'eclairement, dont l'orientation ne veut rien dire.
const seuilContour = 18

// mesureAlignement rend la part des contours dont la direction suit un axe de la grille a
// 15 degres pres. Gradient de Sobel simplifie sur la luminance, deux voisins par axe.
func mesureAlignement(img image.Image, b image.Rectangle) float64 {
	contours, alignes := 0, 0
	for y := b.Min.Y + 1; y < b.Max.Y-1; y++ {
		for x := b.Min.X + 1; x < b.Max.X-1; x++ {
			if !opaque(img, x, y) || !opaque(img, x-1, y) || !opaque(img, x+1, y) ||
				!opaque(img, x, y-1) || !opaque(img, x, y+1) {
				continue
			}
			gx := luminance(img, x+1, y) - luminance(img, x-1, y)
			gy := luminance(img, x, y+1) - luminance(img, x, y-1)
			if gx*gx+gy*gy < seuilContour*seuilContour {
				continue
			}
			contours++
			// Un contour est aligne si son gradient est presque purement horizontal ou
			// presque purement vertical : tan(15 deg) vaut 0,268, soit environ 27 %.
			ax, ay := abs(gx), abs(gy)
			if ay*100 < ax*27 || ax*100 < ay*27 {
				alignes++
			}
		}
	}
	if contours == 0 {
		return 0
	}
	return float64(alignes) / float64(contours)
}
