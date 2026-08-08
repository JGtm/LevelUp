package main

// bc7_mode7.go — LE MODE 7, celui qui fait les artefacts.
//
// CE QUI A RÉDUIT LE PROBLÈME. Le décodeur exact couvre les modes 4, 5 et 6 ; les modes
// 0 à 3 et 7 tombaient en repli (aplat). Mais seul le mode 7 dégrade RÉELLEMENT ce qu'on
// garde : les modes 0 à 3 n'ont pas de canal alpha, donc leur alpha vaut 255 partout — ce
// que le repli produit déjà, exactement. Sur l'icône #21 de l'atlas contour, les 28 blocs
// en repli sont TOUS des modes 7 (mesure `tmp_wicons marks`). Il n'y a donc qu'un mode à
// traiter, pas cinq.
//
// LE VERROU, ET COMMENT IL EST CONTOURNÉ SANS RIEN INVENTER. Un bloc multi-sous-ensembles
// a besoin de deux tables du format : la partition (quel pixel appartient à quel
// sous-ensemble) et la position des ancres (les pixels dont l'index est écrit sur un bit de
// moins). Recopier ces tables de mémoire ferait courir un risque d'erreur SILENCIEUSE — une
// table fausse rend une image plausible mais fausse, et c'est le pire cas.
//
// Tout le reste du bloc se lit EXACTEMENT : largeurs de champs, points extrêmes, bits P,
// index. Ce qui manque se retrouve donc par AJUSTEMENT SUR DES DONNÉES RÉELLES DU FICHIER :
// le niveau de mip inférieur, décodé lui aussi et agrandi, sert de témoin. On essaie les
// 15 positions d'ancre possibles ; pour chacune, chaque pixel prend le sous-ensemble dont
// la valeur colle le mieux au témoin ; on garde la position qui minimise l'écart total.
// Rien n'est deviné : la mesure tranche.
//
// HONNÊTETÉ DU RÉSULTAT : ces blocs sont RECONSTRUITS, pas décodés. Ils sont comptés
// séparément et le compte remonte dans index.json (`bc7_rebuilt_pct`).

import "image"

// mode7IndexBits : largeur d un index de mode 7. La première ancre est toujours le
// pixel 0 (règle du format, pas une hypothèse) ; la seconde est cherchée.
const mode7IndexBits = 2

// decodeMode7 lit un bloc de mode 7 et écrit ses pixels. `guide` est le niveau de mip
// inférieur agrandi, ou nil. Retourne false si aucun témoin n'est disponible : l'appelant
// retombe alors sur le repli en aplat.
func decodeMode7(block []byte, img *image.NRGBA, x0, y0 int, guide *image.NRGBA) bool {
	if guide == nil {
		return false
	}
	r := &bitReader{b: block}
	r.read(8)         // mode
	r.read(6)         // partition — non exploitée, la table n'est pas reproduite ici
	var raw [4][4]int // [canal R,G,B,A][point extrême 0..3]
	for ch := 0; ch < 4; ch++ {
		for e := 0; e < 4; e++ {
			raw[ch][e] = r.read(5)
		}
	}
	var p [4]int
	for e := 0; e < 4; e++ {
		p[e] = r.read(1)
	}
	// 5 bits + 1 bit P = 6 bits significatifs, portés sur 8.
	var ep [4][4]int // [point extrême][canal]
	for e := 0; e < 4; e++ {
		for ch := 0; ch < 4; ch++ {
			ep[e][ch] = expand((raw[ch][e]<<1)|p[e], 6)
		}
	}

	best, bestErr := -1, 1<<62
	var bestPix [16][4]uint8
	for anchor := 1; anchor < 16; anchor++ {
		pix, errSum, ok := mode7Try(block, ep, anchor, guide, img.Bounds(), x0, y0)
		if ok && errSum < bestErr {
			best, bestErr, bestPix = anchor, errSum, pix
		}
	}
	if best < 0 {
		return false
	}
	b := img.Bounds()
	for i := 0; i < 16; i++ {
		x, y := x0+i%4, y0+i/4
		if x >= b.Dx() || y >= b.Dy() {
			continue
		}
		o := img.PixOffset(x, y)
		copy(img.Pix[o:o+4], bestPix[i][:])
	}
	return true
}

// mode7Try lit le flux d'index pour une position d'ancre donnée, choisit pour chaque pixel
// le sous-ensemble le plus proche du témoin, et rend l'écart total.
func mode7Try(block []byte, ep [4][4]int, anchor int, guide *image.NRGBA,
	bounds image.Rectangle, x0, y0 int) (pix [16][4]uint8, errSum int, ok bool) {
	r := &bitReader{b: block}
	r.read(8 + 6 + 4*4*5 + 4) // mode + partition + points extrêmes + bits P
	var idx [16]int
	for i := 0; i < 16; i++ {
		bits := mode7IndexBits
		if i == 0 || i == anchor {
			bits--
		}
		idx[i] = r.read(bits)
	}
	for i := 0; i < 16; i++ {
		w := weightOf(mode7IndexBits, idx[i])
		x, y := x0+i%4, y0+i/4
		var gr, gg, gb, ga int
		hasGuide := x < bounds.Dx() && y < bounds.Dy()
		if hasGuide {
			o := guide.PixOffset(x, y)
			gr, gg, gb, ga = int(guide.Pix[o]), int(guide.Pix[o+1]), int(guide.Pix[o+2]), int(guide.Pix[o+3])
		}
		bestD := 1 << 62
		var bestV [4]uint8
		for s := 0; s < 2; s++ {
			var v [4]uint8
			for ch := 0; ch < 4; ch++ {
				v[ch] = interp(ep[2*s][ch], ep[2*s+1][ch], w)
			}
			if !hasGuide {
				bestV = v
				break
			}
			// L'alpha pèse le plus : c'est le canal conservé dans le glyphe final.
			d := 3*sq(int(v[3])-ga) + sq(int(v[0])-gr) + sq(int(v[1])-gg) + sq(int(v[2])-gb)
			if d < bestD {
				bestD, bestV = d, v
			}
		}
		if hasGuide {
			errSum += bestD
		}
		pix[i] = bestV
	}
	return pix, errSum, true
}

func sq(v int) int { return v * v }

// upscale2 double une image par duplication de pixels : le témoin n'a pas besoin d'être
// lisse, il doit seulement dire vers quelle valeur penche chaque zone.
func upscale2(src *image.NRGBA, w, h int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	if sw == 0 || sh == 0 {
		return dst
	}
	for y := 0; y < h; y++ {
		sy := y / 2
		if sy >= sh {
			sy = sh - 1
		}
		for x := 0; x < w; x++ {
			sx := x / 2
			if sx >= sw {
				sx = sw - 1
			}
			so, do := src.PixOffset(sx, sy), dst.PixOffset(x, y)
			copy(dst.Pix[do:do+4], src.Pix[so:so+4])
		}
	}
	return dst
}
