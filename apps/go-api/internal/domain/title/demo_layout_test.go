package title

import (
	"path/filepath"
	"testing"
)

// TestDemoLayout_Disposition fige la disposition de l'arbre démo : PLAT pour le titre par
// défaut (byte-identique à la démo mono-titre que produit seed-demo), `titles/<slug>` pour
// un titre additionnel, état et exécution sous `auth/` et `runtime/`.
func TestDemoLayout_Disposition(t *testing.T) {
	root := filepath.Join("X:", "demo")
	l := NewDemoLayout(root)
	h5 := filepath.Join(root, "titles", "halo_5")
	cas := map[string][2]string{
		"TitleDir(défaut)":           {l.TitleDir(DefaultSlug), root},
		"TitleDir(vide)":             {l.TitleDir(""), root},
		"TitleDir(halo_5)":           {l.TitleDir("halo_5"), h5},
		"SharedDBPath(défaut)":       {l.SharedDBPath(DefaultSlug), filepath.Join(root, "warehouse", "shared_matches_v2.duckdb")},
		"MetadataDBPath(halo_5)":     {l.MetadataDBPath("halo_5"), filepath.Join(h5, "warehouse", "metadata.duckdb")},
		"SharedSocialDBPath(défaut)": {l.SharedSocialDBPath(""), filepath.Join(root, "warehouse", "shared_social.duckdb")},
		"SharedPVEDBPath(défaut)":    {l.SharedPVEDBPath(""), filepath.Join(root, "warehouse", "shared_pve.duckdb")},
		"PlayerDBPath(défaut, DEMO)": {l.PlayerDBPath("", "DEMO"), filepath.Join(root, "players", "DEMO", "stats.duckdb")},
		"PlayerDBPath(halo_5, DEMO)": {l.PlayerDBPath("halo_5", "DEMO"), filepath.Join(h5, "players", "DEMO", "stats.duckdb")},
		"TitleSettingsPath(halo_5)":  {l.TitleSettingsPath("halo_5"), filepath.Join(h5, "settings.json")},
		"DBProfilesPath":             {l.DBProfilesPath(), filepath.Join(root, "db_profiles.json")},
		"AppSettingsPath":            {l.AppSettingsPath(), filepath.Join(root, "app_settings.json")},
		"AuthDir":                    {l.AuthDir(), filepath.Join(root, "auth")},
		"WatcherTokensDir":           {l.WatcherTokensDir(), filepath.Join(root, "auth", "watcher_tokens")},
		"SessionDir":                 {l.SessionDir(), filepath.Join(root, "runtime", "sessions")},
		"LogsDir":                    {l.LogsDir(), filepath.Join(root, "runtime", "logs")},
		"RuntimePaths.RepoRoot":      {l.RuntimePaths().RepoRoot(), filepath.Join(root, "runtime")},
		"ReplayArtifactPath(défaut)": {l.ReplayArtifactPath("", "abcd1234-0000-0000-0000-000000000000"),
			filepath.Join(root, "replays", "artifacts", "abcd1234.json")},
		"ReplayFilmsCacheRoot(halo_5)": {l.ReplayFilmsCacheRoot("halo_5"), filepath.Join(h5, "replays", "films")},
		"ReplayIndexPath(défaut)":      {l.ReplayIndexPath(DefaultSlug), filepath.Join(root, "replays", "index.json")},
	}
	for nom, v := range cas {
		if v[0] != v[1] {
			t.Errorf("%s = %s, attendu %s", nom, v[0], v[1])
		}
	}
}
