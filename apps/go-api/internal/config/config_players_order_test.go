package config

// config_players_order_test.go — LoadPlayers rend ses joueurs dans un ordre
// déterministe (titre, puis slug) : le parcours d'une map Go est aléatoire, et
// tout ce qui en dérive (available_players, writable_player_slugs, joueur par
// défaut) doit être stable d'un appel à l'autre.

import (
	"path/filepath"
	"testing"
	"time"
)

func playerKeys(t *testing.T, cfg *AppConfig, titleFilter ...string) []string {
	t.Helper()
	players, err := cfg.LoadPlayers(titleFilter...)
	if err != nil {
		t.Fatalf("LoadPlayers : %v", err)
	}
	keys := make([]string, 0, len(players))
	for _, p := range players {
		keys = append(keys, p.TitleSlug+"/"+p.PlayerSlug)
	}
	return keys
}

func equalKeys(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestLoadPlayers_OrderIsDeterministic(t *testing.T) {
	cases := []struct {
		name    string
		content string
		filter  []string
		want    []string
	}{
		{
			name: "v3 tous titres",
			content: `{"version":"3.0","profiles":{
				"halo_infinite":{"zed":{"xuid":"1"},"alice":{"xuid":"2"},"mike":{"xuid":"3"},"bob":{"xuid":"4"}},
				"halo_5":{"yan":{"xuid":"5"},"carl":{"xuid":"6"}}}}`,
			want: []string{
				"halo_5/carl", "halo_5/yan",
				"halo_infinite/alice", "halo_infinite/bob", "halo_infinite/mike", "halo_infinite/zed",
			},
		},
		{
			name: "v3 un titre",
			content: `{"version":"3.0","profiles":{
				"halo_infinite":{"zed":{"xuid":"1"},"alice":{"xuid":"2"},"mike":{"xuid":"3"},"bob":{"xuid":"4"}}}}`,
			filter: []string{"halo_infinite"},
			want:   []string{"halo_infinite/alice", "halo_infinite/bob", "halo_infinite/mike", "halo_infinite/zed"},
		},
		{
			name:    "v2",
			content: `{"version":"2.1","profiles":{"zed":{"xuid":"1"},"alice":{"xuid":"2"},"mike":{"xuid":"3"},"bob":{"xuid":"4"}}}`,
			want:    []string{"halo_infinite/alice", "halo_infinite/bob", "halo_infinite/mike", "halo_infinite/zed"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "db_profiles.json")
			writeProfiles(t, path, []byte(tc.content), time.Now().Add(-time.Hour))
			cfg := &AppConfig{DBProfilesPath: path}
			for i := 0; i < 50; i++ {
				if got := playerKeys(t, cfg, tc.filter...); !equalKeys(got, tc.want) {
					t.Fatalf("appel %d : ordre %v, attendu %v", i, got, tc.want)
				}
			}
		})
	}
}
