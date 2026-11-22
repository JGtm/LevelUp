package teammates

// teammates_impact_matrix_membership_test.go — appartenance à l'escouade du tableau d'impact par
// session décidée par XUID (lot perf L9-go, 2026-09-23, revue adversariale A, P1).

import (
	"context"
	"slices"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// coequipierA : le coéquipier « A » (x_a) des tests du tableau, résolu comme par
// buildTeammateRowWithMatches.
func coequipierA() []domain.TeammateRow {
	return []domain.TeammateRow{{Gamertag: "A", XUID: strPtr("x_a")}}
}

// impactDe rend le tableau d'impact par session d'une population de lignes (étiquetées d'une session
// unique « S » quand elles n'en ont pas) pour le joueur principal « main » et le coéquipier A.
func impactDe(svc *TeammatesService, rows []domain.SquadMatchRow, allies []domain.AllyParticipant) *domain.SquadImpactSessions {
	rows = etiquetees(rows, "S")
	return svc.buildSquadImpact(context.Background(), impactEscouade{
		rows: rows, timeline: rows, evenings: soireesDImpact(rows, rows, true),
		mainXUID: "x_main", selected: []string{"A"}, teammates: coequipierA(), allies: allies,
	})
}

// TestBuildSquadImpactSessions_AppartenanceParXUID : le coéquipier choisi sous son nom Q29
// (« madina ») et nommé « Madina » par Q32b — chaque lecture nomme par l'annuaire de SES
// matchs, un même xuid peut y porter deux casses — garde ses rôles : l'appartenance se
// décide par xuid (teammates[].XUID), la ligne porte le nom choisi.
// Avant : 0 rôle sous « madina » (test de la revue A, TestRevA_MatriceImpact_NomQ32bDifferentDuNomChoisi).
func TestBuildSquadImpactSessions_AppartenanceParXUID(t *testing.T) {
	const mainXUID, matchID = "x_main", "m1"
	allies := []domain.AllyParticipant{
		{MatchID: matchID, XUID: mainXUID, Gamertag: "main", Kills: 5, Deaths: 2, Assists: 3, Outcome: domain.OutcomeWin},
		{MatchID: matchID, XUID: "x_t", Gamertag: "Madina", Kills: 4, Deaths: 0, Assists: 8, Outcome: domain.OutcomeWin},
		{MatchID: matchID, XUID: "x_ns", Gamertag: "NS", Kills: 12, Deaths: 1, Assists: 1, Outcome: domain.OutcomeWin},
	}
	svc := &TeammatesService{repo: &mockSquadRepo{allyRows: allies}, titleSlug: "halo_infinite", gamertag: "main"}
	rows := []domain.SquadMatchRow{{MatchID: matchID, StartTime: time.Date(2026, 4, 6, 18, 0, 0, 0, time.UTC), Outcome: domain.OutcomeWin}}
	for _, choisi := range []string{"Madina", "madina", "MADINA"} {
		rows := etiquetees(rows, "S")
		sessions := svc.buildSquadImpact(context.Background(), impactEscouade{
			rows: rows, timeline: rows, evenings: soireesDImpact(rows, rows, true), mainXUID: mainXUID,
			selected: []string{choisi}, teammates: []domain.TeammateRow{{Gamertag: choisi, XUID: strPtr("x_t")}}, allies: allies,
		})
		if sessions == nil {
			t.Fatalf("choisi %q : bloc nul", choisi)
		}
		ligne := joueurDe(sessions.Sessions[0], choisi)
		if !slices.Contains(rolesDe(ligne), "silent_hero") {
			t.Errorf("choisi %q (nom Q32b « Madina ») : silent_hero absent de sa ligne — %+v", choisi, ligne)
		}
		// NS, allié hors escouade, n'a pas de ligne : son top_killer reste écarté.
		if !slices.Equal(sessions.Players, []string{"main", choisi}) {
			t.Errorf("choisi %q : joueurs %v, attendu le principal et le choisi seuls", choisi, sessions.Players)
		}
		for _, tot := range sessions.Totals {
			if compteDuRole(tot.Counts, "top_killer") != 0 {
				t.Errorf("choisi %q : %s a un top_killer (tombé sur NS, allié hors escouade)", choisi, tot.Player)
			}
		}
	}
}
