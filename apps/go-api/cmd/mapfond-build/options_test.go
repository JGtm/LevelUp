package main

import (
	"bytes"
	"context"
	"flag"
	"strings"
	"testing"
)

// Toute option se range sous un geste et un geste ne cite que des options qui existent : une option
// sans geste est une option qu'on ne sait plus expliquer.
func TestChaqueOptionSeRattacheAUnGeste(t *testing.T) {
	var o options
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	declareOptions(fs, &o)

	rangees := map[string]string{}
	for _, g := range gestes {
		for _, nom := range g.options {
			if fs.Lookup(nom) == nil {
				t.Errorf("le geste %q cite -%s, qui n'est pas déclarée", g.titre, nom)
			}
			if autre, deja := rangees[nom]; deja {
				t.Errorf("-%s est rangée sous deux gestes : %q et %q", nom, autre, g.titre)
			}
			rangees[nom] = g.titre
		}
	}
	fs.VisitAll(func(f *flag.Flag) {
		if _, ok := rangees[f.Name]; !ok {
			t.Errorf("-%s n'est rattachée à aucun des quatre gestes", f.Name)
		}
	})
	if len(gestes) != 4 {
		t.Errorf("%d gestes, attendu quatre : cuire, refaire le plan sans recuire, ajouter le relief sans recuire, tableau avant / après", len(gestes))
	}
}

func TestOptionsParDefautCuisentToutesLesCartes(t *testing.T) {
	o, err := litOptions(nil, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if err := o.valide(); err != nil {
		t.Fatalf("la ligne de commande vide est refusée : %v", err)
	}
	if !o.natives || !o.forge || o.planSeul || o.reliefSeul || o.maps != "" || o.rapports.tsv != "" {
		t.Errorf("défauts inattendus : %+v", o)
	}
}

func TestOptionsValideRefuseLesCombinaisonsSansSens(t *testing.T) {
	cas := []struct {
		nom     string
		args    []string
		attendu string // vide = accepté
	}{
		{"cuisson complète", []string{"--maps", "ridgeline", "--style", "encre", "--mpp", "0.1", "--avis", "a.json"}, ""},
		{"plan seul avec sélection et avis", []string{"--plan-seul", "--maps", "ridgeline", "--forge=false", "--avis", "a.json"}, ""},
		{"rapport avec avant", []string{"--rapport-tsv", "r.tsv", "--avant-dir", "d"}, ""},
		{"plan seul et rapport", []string{"--plan-seul", "--rapport-tsv", "r.tsv", "--avant-dir", "d"}, ""},
		{"relief seul avec sélection, avis et rapport", []string{"--relief-seul", "--maps", "sgh_blueprint", "--forge=false", "--avis", "a.json", "--rapport-tsv", "r.tsv"}, ""},
		{"plan seul et relief seul", []string{"--plan-seul", "--relief-seul"}, "s'excluent"},
		{"relief seul et style", []string{"--relief-seul", "--style", "encre"}, "pas à -relief-seul"},
		{"relief seul et mpp", []string{"--relief-seul", "--mpp", "0.1"}, "-mpp"},
		{"plan seul et style", []string{"--plan-seul", "--style", "encre"}, "-style"},
		{"plan seul et mpp", []string{"--plan-seul", "--mpp", "0.1"}, "-mpp"},
		{"avant sans rapport", []string{"--avant-dir", "d"}, "-avant-dir exige -rapport-tsv"},
		{"rien à traiter", []string{"--natives=false", "--forge=false"}, "aucune carte"},
		{"habillage inconnu", []string{"--style", "aquarelle"}, "habillage inconnu"},
	}
	for _, c := range cas {
		o, err := litOptions(c.args, &bytes.Buffer{})
		if err != nil {
			t.Errorf("%s : syntaxe refusée : %v", c.nom, err)
			continue
		}
		err = o.valide()
		switch {
		case c.attendu == "" && err != nil:
			t.Errorf("%s : refusée à tort : %v", c.nom, err)
		case c.attendu != "" && (err == nil || !strings.Contains(err.Error(), c.attendu)):
			t.Errorf("%s : erreur = %v, attendu une erreur contenant %q", c.nom, err, c.attendu)
		}
	}
}

// Une combinaison refusée l'est avant toute préparation : aucun dossier n'est lu ni créé.
func TestExecuteRefuseAvantDePreparer(t *testing.T) {
	var sortie bytes.Buffer
	if code := execute(context.Background(), []string{"--plan-seul", "--style", "encre", "--out-dir", t.TempDir() + "/nouveau"}, &sortie); code != 1 {
		t.Errorf("code = %d, attendu 1", code)
	}
	if code := execute(context.Background(), []string{"--inconnue"}, &sortie); code != 2 {
		t.Errorf("syntaxe : code = %d, attendu 2", code)
	}
	if code := execute(context.Background(), []string{"-h"}, &sortie); code != 0 {
		t.Errorf("aide : code = %d, attendu 0", code)
	}
}

func TestAideListeLesQuatreGestes(t *testing.T) {
	var sortie bytes.Buffer
	if _, err := litOptions([]string{"-h"}, &sortie); err == nil {
		t.Fatal("-h doit rendre flag.ErrHelp")
	}
	for _, g := range gestes {
		if !strings.Contains(sortie.String(), g.titre) {
			t.Errorf("l'aide ne montre pas le geste %q", g.titre)
		}
	}
	for _, nom := range []string{"-plan-seul", "-relief-seul", "-rapport-tsv", "-avant-dir", "-avis"} {
		if !strings.Contains(sortie.String(), nom) {
			t.Errorf("l'aide ne montre pas %s", nom)
		}
	}
}
