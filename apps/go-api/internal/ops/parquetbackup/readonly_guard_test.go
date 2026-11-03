package parquetbackup

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// writeSQL : une instruction d'écriture ou de maintenance DuckDB écrite en majuscules dans une chaîne.
var writeSQL = regexp.MustCompile(`\b(INSERT|UPDATE|DELETE|CREATE|ALTER|DROP|CHECKPOINT|ATTACH|VACUUM)\b`)

// dataJoin : un chemin « data » écrit à la main dans un filepath.Join.
var dataJoin = regexp.MustCompile(`filepath\.Join\([^)]*"data"`)

// TestSourcesReadOnlyAndPathResolver est le garde-rail de la sauvegarde : le paquet n'écrit dans aucune
// base (ses seules instructions SQL sont les COPY TO de duckdbbackup, hors de ce paquet) et ne construit
// aucun chemin de données à la main (PathResolver seulement).
func TestSourcesReadOnlyAndPathResolver(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("sources introuvables : %v", err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(data), "\n") {
			code := strings.TrimSpace(line)
			if strings.HasPrefix(code, "//") {
				continue
			}
			if idx := strings.Index(code, "//"); idx >= 0 {
				code = code[:idx]
			}
			if writeSQL.MatchString(code) {
				t.Errorf("%s:%d : instruction d'écriture ou de maintenance SQL dans la sauvegarde : %s", f, i+1, code)
			}
			if dataJoin.MatchString(code) {
				t.Errorf("%s:%d : chemin « data » construit à la main : %s", f, i+1, code)
			}
		}
	}
}
