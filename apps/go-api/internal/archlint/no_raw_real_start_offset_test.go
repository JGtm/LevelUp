// Package archlint — no_raw_real_start_offset_test.go : ratchet de l'écart « coup d'envoi du
// registre − début canonique » (T0, countdown d'avant-match).
//
// L'expression `epoch_ms(<alias>.real_start_time AT TIME ZONE 'UTC') - epoch_ms(<début canonique>)`
// s'écrit UNE fois, dans `analysis.SQLRealStartOffsetMs` / `analysis.SQLT0Ms` (délégué local
// `duckdb.T0MsSQL`). Cinq lectures la recopiaient mot pour mot ; une copie qui dérive date le T0
// autrement que ses voisines et décale les horloges du match (règle CLAUDE.md n°6, ≤ 2 copies).
package archlint

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// realStartAtTZ : le fragment qui signe une copie de l'expression.
const realStartAtTZ = "real_start_time AT TIME ZONE"

// realStartOffsetAllowlist : la DÉFINITION de l'expression. Toute autre entrée exige une
// justification datée expliquant pourquoi le helper ne convient pas.
var realStartOffsetAllowlist = map[string]bool{
	"internal/analysis/sql_fragments.go": true,
}

func TestNoRawRealStartOffsetExpression(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	var violations []string
	for _, sub := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(goAPIRoot, sub), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "migrations" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			if realStartOffsetAllowlist[rel] {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			for i, line := range strings.Split(string(data), "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
					continue
				}
				if strings.Contains(line, realStartAtTZ) {
					violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+trimmed)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s/: %v", sub, err)
		}
	}
	if len(violations) > 0 {
		t.Errorf("expression brute de l'écart real_start_time interdite — utiliser "+
			"analysis.SQLT0Ms(alias) (T0, NULL sans coup d'envoi) ou analysis.SQLRealStartOffsetMs(alias) "+
			"(écart seul), ou le délégué duckdb.T0MsSQL :\n  %s", strings.Join(violations, "\n  "))
	}
}
