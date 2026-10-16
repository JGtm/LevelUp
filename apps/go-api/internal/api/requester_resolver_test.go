package api

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
)

type requesterUsers map[string]*domain.User

func (u requesterUsers) Get(name string) (*domain.User, error) {
	if x, ok := u[name]; ok {
		return x, nil
	}
	return nil, errors.New("inconnu")
}

func (requesterUsers) GetByXUID(string) (*domain.User, error) { return nil, errors.New("inconnu") }

func requesterConfig(t *testing.T, authMode string) *config.AppConfig {
	t.Helper()
	path := filepath.Join(t.TempDir(), "db_profiles.json")
	profiles := `{"version": "3.0", "profiles": {"halo_infinite": {
	  "Mallory": {"db_path": "m.duckdb", "xuid": "222", "waypoint_player": "Mallory"},
	  "JGtm": {"db_path": "j.duckdb", "xuid": "999", "waypoint_player": "JGtm"}}}}`
	if err := os.WriteFile(path, []byte(profiles), 0o600); err != nil {
		t.Fatal(err)
	}
	return &config.AppConfig{AuthMode: authMode, DBProfilesPath: path}
}

func sessionCtx(username, currentPlayer string) context.Context {
	sess := &domain.SessionData{SessionID: "s", Username: &username, CurrentPlayerSlug: &currentPlayer}
	return middleware.InjectSession(context.Background(), sess)
}

// TestRequesterResolver_OwnedProfileNotCurrentPlayer : en propriété appliquée, le
// demandeur est le profil du xuid lié, quel que soit le joueur courant de la session.
func TestRequesterResolver_OwnedProfileNotCurrentPlayer(t *testing.T) {
	users := requesterUsers{
		"mallory": {Username: "mallory", Role: domain.RoleUser, XUID: "222"},
		"boss":    {Username: "boss", Role: domain.RoleAdmin},
	}
	resolve := requesterPlayerSlugResolver(requesterConfig(t, "password"), users)
	if got := resolve(sessionCtx("mallory", "JGtm")); got != "Mallory" {
		t.Fatalf("demandeur %q, attendu Mallory (joueur courant forgé ignoré)", got)
	}
	if got := resolve(sessionCtx("boss", "JGtm")); got != "" {
		t.Fatalf("admin sans profil : demandeur %q, attendu vide", got)
	}
	if got := resolve(sessionCtx("inconnu", "JGtm")); got != "" {
		t.Fatalf("utilisateur inconnu : demandeur %q, attendu vide", got)
	}
}

// TestRequesterResolver_NotEnforcedUsesSelectedPlayer : sans propriété appliquée,
// aucune identité n'est opposable ; le joueur choisi dans l'interface est rendu.
func TestRequesterResolver_NotEnforcedUsesSelectedPlayer(t *testing.T) {
	resolve := requesterPlayerSlugResolver(requesterConfig(t, "none"), requesterUsers{})
	if got := resolve(sessionCtx("x", "JGtm")); got != "JGtm" {
		t.Fatalf("demandeur %q, attendu JGtm", got)
	}
}
