package squadagg

import (
	"testing"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
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
	got := SquadPlayers("moi", "JGtm", participants, []domain.SessionUsageSquadPlayer{{XUID: "cop", Gamertag: "Madina97294"}})
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

// LES COÉQUIPIERS SONT RETROUVÉS PAR XUID. Sur les huit matchs du 7 octobre 2026, aucune ligne
// de participant ne portait le gamertag de Chocoboflor ni de Madina97294 : comparés par nom, ils
// disparaissaient de « Répartition de l'objectif dans l'escouade ».
func TestFormesSquadPlayers_CoequipiersSansGamertagDeParticipant(t *testing.T) {
	team := 0
	participants := []sessionusage.ParticipantRow{
		{MatchID: "m1", XUID: "moi", Gamertag: "JGtm", TeamID: &team},
		{MatchID: "m1", XUID: "choco", Gamertag: "", TeamID: &team},
		{MatchID: "m1", XUID: "madina", Gamertag: "", TeamID: &team},
	}
	selection := []domain.SessionUsageSquadPlayer{{XUID: "choco", Gamertag: "Chocoboflor"}, {XUID: "madina", Gamertag: "Madina97294"}}
	got := SquadPlayers("moi", "JGtm", participants, selection)
	if len(got) != 3 || got[1] != selection[0] || got[2] != selection[1] {
		t.Fatalf("escouade = %+v, attendu [JGtm, Chocoboflor, Madina97294]", got)
	}
}

// À défaut de nom connu de la page, les participants restent le repli.
func TestFormesSquadPlayers_RepliSurLesParticipants(t *testing.T) {
	participants := []sessionusage.ParticipantRow{{MatchID: "m1", XUID: "moi", Gamertag: "JGtm"}}
	got := SquadPlayers("moi", "", participants, nil)
	if len(got) != 1 || got[0].Gamertag != "JGtm" {
		t.Fatalf("repli participants attendu, obtenu %+v", got)
	}
}
