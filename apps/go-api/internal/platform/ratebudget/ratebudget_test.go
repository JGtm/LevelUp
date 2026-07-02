package ratebudget

import "testing"

// TestForXUID_SharedByConstruction : deux consommateurs du même xuid reçoivent
// LE MÊME limiteur (pointeur identique) — le budget est partagé par construction.
// xuid vide → limiteurs distincts (non attribuable).
func TestForXUID_SharedByConstruction(t *testing.T) {
	a := ForXUID("xuid-partage-1", 5)
	b := ForXUID("xuid-partage-1", 99) // rps ignoré après création
	if a != b {
		t.Fatal("même xuid doit retourner le MÊME limiteur (budget partagé)")
	}
	if got := float64(a.Limit()); got != 5 {
		t.Errorf("débit initial = %v, want 5 (le rps du 2e appel est ignoré)", got)
	}
	if ForXUID("", 5) == ForXUID("", 5) {
		t.Error("xuid vide doit retourner des limiteurs DISTINCTS")
	}
	if ForXUID("autre-xuid-1", 5) == a {
		t.Error("xuids différents doivent avoir des limiteurs distincts")
	}
}

// TestAIMD_HalveAndRestore : ÷2 planché sur 429, restauration additive plafonnée
// au débit nominal — l'ajustement est visible par tous les consommateurs (même
// pointeur).
func TestAIMD_HalveAndRestore(t *testing.T) {
	lim := ForXUID("xuid-aimd-1", 4)

	if got := HalveRPS("xuid-aimd-1"); got != 2 {
		t.Errorf("HalveRPS = %v, want 2", got)
	}
	if got := HalveRPS("xuid-aimd-1"); got != 1 {
		t.Errorf("HalveRPS = %v, want 1 (plancher minRPS)", got)
	}
	if got := HalveRPS("xuid-aimd-1"); got != 1 {
		t.Errorf("HalveRPS sous plancher = %v, want 1", got)
	}
	// Le limiteur partagé reflète l'ajustement (même pointeur).
	if got := float64(lim.Limit()); got != 1 {
		t.Errorf("limiteur partagé = %v, want 1", got)
	}

	if got := RestoreStep("xuid-aimd-1", 0.5); got != 1.5 {
		t.Errorf("RestoreStep = %v, want 1.5", got)
	}
	for i := 0; i < 20; i++ {
		RestoreStep("xuid-aimd-1", 0.5)
	}
	if got := CurrentRPS("xuid-aimd-1"); got != 4 {
		t.Errorf("restauration plafonnée au nominal : %v, want 4", got)
	}

	// Comptes inconnus : no-op sûrs.
	if HalveRPS("inconnu-x") != 0 || RestoreStep("inconnu-x", 1) != 0 || CurrentRPS("inconnu-x") != 0 {
		t.Error("xuid inconnu doit être un no-op (0)")
	}
}
