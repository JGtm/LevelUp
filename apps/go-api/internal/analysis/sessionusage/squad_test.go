package sessionusage

// squad_test.go — les membres d'un scope de période : union sur le scope, ordre, appartenance par
// xuid (gamertags des participants vides compris), et aucun membre sans désignation.

import (
	"testing"

	"levelup/go-api/internal/domain"
)

func squadParticipantsDeTest() []ParticipantRow {
	return []ParticipantRow{
		{MatchID: "m1", XUID: "P", Gamertag: "Papa", TeamID: intp(0)},
		{MatchID: "m1", XUID: "A", Gamertag: "Alpha", TeamID: intp(0)},
		{MatchID: "m1", XUID: "B", Gamertag: "Bravo", TeamID: intp(0)},
		{MatchID: "m1", XUID: "E1", Gamertag: "Echo", TeamID: intp(1)},
		{MatchID: "m2", XUID: "P", Gamertag: "Papa", TeamID: intp(1)},
		{MatchID: "m2", XUID: "A", Gamertag: "Alpha", TeamID: intp(1)},
		{MatchID: "m2", XUID: "E1", Gamertag: "Echo", TeamID: intp(0)},
	}
}

func membres(paires ...string) []domain.SessionUsageSquadPlayer {
	var out []domain.SessionUsageSquadPlayer
	for i := 0; i+1 < len(paires); i += 2 {
		out = append(out, domain.SessionUsageSquadPlayer{XUID: paires[i], Gamertag: paires[i+1]})
	}
	return out
}

// TestResolveScopeMembers_UnionSurLeScope — le résolveur du grain PÉRIODE retient un membre dès
// qu'il a été mon allié UNE fois, et classe par nombre de matchs partagés décroissant.
func TestResolveScopeMembers_UnionSurLeScope(t *testing.T) {
	got := ResolveScopeMembers("P", squadParticipantsDeTest(), membres("A", "Alpha", "B", "Bravo", "E1", "Echo"))
	// A est allié aux deux matchs, B au seul m1 : A d'abord.
	// E1 est adverse aux deux : jamais retenu, même désigné.
	if len(got) != 2 {
		t.Fatalf("membres du scope = %+v, attendu 2 (Alpha puis Bravo)", got)
	}
	if got[0].XUID != "A" || got[1].XUID != "B" {
		t.Errorf("ordre = %q puis %q, attendu A puis B (matchs partagés décroissants)",
			got[0].XUID, got[1].XUID)
	}
}

// TestResolveScopeMembers_SansMembreDesigneAucunMembre — sur un scope de période, retenir les
// alliés les plus fréquents nommerait « l'escouade » des inconnus.
func TestResolveScopeMembers_SansMembreDesigneAucunMembre(t *testing.T) {
	if got := ResolveScopeMembers("P", squadParticipantsDeTest(), nil); len(got) != 0 {
		t.Errorf("membres du scope = %+v, attendu vide sans membre désigné", got)
	}
	if got := ResolveScopeMembers("P", squadParticipantsDeTest(), membres("", "Alpha")); len(got) != 0 {
		t.Errorf("membres du scope = %+v, attendu vide : un membre sans xuid n'est désigné que par son nom", got)
	}
}

// TestResolveScopeMembers_GamertagsVidesDesParticipants — la soirée du 7 octobre 2026 : aucune
// ligne de participant ne porte le gamertag des coéquipiers. L'appartenance se lit par xuid, le
// nom vient de la page ; le gamertag des participants ne sert qu'à défaut de nom donné.
func TestResolveScopeMembers_GamertagsVidesDesParticipants(t *testing.T) {
	participants := []ParticipantRow{
		{MatchID: "m1", XUID: "jgtm", Gamertag: "JGtm", TeamID: intp(0)},
		{MatchID: "m1", XUID: "choco", Gamertag: "", TeamID: intp(0)},
		{MatchID: "m1", XUID: "madina", Gamertag: "", TeamID: intp(0)},
		{MatchID: "m1", XUID: "inconnu", Gamertag: "", TeamID: intp(0)},
		{MatchID: "m2", XUID: "jgtm", Gamertag: "JGtm", TeamID: intp(1)},
		{MatchID: "m2", XUID: "choco", Gamertag: "", TeamID: intp(1)},
		{MatchID: "m2", XUID: "madina", Gamertag: "Madina97294", TeamID: intp(1)},
	}
	got := ResolveScopeMembers("jgtm", participants, membres("choco", "Chocoboflor", "madina", ""))
	want := []domain.SessionUsageSquadPlayer{
		{XUID: "choco", Gamertag: "Chocoboflor"},
		{XUID: "madina", Gamertag: "Madina97294"},
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("membres du scope = %+v, attendu %+v", got, want)
	}
}
