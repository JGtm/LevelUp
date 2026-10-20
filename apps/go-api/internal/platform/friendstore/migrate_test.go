package friendstore

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain"
)

// writeAppSettings écrit une fixture app_settings.json minimale portant la clé
// legacy `friend_gamertags` (la migration ne lit QUE cette clé).
func writeAppSettings(t *testing.T, dir, body string) string {
	t.Helper()
	path := filepath.Join(dir, "app_settings.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("écriture app_settings.json = %v", err)
	}
	return path
}

func TestMigrate_EachPlayerInheritsGlobalListMinusSelf(t *testing.T) {
	dir := t.TempDir()
	settingsPath := writeAppSettings(t, dir,
		`{"session_gap_minutes":120,"friend_gamertags":["Alpha","Bravo","Charlie"]}`)
	store := NewFriendStore(filepath.Join(dir, "player_friends.json"))

	players := []domain.PlayerSummary{
		{Gamertag: "Alpha", XUID: "xuid_alpha"},
		{Gamertag: "Bravo", XUID: "xuid_bravo"},
	}
	created, err := MigrateFromAppSettings(store, settingsPath, players)
	if err != nil {
		t.Fatalf("MigrateFromAppSettings = %v", err)
	}
	if created != 2 {
		t.Fatalf("created = %d, want 2", created)
	}

	alpha, _ := store.Get("xuid_alpha")
	if len(alpha) != 2 || alpha[0] != "Bravo" || alpha[1] != "Charlie" {
		t.Errorf("amis d'Alpha = %v, want [Bravo Charlie] (soi-même exclu)", alpha)
	}
	bravo, _ := store.Get("xuid_bravo")
	if len(bravo) != 2 || bravo[0] != "Alpha" || bravo[1] != "Charlie" {
		t.Errorf("amis de Bravo = %v, want [Alpha Charlie]", bravo)
	}
}

func TestMigrate_SelfExclusionIsCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	settingsPath := writeAppSettings(t, dir, `{"friend_gamertags":["ALPHA","Bravo"]}`)
	store := NewFriendStore(filepath.Join(dir, "player_friends.json"))

	if _, err := MigrateFromAppSettings(store, settingsPath,
		[]domain.PlayerSummary{{Gamertag: "alpha", XUID: "xuid_alpha"}}); err != nil {
		t.Fatalf("MigrateFromAppSettings = %v", err)
	}
	got, _ := store.Get("xuid_alpha")
	if len(got) != 1 || got[0] != "Bravo" {
		t.Errorf("amis = %v, want [Bravo] (ALPHA == alpha exclu)", got)
	}
}

func TestMigrate_IsIdempotent(t *testing.T) {
	dir := t.TempDir()
	settingsPath := writeAppSettings(t, dir, `{"friend_gamertags":["Alpha","Bravo"]}`)
	store := NewFriendStore(filepath.Join(dir, "player_friends.json"))
	players := []domain.PlayerSummary{{Gamertag: "Delta", XUID: "xuid_delta"}}

	if _, err := MigrateFromAppSettings(store, settingsPath, players); err != nil {
		t.Fatalf("1re migration = %v", err)
	}
	// Édition utilisateur postérieure : la 2e migration ne doit pas l'écraser.
	if _, err := store.Set("xuid_delta", "Delta", []string{"Echo"}); err != nil {
		t.Fatalf("Set = %v", err)
	}

	created, err := MigrateFromAppSettings(store, settingsPath, players)
	if err != nil {
		t.Fatalf("2e migration = %v", err)
	}
	if created != 0 {
		t.Errorf("created = %d au 2e passage, want 0 (idempotence)", created)
	}
	got, _ := store.Get("xuid_delta")
	if len(got) != 1 || got[0] != "Echo" {
		t.Errorf("amis = %v, want [Echo] — la migration a écrasé une édition utilisateur", got)
	}
}

func TestMigrate_SkipsAuthOnlyAndXUIDLessProfiles(t *testing.T) {
	dir := t.TempDir()
	settingsPath := writeAppSettings(t, dir, `{"friend_gamertags":["Alpha"]}`)
	store := NewFriendStore(filepath.Join(dir, "player_friends.json"))

	created, err := MigrateFromAppSettings(store, settingsPath, []domain.PlayerSummary{
		{Gamertag: "Pool", XUID: "xuid_pool", AuthOnly: true},
		{Gamertag: "SansXuid", XUID: ""},
		{Gamertag: "Vrai", XUID: "xuid_vrai"},
	})
	if err != nil {
		t.Fatalf("MigrateFromAppSettings = %v", err)
	}
	if created != 1 {
		t.Fatalf("created = %d, want 1 (seul le vrai profil)", created)
	}
	if got, _ := store.Get("xuid_pool"); len(got) != 0 {
		t.Errorf("profil auth_only doté d'amis = %v", got)
	}
}

func TestMigrate_SameXUIDOnTwoTitlesWrittenOnce(t *testing.T) {
	dir := t.TempDir()
	settingsPath := writeAppSettings(t, dir, `{"friend_gamertags":["Alpha"]}`)
	store := NewFriendStore(filepath.Join(dir, "player_friends.json"))

	created, err := MigrateFromAppSettings(store, settingsPath, []domain.PlayerSummary{
		{Gamertag: "Vrai", XUID: "xuid_vrai", TitleSlug: "halo_infinite"},
		{Gamertag: "Vrai", XUID: "xuid_vrai", TitleSlug: "halo_5"},
	})
	if err != nil {
		t.Fatalf("MigrateFromAppSettings = %v", err)
	}
	if created != 1 {
		t.Errorf("created = %d, want 1 (un xuid = une liste, tous titres confondus)", created)
	}
}

func TestMigrate_NoLegacyKeyOrEmptyIsNoOp(t *testing.T) {
	for name, body := range map[string]string{
		"clé absente": `{"session_gap_minutes":120}`,
		"liste vide":  `{"friend_gamertags":[]}`,
		"null":        `{"friend_gamertags":null}`,
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			settingsPath := writeAppSettings(t, dir, body)
			path := filepath.Join(dir, "player_friends.json")
			created, err := MigrateFromAppSettings(NewFriendStore(path), settingsPath,
				[]domain.PlayerSummary{{Gamertag: "Vrai", XUID: "xuid_vrai"}})
			if err != nil {
				t.Fatalf("MigrateFromAppSettings = %v", err)
			}
			if created != 0 {
				t.Errorf("created = %d, want 0", created)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Errorf("player_friends.json créé sans rien à migrer (err=%v)", err)
			}
		})
	}
}

func TestMigrate_MissingAppSettingsIsNoOp(t *testing.T) {
	dir := t.TempDir()
	store := NewFriendStore(filepath.Join(dir, "player_friends.json"))
	created, err := MigrateFromAppSettings(store, filepath.Join(dir, "absent.json"),
		[]domain.PlayerSummary{{Gamertag: "Vrai", XUID: "xuid_vrai"}})
	if err != nil {
		t.Fatalf("MigrateFromAppSettings = %v, want nil (instance neuve)", err)
	}
	if created != 0 {
		t.Errorf("created = %d, want 0", created)
	}
}

func TestMigrate_CorruptedAppSettingsReturnsError(t *testing.T) {
	dir := t.TempDir()
	settingsPath := writeAppSettings(t, dir, "{ pas du json")
	store := NewFriendStore(filepath.Join(dir, "player_friends.json"))
	if _, err := MigrateFromAppSettings(store, settingsPath, nil); err == nil {
		t.Fatal("MigrateFromAppSettings sur settings corrompu = nil, want erreur")
	}
}
