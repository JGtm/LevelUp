package replay

// player_teams_index_relaye_test.go — LE PONT SLOT -> INDEX SE TAIT SUR UN INDEX QU'UN BOT D'UNE
// AUTRE EQUIPE TIENT (E8.4, decision D-18 du plan du 2026-10-16).
//
// Gabarit de `3f048bf3` : l'index 8 porte l'equipe 1 dans la table par index (la seule entite `ti=9`
// lue est celle de « 343 Ritzy ») ; « 343 Lacuna », declare sur le meme index avec l'equipe 0, n'a
// aucune entite. Le corps sans nom d'index 8 recevait l'equipe 1 par le pont : celle d'un autre
// occupant.
//
// MUTATION : retirer `if p.indexRelayes[idx]` de `equipeDuSlot` rend l'equipe 1 a la vie et fait
// tomber le compte d'abstention a zero.

import (
	"testing"

	"github.com/JGtm/LevelUp/libs/film/internal/grammar"
)

func registreDUnCorpsDIndex8() IdentityRegistry {
	return IdentityRegistry{own: OwnerReport{
		lives: []lifeSpan{{slot: 540, from: 0, to: 500_000}, {slot: 541, from: 0, to: 500_000}},
		Owner: map[uint32]int{540: 8, 541: 9},
	}}
}

func TestEquipeDuSlotSeTaitSurUnIndexRelaye(t *testing.T) {
	p := newTeamPublication(registreDUnCorpsDIndex8(), map[int]int{8: 1, 9: 1},
		grammar.TeamScanReport{Component: "team_designator", Records: 2}, nil)
	zero, un := 0, 1
	p.retenirLesIndexRelayes([]BotIdentity{
		{FilmIndex: 8, Name: "343 Ritzy [bot]", BotID: 0, BotIDLu: true, Team: &un},
		{FilmIndex: 8, Name: "343 Lacuna [bot]", BotID: 17, BotIDLu: true, Team: &zero},
		{FilmIndex: 9, Name: "343 Seul [bot]", BotID: 3, BotIDLu: true, Team: &un},
	})
	tracks := []Track{{Slot: 540, Team: -1}, {Slot: 541, Team: -1}}
	total, nommees, abstenues := p.poserSurLesTraces(tracks)
	if total != 2 || nommees != 1 || abstenues != 1 {
		t.Fatalf("comptes %d / %d / %d, attendu 2 / 1 / 1", total, nommees, abstenues)
	}
	if tracks[0].Team != -1 {
		t.Errorf("corps d'index 8 : equipe %d, attendu -1 — l'equipe de l'index est celle d'un autre occupant",
			tracks[0].Team)
	}
	if tracks[1].Team != 1 {
		t.Errorf("corps d'index 9 : equipe %d, attendu 1 — un index sans occupant d'une autre equipe garde le pont",
			tracks[1].Team)
	}
}

// Un bot dont l'equipe n'est pas lue (Team nil) ne contredit rien.
func TestIndexRelayeIgnoreLEquipeNonLue(t *testing.T) {
	p := newTeamPublication(registreDUnCorpsDIndex8(), map[int]int{8: 1},
		grammar.TeamScanReport{Component: "team_designator", Records: 1}, nil)
	p.retenirLesIndexRelayes([]BotIdentity{{FilmIndex: 8, Name: "343 Lacuna [bot]", BotID: 17, BotIDLu: true}})
	if eq, lue, ambigu := p.equipeDuSlot(540); !lue || ambigu || eq != 1 {
		t.Fatalf("equipe=%d lue=%v ambigu=%v, attendu 1 lue", eq, lue, ambigu)
	}
}
