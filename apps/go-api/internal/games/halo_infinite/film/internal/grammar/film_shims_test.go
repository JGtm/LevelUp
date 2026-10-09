package grammar

// film_shims_test.go — LES ENVELOPPES D2 DES HELPERS INTERNES, RESERVEES AUX TESTS.
//
// Le lot 1 de PLAN_CUISSON_PERF (2026-09-02) a fait passer les balayages et leurs helpers d'un
// REPERTOIRE a un `*source.Film` deja charge : la cuisson ne decompresse plus le film une
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

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// filmDeDir charge le film d'un repertoire, ou rend nil. Les helpers ci-dessous rendent alors
// leur resultat vide ou leur erreur habituelle — exactement ce que faisait un repertoire
// illisible avant le lot 1.
func filmDeDir(dir string) *source.Film {
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		return nil
	}
	return film
}

// bipedSlotBandDir : [bipedSlotBand] depuis un repertoire.
func bipedSlotBandDir(dir string, chunks []int) SlotBand {
	return bipedSlotBand(NewFilmContext(filmDeDir(dir)), chunks)
}

// worldObjectSlotBandDir : [worldObjectSlotBand] depuis un repertoire (`n` ignore).
func worldObjectSlotBandDir(dir string, _, typeIndex int) map[uint32]bool {
	return worldObjectSlotBand(NewFilmContext(filmDeDir(dir)), typeIndex)
}

// observedSlotBandDir : [observedSlotBand] depuis un repertoire (`n` ignore).
func observedSlotBandDir(dir string, _, typeIndex int) map[uint32]bool {
	return observedSlotBand(NewFilmContext(filmDeDir(dir)), typeIndex)
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
func vehicleI0LayoutDir(dir string) (profile.I0Layout, error) {
	return NewFilmContext(filmDeDir(dir)).I0Layout()
}
