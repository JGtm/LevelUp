package coordination

import "testing"

// TestAPortee_LeRayonEstEnVraisMetresLaDistanceEnUnitesDuJeu : la portée du radar (18 m en arène) est
// en vrais mètres, les distances du film sont en unités du jeu (1 unité = 3,048 m). Un coéquipier à
// 7 unités (21,3 m) est HORS d'un radar de 18 m ; l'ancienne comparaison, qui mettait 18 mètres et 18
// unités sur le même plan, le disait dedans.
func TestAPortee_LeRayonEstEnVraisMetresLaDistanceEnUnitesDuJeu(t *testing.T) {
	d := func(v float64) *float64 { return &v }
	cas := []struct {
		nom     string
		unites  *float64
		rayonM  float64
		attendu bool
	}{
		{"7 unites = 21,3 m, hors d'un radar de 18 m", d(7), 18, false},
		{"5 unites = 15,2 m, dans un radar de 18 m", d(5), 18, true},
		{"borne inclusive : 18 m exactement", d(5.905511811023622), 18, true},
		{"7 unites = 21,3 m, dans un radar de 24 m", d(7), 24, true},
		{"distance absente : jamais a portee", nil, 18, false},
	}
	for _, c := range cas {
		if got := APortee(c.unites, c.rayonM); got != c.attendu {
			t.Errorf("%s : APortee = %v, attendu %v", c.nom, got, c.attendu)
		}
	}
}
