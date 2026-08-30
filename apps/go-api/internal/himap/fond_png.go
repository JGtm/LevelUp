// Package himap — fond_png.go : LE FORMAT PUBLIE du fond de carte, et son CALAGE.
//
// LA LECON QUI FONDE CE FICHIER. `carte_validee_v1.png`, la seule image de carte validee a
// l'oeil, a ete produite sans que son calage soit ecrit nulle part. Il a fallu le RETROUVER a
// la main sur les trajectoires de joueur — 0,0920 m/px, X0 -43,5, Y1 61,0. Une image dont on ne
// sait pas ou elle est posee dans le monde est inexploitable : elle se regarde, elle ne se
// superpose a rien. Le calage voyage donc AVEC l'asset, toujours.
//
// LES DECISIONS DE FORMAT, prises ici et publiees ici :
//
//  1. PNG RGBA 8 bits, fond TRANSPARENT (alpha 0 la ou il n'y a pas de matiere). Pas de noir :
//     le noir est une couleur, et il se retrouverait tel quel dans les trous au milieu de la
//     zone jouee. Un canal alpha laisse la couche d'affichage poser son propre fond — demande
//     de l'utilisateur au gate du 2026-08-09.
//  2. Echelle EchelleFondCarte = 0,0920 m/px, la MEME que la carte validee. Un nombre rond
//     (0,10) aurait ete plus joli et aurait oblige a re-valider les trois cartes deja jugees a
//     l'oeil : l'asset produit ne serait plus celui qui a ete approuve. La regle prime sur
//     l'esthetique du nombre.
//  3. Cadre = voisinage des ancres (MargeCadre), donc une taille d'image PROPRE A CHAQUE CARTE.
//     Pas de taille fixe : imposer une toile commune obligerait a choisir entre rogner les
//     grandes cartes et gaspiller sur les petites.
//  4. Le calage voyage dans un SIDECAR JSON par carte, a cote du PNG (cf.
//     internal/analysis/replay/map_background.go pour le type et le lecteur).
//
// POURQUOI UN SIDECAR PAR CARTE ET PAS UN MANIFESTE UNIQUE : une re-cuisson partielle (une
// seule carte) ne peut alors jamais abimer l'entree d'une autre, et chaque PNG porte sa propre
// verite — c'est exactement le defaut de `carte_validee_v1.png` qu'on corrige. Lister les fonds
// disponibles reste un parcours de repertoire.
//
// POURQUOI PAS UN CHUNK tEXt DANS LE PNG : l'encodeur de la bibliotheque standard Go n'expose
// aucun moyen d'ecrire un chunk de texte. Le faire demanderait un encodeur maison — beaucoup de
// code pour rendre le calage MOINS lisible qu'un JSON.
package himap

import (
	"image"
	"image/color"
)

// EchelleFondCarte : cote d'un pixel du fond de carte, en metres.
//
// Egale au calage mesure de `carte_validee_v1.png` (cf. `gateEchelle` du banc), et ce n'est pas
// une coincidence : produire a cette echelle rend l'asset directement comparable pixel a pixel
// avec la seule image que l'utilisateur ait validee, et avec le banc de non-regression.
// `TestEchelleDeProductionEgaleCelleDuBanc` interdit qu'elles divergent en silence.
const EchelleFondCarte = 0.0920

// StyleFond : l'habillage du fond de carte. La geometrie ne change pas d'un style a l'autre —
// seule la mise en couleur.
type StyleFond string

const (
	// StyleDoux : Lambert continu, l'habillage d'origine. Il ne peut PAS separer deux dalles de
	// meme inclinaison a des hauteurs differentes — c'est structurel, pas un reglage.
	StyleDoux StyleFond = "doux"
	// StylePlat : aplats d'eclairement + aretes.
	StylePlat StyleFond = "plat"
	// StyleAltitude : nuancier d'altitude + ombrage.
	StyleAltitude StyleFond = "altitude"
	// StyleCombine : nuancier d'altitude + aplats + aretes. Choix de l'utilisateur au gate du
	// 2026-08-09, avant que la hierarchie visuelle inversee soit diagnostiquee.
	StyleCombine StyleFond = "combine"
	// StyleJeu : teinte par l'ecart au NIVEAU JOUE + aplats + aretes.
	StyleJeu StyleFond = "jeu"
	// StyleEncre : quasi monochrome, la lisibilite vient des aretes.
	StyleEncre StyleFond = "encre"
)

// StyleFondParDefaut : l'habillage des assets produits.
//
// POURQUOI `jeu` ET PAS `combine`. `combine` normalise l'altitude entre les centiles 2 et 98 de
// TOUTE la matiere dessinee : sur une carte encaissee, ce sont les rochers hauts qui prennent
// le haut de la rampe, donc le blanc, donc l'oeil — pendant que l'arene, plus basse, recule
// dans le sombre. La hierarchie visuelle est inversee, le decor domine le sujet. C'est le
// defaut que l'utilisateur a signale le 2026-08-10 (« affiner la couleur ») et que
// `TeinteNiveauDeJeu` corrige en mesurant l'altitude contre le niveau JOUE.
//
// A RE-JUGER PAR L'UTILISATEUR : les trois cartes validees a l'oeil (Cliffhanger, Catalyst,
// Vagabond) l'ont ete en `combine`, AVANT que `jeu` existe. Le style est ecrit dans le sidecar
// de chaque asset, et une re-cuisson sous un autre style est un simple rejeu de la commande.
const StyleFondParDefaut = StyleJeu

