// Package archlint — no_raw_isbot_literal_test.go : ratchet H2 (prédicat bot).
//
// Interdit toute NOUVELLE occurrence du littéral SQL brut du prédicat bot sur
// une colonne xuid — `<col>xuid [NOT ]LIKE 'bid(%'` (ou '%%' en template Sprintf).
// La source unique est analysis.SQLIsBotCol(col) / SQLIsNotBotCol(col). Le prédicat
// avait été « centralisé » en const nue SQLIsBot (0 consommateur SQL) puis re-divergé
// en 34 copies littérales — leçon CLAUDE.md règle 6.
//
// Le motif CIBLE les colonnes xuid : il n'attrape PAS `gamertag LIKE 'bid(%'`
// (colonne distincte — un filtre bot sur gamertag est un autre prédicat, hors
// périmètre H2), ni la forme paramétrée `%s LIKE 'bid(%%'` d'identity.go
// (colonne déjà abstraite en argument).
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

// rawIsBotAllowlist : DÉCROISSANTE. Restent les templates fmt.Sprintf où le
// prédicat bot est statique et %%-échappé — leur migration exige de threader un
// argument %s à travers plusieurs call sites (ordre positionnel fragile, pour un
// SQL identique) → différé, même politique que no_raw_outcome_literal_test.go
// (« allowlister si threading requis »). + 1 commentaire de doc (domain ne peut
// pas importer analysis). Chaque migration RETIRE une entrée.
var rawIsBotAllowlist = map[string]bool{
	"internal/platform/duckdb/cross_game_repo.go":           true,
	"internal/platform/duckdb/leaderboard_world_repo.go":    true,
	"internal/platform/duckdb/queries_career_encounters.go": true,
	"internal/platform/duckdb/queries_relations_moments.go": true,
	"internal/platform/duckdb/queries_squad.go":             true,
	"internal/platform/duckdb/queries_match_detail.go":      true,
	"internal/domain/media.go":                              true,
}

// rawIsBotRE matche `<...>xuid [NOT ]LIKE 'bid(%'` ou '%%'. `%%?` = un ou deux %.
var rawIsBotRE = regexp.MustCompile(`[\w.]*xuid\s+(?:NOT\s+)?LIKE\s+'bid\(%%?'`)

func TestNoNewRawIsBotLiteral(t *testing.T) {
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
			// sql_fragments.go = définition du helper (source unique).
			if rel == "internal/analysis/sql_fragments.go" || rawIsBotAllowlist[rel] {
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
				if rawIsBotRE.MatchString(line) {
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
		t.Errorf("littéral SQL brut du prédicat bot xuid interdit (H2) — utiliser "+
			"analysis.SQLIsBotCol(col)/SQLIsNotBotCol(col) (arg %%s en contexte Sprintf), "+
			"ou allowlister transitoirement si threading requis :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
