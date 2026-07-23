// idx_rec_hist_canonical_test.go — garde-rail F4 (règle CLAUDE.md n°6) : toute
// création de l'index idx_rec_hist_achieved_desc DOIT porter la définition
// canonique (user_id, achieved_at DESC). Historiquement, la dédup le recréait sur
// (achieved_at DESC) sans user_id → divergence avec la création de base et la
// reconstruction post-purge. Ce ratchet interdit toute NOUVELLE divergence, quel
// que soit le package (migrations, ops, …).
//
// Pur (pas de CGO) : scanne les sources .go non-test de tout le module go-api.
package migrations

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// canonicalRecHistIndexTarget : forme canonique attendue à droite du CREATE INDEX.
const canonicalRecHistIndexTarget = "record_history(user_id, achieved_at DESC)"

func TestRecHistAchievedIndexAlwaysCanonical(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	// thisFile = .../internal/games/halo_infinite/migrations/x_test.go
	// Remonter jusqu'à la racine go-api (dossier contenant go.mod).
	goAPIRoot := thisFile
	for {
		goAPIRoot = filepath.Dir(goAPIRoot)
		if goAPIRoot == "." || goAPIRoot == string(filepath.Separator) {
			t.Fatal("racine go-api (go.mod) introuvable")
		}
		if _, err := os.Stat(filepath.Join(goAPIRoot, "go.mod")); err == nil {
			break
		}
	}

	var violations []string
	for _, sub := range []string{"internal", "cmd"} {
		root := filepath.Join(goAPIRoot, sub)
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			// _test.go exclu : certains tests créent VOLONTAIREMENT l'index divergent
			// pour prouver que le step correctif le répare (repair_rec_hist_index_test.go).
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			for i, line := range strings.Split(string(data), "\n") {
				if !strings.Contains(line, "idx_rec_hist_achieved_desc") {
					continue
				}
				if !strings.Contains(line, "CREATE INDEX") {
					continue // DROP INDEX / commentaires / usages non-création
				}
				if !strings.Contains(line, canonicalRecHistIndexTarget) {
					violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s/: %v", sub, err)
		}
	}
	if len(violations) > 0 {
		t.Errorf("CREATE INDEX idx_rec_hist_achieved_desc non canonique (F4, règle CLAUDE.md n°6) — "+
			"la définition DOIT être %q (côté migrations, réutiliser canonicalRecHistAchievedIndexDDL) :\n  %s",
			canonicalRecHistIndexTarget, strings.Join(violations, "\n  "))
	}
}
