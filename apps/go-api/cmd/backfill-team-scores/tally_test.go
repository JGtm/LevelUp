package main

// tally_test.go — LE RÉSUMÉ DE FIN DE COURSE, ET SON AVERTISSEMENT.
//
// POURQUOI CES TESTS EXISTENT. Le découpage en deux phases a cassé le compteur sans que
// rien ne le dise : `fixed` n'est incrémenté que par la phase B, donc en répétition à blanc
// le résumé annonçait « corriges=0 » alors que N corrections étaient prêtes, et
// l'avertissement « RIEN n'a été écrit » — gardé par `t.fixed > 0` — était devenu du code
// mort. L'attendu du mode d'emploi du jour J (« corriges=80 » au dry-run) était donc
// devenu impossible à observer. Défaut P1-a de la ronde 2 de revue.
//
// La leçon est générale : un compteur sans test se désynchronise du code au premier
// refactor, et un compteur faux dans un outil d'écriture est pire qu'absent — il rassure.

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

// captureLogs redirige le logger par défaut vers un tampon, le temps du test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &buf
}

func TestReportTally(t *testing.T) {
	ctx := context.Background()

	t.Run("dry-run annonce les corrections planifiees et avertit", func(t *testing.T) {
		buf := captureLogs(t)
		reportTally(ctx, false, tally{read: 10, identical: 8, planned: 2})
		out := buf.String()
		if !strings.Contains(out, "planifiees=2") {
			t.Errorf("le résumé n'annonce pas les corrections planifiées :\n%s", out)
		}
		if !strings.Contains(out, "corriges=0") {
			t.Errorf("le résumé devrait dire corriges=0 en répétition à blanc :\n%s", out)
		}
		if !strings.Contains(out, "RIEN n'a été écrit") {
			t.Errorf("l'avertissement de répétition à blanc est INATTEIGNABLE — "+
				"c'est exactement le défaut P1-a :\n%s", out)
		}
		if !strings.Contains(out, "a_corriger=2") {
			t.Errorf("l'avertissement doit porter le nombre de corrections en attente :\n%s", out)
		}
	})

	t.Run("apply annonce les corrections ecrites, sans avertissement", func(t *testing.T) {
		buf := captureLogs(t)
		reportTally(ctx, true, tally{read: 10, identical: 8, planned: 2, fixed: 2})
		out := buf.String()
		if !strings.Contains(out, "planifiees=2") || !strings.Contains(out, "corriges=2") {
			t.Errorf("après application, planifiees et corriges doivent tous deux valoir 2 :\n%s", out)
		}
		if strings.Contains(out, "RIEN n'a été écrit") {
			t.Errorf("l'avertissement de répétition à blanc ne doit pas paraître avec --apply :\n%s", out)
		}
	})

	t.Run("rien a corriger : pas d'avertissement", func(t *testing.T) {
		buf := captureLogs(t)
		reportTally(ctx, false, tally{read: 10, identical: 10})
		if strings.Contains(buf.String(), "RIEN n'a été écrit") {
			t.Errorf("aucune correction planifiée : l'avertissement est du bruit :\n%s", buf.String())
		}
	})

	t.Run("apply incomplet : l'ecart planifiees/corriges reste lisible", func(t *testing.T) {
		buf := captureLogs(t)
		// Deux corrections prévues, une seule écrite (l'autre a échoué).
		reportTally(ctx, true, tally{read: 10, planned: 2, fixed: 1, failed: 1})
		out := buf.String()
		if !strings.Contains(out, "planifiees=2") || !strings.Contains(out, "corriges=1") {
			t.Errorf("un apply partiel doit rester visible au résumé :\n%s", out)
		}
	})
}

// TestPlanMatch_CompteLesCorrectionsPlanifiees : la phase A doit compter, sinon le dry-run
// ne peut rien annoncer. C'est la moitié amont du défaut P1-a.
func TestPlanMatch_CompteLesCorrectionsPlanifiees(t *testing.T) {
	ctx := context.Background()
	ids := []string{"a-corriger-1", "a-corriger-2", "deja-juste", "ffa"}

	fetcher := &fakeFetcher{payloads: map[string]map[string]any{
		"a-corriger-1": teamsPayload(team(0, 200), team(1, 126)),
		"a-corriger-2": teamsPayload(team(0, 3), team(1, 0)),
		"deja-juste":   teamsPayload(team(0, 50), team(1, 42)),
		"ffa":          teamsPayload(team(2, 30), team(3, 25)),
	}}
	reader := &fakeReader{rows: map[string]RegistryScores{
		"a-corriger-1": {Team0: ptr(193), Team1: ptr(112)},
		"a-corriger-2": {Team0: ptr(105), Team1: ptr(8)},
		"deja-juste":   {Team0: ptr(50), Team1: ptr(42)},
		"ffa":          {Team0: ptr(30), Team1: ptr(25)},
	}}

	// Même boucle que planPhase, sans l'ouverture de base (seule partie non testable ici).
	var tl tally
	var plans []plannedFix
	for _, id := range ids {
		if p, ok := planMatch(ctx, fetcher, reader, id, &tl); ok {
			plans = append(plans, p)
		}
	}

	if len(plans) != 2 {
		t.Fatalf("%d plans, 2 attendus", len(plans))
	}
	if tl.planned != 2 {
		t.Errorf("planned = %d, attendu 2 — le dry-run afficherait un faux compte", tl.planned)
	}
	if tl.planned != len(plans) {
		t.Errorf("planned (%d) et len(plans) (%d) divergent : le résumé mentirait sur ce qui reste à faire",
			tl.planned, len(plans))
	}
	if tl.fixed != 0 {
		t.Errorf("fixed = %d : la phase A n'écrit rien, elle ne doit jamais l'incrémenter", tl.fixed)
	}
	if tl.identical != 1 || tl.skipped != 1 || tl.read != 4 {
		t.Errorf("tally = %+v, attendu 1 identique / 1 skip / 4 lus", tl)
	}
}

// TestDeuxPhases_PlanifieesPuisCorrigees déroule la séquence complète telle que la vivrait
// le jour J : la phase A planifie 2 corrections, la phase B les écrit toutes les deux.
func TestDeuxPhases_PlanifieesPuisCorrigees(t *testing.T) {
	ctx := context.Background()
	ids := []string{"m1", "m2"}

	fetcher := &fakeFetcher{payloads: map[string]map[string]any{
		"m1": teamsPayload(team(0, 200), team(1, 126)),
		"m2": teamsPayload(team(0, 3), team(1, 0)),
	}}
	reader := &fakeReader{rows: map[string]RegistryScores{
		"m1": {Team0: ptr(193), Team1: ptr(112)},
		"m2": {Team0: ptr(105), Team1: ptr(8)},
	}}
	exec := &fakeExecer{}

	var tl tally
	var plans []plannedFix
	for _, id := range ids {
		if p, ok := planMatch(ctx, fetcher, reader, id, &tl); ok {
			plans = append(plans, p)
		}
	}
	if tl.planned != 2 || tl.fixed != 0 {
		t.Fatalf("après la phase A : tally = %+v, attendu 2 planifiées / 0 écrite", tl)
	}
	for _, p := range plans {
		applyOne(ctx, reader, sqlRegistry{ex: exec}, p, &tl)
	}
	if tl.planned != 2 || tl.fixed != 2 {
		t.Errorf("après la phase B : tally = %+v, attendu 2 planifiées / 2 écrites", tl)
	}
	if len(exec.calls) != 2 {
		t.Errorf("%d écritures, 2 attendues", len(exec.calls))
	}
}
