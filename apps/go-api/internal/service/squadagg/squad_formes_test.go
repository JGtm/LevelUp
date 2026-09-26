package squadagg

import (
	"testing"

	"levelup/go-api/internal/analysis/sessionusage"
)

// LE NOM DU JOUEUR DE LA PAGE NE PEUT PAS ÊTRE UN XUID. Sur un scope dont
// aucune ligne de participant ne porte son gamertag, l'écran affichait
// « 2533274823110022 » à la place de « JGtm » (mesuré le 2026-09-13).
func TestFormesSquadPlayers_NomDuJoueurDeLaPage(t *testing.T) {
	team := 0
	participants := []sessionusage.ParticipantRow{
		{MatchID: "m1", XUID: "moi", Gamertag: "", TeamID: &team},
		{MatchID: "m1", XUID: "cop", Gamertag: "Madina97294", TeamID: &team},
	}
	got := formesSquadPlayers("moi", "JGtm", participants, []string{"Madina97294"})
	if len(got) != 2 {
		t.Fatalf("le joueur de la page et son coéquipier attendus, obtenu %+v", got)
	}
	if got[0].XUID != "moi" || got[0].Gamertag != "JGtm" {
		t.Fatalf("le joueur de la page ouvre la liste, nommé par la page : %+v", got[0])
	}
	if got[1].Gamertag != "Madina97294" {
		t.Fatalf("le coéquipier garde son nom résolu : %+v", got[1])
	}
}

// À défaut de nom connu de la page, les participants restent le repli.
func TestFormesSquadPlayers_RepliSurLesParticipants(t *testing.T) {
	participants := []sessionusage.ParticipantRow{{MatchID: "m1", XUID: "moi", Gamertag: "JGtm"}}
	got := formesSquadPlayers("moi", "", participants, nil)
	if len(got) != 1 || got[0].Gamertag != "JGtm" {
		t.Fatalf("repli participants attendu, obtenu %+v", got)
	}
}
