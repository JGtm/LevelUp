// Package analysis — sql_fragments_test.go : tests de stabilité des
// fragments SQL canoniques.
//
// Ces tests ne valident pas la sémantique SQL (DuckDB :memory: ferait
// l'affaire pour ça, voir platform/duckdb tests) mais figent les chaînes
// publiées pour empêcher une dérive silencieuse.
package analysis

import "testing"

func TestSQLFragments_StableStrings(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"SQLIsBot", SQLIsBot, "xuid LIKE 'bid(%'"},
		{"SQLIsNotBot", SQLIsNotBot, "xuid NOT LIKE 'bid(%'"},
		{"SQLIsWin", SQLIsWin, "outcome = 2"},
		{
			"SQLWinRateExpr",
			SQLWinRateExpr,
			"COALESCE(CAST(SUM(CASE WHEN outcome = 2 THEN 1 ELSE 0 END) AS DOUBLE) / NULLIF(COUNT(*), 0), 0)",
		},
		{
			"SQLKDRExpr",
			SQLKDRExpr,
			"CAST(SUM(kills) AS DOUBLE) / NULLIF(SUM(deaths), 0)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
			}
		})
	}
}
