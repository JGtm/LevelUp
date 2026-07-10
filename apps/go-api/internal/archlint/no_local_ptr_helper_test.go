// Package archlint — no_local_ptr_helper_test.go : ratchet H5.
//
// Interdit toute nouvelle copie locale du helper pointeur pur `func strPtr(...)
// *string { return &s }` (et son clone `strPtrH5`). La source unique est
// pointers.Ptr[T] (internal/util/pointers). Le 1-liner avait été recopié sous
// plusieurs noms — leçon CLAUDE.md règle 6.
//
// Le motif cible EXACTEMENT strPtr / strPtrH5 : il n'attrape PAS les variantes
// sémantiquement distinctes strPtrNonEmpty (sync — nil si vide) ni strPtrOrNil
// (openspartan — TrimSpace + nil si vide), qui restent volontairement locales.
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

// localPtrHelperRE matche `func strPtr(` ou `func strPtrH5(` (les noms purs
// dédupliqués). strPtrNonEmpty/strPtrOrNil (autres noms) ne matchent pas.
var localPtrHelperRE = regexp.MustCompile(`func strPtr(?:H5)?\(`)

func TestNoLocalPtrHelper(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile))
	goAPIRoot := filepath.Dir(internalRoot)

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
				if localPtrHelperRE.MatchString(line) {
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
		t.Errorf("helper pointeur local strPtr/strPtrH5 interdit (H5) — utiliser "+
			"pointers.Ptr[T] (internal/util/pointers) :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
