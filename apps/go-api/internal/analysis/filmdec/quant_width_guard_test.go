package filmdec

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestQuantAxisWidthCentralized est le garde-rail de la règle CLAUDE #6 : la largeur de
// quantification d'un axe de position (6+level, forme fermée de FUN_140be9b88, cf
// .ai/HANDOFF_KEYFRAME_LIVE_CAPTURE.md « LES LARGEURS SONT OFFLINE ») doit passer par le
// helper unique quantAxisWidth — jamais réintroduite en littéral inline. Sans ce garde-rail
// la formule re-diverge dès que le cap min(26,...) ou la source de L évolue (leçon CLAUDE #6 :
// une factorisation sans garde-rail re-croît).
func TestQuantAxisWidthCentralized(t *testing.T) {
	bad := regexp.MustCompile(`6\s*\+\s*uint\(level\)`)
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue // ce fichier référence le motif dans la regex ci-dessus
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			if bad.MatchString(line) {
				t.Errorf("%s:%d largeur de quantification inline interdite (%q) — utiliser quantAxisWidth(uint(level))",
					f, i+1, strings.TrimSpace(line))
			}
		}
	}
}

// TestQuantAxisWidthFormula ancre la forme fermée min(26, 6+L) contre la formule complète
// FUN_140be9b88 pour L=0..8 (valeurs 6..14 mesurées/dérivées) et le plafond à 26.
func TestQuantAxisWidthFormula(t *testing.T) {
	for l := uint(0); l <= 8; l++ {
		if got, want := quantAxisWidth(l), 6+l; got != want {
			t.Errorf("quantAxisWidth(%d) = %d, want %d", l, got, want)
		}
	}
	// cap à 26 (0x1a) pour L élevé.
	for _, l := range []uint{20, 21, 40} {
		if got := quantAxisWidth(l); got > 26 {
			t.Errorf("quantAxisWidth(%d) = %d, dépasse le cap 26", l, got)
		}
	}
	if got := quantAxisWidth(20); got != 26 {
		t.Errorf("quantAxisWidth(20) = %d, want 26", got)
	}
}
