package briques

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

// racineDepot rend la racine du dépôt : deux niveaux au-dessus de `apps/go-api`.
func racineDepot() (string, error) {
	api, err := racineModuleAPI()
	if err != nil {
		return "", err
	}
	return filepath.Dir(filepath.Dir(api)), nil
}

// cloneDeBrique pose, hors du dépôt, un faux clone de la brique du décodeur (un `go.mod` seul) et un
// espace de travail Go qui le lie à apps/go-api ; rend le dossier du clone et les variables d'environnement
// qui désignent l'espace de travail.
func cloneDeBrique(t *testing.T) (clone string, env []string) {
	t.Helper()
	api, err := racineModuleAPI()
	if err != nil {
		t.Fatal(err)
	}
	brut, err := os.ReadFile(filepath.Join(api, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	var version string
	for _, ligne := range strings.Split(string(brut), "\n") {
		if champs := strings.Fields(ligne); len(champs) == 2 && champs[0] == "go" {
			version = champs[1]
			break
		}
	}
	if version == "" {
		t.Fatal("ligne go introuvable dans apps/go-api/go.mod")
	}
	module, _ := Module("film")
	racine := t.TempDir()
	clone = filepath.Join(racine, "clone-decodeur")
	if err := os.Mkdir(clone, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(clone, "go.mod"), []byte("module "+module+"\n\ngo "+version+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	travail := filepath.Join(racine, "test.work")
	contenu := "go " + version + "\n\nuse (\n\t" + strconv.Quote(filepath.ToSlash(api)) + "\n\t" + strconv.Quote(filepath.ToSlash(clone)) + "\n)\n"
	if err := os.WriteFile(travail, []byte(contenu), 0o600); err != nil {
		t.Fatal(err)
	}
	return clone, []string{"GOWORK=" + travail}
}

// TestModifiableAccepteUnCloneHorsDepotLieParGOWORK : avec un espace de travail qui lie un clone situé
// hors du dépôt, la brique se résout vers ce clone et [controlerModifiable] l'accepte ; les autres
// briques, toujours dans le cache de modules, restent refusées.
func TestModifiableAccepteUnCloneHorsDepotLieParGOWORK(t *testing.T) {
	clone, env := cloneDeBrique(t)
	api, err := racineModuleAPI()
	if err != nil {
		t.Fatal(err)
	}
	dossiers, err := resoudreAvec(api, env)
	if err != nil {
		t.Fatal(err)
	}
	cache, err := cacheDeModulesAvec(api, env)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := resoudreLiens(dossiers["film"]), resoudreLiens(clone); got != want {
		t.Fatalf("film résolu vers %s, attendu le clone %s", got, want)
	}
	if err := controlerModifiable("film", dossiers["film"], cache); err != nil {
		t.Errorf("clone hors dépôt lié par GOWORK refusé : %v", err)
	}
	if err := controlerModifiable("himap", dossiers["himap"], cache); err == nil {
		t.Error("himap, résolue vers le cache de modules, devrait être refusée")
	}
}

// TestModifiableRefuseLeCacheDeModules : sans espace de travail (GOWORK=off) toute brique est résolue vers
// le cache de modules ; l'erreur le dit et désigne le script d'espace de travail.
func TestModifiableRefuseLeCacheDeModules(t *testing.T) {
	api, err := racineModuleAPI()
	if err != nil {
		t.Fatal(err)
	}
	env := []string{"GOWORK=off"}
	dossiers, err := resoudreAvec(api, env)
	if err != nil {
		t.Fatal(err)
	}
	cache, err := cacheDeModulesAvec(api, env)
	if err != nil {
		t.Fatal(err)
	}
	for _, nom := range Noms() {
		err := controlerModifiable(nom, dossiers[nom], cache)
		if err == nil {
			t.Errorf("%s (%s) est dans le cache de modules et devrait être refusée", nom, dossiers[nom])
			continue
		}
		for _, attendu := range []string{"cache de modules", "lecture seule", "scripts/briques-workspace.sh", "GOWORK"} {
			if !strings.Contains(err.Error(), attendu) {
				t.Errorf("message pour %s : %q ne contient pas %q", nom, err, attendu)
			}
		}
	}
}

// TestControlerModifiableJugeLeChemin : la comparaison au cache respecte la frontière d'un nom de dossier
// et ne dépend pas du dépôt.
func TestControlerModifiableJugeLeChemin(t *testing.T) {
	racine := t.TempDir()
	cache := filepath.Join(racine, "pkg", "mod")
	for _, c := range []struct {
		nom, dir string
		refuse   bool
	}{
		{"dans le cache", filepath.Join(cache, "github.com", "x@v1.0.0"), true},
		{"le cache lui-même", cache, true},
		{"frère au préfixe commun", cache + "-clone", false},
		{"clone ailleurs", filepath.Join(racine, "clone"), false},
	} {
		if err := controlerModifiable("film", c.dir, cache); (err != nil) != c.refuse {
			t.Errorf("%s : refus = %v, attendu %v (err %v)", c.nom, err != nil, c.refuse, err)
		}
	}
}

// TestModifiableBriqueInconnue : une brique hors table n'est jamais jugée modifiable.
func TestModifiableBriqueInconnue(t *testing.T) {
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
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Path}}\t{{.Dir}}")
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
	depot, err := racineDepot()
	if err != nil {
		t.Fatal(err)
	}
	depotAvecSeparateur := filepath.Clean(depot) + string(filepath.Separator)
	api := filepath.Clean(racine)
	for _, ligne := range strings.Split(strings.TrimSpace(string(sortie)), "\n") {
		chemin, dir, ok := strings.Cut(strings.TrimRight(ligne, "\r"), "\t")
		// Un module du dépôt autre que l'application est une brique.
		dir = filepath.Clean(dir)
		if !ok || dir == api || !strings.HasPrefix(dir+string(filepath.Separator), depotAvecSeparateur) {
			continue
		}
		if !connus[chemin] {
			t.Errorf("le module %s est dans l'espace de travail mais absent de la table des briques (briques.go)", chemin)
		}
	}
}

// TestImportEtGardeRailSontInverses : la correspondance chemin de garde-rail <-> chemin d'import est une
// bijection sur les briques, et refuse ce qui n'est pas une brique (frontière d'un chemin comprise).
func TestImportEtGardeRailSontInverses(t *testing.T) {
	for _, rel := range []string{"libs/film", "libs/film/internal/source", "libs/gamefiles/ooz", "libs/himap/himap", "libs/haloapi/halo5"} {
		imp, ok := ImportDeGardeRail(rel)
		if !ok {
			t.Fatalf("ImportDeGardeRail(%s) : brique non reconnue", rel)
		}
		if retour, ok := GardeRailDeImport(imp); !ok || retour != rel {
			t.Errorf("GardeRailDeImport(ImportDeGardeRail(%s)) = %q ok %v", rel, retour, ok)
		}
	}
	film, _ := Module("film")
	for _, imp := range []string{film + "x", "levelup/go-api/internal/briques", "fmt"} {
		if EstImportDeBrique(imp) {
			t.Errorf("EstImportDeBrique(%q) = true, attendu faux", imp)
		}
	}
	if _, ok := ImportDeGardeRail("libs/inconnue/x"); ok {
		t.Error("ImportDeGardeRail(libs/inconnue/x) devrait être refusé")
	}
	if _, ok := ImportDeGardeRail("internal/briques"); ok {
		t.Error("ImportDeGardeRail(internal/briques) devrait être refusé")
	}
}
