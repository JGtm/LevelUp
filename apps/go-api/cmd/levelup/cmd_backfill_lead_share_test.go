package main

import (
	"strings"
	"testing"

	"levelup/go-api/internal/persist"
)

func TestLigneDryRunMainmise(t *testing.T) {
	mesure := persist.LeadShareBatch{
		MatchID: "m", Measured: true, DocSchema: 71, MeasuredMS: 600000, LeadChanges: 3,
		Rows: []persist.LeadShareRow{{Camp: 0, LeadMS: 300000}, {Camp: 1, LeadMS: 90000}},
	}
	ligne := ligneDryRunMainmise("m", mesure)
	for _, attendu := range []string{"mesure=true", "duree_s= 600", "retournements=3", "camps=0:50% 1:15%", "raison=-"} {
		if !strings.Contains(ligne, attendu) {
			t.Errorf("la ligne ne contient pas %q : %q", attendu, ligne)
		}
	}
	refus := ligneDryRunMainmise("m", persist.LeadShareBatch{MatchID: "m", Reason: "mode_events"})
	if !strings.Contains(refus, "mesure=false") || !strings.Contains(refus, "raison=mode_events") {
		t.Errorf("ligne d'un refus : %q", refus)
	}
}

func TestSyntheseMainmise_TauxEtRaisonsTriees(t *testing.T) {
	b := bilanLeadShareBackfill{mesures: 6, ecrits: 10, raisons: map[string]int{"mode_events": 3, "clock_origin_missing": 1}}
	s := syntheseMainmise(b, 0)
	if !strings.Contains(s, "6 mesures (60 %)") || !strings.Contains(s, "4 non mesures") {
		t.Errorf("synthèse sans le taux : %q", s)
	}
	if strings.Index(s, "mode_events") > strings.Index(s, "clock_origin_missing") {
		t.Errorf("les raisons ne sont pas triées par effectif décroissant : %q", s)
	}
}
