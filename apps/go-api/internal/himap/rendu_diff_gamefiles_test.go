package himap

import (
	"image"
	"image/color"
	"math"
	"sort"
	"testing"
)

// L'INSTRUMENT QUI MANQUAIT — ou la reconstruction differe-t-elle de la reference, PIXEL A
// PIXEL ?
//
// Jusqu'ici le rendu se jugeait sur un pourcentage de couverture et sur un coup d'oeil. Ni
// l'un ni l'autre ne dit OU il manque de la matiere : la couverture globale peut monter
// pendant qu'un pont disparait, et l'oeil ne compte pas. Trois chiffres et une carte des
// ecarts remplacent les deux.
//
// Convention de la troisieme vignette :
//
//	gris   les deux ont de la matiere
//	ROUGE  la reference en a, nous PAS   <- ce qui manque
//	BLEU   nous en avons, la reference PAS
const seuilMatiereRef = 60 // luminance au-dessus de laquelle un pixel de la reference porte
// de la matiere. Le fond de la reference est sombre et uniforme ; le seuil est CONTROLE par
// l'histogramme publie au meme endroit, pas suppose.

// matiereReference dit, pixel par pixel, ou la carte validee porte de la matiere, et rend
// l'histogramme de luminance qui CONTROLE le seuil au lieu de le supposer.
func matiereReference(ref image.Image, bo image.Rectangle, larg, haut int) ([]bool, [8]int) {
	var hist [8]int
	plein := make([]bool, larg*haut)
	for py := 0; py < haut; py++ {
		for px := 0; px < larg; px++ {
			r, g, b, _ := ref.At(bo.Min.X+px, bo.Min.Y+py).RGBA()
			l := (299*int(r>>8) + 587*int(g>>8) + 114*int(b>>8)) / 1000
			hist[l*8/256]++
			plein[py*larg+px] = l > seuilMatiereRef
		}
	}
	return plein, hist
}

func comparePixels(t *testing.T, ref image.Image, bo image.Rectangle, rendu *Rendu,
	out *image.RGBA, larg, haut int) {
	t.Helper()

	refPlein, hist := matiereReference(ref, bo, larg, haut)
	t.Logf("reference — histogramme de luminance par huitieme : %v", hist)

	nRef, nNous, manquants, exces := 0, 0, 0, 0
	for py := 0; py < haut; py++ {
		j := haut - 1 - py
		for px := 0; px < larg; px++ {
			r := refPlein[py*larg+px]
			_, n := rendu.Eclairement(px, j)
			if r {
				nRef++
			}
			if n {
				nNous++
			}
			switch {
			case r && !n:
				manquants++
			case n && !r:
				exces++
			}
		}
	}
	t.Logf("MATIERE — reference %d px (%.1f %%) · nous %d px (%.1f %%)",
		nRef, 100*float64(nRef)/float64(larg*haut), nNous, 100*float64(nNous)/float64(larg*haut))
	t.Logf("MANQUANTS (la reference a, nous pas) : %d px = %.1f %% de la reference",
		manquants, 100*float64(manquants)/float64(max(nRef, 1)))
	t.Logf("EXCES (nous avons, la reference pas)  : %d px = %.1f %% de la reference",
		exces, 100*float64(exces)/float64(max(nRef, 1)))
	// ACCORD : intersection sur union. Le seul chiffre qui ne se laisse pas gagner d'un cote
	// aux depens de l'autre — tout dessiner rend 0 manquant et un accord catastrophique, ne
	// rien dessiner l'inverse. C'est lui qui arbitre un seuil, pas « manquants » seul.
	inter := nRef - manquants
	if u := nRef + exces; u > 0 {
		t.Logf("ACCORD (intersection / union) : %.1f %%", 100*float64(inter)/float64(u))
	}
	altitudeDesEcarts(t, refPlein, rendu, larg, haut)
	dessineCarteDesEcarts(refPlein, rendu, out, larg, haut)
}

// altitudeDesEcarts confronte l'altitude que NOUS dessinons sur les pixels justes et sur les
// pixels en trop.
//
// C'est la question posee par l'utilisateur — « regarde AUSSI les basses altitudes ». Si la
// matiere en trop se concentre dans une bande d'altitude, le tri se fait la, et il se LIT au
// lieu de se deviner. Si elle se repartit comme la matiere juste, l'altitude n'est pas le
// discriminant et il faut chercher ailleurs.
func altitudeDesEcarts(t *testing.T, refPlein []bool, rendu *Rendu, larg, haut int) {
	t.Helper()
	const pas = 5.0 // metres par classe
	justes, trop := map[int]int{}, map[int]int{}
	for py := 0; py < haut; py++ {
		j := haut - 1 - py
		for px := 0; px < larg; px++ {
			z, ok := rendu.Altitude(px, j)
			if !ok {
				continue
			}
			k := int(math.Floor(z / pas))
			if refPlein[py*larg+px] {
				justes[k]++
			} else {
				trop[k]++
			}
		}
	}
	cles := map[int]bool{}
	for k := range justes {
		cles[k] = true
	}
	for k := range trop {
		cles[k] = true
	}
	ordre := make([]int, 0, len(cles))
	for k := range cles {
		ordre = append(ordre, k)
	}
	sort.Ints(ordre)
	t.Logf("altitude de CE QUE NOUS DESSINONS, par tranche de %.0f m :", pas)
	for _, k := range ordre {
		t.Logf("   [%+6.0f ; %+6.0f[  justes %8d   EN TROP %8d",
			float64(k)*pas, float64(k+1)*pas, justes[k], trop[k])
	}
}

// dessineCarteDesEcarts colle la troisieme vignette a droite des deux autres. Une plage
// ROUGE compacte designe une STRUCTURE absente ; un semis disperse, une difference de finesse.
func dessineCarteDesEcarts(refPlein []bool, rendu *Rendu, out *image.RGBA, larg, haut int) {
	dec := 2 * (larg + 8)
	for py := 0; py < haut; py++ {
		j := haut - 1 - py
		for px := 0; px < larg; px++ {
			r := refPlein[py*larg+px]
			_, n := rendu.Eclairement(px, j)
			var c color.RGBA
			switch {
			case r && n:
				c = color.RGBA{110, 110, 110, 255}
			case r:
				c = color.RGBA{225, 45, 45, 255}
			case n:
				c = color.RGBA{45, 110, 225, 255}
			default:
				c = color.RGBA{0, 0, 0, 255}
			}
			if x := dec + px; x < out.Bounds().Dx() {
				out.Set(x, py, c)
			}
		}
	}
}
