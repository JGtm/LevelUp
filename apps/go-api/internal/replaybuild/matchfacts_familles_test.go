package replaybuild

// matchfacts_familles_test.go — LE DENOMINATEUR DE `coverage.objectives` NE COMPTE QUE LES
// FAMILLES D'OBJECTIF (D.2, 2026-09-13).
//
// PREUVE D'ENTREE (audit du 2026-09-10 §12-1) : les tables nommees portent `kills` (ancre
// d'identite du balayage, `comp 2 A`) et `assists` (controle croise). Publies comme les autres,
// ils faisaient 119 des 218 « actions disponibles » de `8bc6074f` et 93 des 148 de `32d9a94f` —
// deux artefacts qui annoncaient 100 % de couverture d'objectifs sur un calque dont la majorite
// n'est pas un objectif.

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// recordsCTFAvecFragsDeTest — la meme fixture que [recordsCTFDeTest], plus le compteur de FRAGS
// (`comp 2 A`). C'est la forme reelle d'un film : les deux emplacements bougent cote a cote.
func recordsCTFAvecFragsDeTest() []objectiveevents.StatRecord {
	var out []objectiveevents.StatRecord
	for i := 1; i <= 4; i++ {
		for _, slot := range []int{10, 12} {
			out = append(out, objectiveevents.StatRecord{
				TimeMS: i * 1_000, Slot: slot,
				Comps: map[int]objectiveevents.StatValue{
					22: {A: int64(i)},     // flag_grabs — famille d'objectif
					2:  {A: int64(2 * i)}, // kills — ancre d'identite, JAMAIS un objectif
				},
			})
		}
	}
	return out
}

// TestComptesDeCouvertureNeGardentQueLesObjectifs couvre les TROIS sorties d'[identifiedEvents]
// qui alimentent la couverture : le pont incomplet, le refus d'effectif et le fil des morts
// illisible.
//
// MUTATION : rendre `len(named)` au lieu de `objectiveevents.CountObjectiveFamily(named)` sur
// l'une des trois -> le compte remonte au total nomme, rouge.
func TestComptesDeCouvertureNeGardentQueLesObjectifs(t *testing.T) {
	recs := recordsCTFAvecFragsDeTest()
	nommees := objectiveevents.NamedEventsFrom(recs, objectiveevents.ObjectiveTypeFlag)
	objectifs := objectiveevents.CountObjectiveFamily(nommees)
	if objectifs == 0 || objectifs == len(nommees) {
		t.Fatalf("fixture non discriminante : %d objectif(s) sur %d nommee(s)",
			objectifs, len(nommees))
	}
	pont := func() *pontParManche { return &pontParManche{recs: recs} }
	ctx := context.Background()

	// PONT INCOMPLET — sans identite, tout le calque part en `noSlot` ; seuls les objectifs
	// entrent au denominateur.
	_, nonNommes, refuses := identifiedEvents(ctx, "m", filmDeaths{}, recs,
		rosterDe(objectiveevents.StatPlayerSlots), pont())
	if nonNommes != objectifs || refuses != 0 {
		t.Errorf("pont incomplet : nonNommes = %d, refuses = %d ; attendu %d et 0",
			nonNommes, refuses, objectifs)
	}

	// REFUS D'EFFECTIF — le calque se tait entierement, et le compte du refus est le meme
	// denominateur.
	_, _, refuses = identifiedEvents(ctx, "m", filmDeaths{}, recs,
		rosterDe(objectiveevents.StatPlayerSlots+1), pont())
	if refuses != objectifs {
		t.Errorf("refus d'effectif : refuses = %d, attendu %d", refuses, objectifs)
	}

	// FIL DES MORTS ILLISIBLE — le calque est integralement perdu, et c'est cette perte-la,
	// restreinte aux objectifs, qu'il faut publier.
	_, nonNommes, _ = identifiedEvents(ctx, "m", filmDeaths{err: errors.New("fil illisible")},
		recs, rosterDe(objectiveevents.StatPlayerSlots), pont())
	if nonNommes != objectifs {
		t.Errorf("fil des morts illisible : nonNommes = %d, attendu %d", nonNommes, objectifs)
	}
}
