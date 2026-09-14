package replay

// golden_inputs_budget_test.go — CE QUE LE JEU DE FIXTURES D ENTREES PESE, ET SON PLAFOND.
//
// # POURQUOI UN PLAFOND, ET PAS SEULEMENT UNE MESURE (revue R1 du lot 1.0, exigence R1-6)
//
// Les huit `inputs_*.bin.gz` sont VERSIONNES : chaque montee de la magie du codec en depose un
// jeu complet de plus dans l historique du depot. Une taille qu on se contente de mesurer derive
// d un lot a l autre sans que personne ne la decide — le lot 1.0 vient d en faire la
// demonstration, en passant de 10,35 a 10,53 Mio (+12 % sur le blob brut) sans qu aucune porte ne
// sonne. Un plafond force la decision (porter moins de canaux, moins de builds) le jour ou il est
// atteint, au lieu de la decouvrir dans un `git clone` de plus en plus long.
//
// LE PLAFOND NE SE RELEVE PAS PAR REFLEXE. Il se releve par DECISION ECRITE, datee, avec la
// raison — exactement comme celui des fixtures de contrat (`contract_fixtures_budget_test.go`).
// Un test qui rougit dit qu une question se pose, pas qu il faut changer sa constante.

import (
	"os"
	"testing"
)

// goldenInputsBudget : LE PLAFOND DU JEU ENTIER, en octets compresses.
//
// POSE LE 2026-09-14 (lot 1.0) A 12 MIO. Mesure du jour : 10,53 Mio pour les huit fixtures
// (11 044 407 o), soit ~14 % de marge. Historique : 10,35 Mio a l origine du jeu par build (lot 0.A.2), 17,79 Mio a
// l etape flottants du lot 0.D.3 bis (revenue a 9,86 Mio en quanta), 10,32 Mio a la cloture de
// 0.D, 10,53 Mio depuis que le fixture porte les six canaux qui manquaient (lot 1.0.2).
//
// LA MARGE EST VOULUE ETROITE : un canal de plus se voit. Elle n est PAS la pour absorber un
// build supplementaire — un neuvieme build est precisement la decision que ce plafond existe
// pour rendre explicite.
const goldenInputsBudget = 12 << 20

// TestGoldenInputsTiennentDansLeBudget : le jeu entier tient-il sous le plafond, et combien pese
// chaque fixture ?
//
// LA MESURE EST LA POUR ETRE LUE (`go test -run GoldenInputsTiennent -v`) : c est elle que le
// plan consigne au journal des gates.
func TestGoldenInputsTiennentDansLeBudget(t *testing.T) {
	builds := goldenBuilds()
	if len(builds) == 0 {
		t.Fatal("aucun build a la table : le budget ne mesure plus rien")
	}
	total := 0
	for _, b := range builds {
		info, err := os.Stat(b.inputsPath())
		if err != nil {
			t.Fatalf("fixture d entrees absent (%s) : %v — regenerer avec -update et "+
				miniFilmCacheEnv, b.inputsPath(), err)
		}
		total += int(info.Size())
		t.Logf("%-10s %-10s %9d octets", b.Build, b.Short8, info.Size())
	}
	t.Logf("TOTAL %d fixture(s) : %d octets (%.2f Mio), plafond %d octets (%.0f Mio)",
		len(builds), total, float64(total)/(1<<20),
		goldenInputsBudget, float64(goldenInputsBudget)/(1<<20))
	if total > goldenInputsBudget {
		t.Errorf("le jeu de fixtures d entrees pese %d octets (%.2f Mio), au-dela du plafond de "+
			"%d octets (%.0f Mio) : couper (moins de canaux portes, moins de builds) ou relever "+
			"le plafond par decision ECRITE et datee, jamais par reflexe",
			total, float64(total)/(1<<20), goldenInputsBudget,
			float64(goldenInputsBudget)/(1<<20))
	}
}
