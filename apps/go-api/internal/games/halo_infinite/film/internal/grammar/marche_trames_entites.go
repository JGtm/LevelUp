package grammar

// marche_trames_entites.go — LA TABLE D ENTITES DE LA MARCHE, EN LECTURE SEULE (ADR 0037 IR-5).
//
// Le [World] est l implantation : UNE table, indexee par le slot de l eid, la vue en attribut.
// La structure de lecture n en voit que cette vue-ci, qui ne pose ni ne retire aucune liaison.

import "levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"

// entitesDuMonde expose un [World] comme [lecture.Entites].
type entitesDuMonde struct{ w *World }

// Entite rend la liaison du slot ; faux quand il n est pas lie.
func (e entitesDuMonde) Entite(slot uint32) (lecture.Entite, bool) {
	s, ok := e.w.slots[slot&0x3fffffff]
	if !ok {
		return lecture.Entite{}, false
	}
	return lecture.Entite{EID: s.FullID, TI: uint16(s.TypeIndex), Vue: rangDeVueDuFilm(s.Vue), //nolint:gosec // archetype < 64
		Liaison: s.Liaison, GenerationConnue: !s.GenAny}, true
}

// Liees rend le nombre de slots lies.
func (e entitesDuMonde) Liees() int { return e.w.Bound() }

// rangDeVueDuFilm convertit un index de vue de la marche hors ligne en rang de vue du film : la
// marche numerote a partir de la vue des entites, que le film range au rang 1
// ([vueDeLImageCle]).
func rangDeVueDuFilm(v int8) int8 {
	if v < 0 {
		return lecture.VueInconnue
	}
	return v + int8(lecture.RangVueB) //nolint:gosec // rang 1
}
