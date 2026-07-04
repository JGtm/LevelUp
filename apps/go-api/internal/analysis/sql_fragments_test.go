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
		{"SQLIsBotCol(xuid)", SQLIsBotCol("xuid"), "xuid LIKE 'bid(%'"},
		{"SQLIsBotCol(mp.xuid)", SQLIsBotCol("mp.xuid"), "mp.xuid LIKE 'bid(%'"},
		{"SQLIsNotBotCol(xuid)", SQLIsNotBotCol("xuid"), "xuid NOT LIKE 'bid(%'"},
		{"SQLIsNotBotCol(opp.xuid)", SQLIsNotBotCol("opp.xuid"), "opp.xuid NOT LIKE 'bid(%'"},
		{"SQLStartTimeCanonical(mr)", SQLStartTimeCanonical("mr"), "COALESCE(mr.start_time_utc, mr.start_time AT TIME ZONE 'UTC')"},
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
