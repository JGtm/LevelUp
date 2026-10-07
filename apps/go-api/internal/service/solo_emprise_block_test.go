package service

// solo_emprise_block_test.go — L'ASSEMBLAGE DE L'EMPRISE avec une liste de joueurs fournie (Vue match :
// les joueurs de l'équipe, dans l'ordre de la page) : les fiches sont CETTE liste, dans cet ordre, et
// les prises de chaque objet se partagent entre ces joueurs et le reste de l'équipe. Sans liste,
// l'assemblage garde le joueur seul (Séries temporelles, Sessions : leurs propres tests).

import (
	"context"
	"fmt"
	"testing"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/analysis/squademprise"
	"levelup/go-api/internal/domain"
)

func TestBuildSoloEmpriseBlock_JoueursFournis(t *testing.T) {
	usage := usageTestRepoMock()
	usage.padTiers = []sessionusage.PadTierRow{
		{MatchID: "m1", XUID: "A", Tier: domain.PadTierPower, WeaponFamily: "sniper", Pickups: 2, PadsTotal: 2, PadsConfirmed: 2},
		{MatchID: "m1", XUID: "P", Tier: domain.PadTierPower, WeaponFamily: "sniper", Pickups: 1, PadsTotal: 2, PadsConfirmed: 2},
	}
	joueurs := []domain.SessionUsageSquadPlayer{{XUID: "P", Gamertag: "Papa"}, {XUID: "A", Gamertag: "Alpha"}}
	b := buildSoloEmpriseBlock(context.Background(), soloEmpriseQuery{
		Page: "match_view", Player: "Papa", PlayerXUID: "P", Locale: "fr",
		Current: []squademprise.Match{{MatchID: "m1"}}, UsageRepo: usage, Players: joueurs,
	})
	if fmt.Sprint(b.Players) != fmt.Sprint(joueurs) {
		t.Fatalf("fiches = %+v, attendu %+v dans cet ordre", b.Players, joueurs)
	}
	var sniper *domain.SquadEmpriseObject
	for i := range b.Objects {
		if b.Objects[i].Key == "sniper" {
			sniper = &b.Objects[i]
		}
	}
	if sniper == nil || len(sniper.Squad) != 3 {
		t.Fatalf("objet sniper = %+v, attendu trois parts (P, A, reste de l'équipe)", sniper)
	}
	if sniper.Squad[0].XUID != "P" || sniper.Squad[0].Taken != 1 || sniper.Squad[1].XUID != "A" || sniper.Squad[1].Taken != 2 {
		t.Errorf("parts = %+v, attendu P 1, A 2", sniper.Squad)
	}
}
