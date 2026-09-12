package replay

// flag_objects_drop_test.go — LE LACHER FERME AUSSI UN PORTAGE DEJA FERME (lot 6.7-B1, item 3).
//
// CE QUE CES TESTS FERMENT. `closeByFreeLives` ne regardait que les portages `!closed` — c'est-a-
// dire ceux que RIEN ne bornait, un seul span sur tout le parc mesure. Les portages fermes par
// une mort, une capture ou une reprise gardaient leur borne, alors qu'un lacher VOLONTAIRE
// survenu AVANT elle les rend trop longs. Mesure de l'audit du 2026-09-10 sur les 11 films CTF a
// calque : +1 129,3 s publiees en trop, 69 joueurs sur 95 au-dessus de leur propre oracle, et
// `coverage.flagCarries.closedByObject` a ZERO sur les onze films alors que le canal des vies
// d'objet est lu et compte (`objectLives = 37` sur `16ea3668`).
//
// LA SOUS-POPULATION NE BOUGE PAS : seules comptent les vies nees AUX PIEDS d'un porteur et HORS
// SOCLE — celle que le controle 3 valide (cf. l'en-tete de `flag_objects.go`). Les deux temoins
// negatifs de `flag_objects_test.go` (loin du porteur, ne a un socle) valent donc aussi ici, et
// le troisieme est ci-dessous : une vie nee APRES la fermeture ne raccourcit rien.

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// flagTestClosedScan monte un portage FERME PAR UN FAIT : une prise a 1 000 ms, puis la mort du
// porteur a 6 000 ms (frame 60).
func flagTestClosedScan(free []flagFreeLife) FlagCarryScan {
	return FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "1"}),
		Spawns:   flagTestSpawns(),
		Free:     free,
	}
}

// TestUneVieLibreFermeUnPortageDejaFermeParLaMort — LE POINT DU LOT. Le porteur meurt a la
// frame 60, mais il avait DEJA lache le drapeau a la frame 20 : c'est la frame 20 qui borne.
//
// MUTATION : retablir `if raws[i].closed { continue }` dans [closeByFreeLives] rougit ce test
// (le portage se rouvre jusqu'a la frame 60) et lui seul — c'est exactement l'etat d'avant ce lot.
func TestUneVieLibreFermeUnPortageDejaFermeParLaMort(t *testing.T) {
	tracks := []Track{flagTestTrack(10, "1", 0, 99, 50, 50)}
	ctx := flagTestCtx(tracks, []Death{{XUID: 1, TimeMS: 6000}}, 100)

	// TEMOIN — sans vie libre, la mort borne le portage a la frame 60.
	got, cov := buildFlagCarries(flagTestClosedScan(nil), ctx)
	f := flagOfTeam(t, got, 0)
	assertFlagStates(t, f, []string{FlagStateHome, FlagStateCarried, FlagStateDropped})
	if f.Spans[1].T1 != 60 || cov.ClosedByObject != 0 {
		t.Fatalf("temoin : portage ferme a %d, closedByObject %d — attendu 60 et 0",
			f.Spans[1].T1, cov.ClosedByObject)
	}

	// LA REGLE — l'objet reapparait a la frame 20, a 0,7 m du porteur : le lacher est la.
	free := []flagFreeLife{flagTestLife(20, [2]float32{50.5, 50.5}, [2]float32{51, 51})}
	got, cov = buildFlagCarries(flagTestClosedScan(free), ctx)
	f = flagOfTeam(t, got, 0)
	assertFlagStates(t, f, []string{FlagStateHome, FlagStateCarried, FlagStateDropped})
	if f.Spans[1].T1 != 20 {
		t.Errorf("portage ferme a la frame %d, attendu 20 (le lacher, plus petit que la mort)",
			f.Spans[1].T1)
	}
	if cov.ClosedByObject != 1 || cov.Closed != 1 || cov.Open != 0 || !cov.Balanced() {
		t.Errorf("couverture %+v, attendu 1 ferme par l'objet et l'invariant tenu", *cov)
	}
}

// TestUneVieLibreApresLaFermetureNeChangeRien — LE PLUS PETIT FERMOIR GAGNE, et une vie libre
// posterieure a la fermeture n'en est pas un. Sans ce refus, la regle allongerait des portages.
func TestUneVieLibreApresLaFermetureNeChangeRien(t *testing.T) {
	tracks := []Track{flagTestTrack(10, "1", 0, 99, 50, 50)}
	ctx := flagTestCtx(tracks, []Death{{XUID: 1, TimeMS: 6000}}, 100)
	free := []flagFreeLife{flagTestLife(80, [2]float32{50.5, 50.5})}
	got, cov := buildFlagCarries(flagTestClosedScan(free), ctx)
	f := flagOfTeam(t, got, 0)
	if f.Spans[1].T1 != 60 {
		t.Errorf("portage ferme a %d, attendu 60 : une vie libre NEE APRES la fermeture "+
			"n'allonge ni ne raccourcit rien", f.Spans[1].T1)
	}
	if cov.ClosedByObject != 0 || !cov.Balanced() {
		t.Errorf("couverture %+v, attendu 0 ferme par l'objet", *cov)
	}
}

// TestUneVieLibreDementLaCapture — un portage ferme par une CAPTURE dont l'objet reapparait aux
// pieds du porteur AVANT elle n'est plus une capture : le drapeau etait au sol.
func TestUneVieLibreDementLaCapture(t *testing.T) {
	tracks := []Track{flagTestTrack(10, "1", 0, 99, 50, 50)}
	ctx := flagTestCtx(tracks, nil, 100)
	scan := flagTestClosedScan(nil)
	scan.Events = append(scan.Events, objectiveevents.NamedEvent{
		TimeMS: 6000, Slot: 12, Stat: objectiveevents.StatFlagCaptures})

	// TEMOIN — sans vie libre, la capture borne le portage et l'etat suivant est `home`.
	got, cov := buildFlagCarries(scan, ctx)
	f := flagOfTeam(t, got, 0)
	assertFlagStates(t, f, []string{FlagStateHome, FlagStateCarried, FlagStateHome})
	if f.Spans[1].T1 != 60 || cov.ClosedByObject != 0 {
		t.Fatalf("temoin : portage ferme a %d, closedByObject %d — attendu 60 et 0",
			f.Spans[1].T1, cov.ClosedByObject)
	}

	// LA REGLE — l'objet est au sol a la frame 20 : ce porteur-la n'a pas capture.
	scan.Free = []flagFreeLife{flagTestLife(20, [2]float32{50.5, 50.5}, [2]float32{51, 51})}
	got, cov = buildFlagCarries(scan, ctx)
	f = flagOfTeam(t, got, 0)
	assertFlagStates(t, f, []string{FlagStateHome, FlagStateCarried, FlagStateDropped})
	if f.Spans[1].T1 != 20 || cov.ClosedByObject != 1 || !cov.Balanced() {
		t.Errorf("portage ferme a %d, couverture %+v — attendu 20 et 1 ferme par l'objet",
			f.Spans[1].T1, *cov)
	}
}
