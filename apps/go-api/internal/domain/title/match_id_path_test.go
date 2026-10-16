package title

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// matchIDPieges : identifiants qu'une requête ou un fichier forgé peut porter.
var matchIDPieges = []string{
	"/", `\`, "..", ".", "../../etc/passwd", `..\..\windows\x`, "..-x", "./x",
	"000d5950/../../x", `000d5950\..\..\x`, "a/b", strings.Repeat("z", 300),
	strings.Repeat("a", 300) + "/../x", "", "000d5950\x00",
}

// matchPathConstructors : les constructeurs de chemin qui prennent un match_id, avec le
// dossier sous lequel le chemin doit rester.
func matchPathConstructors(root string) []struct {
	name  string
	dir   string
	build func(matchID string) string
} {
	p := NewPathResolver(root)
	d := NewDemoLayout(filepath.Join(root, "demo"))
	return []struct {
		name  string
		dir   string
		build func(matchID string) string
	}{
		{"ReplayArtifactPath", p.ReplayArtifactsDir(DefaultSlug),
			func(id string) string { return p.ReplayArtifactPath(DefaultSlug, id) }},
		{"FilmFactsPath", p.FilmFactsDir(DefaultSlug),
			func(id string) string { return p.FilmFactsPath(DefaultSlug, id) }},
		{"TacticalRasterPath", p.TacticalRasterDir(DefaultSlug),
			func(id string) string { return p.TacticalRasterPath(DefaultSlug, id) }},
		{"DemoLayout.ReplayArtifactPath", d.ReplayArtifactsDir(DefaultSlug),
			func(id string) string { return d.ReplayArtifactPath(DefaultSlug, id) }},
	}
}

// TestMatchPathsStayUnderTheirRoot : un match_id mal formé ne fait jamais sortir un
// chemin de son dossier, et le chemin obtenu ne désigne aucun fichier : toute lecture
// et toute écriture échouent (fermé par défaut).
func TestMatchPathsStayUnderTheirRoot(t *testing.T) {
	root := t.TempDir()
	for _, c := range matchPathConstructors(root) {
		if err := os.MkdirAll(c.dir, 0o755); err != nil {
			t.Fatal(err)
		}
		for _, id := range matchIDPieges {
			got := c.build(id)
			if filepath.Dir(got) != filepath.Clean(c.dir) {
				t.Errorf("%s(%q) = %q : sort de %q", c.name, id, got, c.dir)
				continue
			}
			if err := os.WriteFile(got, []byte("x"), 0o644); err == nil {
				_ = os.Remove(got)
				t.Errorf("%s(%q) = %q : écriture acceptée, attendu un refus (identifiant mal formé)", c.name, id, got)
			}
			if _, err := os.Stat(got); err == nil {
				t.Errorf("%s(%q) = %q : désigne un fichier existant", c.name, id, got)
			}
		}
	}
}

// TestMatchPathsKeepValidIDs : les identifiants réels (complet, forme courte, majuscules)
// gardent leur chemin, et les deux formes d'un même match donnent le même.
func TestMatchPathsKeepValidIDs(t *testing.T) {
	root := t.TempDir()
	for _, c := range matchPathConstructors(root) {
		complet := c.build("000d5950-1234-4abc-9def-0123456789ab")
		court := c.build("000d5950")
		if complet != court {
			t.Errorf("%s : deux chemins pour un même match (%q, %q)", c.name, complet, court)
		}
		if !strings.HasPrefix(filepath.Base(complet), "000d5950.") {
			t.Errorf("%s : nom inattendu %q", c.name, complet)
		}
		if err := os.MkdirAll(filepath.Dir(complet), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(complet, []byte("x"), 0o644); err != nil {
			t.Errorf("%s : écriture refusée pour un identifiant valide : %v", c.name, err)
		}
	}
}
