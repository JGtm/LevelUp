package classification

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTOML(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "ranked_hoppers.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestLoadSetClassifier_Populated(t *testing.T) {
	path := writeTOML(t, `
schema_version = 1
ranked_hopper_ids = ["ranked-a", "ranked-b"]
pve_hopper_ids = ["warzone-ff"]
`)
	c, err := LoadSetClassifier(path)
	if err != nil {
		t.Fatalf("LoadSetClassifier: %v", err)
	}
	if v := c.IsRanked("ranked-a"); v == nil || !*v {
		t.Errorf("ranked-a → want &true, got %v", v)
	}
	if v := c.IsRanked("ranked-zzz"); v == nil || *v {
		t.Errorf("ranked-zzz → want &false (set exhaustif), got %v", v)
	}
	if v := c.IsPvE("warzone-ff"); v == nil || !*v {
		t.Errorf("warzone-ff → want &true, got %v", v)
	}
}

func TestLoadSetClassifier_EmptyArrays_Indeterminate(t *testing.T) {
	// Le cas RÉEL de Halo 5 aujourd'hui : fichier présent, listes vides → nil.
	path := writeTOML(t, `
schema_version = 1
ranked_hopper_ids = []
pve_hopper_ids = []
`)
	c, err := LoadSetClassifier(path)
	if err != nil {
		t.Fatalf("LoadSetClassifier: %v", err)
	}
	if v := c.IsRanked("any"); v != nil {
		t.Errorf("listes vides → IsRanked nil, got %v", *v)
	}
	if v := c.IsPvE("any"); v != nil {
		t.Errorf("listes vides → IsPvE nil, got %v", *v)
	}
}

func TestLoadSetClassifier_MissingFile_EmptyNoError(t *testing.T) {
	// Fichier absent = config optionnelle non publiée → classifier vide, PAS d'erreur.
	c, err := LoadSetClassifier(filepath.Join(t.TempDir(), "does_not_exist.toml"))
	if err != nil {
		t.Fatalf("fichier absent ne doit pas être une erreur, got %v", err)
	}
	if v := c.IsRanked("any"); v != nil {
		t.Errorf("classifier vide attendu, got %v", *v)
	}
}

func TestLoadSetClassifier_BadSchemaVersion(t *testing.T) {
	path := writeTOML(t, "schema_version = 2\nranked_hopper_ids = []\n")
	if _, err := LoadSetClassifier(path); err == nil {
		t.Errorf("schema_version=2 → want erreur")
	}
}

func TestLoadSetClassifier_Malformed(t *testing.T) {
	path := writeTOML(t, "schema_version = 1\nranked_hopper_ids = [unclosed")
	if _, err := LoadSetClassifier(path); err == nil {
		t.Errorf("TOML malformé → want erreur")
	}
}

// TestLoadSetClassifier_ShippedHalo5Config valide le fichier RÉELLEMENT versionné
// (config/titles/halo_5/catalog/ranked_hoppers.toml) : il doit parser et, vide,
// rendre des verdicts nil. Skip si le chemin relatif ne résout pas (cwd atypique).
func TestLoadSetClassifier_ShippedHalo5Config(t *testing.T) {
	rel := filepath.Join("..", "..", "..", "..", "..",
		"config", "titles", "halo_5", "catalog", "ranked_hoppers.toml")
	if _, statErr := os.Stat(rel); os.IsNotExist(statErr) {
		t.Skipf("config h5 introuvable depuis %s (cwd atypique) — skip", rel)
	}
	c, err := LoadSetClassifier(rel)
	if err != nil {
		t.Fatalf("le fichier versionné doit parser : %v", err)
	}
	if v := c.IsRanked("any"); v != nil {
		t.Errorf("h5 ranked_hoppers vide → IsRanked nil, got %v", *v)
	}
}
