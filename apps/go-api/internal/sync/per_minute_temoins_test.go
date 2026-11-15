package sync

import (
	"strconv"
	"testing"
)

// Témoins de caractérisation des cadences par minute de la note de performance : valeurs attendues au bit
// près (chaîne 'g', -1) sur une durée non multiple de 60, une durée nulle (repli de durée) et une couverture
// d'objectif à 0. La note de performance dépend de ces valeurs (ADR 0047) : elles ne doivent pas bouger.
func TestTemoinMinute_MetriquesDeMatch(t *testing.T) {
	obj := 321.0
	zero := 0.0
	cas := []struct {
		label string
		row   historyRow
		want  string
	}{
		{"perf.937", historyRow{Kills: 11, Deaths: 7, Assists: 3, TimePlayedSeconds: 937, PersonalScore: 1234, DamageDealt: 4321, ObjectiveScore: &obj},
			"kpm=0.7043756670224119 dpm=0.44823906083244397 apm=0.192102454642476 pspm=79.01814300960513 dpmDamage=276.6915688367129 ospm=20.55496264674493"},
		{"perf.59", historyRow{Kills: 4, Deaths: 2, Assists: 1, TimePlayedSeconds: 59, PersonalScore: 77, DamageDealt: 900, ObjectiveScore: &zero},
			"kpm=4.067796610169491 dpm=2.0338983050847457 apm=1.0169491525423728 pspm=78.30508474576271 dpmDamage=915.2542372881356 ospm=0"},
		{"perf.zero", historyRow{Kills: 9, Deaths: 4, Assists: 2, TimePlayedSeconds: 0, PersonalScore: 500, DamageDealt: 1500, ObjectiveScore: &obj},
			"kpm=0.9 dpm=0.4 apm=0.2 pspm=50 dpmDamage=150 ospm=32.1"},
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
		got := "kpm=" + f(m.KPM) + " dpm=" + f(m.DPMDeaths) + " apm=" + f(m.APM) + " pspm=" + p(m.PSPM) +
			" dpmDamage=" + p(m.DPMDamage) + " ospm=" + p(m.OSPM)
		t.Logf("%s %s", c.label, got)
		if got != c.want {
			t.Errorf("%s : %s, attendu %s", c.label, got, c.want)
		}
	}
}
