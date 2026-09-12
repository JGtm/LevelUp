package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// hhRecs fabrique des enregistrements de statistiques porteurs du seul composant 23, valeur A :
// `slot -> instant -> valeur cumulee`. Aucune I/O, aucun film.
func hhRecs(vals map[int]map[int]int) []objectiveevents.StatRecord {
	var out []objectiveevents.StatRecord
	for slot, byTime := range vals {
		for t, v := range byTime {
			out = append(out, objectiveevents.StatRecord{
				TimeMS: t, Slot: slot,
				Comps: map[int]objectiveevents.StatValue{23: {A: int64(v)}},
			})
		}
	}
	return out
}

var hhClock = scoreClock{intervalMS: 100, frames: 10000, originMS: 0}

// TestHoldTicksUnionCompteUneFoisLeMemeInstant — DEUX JOUEURS D'UN CAMP SUR LA COLLINE PRENNENT
// LE MEME TIC. La somme le compterait deux fois ; l'union ne le compte qu'une.
func TestHoldTicksUnionCompteUneFoisLeMemeInstant(t *testing.T) {
	recs := hhRecs(map[int]map[int]int{
		10: {1000: 1, 2000: 2, 3000: 3},
		12: {1000: 1, 2000: 2, 3000: 3},
	})
	identity := map[int]string{10: "a", 12: "b"}
	teams := map[string]int{"a": 0, "b": 0}

	got := buildHoldTicks(recs, identity, teams, hhClock)
	if len(got) != 1 {
		t.Fatalf("un seul camp attendu, got %d", len(got))
	}
	last := got[0].Ticks[len(got[0].Ticks)-1]
	if last.V != 3 {
		t.Errorf("union = %d, want 3 (la somme donnerait 6)", last.V)
	}
}

// TestHoldTicksUnionSuitLesRelais — LE MAXIMUM SUR LA PERIODE PERD LES RELAIS : deux joueurs qui
// se relaient sur la colline font avancer la barre du camp, chacun pour sa part.
func TestHoldTicksUnionSuitLesRelais(t *testing.T) {
	recs := hhRecs(map[int]map[int]int{
		10: {1000: 1, 2000: 2}, // le premier tient, puis s'arrete
		12: {3000: 1, 4000: 2}, // le second prend le relais
	})
	identity := map[int]string{10: "a", 12: "b"}
	teams := map[string]int{"a": 1, "b": 1}

	got := buildHoldTicks(recs, identity, teams, hhClock)
	if len(got) != 1 {
		t.Fatalf("un seul camp attendu, got %d", len(got))
	}
	last := got[0].Ticks[len(got[0].Ticks)-1]
	if last.V != 4 {
		t.Errorf("union = %d, want 4 (le maximum sur la periode donnerait 2)", last.V)
	}
}

// TestHoldTicksSepareLesCamps — chaque camp a sa barre, et le camp est celui du ROSTER.
func TestHoldTicksSepareLesCamps(t *testing.T) {
	recs := hhRecs(map[int]map[int]int{
		10: {1000: 1, 2000: 2, 3000: 3},
		12: {2000: 1},
	})
	identity := map[int]string{10: "a", 12: "b"}
	teams := map[string]int{"a": 0, "b": 1}

	got := buildHoldTicks(recs, identity, teams, hhClock)
	if len(got) != 2 {
		t.Fatalf("deux camps attendus, got %d", len(got))
	}
	if *got[0].TeamID != 0 || got[0].Ticks[len(got[0].Ticks)-1].V != 3 {
		t.Errorf("camp 0 : got %+v", got[0])
	}
	if *got[1].TeamID != 1 || got[1].Ticks[len(got[1].Ticks)-1].V != 1 {
		t.Errorf("camp 1 : got %+v", got[1])
	}
}

// TestHoldTicksEcarteLesSlotsNonSitues — un joueur dont le camp est inconnu ne fait avancer
// AUCUNE barre : on ne devine pas son camp.
func TestHoldTicksEcarteLesSlotsNonSitues(t *testing.T) {
	recs := hhRecs(map[int]map[int]int{10: {1000: 1, 2000: 5}})
	got := buildHoldTicks(recs, map[int]string{10: "inconnu"}, map[string]int{}, hhClock)
	if got != nil {
		t.Errorf("aucun camp situe : attendu nil, got %+v", got)
	}
}

// TestHoldTicksSansEmissionNePublieRien — un film qui ne porte pas ce composant ne produit
// aucune serie (et surtout pas une serie a zero, qui se lirait comme une mesure).
func TestHoldTicksSansEmissionNePublieRien(t *testing.T) {
	got := buildHoldTicks(nil, map[int]string{10: "a"}, map[string]int{"a": 0}, hhClock)
	if got != nil {
		t.Errorf("aucune emission : attendu nil, got %+v", got)
	}
}

// TestHoldTicksEstCumulativeEtCroissante — le client lit la serie en differentiel depuis le
// dernier point : elle doit etre cumulative et ne jamais reculer.
func TestHoldTicksEstCumulativeEtCroissante(t *testing.T) {
	recs := hhRecs(map[int]map[int]int{
		10: {1000: 1, 2000: 3, 5000: 7},
		12: {1500: 2, 3000: 4},
	})
	identity := map[int]string{10: "a", 12: "b"}
	teams := map[string]int{"a": 0, "b": 0}

	got := buildHoldTicks(recs, identity, teams, hhClock)
	if len(got) != 1 {
		t.Fatalf("un seul camp attendu, got %d", len(got))
	}
	prev := 0
	for _, tick := range got[0].Ticks {
		if tick.V < prev {
			t.Fatalf("serie decroissante : %+v", got[0].Ticks)
		}
		prev = tick.V
	}
}
