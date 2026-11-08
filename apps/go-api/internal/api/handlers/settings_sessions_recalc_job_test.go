package handlers

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/jobs"
	settings_platform "levelup/go-api/internal/platform/settings"
)

// TestPostRecalculateSessions_EchecDUnJoueurFaitEchouerLeJob : un joueur dont le recalcul échoue
// (ici, sa base joueur est illisible : un dossier occupe son chemin) fait finir le job en échec,
// avec le compte des joueurs en échec. Le 2026-10-19, le job se déclarait réussi (« 0 matchs mis à jour ») alors que
// tous les joueurs Halo 5 avaient échoué.
func TestPostRecalculateSessions_EchecDUnJoueurFaitEchouerLeJob(t *testing.T) {
	root := t.TempDir()
	profilesPath := filepath.Join(root, "db_profiles.json")
	profiles := `{"version":"3.0","admin":"H5Only","profiles":{"halo_5":{"H5Only":{"xuid":"2001"}}}}`
	if err := os.WriteFile(profilesPath, []byte(profiles), 0o600); err != nil {
		t.Fatalf("db_profiles: %v", err)
	}
	cfg := &config.AppConfig{RepoRoot: root, DBProfilesPath: profilesPath}
	// Un dossier à la place de la base joueur : son ouverture échoue.
	if err := os.MkdirAll(titlePkg.NewPathResolver(root).PlayerDBPath("halo_5", "H5Only"), 0o750); err != nil {
		t.Fatal(err)
	}
	store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
	h := NewSettingsHandler(cfg, settings_platform.NewStore(filepath.Join(root, "app_settings.json")), store)

	out, err := h.handlePostRecalculateSessions(context.Background(), nil)
	if err != nil {
		t.Fatalf("handlePostRecalculateSessions: %v", err)
	}
	var j *domain.AsyncJobStatus
	for deadline := time.Now().Add(30 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if j = store.Get(out.Body.JobID); j != nil && j.IsTerminal() {
			break
		}
	}
	if j == nil || !j.IsTerminal() {
		t.Fatalf("job non terminé: %+v", j)
	}
	if j.Status != domain.JobStatusFailed {
		t.Fatalf("statut %q, attendu %q : un joueur a échoué", j.Status, domain.JobStatusFailed)
	}
	if j.Error == nil || j.Error.Code != "sessions_recalc_failed" || !j.Error.Retryable {
		t.Errorf("erreur du job %+v, attendu le code sessions_recalc_failed, relançable", j.Error)
	}
	if j.Result["players"] != 1 || j.Result["players_failed"] != 1 {
		t.Errorf("résultat %v, attendu 1 joueur dont 1 en échec", j.Result)
	}
}

// TestPostRecalculateSessions_ProfilAuthOnlyEcarte : un profil auth_only n'a pas de base joueur
// (il ne sert qu'à porter un jeton) : le recalcul ne le compte pas et ne l'ouvre pas. Ici un
// dossier occupe le chemin de sa base joueur, de sorte qu'une ouverture ferait échouer le job.
func TestPostRecalculateSessions_ProfilAuthOnlyEcarte(t *testing.T) {
	root := t.TempDir()
	profilesPath := filepath.Join(root, "db_profiles.json")
	profiles := `{"version":"3.0","admin":"TokenOnly","profiles":{"halo_5":{"TokenOnly":{"xuid":"2002","auth_only":true}}}}`
	if err := os.WriteFile(profilesPath, []byte(profiles), 0o600); err != nil {
		t.Fatalf("db_profiles: %v", err)
	}
	cfg := &config.AppConfig{RepoRoot: root, DBProfilesPath: profilesPath}
	playerDB := titlePkg.NewPathResolver(root).PlayerDBPath("halo_5", "TokenOnly")
	if err := os.MkdirAll(playerDB, 0o750); err != nil {
		t.Fatal(err)
	}
	store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
	h := NewSettingsHandler(cfg, settings_platform.NewStore(filepath.Join(root, "app_settings.json")), store)

	out, err := h.handlePostRecalculateSessions(context.Background(), nil)
	if err != nil {
		t.Fatalf("handlePostRecalculateSessions: %v", err)
	}
	var j *domain.AsyncJobStatus
	for deadline := time.Now().Add(30 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if j = store.Get(out.Body.JobID); j != nil && j.IsTerminal() {
			break
		}
	}
	if j == nil || !j.IsTerminal() {
		t.Fatalf("job non terminé: %+v", j)
	}
	if j.Status != domain.JobStatusSucceeded {
		t.Fatalf("statut %q (erreur %+v), attendu %q : le profil auth_only ne doit pas être ouvert", j.Status, j.Error, domain.JobStatusSucceeded)
	}
	if j.Result["players"] != 0 || j.Result["players_failed"] != 0 {
		t.Errorf("résultat %v, attendu 0 joueur compté et 0 en échec", j.Result)
	}
}
