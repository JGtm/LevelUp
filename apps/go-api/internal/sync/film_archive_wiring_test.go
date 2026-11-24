package sync

// film_archive_wiring_test.go — la source des frags suit le réglage « archiver les films » par le hook des
// rejeux, dont le défaut vient de la capacité ArchiveFilmsByDefault du profil : le site hébergé n'écrit rien à
// l'archive tant que l'administrateur n'a rien écrit, le bureau et le développement archivent, une valeur
// écrite gagne dans les deux sens, et archive coupée la source des frags n'écrit rien, quel que soit le profil.

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/sync/replayartifacts"
)

func TestEngine_ArchivageDesFragsSuitLeReglageEtLeProfil(t *testing.T) {
	cas := []struct {
		nom      string
		profil   config.Deployment
		env      string
		launch   string
		fichier  string // contenu d'app_settings.json ; vide = fichier absent
		archiver bool
	}{
		{"hébergé, rien d'écrit : pas d'archive", config.DeploymentHosted, "production", config.LaunchModeDefault, "", false},
		{"hébergé, archive écrite : elle gagne", config.DeploymentHosted, "production", config.LaunchModeDefault, `{"archive_films": true}`, true},
		{"développement, rien d'écrit : archive", config.DeploymentDev, "", config.LaunchModeDefault, "", true},
		{"bureau, rien d'écrit : archive", config.DeploymentDesktop, "", config.LaunchModeDesktop, "", true},
		// Le client Halo du serveur n'a pas de cache de film : la cuisson re-télécharge de toute façon, la source des
		// frags n'a donc aucune raison d'écrire quand le réglage est coupé, dans aucun profil.
		{"bureau, archive coupée : la source des frags n'écrit rien", config.DeploymentDesktop, "", config.LaunchModeDesktop, `{"archive_films": false}`, false},
		{"hébergé, archive coupée : pas d'archive", config.DeploymentHosted, "production", config.LaunchModeDefault, `{"archive_films": false}`, false},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			t.Setenv("LEVELUP_REPO_ROOT", t.TempDir())
			t.Setenv("LEVELUP_ENV", c.env)
			t.Setenv("LEVELUP_DEMO_MODE", "")
			t.Setenv("LEVELUP_AUTH_MODE", "")
			t.Setenv("LEVELUP_API_HOST", "")
			cfg, err := config.LoadServer(c.launch)
			if err != nil {
				t.Fatalf("LoadServer : %v", err)
			}
			if cfg.Deployment != c.profil {
				t.Fatalf("profil = %q, attendu %q", cfg.Deployment, c.profil)
			}
			chemin := filepath.Join(t.TempDir(), "app_settings.json")
			if c.fichier != "" {
				if err := os.WriteFile(chemin, []byte(c.fichier), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			store := settings.NewStore(chemin).WithArchiveFilmsDefault(cfg.Capabilities.ArchiveFilmsByDefault)

			e := NewSyncEngineForTitle(t.TempDir(), "halo_infinite", "JGtm", "1111111111111111", nil, nil)
			e.WithReplayArtifacts(replayartifacts.NewHook(cfg, store, nil))

			if got := e.killSource.ArchiveFilms(t.Context()); got != c.archiver {
				t.Errorf("la source des frags archive = %v, attendu %v", got, c.archiver)
			}
		})
	}
}
