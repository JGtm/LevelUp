package feature

import "testing"

// TestStatusAvailable : une feature est exposable si available OU degraded (mode
// partiel), mais PAS unavailable. C'est le prédicat qui décide si le frontend
// montre la surface produit.
func TestStatusAvailable(t *testing.T) {
	cases := []struct {
		status Status
		want   bool
	}{
		{StatusAvailable, true},
		{StatusDegraded, true},
		{StatusUnavailable, false},
		{Status("bogus"), false}, // valeur inconnue → non exposable (fail-closed)
	}
	for _, tc := range cases {
		if got := tc.status.Available(); got != tc.want {
			t.Errorf("Status(%q).Available() = %v, want %v", tc.status, got, tc.want)
		}
	}
}

// TestMatrixGet : Get retourne le statut stocké, et StatusUnavailable (dégradation
// gracieuse) pour une feature absente de la matrice — jamais le zéro-value vide.
func TestMatrixGet(t *testing.T) {
	m := Matrix{
		KeyMatchHistory: StatusAvailable,
		KeyCitations:    StatusDegraded,
	}
	if got := m.Get(KeyMatchHistory); got != StatusAvailable {
		t.Errorf("Get(present available) = %q, want available", got)
	}
	if got := m.Get(KeyCitations); got != StatusDegraded {
		t.Errorf("Get(present degraded) = %q, want degraded", got)
	}
	if got := m.Get(KeyBattlePass); got != StatusUnavailable {
		t.Errorf("Get(absent) = %q, want unavailable", got)
	}
	// Matrice nil : pas de panic, défaut unavailable.
	var nilM Matrix
	if got := nilM.Get(KeyTimeseries); got != StatusUnavailable {
		t.Errorf("nilMatrix.Get = %q, want unavailable", got)
	}
}
