package replay

// map_background_index_test.go — l'index inverse identité de carte -> clé de fond publiée.
//
// Ces tests posent des sidecars SYNTHÉTIQUES dans un répertoire temporaire : ils décrivent le
// contrat de l'index, pas l'état du catalogue livré. Le catalogue réel a son propre garde-rail
// (map_background_index_catalogue_test.go).

import (
	"os"
	"path/filepath"
	"testing"
)

// sidecarFond écrit un sidecar minimal mais CONFORME au schéma de production : un test qui
// invente sa propre forme ne dirait rien du fichier réel.
func sidecarFond(t *testing.T, dir, cle string, noms ...string) {
	t.Helper()
	blob := `{"schemaVersion":1,"module":"` + cle + `","mapNames":[`
	for i, n := range noms {
		if i > 0 {
			blob += ","
		}
		blob += `"` + n + `"`
	}
	blob += `],"image":"` + cle + `.png","source":"test","generatedAt":"2026-08-27T10:00:00Z",` +
		`"style":"encre","calibration":{"metersPerPixel":0.05,"originX":-1,"originY":1,` +
		`"widthPx":10,"heightPx":10,"convention":"test"},"stats":{"anchors":1}}`
	if err := os.WriteFile(filepath.Join(dir, cle+".json"), []byte(blob), 0o644); err != nil {
		t.Fatalf("écriture sidecar %s : %v", cle, err)
	}
}

func TestNormalizeMapIdentity(t *testing.T) {
	cas := []struct {
		entree, veut string
	}{
		// Le nom affiché et le nom de module du catalogue d'objectifs tombent sur LA MÊME clé :
		// c'est tout l'intérêt de la normalisation.
		{"Oasis Sentry Defense", "oasis_sentry_defense"},
		{"oasis_sentry_defense_map", "oasis_sentry_defense"},
		{"Salvation", "salvation"},
		{"salvation_map", "salvation"},
		// Les variantes de playlist restent DISTINCTES : les sidecars les déclarent
		// explicitement, les fondre ne servirait qu'à fabriquer des ambiguïtés.
		{"Aquarius - Ranked", "aquarius_-_ranked"},
		{"aquarius_-_ranked_map", "aquarius_-_ranked"},
		{"Oasis Heavies", "oasis_heavies"},
		// Blancs multiples, tabulations, bords.
		{"  Rat's   Nest  ", "rat's_nest"},
		{"\tHigh Ground\n", "high_ground"},
		{"", ""},
		// « map » n'est pas un suffixe ici : il n'y a rien devant à garder.
		{"map", "map"},
		{"MAP", "map"},
	}
	for _, c := range cas {
		if got := NormalizeMapIdentity(c.entree); got != c.veut {
			t.Errorf("NormalizeMapIdentity(%q) = %q, veut %q", c.entree, got, c.veut)
		}
	}
}

func TestIndexFondsResoutNomEtModule(t *testing.T) {
	dir := t.TempDir()
	sidecarFond(t, dir, "btb_exiled", "oasis", "oasis_map", "oasis_sentry_defense_map", "oasis_firefight_map")
	sidecarFond(t, dir, "cd08bc7a-7ba5-4502-be87-c58b641fc94d", "Salvation", "salvation_map")

	idx, err := BuildMapBackgroundIndex(dir)
	if err != nil {
		t.Fatalf("BuildMapBackgroundIndex : %v", err)
	}
	if idx.Cles() != 2 {
		t.Fatalf("Cles() = %d, veut 2", idx.Cles())
	}
	cas := map[string]string{
		// Nom affiché d'une VARIANTE : c'est le cas que le catalogue de bornes ne couvrait pas.
		"Oasis Sentry Defense": "btb_exiled",
		"Oasis Firefight":      "btb_exiled",
		"Oasis":                "btb_exiled",
		// La clé elle-même est une identité.
		"btb_exiled": "btb_exiled",
		// LA DÉRIVE D'ASSET : la carte a été jouée sous un map_id mort, son nom la retrouve.
		"Salvation":     "cd08bc7a-7ba5-4502-be87-c58b641fc94d",
		"salvation_map": "cd08bc7a-7ba5-4502-be87-c58b641fc94d",
	}
	for nom, veut := range cas {
		got, ok := idx.Lookup(nom)
		if !ok || got != veut {
			t.Errorf("Lookup(%q) = %q,%v — veut %q,true", nom, got, ok, veut)
		}
	}
	if cle, ok := idx.Lookup("Detachment"); ok {
		t.Errorf("Lookup(\"Detachment\") = %q,true — une carte sans fond ne doit rien rendre", cle)
	}
	if len(idx.Ambigues()) != 0 {
		t.Errorf("Ambigues() = %v, veut vide", idx.Ambigues())
	}
}