// StylesFond liste les habillages connus, pour la validation d'une entree utilisateur.
var StylesFond = []StyleFond{StyleDoux, StylePlat, StyleAltitude, StyleCombine, StyleJeu, StyleEncre}

// StyleFondValide dit si une chaine designe un habillage connu.
func StyleFondValide(s StyleFond) bool {
	for _, c := range StylesFond {
		if c == s {
			return true
		}
	}
	return false
}

// CalageDuRendu rend le calage monde -> pixel d'un rendu : abscisse du bord GAUCHE, ordonnee du
// bord HAUT, et cote d'un pixel en metres.
//
// L'ordonnee est celle du bord HAUT parce que l'image descend en Y quand le monde monte — c'est
// le seul endroit du package ou ce retournement est ecrit, et `FondPNG` en est l'unique autre
// depositaire. `TestCalageEtImageDisentLaMemeChose` les confronte.
//
// Convention publiee :
//
//	xMonde = origineX + (px + 0,5) x metresParPixel
//	yMonde = origineY - (py + 0,5) x metresParPixel
func CalageDuRendu(r *Rendu) (origineX, origineY, metresParPixel float64) {
	return r.Min[0], r.Min[1] + float64(r.NY)*r.Cell, r.Cell
}

// ConventionCalage est la formule de calage, publiee TELLE QUELLE dans chaque sidecar.
//
// Une formule dans un fichier de code ne suit pas l'asset ; celle-ci si. C'est la reponse
// directe au cout paye sur `carte_validee_v1.png`.
const ConventionCalage = "xMonde = originX + (px + 0.5) * metersPerPixel ; " +
	"yMonde = originY - (py + 0.5) * metersPerPixel"

// FondPNG encode un rendu en image : le LIVRABLE, par opposition aux planches de revue.
//
// L'empreinte des instances non resolues n'y figure pas — c'est un outil de diagnostic, pas un
// fond de carte.
//
// `larg`/`haut` valent normalement `r.NX`/`r.NY` ; le banc de Cliffhanger les fournit differents
// pour rendre dans la grille de pixels de la carte validee. `masque` non nil ne garde que les
// pixels ou il vaut true — repli explicite reserve a Cliffhanger, jamais une regle.
func FondPNG(r *Rendu, larg, haut int, masque []bool, style StyleFond) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, larg, haut))
	p := peintre{r: r, style: style}
	p.bas, p.haute, p.bornesOK = r.BornesAltitudeRobustes()
	for py := 0; py < haut; py++ {
		j := haut - 1 - py
		for px := 0; px < larg; px++ {
			if masque != nil && !masque[py*larg+px] {
				continue
			}
			if c, ok := p.pixel(px, j); ok {
				img.Set(px, py, c)
			}
		}
	}
	return img
}

// peintre porte ce qui est constant sur toute l'image : les bornes d'altitude robustes sont
// calculees une fois, pas par pixel.
type peintre struct {
	r        *Rendu
	style    StyleFond
	bas      float64
	haute    float64
	bornesOK bool
}

// pixel rend la couleur d'une cellule, ou `false` s'il n'y a rien a dessiner (alpha 0).
func (p peintre) pixel(px, j int) (color.RGBA, bool) {
	// L'eau (volumes sddt) prend le pixel — y compris la ou le rendu n'a pas de matiere :
	// l'eau n'est pas dans les instances de rendu, c'est justement pourquoi elle manquait.
	if p.r.Eau(px, j) {
		return TeinteEau(p.r.BordEau(px, j)), true
	}
	// Le SOL SUPPOSE se peint la ou aucune surface n a ete relevee, et il se peint AUTREMENT :
	// c est un aplat assume, pas une mesure (cf. Rendu.CombleTrous).
	if p.r.SolSuppose(px, j) {
		return TeinteSolSuppose(p.style), true
	}
	e, ok := p.r.Eclairement(px, j)
	if !ok {
		return color.RGBA{}, false // alpha 0 : pas de matiere, pas de pixel
	}
	if p.aplats() {
		e, _ = p.r.EclairementPlat(px, j)
	}
	c := p.teinte(px, j, e)
	// L'arete ASSOMBRIT ce que la teinte a rendu : c'est elle qui separe deux dalles de meme
	// inclinaison a des hauteurs differentes, ce qu'aucun degrade ne peut faire.
	if p.style != StyleDoux && p.style != StyleAltitude && p.r.Arete(px, j) {
		c = color.RGBA{c.R / 3, c.G / 3, c.B / 3, 255}
	}
	return c, true
}

// aplats dit si l'habillage quantifie l'eclairement en paliers.
func (p peintre) aplats() bool {
	switch p.style {
	case StylePlat, StyleCombine, StyleJeu, StyleEncre:
		return true
	default:
		return false
	}
}

// teinte rend la couleur d'une cellule avant le trace des aretes.
func (p peintre) teinte(px, j int, e float64) color.RGBA {
	switch p.style {
	// Les deux habillages qui remettent l'ARENE au premier plan : l'altitude y est mesuree
	// contre le niveau JOUE, pas contre elle-meme.
	case StyleJeu, StyleEncre:
		dz, ok := p.r.EcartAuNiveauDeJeu(px, j)
		if !ok {
			dz = 0
		}
		if p.style == StyleJeu {
			return TeinteNiveauDeJeu(dz, e)
		}
		return TeinteEncre(dz, e)
	case StyleAltitude, StyleCombine:
		if p.bornesOK {
			z, _ := p.r.Altitude(px, j)
			return TeinteAltitude((z-p.bas)/(p.haute-p.bas), e)
		}
	}
	g := uint8(255 * e)
	return color.RGBA{g, g, g, 255}
}
