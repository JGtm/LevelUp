package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
)

// defaultPlayerFixture : trois profils dont l'ordre alphabétique (ordre de LoadPlayers) place
// celui du demandeur ("zed") après un ami de groupe ("alice") et avant un autre ("yan").
func defaultPlayerFixture(t *testing.T, authMode string) *BootstrapService {
	t.Helper()
	dir := t.TempDir()
	profilesPath := filepath.Join(dir, "db_profiles.json")
	profiles := `{"version": "3.0", "profiles": {"halo_infinite": {
      "alice": {"db_path": "a.duckdb", "xuid": "111", "waypoint_player": "alice"},
      "yan": {"db_path": "y.duckdb", "xuid": "333", "waypoint_player": "yan"},
      "zed": {"db_path": "z.duckdb", "xuid": "222", "waypoint_player": "zed"}}}}`
	if err := os.WriteFile(profilesPath, []byte(profiles), 0o644); err != nil {
		t.Fatal(err)
	}
	lookup := fakeBootstrapLookup{byName: map[string]*domain.User{
		"zed-user":  {Username: "zed-user", Role: domain.RoleUser, XUID: "222"},
		"boss":      {Username: "boss", Role: domain.RoleAdmin, XUID: "222"},
		"guest":     {Username: "guest", Role: domain.RoleUser, XUID: "555"},
		"unlinked":  {Username: "unlinked", Role: domain.RoleUser},
		"admin-nox": {Username: "admin-nox", Role: domain.RoleAdmin},
	}}
	family := func(xuid string) map[string]bool {
		if xuid == "222" {
			return map[string]bool{"111": true, "222": true, "333": true}
		}
		return map[string]bool{"111": true, "333": true, "555": true}
	}
	cfg := &config.AppConfig{AuthMode: authMode, DBProfilesPath: profilesPath}
	return NewBootstrapService(cfg, &mockBootRepo{matchCount: 1}).
		WithUserLookup(lookup).WithCoMemberResolver(family)
}

func buildDefaults(t *testing.T, svc *BootstrapService, sess *domain.SessionData) (current, listDefault string) {
	t.Helper()
	resp, err := svc.Build(context.Background(), sess)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if resp.CurrentPlayer != nil {
		current = resp.CurrentPlayer.PlayerSlug
	}
	list, err := svc.BuildPlayersList(context.Background(), sess)
	if err != nil {
		t.Fatalf("BuildPlayersList: %v", err)
	}
	if list.DefaultPlayerSlug != nil {
		listDefault = *list.DefaultPlayerSlug
	}
	return current, listDefault
}

// Le joueur par défaut d'un utilisateur connecté est SON profil, pas le premier par ordre
// alphabétique de ceux que son groupe lui rend visibles.
func TestBootstrap_DefaultPlayerIsTheCallersOwnProfile(t *testing.T) {
	cases := []struct {
		name, user, want string
	}{
		{"utilisateur avec ses amis devant lui", "zed-user", "zed"},
		{"admin lié à un profil", "boss", "zed"},
		{"aucun profil en propre : premier visible", "guest", "alice"},
		{"admin sans xuid : premier visible", "admin-nox", "alice"},
		{"compte sans xuid : aucun profil visible", "unlinked", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := defaultPlayerFixture(t, "password")
			current, listDefault := buildDefaults(t, svc, &domain.SessionData{Username: strPtr(tc.user)})
			if current != tc.want {
				t.Fatalf("current_player = %q, attendu %q", current, tc.want)
			}
			if listDefault != tc.want {
				t.Fatalf("default_player_slug = %q, attendu %q", listDefault, tc.want)
			}
		})
	}
}

// Le choix mémorisé en session reste prioritaire sur le profil par défaut.
func TestBootstrap_SessionPlayerChoiceBeatsTheDefault(t *testing.T) {
	svc := defaultPlayerFixture(t, "password")
	sess := &domain.SessionData{Username: strPtr("zed-user"), CurrentPlayerSlug: strPtr("yan")}
	resp, err := svc.Build(context.Background(), sess)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if resp.CurrentPlayer == nil || resp.CurrentPlayer.PlayerSlug != "yan" {
		t.Fatalf("current_player = %+v, attendu yan (choix de session)", resp.CurrentPlayer)
	}
}

// Sans propriété appliquée, personne n'a de profil « en propre » : ordre stable, premier profil.
func TestBootstrap_DefaultPlayerWithoutEnforcementIsTheFirst(t *testing.T) {
	svc := defaultPlayerFixture(t, "none")
	current, listDefault := buildDefaults(t, svc, &domain.SessionData{Username: strPtr("zed-user")})
	if current != "alice" || listDefault != "alice" {
		t.Fatalf("current=%q default=%q, attendu alice et alice", current, listDefault)
	}
}
