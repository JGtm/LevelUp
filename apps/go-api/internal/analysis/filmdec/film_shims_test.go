package filmdec

// film_shims_test.go — LES ENVELOPPES D2 DES HELPERS INTERNES, RESERVEES AUX TESTS.
//
// Le lot 1 de PLAN_CUISSON_PERF (2026-09-02) a fait passer les balayages et leurs helpers d'un
// REPERTOIRE a un `*filmsource.Film` deja charge : la cuisson ne decompresse plus le film une
// fois par balayage. Les helpers NON EXPORTES (`bipedSlotBand`, `worldObjectSlotBand`,
// `observedSlotBand`, les lecteurs d'archetype) sont appeles par une quarantaine de tests
// internes — instruments de mesure, sondes de corpus, gardes de recherche — qui, eux, n'ont
// qu'un chemin de film sous la main.
//
// CES ENVELOPPES VIVENT DANS UN `_test.go`, ET C'EST LE POINT : elles ne sont compilees que par
// la suite de tests, donc elles ne peuvent pas etre appelees par erreur depuis un chemin de
// production — la regle D2 (« enveloppes tolerees hors production, interdites en production »)
// est tenue par le compilateur, pas par une convention.
//
// Le parametre `n` (l'ancien `CountFilmChunks(dir)`) est CONSERVE et IGNORE : les sites d'appel
// restent inchanges, et le film charge enumere lui-meme ses chunks de donnees. C'est ce qui a
// permis une migration purement mecanique de ces sites.

import "levelup/go-api/internal/analysis/filmsource"

// filmDeDir charge le film d'un repertoire, ou rend nil. Les helpers ci-dessous rendent alors
// leur resultat vide ou leur erreur habituelle — exactement ce que faisait un repertoire
// illisible avant le lot 1.
func filmDeDir(dir string) *filmsource.Film {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil
	}
	return film
}

// bipedSlotBandDir : [bipedSlotBand] depuis un repertoire.
func bipedSlotBandDir(dir string, chunks []int) SlotBand {
	return bipedSlotBand(filmDeDir(dir), chunks)
}

// worldObjectSlotBandDir : [worldObjectSlotBand] depuis un repertoire (`n` ignore).
func worldObjectSlotBandDir(dir string, _, typeIndex int) map[uint32]bool {
	return worldObjectSlotBand(filmDeDir(dir), typeIndex)
}

// observedSlotBandDir : [observedSlotBand] depuis un repertoire (`n` ignore).
func observedSlotBandDir(dir string, _, typeIndex int) map[uint32]bool {
	return observedSlotBand(filmDeDir(dir), typeIndex)
}

// bipedArchetypeDir : [FilmContext.bipedArchetype] depuis un repertoire.
func bipedArchetypeDir(dir string) (Archetype, error) {
	return NewFilmContext(filmDeDir(dir)).bipedArchetype()
}

// objectiveArchetypeDir : [FilmContext.objectiveArchetype] depuis un repertoire.
func objectiveArchetypeDir(dir string) (Archetype, *Registry, error) {
	return NewFilmContext(filmDeDir(dir)).objectiveArchetype()
}

// filmArchetypeDir : [FilmContext.filmArchetype] depuis un repertoire.
func filmArchetypeDir(dir string, ti int) (Archetype, *Registry, error) {
	return NewFilmContext(filmDeDir(dir)).filmArchetype(ti)
}

// vehicleArchetypeDir : [FilmContext.vehicleArchetype] depuis un repertoire.
func vehicleArchetypeDir(dir string) (Archetype, error) {
	return NewFilmContext(filmDeDir(dir)).vehicleArchetype()
}

// vehicleI0LayoutDir : le decoupage d'i0 d'un film de repertoire, AUTO-DETECTE.
//
// C'est bien l'auto-detection ici, et non la regle du catalogue : ces instruments de mesure ne
// disposent d'aucune entree de carte, et c'est exactement ce que faisait `vehicleI0Layout(dir)`
// avant la migration.
func vehicleI0LayoutDir(dir string) (I0Layout, error) {
	return NewFilmContext(filmDeDir(dir)).I0Layout()
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
