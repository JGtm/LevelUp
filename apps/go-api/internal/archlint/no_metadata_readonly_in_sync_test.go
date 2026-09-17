package archlint

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// TestNoMetadataReadOnlyInSync — ratchet : aucune ouverture de metadata.duckdb en lecture
// seule directe (`duckdbpkg.OpenReadOnly(e.metadataDBPath)` ou `OpenReadOnly(metadataDBPath)`)
// dans internal/sync/.
//
// POURQUOI (2026-09-16/17). Le serveur tient metadata en `rw:` depuis le boot, et le moteur de
// sync le tient aussi en `rw:` partage pendant tout un run. Un OpenReadOnly direct sur le meme
// fichier dans le meme process echoue (« Can't open a connection to same database file with a
// different configuration ») : loadMedalExploitMap rendait nil en silence (medal_exploit = 0
// pour tous les matchs) et les trois passes de citations echouaient depuis l orchestrateur de
// backfill de l admin. La forme canonique est OpenReadForQuery, qui reutilise le handle en cache
// et n ouvre en lecture seule qu a defaut. Quatre copies corrigees ; ce ratchet interdit la
// cinquieme (regle « <= 2 copies » de CLAUDE.md).
func TestNoMetadataReadOnlyInSync(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	syncRoot := filepath.Join(filepath.Dir(filepath.Dir(thisFile)), "sync")

	var violations []string
	err := filepath.WalkDir(syncRoot, func(path string, d os.DirEntry, err error) error {
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
		for i, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			if strings.Contains(trimmed, "OpenReadOnly(") && strings.Contains(trimmed, "metadataDBPath") {
				rel, _ := filepath.Rel(filepath.Dir(syncRoot), path)
				violations = append(violations, filepath.ToSlash(rel)+":"+strconv.Itoa(i+1)+": "+trimmed)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de internal/sync : %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("%d ouverture(s) de metadata en lecture seule directe dans internal/sync — "+
			"utiliser duckdbpkg.OpenReadForQuery (reutilise le handle rw tenu par le process) :\n  %s",
			len(violations), strings.Join(violations, "\n  "))
	}
}
