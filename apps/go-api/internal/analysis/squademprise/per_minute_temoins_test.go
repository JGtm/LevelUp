package squademprise

import (
	"strconv"
	"testing"
)

// Témoins de caractérisation des frags par minute d'effet : valeur au bit près (chaîne 'g', -1) sur un temps
// d'effet non multiple de 60 000 ms, et nil sans temps d'effet.
func TestTemoinMinute_FragsParMinuteDEffet(t *testing.T) {
	cas := []struct {
		label string
		kills int
		ms    int64
		want  string
	}{
		{"effet.11sur937000", 11, 937000, "0.7043756670224119"},
		{"effet.23sur736123", 23, 736123, "1.8746867031732468"},
		{"effet.5sur59999", 5, 59999, "5.0000833347222455"},
		{"effet.zero", 9, 0, "nil"},
		{"effet.negatif", 9, -10, "nil"},
	}
	for _, c := range cas {
		got := "nil"
		if v := parMinute(c.kills, c.ms); v != nil {
			got = strconv.FormatFloat(*v, 'g', -1, 64)
		}
		t.Logf("%s = %s", c.label, got)
		if got != c.want {
			t.Errorf("%s = %s, attendu %s", c.label, got, c.want)
		}
	}
}
