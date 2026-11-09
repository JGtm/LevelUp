package filmrun

// runner_commande_test.go — LA COMMANDE DE L'ENFANT, RELUE TELLE QU'ELLE PARTIRAIT.
//
// Le lanceur re-execute son propre binaire avec un environnement ou les racines sont imposees
// (cf. [withRoots]). Ce test construit la commande des trois formes de lanceur sans la lancer et
// la decrit ligne a ligne : binaire, arguments, environnement dans l'ordre exact, attributs de
// processus. Quand la variable [envTemoinCommande] nomme un fichier, la description y est
// ecrite : c'est le temoin qu'un deplacement du lanceur compare octet pour octet.

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
)

// envTemoinCommande : chemin du fichier ou ecrire la description des commandes (vide = rien
// d'ecrit, le test ne fait que ses controles).
const envTemoinCommande = "FILMRUN_TEMOIN_COMMANDE"

func TestCommandeDeLEnfantDecriteOctetPourOctet(t *testing.T) {
	t.Setenv(EnvRepoRoot, "ancienne-racine")
	t.Setenv("levelup_data_dir", "donnees-en-minuscules")
	t.Setenv(EnvLogsDir, "anciens-journaux")
	t.Setenv("TEMOIN_AUTRE", "conserve")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"-enfant", "abc12345", "--drapeau=1"}
	var b strings.Builder
	for _, c := range lanceursDuTemoin(t) {
		cmd := c.r.command(context.Background(), args)
		if cmd.Path != exe || cmd.Args[0] != exe {
			t.Fatalf("%s : binaire %q, attendu le binaire courant %q", c.nom, cmd.Path, exe)
		}
		fmt.Fprintf(&b, "== %s\nargs=%q\n", c.nom, cmd.Args[1:])
		for _, kv := range cmd.Env {
			fmt.Fprintf(&b, "env=%s\n", kv)
		}
		fmt.Fprintf(&b, "sysproc=%+v\n", cmd.SysProcAttr)
	}
	if !strings.Contains(b.String(), "env=TEMOIN_AUTRE=conserve\n") {
		t.Fatal("une variable non imposee n'est pas transmise a l'enfant")
	}
	if p := os.Getenv(envTemoinCommande); p != "" {
		if err := os.WriteFile(p, []byte(b.String()), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// lanceurDuTemoin : un lanceur nomme pour la description.
type lanceurDuTemoin struct {
	nom string
	r   *Runner
}

func lanceursDuTemoin(t *testing.T) []lanceurDuTemoin {
	t.Helper()
	simple, err := NewRunner("racine-depot", nil)
	if err != nil {
		t.Fatal(err)
	}
	partiel, err := NewRunnerRoots(Roots{Resources: "ressources", Data: "donnees"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	complet, err := NewRunnerRoots(Roots{Resources: "ressources", Data: "donnees", Logs: "journaux"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return []lanceurDuTemoin{{"NewRunner", simple}, {"NewRunnerRoots partiel", partiel},
		{"NewRunnerRoots complet", complet}}
}
