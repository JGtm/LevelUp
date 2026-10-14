package sync

// append_only_ops_guard_test.go — garde-rail STATEMENT-level des tables append-only étendu à
// internal/ops (lot recos-d, D2, 2026-10-09).
//
// POURQUOI. TestNoMutationOnAppendOnlyStateTables exclut internal/ops (outillage) ; or ops/
// tourne aussi IN-PROCESS (actions admin, seed de la démo), et c'est là que vivait
// `UPDATE kill_positions SET killer_xuid = …` (anonymisation de la démo), sur une table
// append-only par passe. Le modèle est TestNoMutationOnMediaAppendOnlyTables : motifs ancrés
// sur le NOM EXACT de chaque table de appendOnlyStateTables, donc aucun faux positif sur une
// AUTRE table du même fichier.
//
// LE TROU DES NOMS INTERPOLÉS. Un `fmt.Sprintf("UPDATE %s …", table)` ne contient aucun
// littéral `UPDATE <table>` : c'est exactement ainsi que la mutation de kill_positions
// échappait aux deux garde-rails existants. La forme interpolée est donc interdite dans
// internal/ops, quelle que soit la table visée : une écriture de démo ou d'outil se fait par
// copie transformée (seed_demo_anonymize.go) ou par un statement littéral que ce scan voit.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// reInterpolatedMutation : UPDATE … SET / DELETE FROM dont le nom de table est un verbe de
// format. `SET` est exigé après UPDATE : un message d'erreur (« UPDATE %s/id=%d ») n'est pas
// une instruction SQL.
var reInterpolatedMutation = regexp.MustCompile(`(?i)\bUPDATE\s+%[sv]\s+SET\b|\bDELETE\s+FROM\s+%[sv]\b`)

// opsMutationViolations rend les mutations interdites de `text` (déjà débarrassé de ses
// commentaires) pour le fichier `rel`.
func opsMutationViolations(text, rel string) []string {
	var out []string
	for _, table := range appendOnlyStateTables {
		if reMediaDelete(table).MatchString(text) {
			out = append(out, "DELETE FROM "+table+" dans "+rel)
		}
		if reMediaUpdate(table).MatchString(text) {
			out = append(out, "UPDATE "+table+" dans "+rel)
		}
	}
	if reInterpolatedMutation.MatchString(text) {
		out = append(out, "UPDATE/DELETE à nom de table interpolé dans "+rel)
	}
	return out
}

func TestNoMutationOnAppendOnlyTablesInOps(t *testing.T) {
	opsDir := filepath.Join(findRepoRoot(t), "internal", "ops")
	entries, err := os.ReadDir(opsDir)
	if err != nil {
		t.Fatalf("lecture de %s : %v", opsDir, err)
	}
	var violations []string
	scanned := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(opsDir, name))
		if err != nil {
			t.Fatalf("lecture de %s : %v", name, err)
		}
		scanned++
		violations = append(violations, opsMutationViolations(stripGoComments(string(content)), "internal/ops/"+name)...)
	}
	if scanned == 0 {
		t.Fatal("aucun fichier Go scanné sous internal/ops : le garde-rail ne vérifie rien")
	}
	if len(violations) > 0 {
		t.Errorf("RÉGRESSION append-only (internal/ops) : %d mutation(s) interdite(s) "+
			"(append-only = INSERT pur + vue _latest ; anonymiser À LA COPIE) :\n  - %s",
			len(violations), strings.Join(violations, "\n  - "))
	}
}

// TestOpsMutationDetection_Sanity : le garde-rail MORD sur les trois formes, et laisse passer
// l'INSERT pur, la copie transformée et une mutation littérale d'une table hors liste.
func TestOpsMutationDetection_Sanity(t *testing.T) {
	for _, src := range []string{
		"q := `UPDATE kill_positions SET killer_xuid = ? WHERE killer_xuid = ?`",
		"q := `DELETE FROM match_kill_events WHERE match_id = ?`",
		"stmt := fmt.Sprintf(`UPDATE %s SET %s WHERE %s = ?`, t.table, set, col)",
	} {
		if len(opsMutationViolations(src, "x.go")) == 0 {
			t.Errorf("mutation NON détectée (garde-rail aveugle) : %q", src)
		}
	}
	for _, src := range []string{
		"q := `INSERT INTO kill_positions SELECT * FROM src.kill_positions`",
		"q := `CREATE TABLE kill_positions AS SELECT * REPLACE (x AS killer_xuid) FROM src.kill_positions t`",
		"q := `UPDATE media_files SET status = NULL WHERE id = ?`",
		`return fmt.Errorf("backfill medailles: UPDATE %s/id=%d: %w", matchID, id, err)`,
		"q := `SELECT * FROM kill_positions_latest`",
	} {
		if v := opsMutationViolations(src, "x.go"); len(v) > 0 {
			t.Errorf("faux positif sur %q : %v", src, v)
		}
	}
}
