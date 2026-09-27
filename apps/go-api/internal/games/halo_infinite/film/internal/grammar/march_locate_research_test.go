//go:build research

package grammar

// march_locate_research_test.go — L ENVELOPPE A UN RENDU DU LOCALISATEUR, POUR LES INSTRUMENTS.
//
// Le localisateur de production est [marchLocalise] (lot J8.7, 2026-09-27) : il rend aussi le verdict
// du repli `repli_localisation_largeur_libre`. Les instruments de recherche du paquet n en lisent que
// la position ; l enveloppe qu ils appelaient n avait plus d appelant de production et vit donc ici.

// marchLocate rend la seule position de [marchLocalise].
func marchLocate(pay []byte, w *World, cfg FrameConfig) int {
	s, _ := marchLocalise(pay, w, cfg)
	return s
}
