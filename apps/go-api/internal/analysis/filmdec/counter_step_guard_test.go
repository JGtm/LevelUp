package filmdec

// counter_step_guard_test.go — LE GARDE-RAIL de counterStep (règle des <= 2 copies,
// CLAUDE.md n°6 ; revue adversariale ronde 1, F2 : l'arithmétique modulo 8 du compteur R(3)
// était copiée cinq fois). Une factorisation sans garde-rail re-diverge — leçon du prédicat
// bot passé de 8 à 36 copies après centralisation.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// counterStepLiteral est le motif interdit : la différence de compteurs repliée modulo 8.
// L'écriture par classes de caractères évite que ce fichier se dénonce lui-même.
var counterStepLiteral = regexp.MustCompile(`[+]\s*8\s*[)]\s*%\s*8`)

// TestCounterStepLitteralUnique interdit le littéral hors de son fichier hôte.
//
// EXCLUSIONS, chacune justifiée (2026-09-03) :
//   - equipment_changes.go : le fichier du helper — le littéral n'y vit qu'UNE fois,
//     dans counterStep.
//   - *_research_test.go : les instruments de mesure des lots R1-R3/R5, FIGÉS par le
//     protocole du chantier (PLAN_LECTURE_FIABLE_EQUIPEMENT §pilotage : « ne pas modifier
//     les instruments ») — ils rejouent les mesures des rapports tels quels, hors
//     production, gardés par variable d'environnement.
func TestCounterStepLitteralUnique(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob : %v", err)
	}
	if len(files) == 0 {
		t.Fatal("aucun fichier Go vu : le garde-rail ne garde rien")
	}
	helperSeen := false
	for _, f := range files {
		data, err := os.ReadFile(f) //nolint:gosec // fichiers du paquet lui-même
		if err != nil {
			t.Fatalf("lecture de %s : %v", f, err)
		}
		hits := counterStepLiteral.FindAll(data, -1)
		switch {
		case f == "equipment_changes.go":
			helperSeen = len(hits) > 0
			if len(hits) > 1 {
				t.Errorf("%s : %d occurrences du littéral modulo 8 — le fichier hôte n'en "+
					"porte qu'UNE, dans counterStep", f, len(hits))
			}
		case strings.HasSuffix(f, "_research_test.go"):
			// Instruments figés : voir l'en-tête. Jamais un blanc-seing pour du neuf —
			// un NOUVEL instrument appelle counterStep comme tout le monde.
		default:
			if len(hits) > 0 {
				t.Errorf("%s : littéral modulo 8 interdit (%d occurrence(s)) — appeler "+
					"counterStep (equipment_changes.go), le SEUL exemplaire", f, len(hits))
			}
		}
	}
	if !helperSeen {
		t.Error("counterStep ne porte plus le littéral : le garde-rail garde un fantôme — " +
			"le déplacer avec le helper")
	}
}
