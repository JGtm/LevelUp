package briques

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestChaqueBriqueSeResoutVersSonModule : le dossier rendu porte le `go.mod` du module annoncé.
func TestChaqueBriqueSeResoutVersSonModule(t *testing.T) {
	if len(Noms()) != 4 {
		t.Fatalf("briques connues = %v, attendu quatre", Noms())
	}
	for _, nom := range Noms() {
		dir, err := Dir(nom)
		if err != nil {
			t.Fatalf("Dir(%s) : %v", nom, err)
		}
		brut, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err != nil {
			t.Fatalf("go.mod de %s : %v", nom, err)
		}
		module, _ := Module(nom)
		if !strings.HasPrefix(string(brut), "module "+module+"\n") && !strings.HasPrefix(string(brut), "module "+module+"\r\n") {
			t.Errorf("le go.mod de %s ne déclare pas le module %s", nom, module)
		}
	}
}

// TestBriqueInconnueEstUneErreur : un nom hors table n'est jamais résolu en silence.
func TestBriqueInconnueEstUneErreur(t *testing.T) {
	if _, err := Dir("inexistante"); err == nil {
		t.Fatal("Dir(inexistante) devrait échouer")
	}
	if _, _, err := SurDisque("libs/inexistante/x.go"); err == nil {
		t.Fatal("SurDisque(libs/inexistante/x.go) devrait échouer")
	}
}

// TestSurDisqueEtDeGardeRailSontInverses : la correspondance chemin de garde-rail <-> disque est
// une bijection sur les briques, et laisse les chemins de apps/go-api tels quels.
func TestSurDisqueEtDeGardeRailSontInverses(t *testing.T) {
	for _, rel := range []string{"libs/film", "libs/film/replay/document_chronicle.go", "libs/himap/himap/testdata", "libs/haloapi/halo5"} {
		abs, ok, err := SurDisque(rel)
		if err != nil || !ok {
			t.Fatalf("SurDisque(%s) = ok %v err %v", rel, ok, err)
		}
		retour, ok, err := DeGardeRail(abs)
		if err != nil || !ok || retour != rel {
			t.Errorf("DeGardeRail(SurDisque(%s)) = %q ok %v err %v", rel, retour, ok, err)
		}
	}
	if _, ok, err := SurDisque("internal/briques"); ok || err != nil {
		t.Errorf("SurDisque(internal/briques) = ok %v err %v, attendu ok=false", ok, err)
	}
	api, _ := racineModuleAPI()
	if _, ok, err := DeGardeRail(filepath.Join(api, "internal")); ok || err != nil {
		t.Errorf("DeGardeRail(internal) = ok %v err %v, attendu ok=false", ok, err)
	}
}

// TestModifiableRefuseLeHorsDepot : une brique du dépôt est modifiable ; un dossier hors du dépôt ne
// l'est jamais.
func TestModifiableRefuseLeHorsDepot(t *testing.T) {
	dir, err := Dir("film")
	if err != nil {
		t.Fatal(err)
	}
	depot, err := racineDepot()
	if err != nil {
		t.Fatal(err)
	}
	if rel, errRel := filepath.Rel(depot, dir); errRel == nil && !strings.HasPrefix(rel, "..") {
		if err := Modifiable("film"); err != nil {
			t.Errorf("Modifiable(film) = %v, la brique est dans le dépôt", err)
		}
	}
	if err := Modifiable("inexistante"); err == nil {
		t.Error("Modifiable(inexistante) devrait échouer")
	}
}

// TestToutModuleDeLibsEstDansLaTable : chaque module de l'espace de travail sous `libs/` figure dans la
// table du résolveur. Sans cela une brique neuve ne serait balayée par aucun garde-rail, qui ne voient que
// les briques de [Noms].
func TestToutModuleDeLibsEstDansLaTable(t *testing.T) {
	racine, err := racineModuleAPI()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Path}}")
	cmd.Dir = racine
	sortie, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -m : %v", err)
	}
	connus := map[string]bool{}
	for _, nom := range Noms() {
		m, _ := Module(nom)
		connus[m] = true
	}
	for _, chemin := range strings.Fields(string(sortie)) {
		if !strings.Contains(chemin, "/libs/") {
			continue
		}
		if !connus[chemin] {
			t.Errorf("le module %s est dans l'espace de travail mais absent de la table des briques (briques.go)", chemin)
		}
	}
}
