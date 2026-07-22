package skill

// skill_v2_metrics_test.go — tests des accesseurs de la jauge interior_gaps (delta
// borné du chemin de réparation manuelle). Pur expvar, aucune DB.

import "testing"

// TestAddLUSRInteriorGapsGauge couvre le contrat de l'accesseur : delta positif,
// delta négatif partiel, clamp à 0 sous zéro, et no-op sur delta nul.
func TestAddLUSRInteriorGapsGauge(t *testing.T) {
	SetLUSRInteriorGapsGauge(0)
	if got := LUSRInteriorGapsGaugeValue(); got != 0 {
		t.Fatalf("init: got %d, want 0", got)
	}

	AddLUSRInteriorGapsGauge(5) // delta positif
	if got := LUSRInteriorGapsGaugeValue(); got != 5 {
		t.Errorf("delta +5: got %d, want 5", got)
	}

	AddLUSRInteriorGapsGauge(-2) // delta négatif partiel
	if got := LUSRInteriorGapsGaugeValue(); got != 3 {
		t.Errorf("delta -2: got %d, want 3", got)
	}

	AddLUSRInteriorGapsGauge(-10) // sous zéro → clamp à 0
	if got := LUSRInteriorGapsGaugeValue(); got != 0 {
		t.Errorf("clamp: got %d, want 0", got)
	}

	SetLUSRInteriorGapsGauge(7)
	AddLUSRInteriorGapsGauge(0) // no-op
	if got := LUSRInteriorGapsGaugeValue(); got != 7 {
		t.Errorf("delta 0: got %d, want 7", got)
	}

	// Nettoyage : ne pas laisser la jauge globale polluée pour d'autres tests.
	SetLUSRInteriorGapsGauge(0)
}
