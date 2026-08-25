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

// TestRackDuCanevasEcarte — le canevas Forge de Vagabond range 25 objectifs de tous les
// modes (20 + les 5 collines de KOTH depuis le lot C-ter volet 2) sur 8,2 m alors que ses
// objets couvrent 356 m. Il doit être écarté.
func TestRackDuCanevasEcarte(t *testing.T) {
	v := varianteFixture(t, "vagabond_fo08_wetland.mvar")
	if n := len(v.Objectives()); n != 25 {
		t.Fatalf("fixture inattendue : %d objectifs, attendu 25 (20 + 5 collines de rack)", n)
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

// variantePourGarde construit une variante SYNTHÉTIQUE : deux objets de décor qui fixent
// l'emprise du canevas, et deux objectifs de Bastion qui fixent celle des objectifs. C'est
// tout ce que le garde-fou regarde, et cela permet de rejouer en CI une mesure faite sur des
// `.mvar` de 1 Mo non versionnables.
func variantePourGarde(empriseCanevas, empriseObjectifs float64) *mapvar.Variant {
	zone := []int32{mapvar.LabelHash("strongholds_include"), mapvar.LabelHash("strongholds_zone")}
	return &mapvar.Variant{Objects: []mapvar.Object{
		{Index: 0, Pos: mapvar.Vec3{}},
		{Index: 1, Pos: mapvar.Vec3{X: empriseCanevas}},
		{Index: 2, Pos: mapvar.Vec3{}, Labels: zone},
		{Index: 3, Pos: mapvar.Vec3{X: empriseObjectifs}, Labels: zone},
	}}
}

// TestGrandCanevasNEcartePasLaCarteJouee — LE témoin du plancher absolu, aux chiffres
// mesurés sur des variantes réelles. Il porte deux démonstrations à la fois.
//
// 1. Empyrean (asset d035fc3e, 2026-08-20) : les deux variantes de cette carte ont un
// rapport d'emprise à 0,13 point l'une de l'autre — 3,609 % pour la carte JOUÉE, 3,735 %
// pour le rack — et pourtant l'une doit être retenue et l'autre écartée. Aucun réglage
// d'un seuil RELATIF ne les sépare : seule l'emprise absolue le fait.
//
// 2. Le canevas ROGNÉ (Dynasty cfd90b63 et Kaiketsu 98a83f87, 2026-08-25) : MÊME rack que
// celui d'Empyrean (13,30 m), mais sur un canevas de 34,40 m et non de 356 m. Un critère
// relatif à 5 % y placerait son seuil à 1,72 m et déclarerait ce rack POSÉ — c'est
// exactement ce qui faisait retenir 25 objectifs de rack contre les 13 et 16 objectifs des
// cartes jouées. Ces deux lignes sont la recette de non-retour du critère relatif.
//
// Mutation qui doit le faire rougir : remplacer le plancher absolu par un rapport à
// l'emprise du canevas, sous quelque seuil que ce soit (les lignes « canevas rogné »
// tombent) ; ou retirer le plancher (les lignes Empyrean et cliffside tombent).
func TestGrandCanevasNEcartePasLaCarteJouee(t *testing.T) {
	cas := []struct {
		nom                        string
		empriseCanevas, empriseObj float64
		wantParked                 bool
	}{
		{"empyrean map.mvar (carte jouée sur canevas Forge)", 1061.64, 38.32, false},
		{"empyrean fo11_blank.mvar (rack du canevas)", 356.10, 13.30, true},
		{"vagabond fo08_wetland.mvar (rack, mesure 2026-08-08)", 356.10, 8.20, true},
		{"cliffside_map (plus petite emprise réellement posée du catalogue)", 5384, 21.21, false},
		{"dynasty fo08_wetland.mvar (rack sur canevas ROGNÉ)", 34.40, 13.30, true},
		{"kaiketsu fo05_desert.mvar (rack sur canevas ROGNÉ)", 34.40, 13.30, true},
	}
	for _, c := range cas {
		got := isParkedPalette(variantePourGarde(c.empriseCanevas, c.empriseObj))
		if got != c.wantParked {
			t.Errorf("%s : isParkedPalette = %v, attendu %v (objectifs %.2f m sur un canevas de %.2f m, soit %.3f %%)",
				c.nom, got, c.wantParked, c.empriseObj, c.empriseCanevas,
				100*c.empriseObj/c.empriseCanevas)
		}
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
