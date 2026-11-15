package teammates

import (
	"strconv"
	"testing"
)

// Témoins de caractérisation de SynergyScorePerMinute : valeur exacte (au bit près) sur une durée non multiple
// de 60, une durée nulle et une durée négative.
func TestTemoinMinute_SynergyScorePerMinute(t *testing.T) {
	cas := []struct {
		label       string
		score, secs float64
	}{
		{"synergy.937", 1234, 937},
		{"synergy.59", 77, 59},
		{"synergy.zero", 500, 0},
		{"synergy.negatif", 500, -10},
	}
	for _, c := range cas {
		t.Logf("%s = %s", c.label, strconv.FormatFloat(SynergyScorePerMinute(c.score, c.secs), 'g', -1, 64))
	}
}
