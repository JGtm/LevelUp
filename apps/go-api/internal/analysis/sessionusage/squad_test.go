package sessionusage

// squad_test.go — la résolution des coéquipiers suivis : intersection sur TOUS
// les matchs (règle de l'accueil), restriction amis, plafond, ordre, FFA.

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

func TestResolveTrackedSquad_IntersectionSurTousLesMatchs(t *testing.T) {
	got := ResolveTrackedSquad("P", []string{"m1", "m2"}, squadParticipantsDeTest(), nil)
	// B n'est allié qu'au m1 : hors composition (intersection volontaire).
	// E1 est adverse au m1 : jamais un coéquipier suivi.
	if len(got) != 1 || got[0].XUID != "A" || got[0].Gamertag != "Alpha" {
		t.Fatalf("escouade = %+v, attendu [A/Alpha]", got)
	}
}

func TestResolveTrackedSquad_RestrictionAuxAmisConfigures(t *testing.T) {
	participants := squadParticipantsDeTest()
	if got := ResolveTrackedSquad("P", []string{"m1", "m2"}, participants, []string{"zulu"}); len(got) != 0 {
		t.Errorf("escouade = %+v, attendu vide (Alpha n'est pas un ami configuré)", got)
	}
	// Insensible à la casse (même convention que configuredFriendSet côté Home).
	if got := ResolveTrackedSquad("P", []string{"m1", "m2"}, participants, []string{" ALPHA "}); len(got) != 1 {
		t.Errorf("escouade = %+v, attendu [Alpha] (restriction insensible à la casse)", got)
	}
}

func TestResolveTrackedSquad_PlafondEtOrdreAlphabetique(t *testing.T) {
	var participants []ParticipantRow
	participants = append(participants,
		ParticipantRow{MatchID: "m1", XUID: "P", Gamertag: "Papa", TeamID: intp(0)})
	for _, gt := range []struct{ xuid, gt string }{
		{"D", "Delta"}, {"A", "Alpha"}, {"C", "Charlie"}, {"B", "Bravo"},
	} {
		participants = append(participants,
			ParticipantRow{MatchID: "m1", XUID: gt.xuid, Gamertag: gt.gt, TeamID: intp(0)})
	}
	got := ResolveTrackedSquad("P", []string{"m1"}, participants, nil)
	if len(got) != MaxTrackedSquadPlayers {
		t.Fatalf("escouade = %+v, attendu %d lignes (plafond)", got, MaxTrackedSquadPlayers)
	}
	if got[0].Gamertag != "Alpha" || got[1].Gamertag != "Bravo" || got[2].Gamertag != "Charlie" {
		t.Errorf("ordre = %+v, attendu alphabétique Alpha, Bravo, Charlie", got)
	}
}

func TestResolveTrackedSquad_FFAVideLaComposition(t *testing.T) {
	participants := squadParticipantsDeTest()
	// m3 : FFA — le joueur n'a pas de camp, donc aucun allié identifiable.
	participants = append(participants,
		ParticipantRow{MatchID: "m3", XUID: "P", Gamertag: "Papa"},
		ParticipantRow{MatchID: "m3", XUID: "A", Gamertag: "Alpha"},
	)
	if got := ResolveTrackedSquad("P", []string{"m1", "m2", "m3"}, participants, nil); len(got) != 0 {
		t.Errorf("escouade = %+v, attendu vide (un match FFA casse l'intersection)", got)
	}
}

// TestResolveScopeFriends_UnionSurLeScope — le résolveur du grain PÉRIODE retient
// un ami dès qu'il a été mon allié UNE fois (là où ResolveTrackedSquad exige
// toutes les fois), et classe par nombre de matchs partagés décroissant.
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

// TestResolveScopeFriends_SansAmiConfigureAucunAmi — l'INVERSE de la convention de
// ResolveTrackedSquad, et c'est délibéré : sur un scope de période, retenir les
// alliés les plus fréquents nommerait « mes amis » des inconnus.
func TestResolveScopeFriends_SansAmiConfigureAucunAmi(t *testing.T) {
	if got := ResolveScopeFriends("P", squadParticipantsDeTest(), nil); len(got) != 0 {
		t.Errorf("amis du scope = %+v, attendu vide sans ami configuré", got)
	}
}
