package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// TestFlagCarriesPriseSansPontComptee — une prise dont le slot statborg n'est pas nomme ne publie
// RIEN et se compte sous `NoBridge`. Se taire vaut mieux qu'attribuer le drapeau au hasard.
func TestFlagCarriesPriseSansPontComptee(t *testing.T) {
	tracks := []Track{flagTestTrack(10, "aaa", 0, 99, 30, 40)}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 2000, Slot: 20, Stat: objectiveevents.StatFlagSteals}, // slot non apparie
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "aaa"}),
		Spawns:   []FlagSpawn{{Team: 0}, {Team: 1, X: 100, Y: 100}},
	}
	_, cov := buildFlagCarries(scan, flagTestCtx(tracks, nil, 100))
	if cov.Openings != 2 || cov.NoBridge != 1 || cov.Carries != 1 {
		t.Errorf("couverture %+v : 2 prises, 1 sans pont, 1 publiee attendues", *cov)
	}
	if !cov.Balanced() {
		t.Errorf("invariant de couverture rompu : %+v", *cov)
	}
}

// TestFlagCarriesPorteurTueAmbigu — `flag_carriers_killed` est credite au TUEUR : quand deux
// portages sont ouverts, rien ne dit lequel tombe. Il ne doit alors fermer personne, et se
// compter.
func TestFlagCarriesPorteurTueAmbigu(t *testing.T) {
	tracks := []Track{
		flagTestTrack(10, "1", 0, 99, 10, 10),
		flagTestTrack(12, "2", 0, 99, 90, 90),
	}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 1000, Slot: 14, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 3000, Slot: 16, Stat: objectiveevents.StatFlagCarriersKilled},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "1", 14: "2", 16: "3"}),
		Spawns:   []FlagSpawn{{Team: 0}, {Team: 1, X: 100, Y: 100}},
	}
	got, cov := buildFlagCarries(scan, flagTestCtx(tracks, nil, 100))
	if cov.AmbiguousCarrierKills != 1 {
		t.Errorf("%d porteurs tues ambigus, attendu 1", cov.AmbiguousCarrierKills)
	}
	for _, f := range got {
		for _, s := range f.Spans {
			if flagStateCarrying(s.State) && s.T1 < 99 {
				t.Errorf("portage ferme a la frame %d par un evenement ambigu", s.T1)
			}
		}
	}
}

// TestFlagCarriesSimultaneiteFermeeComptee — la simultaneite se compte DEUX FOIS, et seul le
// second compte juge : trois portages OUVERTS a la fois s'expliquent par leur duree (rien ne les
// ferme), trois portages FERMES a la fois seraient une contradiction entre faits dates.
func TestFlagCarriesSimultaneiteFermeeComptee(t *testing.T) {
	tracks := []Track{
		flagTestTrack(10, "1", 0, 99, 10, 10),
		flagTestTrack(12, "2", 0, 99, 20, 20),
		flagTestTrack(14, "3", 0, 99, 30, 30),
	}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 1000, Slot: 14, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 1000, Slot: 16, Stat: objectiveevents.StatFlagSteals},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "1", 14: "2", 16: "3"}),
		Spawns:   []FlagSpawn{{Team: 0}, {Team: 1, X: 100, Y: 100}},
	}
	_, cov := buildFlagCarries(scan, flagTestCtx(tracks, nil, 100))
	if cov.Overlaps != 3 {
		t.Errorf("%d depassements de simultaneite, attendu 3 (trois portages a la fois)", cov.Overlaps)
	}
	if cov.ClosedOverlaps != 0 {
		t.Errorf("%d depassements ENTRE FERMES, attendu 0 : aucun de ces portages n'est ferme",
			cov.ClosedOverlaps)
	}
}

// TestFlagCarriesFilmNonCTF — un film que le discriminant n'a pas reconnu ne publie AUCUN
// drapeau, et sa couverture dit pourquoi.
func TestFlagCarriesFilmNonCTF(t *testing.T) {
	scan := FlagCarryScan{
		Scanned: true,
		Signals: objectiveevents.FlagFilmSignals{Bursts: 2, Captures: 6, Steals: 994, Grabs: 1470},
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "aaa"}),
	}
	got, cov := buildFlagCarries(scan, flagTestCtx(nil, nil, 100))
	if got != nil {
		t.Errorf("%d drapeaux publies sur un film non-CTF", len(got))
	}
	if cov == nil || cov.FlagFilm || cov.Openings != 0 {
		t.Errorf("couverture %+v : le film ne doit pas etre reconnu CTF", cov)
	}
}

// TestFlagCarriesSansBalayage — rien n'a ete lu : le calque est absent, pas vide.
func TestFlagCarriesSansBalayage(t *testing.T) {
	got, cov := buildFlagCarries(FlagCarryScan{}, flagTestCtx(nil, nil, 100))
	if got != nil || cov != nil {
		t.Errorf("calque %v / couverture %v : les deux doivent etre absents", got, cov)
	}
}
