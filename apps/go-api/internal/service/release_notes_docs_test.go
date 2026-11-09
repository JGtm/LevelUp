package service

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// repoDocsPath résout un chemin sous docs/ du dépôt à partir de l'emplacement de ce
// fichier (apps/go-api/internal/service -> racine du dépôt à quatre niveaux).
func repoDocsPath(t *testing.T, rel string) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller: emplacement du test introuvable")
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	return filepath.Join(root, filepath.FromSlash(rel))
}

func readRepoDoc(t *testing.T, rel string) string {
	t.Helper()
	data, err := os.ReadFile(repoDocsPath(t, rel))
	if err != nil {
		t.Fatalf("lecture de %s : %v", rel, err)
	}
	return string(data)
}

// countVersionLines compte les lignes de version de la section « What's new » :
// c'est le nombre de blocs attendu, indépendamment de la forme de la clé.
func countVersionLines(content string) int {
	n := 0
	in := false
	for _, line := range strings.Split(content, "\n") {
		s := strings.TrimSpace(line)
		if !in {
			in = s == "## What's new" || s == "## Dernières nouveautés"
			continue
		}
		if strings.HasPrefix(s, "## ") {
			break
		}
		if isVersionLine(s) {
			n++
		}
	}
	return n
}

func sortedBlockKeys(blocks map[string]string) []string {
	keys := make([]string, 0, len(blocks))
	for k := range blocks {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return compareVersionDesc(keys[i], keys[j]) })
	return keys
}

var changelogHeadingRe = regexp.MustCompile(`(?m)^## \[(\d+)\.(\d+)\.(\d+)\]`)

// La clé d'un bloc porte le correctif quand la ligne le nomme, et un correctif se classe avant
// sa version mineure.
func TestExtractVersionKey(t *testing.T) {
	cases := map[string]string{
		"**v8.1.1 — Scores settle**": "v8.1.1",
		"**v8.1 — LevelUp**":         "v8.1",
		"**v8.0 — 2D replay**":       "v8.0",
		"**v8.0 — 2 new modes**":     "v8.0",
		"- **v7.2.1 — Objective**":   "v7.2.1",
		"**v8.1-rc1 — Preview**":     "v8.1",
	}
	for line, want := range cases {
		if got := extractVersionKey(line); got != want {
			t.Errorf("extractVersionKey(%q) = %q, attendu %q", line, got, want)
		}
	}
	got := sortedBlockKeys(map[string]string{"v8.0": "", "v8.1": "", "v8.1.1": "", "v7.2.1": ""})
	if want := "[v8.1.1 v8.1 v8.0 v7.2.1]"; fmt.Sprint(got) != want {
		t.Errorf("ordre = %v, attendu %s", got, want)
	}
}

// TestReleaseNotesDocs_BlocksExtractedWithoutCollision garde les notes de version servies
// par l'app : la première occurrence d'une clé gagne, donc deux lignes de version de même
// clé feraient disparaître un bloc en silence.
func TestReleaseNotesDocs_BlocksExtractedWithoutCollision(t *testing.T) {
	langs := map[string]string{"en": "docs/RELEASE_NOTES.md", "fr": "docs/FR/RELEASE_NOTES.md"}
	keys := map[string][]string{}
	for lang, rel := range langs {
		content := readRepoDoc(t, rel)
		blocks := extractWhatsNewBlocks(content)
		if want := countVersionLines(content); len(blocks) != want {
			t.Errorf("%s : %d blocs extraits pour %d lignes de version (collision de clé vX.Y ou bloc perdu)", rel, len(blocks), want)
		}
		if len(blocks) == 0 {
			t.Fatalf("%s : aucun bloc extrait", rel)
		}
		keys[lang] = sortedBlockKeys(blocks)
	}
	if got, want := fmt.Sprint(keys["fr"]), fmt.Sprint(keys["en"]); got != want {
		t.Errorf("clés de blocs FR %s différentes des clés EN %s", got, want)
	}
}

// TestReleaseNotesDocs_NewestBlockMatchesChangelog garde la cohérence de numéro : le bloc
// le plus récent des notes de version porte le numéro de la première entrée du changelog
// (vX.Y pour une version mineure en .0, vX.Y.Z pour un correctif), en anglais et en français.
func TestReleaseNotesDocs_NewestBlockMatchesChangelog(t *testing.T) {
	pairs := [][2]string{
		{"docs/RELEASE_NOTES.md", "docs/CHANGELOG.md"},
		{"docs/FR/RELEASE_NOTES.md", "docs/FR/CHANGELOG.md"},
	}
	for _, p := range pairs {
		keys := sortedBlockKeys(extractWhatsNewBlocks(readRepoDoc(t, p[0])))
		m := changelogHeadingRe.FindStringSubmatch(readRepoDoc(t, p[1]))
		if m == nil || len(keys) == 0 {
			t.Fatalf("%s / %s : entrée de changelog ou bloc introuvable", p[0], p[1])
		}
		want := fmt.Sprintf("v%s.%s", m[1], m[2])
		if m[3] != "0" {
			want = fmt.Sprintf("%s.%s", want, m[3])
		}
		if keys[0] != want {
			t.Errorf("%s : bloc le plus récent %s, première entrée de %s : %s", p[0], keys[0], p[1], want)
		}
	}
}
