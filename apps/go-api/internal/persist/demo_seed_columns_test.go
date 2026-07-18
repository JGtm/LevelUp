package persist

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// insertColumnsUnion extrait l'UNION des colonnes de tous les `INSERT INTO <table> (
// ... )` d'un blob source Go. Robuste au formatage multi-ligne ; ignore les INSERT
// sans liste de colonnes explicite (ex. `INSERT INTO t (match_id) VALUES` compte
// aussi, mais ceux d'un test sont écartés par le choix des fichiers scannés).
func insertColumnsUnion(src, table string) map[string]bool {
	out := map[string]bool{}
	re := regexp.MustCompile(`INSERT\s+(?:OR\s+IGNORE\s+)?INTO\s+` + regexp.QuoteMeta(table) + `\s*\(`)
	for _, loc := range re.FindAllStringIndex(src, -1) {
		rest := src[loc[1]:]
		end := strings.IndexByte(rest, ')')
		if end < 0 {
			continue
		}
		for _, col := range strings.Split(rest[:end], ",") {
			if c := strings.TrimSpace(col); c != "" {
				out[c] = true
			}
		}
	}
	return out
}

func readSourceFile(t *testing.T, name string) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), name))
	if err != nil {
		t.Fatalf("lecture %s: %v", name, err)
	}
	return string(data)
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TestDemoSeedColumns_MatchPersistInserts (F7) : chaque constante exportée est
// EXACTEMENT l'ensemble des colonnes écrites par le persister primaire de sa table.
// Verrou d'honnêteté : changer l'INSERT persist sans mettre à jour la constante
// (ou l'inverse) casse ce test → la constante ne peut pas se périmer en silence.
func TestDemoSeedColumns_MatchPersistInserts(t *testing.T) {
	shared := readSourceFile(t, "shared_persister.go")
	player := readSourceFile(t, "player_persister.go")

	cases := []struct {
		table string
		src   string
		want  []string
	}{
		{"match_registry", shared, MatchRegistryColumns},
		{"match_participants", shared, MatchParticipantsColumns},
		{"match_csrs", shared, MatchCSRColumns},
		{"match_skill_rank", player, MatchSkillRankColumns},
	}
	for _, tc := range cases {
		got := insertColumnsUnion(tc.src, tc.table)
		want := map[string]bool{}
		for _, c := range tc.want {
			want[c] = true
		}
		gotK, wantK := strings.Join(sortedKeys(got), ","), strings.Join(sortedKeys(want), ",")
		if gotK != wantK {
			t.Errorf("%s : constante exportée désynchronisée de l'INSERT persist\n  INSERT: %s\n  const : %s",
				tc.table, gotK, wantK)
		}
	}
}
