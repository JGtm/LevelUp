//go:build research

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLeDrapeauMPPDeclarePoseEtJournalise : sous `-mpp-declare`, une bobine du format 25 est mesuree
// sous le decoupage qu elle declare (8/3, presume par mesure), une bobine du format 27 sous celui
// de son format (9/5, relu), et le journal porte une ligne par film.
func TestLeDrapeauMPPDeclarePoseEtJournalise(t *testing.T) {
	dir := t.TempDir()
	tab, err := lireTable(cheminTable)
	if err != nil {
		t.Fatal(err)
	}
	rap, err := ouvrirRapport(dir, tab, modes{fermeture: true}, optionsV2{})
	if err != nil {
		t.Fatal(err)
	}
	jm, err := ouvrirJournalMPP(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"minifilm_e5adf7b2", "minifilm_bcb6d393"} {
		if err := mesurerUnFilm(racineRejeu, id, 4, rap, jm); err != nil {
			t.Fatalf("%s : %v", id, err)
		}
	}
	if err := jm.fermer(); err != nil {
		t.Fatal(err)
	}
	if err := rap.terminer(10); err != nil {
		t.Fatal(err)
	}
	brut, err := os.ReadFile(filepath.Join(dir, fichierMPPDeclare)) //nolint:gosec // repertoire du test
	if err != nil {
		t.Fatal(err)
	}
	for _, attendu := range []string{
		"minifilm_e5adf7b2\t25\t8/3\tpresume_par_mesure\t",
		"minifilm_bcb6d393\t27\t9/5\trelu\t",
	} {
		if !strings.Contains(string(brut), attendu) {
			t.Errorf("journal sans la ligne %q :\n%s", attendu, brut)
		}
	}
}
