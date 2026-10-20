package replay

// flag_carries_lives_order_test.go — L ORDRE DES TRANSITIONS D UN DRAPEAU EST TOTAL : l instant,
// et le lacher avant la reprise (constat RB1-2 de l audit du 2026-09-24, lot J9.4).
//
// LE DEFAUT : la fin d un portage se pose a la frame SUIVANT celle de son instant (la frame de la
// fin reste portee — « fins arrondies »), l ouverture a la frame de son instant, et les transitions
// d un drapeau etaient ensuite retriees par FRAME SEULE. Une reprise tombee dans la MEME frame que
// le lacher, quelques millisecondes apres lui, passait donc AVANT lui : le portage repris se
// reduisait a une frame, puis le drapeau etait publie AU SOL jusqu a la fin pendant qu un joueur
// courait avec.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// TestRepriseDansLaFrameDuLacherPasseApresLui — LE POINT DU LOT. « 1 » vole le drapeau de
// l equipe 1 a 1 000 ms et meurt a 3 010 ms (frame 30) ; son coequipier « 2 » le ramasse a
// 3 050 ms, dans la MEME frame, et rien ne ferme son portage. Le drapeau doit etre porte sans
// interruption jusqu a la fin de l axe.
//
// MUTATION : retrier les transitions par frame seule dans `spansOfTransitions` (l ancien
// `sort.SliceStable` sur `frame`) rougit ce test.
func TestRepriseDansLaFrameDuLacherPasseApresLui(t *testing.T) {
	tracks := []Track{
		flagTestTrack(10, "1", 0, 99, 2, 2),
		flagTestTrack(11, "2", 0, 99, 3, 3),
	}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectives.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectives.StatFlagSteals},
			{TimeMS: 3050, Slot: 14, Stat: objectives.StatFlagGrabs},
		},
		Identity: objectives.FlatRoundIdentity(map[int]string{12: "1", 14: "2"}),
		Spawns:   flagInvariantSpawns(),
		TeamOf:   map[string]int{"1": 0, "2": 0},
	}
	got, cov := buildFlagCarries(scan, flagTestCtx(tracks, []types.Death{{XUID: 1, TimeMS: 3010}}, 100))
	if cov == nil || cov.Carries != 2 {
		t.Fatalf("couverture %+v : deux portages publies attendus", cov)
	}
	f := flagOfTeam(t, got, 1)
	assertFlagStates(t, f, []string{FlagStateHome, FlagStateCarried, FlagStateCarriedOpen})
	if last := f.Spans[len(f.Spans)-1]; last.T1 != 99 || last.XUID == nil || *last.XUID != "2" {
		t.Errorf("dernier span %+v : attendu le portage de « 2 » jusqu a la frame 99", last)
	}
}

// TestUnRetourDansLaFrameDuLacherPasseApresLui — la MEME faute sur un retour credite : le retour
// tombe dans la frame du lacher, quelques millisecondes apres lui. Le drapeau doit rentrer chez lui
// et y rester, pas finir au sol.
func TestUnRetourDansLaFrameDuLacherPasseApresLui(t *testing.T) {
	tracks := []Track{flagTestTrack(10, "1", 0, 99, 2, 2)}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectives.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectives.StatFlagSteals},
			{TimeMS: 3050, Slot: 16, Stat: objectives.StatFlagReturns},
		},
		Identity: objectives.FlatRoundIdentity(map[int]string{12: "1", 16: "3"}),
		Spawns:   flagInvariantSpawns(),
		TeamOf:   map[string]int{"1": 0, "3": 1},
	}
	got, _ := buildFlagCarries(scan, flagTestCtx(tracks, []types.Death{{XUID: 1, TimeMS: 3010}}, 100))
	f := flagOfTeam(t, got, 1)
	if last := f.Spans[len(f.Spans)-1]; last.State != FlagStateHome || last.T1 != 99 {
		t.Errorf("dernier etat %q jusqu a la frame %d : attendu `home` jusqu a 99 (spans %+v)",
			last.State, last.T1, f.Spans)
	}
}
