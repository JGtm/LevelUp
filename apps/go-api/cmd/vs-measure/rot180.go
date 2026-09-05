package main

// rot180.go — sous-commande `rot180` du driver jetable vs-measure (verif echelle, 2026-09-02).
//
// Pivote un PNG deja rendu de 180 degres DANS LE PLAN DE L'IMAGE (une rotation, jamais un
// miroir : (x,y) -> (W-1-x, H-1-y), la meme operation que `-rot180` de `plateau.go`). Sert a
// mettre "nez en haut" les sprites dont le sens de +X local (avant/arriere) a ete determine
// APRES le rendu (vue de dessus + vue de profil), sans re-rendre le modele. Ne modifie ni ne
// lit aucun fichier de jeu : entree et sortie sont des PNG deja produits par
// `cmd/vehicle-sprite render`.
//
// Usage : vsmeasure.exe rot180 -files=a.png,b.png -out=DIR
import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

func rot180Main(args []string) {
	fs := flag.NewFlagSet("rot180", flag.ExitOnError)
	files := fs.String("files", "", "PNG a pivoter (chemins, virgule)")
	out := fs.String("out", ".", "dossier de sortie (meme nom de fichier)")
	_ = fs.Parse(args)

	if err := os.MkdirAll(*out, 0o755); err != nil {
		must(err)
	}
	for _, s := range strings.Split(*files, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if err := rot180Fichier(s, filepath.Join(*out, filepath.Base(s))); err != nil {
			fmt.Printf("%s : %v\n", s, err)
			continue
		}
		fmt.Printf("ROT180 %s -> %s\n", s, filepath.Join(*out, filepath.Base(s)))
	}
}

// rot180Fichier lit un PNG, le pivote de 180 degres (rotation, pas miroir), l'ecrit ailleurs.
func rot180Fichier(entree, sortie string) error {
	f, err := os.Open(entree)
	if err != nil {
		return err
	}
	src, err := png.Decode(f)
	f.Close()
	if err != nil {
		return err
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	rgba := imageAsNRGBA(src)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := rgba.NRGBAAt(b.Min.X+x, b.Min.Y+y)
			dst.SetNRGBA(w-1-x, h-1-y, c)
		}
	}
	out, err := os.Create(sortie)
	if err != nil {
		return err
	}
	defer out.Close()
	return png.Encode(out, dst)
}

// imageAsNRGBA convertit une image.Image quelconque en *image.NRGBA (les sprites de
// `vehicle-sprite` sont deja NRGBA, mais on reste robuste a un autre codec de source).
func imageAsNRGBA(img image.Image) *image.NRGBA {
	if n, ok := img.(*image.NRGBA); ok {
		return n
	}
	b := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			out.Set(x-b.Min.X, y-b.Min.Y, img.At(x, y))
		}
	}
	return out
}
