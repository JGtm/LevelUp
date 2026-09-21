package killcollector

// credit_progression_test.go — LE CADENCEMENT DE LA PROGRESSION, SANS HORLOGE (lot 5.12).
//
// Le pas se decide sur un COMPTEUR, pas sur une duree : c est ce qui rend ce test une table de
// valeurs et non une attente. Un cadencement par horloge se testerait par un `time.Sleep` ou une
// horloge injectee — les deux paient un cout (lenteur, ou une abstraction de plus) pour une
// information que le compteur porte deja.

import (
	"testing"
	"time"
)

func TestDoitJournaliserProgression(t *testing.T) {
	cas := []struct {
		nom      string
		examines int
		total    int
		veut     bool
	}{
		{"avant le premier pas", 499, 9144, false},
		{"le premier pas", 500, 9144, true},
		{"juste apres un pas", 501, 9144, false},
		{"un pas plus loin", 1000, 9144, true},
		{"le dernier pas plein", 9000, 9144, true},
		// Le dernier match n est jamais un jalon, meme quand il tombe pile sur le pas : le bilan
		// final l annonce deja.
		{"dernier match, pile sur le pas", 9000, 9000, false},
		{"dernier match hors du pas", 9144, 9144, false},
		{"passe vide", 0, 0, false},
		{"premier match d une passe plus courte que le pas", 1, 20, false},
		{"compteur negatif (ne peut pas arriver, ne doit pas journaliser)", -1, 9144, false},
	}
	for _, c := range cas {
		if got := doitJournaliserProgression(c.examines, c.total); got != c.veut {
			t.Errorf("%s: doitJournaliserProgression(%d, %d) = %v, attendu %v",
				c.nom, c.examines, c.total, got, c.veut)
		}
	}
}

// TestNombreDeLignesSurUnePasseEntiere — le journal reste lisible : 18 lignes pour les 9 144
// matchs de la passe de production, pas une par match.
func TestNombreDeLignesSurUnePasseEntiere(t *testing.T) {
	const total = 9144
	lignes := 0
	for examines := 1; examines <= total; examines++ {
		if doitJournaliserProgression(examines, total) {
			lignes++
		}
	}
	if lignes != 18 {
		t.Errorf("passe de %d matchs : %d lignes de progression, attendu 18 (pas de %d)",
			total, lignes, pasDeProgressionCredit)
	}
}

func TestETALineaire(t *testing.T) {
	cas := []struct {
		nom      string
		examines int
		total    int
		ecoule   time.Duration
		veut     time.Duration
	}{
		{"moitie de passe", 500, 1000, 10 * time.Second, 10 * time.Second},
		{"un dixieme", 100, 1000, time.Minute, 9 * time.Minute},
		{"passe finie", 1000, 1000, time.Minute, 0},
		{"rien d examine", 0, 1000, time.Minute, 0},
		{"aucune duree ecoulee", 10, 1000, 0, 0},
		{"passe vide", 0, 0, time.Minute, 0},
	}
	for _, c := range cas {
		if got := etaLineaire(c.examines, c.total, c.ecoule); got != c.veut {
			t.Errorf("%s: etaLineaire(%d, %d, %s) = %s, attendu %s",
				c.nom, c.examines, c.total, c.ecoule, got, c.veut)
		}
	}
}
