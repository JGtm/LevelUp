package himap

// cadre_utile.go — LE CADRE PUBLIÉ N'EST PLUS CELUI DE LA GRILLE.
//
// LE DÉFAUT, MESURÉ LE 2026-08-26. Le cadre de cuisson est la boîte des ancres d'objectifs
// élargie d'une CONSTANTE de 50 m (`MargeCadre`). La coquille de mort efface ensuite tout ce
// qui tombe hors de la frontière déclarée par la carte — mais personne ne recalculait le cadre
// après elle. Résultat sur les 19 fonds natifs publiés : la matière n'occupe que **53,5 % de la
// largeur en médiane**, et jusqu'à 28,8 % (Recharge), 33,7 % (Aquarius), 40,6 % (Streets). Le
// reste est du vide transparent. À l'écran, la carte se dessine petite au milieu de rien.
//
// CE QUE FAIT CE FICHIER : il rend la boîte de la matière RÉELLEMENT dessinée, élargie d'une
// marge en mètres, et le calage qui va avec. Le monde ne bouge pas d'un centimètre — c'est le
// même repère, borné plus près.
//
// POURQUOI UNE MARGE, ET POURQUOI EN MÈTRES. Rogner au pixel près collerait le bord de l'image
// au bord du terrain : une trajectoire qui frôle la limite se dessinerait sur l'arête. La marge
// est en mètres pour valoir la même chose à toutes les échelles — c'est une distance de jeu,
// pas une épaisseur d'image.
//
// CE QU'IL NE FAUT PAS EN ATTENDRE : il ne corrige pas une carte dont le DÉCOR remplit le
// cadre (les cartes Forge, à 88,3 % de largeur occupée en médiane). Là, la matière est bien
// présente partout — c'est son contenu qui est en cause, pas son cadre.

import "image"

// MargeCadreUtile : marge conservée autour de la matière, en mètres.
//
// 6 m ≈ deux fois la portée d'un pas de côté : assez pour qu'une trajectoire au bord du terrain
// ne touche pas l'arête de l'image, assez peu pour que le gain de cadrage reste l'essentiel.
const MargeCadreUtile = 6.0

// SeuilAlphaMatiere : en deçà, un pixel est tenu pour vide. Les fonds sont produits en RGBA sur
// fond strictement transparent ; le seuil laisse passer un liseré d'antialiasing sans le
// compter comme matière.
const SeuilAlphaMatiere = 8

// CadreUtile rend la zone de l'image qui porte de la matière, élargie de `MargeCadreUtile`, et
// bornée à l'image. Le second retour est faux quand l'image est ENTIÈREMENT vide : il n'y a
// alors pas de cadre utile, et l'appelant doit publier l'image telle quelle plutôt que de
// rogner sur du néant.
func CadreUtile(img *image.RGBA, metresParPixel float64) (image.Rectangle, bool) {
	b := img.Bounds()
	minX, minY, maxX, maxY := b.Max.X, b.Max.Y, b.Min.X-1, b.Min.Y-1
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if img.Pix[img.PixOffset(x, y)+3] < SeuilAlphaMatiere {
				continue
			}
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
	if maxX < minX || maxY < minY {
		return b, false
	}
	marge := 0
	if metresParPixel > 0 {
		marge = int(MargeCadreUtile/metresParPixel + 0.5)
	}
	r := image.Rect(minX-marge, minY-marge, maxX+1+marge, maxY+1+marge).Intersect(b)
	return r, true
}

// CalageRogne translate le calage d'un rendu vers un sous-rectangle de son image.
//
// La convention publiée reste la même (`ConventionCalage`) : seule l'ORIGINE bouge, du nombre
// de pixels retirés en haut et à gauche. `metresParPixel` est inchangé — rogner ne change pas
// l'échelle, et confondre les deux ferait dériver toutes les positions.
func CalageRogne(origineX, origineY, metresParPixel float64, r image.Rectangle) (float64, float64) {
	return origineX + float64(r.Min.X)*metresParPixel,
		origineY - float64(r.Min.Y)*metresParPixel
}
