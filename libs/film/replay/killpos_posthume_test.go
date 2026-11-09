package replay

// killpos_posthume_test.go — LE TUEUR DEJA MORT AU MOMENT DU FRAG (D-17, killpos_posthume.go).
//
// Le scenario de base : le tueur (111, siege 1) meurt a 9 s ; sa grenade tue la victime (222,
// siege 2) a 10 s. A 10 s, le siege 1 n'a plus d'echantillon : le placement direct rend nil.

import (
	"testing"

	"github.com/JGtm/LevelUp/libs/film/internal/facts/fallback"
	"github.com/JGtm/LevelUp/libs/film/internal/grammar"
)

func posthumeScenario() ([]grammar.BipedPosition, []KillRef, []MortDuJournal) {
	pos := []grammar.BipedPosition{
		posAt(1, 8_900_000, 7, 8, 0), posAt(1, 9_000_000, 7, 8, 0), // le tueur, jusqu a sa mort
		posAt(2, 10_000_000, 4, 5, 0), // la victime, au frag
	}
	kills := []KillRef{{KillerXUID: 111, VictimXUID: 222, TimeMS: 10_000}}
	morts := []MortDuJournal{{VictimeXUID: 111, TempsMS: 9_000}, {VictimeXUID: 222, TempsMS: 10_000}}
	return pos, kills, morts
}

// TestKillPositions_TueurPosthumeAuLieuDeSaMort : sans la regle, le tueur restait nil ; avec elle,
// il est pose ou il est mort, et le repli se compte.
func TestKillPositions_TueurPosthumeAuLieuDeSaMort(t *testing.T) {
	pos, kills, morts := posthumeScenario()
	reg := regPlat(map[uint32]uint64{1: 111, 2: 222})

	sans, _ := BuildKillPositions(EntreeDesPositions{Positions: pos, Registre: reg, Kills: kills})
	if len(sans) != 1 || sans[0].Killer != nil {
		t.Fatalf("sans les morts du journal, le tueur reste nil : %+v", sans)
	}

	fb := fallback.NouveauCompteur()
	got, rep := BuildKillPositions(EntreeDesPositions{
		Positions: pos, Registre: reg, Kills: kills, Morts: morts, Fallbacks: fb,
	})
	if len(got) != 1 || got[0].Killer == nil || got[0].Victim == nil {
		t.Fatalf("le tueur posthume devait etre pose : %+v", got)
	}
	if got[0].Killer.X != 7 || got[0].Killer.Y != 8 {
		t.Errorf("tueur pose en %+v, attendu (7, 8) : le lieu de sa mort", *got[0].Killer)
	}
	if rep.Posthumous != 1 || rep.PosthumousUnplaced != 0 || rep.Both != 1 {
		t.Errorf("rapport %+v, attendu 1 posthume, 2 cotes", rep)
	}
	if n := fb.Compte(fallback.NomPositionPosthumeDuTueur); n != 1 {
		t.Errorf("repli compte %d fois, attendu 1", n)
	}
}

// TestKillPositions_TueurReapparuNEstPasPosthume : une vie NOMMEE du tueur s'ouvre entre sa mort et
// le frag — il est vivant ailleurs, la regle ne s'applique pas.
func TestKillPositions_TueurReapparuNEstPasPosthume(t *testing.T) {
	pos, kills, morts := posthumeScenario()
	pos = append(pos, posAt(3, 9_800_000, 50, 50, 0))
	vies := []lifeSpan{
		{slot: 1, from: 8_900_000, to: 9_000_000, xuid: 111},
		{slot: 3, from: 9_800_000, to: 9_800_000, xuid: 111},
		{slot: 2, from: 10_000_000, to: 10_000_000, xuid: 222},
	}
	reg := regDeTest(vies, map[uint32]uint64{1: 111, 2: 222, 3: 111}, nil)
	got, rep := BuildKillPositions(EntreeDesPositions{Positions: pos, Registre: reg, Kills: kills, Morts: morts})
	if len(got) != 1 || got[0].Killer != nil {
		t.Fatalf("le tueur a reapparu : il ne se pose pas au lieu de sa mort, %+v", got)
	}
	if rep.Posthumous != 0 || rep.PosthumousUnplaced != 0 {
		t.Errorf("rapport %+v : aucun posthume attendu", rep)
	}
}

// TestKillPositions_MortALaMemeMillisecondeNEstPasPosthume : un echange (les deux meurent a la meme
// milliseconde) n'est pas un frag posthume — le placement direct en a decide.
func TestKillPositions_MortALaMemeMillisecondeNEstPasPosthume(t *testing.T) {
	pos, kills, _ := posthumeScenario()
	morts := []MortDuJournal{{VictimeXUID: 111, TempsMS: 10_000}, {VictimeXUID: 222, TempsMS: 10_000}}
	reg := regPlat(map[uint32]uint64{1: 111, 2: 222})
	_, rep := BuildKillPositions(EntreeDesPositions{Positions: pos, Registre: reg, Kills: kills, Morts: morts})
	if rep.Posthumous != 0 || rep.PosthumousUnplaced != 0 {
		t.Errorf("rapport %+v : une mort au meme instant n est pas posthume", rep)
	}
}

// TestKillPositions_TueurPosthumeSansLieuResteNil : le tueur est mort, mais sa mort n'est pas
// localisable (aucun echantillon a ± 120 ms) — il reste nil, et le cas se compte a part.
func TestKillPositions_TueurPosthumeSansLieuResteNil(t *testing.T) {
	pos := []grammar.BipedPosition{posAt(1, 5_000_000, 7, 8, 0), posAt(2, 10_000_000, 4, 5, 0)}
	kills := []KillRef{{KillerXUID: 111, VictimXUID: 222, TimeMS: 10_000}}
	morts := []MortDuJournal{{VictimeXUID: 111, TempsMS: 9_000}}
	fb := fallback.NouveauCompteur()
	got, rep := BuildKillPositions(EntreeDesPositions{
		Positions: pos, Registre: regPlat(map[uint32]uint64{1: 111, 2: 222}), Kills: kills, Morts: morts, Fallbacks: fb,
	})
	if len(got) != 1 || got[0].Killer != nil {
		t.Fatalf("mort non localisable : le tueur reste nil, %+v", got)
	}
	if rep.Posthumous != 0 || rep.PosthumousUnplaced != 1 {
		t.Errorf("rapport %+v, attendu 1 posthume sans lieu", rep)
	}
	if n := fb.Compte(fallback.NomPositionPosthumeDuTueur); n != 0 {
		t.Errorf("repli compte %d fois : rien n a ete pose", n)
	}
}

// TestKillOpenings_IgnorentLaRegleDuTueurPosthume : l'entame d'un tueur mort n'a pas de sens — la
// fonction de l'entame ne recoit pas les morts du journal et n'en compte aucune.
func TestKillOpenings_IgnorentLaRegleDuTueurPosthume(t *testing.T) {
	pos, kills, _ := posthumeScenario()
	_, rep := BuildKillOpenings(pos, regPlat(map[uint32]uint64{1: 111, 2: 222}), kills, 0)
	if rep.Posthumous != 0 || rep.PosthumousUnplaced != 0 {
		t.Errorf("rapport d entame %+v : aucun posthume attendu", rep)
	}
}
