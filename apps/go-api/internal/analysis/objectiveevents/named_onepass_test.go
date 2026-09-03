package objectiveevents

// named_onepass_test.go — [NamedEventsFrom] regroupe desormais les emissions en UN SEUL
// balayage de `recs` ([rawSeriesByKey]) au lieu d'un balayage par emplacement. C'est un
// REFACTO PUR : ce test oppose la nouvelle sortie a une COPIE DE REFERENCE de l'ancienne
// forme (une serie par emplacement via [seriesBySlot], tri a trois cles), sur des
// enregistrements construits pour toucher chaque filtre de la marche : plusieurs
// emplacements du meme composant (cotes A et B), plusieurs manches, slots d'equipe a
// ignorer, valeurs negatives a rejeter, score de mode hors domaine a rejeter, et une manche
// fortuite que [RealRounds] ne retient pas.

import (
	"reflect"
	"sort"
	"testing"
)

// refNamedEventsFrom est [NamedEventsFrom] d'AVANT : une serie par emplacement, donc un
// balayage de `recs` et un calcul de [RealRounds] par emplacement, et un tri a trois cles.
func refNamedEventsFrom(recs []StatRecord, objectiveType string) []NamedEvent {
	table, ok := namedStatSlots[objectiveType]
	if !ok {
		return nil
	}
	b := newEventBudget("reference")
	var out []NamedEvent
	for key, slot := range table {
		if slot.Redundant {
			continue
		}
		for entity, pts := range seriesBySlot(recs, key) {
			for _, t := range incrementTimes(pts, key, b) {
				out = append(out, NamedEvent{
					TimeMS: t, Slot: entity, Stat: slot.Stat, Comp: key.Comp, Side: key.Side,
				})
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].TimeMS != out[j].TimeMS {
			return out[i].TimeMS < out[j].TimeMS
		}
		if out[i].Slot != out[j].Slot {
			return out[i].Slot < out[j].Slot
		}
		return out[i].Stat < out[j].Stat
	})
	return out
}

// onePassCorpus construit un jeu d'enregistrements qui touche chaque filtre de la marche.
func onePassCorpus() []StatRecord {
	rec := func(ms, slot, round int, comps map[int]StatValue) StatRecord {
		return StatRecord{TimeMS: ms, Slot: slot, Round: round, Comps: comps}
	}
	var out []StatRecord
	// Manche 0 : deux joueurs, plusieurs emplacements, dont les deux cotes de `comp 21`.
	for i, ms := range []int{1000, 2000, 3000, 4000, 5000, 6000} {
		v := int64(i + 1)
		out = append(out,
			rec(ms, 10, 0, map[int]StatValue{
				2: {A: v, B: 0}, 21: {A: v, B: v / 2}, 22: {A: v * 2, B: 0},
				0: {A: v, B: 0},
			}),
			rec(ms+50, 12, 0, map[int]StatValue{
				3: {A: v, B: 0}, 20: {A: 0, B: v}, 23: {A: v, B: 0},
			}),
			// Slot d'EQUIPE : ignore par le nommage.
			rec(ms+10, 6, 0, map[int]StatValue{0: {A: v * 10, B: 0}}),
		)
	}
	// Manche 1 : le compteur repart de zero — c'est le cumul qui doit le rattraper.
	for i, ms := range []int{20000, 21000, 22000, 23000, 24000, 25000} {
		v := int64(i + 1)
		out = append(out,
			rec(ms, 10, 1, map[int]StatValue{2: {A: v, B: 0}, 21: {A: v, B: 0}, 0: {A: v, B: 0}}),
			rec(ms+50, 12, 1, map[int]StatValue{3: {A: v, B: 0}, 20: {A: 0, B: v}}),
		)
	}
	// Emission NEGATIVE (ancrage parasite) et score de mode HORS DOMAINE (canal B aberrant).
	out = append(out,
		rec(7000, 10, 0, map[int]StatValue{2: {A: -115, B: 0}}),
		rec(7100, 10, 0, map[int]StatValue{0: {A: 66, B: 16635}}),
		// Manche 9 fortuite : sans contiguite, RealRounds ne la retient pas.
		rec(90000, 10, 9, map[int]StatValue{2: {A: 400, B: 0}}),
	)
	return out
}

func TestNamedEventsFromOnePassMatchesReference(t *testing.T) {
	recs := onePassCorpus()
	for _, mode := range []string{
		ObjectiveTypeFlag, ObjectiveTypeZone, ObjectiveTypeVip, ObjectiveTypeBomb, "koth",
	} {
		want := refNamedEventsFrom(recs, mode)
		got := NamedEventsFrom(recs, mode)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("mode %s : %d evenements, la reference en donne %d\n got=%v\nwant=%v",
				mode, len(got), len(want), got, want)
		}
	}
	// Le corpus doit reellement produire des evenements, sinon le test ne prouve rien.
	if n := len(NamedEventsFrom(recs, ObjectiveTypeFlag)); n == 0 {
		t.Fatal("corpus vide d'evenements : le differentiel serait vacant")
	}
}

// TestSortNamedEventsOrdreTotal verifie que la cle de tri distingue desormais deux
// evenements que seule leur PROVENANCE separe — ce que les trois cles d'avant ne faisaient
// pas (plan §9 R-8 : le determinisme n'etait qu'accidentel).
func TestSortNamedEventsOrdreTotal(t *testing.T) {
	evs := []NamedEvent{
		{TimeMS: 10, Slot: 12, Stat: StatKills, Comp: 12, Side: sideB},
		{TimeMS: 10, Slot: 12, Stat: StatKills, Comp: 2, Side: sideA},
		{TimeMS: 10, Slot: 12, Stat: StatKills, Comp: 2, Side: sideB},
	}
	sortNamedEvents(evs)
	want := []NamedEvent{
		{TimeMS: 10, Slot: 12, Stat: StatKills, Comp: 2, Side: sideA},
		{TimeMS: 10, Slot: 12, Stat: StatKills, Comp: 2, Side: sideB},
		{TimeMS: 10, Slot: 12, Stat: StatKills, Comp: 12, Side: sideB},
	}
	if !reflect.DeepEqual(evs, want) {
		t.Fatalf("tri = %v, attendu %v", evs, want)
	}
}
