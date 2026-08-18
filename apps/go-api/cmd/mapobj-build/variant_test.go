// cmd/mapobj-build — variant_test.go : recette du choix de variante.
//
// Le témoin est REEL et il est décisif : les deux `.mvar` de Vagabond (asset 105f5d84)
// sont exactement le cas où le critère « le plus d'objectifs » se trompe. Le rack du
// canevas en déclare 20, la carte jouée 4. Une mutation qui rendrait le rack acceptable
// fait tomber ce test.
//
// FIXTURES — `.ai/V7.5/dumps/mapvar/` n'est pas versionné (77 Mo de variantes
// re-téléchargeables). Deux exceptions y sont suivies parce qu'elles sont des témoins de
// test : les deux `.mvar` de Cliffhanger et `vagabond_fo08_wetland.mvar` (17 Ko), le rack
// qui porte la démonstration. `vagabond_map.mvar` (882 Ko) reste hors dépôt : le test qui
// s'en sert se déclare absent plutôt que de passer en silence. Pour le remettre :
//
//	go run ./cmd/mapobj-build --player <GT> --dry-run \
//	  --save-mvar <dossier> --map-id 105f5d84-8de1-4908-af3a-1c4f3bf9d642
//
// puis copier <dossier>/105f5d84-.../map.mvar en vagabond_map.mvar.
package main

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// varianteFixture parse un .mvar de référence, ou saute le test s'il n'est pas là.
func varianteFixture(t *testing.T, nom string) *mapvar.Variant {
	t.Helper()
	src := filepath.Join("..", "..", "..", "..", ".ai", "V7.5", "dumps", "mapvar", nom)
	buf, err := os.ReadFile(src)
	if err != nil {
		t.Skipf("fixture %s absente (%v)", src, err)
	}
	v, err := mapvar.Parse(buf)
	if err != nil {
		t.Fatalf("parse %s: %v", nom, err)
	}
	return v
}

// TestRackDuCanevasEcarte — le canevas Forge de Vagabond range 20 objectifs de tous les
// modes sur 8,2 m alors que ses objets couvrent 356 m. Il doit être écarté.
func TestRackDuCanevasEcarte(t *testing.T) {
	v := varianteFixture(t, "vagabond_fo08_wetland.mvar")
	if n := len(v.Objectives()); n != 20 {
		t.Fatalf("fixture inattendue : %d objectifs, attendu 20", n)
	}
	if !isParkedPalette(v) {
		t.Error("le rack du canevas est accepté comme placement — la carte publierait " +
			"trois zones de Bastion de 1 m de rayon à 2 m l'une de l'autre")
	}
}

// TestCarteJoueeRetenue — la carte réellement bâtie n'a que 4 objectifs, elle doit
// néanmoins être retenue : c'est elle qui porte les vraies zones de Bastion.
func TestCarteJoueeRetenue(t *testing.T) {
	v := varianteFixture(t, "vagabond_map.mvar")
	if isParkedPalette(v) {
		t.Fatal("la carte jouée est écartée comme rack")
	}
	zones := 0
	for _, o := range v.Objectives() {
		if o.Role == mapvar.RoleStrongholdZone {
			zones++
		}
	}
	if zones != 3 {
		t.Errorf("zones de Bastion = %d, attendu 3 (oracle du relevé terrain Vagabond)", zones)
	}
}

// TestNomDeMvarPartageRefuseLeRepliAPlat — deux cartes exposent un fichier `map.mvar`
// (Vagabond et une variante de Highpower). À plat, elles liraient le MÊME fichier et le
// catalogue publierait les zones de l'une sous le nom de l'autre. Le repli doit être
// refusé, et le sous-dossier par map_id accepté.
//
// Mutation qui doit le faire rougir : rendre `filepath.Join(dir, mvarFile)` sans regarder
// `shared`.
func TestNomDeMvarPartageRefuseLeRepliAPlat(t *testing.T) {
	dir := t.TempDir()
	partages := map[string]int{"map.mvar": 2, "catalyst.mvar": 1}

	if _, err := mvarPath(dir, "carte-a", "map.mvar", partages); err == nil {
		t.Error("nom partagé accepté à plat — une carte servirait les objectifs d'une autre")
	}
	if _, err := mvarPath(dir, "carte-a", "catalyst.mvar", partages); err != nil {
		t.Errorf("nom unique refusé à plat: %v", err)
	}

	perMap := filepath.Join(dir, "carte-a")
	if err := os.MkdirAll(perMap, 0o755); err != nil {
		t.Fatalf("préparation: %v", err)
	}
	if err := os.WriteFile(filepath.Join(perMap, "map.mvar"), []byte("x"), 0o644); err != nil {
		t.Fatalf("préparation: %v", err)
	}
	got, err := mvarPath(dir, "carte-a", "map.mvar", partages)
	if err != nil {
		t.Fatalf("sous-dossier par map_id refusé: %v", err)
	}
	if got != filepath.Join(perMap, "map.mvar") {
		t.Errorf("chemin = %s, attendu le sous-dossier par map_id", got)
	}
}

// TestCliffhangerNonEcarte — témoin de non-régression sur une carte non-Forge : ses deux
// variantes portent les mêmes 14 objectifs étalés sur 50 m, aucune ne doit être écartée.
func TestCliffhangerNonEcarte(t *testing.T) {
	for _, nom := range []string{"cliffhanger_map.mvar", "cliffhanger_ridgeline.mvar"} {
		if isParkedPalette(varianteFixture(t, nom)) {
			t.Errorf("%s écartée à tort", nom)
		}
	}
}

// TestDumpPublieLesLabelsNonResolusEtLInstance — le dump de diagnostic doit montrer les
// labels que la table ne sait PAS nommer (en hash brut), pas seulement ceux qu'elle sait :
// c'est sur eux qu'un inventaire de mode se fait (lot C-ter volet 2). Il porte aussi
// l'instance_id, clé candidate du catalogue.
//
// Mutation qui doit le faire rougir : retirer le `continue` et l'append d'Unresolved dans
// dumpedObjectsOf.
func TestDumpPublieLesLabelsNonResolusEtLInstance(t *testing.T) {
	v := varianteFixture(t, "cliffhanger_map.mvar")
	want := 0
	for _, n := range v.UnresolvedLabels() {
		want += n
	}
	if want == 0 {
		t.Skip("la fixture ne porte aucun label non résolu — le témoin ne dit rien")
	}
	got, named := 0, 0
	for _, d := range dumpedObjectsOf(v) {
		got += len(d.Unresolved)
		named += len(d.Labels)
	}
	if got != want {
		t.Errorf("labels non résolus publiés = %d, la variante en porte %d", got, want)
	}
	if named == 0 {
		t.Error("aucun label résolu publié — le dump a perdu les noms")
	}
	if len(dumpedObjectsOf(v)) != len(v.Objects) {
		t.Errorf("objets publiés = %d, la variante en a %d", len(dumpedObjectsOf(v)), len(v.Objects))
	}
}
