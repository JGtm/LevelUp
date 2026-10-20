package tactical

import (
	"testing"

	"levelup/go-api/internal/domain"
)

func TestMedianeM(t *testing.T) {
	if MedianeM(nil) != nil {
		t.Fatal("sans valeur : nil")
	}
	if m := MedianeM([]float64{5, 1, 3}); *m != 3 {
		t.Errorf("impair : %v", *m)
	}
	if m := MedianeM([]float64{4, 1, 3, 2}); *m != 2.5 {
		t.Errorf("pair : %v", *m)
	}
}

func TestDistanceM_DeuxDecimales(t *testing.T) {
	if d := DistanceM(domain.TacticalPoint{X: 0, Y: 0}, domain.TacticalPoint{X: 1, Y: 1}); d != 1.41 {
		t.Errorf("DistanceM = %v, attendu 1.41", d)
	}
}
