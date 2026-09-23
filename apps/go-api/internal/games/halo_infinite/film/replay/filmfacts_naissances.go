package replay

// filmfacts_naissances.go — CE QUE LE FILM DIT DES ARMES A L INSTANT, DANS LES FAITS PERSISTES
// (lots M3.1 et M3.2 de la campagne « retours rejeu », 2026-09-23, blob v26).
//
// LA SANTE DE LA MARCHE D IMAGE-CLE VOYAGE AVEC LES FAITS pour la meme raison que les stats des
// canaux delta : le document la publie (`coverage.keyframes`), et un rejeu depuis les faits doit
// rendre le MEME document qu un decodage du film (equivalence S8 du lot 4.1). Une couverture
// recalculee a la relecture n existerait pas : elle se mesure pendant la marche.

import "levelup/go-api/internal/games/halo_infinite/film/internal/grammar"

// encodeMarcheImageCle ecrit la couverture de la marche d image-cle, champ par champ.
func encodeMarcheImageCle(w *gwriter, c grammar.KeyframeWalkCoverage) {
	for _, v := range []int{c.Payloads, c.Records, c.Bipedes, c.Voisins, c.Sauts, c.Recalages,
		c.Elections, c.Glissements, c.BipedesAbsentsEncadres} {
		w.u(uint64(v)) //nolint:gosec // compteurs positifs
	}
}

// decodeMarcheImageCle relit ce que [encodeMarcheImageCle] a ecrit, dans le meme ordre.
func decodeMarcheImageCle(r *greader) grammar.KeyframeWalkCoverage {
	var c grammar.KeyframeWalkCoverage
	for _, p := range []*int{&c.Payloads, &c.Records, &c.Bipedes, &c.Voisins, &c.Sauts, &c.Recalages,
		&c.Elections, &c.Glissements, &c.BipedesAbsentsEncadres} {
		*p = int(r.u()) //nolint:gosec // compteurs ecrits positifs
	}
	return c
}
