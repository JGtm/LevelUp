package replay

// identity_registry_vacant_intercale_test.go — LA TABLE A VACANT INTERCALE, EMPLOYEE QUAND LES RECORDS
// DE CREATION CONFIRMENT LE RANG ABSOLU, REFUSEE SINON.
//
// # MUTATION
//
// Faire rendre vrai a [rangAbsoluConfirmeParLesCreations] des que la table est lue a vacant intercale
// (retirer les deux boucles de confirmation) : les deux sous-tests de refus rougissent. Retirer la
// seule seconde condition (index sur un vacant) : le sous-test « corps au rang du vacant » rougit.

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/identity"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// tableAVacantIntercale : sept sieges aux rangs 0, 1, 3..7, le rang 2 vacant entre deux occupes.
func tableAVacantIntercale() grammar.FilmPlayerTable {
	var sieges []grammar.FilmPlayerSeat
	for _, r := range []int{0, 1, 3, 4, 5, 6, 7} {
		sieges = append(sieges, grammar.FilmPlayerSeat{FilmIndex: r, XUID: uint64(100 + r)})
	}
	t := tableDuFilm(sieges...)
	t.InterleavedVacant = true
	return t
}

// creationsAuxIndex fabrique un record de creation par index, un slot de bipede chacun.
func creationsAuxIndex(index ...uint32) []grammar.BipedCreation {
	out := make([]grammar.BipedCreation, 0, len(index))
	for i, pi := range index {
		out = append(out, grammar.BipedCreation{Slot: uint32(512 + i), Generation: 1, HasIndex: true,
			ParticipantIndex: pi, TimestampUS: uint64(1000 * (i + 1))})
	}
	return out
}

func TestCompositionEmploieUnVacantIntercaleConfirmeParLesCreations(t *testing.T) {
	in := entreeIdentite(tableAVacantIntercale(), map[uint64]int{})
	// Les sept rangs absolus portent un corps ; l index 9 (un arrivant) tombe au-dela du dernier siege.
	in.BipedCreations = creationsAuxIndex(0, 1, 3, 4, 5, 6, 7, 9)
	got := composerTableDIndex(in)

	if !got.couverture.Read || got.couverture.Refusal != "" {
		t.Fatalf("lu=%v refus=%q : le rang absolu est confirme, la table doit etre employee",
			got.couverture.Read, got.couverture.Refusal)
	}
	for _, r := range []int{0, 1, 3, 4, 5, 6, 7} {
		if got.table.ByXUID[uint64(100+r)] != r {
			t.Errorf("siege de rang %d -> index %d : le rang ABSOLU est l index", r,
				got.table.ByXUID[uint64(100+r)])
		}
		if got.voieDuLienDIndex(uint64(100+r)) != identity.MethodFilmPlayerTable {
			t.Errorf("siege de rang %d : voie %q, attendue %q", r,
				got.voieDuLienDIndex(uint64(100+r)), identity.MethodFilmPlayerTable)
		}
	}
	if got.couverture.Direct != 7 || got.couverture.Fallback != 0 {
		t.Errorf("direct=%d repli=%d, attendus 7 et 0", got.couverture.Direct, got.couverture.Fallback)
	}
}

func TestCompositionRefuseUnVacantIntercaleQueLesCreationsNeConfirmentPas(t *testing.T) {
	cas := map[string][]grammar.BipedCreation{
		// Le siege de rang 3 n a aucun corps a son rang : rien ne confirme le rang absolu.
		"siege sans corps a son rang": creationsAuxIndex(0, 1, 4, 5, 6, 7, 8),
		// Un corps porte l index du vacant intercale : c est ce que predit l index parmi les occupes.
		"corps au rang du vacant":  creationsAuxIndex(0, 1, 2, 3, 4, 5, 6, 7),
		"aucun record de creation": nil,
	}
	for nom, creations := range cas {
		t.Run(nom, func(t *testing.T) {
			in := entreeIdentite(tableAVacantIntercale(), map[uint64]int{103: 3})
			in.BipedCreations = creations
			got := composerTableDIndex(in)
			if got.couverture.Read {
				t.Fatal("rang absolu non confirme : la table ne doit PAS etre employee")
			}
			if got.couverture.Refusal != identity.FilmTableInterleavedVacant {
				t.Errorf("refus %q, attendu %q", got.couverture.Refusal, identity.FilmTableInterleavedVacant)
			}
			if got.couverture.Direct != 0 || got.couverture.Fallback != 1 || got.table.ByXUID[103] != 3 {
				t.Errorf("direct=%d repli=%d lien=%d : seul le controle doit poser le lien",
					got.couverture.Direct, got.couverture.Fallback, got.table.ByXUID[103])
			}
		})
	}
}

// TestRangAbsoluNeSePoseQueSurUnVacantIntercale : la question ne se pose pas hors d une table LUE a
// vacant intercale, et la reponse y est toujours non.
func TestRangAbsoluNeSePoseQueSurUnVacantIntercale(t *testing.T) {
	creations := creationsAuxIndex(0, 1, 2, 3)
	sansVacant := tableDuFilm(grammar.FilmPlayerSeat{FilmIndex: 0, XUID: 1}, grammar.FilmPlayerSeat{FilmIndex: 1, XUID: 2})
	refusee := tableAVacantIntercale()
	refusee.Refusal = grammar.FilmTableTruncated
	for nom, tb := range map[string]grammar.FilmPlayerTable{"sans vacant": sansVacant, "refusee": refusee, "vide": {}} {
		if rangAbsoluConfirmeParLesCreations(tb, creations) {
			t.Errorf("%s : le rang absolu ne se confirme que sur une table lue a vacant intercale", nom)
		}
	}
}
