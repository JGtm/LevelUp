package gameunits

import (
	"math"
	"testing"
)

func TestConversion(t *testing.T) {
	if got := UnitsToMeters(7); math.Abs(got-21.336) > 1e-9 {
		t.Errorf("UnitsToMeters(7) = %v, attendu 21,336", got)
	}
	if got := MetersToUnits(18); math.Abs(got-5.905511811) > 1e-6 {
		t.Errorf("MetersToUnits(18) = %v, attendu 5,9055", got)
	}
	if got := MetersToUnits(UnitsToMeters(12.5)); math.Abs(got-12.5) > 1e-9 {
		t.Errorf("aller-retour = %v", got)
	}
}
