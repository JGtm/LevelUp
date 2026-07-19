package service_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestFragDistributionLoggingCentralized verrouille la centralisation du log
// d'agrégation FragDistribution (CLAUDE.md règle 6, « <= 2 copies -> helper +
// garde-rail » ; PLAN_FRAG_DISTRIBUTION_V2 §6 D-P5-2). Toute surface qui construit une
// FragDistribution (Synthesis, Timeseries, Sessions, …) DOIT logguer ses compteurs via
// le helper PARTAGÉ logFragDistribution (synthesis_service_builders.go) — jamais en
// ré-inlinant un slog dédié. Ce test échoue si les marqueurs de message du helper
// réapparaissent dans un AUTRE fichier du package service (copie inline = re-divergence).
//
// Portée : les fichiers .go non-test du package service. Le helper vit dans
// logFragDistributionHelperFile (seule occurrence légitime des marqueurs).
const logFragDistributionHelperFile = "synthesis_service_builders.go"

// Marqueurs des messages émis par logFragDistribution (Debug + Warn). Un copier-coller
// du log inline embarquerait au moins l'un d'eux → détecté ici.
var fragDistributionLogMarkers = []string{
	"frag distribution built",
	"frag distribution over-count",
}

func TestFragDistributionLoggingCentralized(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	serviceDir := filepath.Dir(thisFile)

	entries, err := os.ReadDir(serviceDir)
	if err != nil {
		t.Fatalf("read service dir: %v", err)
	}
	var violations []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if name == logFragDistributionHelperFile {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(serviceDir, name))
		if readErr != nil {
			t.Fatalf("read %s: %v", name, readErr)
		}
		content := string(data)
		for _, marker := range fragDistributionLogMarkers {
			if strings.Contains(content, marker) {
				violations = append(violations, name+" (marqueur \""+marker+"\")")
			}
		}
	}
	if len(violations) > 0 {
		t.Errorf("log FragDistribution inline détecté hors du helper logFragDistribution "+
			"(%s) — router via logFragDistribution(ctx, surface, …) au lieu de ré-inliner un slog "+
			"(D-P5-2, règle <= 2 copies) :\n  %s",
			logFragDistributionHelperFile, strings.Join(violations, "\n  "))
	}
}
