package fallback

import "testing"

// TestNomHorsRegistreDevientUnDiagnostic : un nom hors registre se compte ET se signale — un
// diagnostic de niveau erreur, releve une seule fois par l orchestrateur (lot J12.3, ADR 0034 D-4).
func TestNomHorsRegistreDevientUnDiagnostic(t *testing.T) {
	c := NouveauCompteur()
	c.DeclencheN("repli_qui_n_existe_pas", 2)
	if c.Compte("repli_qui_n_existe_pas") != 2 {
		t.Fatalf("le compte d un nom hors registre doit rester tenu")
	}
	ds := c.Diagnostics().Relever()
	if len(ds) != 1 || ds[0].Code != DiagRepliHorsRegistre {
		t.Fatalf("diagnostics = %+v, attendu un seul %s", ds, DiagRepliHorsRegistre)
	}
	if again := c.Diagnostics().Relever(); len(again) != 0 {
		t.Fatalf("un diagnostic se releve une fois : %+v", again)
	}
	var nul *Compteur
	if nul.Diagnostics().Relever() != nil {
		t.Fatal("un compteur nil ne recueille rien")
	}
}
