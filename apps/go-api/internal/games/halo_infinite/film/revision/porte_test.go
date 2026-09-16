package revision_test

// porte_test.go — LA PORTE DE REGENERATION : ses deux verrous, et ce qu elle ecrit.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/revision"
)

func TestPorteNommeSonDrapeauEtSaVariable(t *testing.T) {
	p := revision.Porte{Nom: "grammar-rev"}
	if got := p.Drapeau(); got != "update-grammar-rev" {
		t.Errorf("drapeau = %q, attendu update-grammar-rev (NOMME : un `-update` generique "+
			"refige une couche cassee sans -run, revue R1 / P1-1)", got)
	}
	if got := p.Variable(); got != "LEVELUP_UPDATE_GRAMMAR_REV" {
		t.Errorf("variable = %q, attendu LEVELUP_UPDATE_GRAMMAR_REV", got)
	}
}

func TestPorteExigeLesDeuxVerrous(t *testing.T) {
	p := revision.Porte{Nom: "facts-rev"}
	for _, cas := range []struct {
		drapeau bool
		env     string
		ouverte bool
		cite    string
	}{
		{drapeau: false, env: "", ouverte: false, cite: "-update-facts-rev"},
		{drapeau: false, env: "1", ouverte: false, cite: "-update-facts-rev"},
		{drapeau: true, env: "", ouverte: false, cite: "LEVELUP_UPDATE_FACTS_REV"},
		{drapeau: true, env: "1", ouverte: true},
	} {
		ouverte, raison := p.Ouverte(cas.drapeau, cas.env)
		if ouverte != cas.ouverte {
			t.Errorf("drapeau=%v env=%q : ouverte=%v, attendu %v", cas.drapeau, cas.env, ouverte, cas.ouverte)
			continue
		}
		if !ouverte && !strings.Contains(raison, cas.cite) {
			t.Errorf("drapeau=%v env=%q : la raison %q ne dit pas ce qui manque (%s)",
				cas.drapeau, cas.env, raison, cas.cite)
		}
		if ouverte && raison != "" {
			t.Errorf("porte ouverte avec une raison de fermeture : %q", raison)
		}
	}
}

// goldenTemporaire pose un golden avec sa prose et ses lignes de donnees.
func goldenTemporaire(t *testing.T, lignes ...string) string {
	t.Helper()
	chemin := filepath.Join(t.TempDir(), "rev.golden")
	contenu := "# PROSE EN TETE.\n#\n#   HISTORIQUE lisible.\n" + strings.Join(lignes, "\n")
	if len(lignes) > 0 {
		contenu += "\n"
	}
	if err := os.WriteFile(chemin, []byte(contenu), 0o600); err != nil {
		t.Fatalf("ecriture : %v", err)
	}
	return chemin
}

func TestPorteAjouteUneLigneEtGardeLaProse(t *testing.T) {
	chemin := goldenTemporaire(t, "couche-2026-09-16\taaa")
	msg, err := revision.Porte{Nom: "facts-rev"}.Reecrire(chemin, "couche-2026-09-17", "bbb")
	if err != nil {
		t.Fatalf("reecriture : %v", err)
	}
	blob, err := os.ReadFile(chemin) //nolint:gosec // chemin temporaire du test
	if err != nil {
		t.Fatalf("relecture : %v", err)
	}
	attendu := "# PROSE EN TETE.\n#\n#   HISTORIQUE lisible.\n" +
		"couche-2026-09-16\taaa\ncouche-2026-09-17\tbbb\n"
	if string(blob) != attendu {
		t.Errorf("golden reecrit :\n%q\nattendu :\n%q", string(blob), attendu)
	}
	// LE MESSAGE EST UN MESSAGE D ECHEC : une porte de regeneration ne rend jamais `ok`
	// (revue R2, C1). Il doit dire quoi relancer.
	if !strings.Contains(msg, "-update-facts-rev") || !strings.Contains(msg, "couche-2026-09-17") {
		t.Errorf("message de reecriture peu diagnostique : %q", msg)
	}
}

func TestPorteRemplaceLaDerniereLigneARevisionConstante(t *testing.T) {
	chemin := goldenTemporaire(t, "couche-2026-09-16\taaa", "couche-2026-09-17\tbbb")
	if _, err := (revision.Porte{Nom: "facts-rev"}).Reecrire(chemin, "couche-2026-09-17", "ccc"); err != nil {
		t.Fatalf("reecriture : %v", err)
	}
	blob, err := os.ReadFile(chemin) //nolint:gosec // chemin temporaire du test
	if err != nil {
		t.Fatalf("relecture : %v", err)
	}
	if strings.Contains(string(blob), "bbb") || !strings.Contains(string(blob), "ccc") {
		t.Errorf("l empreinte d une revision inchangee doit etre REMPLACEE, pas empilee :\n%s", string(blob))
	}
	if strings.Count(string(blob), "couche-2026-09-17\t") != 1 {
		t.Errorf("la revision courante apparait plusieurs fois :\n%s", string(blob))
	}
}

func TestPorteRefuseUnGoldenAbsent(t *testing.T) {
	_, err := revision.Porte{Nom: "facts-rev"}.Reecrire(
		filepath.Join(t.TempDir(), "absent.golden"), "couche-2026-09-17", "bbb")
	if err == nil {
		t.Error("golden absent accepte — un golden est VERSIONNE, son absence est une erreur")
	}
}
