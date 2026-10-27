// Package archlint — i18n_placement_single_source_test.go : le libellé stocké de la phase de
// placement n'a qu'une source (ADR 0040 D-5).
//
// `tier_label` d'une ligne de rang en placement s'écrit et se relit par `domain.PlacementLabel` et
// `domain.ParsePlacementRemaining`. Un littéral « Placement (… » ou une expression régulière sur
// « restant » ailleurs recrée une forme en français lue par un seul des deux côtés.
package archlint

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

var placementLiteralRE = regexp.MustCompile(`"Placement \(|\d\+\)?\s\*restant`)

func TestPlacementLabelSingleSource(t *testing.T) {
	const home = "internal/domain/placement_label.go"
	var offenders []string
	for _, s := range goSourcesUnder(t, "cmd", "internal", "pkg", "tools", "scripts") {
		if s.Rel == home {
			continue
		}
		for i, line := range strings.Split(string(s.Src), "\n") {
			if placementLiteralRE.MatchString(line) {
				offenders = append(offenders, fmt.Sprintf("%s:%d", s.Rel, i+1))
			}
		}
	}
	if len(offenders) > 0 {
		t.Fatalf("libellé de placement écrit hors %s (utiliser domain.PlacementLabel / domain.ParsePlacementRemaining) :\n%s",
			home, strings.Join(offenders, "\n"))
	}
}
