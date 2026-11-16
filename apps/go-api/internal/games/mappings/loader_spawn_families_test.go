package mappings

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/testutil"
)

func spawnFamiliesDuDepot(t *testing.T, slug string) (*SpawnFamilySet, error) {
	t.Helper()
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt introuvable : %v", err)
	}
	return LoadSpawnFamiliesFromFile(filepath.Join(root, "config", "titles", slug, "mappings", "spawn_families.toml"))
}

// Le fichier versionné de Halo Infinite range les variantes du parc dans les familles de la
// mesure de réapparition : Fiesta avant les modes d'objectif, le BTB et Husky Raid hors mesure.
func TestSpawnFamilies_FichierDuDepot(t *testing.T) {
	set, err := spawnFamiliesDuDepot(t, "halo_infinite")
	if err != nil {
		t.Fatalf("le fichier versionné doit charger : %v", err)
	}
	if got := strings.Join(set.Families(), ","); got != "fiesta,drapeau,bases,colline,oddball,assassin" {
		t.Errorf("familles = %s", got)
	}
	cas := []struct{ variante, categorie, attendu string }{
		{"Team Slayer:Arena", "Assassin", "assassin"},
		{"Ranked:Slayer", "Ranked", "assassin"},
		{"Slayer:Arena Super Fiesta", "Super Fiesta", "fiesta"},
		{"Slayer:Arena Fiesta", "Assassin", "fiesta"},
		{"CTF:Arena", "Assassin", "drapeau"},
		{"Strongholds:Arena", "Assassin", "bases"},
		{"KOTH:Arena", "Assassin", "colline"},
		{"Ranked:King of the Hill", "Ranked", "colline"},
		{"Oddball:Arena", "Assassin", "oddball"},
		{"  SLAYER:ARENA ", "", "assassin"},
		{"BTB:CTF", "BTB", ""},
		{"CTF:BTB", "BTB", ""},
		{"Slayer:BTB Heavies", "BTB", ""},
		{"BTB:Fiesta Slayer", "BTB", ""},
		{"Husky Raid:CTF", "Husky Raid", ""},
		{"Slayer:Arena", "Super Husky Raid", ""},
		{"Arena:VIP", "Assassin", ""},
		{"", "", ""},
	}
	for _, c := range cas {
		if got := set.FamilyOf(c.variante, c.categorie); got != c.attendu {
			t.Errorf("FamilyOf(%q, %q) = %q, attendu %q", c.variante, c.categorie, got, c.attendu)
		}
	}
}

// Un titre qui n'a pas de fichier n'a pas de famille : l'absence remonte telle quelle, et un
// ensemble nil ne reconnaît rien (la prévision retombe alors sur la famille « tous »).
func TestSpawnFamilies_TitreSansFamille(t *testing.T) {
	for _, slug := range []string{"halo_5", "synthetic_title_b"} {
		set, err := spawnFamiliesDuDepot(t, slug)
		if !errors.Is(err, fs.ErrNotExist) || set != nil {
			t.Errorf("%s : (%v, %v), attendu une absence de fichier", slug, set, err)
		}
	}
	var nul *SpawnFamilySet
	if nul.FamilyOf("Slayer", "Assassin") != "" || nul.Families() != nil {
		t.Error("un ensemble nil ne reconnaît aucune famille")
	}
}

func TestSpawnFamilies_NomReserveEgalALaFamilleTous(t *testing.T) {
	if SpawnFamilyAll != domain.FamilleToutesModes {
		t.Errorf("%q != %q : le nom réservé doit être celui du fichier de poids", SpawnFamilyAll, domain.FamilleToutesModes)
	}
}

func TestSpawnFamilies_ValidationStricte(t *testing.T) {
	meta := "[meta]\ntitle_slug = \"t\"\nschema_version = 1\n"
	cas := []struct{ nom, toml, fragment string }{
		{"meta absente", "[[families]]\nid = \"a\"\nvariant_contains = [\"x\"]\n", "title_slug manquant"},
		{"aucune famille", meta, "aucune entrée"},
		{"id vide", meta + "[[families]]\nvariant_contains = [\"x\"]\n", "id vide"},
		{"id reserve", meta + "[[families]]\nid = \"tous\"\nvariant_contains = [\"x\"]\n", "réservé"},
		{"id double", meta + "[[families]]\nid = \"a\"\nvariant_contains = [\"x\"]\n[[families]]\nid = \"a\"\nvariant_contains = [\"y\"]\n", "déjà déclaré"},
		{"famille sans jeton", meta + "[[families]]\nid = \"a\"\nvariant_contains = [\" \"]\n", "aucun jeton"},
	}
	for _, c := range cas {
		_, err := LoadSpawnFamiliesFromBytes("test.toml", []byte(c.toml))
		if err == nil || !strings.Contains(err.Error(), c.fragment) {
			t.Errorf("%s : erreur %v, attendu un message contenant %q", c.nom, err, c.fragment)
		}
	}
}

// L'ordre du fichier est l'ordre d'essai : la première famille dont un jeton convient l'emporte.
func TestSpawnFamilies_OrdreDEssai(t *testing.T) {
	set, err := LoadSpawnFamiliesFromBytes("test.toml", []byte(
		"[meta]\ntitle_slug = \"t\"\nschema_version = 1\n"+
			"[[families]]\nid = \"specifique\"\nvariant_contains = [\"Alpha Beta\"]\n"+
			"[[families]]\nid = \"general\"\nvariant_contains = [\"beta\"]\ncategory_contains = [\"CAT\"]\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := set.FamilyOf("alpha beta", ""); got != "specifique" {
		t.Errorf("la première famille de la liste l'emporte : %q", got)
	}
	if got := set.FamilyOf("gamma beta", ""); got != "general" {
		t.Errorf("jeton de variante : %q", got)
	}
	if got := set.FamilyOf("gamma", "une cat"); got != "general" {
		t.Errorf("jeton de catégorie : %q", got)
	}
	if got := set.FamilyOf("gamma", "autre"); got != "" {
		t.Errorf("aucune famille ne convient : %q", got)
	}
}
