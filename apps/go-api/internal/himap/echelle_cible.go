package himap

import "math"

// echelle_cible.go — CHOISIR L'ECHELLE SANS REGLAGE PAR CARTE.
//
// Le probleme, constate au gate du 2026-08-26 : a l'echelle de production (0,0920 m/px), une
// arene 4v4 de 65 m rend une image de 700 px de cote. Agrandie dans le rejeu, elle est
// pixelisee — l'utilisateur l'a dit des la premiere planche (« faut generer des fonds de
// cartes a une taille minimale au moins pour pas que ce soit trop pixelise »).
//
// La reponse ecrite jusqu'ici etait une `echelle` par carte, calculee a la main, entree par
// entree. C'est exactement le « copy-paste config » que la revue interdit a la troisieme
// occurrence : la valeur n'exprime aucun choix propre a la carte, seulement la taille de son
// arene, qui est DEJA connue avant le rendu — le cadre monde vient des ancres.
//
// D'ou cette regle unique : viser une taille de GRILLE, en deduire le cote du pixel. La
// bornage a `EchelleFondCarte` garantit qu'on ne rend jamais plus GROSSIER que la production ;
// `EchelleLaPlusFine` empeche une carte minuscule de produire une image demesuree.
const (
	// CibleCadrePx est la cible, en pixels, du plus grand cote de la GRILLE. Le cadre publie
	// est ensuite rogne a la matiere : sur les onze cartes cuites le 2026-08-26, le rogne
	// garde entre 42 et 55 pour cent de la grille, soit 1 260 a 1 650 px de cote publie.
	CibleCadrePx = 3000
	// EchelleLaPlusFine borne le cote du pixel par le bas (garde-fou memoire : une grille de
	// 3 000 x 3 000 en cellules de 2,5 cm couvre deja 75 m).
	EchelleLaPlusFine = 0.025
)

// EchellePourCadre rend le cote de pixel a employer pour ces ancres.
//
// Priorite : l'echelle EXPLICITE de la carte si elle en a une (un gate utilisateur l'a fixee,
// il fait foi) ; sinon la cible automatique ; sinon l'echelle de production.
//
// La regle ne touche donc jamais une carte deja reglee a la main.
func EchellePourCadre(ancres [][3]float64, explicite float64, ciblePx int) float64 {
	if explicite > 0 {
		return explicite
	}
	if ciblePx <= 0 || len(ancres) == 0 {
		return EchelleFondCarte
	}
	lo := [2]float64{math.Inf(1), math.Inf(1)}
	hi := [2]float64{math.Inf(-1), math.Inf(-1)}
	for _, a := range ancres {
		for k := 0; k < 2; k++ {
			lo[k] = math.Min(lo[k], a[k]-MargeCadre)
			hi[k] = math.Max(hi[k], a[k]+MargeCadre)
		}
	}
	cote := math.Max(hi[0]-lo[0], hi[1]-lo[1])
	if !(cote > 0) {
		return EchelleFondCarte
	}
	return math.Min(EchelleFondCarte, math.Max(EchelleLaPlusFine, cote/float64(ciblePx)))
}
