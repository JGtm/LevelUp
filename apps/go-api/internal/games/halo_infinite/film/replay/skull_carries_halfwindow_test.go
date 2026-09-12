package replay

// skull_carries_halfwindow_test.go — LA DEMI-FENETRE DE TIC, AUX DEUX BORNES (lot 6.7-B1,
// item 2).
//
// Ce que ces tests ferment : un train de tics etait borne par son PREMIER et son DERNIER tic,
// donc un train de n tics ne publiait que (n-1) largeurs de tic — la seconde d'amorce et celle
// de chute manquaient (rapport 6.2, decouverte 4 ; audit du 2026-09-10 §3.4). La largeur du tic
// n'est PAS une constante ecrite ici : elle se MESURE sur le film, et le dernier test le prouve
// en retirant la matiere de la mesure.

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// La fixture partagee ([skullFixture]) cadence ses tics a 1 000 ms ; l'axe des tests vaut
// 1 000 us par image, donc une largeur de tic = 1 000 images et la demi-fenetre = 499 images
// (cf. [skullHalfTickFrames] : une image est deja rendue par la borne fermee de chaque cote).
const (
	testTickWidthFrames = 1000
	testHalfTickFrames  = (testTickWidthFrames - 1) / 2
)

// TestSkullTickWidthMesureeSurLeFilm — la largeur du tic est LUE dans les trains du film.
func TestSkullTickWidthMesureeSurLeFilm(t *testing.T) {
	recs, _ := skullFixture()
	ctx := matchClock{origin: 0, step: 1000, frames: 100000}
	if got := skullTickWidthFrames(recs, ctx); got != testTickWidthFrames {
		t.Errorf("largeur de tic = %d images, attendu %d", got, testTickWidthFrames)
	}
	if got := skullHalfTickFrames(recs, ctx); got != testHalfTickFrames {
		t.Errorf("demi-fenetre = %d images, attendu %d", got, testHalfTickFrames)
	}
}

// TestSkullCarriesDemiFenetreAuxDeuxBornes — chaque periode gagne la demi-fenetre des DEUX
// cotes, et un train de n tics publie donc n largeurs de tic moins une image.
func TestSkullCarriesDemiFenetreAuxDeuxBornes(t *testing.T) {
	recs, deaths := skullFixture()
	carries, _ := buildSkullCarries(skullTestScan(recs, deaths),
		matchClock{origin: 0, step: 1000, frames: 100000}, carrierPresence{})
	if len(carries) != 4 {
		t.Fatalf("portages = %d, attendu 4", len(carries))
	}
	// Les quatre trains bruts de la fixture, en images (= ms ici), AVANT demi-fenetre.
	bruts := []struct {
		xuid   string
		t0, t1 int
		tics   int
	}{
		{"A", 1000, 4000, 4}, {"C", 5000, 7000, 3}, {"A", 9000, 10000, 2}, {"B", 20000, 22000, 3},
	}
	for i, b := range bruts {
		wantT0, wantT1 := b.t0-testHalfTickFrames, b.t1+testHalfTickFrames
		if carries[i].XUID != b.xuid || carries[i].T0 != wantT0 || carries[i].T1 != wantT1 {
			t.Errorf("portage %d = %+v, attendu {%s %d %d}", i, carries[i], b.xuid, wantT0, wantT1)
		}
		// n tics = n largeurs de tic, a une image pres (la borne fermee en rend une de trop
		// si on ne la retranche pas — c'est pourquoi la demi-fenetre vaut (w-1)/2).
		if duree := carries[i].T1 - carries[i].T0 + 1; duree > b.tics*testTickWidthFrames {
			t.Errorf("portage %d dure %d images, plus que %d tics x %d",
				i, duree, b.tics, testTickWidthFrames)
		}
	}
}

// TestSkullCarriesDemiFenetreBorneeParLAxe — la demi-fenetre ne sort jamais de l'axe publie.
func TestSkullCarriesDemiFenetreBorneeParLAxe(t *testing.T) {
	recs, deaths := skullFixture()
	// frames = 22200 : le dernier train (22000 + 499) deborderait de l'axe.
	carries, _ := buildSkullCarries(skullTestScan(recs, deaths),
		matchClock{origin: 0, step: 1000, frames: 22200}, carrierPresence{})
	if len(carries) != 4 {
		t.Fatalf("portages = %d, attendu 4", len(carries))
	}
	if carries[0].T0 != 1000-testHalfTickFrames {
		t.Errorf("borne basse = %d, attendu %d", carries[0].T0, 1000-testHalfTickFrames)
	}
	if carries[3].T1 != 22199 {
		t.Errorf("borne haute = %d, attendu 22199 (derniere image de l'axe)", carries[3].T1)
	}
}

// TestSkullCarriesSansCadenceMesurableNeDeplaceRien — LA MUTATION. Un film dont aucun train ne
// porte deux tics ne donne AUCUNE cadence a mesurer : les bornes restent celles des tics. Si la
// largeur etait une constante ecrite dans le code, ce test rougirait.
func TestSkullCarriesSansCadenceMesurableNeDeplaceRien(t *testing.T) {
	tick := func(tms, slot, round int, v int64) objectiveevents.StatRecord {
		return objectiveevents.StatRecord{TimeMS: tms, Slot: slot, Round: round,
			Comps: map[int]objectiveevents.StatValue{0: {A: v}}}
	}
	death := func(tms, slot, round int, v int64) objectiveevents.StatRecord {
		return objectiveevents.StatRecord{TimeMS: tms, Slot: slot, Round: round,
			Comps: map[int]objectiveevents.StatValue{2: {B: v}}}
	}
	// Trois tics du meme slot, tous separes de plus de `skullTickGapMS` : trois trains d'UN tic,
	// donc aucun ecart intra-train a mesurer.
	recs := []objectiveevents.StatRecord{
		tick(1000, 22, 0, 1), tick(10000, 22, 0, 2), tick(20000, 22, 0, 3),
		death(30000, 22, 0, 1), death(30100, 22, 0, 2), death(30200, 22, 0, 3),
	}
	deaths := []objectiveevents.DeathInstant{
		{XUID: "A", TimeMS: 30000}, {XUID: "A", TimeMS: 30100}, {XUID: "A", TimeMS: 30200},
	}
	ctx := matchClock{origin: 0, step: 1000, frames: 100000}
	if got := skullHalfTickFrames(recs, ctx); got != 0 {
		t.Fatalf("demi-fenetre = %d, attendu 0 (aucune cadence mesurable)", got)
	}
	carries, _ := buildSkullCarries(skullTestScan(recs, deaths), ctx, carrierPresence{})
	if len(carries) != 3 {
		t.Fatalf("portages = %d, attendu 3", len(carries))
	}
	for i, want := range []int{1000, 10000, 20000} {
		if carries[i].T0 != want || carries[i].T1 != want {
			t.Errorf("portage %d = [%d,%d], attendu [%d,%d] (bornes des tics)",
				i, carries[i].T0, carries[i].T1, want, want)
		}
	}
}
