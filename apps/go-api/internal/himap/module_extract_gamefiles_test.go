package himap

import (
	"testing"

	"levelup/go-api/internal/himodule"
)

// TestModuleExtractionRate est le garde-rail du calcul de dataBase (internal/himodule).
// L'ancienne règle `tailleFichier - max(dataOffset+compSize)` faisait échouer 1965/2047
// entrées sur any/levels/multi/catalyst et 100 % sur pc/. Le seuil ci-dessous bloque tout
// retour en arrière.
func TestModuleExtractionRate(t *testing.T) {
	for _, tc := range []struct {
		name    string
		path    string
		minRate float64
	}{
		{"ds/ridgeline", ridgelineDS, 1.0},
		{"any/catalyst", catalystAny, 0.99},
		{"pc/ridgeline", ridgelinePC, 0.95},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requireModule(t, tc.path)
			m, err := himodule.Open(tc.path)
			if err != nil {
				t.Fatal(err)
			}
			files := m.Files("")
			ok := 0
			for _, f := range files {
				d, err := m.Extract(f)
				if err == nil && len(d) == f.UncompSize {
					ok++
				}
			}
			rate := float64(ok) / float64(len(files))
			if rate < tc.minRate {
				t.Fatalf("taux d'extraction %.3f (%d/%d) < seuil %.3f", rate, ok, len(files), tc.minRate)
			}
		})
	}
}
