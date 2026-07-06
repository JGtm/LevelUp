package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
)

// fakeBootstrapLookup implémente authz.UserLookup pour les tests d'ownership bootstrap.
type fakeBootstrapLookup struct {
	byName map[string]*domain.User
	byXUID map[string]*domain.User
}

func (f fakeBootstrapLookup) Get(username string) (*domain.User, error) {
	if u, ok := f.byName[username]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}

func (f fakeBootstrapLookup) GetByXUID(xuid string) (*domain.User, error) {
	if u, ok := f.byXUID[xuid]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}

func ownershipPlayers() []domain.PlayerSummary {
	return []domain.PlayerSummary{
		{PlayerSlug: "alice", Gamertag: "alice", XUID: "222"},
		{PlayerSlug: "bob", Gamertag: "bob", XUID: "999"},
	}
}

func newOwnershipBootstrap(authMode string) *BootstrapService {
	cfg := &config.AppConfig{AuthMode: authMode}
	svc := NewBootstrapService(cfg, &mockBootRepo{})
	return svc.WithUserLookup(fakeBootstrapLookup{
		byName: map[string]*domain.User{
			"alice": {Username: "alice", Role: domain.RoleUser, XUID: "222"},
			"boss":  {Username: "boss", Role: domain.RoleAdmin, XUID: "111"},
		},
	})
}

func slugsOf(players []domain.PlayerSummary) []string {
	out := make([]string, len(players))
	for i, p := range players {
		out[i] = p.PlayerSlug
	}
	return out
}

func TestFilterOwnedPlayers_UserSeesOnlyOwn(t *testing.T) {
	svc := newOwnershipBootstrap("password")
	sess := &domain.SessionData{Username: strPtr("alice")}
	got := slugsOf(svc.filterOwnedPlayers(sess, ownershipPlayers(), nil))
	if len(got) != 1 || got[0] != "alice" {
		t.Fatalf("attendu [alice], obtenu %v", got)
	}
}

func TestFilterOwnedPlayers_AdminSeesAll(t *testing.T) {
	svc := newOwnershipBootstrap("password")
	sess := &domain.SessionData{Username: strPtr("boss")}
	if got := svc.filterOwnedPlayers(sess, ownershipPlayers(), nil); len(got) != 2 {
		t.Fatalf("admin attendu 2 joueurs, obtenu %v", slugsOf(got))
	}
}

func TestFilterOwnedPlayers_NotEnforcedReturnsAll(t *testing.T) {
	svc := newOwnershipBootstrap("none") // auth non activée → pas de filtrage
	sess := &domain.SessionData{Username: strPtr("alice")}
	if got := svc.filterOwnedPlayers(sess, ownershipPlayers(), nil); len(got) != 2 {
		t.Fatalf("mode none attendu 2 joueurs, obtenu %v", slugsOf(got))
	}
}

func TestFilterOwnedPlayers_UnlinkedUserSeesNothing(t *testing.T) {
	svc := newOwnershipBootstrap("password")
	sess := &domain.SessionData{Username: strPtr("charlie")} // absent du store
	if got := svc.filterOwnedPlayers(sess, ownershipPlayers(), nil); len(got) != 0 {
		t.Fatalf("utilisateur non lié attendu 0 joueur, obtenu %v", slugsOf(got))
	}
}

func TestFilterOwnedPlayers_FamilyMemberSeesFamily(t *testing.T) {
	// #21 Phase A : alice (222) et bob (999) dans la même famille → le sélecteur
	// L1 d'alice liste les deux profils (en mode strict elle ne verrait qu'alice).
	svc := newOwnershipBootstrap("password")
	sess := &domain.SessionData{Username: strPtr("alice")}
	family := map[string]bool{"222": true, "999": true}
	got := slugsOf(svc.filterOwnedPlayers(sess, ownershipPlayers(), family))
	if len(got) != 2 {
		t.Fatalf("membre famille attendu 2 joueurs, obtenu %v", got)
	}
}

// Intégration : Build() bout-en-bout (LoadPlayers depuis db_profiles.json réel)
// ne renvoie dans available_players que les profils possédés par l'utilisateur.
func TestBootstrapBuild_AvailablePlayersFilteredByOwnership(t *testing.T) {
	dir := t.TempDir()
	profilesPath := filepath.Join(dir, "db_profiles.json")
	profiles := `{
      "version": "3.0",
      "profiles": {
        "halo_infinite": {
          "alice": {"db_path": "a.duckdb", "xuid": "222", "waypoint_player": "alice"},
          "bob": {"db_path": "b.duckdb", "xuid": "999", "waypoint_player": "bob"}
        }
      }
    }`
	if err := os.WriteFile(profilesPath, []byte(profiles), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.AppConfig{AuthMode: "password", DBProfilesPath: profilesPath}
	svc := NewBootstrapService(cfg, &mockBootRepo{matchCount: 1}).
		WithUserLookup(fakeBootstrapLookup{
			byName: map[string]*domain.User{
				"alice": {Username: "alice", Role: domain.RoleUser, XUID: "222"},
			},
		})

	resp, err := svc.Build(context.Background(), &domain.SessionData{Username: strPtr("alice")})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got := slugsOf(resp.AvailablePlayers); len(got) != 1 || got[0] != "alice" {
		t.Fatalf("available_players attendu [alice], obtenu %v", got)
	}
	if resp.CurrentPlayer == nil || resp.CurrentPlayer.PlayerSlug != "alice" {
		t.Fatalf("current_player attendu alice, obtenu %#v", resp.CurrentPlayer)
	}
}

// S4 (lot S) : GET /players (BuildPlayersList) restreint la liste aux profils
// possédés — auparavant il énumérait TOUS les joueurs sans tenir compte de la
// session (contournait le filtrage de /bootstrap).
func TestBuildPlayersList_FilteredByOwnership(t *testing.T) {
	dir := t.TempDir()
	profilesPath := filepath.Join(dir, "db_profiles.json")
	profiles := `{
      "version": "3.0",
      "profiles": {
        "halo_infinite": {
          "alice": {"db_path": "a.duckdb", "xuid": "222", "waypoint_player": "alice"},
          "bob": {"db_path": "b.duckdb", "xuid": "999", "waypoint_player": "bob"}
        }
      }
    }`
	if err := os.WriteFile(profilesPath, []byte(profiles), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.AppConfig{AuthMode: "password", DBProfilesPath: profilesPath}
	svc := NewBootstrapService(cfg, &mockBootRepo{}).
		WithUserLookup(fakeBootstrapLookup{
			byName: map[string]*domain.User{
				"alice": {Username: "alice", Role: domain.RoleUser, XUID: "222"},
				"boss":  {Username: "boss", Role: domain.RoleAdmin, XUID: "111"},
			},
		})

	// alice (non-admin) ne voit que son propre profil.
	resp, err := svc.BuildPlayersList(context.Background(), &domain.SessionData{Username: strPtr("alice")})
	if err != nil {
		t.Fatalf("BuildPlayersList(alice): %v", err)
	}
	if got := slugsOf(resp.Items); len(got) != 1 || got[0] != "alice" {
		t.Fatalf("S4 : /players non filtré — attendu [alice], obtenu %v", got)
	}

	// admin voit tout le parc.
	respAdmin, err := svc.BuildPlayersList(context.Background(), &domain.SessionData{Username: strPtr("boss")})
	if err != nil {
		t.Fatalf("BuildPlayersList(boss): %v", err)
	}
	if got := slugsOf(respAdmin.Items); len(got) != 2 {
		t.Fatalf("S4 : admin doit voir 2 joueurs, obtenu %v", got)
	}
}
