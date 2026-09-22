package main

// cmd_backfill_killsource_ouvriers_test.go — LE PLAFOND MEMOIRE REFUSE AVANT D OUVRIR (5.24.2).

import (
	"strings"
	"testing"

	"levelup/go-api/internal/sync/killcollector"
)

// TestVerifierLesOuvriers — le refus porte le CHIFFRE, pas seulement le verdict.
func TestVerifierLesOuvriers(t *testing.T) {
	max := killcollector.OuvriersMaximum()
	if max < 2 {
		t.Fatalf("le plafond n autorise que %d ouvrier(s) : la mesure 5.24.1 (pic %d Mio) ou le "+
			"plafond (%d Mio) a change — relire `collector_ouvriers.go` avant de toucher a ce test",
			max, killcollector.PicMemoireParFilm>>20, killcollector.PlafondMemoireDeLaPasse>>20)
	}
	if killcollector.OuvriersParDefaut > max {
		t.Errorf("le defaut (%d) depasse le plafond (%d) : la commande refuserait son propre defaut",
			killcollector.OuvriersParDefaut, max)
	}
	for _, n := range []int{1, 2, max} {
		if err := verifierLesOuvriers(n); err != nil {
			t.Errorf("--workers %d refuse : %v", n, err)
		}
	}
	for _, n := range []int{0, -1} {
		if err := verifierLesOuvriers(n); err == nil {
			t.Errorf("--workers %d accepte : il en faut au moins un", n)
		}
	}
	err := verifierLesOuvriers(max + 1)
	if err == nil {
		t.Fatalf("--workers %d accepte alors qu il depasse le plafond memoire", max+1)
	}
	// LE MESSAGE DOIT PORTER LA MESURE : un plafond qui dit seulement « trop » oblige a relire le
	// code pour savoir pourquoi, et c est exactement ce que ce depot refuse d imposer.
	for _, attendu := range []string{"422 Mio", "4096 Mio", "1c4c63c2"} {
		if !strings.Contains(err.Error(), attendu) {
			t.Errorf("le refus ne cite pas %q : %v", attendu, err)
		}
	}
}

// TestValiderLesOptions — les incompatibilites de drapeaux, toutes au meme endroit.
func TestValiderLesOptions(t *testing.T) {
	cas := []struct {
		nom    string
		o      killsourceOptions
		refuse string
	}{
		{"defaut", killsourceOptions{workers: killcollector.OuvriersParDefaut}, ""},
		{"films-only et credit-only", killsourceOptions{workers: 1, filmsOnly: true, creditOnly: true}, "s excluent"},
		{"online sans gamertag", killsourceOptions{workers: 1, online: true}, "--online exige --gamertag"},
		{"gamertag sans online", killsourceOptions{workers: 1, gamertag: "JGtm"}, "n a de sens qu avec --online"},
		{"workers hors plafond", killsourceOptions{workers: 99}, "au maximum"},
	}
	for _, c := range cas {
		err := validerLesOptions(c.o)
		if c.refuse == "" {
			if err != nil {
				t.Errorf("%s : refuse alors qu il devrait passer : %v", c.nom, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), c.refuse) {
			t.Errorf("%s : attendu un refus contenant %q, got %v", c.nom, c.refuse, err)
		}
	}
}
