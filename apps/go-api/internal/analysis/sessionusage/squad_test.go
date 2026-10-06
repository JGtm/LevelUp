package sessionusage

// squad_test.go — les amis d'un scope de période : union sur le scope, ordre, et aucun ami sans
// liste configurée.

import "testing"

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

// TestResolveScopeFriends_UnionSurLeScope — le résolveur du grain PÉRIODE retient
// un ami dès qu'il a été mon allié UNE fois,
// et classe par nombre de matchs partagés décroissant.
func TestResolveScopeFriends_UnionSurLeScope(t *testing.T) {
	got := ResolveScopeFriends("P", squadParticipantsDeTest(), []string{"Alpha", "Bravo", "Echo"})
	// A est allié aux deux matchs, B au seul m1 : A d'abord.
	// E1 est adverse aux deux : jamais retenu, même déclaré ami.
	if len(got) != 2 {
		t.Fatalf("amis du scope = %+v, attendu 2 (Alpha puis Bravo)", got)
	}
	if got[0].XUID != "A" || got[1].XUID != "B" {
		t.Errorf("ordre = %q puis %q, attendu A puis B (matchs partagés décroissants)",
			got[0].XUID, got[1].XUID)
	}
}

// TestResolveScopeFriends_SansAmiConfigureAucunAmi — sur un scope de période, retenir les
// alliés les plus fréquents nommerait « mes amis » des inconnus.
func TestResolveScopeFriends_SansAmiConfigureAucunAmi(t *testing.T) {
	if got := ResolveScopeFriends("P", squadParticipantsDeTest(), nil); len(got) != 0 {
		t.Errorf("amis du scope = %+v, attendu vide sans ami configuré", got)
	}
}
