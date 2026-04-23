// Package duckdb_test — sanitize_test.go : tests unitaires de sanitizeTimezone.
// Pas de tag d'intégration : aucun accès DuckDB.
package duckdb

import "testing"

func TestSanitizeTimezone(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Europe/Paris", "Europe/Paris"},
		{"America/New_York", "America/New_York"},
		{"Asia/Tokyo", "Asia/Tokyo"},
		{"UTC", "UTC"},
		{"Etc/GMT+5", "Etc/GMT+5"},
		{"", ""},
		// Injections SQL — doivent retourner ""
		{"Europe/Paris'; DROP TABLE x; --", ""},
		{"UTC\x00evil", ""},
		{"Europe/Paris\nSET TimeZone='UTC'", ""},
		{" UTC", ""},
		{"UTC ", ""},
	}

	for _, tc := range cases {
		got := sanitizeTimezone(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeTimezone(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
