// Package himap — rendu_couleur.go : ce qui rend la carte LISIBLE, une fois la geometrie juste.
//
// LE CONSTAT (utilisateur, 2026-08-09, apres validation de la geometrie) : « ca manque un peu
// de nettete ». L'eclairage de `rendu.go` est un Lambert avec ambiante hemispherique
// `0,25 + 0,75 x d` : il ne descend jamais sous 25 % et sature vite, donc tout se tasse vers
// le blanc et les aretes disparaissent.
//
// Trois leviers, tous SANS reglage par carte :
//
//  1. PALIERS — quantifier l'eclairement en quelques niveaux. Un degrade continu lit mal a
//     plat ; des aplats separent les faces.
//  2. ARETES — une rupture d'altitude entre deux pixels voisins est un bord. C'est ce que le
//     degrade ne montre pas : deux surfaces de meme inclinaison a des hauteurs differentes ont
//     la meme teinte et se confondent.
//  3. NUANCIER D'ALTITUDE — teinter par la hauteur, sur des bornes ROBUSTES.
//
// Le nuancier se borne aux CENTILES, pas au min/max : lecon deja payee sur ce chantier le
// 2026-08-08 — une seule cellule a -131 m ecrasait toute la carte dans deux nuances de blanc.
package himap

import (
	"image/color"
	"math"
	"sort"
)

// PaliersEclairement : nombre d'aplats du rendu plat. Constante, jamais un reglage par carte.
const PaliersEclairement = 5

// SeuilAreteMetres : denivele entre deux pixels voisins au-dela duquel on trace un bord.
//
// 0,5 m est PHYSIQUE, pas esthetique : a la resolution de la carte (~9 cm par pixel), une
// marche d'un demi-metre sur 9 cm n'est pas une pente, c'est un rebord. En dessous, un Spartan
// franchit sans sauter.
const SeuilAreteMetres = 0.5

// SeuilArete remplace le seuil par defaut pour CE rendu. Zero = SeuilAreteMetres.
//
// POURQUOI UN REGLAGE. Sur une carte faite de pieces ORGANIQUES qui se chevauchent — les
// remakes Forge en rochers — deux pixels voisins different de quelques centimetres partout :
// le predicat d arete devient vrai presque partout et l image se couvre d un GRIBOUILLIS de
// traits qui masque les formes. C est le defaut signale par l utilisateur sur Isolation le
// 2026-08-27, et la mesure qui l a designe : le meme rendu en habillage altitude — le seul
// qui ne trace AUCUNE arete — montre les formes intactes sous le gribouillis.
//
// Le seuil ne change pas la geometrie, seulement ce qu on souligne.

// EclairementPlat rend l'eclairement quantifie en `PaliersEclairement` aplats.
func (r *Rendu) EclairementPlat(i, j int) (float64, bool) {
	e, ok := r.Eclairement(i, j)
	if !ok {
		return 0, false
	}
	n := float64(PaliersEclairement - 1)
	return math.Round(e*n) / n, true
}

// Arete dit si le pixel borde une rupture d'altitude — le bord d'une plateforme, d'un mur ou
// d'une passerelle.
//
// Le test porte sur les quatre voisins et sur la DIFFERENCE, pas sur la normale : deux dalles
// horizontales a deux hauteurs ont la meme normale et doivent quand meme se separer. Un pixel
// dont un voisin est vide est aussi un bord — c'est la silhouette.
// seuilArete rend le denivele a partir duquel ce rendu trace un bord.
func (r *Rendu) seuilArete() float64 {
	if r.SeuilArete > 0 {
		return r.SeuilArete
	}
	return SeuilAreteMetres
}

func (r *Rendu) Arete(i, j int) bool {
	z, ok := r.Altitude(i, j)
	if !ok {
		return false
	}
	for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		zv, okv := r.Altitude(i+d[0], j+d[1])
		if !okv || math.Abs(z-zv) > r.seuilArete() {
			return true
		}
	}
	return false
}

// BornesAltitudeRobustes rend les centiles 2 et 98 des altitudes dessinees.
//
// Centiles et non min/max : une poignee de cellules aberrantes suffirait a ecraser tout le
// nuancier — c'est arrive sur ce chantier, une cellule a -131 m rendait la carte blanche.
func (r *Rendu) BornesAltitudeRobustes() (bas, haut float64, ok bool) {
	zs := make([]float64, 0, r.NX*r.NY/4)
	for j := 0; j < r.NY; j++ {
		for i := 0; i < r.NX; i++ {
			if z, okz := r.Altitude(i, j); okz {
				zs = append(zs, z)
			}
		}
	}
	if len(zs) < 2 {
		return 0, 0, false
	}
	sort.Float64s(zs)
	bas = zs[len(zs)*2/100]
	haut = zs[len(zs)*98/100]
	if !(haut > bas) {
		return 0, 0, false
	}
	return bas, haut, true
}

// nuancierAltitude : rampe SEQUENTIELLE, du bas froid et sombre au haut chaud et clair.
//
// Sequentielle et non arc-en-ciel : l'altitude est une grandeur ordonnee, une rampe qui change
// de teinte sans ordre percu la rend illisible et trompe sur les paliers.
var nuancierAltitude = [...][3]float64{
	{0.16, 0.20, 0.28}, // ardoise
	{0.27, 0.35, 0.44},
	{0.45, 0.52, 0.56},
	{0.68, 0.68, 0.64},
	{0.86, 0.82, 0.72},
	{0.97, 0.95, 0.90}, // craie
}

