package main

// sprite.go — sous-commande `sprite` du driver jetable vs-measure (verif echelle, 2026-09-02).
//
// Mesure l'emprise OPAQUE (pixels alpha > 0) d'un ou plusieurs PNG deja rendus, pour verifier
// leur echelle (mm/px) sans toucher au code de production (`internal/himap`, `cmd/vehicle-sprite`).
// Usage :
//
//	vsmeasure.exe sprite -files=a.png,b.png
//	vsmeasure.exe sprite -dir=.../sprites_v4
//
// Imprime, par fichier : taille du canevas, boite englobante opaque (px), et sa largeur/hauteur.
// Ne modifie ni ne lit aucun fichier de jeu.
import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func spriteMain(args []string) {
	fs := flag.NewFlagSet("sprite", flag.ExitOnError)
	files := fs.String("files", "", "PNG a mesurer (chemins, virgule)")
	dir := fs.String("dir", "", "dossier dont mesurer tous les *.png")
	_ = fs.Parse(args)

	var chemins []string
	if *dir != "" {
		entries, err := os.ReadDir(*dir)
		must(err)
		for _, e := range entries {
			if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".png") {
				chemins = append(chemins, filepath.Join(*dir, e.Name()))
			}
		}
		sort.Strings(chemins)
	}
	for _, s := range strings.Split(*files, ",") {
		if s = strings.TrimSpace(s); s != "" {
			chemins = append(chemins, s)
		}
	}
	if len(chemins) == 0 {
		fmt.Println("sprite: aucun fichier (-files ou -dir requis)")
		return
	}
	for _, c := range chemins {
		mesureSprite(c)
	}
}

// mesureSprite charge un PNG et imprime son emprise opaque (alpha > 0), en pixels, avec le
// canevas complet pour reference.
func mesureSprite(chemin string) {
	f, err := os.Open(chemin)
	if err != nil {
		fmt.Printf("%s : ouverture KO: %v\n", chemin, err)
		return
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		fmt.Printf("%s : decodage KO: %v\n", chemin, err)
		return
	}
	minX, minY, maxX, maxY, n := bboxOpaque(img)
	nom := filepath.Base(chemin)
	if n == 0 {
		fmt.Printf("SPRITE %-24s canevas=%dx%d  AUCUN pixel opaque\n", nom, img.Bounds().Dx(), img.Bounds().Dy())
		return
	}
	w, h := maxX-minX+1, maxY-minY+1
	fmt.Printf("SPRITE %-24s canevas=%dx%d  opaque=%dx%d  bbox=[%d..%d]x[%d..%d]  px_opaques=%d\n",
		nom, img.Bounds().Dx(), img.Bounds().Dy(), w, h, minX, maxX, minY, maxY, n)
}

// bboxOpaque rend la boite englobante des pixels d'alpha > 0 (coordonnees image, origine 0,0
// en haut-gauche) et le nombre de pixels opaques trouves.
func bboxOpaque(img image.Image) (minX, minY, maxX, maxY, n int) {
	b := img.Bounds()
	minX, minY = b.Dx(), b.Dy()
	maxX, maxY = -1, -1
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a == 0 {
				continue
			}
			n++
			ix, iy := x-b.Min.X, y-b.Min.Y
			if ix < minX {
				minX = ix
			}
			if ix > maxX {
				maxX = ix
			}
			if iy < minY {
				minY = iy
			}
			if iy > maxY {
				maxY = iy
			}
		}
	}
	return minX, minY, maxX, maxY, n
}
