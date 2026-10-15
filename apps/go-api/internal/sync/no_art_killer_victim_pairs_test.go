// Package sync — no_art_killer_victim_pairs_test.go : `killer_victim_pairs` reste SANS INDEX, et
// un seul ecrivain la reecrit par DELETE.
//
// `no_art_patterns_test.go` laisse cette table hors de ses tables protegees (elle n est pas
// append-only) et hors de son perimetre les dossiers `migration` et `cmd`. Or la completion combat
// (`persist.EventsCompletionPersister`) la reecrit par match en DELETE + INSERT, ce qui n est sur
// que sur une table sans index ART : les trois index herites sont retires par la migration
// `shared_killer_victim_pairs_drop_legacy_indexes_v1`, et l entretien `repair-highlight-phantoms`
// reecrit par echange de table (`migration.ReplaceKillerVictimPairs`). Ce garde-rail tient les deux
// moities, sur TOUT le module (cmd, migration compris) :
//
//   - aucun `CREATE INDEX ... ON killer_victim_pairs` hors tests ;
//   - `DELETE FROM killer_victim_pairs` dans le seul fichier de la completion.
package sync

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// seulEcrivainDeleteKillerVictimPairs : le fichier qui a le droit de reecrire la table par DELETE
// (2026-10-15 : table sans index depuis la migration ci-dessus).
const seulEcrivainDeleteKillerVictimPairs = "internal/persist/events_completion_persister.go"

var (
	reDeleteKillerVictimPairs = regexp.MustCompile(`(?i)\bDELETE\s+FROM\s+killer_victim_pairs\b`)
	reIndexKillerVictimPairs  = regexp.MustCompile(`(?is)\bCREATE\s+(UNIQUE\s+)?INDEX\b[^;]*?\bON\s+killer_victim_pairs\b`)
)

// TestKillerVictimPairsSansIndex — mutation qui doit le faire rougir : remettre un
// `DELETE FROM killer_victim_pairs WHERE match_id = ?` dans `internal/sync/killcollector/` (ou dans
// `internal/migration/`), ou un `CREATE INDEX idx_kv_match ON killer_victim_pairs(match_id)` dans
// une migration.
func TestKillerVictimPairsSansIndex(t *testing.T) {
	root := findRepoRoot(t)
	var fautes []string
	for _, sub := range []string{"cmd", "internal"} {
		err := filepath.Walk(filepath.Join(root, sub), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			buf, err := os.ReadFile(path) //nolint:gosec // chemin de test, lecture seule
			if err != nil {
				return err
			}
			code := stripGoComments(string(buf))
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			if reIndexKillerVictimPairs.MatchString(code) {
				fautes = append(fautes, rel+" : index sur killer_victim_pairs")
			}
			if reDeleteKillerVictimPairs.MatchString(code) && rel != seulEcrivainDeleteKillerVictimPairs {
				fautes = append(fautes, rel+" : DELETE sur killer_victim_pairs (reecrire par echange de table)")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("parcours de %s : %v", sub, err)
		}
	}
	if len(fautes) > 0 {
		t.Errorf("killer_victim_pairs : %d faute(s) (bug ART #23645, ADR 0019) :\n  - %s",
			len(fautes), strings.Join(fautes, "\n  - "))
	}
}