// couleurEau / couleurBergeEau : l'eau du fond de carte (volumes d'eau du tag sddt).
//
// L'eau est PLATE : pas d'ombrage, pas de nuancier d'altitude — une surface d'eau n'a ni
// relief ni etages. Un bleu moyen, desature, hors de la rampe ardoise->craie du terrain pour
// que la riviere se lise sans ecraser le sol qui l'entoure ; la berge est le meme bleu
// assombri, meme role que l'arete du terrain (une rupture, pas une couleur de plus).
var (
	couleurEau      = color.RGBA{62, 112, 152, 255}
	couleurBergeEau = color.RGBA{38, 70, 96, 255}
)

// TeinteEau rend la couleur d'une cellule d'eau, berge ou pleine eau.
func TeinteEau(berge bool) color.RGBA {
	if berge {
		return couleurBergeEau
	}
	return couleurEau
}

// PorteeNiveauDeJeu : distance verticale, en metres, au-dela de laquelle une surface n'est
// plus tenue pour un etage joue et recule dans le fond.
//
// 10 m est de l'ordre de deux etages de carte. Ce n'est pas un reglage d'image : le niveau de
// reference vient des ancres d'objectifs (`AncrageDecalageSol`, `reference.go`), donc de la
// carte elle-meme, et cette portee dit seulement a partir de quand on quitte l'aire de jeu.
const PorteeNiveauDeJeu = 10.0

// TeinteNiveauDeJeu rend la couleur d'une surface selon son ECART SIGNE au niveau de jeu.
//
// POURQUOI ELLE EXISTE. `TeinteAltitude` normalise entre les centiles 2 et 98 de TOUTE la
// matiere dessinee : sur une carte encaissee comme Cliffhanger, ce sont les rochers hauts qui
// prennent le haut de la rampe, donc le blanc, donc l'oeil — pendant que l'arene, plus basse,
// recule dans le sombre. **La hierarchie visuelle est inversee** : le decor domine le sujet.
//
// Ici l'altitude n'est plus mesuree contre elle-meme mais contre LE NIVEAU JOUE. Ce qui est a
// hauteur de jeu reste clair et contraste ; ce qui s'en eloigne, au-dessus comme au-dessous,
// perd en clarte et en saturation. Le decor ne disparait pas — il cesse d'etre le sujet.
//
// `dz` est l'ecart signe a ce niveau, en metres.
func TeinteNiveauDeJeu(dz, eclairement float64) color.RGBA {
	// Recession : 1 au niveau de jeu, tend vers un plancher quand on s'en eloigne. Le carre
	// donne un plateau autour du niveau joue au lieu d'un cone — les etages proches restent
	// tous lisibles, la falaise lointaine seule s'efface.
	x := math.Min(1, math.Abs(dz)/PorteeNiveauDeJeu)
	recul := 1 - 0.62*x*x

	// Teinte : legerement chaude au niveau de jeu, froide en s'eloignant vers le haut, plus
	// sombre et neutre vers le bas. L'ordre reste lisible sans inverser la hierarchie.
	chaud := [3]float64{0.92, 0.90, 0.86}
	froid := [3]float64{0.42, 0.50, 0.60}
	if dz < 0 {
		froid = [3]float64{0.34, 0.36, 0.42}
	}
	var c [3]uint8
	for a := 0; a < 3; a++ {
		v := chaud[a]*(1-x) + froid[a]*x
		c[a] = uint8(math.Round(255 * math.Max(0, math.Min(1, v*eclairement*recul))))
	}
	return color.RGBA{c[0], c[1], c[2], 255}
}

// TeinteEncre rend une carte quasi monochrome : la lisibilite vient de l'ombrage et des
// aretes, pas de la couleur. L'ecart au niveau de jeu ne joue que sur la valeur.
func TeinteEncre(dz, eclairement float64) color.RGBA {
	x := math.Min(1, math.Abs(dz)/PorteeNiveauDeJeu)
	v := (0.30 + 0.70*eclairement) * (1 - 0.45*x)
	g := uint8(math.Round(255 * math.Max(0, math.Min(1, v))))
	return color.RGBA{g, g, g, 255}
}

// TeinteAltitude rend la couleur d'une altitude normalisee dans [0,1], modulee par
// l'eclairement. `t` hors bornes est ECRETE, jamais extrapole.
func TeinteAltitude(t, eclairement float64) color.RGBA {
	t = math.Max(0, math.Min(1, t))
	x := t * float64(len(nuancierAltitude)-1)
	k := int(x)
	if k >= len(nuancierAltitude)-1 {
		k = len(nuancierAltitude) - 2
	}
	f := x - float64(k)
	var c [3]uint8
	for a := 0; a < 3; a++ {
		v := nuancierAltitude[k][a]*(1-f) + nuancierAltitude[k+1][a]*f
		c[a] = uint8(math.Round(255 * math.Max(0, math.Min(1, v*eclairement))))
	}
	return color.RGBA{c[0], c[1], c[2], 255}
}

// TeinteSolSuppose rend l'aplat des cellules comblees (cf. Rendu.CombleTrous).
//
// IL DOIT SE DISTINGUER DU RELEVE, sans crier. Un gris neutre, sans arete ni ombrage — c'est
// justement l'absence de modele qui le caracterise : la ou le rendu a des surfaces, il a aussi
// des aretes et un eclairement, et un aplat parfaitement uni se lit comme « ici, on ne sait
// pas ». La valeur est prise LEGEREMENT SOUS le sol joue moyen des deux habillages, pour que
// le comblement recule au lieu d'attirer l'oeil.
func TeinteSolSuppose(style StyleFond) color.RGBA {
	if style == StyleEncre {
		return color.RGBA{178, 178, 178, 255}
	}
	return color.RGBA{92, 96, 104, 255}
}
