package replay

// build_arrondis.go — LES DEUX ARRONDIS DE PUBLICATION.
//
// EXTRAITS DE `build.go` AU LOT 1.9.14, par DEPLACEMENT PUR : ce fichier-la passait 500 lignes
// et le lot lui ajoute la pose des sieges. Ces deux fonctions ne decident rien de
// l assemblage — elles mettent un flottant a la precision du contrat — et se lisent tres bien
// seules.

import "math"

// round2 arrondit au centième (cf. coordScale).
func round2(v float32) float32 {
	return float32(math.Round(float64(v)*coordScale) / coordScale)
}

// fractionForJSON arrondit une fraction [0,1] au millième et la rend par POINTEUR : c'est
// ce pointeur qui permet de publier un ZÉRO (bouclier brisé) sans qu'omitempty le confonde
// avec une absence de mesure. Cf. Point.Sh.
func fractionForJSON(v float32) *float32 {
	r := float32(math.Round(float64(v)*1000) / 1000)
	return &r
}
