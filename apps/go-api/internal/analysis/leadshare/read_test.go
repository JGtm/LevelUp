package leadshare

import (
	"math"
	"testing"
)

func lectureDeTest() MatchRead {
	return MatchRead{MeasuredMS: 600000, LeadChanges: 3, LeadMS: map[int]int64{0: 300000, 1: 150000}}
}

func TestShareOfCamp(t *testing.T) {
	s, ok := ShareOfCamp(lectureDeTest(), 0)
	if !ok || math.Abs(s.Share-0.5) > 1e-9 || s.Changes != 3 {
		t.Errorf("camp 0 : %+v ok=%v, attendu 0,5 et 3 retournements", s, ok)
	}
	if s, ok := ShareOfCamp(lectureDeTest(), 1); !ok || math.Abs(s.Share-0.25) > 1e-9 {
		t.Errorf("camp 1 : %+v ok=%v, attendu 0,25", s, ok)
	}
}

func TestShareOfCamp_CampAbsent(t *testing.T) {
	if _, ok := ShareOfCamp(lectureDeTest(), 5); ok {
		t.Error("un camp absent sans camp muet ne se lit pas : ce n'est pas un zéro")
	}
	muet := lectureDeTest()
	muet.SilentCamps = 1
	if s, ok := ShareOfCamp(muet, 5); !ok || s.Share != 0 {
		t.Errorf("camp muet : %+v ok=%v, attendu une part nulle MESURÉE", s, ok)
	}
}

func TestShareOfCamp_DureeNulle(t *testing.T) {
	m := lectureDeTest()
	m.MeasuredMS = 0
	if _, ok := ShareOfCamp(m, 0); ok {
		t.Error("une durée nulle ne se divise pas")
	}
}

func TestAverage_SeulsLesMatchsMesuresCompteront(t *testing.T) {
	avg, n := Average([]float64{0.2, 0.4, 0.9})
	if n != 3 || math.Abs(avg-0.5) > 1e-9 {
		t.Errorf("moyenne = %v sur %d, attendu 0,5 sur 3", avg, n)
	}
	if _, n := Average(nil); n != 0 {
		t.Errorf("aucune part : n = %d, attendu 0", n)
	}
}
