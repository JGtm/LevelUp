package sessionusage

// team_context_test.go — le contexte de camp d'un scope et l'assemblage des matchs : camp du
// joueur, effectifs présents à la fin, FFA sans camp ni effectif de camp, match mesuré = ligne film,
// ordre du scope gardé.

import "testing"

func intp(v int) *int { return &v }

func participantsDeContexte() []ParticipantRow {
	return []ParticipantRow{
		{MatchID: "m1", XUID: "P", TeamID: intp(0), PresentAtCompletion: true},
		{MatchID: "m1", XUID: "A", TeamID: intp(0), PresentAtCompletion: true},
		{MatchID: "m1", XUID: "Q", TeamID: intp(0), PresentAtCompletion: false}, // parti avant la fin
		{MatchID: "m1", XUID: "E1", TeamID: intp(1), PresentAtCompletion: true},
		{MatchID: "ffa", XUID: "P", PresentAtCompletion: true},
		{MatchID: "ffa", XUID: "E1", PresentAtCompletion: true},
	}
}

func TestBuildTeamContext_CampEtEffectifsPresentsALaFin(t *testing.T) {
	tc := BuildTeamContext("P", participantsDeContexte())
	if team, ok := tc.PlayerTeam["m1"]; !ok || team != 0 {
		t.Errorf("camp du joueur sur m1 = %d (%v), attendu 0", team, ok)
	}
	if tc.TeamSize["m1"] != 2 || tc.LobbySize["m1"] != 3 {
		t.Errorf("effectifs m1 = camp %d, lobby %d ; attendu 2 et 3 (Q est parti avant la fin)",
			tc.TeamSize["m1"], tc.LobbySize["m1"])
	}
	if tc.TeamOf["m1"]["Q"] != 0 || len(tc.TeamOf["m1"]) != 4 {
		t.Errorf("camps de m1 = %v, attendu les quatre participants, présents ou non", tc.TeamOf["m1"])
	}
	if _, ok := tc.PlayerTeam["ffa"]; ok || tc.TeamSize["ffa"] != 0 || tc.LobbySize["ffa"] != 2 {
		t.Errorf("FFA : camp %v, effectif de camp %d, lobby %d ; attendu aucun camp, 0, 2",
			ok, tc.TeamSize["ffa"], tc.LobbySize["ffa"])
	}
}

func TestBuildMatchInputs_OrdreMesureEtContexte(t *testing.T) {
	tc := BuildTeamContext("P", participantsDeContexte())
	films := map[string]FilmRow{"m1": {MatchID: "m1"}}
	players := []PlayerRow{{MatchID: "m1", XUID: "P"}, {MatchID: "m1", XUID: "E1"}, {MatchID: "ffa", XUID: "P"}}
	got := BuildMatchInputs([]string{"ffa", "m1"}, films, players, tc)
	if len(got) != 2 || got[0].MatchID != "ffa" || got[1].MatchID != "m1" {
		t.Fatalf("matchs = %+v, attendu ffa puis m1 (ordre du scope)", got)
	}
	if got[0].Measured || !got[1].Measured {
		t.Errorf("mesurés = %v / %v, attendu non / oui (ligne film)", got[0].Measured, got[1].Measured)
	}
	if got[0].PlayerTeam != nil || got[1].PlayerTeam == nil || *got[1].PlayerTeam != 0 {
		t.Errorf("camps = %v / %v, attendu nil puis 0", got[0].PlayerTeam, got[1].PlayerTeam)
	}
	if len(got[1].Players) != 2 || got[1].TeamSize != 2 || got[1].LobbySize != 3 {
		t.Errorf("m1 = %+v, attendu 2 lignes joueur, camp 2, lobby 3", got[1])
	}
}
