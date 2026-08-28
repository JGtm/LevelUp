package replay

import (
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// skull_carries_test.go — LE PORTEUR DU CRANE, sur des enregistrements synthetiques (CI, sans
// film). On y prouve trois choses : les trains de tics deviennent des portages, un TROU de tics
// coupe un portage, et le porteur d'un train est nomme par l'identite de SA manche — donc le
// calque gere PLUSIEURS MANCHES sans melanger les porteurs.

// skullFixture fabrique un film Oddball a DEUX manches :
//   - manche 0 : slot 22 = "A" (deux portages, coupes par un trou), slot 20 = "C" ;
//   - manche 1 : slot 22 REATTRIBUE a "B".
//
// Le score de mode (`comp 0 A`) porte les tics ; le compteur de morts (`comp 2 B`) et le fil des
// morts nomment les slots par manche ; les prises (`comp 21 B`) alimentent la couverture.
func skullFixture() ([]objectiveevents.StatRecord, []objectiveevents.DeathInstant) {
	tick := func(t, slot, round int, v int64) objectiveevents.StatRecord {
		return objectiveevents.StatRecord{TimeMS: t, Slot: slot, Round: round,
			Comps: map[int]objectiveevents.StatValue{0: {A: v}}}
	}
	death := func(t, slot, round int, v int64) objectiveevents.StatRecord {
		return objectiveevents.StatRecord{TimeMS: t, Slot: slot, Round: round,
			Comps: map[int]objectiveevents.StatValue{2: {B: v}}}
	}
	grab := func(t, slot, round int, v int64) objectiveevents.StatRecord {
		return objectiveevents.StatRecord{TimeMS: t, Slot: slot, Round: round,
			Comps: map[int]objectiveevents.StatValue{21: {B: v}}}
	}
	recs := []objectiveevents.StatRecord{
		// Manche 0 — slot 22 (A) : deux portages separes par un trou (4000 -> 9000, > 3 s).
		tick(1000, 22, 0, 1), tick(2000, 22, 0, 2), tick(3000, 22, 0, 3), tick(4000, 22, 0, 4),
		tick(9000, 22, 0, 5), tick(10000, 22, 0, 6),
		// Manche 0 — slot 20 (C).
		tick(5000, 20, 0, 1), tick(6000, 20, 0, 2), tick(7000, 20, 0, 3),
		// Manche 1 — slot 22 REATTRIBUE (B).
		tick(20000, 22, 1, 1), tick(21000, 22, 1, 2), tick(22000, 22, 1, 3),
		// Compteur de morts (identite par manche).
		death(4500, 22, 0, 1), death(4600, 22, 0, 2), death(4700, 22, 0, 3),
		death(7500, 20, 0, 1), death(7600, 20, 0, 2), death(7700, 20, 0, 3),
		death(22500, 22, 1, 1), death(22600, 22, 1, 2), death(22700, 22, 1, 3),
		// Prises (couverture) : 2 + 1 en manche 0, 1 en manche 1 = 4.
		grab(1000, 22, 0, 1), grab(9000, 22, 0, 2), grab(5000, 20, 0, 1), grab(20000, 22, 1, 1),
	}
	sort.SliceStable(recs, func(i, j int) bool { return recs[i].TimeMS < recs[j].TimeMS })
	deaths := []objectiveevents.DeathInstant{
		{XUID: "A", TimeMS: 4500}, {XUID: "A", TimeMS: 4600}, {XUID: "A", TimeMS: 4700},
		{XUID: "C", TimeMS: 7500}, {XUID: "C", TimeMS: 7600}, {XUID: "C", TimeMS: 7700},
		{XUID: "B", TimeMS: 22500}, {XUID: "B", TimeMS: 22600}, {XUID: "B", TimeMS: 22700},
	}
	return recs, deaths
}

// skullTestScan construit le scan de production a partir de la fixture.
func skullTestScan(recs []objectiveevents.StatRecord, deaths []objectiveevents.DeathInstant) SkullCarryScan {
	return SkullCarryScan{
		Scanned:  true,
		Records:  recs,
		Identity: objectiveevents.ResolveRoundIdentity(recs, deaths),
	}
}

func TestSkullCarriesTwoRounds(t *testing.T) {
	recs, deaths := skullFixture()
	// step = 1000 us/frame => 1 frame par ms (frame = instant en ms). frames grand : tout ferme.
	carries, cov := buildSkullCarries(skullTestScan(recs, deaths),
		skullCarryCtx{origin: 0, step: 1000, frames: 100000})

	if cov == nil || !cov.SkullFilm {
		t.Fatalf("couverture absente ou SkullFilm faux : %+v", cov)
	}
	if cov.Grabs != 4 {
		t.Errorf("prises = %d, attendu 4", cov.Grabs)
	}
	if len(carries) != 4 {
		t.Fatalf("portages = %d, attendu 4 : %+v", len(carries), carries)
	}
	// Ordre TOTAL par instant : A(1000), C(5000), A(9000), B(20000).
	want := []struct {
		xuid   string
		t0, t1 int
	}{
		{"A", 1000, 4000}, {"C", 5000, 7000}, {"A", 9000, 10000}, {"B", 20000, 22000},
	}
	for i, w := range want {
		if carries[i].XUID != w.xuid || carries[i].T0 != w.t0 || carries[i].T1 != w.t1 {
			t.Errorf("portage %d = %+v, attendu {%s %d %d}", i, carries[i], w.xuid, w.t0, w.t1)
		}
	}
	// LE POINT DU LOT : en manche 1 le porteur est B (slot reattribue), pas A.
	if carries[3].XUID != "B" {
		t.Errorf("porteur manche 1 = %q, attendu \"B\" (le pont plat aurait dit A)", carries[3].XUID)
	}
	if cov.Trains != 4 || cov.Carries != 4 || cov.NoBridge != 0 || cov.OutOfWindow != 0 {
		t.Errorf("couverture = %+v, attendu 4 trains / 4 portages / 0 rejet", cov)
	}
	if !cov.Balanced() {
		t.Errorf("couverture desequilibree : %+v", cov)
	}
}

// TestSkullCarriesOpenAtAxisEnd — un portage dont le dernier tic bute sur la fin de l'axe n'est
// pas ferme (borne haute), les autres le sont.
func TestSkullCarriesOpenAtAxisEnd(t *testing.T) {
	recs, deaths := skullFixture()
	// frames = 23000 : le dernier portage (fin 22000) tombe dans le mou de fin (3 s) -> ouvert.
	carries, cov := buildSkullCarries(skullTestScan(recs, deaths),
		skullCarryCtx{origin: 0, step: 1000, frames: 23000})
	if len(carries) != 4 {
		t.Fatalf("portages = %d, attendu 4", len(carries))
	}
	if carries[3].Closed {
		t.Errorf("le portage de fin d'axe devrait etre OUVERT (borne haute)")
	}
	for i := 0; i < 3; i++ {
		if !carries[i].Closed {
			t.Errorf("portage %d devrait etre FERME (un fait le borne avant la fin)", i)
		}
	}
	if cov.Open != 1 || cov.Closed != 3 {
		t.Errorf("couverture ouverts/fermes = %d/%d, attendu 1/3", cov.Open, cov.Closed)
	}
	if !cov.Balanced() {
		t.Errorf("couverture desequilibree : %+v", cov)
	}
}

// TestSkullCarriesUnscanned — hors Oddball (Scanned faux), ni calque ni couverture.
func TestSkullCarriesUnscanned(t *testing.T) {
	carries, cov := buildSkullCarries(SkullCarryScan{Scanned: false}, skullCarryCtx{step: 1000, frames: 100})
	if carries != nil || cov != nil {
		t.Errorf("film non-Oddball : attendu (nil, nil), obtenu (%v, %v)", carries, cov)
	}
}
