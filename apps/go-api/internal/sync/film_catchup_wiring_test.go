package sync

// film_catchup_wiring_test.go — les bornes des deux étapes film du post-sync suivent la capacité FilmCatchUp du
// profil de déploiement, par le hook des rejeux que NewHook construit sur la configuration : desktop = garde-fou
// de 200 films (le temps borne la passe) et retard lu sur 2000 matchs ; serveur hébergé (et développement) = 8
// films de frags, 5 rejeux et 64 matchs lus. Un moteur dont le hook n'est pas câblé part des bornes hébergées.

import (
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/sync/replayartifacts"
)

func TestEngine_BornesDesFilmsSuiventLeProfil(t *testing.T) {
	cas := []struct {
		profil  config.Deployment
		env     string
		launch  string
		frags   int
		rejeux  int
		horizon int
	}{
		{config.DeploymentHosted, "production", config.LaunchModeDefault, 8, 5, 64},
		{config.DeploymentDev, "", config.LaunchModeDefault, 8, 5, 64},
		{config.DeploymentDesktop, "", config.LaunchModeDesktop, 200, 200, 2000},
	}
	for _, c := range cas {
		t.Run(string(c.profil), func(t *testing.T) {
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

			e := NewSyncEngineForTitle(t.TempDir(), "halo_infinite", "JGtm", "1111111111111111", nil, nil)
			if got := e.killSource.PerCycle(); got != 8 {
				t.Fatalf("moteur non câblé : frags par cycle = %d, attendu les 8 de l'hébergé", got)
			}
			e.WithReplayArtifacts(replayartifacts.NewHook(cfg, nil, nil))

			if got := e.killSource.PerCycle(); got != c.frags {
				t.Errorf("frags par cycle = %d, attendu %d", got, c.frags)
			}
			if got := e.replayArtifacts.PerCycle(); got != c.rejeux {
				t.Errorf("rejeux par cycle = %d, attendu %d", got, c.rejeux)
			}
			if got := e.replayArtifacts.Horizon(); got != c.horizon {
				t.Errorf("horizon de lecture du retard = %d, attendu %d", got, c.horizon)
			}
		})
	}
}
