package mapcatalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ecrireLexique pose un lexique de test et rend son chemin.
func ecrireLexique(t *testing.T, lignes ...string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "callouts_lexique.csv")
	if err := os.WriteFile(p, []byte(strings.Join(lignes, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestChargerLexiqueLitLesDeuxLangues — le cas nominal, BOM de tableur toléré.
func TestChargerLexiqueLitLesDeuxLangues(t *testing.T) {
	p := ecrireLexique(t, bomUTF8+"string_id;en;fr", "0x4E9E5E09;River;Rivière", "0x22908E62;Ramp;Rampe")
	lex, err := ChargerLexique(p)
	if err != nil {
		t.Fatalf("ChargerLexique : %v", err)
	}
	if len(lex) != 2 || lex[0x4E9E5E09] != (Libelle{EN: "River", FR: "Rivière"}) {
		t.Fatalf("lexique = %+v", lex)
	}
}

// TestChargerLexiqueRefuseLesFormesFausses — un en-tête réordonné, un libellé vide ou un
// string_id porteur de deux textes font échouer la lecture : un faux nom est pire qu'un nom
// absent.
func TestChargerLexiqueRefuseLesFormesFausses(t *testing.T) {
	cas := map[string][]string{
		"en-tête réordonné":   {"string_id;fr;en", "0x1;A;B"},
		"libellé vide":        {"string_id;en;fr", "0x1;River;"},
		"deux libellés":       {"string_id;en;fr", "0x1;River;Rivière", "0x1;Creek;Ruisseau"},
		"string_id illisible": {"string_id;en;fr", "zz;River;Rivière"},
	}
	for nom, lignes := range cas {
		if _, err := ChargerLexique(ecrireLexique(t, lignes...)); err == nil {
			t.Errorf("%s : erreur attendue", nom)
		}
	}
	if _, err := ChargerLexique(filepath.Join(t.TempDir(), "absent.csv")); !os.IsNotExist(err) {
		t.Errorf("fichier absent : err = %v, attendu une absence reconnaissable", err)
	}
}
