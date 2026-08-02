package canonical

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestAllFieldKeysCount fixe la cardinalité du canonique pour repérer toute
// dérive non documentée (ajout/suppression sans mise à jour du plan).
func TestAllFieldKeysCount(t *testing.T) {
	t.Parallel()
	// 66 = 60 + 6 (v7.3 lot 2, item 3.3) : wins, personal_score_per_min,
	// accuracy_threshold_days, combat_{precision,endurance,excellence}_matches.
	// Ces clés étaient consommées à l'écran (PB Ascension, jalons) mais absentes
	// des TOML : leurs libellés vivaient dans un dictionnaire TS parallèle,
	// supprimé au profit de la source unique. Voir golden fields.golden.txt.
	const expected = 66
	if got := len(AllFieldKeys()); got != expected {
		t.Fatalf("AllFieldKeys count = %d, want %d (mettre à jour fields.golden.txt + annexe §17 si intentionnel)", got, expected)
	}
}

// TestAllFieldKeysUnique vérifie qu'aucun FieldKey n'est dupliqué.
func TestAllFieldKeysUnique(t *testing.T) {
	t.Parallel()
	seen := make(map[FieldKey]struct{})
	for _, k := range AllFieldKeys() {
		if _, dup := seen[k]; dup {
			t.Errorf("FieldKey dupliqué: %q", k)
		}
		seen[k] = struct{}{}
	}
}

// TestAllFieldKeysSnakeCase enforce la convention snake_case.
func TestAllFieldKeysSnakeCase(t *testing.T) {
	t.Parallel()
	for _, k := range AllFieldKeys() {
		s := string(k)
		if s == "" {
			t.Errorf("FieldKey vide")
			continue
		}
		if strings.ToLower(s) != s {
			t.Errorf("FieldKey non snake_case (majuscule): %q", k)
		}
		if strings.ContainsAny(s, " -.") {
			t.Errorf("FieldKey non snake_case (caractère interdit): %q", k)
		}
	}
}

// TestIsKnownFieldKey couvre l'API publique du package.
func TestIsKnownFieldKey(t *testing.T) {
	t.Parallel()
	if !IsKnownFieldKey(FieldKills) {
		t.Errorf("FieldKills devrait être connu")
	}
	if IsKnownFieldKey(FieldKey("not_a_real_field")) {
		t.Errorf("not_a_real_field ne devrait pas être connu")
	}
}

// TestFieldKey_String vérifie la conversion FieldKey → string.
func TestFieldKey_String(t *testing.T) {
	t.Parallel()
	if FieldKills.String() != "kills" {
		t.Errorf("FieldKills.String() = %q", FieldKills.String())
	}
	if FieldKey("custom").String() != "custom" {
		t.Errorf("FieldKey(custom).String() = %q", FieldKey("custom").String())
	}
}

// TestFieldKeysGolden compare les FieldKey à un fichier golden pour détecter
// tout renommage silencieux. Régénération volontaire :
// `go test -run TestFieldKeysGolden -update` (env var TESTING_UPDATE_GOLDEN=1).
func TestFieldKeysGolden(t *testing.T) {
	t.Parallel()
	keys := AllFieldKeys()
	lines := make([]string, len(keys))
	for i, k := range keys {
		lines[i] = string(k)
	}
	sort.Strings(lines)
	got := strings.Join(lines, "\n") + "\n"

	goldenPath := filepath.Join("testdata", "fields.golden.txt")

	if os.Getenv("TESTING_UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (lancer avec TESTING_UPDATE_GOLDEN=1 pour régénérer): %v", err)
	}
	if string(want) != got {
		t.Errorf("golden FieldKey diverge (set TESTING_UPDATE_GOLDEN=1 pour régénérer après revue):\nwant:\n%s\ngot:\n%s", want, got)
	}
}
