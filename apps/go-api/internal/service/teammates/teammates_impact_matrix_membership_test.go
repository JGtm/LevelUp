package teammates

// teammates_impact_matrix_membership_test.go — appartenance à l'escouade de la matrice
// d'impact décidée par XUID (lot perf L9-go, 2026-09-23, revue adversariale A, P1).

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// coequipierA : le coéquipier « A » (x_a) des tests de la matrice, résolu comme par
// buildTeammateRowWithMatches.
func coequipierA() []domain.TeammateRow {
	return []domain.TeammateRow{{Gamertag: "A", XUID: strPtr("x_a")}}
}

// TestBuildSquadImpactMatrix_AppartenanceParXUID : le coéquipier choisi sous son nom Q29
// (« madina ») et nommé « Madina » par Q32b — chaque lecture nomme par l'annuaire de SES
// matchs, un même xuid peut y porter deux casses — garde ses badges : l'appartenance se
// décide par xuid (teammates[].XUID), la cellule porte le nom de sa ligne (le nom choisi).
// Avant : 0 badge sous « madina » (test de la revue A, TestRevA_MatriceImpact_NomQ32bDifferentDuNomChoisi).
func TestBuildSquadImpactMatrix_AppartenanceParXUID(t *testing.T) {
	const mainXUID, matchID = "x_main", "m1"
	allies := []domain.AllyParticipant{
		{MatchID: matchID, XUID: mainXUID, Gamertag: "main", Kills: 5, Deaths: 2, Assists: 3, Outcome: domain.OutcomeWin},
		{MatchID: matchID, XUID: "x_t", Gamertag: "Madina", Kills: 4, Deaths: 0, Assists: 8, Outcome: domain.OutcomeWin},
		{MatchID: matchID, XUID: "x_ns", Gamertag: "NS", Kills: 12, Deaths: 1, Assists: 1, Outcome: domain.OutcomeWin},
	}
	svc := &TeammatesService{repo: &mockSquadRepo{allyRows: allies}, titleSlug: "halo_infinite", gamertag: "main"}
	rows := []domain.SquadMatchRow{{MatchID: matchID, StartTime: time.Date(2026, 4, 6, 18, 0, 0, 0, time.UTC), Outcome: domain.OutcomeWin}}
	for _, choisi := range []string{"Madina", "madina", "MADINA"} {
		teammates := []domain.TeammateRow{{Gamertag: choisi, XUID: strPtr("x_t")}}
		m := svc.buildSquadImpactMatrix(context.Background(), rows, mainXUID, []string{choisi}, teammates, allies)
		if m == nil {
			t.Fatalf("choisi %q : matrice nulle", choisi)
		}
		silentHero := false
		for _, c := range m.Cells {
			if c.Player != choisi {
				continue
			}
			for _, k := range c.BadgeKeys {
				silentHero = silentHero || k == "silent_hero"
			}
		}
		if !silentHero {
			t.Errorf("choisi %q (nom Q32b « Madina ») : silent_hero absent de sa ligne — cellules %+v", choisi, m.Cells)
		}
		// NS, allié hors escouade, n'a pas de ligne : son top_killer reste écarté.
		for _, c := range m.Cells {
			if c.Player == "NS" {
				t.Errorf("choisi %q : cellule pour l'allié hors escouade NS", choisi)
			}
		}
	}
}
