package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/jobs"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
)

// recalcFixture : un service sur une configuration réelle (db_profiles.json donné) et un registre de
// jobs fichier.
func recalcFixture(t *testing.T, profiles string) (*SyncJobService, *jobs.Store, string) {
	t.Helper()
	root := t.TempDir()
	profilesPath := filepath.Join(root, "db_profiles.json")
	if err := os.WriteFile(profilesPath, []byte(profiles), 0o600); err != nil {
		t.Fatalf("db_profiles: %v", err)
	}
	cfg := &config.AppConfig{RepoRoot: root, DBProfilesPath: profilesPath}
	store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
	settings := settings_platform.NewStore(filepath.Join(root, "app_settings.json"))
	return NewSyncJobService(cfg, settings, store, nil), store, root
}

// runRecalc lance le recalcul et rend le job terminal.
func runRecalc(t *testing.T, s *SyncJobService, store *jobs.Store) *domain.AsyncJobStatus {
	t.Helper()
	job, err := s.StartSessionsRecalc(context.Background())
	if err != nil {
		t.Fatalf("StartSessionsRecalc: %v", err)
	}
	var j *domain.AsyncJobStatus
	for deadline := time.Now().Add(30 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if j = store.Get(job.JobID); j != nil && j.IsTerminal() {
			return j
		}
	}
	t.Fatalf("job non terminé: %+v", j)
	return nil
}

// TestSessionsRecalcPaths : un profil Halo 5 se recalcule sur les bases de Halo 5 (joueur ET partagée),
// un profil sans titre sur celles du titre par défaut.
func TestSessionsRecalcPaths(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	for _, c := range []struct{ titre, attendu string }{
		{"halo_5", "halo_5"},
		{titlePkg.DefaultSlug, titlePkg.DefaultSlug},
		{"", titlePkg.DefaultSlug},
	} {
		player, shared := sessionsRecalcPaths(cfg, domain.PlayerSummary{Gamertag: "JGtm", TitleSlug: c.titre})
		if want := pr.PlayerDBPath(c.attendu, "JGtm"); player != want {
			t.Errorf("titre %q : base joueur %q, attendu %q", c.titre, player, want)
		}
		if want := pr.SharedDBPath(c.attendu); shared != want {
			t.Errorf("titre %q : base partagée %q, attendu %q", c.titre, shared, want)
		}
	}
}

// La route ouvre la base joueur du titre du profil : un joueur suivi seulement sur Halo 5 ne voit pas
// créer une base Infinite vide.
func TestStartSessionsRecalc_ProfilHalo5SurLesBasesDeHalo5(t *testing.T) {
	s, store, root := recalcFixture(t,
		`{"version":"3.0","admin":"H5Only","profiles":{"halo_5":{"H5Only":{"xuid":"2001"}}}}`)
	runRecalc(t, s, store)
	pr := titlePkg.NewPathResolver(root)
	if _, err := os.Stat(pr.PlayerDBPath("halo_5", "H5Only")); err != nil {
		t.Errorf("la base joueur de Halo 5 devait être ouverte: %v", err)
	}
	if _, err := os.Stat(pr.PlayerDBPath(titlePkg.DefaultSlug, "H5Only")); !os.IsNotExist(err) {
		t.Errorf("aucune base du titre par défaut ne devait être créée pour ce profil (stat: %v)", err)
	}
}

// Un joueur dont le recalcul échoue (ici, sa base joueur est illisible : un dossier occupe son chemin)
// fait finir le job en échec, avec le compte des joueurs en échec.
func TestStartSessionsRecalc_EchecDUnJoueurFaitEchouerLeJob(t *testing.T) {
	s, store, root := recalcFixture(t,
		`{"version":"3.0","admin":"H5Only","profiles":{"halo_5":{"H5Only":{"xuid":"2001"}}}}`)
	if err := os.MkdirAll(titlePkg.NewPathResolver(root).PlayerDBPath("halo_5", "H5Only"), 0o750); err != nil {
		t.Fatal(err)
	}
	j := runRecalc(t, s, store)
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

// Un profil auth_only n'a pas de base joueur (il ne sert qu'à porter un jeton) : le recalcul ne le
// compte pas et ne l'ouvre pas. Un dossier occupe le chemin de sa base : une ouverture ferait échouer
// le job.
func TestStartSessionsRecalc_ProfilAuthOnlyEcarte(t *testing.T) {
	s, store, root := recalcFixture(t,
		`{"version":"3.0","admin":"TokenOnly","profiles":{"halo_5":{"TokenOnly":{"xuid":"2002","auth_only":true}}}}`)
	if err := os.MkdirAll(titlePkg.NewPathResolver(root).PlayerDBPath("halo_5", "TokenOnly"), 0o750); err != nil {
		t.Fatal(err)
	}
	j := runRecalc(t, s, store)
	if j.Status != domain.JobStatusSucceeded {
		t.Fatalf("statut %q (erreur %+v), attendu %q : le profil auth_only ne doit pas être ouvert", j.Status, j.Error, domain.JobStatusSucceeded)
	}
	if j.Result["players"] != 0 || j.Result["players_failed"] != 0 {
		t.Errorf("résultat %v, attendu 0 joueur compté et 0 en échec", j.Result)
	}
}

// Des profils illisibles refusent le recalcul sans créer de job.
func TestStartSessionsRecalc_ProfilsIllisibles(t *testing.T) {
	f := newSyncJobsFixture(t)
	f.profiles.err = errors.New("boom")
	_, err := f.svc.StartSessionsRecalc(context.Background())
	requireSyncJobError(t, err, port.SyncErrPlayersLoad)
	if f.jobs.count() != 0 {
		t.Error("aucun job ne doit être créé")
	}
}
