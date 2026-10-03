//go:build research

package grammar

// instruments_partages_film_shims_research_test.go — adaptateurs de film_shims_test.go dont les seuls
// utilisateurs sont des instruments `research` (J12.7 lint). Deplacement pur.

// bipedArchetypeDir : [FilmContext.bipedArchetype] depuis un repertoire.
func bipedArchetypeDir(dir string) (Archetype, error) {
	return NewFilmContext(filmDeDir(dir)).bipedArchetype()
}

// bipedSlotBandMapDir : [bipedSlotBand] depuis un repertoire, rendue en ENSEMBLE.
//
// Les instruments de mesure des lots V5 a V8 tiennent leur bande dans une `map[uint32]bool` :
// ils la croisent avec des bandes d'objets du monde (qui sont des ensembles), en retirent des
// slots, la parcourent. Leur convertir la bande dense ici coute une allocation par appel dans un
// test, et evite de reecrire des dizaines de lignes de mesure — la bande dense sert la
// PRODUCTION, ou elle est consultee par bit candidat.
func bipedSlotBandMapDir(dir string, chunks []int) map[uint32]bool {
	out := map[uint32]bool{}
	for _, s := range bipedSlotBandDir(dir, chunks).Slots() {
		out[s] = true
	}
	return out
}
