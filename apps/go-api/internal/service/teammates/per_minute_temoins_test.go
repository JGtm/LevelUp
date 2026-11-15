package teammates

import (
	"strconv"
	"testing"
)

// Témoins de caractérisation de SynergyScorePerMinute : valeur attendue au bit près (chaîne 'g', -1) sur une
// durée non multiple de 60, une durée nulle et une durée négative.
func TestTemoinMinute_SynergyScorePerMinute(t *testing.T) {
	cas := []struct {
		label       string
		score, secs float64
		want        string
	}{
		{"synergy.937", 1234, 937, "79.01814300960513"},
		{"synergy.59", 77, 59, "78.30508474576271"},
		{"synergy.zero", 500, 0, "0"},
		{"synergy.negatif", 500, -10, "0"},
	}
	for _, c := range cas {
		got := strconv.FormatFloat(SynergyScorePerMinute(c.score, c.secs), 'g', -1, 64)
		t.Logf("%s = %s", c.label, got)
		if got != c.want {
			t.Errorf("%s = %s, attendu %s", c.label, got, c.want)
		}
	}
}
