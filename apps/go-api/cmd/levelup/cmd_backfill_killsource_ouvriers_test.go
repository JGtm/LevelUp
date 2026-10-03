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

// TestValiderLesOptions_OnlineRefuseDesOuvriers — le constat de revue du 2026-09-22 : la
// commande ACCEPTAIT `--online --workers 6`, le validait contre le plafond memoire, puis
// decodait en serie sans rien dire (la passe en ligne n a jamais lu `o.workers`). L utilisateur
// croyait tourner a six ouvriers et tournait a un.
//
// ⚠ ET ELLE NE DOIT PAS REFUSER `--online` TOUT COURT : le defaut vaut 3, donc le refus ne peut
// porter que sur un `--workers` REELLEMENT ECRIT sur la ligne de commande.
func TestValiderLesOptions_OnlineRefuseDesOuvriers(t *testing.T) {
	enLigne := killsourceOptions{online: true, gamertag: "JGtm", workers: killcollector.OuvriersParDefaut}
	if err := validerLesOptions(enLigne); err != nil {
		t.Errorf("`--online --gamertag JGtm` refuse alors que --workers n a pas ete demande : %v", err)
	}
	demande := enLigne
	demande.workersExplicite = true
	demande.workers = 6
	err := validerLesOptions(demande)
	if err == nil {
		t.Fatal("`--online --workers 6` accepte : la passe en ligne decode en serie et ignore " +
			"la valeur — l utilisateur croirait tourner a six ouvriers")
	}
	for _, attendu := range []string{"EN SERIE", "--rps", "--workers 1"} {
		if !strings.Contains(err.Error(), attendu) {
			t.Errorf("le refus ne dit pas %q : %v", attendu, err)
		}
	}
	// `--online --workers 1` reste legitime : c est ce que la passe fait deja.
	unSeul := demande
	unSeul.workers = 1
	if err := validerLesOptions(unSeul); err != nil {
		t.Errorf("`--online --workers 1` refuse alors qu il decrit le comportement reel : %v", err)
	}
}
