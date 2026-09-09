package service

// squad_service_v2_usage_test.go — étape E6.1 : la page Escouade publie le MÊME
// bloc que la Synthèse, agrégé PAR JOUEUR SUIVI. Les coéquipiers sélectionnés dans
// l'UI SONT les joueurs suivis du bloc — aucune seconde résolution.

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/games/canonical"
)

// squadUsageRow — une ligne canonical qui porte le xuid du joueur (le row() des
// autres tests n'en a pas, et sans xuid l'escouade n'a personne à suivre).
func squadUsageRow(matchID, xuid string, startedAt time.Time) canonical.PlayerMatchRow {
	return canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			MatchID: matchID, StartedAtUTC: startedAt, Outcome: canonical.OutcomeWin,
		},
		Self: canonical.MatchParticipant{
			Outcome:  canonical.OutcomeWin,
			Identity: canonical.PlayerIdentity{XUID: xuid, Gamertag: "x"},
		},
	}
}

func TestSquadPageV2_PublieLeBlocParJoueurSuivi(t *testing.T) {
	t.Parallel()
	t0 := time.Now().UTC().Add(-time.Hour)
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"Papa":  {squadUsageRow("m1", "P", t0)},
			"Alpha": {squadUsageRow("m1", "A", t0)},
		},
	}
	svc := NewSquadServiceV2(loader).WithEquipmentUsage(overviewRepoMock())
	resp, err := svc.GetSquadPage(context.Background(), "halo_infinite", "Papa",
		[]string{"Alpha"}, temporal.Period1Y, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("GetSquadPage : %v", err)
	}
	block := resp.EquipmentUsage
	if block == nil || !block.Available {
		t.Fatalf("bloc équipement = %+v, attendu disponible", block)
	}
	// Le coéquipier SÉLECTIONNÉ est le joueur suivi du bloc.
	if len(block.TrackedPlayers) != 1 || block.TrackedPlayers[0].XUID != "A" {
		t.Fatalf("coéquipiers suivis = %+v, attendu [A/Alpha]", block.TrackedPlayers)
	}
	if len(block.Players) != 2 || block.Players[0].XUID != "P" || block.Players[1].XUID != "A" {
		t.Fatalf("lignes joueur = %+v, attendu [P, A]", block.Players)
	}
	// Alpha a posé ses 2 murs : sa ligne dit « rien de gâché ».
	if ami := block.Players[1]; ami.Used != 2 || ami.Kept != 0 || ami.Dropped != 0 {
		t.Errorf("ligne de l'ami = %+v, attendu (utilisé 2, gardé 0, lâché 0)", ami)
	}
	// Le donut ferme sur le lobby des matchs partagés.
	eq := block.EquipmentParties
	if eq == nil || eq.Player+eq.Friends+eq.RestOfTeam+eq.Opponents != eq.LobbyTotal {
		t.Errorf("parts du donut = %+v, la somme doit valoir le lobby", eq)
	}
}

// TestSquadPageV2_SansCapabiliteLeBlocDitPourquoi — repo non câblé (titre sans
// film.usage_summary) : réponse partielle propre, jamais un 500.
func TestSquadPageV2_SansCapabiliteLeBlocDitPourquoi(t *testing.T) {
	t.Parallel()
	t0 := time.Now().UTC().Add(-time.Hour)
	loader := &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"Papa":  {squadUsageRow("m1", "P", t0)},
			"Alpha": {squadUsageRow("m1", "A", t0)},
		},
	}
	resp, err := NewSquadServiceV2(loader).GetSquadPage(context.Background(), "halo_infinite",
		"Papa", []string{"Alpha"}, temporal.Period1Y, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("GetSquadPage : %v", err)
	}
	if resp.EquipmentUsage == nil || resp.EquipmentUsage.Available {
		t.Errorf("bloc = %+v, attendu présent et indisponible", resp.EquipmentUsage)
	}
}
