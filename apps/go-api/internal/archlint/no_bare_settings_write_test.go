// Package archlint — no_bare_settings_write_test.go : ratchet « écriture de
// settings hors atomicfile » (v7.3, item backlog ops/prod du 2026-07-25).
//
// POURQUOI. app_settings.json (et les overlays par titre) sont bind-montés FICHIER
// dans le conteneur de production : le pattern temp + os.Rename y échoue en EBUSY
// (l'inode est épinglé) et l'écriture est PERDUE en silence. Le point d'écriture
// unique est désormais internal/platform/atomicfile.WriteFile (atomique d'abord,
// repli in-place sur EBUSY). Trois call sites l'utilisent (settings.Store.Save,
// settings.Store.SaveTitleOverlay, notify.writeLastNotifiedVersion) — au 3e
// exemplaire, la règle CLAUDE.md n°6 impose la centralisation ET ce garde-rail :
// sans lui, un 4e writer re-poserait un os.WriteFile/os.Rename nu et la dette
// re-divergerait.
//
// PORTÉE. Les packages qui écrivent des fichiers de settings : internal/platform/
// settings et internal/notify. Les tests sont exclus (ils sèment des fixtures).
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// bareFileWriteRE — écriture de fichier NUE (contourne atomicfile.WriteFile).
// `atomicfile.WriteFile(` ne matche pas (préfixe de sélecteur différent).
var bareFileWriteRE = regexp.MustCompile(`\bos\.(WriteFile|Rename)\(`)

// settingsWriterPackages — packages sous ratchet, relatifs à internal/.
var settingsWriterPackages = []string{
	filepath.Join("platform", "settings"),
	"notify",
}

func TestNoBareSettingsWrite(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile))

	var violations []string
	for _, pkg := range settingsWriterPackages {
		root := filepath.Join(internalRoot, pkg)
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
			rel, _ := filepath.Rel(internalRoot, path)
			rel = filepath.ToSlash(rel)
			for i, line := range strings.Split(string(data), "\n") {
				if bareFileWriteRE.MatchString(line) {
					violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", pkg, err)
		}
	}

	if len(violations) > 0 {
		t.Errorf("écriture de fichier NUE dans un package writer de settings — "+
			"passer par internal/platform/atomicfile.WriteFile (repli in-place sur "+
			"bind-mount fichier, sinon l'écriture est perdue en prod) :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
