package objectivescore

// goldenhelper_test.go — la comparaison à un fichier figé, pour ce paquet.
//
// Même mécanique que dans `killsource` : un fichier texte sous `testdata/`, régénéré par
// `-update`, comparé ligne à ligne le reste du temps. Le diff est rendu à la PREMIÈRE ligne
// qui diverge, avec son numéro : sur une sortie de plusieurs centaines de lignes, un
// « attendu != obtenu » global est illisible et pousse à régénérer sans lire.

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// majGolden : régénère les fichiers figés au lieu de les comparer.
var majGolden = flag.Bool("update", false, "régénère les fichiers golden de ce paquet")

// comparerGolden compare `obtenu` au fichier `testdata/<nom>.golden`.
func comparerGolden(t *testing.T, nom, obtenu string) {
	t.Helper()
	chemin := filepath.Join("testdata", nom+".golden")
	if *majGolden {
		if err := os.MkdirAll("testdata", 0o750); err != nil {
			t.Fatalf("création de testdata/ : %v", err)
		}
		if err := os.WriteFile(chemin, []byte(obtenu), 0o600); err != nil {
			t.Fatalf("écriture de %s : %v", chemin, err)
		}
		t.Logf("golden régénéré : %s (%d octets)", chemin, len(obtenu))
		return
	}
	brut, err := os.ReadFile(chemin) //nolint:gosec // chemin construit d'une constante de test
	if err != nil {
		t.Fatalf("golden %s illisible : %v — le régénérer : go test ./internal/analysis/"+
			"objectivescore/ -run <le test> -update", chemin, err)
	}
	attendu := strings.ReplaceAll(string(brut), "\r\n", "\n")
	if obtenu == attendu {
		return
	}
	la, lo := strings.Split(attendu, "\n"), strings.Split(obtenu, "\n")
	for i := 0; i < len(la) || i < len(lo); i++ {
		a, o := ligne(la, i), ligne(lo, i)
		if a != o {
			t.Fatalf("golden %s : divergence ligne %d\n  figé   : %s\n  obtenu : %s\n"+
				"(%d ligne(s) figées, %d obtenues) — si le changement est VOULU, le relire "+
				"puis régénérer avec -update", chemin, i+1, a, o, len(la), len(lo))
		}
	}
}

// ligne : la i-ème ligne d'une tranche, ou une marque d'absence.
func ligne(s []string, i int) string {
	if i < len(s) {
		return s[i]
	}
	return "<absente>"
}