// TestIndexFondsEcarteLesAmbiguites : servir le fond d'une AUTRE carte est pire que n'en servir
// aucun. Deux cartes publiées qui portent le même nom ne résolvent donc NI l'une NI l'autre.
func TestIndexFondsEcarteLesAmbiguites(t *testing.T) {
	dir := t.TempDir()
	sidecarFond(t, dir, "aaaa1111-0000-0000-0000-000000000001", "Warehouse", "warehouse_map")
	sidecarFond(t, dir, "bbbb2222-0000-0000-0000-000000000002", "Warehouse", "warehouse_map")
	sidecarFond(t, dir, "cccc3333-0000-0000-0000-000000000003", "Curfew", "curfew_map")

	idx, err := BuildMapBackgroundIndex(dir)
	if err != nil {
		t.Fatalf("BuildMapBackgroundIndex : %v", err)
	}
	if cle, ok := idx.Lookup("Warehouse"); ok {
		t.Errorf("Lookup(\"Warehouse\") = %q,true — une identité ambiguë ne doit RIEN rendre", cle)
	}
	amb := idx.Ambigues()["warehouse"]
	if len(amb) != 2 || amb[0] != "aaaa1111-0000-0000-0000-000000000001" ||
		amb[1] != "bbbb2222-0000-0000-0000-000000000002" {
		t.Errorf("Ambigues()[\"warehouse\"] = %v — veut les deux clés, triées", amb)
	}
	// L'ambiguïté est LOCALE : elle ne prive pas les autres cartes de leur fond.
	if cle, ok := idx.Lookup("Curfew"); !ok || cle != "cccc3333-0000-0000-0000-000000000003" {
		t.Errorf("Lookup(\"Curfew\") = %q,%v — la voisine doit rester résolvable", cle, ok)
	}
}

// TestIndexFondsIgnoreLeModuleGenerique : `map` est le nom de module générique que le catalogue
// d'objectifs donne à plusieurs cartes Forge. L'indexer ferait résoudre vers la première venue.
func TestIndexFondsIgnoreLeModuleGenerique(t *testing.T) {
	dir := t.TempDir()
	sidecarFond(t, dir, "105f5d84-8de1-4908-af3a-1c4f3bf9d642", "Vagabond", "map")

	idx, err := BuildMapBackgroundIndex(dir)
	if err != nil {
		t.Fatalf("BuildMapBackgroundIndex : %v", err)
	}
	if cle, ok := idx.Lookup("map"); ok {
		t.Errorf("Lookup(\"map\") = %q,true — le module générique ne désigne aucune carte", cle)
	}
	if cle, ok := idx.Lookup("Vagabond"); !ok || cle != "105f5d84-8de1-4908-af3a-1c4f3bf9d642" {
		t.Errorf("Lookup(\"Vagabond\") = %q,%v — le vrai nom doit résoudre", cle, ok)
	}
}

// TestIndexFondsSautLesSidecarsAbimes : un fond corrompu ne doit pas priver de fond les autres.
func TestIndexFondsSautLesSidecarsAbimes(t *testing.T) {
	dir := t.TempDir()
	sidecarFond(t, dir, "chasm", "Chasm", "chasm_map")
	if err := os.WriteFile(filepath.Join(dir, "casse.json"), []byte("{pas du json"), 0o644); err != nil {
		t.Fatalf("écriture sidecar cassé : %v", err)
	}
	// Un fichier hors .json (l'image) ne doit être ni lu ni compté.
	if err := os.WriteFile(filepath.Join(dir, "chasm.png"), []byte("\x89PNG"), 0o644); err != nil {
		t.Fatalf("écriture png : %v", err)
	}

	idx, err := BuildMapBackgroundIndex(dir)
	if err != nil {
		t.Fatalf("BuildMapBackgroundIndex : %v", err)
	}
	if idx.Cles() != 1 {
		t.Errorf("Cles() = %d, veut 1 (le sidecar cassé est sauté, le PNG ignoré)", idx.Cles())
	}
	if cle, ok := idx.Lookup("Chasm"); !ok || cle != "chasm" {
		t.Errorf("Lookup(\"Chasm\") = %q,%v — veut chasm,true", cle, ok)
	}
}

func TestIndexFondsRepertoireAbsent(t *testing.T) {
	if _, err := BuildMapBackgroundIndex(filepath.Join(t.TempDir(), "inexistant")); err == nil {
		t.Fatal("BuildMapBackgroundIndex sur un répertoire absent : veut une erreur")
	}
	if _, err := MapBackgroundIndexFor(filepath.Join(t.TempDir(), "inexistant")); err == nil {
		t.Fatal("MapBackgroundIndexFor sur un répertoire absent : veut une erreur")
	}
}

// TestIndexFondsCacheSuitLeDisque : le cache doit rendre le MÊME index tant que rien ne bouge,
// et un index NEUF dès qu'une carte est publiée. Un cache qui survit à une cuisson servirait
// éternellement l'ancien catalogue au serveur de développement.
func TestIndexFondsCacheSuitLeDisque(t *testing.T) {
	dir := t.TempDir()
	sidecarFond(t, dir, "chasm", "Chasm", "chasm_map")

	premier, err := MapBackgroundIndexFor(dir)
	if err != nil {
		t.Fatalf("MapBackgroundIndexFor : %v", err)
	}
	second, err := MapBackgroundIndexFor(dir)
	if err != nil {
		t.Fatalf("MapBackgroundIndexFor (2e) : %v", err)
	}
	if premier != second {
		t.Error("le cache n'a pas resservi l'index alors que le répertoire n'a pas bougé")
	}

	sidecarFond(t, dir, "ridgeline", "Cliffhanger", "cliffhanger_ridgeline")
	apres, err := MapBackgroundIndexFor(dir)
	if err != nil {
		t.Fatalf("MapBackgroundIndexFor (après ajout) : %v", err)
	}
	if apres == premier {
		t.Fatal("le cache a resservi l'ancien index après publication d'une carte")
	}
	if cle, ok := apres.Lookup("Cliffhanger"); !ok || cle != "ridgeline" {
		t.Errorf("Lookup(\"Cliffhanger\") = %q,%v après publication — veut ridgeline,true", cle, ok)
	}
}
