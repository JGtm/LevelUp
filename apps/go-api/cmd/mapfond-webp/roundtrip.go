package main

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"time"

	"github.com/HugoSmits86/nativewebp"
	"golang.org/x/image/webp"
)

// optionsWebP fixe la compression au maximum : l'encodage reste sans perte dans tous les cas
// (nativewebp.Encode ne produit que du VP8L, D5) — CompressionLevel ne joue que sur l'effort de
// recherche du meilleur agencement, pas sur la fidelite des pixels.
var optionsWebP = &nativewebp.Options{CompressionLevel: nativewebp.BestCompression}

// resultatAllerRetour porte les mesures d'un aller-retour PNG -> WebP -> RGBA pour un fond.
type resultatAllerRetour struct {
	OctetsWebP  int
	DureeEncode time.Duration
	DureeDecode time.Duration
	// Identique dit si le redecodage rend EXACTEMENT les memes pixels que l'origine (Pix ET
	// Rect). Toute difference est traitee comme une erreur fatale par l'appelant (Etape 0 du
	// plan) : ce champ n'est jamais silencieusement ignore.
	Identique bool
}

// allerRetourWebP encode img en WebP sans perte (D5), le redecode (D6) et compare le resultat a
// img canonicalise (versRGBA), octet a octet. Rend aussi le blob encode : le mode -convertir
// l'ecrit sur disque sans re-encoder une seconde fois.
func allerRetourWebP(img image.Image) (resultatAllerRetour, []byte, error) {
	var res resultatAllerRetour
	origine := versRGBA(img)

	var buf bytes.Buffer
	debutEncode := time.Now()
	if err := nativewebp.Encode(&buf, img, optionsWebP); err != nil {
		return res, nil, fmt.Errorf("encodage webp: %w", err)
	}
	res.DureeEncode = time.Since(debutEncode)
	encode := buf.Bytes()
	res.OctetsWebP = len(encode)

	debutDecode := time.Now()
	decode, err := webp.Decode(bytes.NewReader(encode))
	if err != nil {
		return res, encode, fmt.Errorf("decodage webp: %w", err)
	}
	res.DureeDecode = time.Since(debutDecode)

	apres := versRGBA(decode)
	res.Identique = origine.Rect == apres.Rect && bytes.Equal(origine.Pix, apres.Pix)
	return res, encode, nil
}

// versRGBA ramene toute image.Image dans le meme espace canonique (image.RGBA, alpha
// premultiplie). PNG et WebP ne rendent pas necessairement le meme type Go concret (NRGBA le
// plus souvent pour l'un comme pour l'autre selon la presence d'alpha) : comparer des `Pix` de
// types differents comparerait des encodages memoire, pas des pixels. La MEME fonction est
// appliquee a l'origine et au resultat de l'aller-retour, donc la comparaison reste juste meme
// si les deux decodeurs ne rendent pas le meme type concret.
func versRGBA(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}
	b := img.Bounds()
	out := image.NewRGBA(b)
	draw.Draw(out, b, img, b.Min, draw.Src)
	return out
}
