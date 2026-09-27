package replay

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// identity_registry_corps_test.go — LE REGISTRE RAISONNE PAR CORPS (slot, generation) (lot J5.4 du
// plan de suite de l'audit du decodeur, 2026-09-27).

// creationGen fabrique un record de creation lu d'une GENERATION donnee : le corps que le slot
// porte a partir de cette date.
func creationGen(slot uint32, tUS uint64, index, gen uint32) grammar.BipedCreation {
	c := creationDe(slot, tUS, index)
	c.Generation = gen
	return c
}

// TestRegistre_RecordDeCreationNOuvreQueSaVie (RA2-1) : sur un slot RECYCLE, le record d'un corps
// n'ouvre qu'une vie de CE corps. Ici le premier corps du slot 100 (generation 1, index 0) ne
// replique aucune position ; le second (generation 2, index 1) nait a 12 s et vit [14 s..18 s].
//
// ROUGE AVANT : `viesOuvertes` donnait au premier record « la premiere vie non terminee a sa date »
// — la vie du corps SUIVANT —, publiee `direct` sous le joueur de l'index 0.
//
// MUTATION : retirer la garde de corps de `viesOuvertes` -> la vie porte 111, rouge.
func TestRegistre_RecordDeCreationNOuvreQueSaVie(t *testing.T) {
	var pos []grammar.BipedPosition
	for ts := uint64(14_000_000); ts <= 18_000_000; ts += 500_000 {
		pos = append(pos, posAt(100, ts, 1, 1, 1))
	}
	for ts := uint64(1_000_000); ts <= 18_000_000; ts += 500_000 {
		pos = append(pos, posAt(200, ts, 2, 2, 2))
	}
	in := filmDeuxCorps()
	in.Positions = pos
	in.BipedCreations = []grammar.BipedCreation{
		creationGen(100, 500_000, 0, 1),
		creationGen(100, 12_000_000, 1, 2),
		creationDe(200, 1_000_000, 1),
	}
	in.PlayerIndices.ByXUID = map[uint64]int{111: 0, 222: 1, 333: 2}
	in.BipedCreations[2].ParticipantIndex = 2
	reg := BuildIdentityRegistry(in)
	var vue bool
	for _, l := range reg.Vies() {
		if l.slot != 100 {
			continue
		}
		vue = true
		if l.xuid != 222 {
			t.Fatalf("vie 100[%d..%d] : xuid = %d, attendu 222 — le record du corps precedent a "+
				"ouvert la vie du corps suivant", l.from, l.to, l.xuid)
		}
		if l.nomPar != NomParCreation {
			t.Fatalf("vie 100 : voie %q, attendu %q (le record de SON corps l'ouvre)", l.nomPar,
				NomParCreation)
		}
	}
	if !vue {
		t.Fatal("aucune vie sur le slot 100 : la decoupe a change")
	}
}
