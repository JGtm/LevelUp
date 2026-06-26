package livesync

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
)

// validPNG est la signature magique PNG (8 octets) suivie d'un octet quelconque.
var validPNG = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0xff}

func TestIsPNGBytes(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
		want bool
	}{
		{"nil", nil, false},
		{"vide", []byte{}, false},
		{"trop court (7 octets de la signature)", []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a}, false},
		{"signature exacte 8 octets", []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}, true},
		{"signature + payload", validPNG, true},
		// Corps HTML/JSON renvoyé par le CDN sous un 200 inattendu : longueur OK mais
		// 1er octet faux → rejet (c'est exactement le cas de défense documenté).
		{"premier octet faux (HTML '<')", []byte{0x3c, 0x68, 0x74, 0x6d, 0x6c, 0x3e, 0x0a, 0x0a, 0x0a}, false},
		{"dernier octet de signature faux", []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x00}, false},
		{"octet intermédiaire faux (0x4e->0x4f)", []byte{0x89, 0x50, 0x4f, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}, false},
		// JPEG (FF D8) : longueur suffisante mais mauvaise signature.
		{"signature JPEG", []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isPNGBytes(tc.in); got != tc.want {
				t.Errorf("isPNGBytes(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestAppearanceAssetSlug(t *testing.T) {
	cases := []struct {
		name     string
		gamertag string
		want     string
	}{
		{"minuscule simple", "jgtm", "jgtm"},
		{"mise en minuscule", "JGtm", "jgtm"},
		{"espaces remplacés par tiret", "Hello World", "hello-world"},
		{"trim des tirets de bordure (espaces avant/après)", "  spaced  ", "spaced"},
		{"caractères spéciaux -> tiret", "a@b#c", "a-b-c"},
		{"chiffres et underscore conservés", "Player_123", "player_123"},
		{"tiret existant conservé", "a-b", "a-b"},
		{"unicode/accents remplacés", "café", "caf"},
		// Bornes : trim '-' aux extrémités après remplacement.
		{"tirets de tête et de queue trimés", "--abc--", "abc"},
		{"ponctuation autour", ".abc.", "abc"},
		// Fallback "player" quand le résultat est vide.
		{"gamertag vide -> player", "", "player"},
		{"que des espaces -> player", "   ", "player"},
		{"que des caractères spéciaux -> player", "@#$%", "player"},
		{"que des tirets -> player", "----", "player"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := appearanceAssetSlug(tc.gamertag); got != tc.want {
				t.Errorf("appearanceAssetSlug(%q) = %q, want %q", tc.gamertag, got, tc.want)
			}
		})
	}
}

// writeProfilesV3 écrit un db_profiles.json v3 minimal (title_slug -> gamertag ->
// entry) et retourne un AppConfig pointant dessus. xuid vide toléré (non requis ici).
func writeProfilesV3(t *testing.T, raw string) *config.AppConfig {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "db_profiles.json")
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write db_profiles.json: %v", err)
	}
	return &config.AppConfig{RepoRoot: dir, DBProfilesPath: path}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func TestOtherPlayerGamertags(t *testing.T) {
	// 3 joueurs sur 2 titres + un gamertag vide (clé "") qui doit être ignoré.
	cfg := writeProfilesV3(t, `{
		"version": "3.0",
		"profiles": {
			"halo_5": {
				"JGtm": {"xuid": "1"},
				"Madina": {"xuid": "2"}
			},
			"halo_infinite": {
				"Choco": {"xuid": "3"},
				"": {"xuid": "4"}
			}
		}
	}`)

	got := otherPlayerGamertags(cfg, "JGtm")

	if contains(got, "JGtm") {
		t.Errorf("le viewer ne doit pas figurer dans la liste: %v", got)
	}
	if contains(got, "") {
		t.Errorf("un gamertag vide ne doit jamais figurer: %v", got)
	}
	for _, want := range []string{"Madina", "Choco"} {
		if !contains(got, want) {
			t.Errorf("%q attendu dans la liste, got %v", want, got)
		}
	}
	if len(got) != 2 {
		t.Errorf("attendu 2 autres joueurs (Madina, Choco), got %d: %v", len(got), got)
	}
}

func TestOtherPlayerGamertags_LoadError(t *testing.T) {
	// JSON malformé -> LoadPlayers renvoie une erreur -> liste vide (best-effort).
	cfg := writeProfilesV3(t, `{ this is not json `)
	if got := otherPlayerGamertags(cfg, "JGtm"); len(got) != 0 {
		t.Errorf("err LoadPlayers -> liste vide attendue, got %v", got)
	}
}

func TestOtherPlayerGamertags_MissingFile(t *testing.T) {
	// Fichier absent -> LoadPlayers renvoie ([], nil) -> liste vide, pas de panic.
	cfg := &config.AppConfig{RepoRoot: t.TempDir(), DBProfilesPath: filepath.Join(t.TempDir(), "absent.json")}
	if got := otherPlayerGamertags(cfg, "JGtm"); len(got) != 0 {
		t.Errorf("fichier absent -> liste vide attendue, got %v", got)
	}
}

func TestAllResolverGamertags_ViewerHeadAndDedup(t *testing.T) {
	// Le viewer (JGtm) est AUSSI déclaré dans les profils -> ne doit pas être dupliqué.
	cfg := writeProfilesV3(t, `{
		"version": "3.0",
		"profiles": {
			"halo_5": {
				"JGtm": {"xuid": "1"},
				"Madina": {"xuid": "2"}
			},
			"halo_infinite": {
				"Choco": {"xuid": "3"},
				"": {"xuid": "4"}
			}
		}
	}`)

	got := allResolverGamertags(cfg, "JGtm")

	if len(got) == 0 || got[0] != "JGtm" {
		t.Fatalf("le viewer doit être en tête, got %v", got)
	}
	// Dédup : JGtm apparaît une seule fois bien qu'il soit aussi dans les profils.
	count := 0
	for _, g := range got {
		if g == "JGtm" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("viewer dédupliqué attendu (1 occurrence), got %d dans %v", count, got)
	}
	if contains(got, "") {
		t.Errorf("gamertag vide ne doit jamais figurer: %v", got)
	}
	for _, want := range []string{"Madina", "Choco"} {
		if !contains(got, want) {
			t.Errorf("%q attendu, got %v", want, got)
		}
	}
	// viewer + Madina + Choco (le vide ignoré, JGtm non dupliqué).
	if len(got) != 3 {
		t.Errorf("attendu 3 comptes, got %d: %v", len(got), got)
	}
}

func TestAllResolverGamertags_LoadError_ViewerOnly(t *testing.T) {
	// JSON malformé -> LoadPlayers err -> seul le viewer (toujours en tête).
	cfg := writeProfilesV3(t, `nope`)
	got := allResolverGamertags(cfg, "JGtm")
	if len(got) != 1 || got[0] != "JGtm" {
		t.Errorf("err LoadPlayers -> [viewer] attendu, got %v", got)
	}
}
