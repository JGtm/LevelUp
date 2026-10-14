package filmproc

import (
	"math"
	"runtime/debug"
	"testing"
)

// TestArmBytes_PlafondSousLeGibioctet — un plafond en OCTETS (la recuisson de la démo arme
// 768 Mio, sous la granularité de Arm) : le souple est posé tel quel, le dur 25 % au-dessus ;
// zéro désarme.
func TestArmBytes_PlafondSousLeGibioctet(t *testing.T) {
	t.Cleanup(func() { debug.SetMemoryLimit(math.MaxInt64) })
	const soft = 768 << 20
	g := ArmBytes("test", soft, func(uint64) {})
	defer g.Disarm()
	if g.hardLimit != hardMargin(soft) {
		t.Errorf("plafond dur %d, attendu %d", g.hardLimit, hardMargin(soft))
	}
	if got := debug.SetMemoryLimit(-1); got != soft {
		t.Errorf("plafond souple du runtime %d, attendu %d", got, soft)
	}
	g0 := ArmBytes("test", 0, func(uint64) {})
	defer g0.Disarm()
	if g0.hardLimit != 0 {
		t.Errorf("zéro doit désarmer, plafond dur %d", g0.hardLimit)
	}
}
