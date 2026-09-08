package replaydiff

import "testing"

// Un compteur d'echec de la couverture qui BAISSE est un gain ; qui MONTE, une perte ; les
// autres mesures gardent la lecture generique (plus = mieux). Mutation : retirer l'inversion
// dans `ajouter` -> les trois premiers cas rougissent.
func TestComparer_CompteursDEchecLusAlEnvers(t *testing.T) {
	num := func(v float64) Mesure { return Mesure{EstNum: true, Num: v} }
	a := Empreinte{Mesures: map[string]Mesure{
		"coverage.objectives.unpublished": num(35),
		"coverage.shots.noSlot":           num(35),
		"coverage.bridge.unnamedLives":    num(0),
		"coverage.bridge.livesNamed":      num(40),
		"objectifs.captures":              num(3),
	}}
	b := Empreinte{Mesures: map[string]Mesure{
		"coverage.objectives.unpublished": num(0),  // echec qui baisse : GAIN
		"coverage.shots.noSlot":           num(50), // echec qui monte : PERTE
		"coverage.bridge.unnamedLives":    num(2),  // echec qui apparait : PERTE
		"coverage.bridge.livesNamed":      num(38), // richesse qui baisse : PERTE (generique)
		"objectifs.captures":              num(4),  // richesse qui monte : GAIN (generique)
	}}
	rap := Comparer(a, b)
	attendu := map[string]string{
		"coverage.objectives.unpublished": SensGain,
		"coverage.shots.noSlot":           SensPerte,
		"coverage.bridge.unnamedLives":    SensPerte,
		"coverage.bridge.livesNamed":      SensPerte,
		"objectifs.captures":              SensGain,
	}
	vus := map[string]string{}
	for _, d := range rap.Differences {
		vus[d.Axe+"."+d.Metrique] = d.Sens
	}
	for cle, sens := range attendu {
		got, ok := vus[cle]
		if !ok {
			// La cle peut etre decoupee autrement par `decouper` : chercher par suffixe.
			for k, s := range vus {
				if len(k) >= len(cle) && k[len(k)-len(cle):] == cle {
					got, ok = s, true
				}
			}
		}
		if !ok {
			t.Fatalf("%s : aucun ecart rapporte (vus : %v)", cle, vus)
		}
		if got != sens {
			t.Errorf("%s : sens %q, attendu %q", cle, got, sens)
		}
	}
	if bil := rap.Bilans["couverture"]; bil.Gains != 1 || bil.Pertes != 3 {
		t.Errorf("bilan couverture : gains=%d pertes=%d, attendu 1 gain / 3 pertes", bil.Gains, bil.Pertes)
	}
}

func TestEstCompteurDEchec_HorsCouvertureJamais(t *testing.T) {
	if estCompteurDEchec("objectifs.unpublished") {
		t.Fatal("hors de l'axe coverage, aucune inversion")
	}
	if estCompteurDEchec("coverage.bridge.deathOffsetRunnerUp") {
		t.Fatal("deathOffsetRunnerUp est un nombre de voix, pas un echec")
	}
	if !estCompteurDEchec("coverage.objectives.unpublished") {
		t.Fatal("unpublished est un compteur d'echec")
	}
}
