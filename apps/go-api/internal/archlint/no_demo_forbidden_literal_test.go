// Package archlint — no_demo_forbidden_literal_test.go : ratchet du refus en mode démo
// (backlog 2026-09-26, lot B-C1).
//
// Le refus 403 `demo_mode_forbidden` d'une action de mutation en démo a quatre appelants
// (sauvegarde, création de profil, purge d'un titre, abonnements du watcher). Règle n°6 de
// CLAUDE.md : à la troisième copie, un helper ET un garde-rail. Source unique :
// handlers.refuseInDemo (internal/api/handlers/demo_guard.go). Ce test interdit le
// littéral du code d'erreur dans tout autre fichier Go de production.
package archlint

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

const demoForbiddenLiteral = `"demo_mode_forbidden"`

// demoForbiddenHome est le seul fichier autorisé à porter le littéral (relatif à la
// racine du module).
const demoForbiddenHome = "internal/api/handlers/demo_guard.go"

func TestNoDemoForbiddenLiteral(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	var violations []string
	homeSeen := false
	for _, sub := range []string{"internal", "cmd", "pkg"} {
		root := filepath.Join(moduleRoot, sub)
		if _, err := os.Stat(root); err != nil {
			continue
		}
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			rel, _ := filepath.Rel(moduleRoot, path)
			rel = filepath.ToSlash(rel)
			if rel == demoForbiddenHome {
				homeSeen = true
				return nil
			}
			for i, line := range strings.Split(string(data), "\n") {
				if strings.Contains(line, demoForbiddenLiteral) {
					violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s : %v", sub, err)
		}
	}
	if !homeSeen {
		t.Errorf("%s introuvable : le helper du refus en démo a bougé, mettre ce garde-rail à jour", demoForbiddenHome)
	}
	if len(violations) > 0 {
		t.Errorf("littéral %s hors de %s — passer par handlers.refuseInDemo :\n  %s",
			demoForbiddenLiteral, demoForbiddenHome, strings.Join(violations, "\n  "))
	}
}
