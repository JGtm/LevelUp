package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/dbprofiles"
)

// newTitleSyncFixture crée un db_profiles.json v3 avec deux titres pour JGtm et
// un ProfileService câblé dessus (repoRoot = tmp).
func newTitleSyncFixture(t *testing.T) (*ProfileService, string) {
	t.Helper()
	repoRoot := t.TempDir()
	profilesPath := filepath.Join(repoRoot, "db_profiles.json")
	content := `{"version":"3.0","profiles":{
		"halo_infinite":{"JGtm":{"db_path":"d","xuid":"1"}},
		"halo_5":{"JGtm":{"db_path":"d5","xuid":"1"}}
	}}`
	if err := os.WriteFile(profilesPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return NewProfileService(profilesPath, repoRoot), repoRoot
}

func TestSetTitleSyncEnabled_EnforcesMin1(t *testing.T) {
	svc, _ := newTitleSyncFixture(t)
	// Mettre halo_5 en pause : OK (halo_infinite reste actif).
	if err := svc.SetTitleSyncEnabled("halo_5", "JGtm", false); err != nil {
		t.Fatalf("pause halo_5 devrait passer: %v", err)
	}
	// Mettre halo_infinite en pause : refusé (dernier actif).
	if err := svc.SetTitleSyncEnabled("halo_infinite", "JGtm", false); !errors.Is(err, dbprofiles.ErrLastActiveTitle) {
		t.Fatalf("pause du dernier titre actif devrait échouer, reçu %v", err)
	}
}

func TestPurgeTitleData_RemovesDirAndCallsEvictor(t *testing.T) {
	svc, repoRoot := newTitleSyncFixture(t)

	// Créer un faux dossier de données pour (halo_5, JGtm).
	pr := title.NewPathResolver(repoRoot)
	playerDir := pr.PlayerDir("halo_5", "JGtm")
	if err := os.MkdirAll(playerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(playerDir, "stats.duckdb"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	var evicted string
	svc.WithDBEvictor(func(p string) { evicted = p })

	dataRemoved, err := svc.PurgeTitleData("halo_5", "JGtm")
	if err != nil {
		t.Fatalf("PurgeTitleData: %v", err)
	}
	if !dataRemoved {
		t.Fatalf("dataRemoved devrait être true")
	}
	if evicted != pr.PlayerDBPath("halo_5", "JGtm") {
		t.Fatalf("évinceur appelé avec %q, attendu %q", evicted, pr.PlayerDBPath("halo_5", "JGtm"))
	}
	if _, statErr := os.Stat(playerDir); !os.IsNotExist(statErr) {
		t.Fatalf("le dossier joueur devrait être supprimé")
	}
	// L'entrée halo_5 doit avoir disparu de db_profiles.json (mais halo_infinite reste).
	players, _ := svc.store.Load()
	if _, ok := players.Get("halo_5", "JGtm"); ok {
		t.Fatalf("l'entrée halo_5 devrait être retirée du profil")
	}
	if _, ok := players.Get("halo_infinite", "JGtm"); !ok {
		t.Fatalf("l'entrée halo_infinite ne devrait PAS être touchée")
	}
}

func TestPurgeTitleData_RefusesLastActive(t *testing.T) {
	svc, _ := newTitleSyncFixture(t)
	// Purger halo_5 (autorisé), puis halo_infinite (dernier actif → refus).
	if _, err := svc.PurgeTitleData("halo_5", "JGtm"); err != nil {
		t.Fatalf("purge halo_5 devrait passer: %v", err)
	}
	if _, err := svc.PurgeTitleData("halo_infinite", "JGtm"); !errors.Is(err, dbprofiles.ErrLastActiveTitle) {
		t.Fatalf("purge du dernier titre actif devrait échouer, reçu %v", err)
	}
}

func TestPurgeTitleData_NotFound(t *testing.T) {
	svc, _ := newTitleSyncFixture(t)
	if _, err := svc.PurgeTitleData("halo_2", "JGtm"); !errors.Is(err, dbprofiles.ErrEntryNotFound) {
		t.Fatalf("attendu ErrEntryNotFound, reçu %v", err)
	}
}
