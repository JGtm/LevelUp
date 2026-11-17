package analysis

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

// jeuDeCasParMinute : le fichier PARTAGÉ avec le test web de `pooledPerMinute`.
const jeuDeCasParMinute = "testdata/par_minute_cas.json"

type casParMinute struct {
	Nom    string `json:"nom"`
	Matchs []struct {
		Count   *float64 `json:"count"`
		Seconds *float64 `json:"seconds"`
	} `json:"matchs"`
	Attendu *float64 `json:"attendu"`
}

func TestPooledPerMinute_JeuDeCasPartage(t *testing.T) {
	raw, err := os.ReadFile(jeuDeCasParMinute)
	if err != nil {
		t.Fatalf("lecture de %s : %v", jeuDeCasParMinute, err)
	}
	var fichier struct {
		Cas []casParMinute `json:"cas"`
	}
	if err := json.Unmarshal(raw, &fichier); err != nil {
		t.Fatalf("décodage de %s : %v", jeuDeCasParMinute, err)
	}
	if len(fichier.Cas) == 0 {
		t.Fatalf("%s ne porte aucun cas : le test ne garderait rien", jeuDeCasParMinute)
	}
	type match = struct {
		Count   *float64 `json:"count"`
		Seconds *float64 `json:"seconds"`
	}
	for _, c := range fichier.Cas {
		got, ok := PooledPerMinute(c.Matchs,
			func(m match) (float64, bool) {
				if m.Count == nil {
					return 0, false
				}
				return *m.Count, true
			},
			func(m match) float64 {
				if m.Seconds == nil {
					return 0
				}
				return *m.Seconds
			})
		switch {
		case c.Attendu == nil && ok:
			t.Errorf("%s : valeur %v rendue, attendu aucune valeur", c.Nom, got)
		case c.Attendu != nil && !ok:
			t.Errorf("%s : aucune valeur rendue, attendu %v", c.Nom, *c.Attendu)
		case c.Attendu != nil && math.Abs(got-*c.Attendu) > 1e-9:
			t.Errorf("%s : %v, attendu %v", c.Nom, got, *c.Attendu)
		}
	}
}
