package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seed_citation_assets_test.go — garde-rails sur les identifiants et les visuels des
// citations seedées. Pure data + accès disque en lecture seule (aucune DB), donc hors
// build cgo.
//
// Motivation : un image_path mort ne produit AUCUN signal côté serveur. Le handler
// /static/commendations/* répond 404 et le front (MedalIcon) masque l'image sur onError —
// la vignette disparaît simplement. Le défaut ne se voit donc qu'à l'œil, sur la page
// Citations d'un joueur. Ce test le transforme en échec de build.

// citationRepoRoot — racine du dépôt vue depuis internal/ops (4 niveaux : ops →
// internal → go-api → apps → racine). Les image_path seedés sont relatifs à cette racine
// ("static/commendations/<titre>/<fichier>").
const citationRepoRoot = "../../../.."

// citationImagePrefix — préfixe imposé à tout image_path (wpH5/wpHI dans le seed).
const citationImagePrefix = "static/commendations/"

// TestCitationNorms_Unique : citation_name_norm est la PRIMARY KEY de citation_mappings.
// Un doublon dans defaultCitationMappings() ne casserait pas le seed (SELECT-then-INSERT-
// or-UPDATE : la 2e ligne écraserait silencieusement la 1re), il ferait juste disparaître
// une citation. Interdit ici.
func TestCitationNorms_Unique(t *testing.T) {
	seen := make(map[string]string)
	for _, m := range defaultCitationMappings() {
		if m.Norm == "" {
			t.Errorf("citation %q: Norm vide", m.Display)
			continue
		}
		if prev, dup := seen[m.Norm]; dup {
			t.Errorf("norm %q dupliqué: %q et %q — la seconde écraserait la première au seed", m.Norm, prev, m.Display)
			continue
		}
		seen[m.Norm] = m.Display
	}
}

// TestCitationImagePaths_ExistOnDisk : tout image_path seedé désigne un fichier réellement
// présent sous static/commendations/. Volontairement AGNOSTIQUE À L'EXTENSION : le jour où
// un visuel provisoire (.svg) est remplacé par le visuel définitif (.png), basculer
// l'extension dans le seed AVANT que le fichier n'arrive fait échouer ce test — c'est
// l'effet recherché.
//
// Les noms de fichiers pré-encodés sur disque (caractères interdits sous Windows, ex. `?`
// stocké littéralement `%3F`) sont comparés tels quels : le seed porte exactement le nom du
// fichier, et ce sont les constructeurs d'URL côté analysis (BuildCitationSnippets,
// MergeCitationTotals) qui ré-encodent (url.PathEscape par segment, `%` → `%25`).
func TestCitationImagePaths_ExistOnDisk(t *testing.T) {
	staticRoot := filepath.Join(citationRepoRoot, filepath.FromSlash(citationImagePrefix))
	if _, err := os.Stat(staticRoot); err != nil {
		t.Skipf("arborescence %s absente — garde-rail NON joué (%v)", staticRoot, err)
	}
	for _, m := range defaultCitationMappings() {
		if m.ImagePath == "" {
			continue
		}
		if !strings.HasPrefix(m.ImagePath, citationImagePrefix) {
			t.Errorf("citation %q: image_path %q hors de %s", m.Norm, m.ImagePath, citationImagePrefix)
			continue
		}
		full := filepath.Join(citationRepoRoot, filepath.FromSlash(m.ImagePath))
		if _, err := os.Stat(full); err != nil {
			t.Errorf("citation %q (%q): image_path %q introuvable sur disque (%v)", m.Norm, m.Display, m.ImagePath, err)
		}
	}
}

// TestCitationEnabled_HasImagePath : une citation active sans visuel s'affiche sans
// vignette dans la grille Citations. Les seules citations sans image_path tolérées sont
// celles désactivées (inventaire conservé, moteur qui les ignore).
func TestCitationEnabled_HasImagePath(t *testing.T) {
	for _, m := range defaultCitationMappings() {
		if m.Enabled && m.ImagePath == "" {
			t.Errorf("citation %q (%q) est active mais n'a aucun image_path", m.Norm, m.Display)
		}
	}
}
