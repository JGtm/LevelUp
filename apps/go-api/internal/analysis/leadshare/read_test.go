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
		t.Error("un camp absent de la passe ne se lit pas : ce n'est pas un zéro")
	}
}

// Un camp muet (jamais marqué) n'a pas de série : la passe le porte comme une ligne de camp à zéro.
// C'est cette ligne, et elle seule, qui rend sa part nulle MESURÉE ; le seul compte des camps muets
// ne suffit pas, il validerait n'importe quel camp absent — même un camp qui n'est pas au match.
func TestShareOfCamp_CampMuetNomme(t *testing.T) {
	muet := lectureDeTest()
	muet.SilentCamps = 1
	muet.LeadMS[2] = 0
	if s, ok := ShareOfCamp(muet, 2); !ok || s.Share != 0 {
		t.Errorf("camp muet nommé : %+v ok=%v, attendu une part nulle MESURÉE", s, ok)
	}
	if _, ok := ShareOfCamp(muet, 5); ok {
		t.Error("avec un camp muet, un camp ABSENT de la passe (hors roster) ne rend pas zéro : il ne se lit pas")
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
