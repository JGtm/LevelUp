//go:build research

package grammar

// march_locate_research_test.go — L ENVELOPPE A UN RENDU DU LOCALISATEUR, POUR LES INSTRUMENTS.
//
// Le localisateur de production est [LocaliserBoucleDeRecords] : il rend aussi le verdict du repli
// `repli_localisation_largeur_libre`. Les instruments de recherche du paquet n en lisent que la
// position, dans l ordre des marches qui lisent les morts.

// marchLocate rend la seule position de [LocaliserBoucleDeRecords], ordre [SignaturePuisLargeurLibre].
func marchLocate(pay []byte, w *World, cfg FrameConfig) int {
	s, _ := LocaliserBoucleDeRecords(pay, w, cfg, SignaturePuisLargeurLibre)
	return s
}
