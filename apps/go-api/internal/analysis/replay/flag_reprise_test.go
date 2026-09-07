package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// flag_reprise_test.go — LA REPRISE DU MEME PORTEUR NE POSE PAS LE DRAPEAU AU SOL.
//
// Un portage ferme par la prise suivante du MEME slot n'a aucun lacher DATE : le modele borne
// lui-meme le sejour au sol a zero, puisque la fin et la reprise tombent sur la meme
// milliseconde. Emettre sa fin quand meme posait un `dropped` a la frame SUIVANTE de l'ouverture
// de la reprise (la fin est datee `frame(t1) + 1`, l'ouverture `frame(t0)`) : le portage repris
// se reduisait a UNE frame, et le drapeau se dessinait au sol pendant qu'un joueur courait avec.

// TestFlagRepriseNePubliePasDeLacher — LE PORTAGE REPRIS GARDE SA DUREE.
//
// Mesure d'origine sur `bcb6d393` : 10 portages sur 16 reduits a une frame, dont les 77 frames
// de celui qui finit par la capture — la duree PUBLIEE de son porteur tombait de 358 a 282.
func TestFlagRepriseNePubliePasDeLacher(t *testing.T) {
	tracks := []Track{flagTestTrack(12, "1", 0, 99, 2, 2)}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 3000, Slot: 12, Stat: objectiveevents.StatFlagGrabs},
			{TimeMS: 5000, Slot: 12, Stat: objectiveevents.StatFlagCaptures},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "1"}),
		Spawns:   []FlagSpawn{{Team: 1, X: 0, Y: 0}, {Team: 0, X: 100, Y: 100}},
	}

	got, cov := buildFlagCarries(scan, flagTestCtx(tracks, nil, 100))
	if cov.Carries != 2 || !cov.Balanced() {
		t.Fatalf("couverture %+v : 2 portages attendus, invariant tenu", *cov)
	}
	if cov.DropsWithheld != 1 {
		t.Errorf("dropsWithheld = %d, attendu 1 : la reprise doit se compter", cov.DropsWithheld)
	}
	f := flagOfTeam(t, got, 1)
	// AUCUN `dropped` : le drapeau passe d'une main a la MEME main, et les deux portages se
	// fondent en UN SEUL span continu (meme etat, meme porteur, meme position — `sameFlagPos`).
	assertFlagStates(t, f, []string{FlagStateHome, FlagStateCarried, FlagStateHome})
	if f.Spans[1].T0 != 10 || f.Spans[1].T1 != 50 {
		t.Errorf("portage continu sur [%d,%d], attendu [10,50] — le repris durait UNE frame avant",
			f.Spans[1].T0, f.Spans[1].T1)
	}
	if d := dureePortee(f, "1"); d != 41 {
		t.Errorf("duree portee par « 1 » = %d frames, attendu 41 (les deux portages bout a bout)", d)
	}
}

// TestFlagRepriseSurUnAutreDrapeauLacheBien — LA CONTRE-EPREUVE.
//
// La regle ne dit pas « une reprise n'a jamais de lacher » : elle dit que le drapeau ne touche
// pas le sol quand c'est LE MEME drapeau qui repart. Si le meme joueur reprend l'AUTRE drapeau,
// le premier est bel et bien lache, et son etat `dropped` se publie comme avant.
func TestFlagRepriseSurUnAutreDrapeauLacheBien(t *testing.T) {
	tracks := []Track{flagTestPath(12, "1", 0, 99, 30, 2, 2, 98, 98)}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 3000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: "1"}),
		Spawns:   []FlagSpawn{{Team: 1, X: 0, Y: 0}, {Team: 0, X: 100, Y: 100}},
	}

	got, cov := buildFlagCarries(scan, flagTestCtx(tracks, nil, 100))
	if cov.Carries != 2 || !cov.Balanced() {
		t.Fatalf("couverture %+v : 2 portages attendus, invariant tenu", *cov)
	}
	if cov.DropsWithheld != 0 {
		t.Errorf("dropsWithheld = %d, attendu 0 : la reprise porte sur l'AUTRE drapeau",
			cov.DropsWithheld)
	}
	assertFlagStates(t, flagOfTeam(t, got, 1),
		[]string{FlagStateHome, FlagStateCarried, FlagStateDropped})
	assertFlagStates(t, flagOfTeam(t, got, 0),
		[]string{FlagStateHome, FlagStateCarriedOpen})
}

// dureePortee somme les frames pendant lesquelles un joueur tient le drapeau.
func dureePortee(f FlagCarry, xuid string) int {
	n := 0
	for _, s := range f.Spans {
		if s.State != FlagStateCarried && s.State != FlagStateCarriedOpen {
			continue
		}
		if s.XUID == nil || *s.XUID != xuid {
			continue
		}
		n += s.T1 - s.T0 + 1
	}
	return n
}
