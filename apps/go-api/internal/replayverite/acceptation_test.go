package replayverite

import "testing"

// TestAccepterRequalifieLeSeulFauxNomme : seul le constat FAUX de la mesure nommée devient accepté ;
// un autre FAUX garde le verdict FAUX ; le statut se recalcule ; un constat non FAUX n'est pas touché.
func TestAccepterRequalifieLeSeulFauxNomme(t *testing.T) {
	c := Comparaison{Statut: StatutFaux, Constats: []Constat{
		{Mesure: "R-1 repli repli_a", Sens: StatutFaux},
		{Mesure: "R-1 repli repli_b", Sens: StatutFaux},
		{Mesure: "R-1 repli repli_c", Sens: sensInfo},
	}}
	if n := c.Accepter("R-1 repli repli_a", "accepte"); n != 1 {
		t.Fatalf("requalifies : %d, attendu 1", n)
	}
	if c.Constats[0].Sens != StatutAccepte || len(c.Constats[0].Detail) != 1 {
		t.Fatalf("constat accepte : %+v", c.Constats[0])
	}
	if c.Statut != StatutFaux {
		t.Fatalf("statut %s : un autre FAUX reste, le verdict reste FAUX", c.Statut)
	}
	if n := c.Accepter("R-1 repli repli_c", "accepte"); n != 0 || c.Constats[2].Sens != sensInfo {
		t.Fatalf("un constat non FAUX ne se requalifie pas")
	}
	c.Accepter("R-1 repli repli_b", "accepte")
	if c.Statut != StatutOK {
		t.Fatalf("statut %s apres acceptation de tous les FAUX, attendu ok", c.Statut)
	}
	m := Comparaison{Constats: []Constat{{Mesure: "x", Sens: StatutManque}, {Mesure: "y", Sens: StatutFaux}}}
	m.Accepter("y", "accepte")
	if m.Statut != StatutManque {
		t.Fatalf("statut %s : un MANQUE reste, attendu MANQUE", m.Statut)
	}
}
