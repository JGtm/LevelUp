// Package api — registry_notifications_test.go : tests des helpers purs du
// MediaRecipientResolver (xuidsToSlugs + joinComma). Les helpers DB-bound
// (loadRecentMediaMatchIDs, loadParticipantXUIDs) sont couverts via les
// tests d'intégration repo (DuckDB).
package wire

import (
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
)

func TestJoinComma(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{[]string{}, ""},
		{[]string{"?"}, "?"},
		{[]string{"?", "?"}, "?,?"},
		{[]string{"a", "b", "c"}, "a,b,c"},
	}
	for _, tc := range cases {
		got := joinComma(tc.in)
		if got != tc.want {
			t.Errorf("joinComma(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// writeFakeProfiles construit un db_profiles.json v2 minimal avec les joueurs fournis.
// Format v2 : {"profiles": {"gamertag": {"xuid": "..."}}}.
func writeFakeProfiles(t *testing.T, players []domain.PlayerSummary) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "db_profiles.json")
	body := `{"profiles":{`
	for i, p := range players {
		if i > 0 {
			body += ","
		}
		body += `"` + p.Gamertag + `":{"xuid":"` + p.XUID + `"}`
	}
	body += `}}`
	if err := writeFile(path, body); err != nil {
		t.Fatalf("writeFakeProfiles: %v", err)
	}
	return path
}

func writeFile(path, content string) error {
	return writeFileImpl(path, content)
}

// Note : en v2 du format db_profiles.json, PlayerSlug = Gamertag.
// Les tests utilisent ce mapping.

func TestXuidsToSlugs_ExcludeUploader_DedupePlafond(t *testing.T) {
	cfg := &config.AppConfig{}
	cfg.DBProfilesPath = writeFakeProfiles(t, []domain.PlayerSummary{
		{Gamertag: "Alice", XUID: "xuid-a"},
		{Gamertag: "Bob", XUID: "xuid-b"},
		{Gamertag: "Carol", XUID: "xuid-c"},
		{Gamertag: "Dave", XUID: "xuid-d"},
	})

	xuids := []string{"xuid-a", "xuid-b", "xuid-c", "xuid-d", "xuid-unknown"}
	got := xuidsToSlugs(cfg, "Alice", xuids, 5)

	// Alice exclu (uploader). xuid-unknown non mappé. Bob/Carol/Dave gardés (max=5).
	want := map[string]bool{"Bob": true, "Carol": true, "Dave": true}
	if len(got) != 3 {
		t.Fatalf("expected 3 recipients, got %d (%v)", len(got), got)
	}
	for _, slug := range got {
		if !want[slug] {
			t.Errorf("unexpected slug %q in result", slug)
		}
	}
}

func TestXuidsToSlugs_PlafondMax(t *testing.T) {
	cfg := &config.AppConfig{}
	cfg.DBProfilesPath = writeFakeProfiles(t, []domain.PlayerSummary{
		{Gamertag: "P1", XUID: "x1"},
		{Gamertag: "P2", XUID: "x2"},
		{Gamertag: "P3", XUID: "x3"},
	})

	got := xuidsToSlugs(cfg, "uploader", []string{"x1", "x2", "x3"}, 2)
	if len(got) != 2 {
		t.Errorf("plafond=2 expected 2 results, got %d", len(got))
	}
}

func TestXuidsToSlugs_NoMatchingProfiles(t *testing.T) {
	cfg := &config.AppConfig{}
	cfg.DBProfilesPath = writeFakeProfiles(t, []domain.PlayerSummary{
		{Gamertag: "Alice", XUID: "xuid-a"},
	})
	got := xuidsToSlugs(cfg, "Alice", []string{"xuid-z", "xuid-y"}, 5)
	if len(got) != 0 {
		t.Errorf("expected 0 (no match), got %d (%v)", len(got), got)
	}
}
