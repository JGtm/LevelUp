package replay

// golden_inputs_budget_test.go — CE QUE LE JEU DE FIXTURES D ENTREES PESE, ET SON PLAFOND.
//
// # POURQUOI UN PLAFOND, ET PAS SEULEMENT UNE MESURE (revue R1 du lot 1.0, exigence R1-6)
//
// Les huit `inputs_*.bin.gz` sont VERSIONNES : chaque montee de la magie du codec en depose un
// jeu complet de plus dans l historique du depot. Une taille qu on se contente de mesurer derive
// d un lot a l autre sans que personne ne la decide — le lot 1.0 vient d en faire la
// demonstration, en passant de 10 323 769 a 11 044 446 octets compresses (+7,0 %) sans qu aucune
// porte ne sonne. Un plafond force la decision (porter moins de canaux, moins de builds) le jour
// ou il est atteint, au lieu de la decouvrir dans un `git clone` de plus en plus long.
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
// POSE LE 2026-09-14 (lot 1.0) A 12 MIO (12 582 912 octets). UNE SEULE MESURE FAIT FOI, celle
// que ce test lit sur le disque : 11 044 446 octets compresses pour les huit fixtures, soit
// 10,53 Mio. Il reste 1 538 466 octets libres, soit 12 % du plafond. (Le commentaire d origine
// citait TROIS totaux differents pour une seule mesure — un d avant regeneration, un du plan, un
// mesure : revue R2, constat R2-2.)
//
// HISTORIQUE, CHAQUE CHIFFRE AVEC SA BASE : 10 849 119 o a l origine du jeu par build
// (lot 0.A.2) ; 18 656 453 o a l etape flottants du lot 0.D.3 bis, revenus a 10 337 463 o en
// quanta ; 10 323 769 o a la cloture de 0.D ; 11 044 446 o depuis que le fixture porte les six
// canaux qui manquaient et le roster de la feuille (lot 1.0). Soit +7,0 % contre la cloture de
// 0.D, et +1,8 % contre le jeu d origine.
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
