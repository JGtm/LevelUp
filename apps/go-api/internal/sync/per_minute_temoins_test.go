package sync

import (
	"strconv"
	"testing"
)

// Témoins de caractérisation des cadences par minute de la note de performance : valeurs exactes (au bit près)
// sur une durée non multiple de 60, une durée nulle (repli de durée) et une couverture d'objectif à 0.
func TestTemoinMinute_MetriquesDeMatch(t *testing.T) {
	obj := 321.0
	zero := 0.0
	cas := []struct {
		label string
		row   historyRow
	}{
		{"perf.937", historyRow{Kills: 11, Deaths: 7, Assists: 3, TimePlayedSeconds: 937, PersonalScore: 1234, DamageDealt: 4321, ObjectiveScore: &obj}},
		{"perf.59", historyRow{Kills: 4, Deaths: 2, Assists: 1, TimePlayedSeconds: 59, PersonalScore: 77, DamageDealt: 900, ObjectiveScore: &zero}},
		{"perf.zero", historyRow{Kills: 9, Deaths: 4, Assists: 2, TimePlayedSeconds: 0, PersonalScore: 500, DamageDealt: 1500, ObjectiveScore: &obj}},
	}
	f := func(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }
	p := func(v *float64) string {
		if v == nil {
			return "nil"
		}
		return f(*v)
	}
	for _, c := range cas {
		row := c.row
		m := extractMatchMetrics(&row)
		t.Logf("%s kpm=%s dpm=%s apm=%s pspm=%s dpmDamage=%s ospm=%s", c.label, f(m.KPM), f(m.DPMDeaths), f(m.APM), p(m.PSPM), p(m.DPMDamage), p(m.OSPM))
	}
}
