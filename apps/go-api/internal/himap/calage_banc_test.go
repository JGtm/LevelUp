package himap

// calage_banc_test.go — le calage MESURE de la carte de reference du banc visuel.
//
// POURQUOI CES TROIS CONSTANTES VIVENT HORS DU TAG `gamefiles`. Elles decrivent un ASSET
// versionne (`carte_validee_v1.png`), pas l'installation du jeu. `fond_png_test.go` les oppose
// a `EchelleFondCarte` : si la production s'ecartait du calage du banc, l'asset produit ne
// serait plus comparable a la reference — et ce garde-rail doit rougir PARTOUT, y compris en
// CI ou le jeu n'est pas installe.
//
// Le banc qui les consomme (`carte_gate_gamefiles_test.go`, `cloture_gamefiles_test.go`,
// `rendu_gamefiles_test.go`, ...) reste, lui, derriere le tag.

// Calage de `carte_validee_v1.png`, mesure sur les trajectoires de joueur
// (0,0920 m/px, X0 -43,5, Y1 61,0 — cf. `ETAT_DU_POC.md`).
const (
	gateEchelle = 0.0920 // metres par pixel
	gateX0      = -43.5
	gateY1      = 61.0
)
