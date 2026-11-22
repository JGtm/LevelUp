// auto_sync_cadence_films_profil_test.go — la cadence liée à l'arriéré de films dépend de la capacité
// FilmCatchUp du profil de déploiement : sur un serveur hébergé, avec un arriéré de films non vide et un
// historique complet, le cycle suivant reste à l'intervalle réglé ; sur le bureau il revient dans 10 minutes.
// La chaîne est jouée de bout en bout : profil résolu par config.LoadServer, capacité, lecteur choisi comme au
// démarrage (filmbacklog.ReaderFor), cadence du cycle.
package scheduler

import (
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/sync/filmbacklog"
)

func TestCadence_ArriereDeFilmsSuitLaCapaciteDuProfil(t *testing.T) {
	cas := []struct {
		profil config.Deployment
		env    string
		launch string
		want   time.Duration
	}{
		{config.DeploymentHosted, "production", config.LaunchModeDefault, 6 * time.Hour},
		{config.DeploymentDev, "", config.LaunchModeDefault, 6 * time.Hour},
		{config.DeploymentDesktop, "", config.LaunchModeDesktop, catchUpCycleInterval},
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
			// Un arriéré de films publié par les étapes film (elles publient sous tous les profils).
			filmbacklog.Reset()
			t.Cleanup(filmbacklog.Reset)
			filmbacklog.Publier("halo_infinite", filmbacklog.Frags, 395)
			filmbacklog.Publier("halo_infinite", filmbacklog.Rejeux, 49)

			s := newCadenceScheduler(t, settings6h, false)
			s.WithFilmBacklogReader(filmbacklog.ReaderFor(cfg.Capabilities.FilmCatchUp))
			// Historique COMPLET : seul l'arriéré de films peut ramener la cadence à 10 minutes.
			s.cycleOrchestrator = &scriptedOrch{outcome: okOutcome(false)}
			tk := newFakeTicker()
			runOneTick(t, s, &tickState{configured: 6 * time.Hour, period: 6 * time.Hour}, tk)
			if got := tk.Resets(); len(got) != 1 || got[0] != c.want {
				t.Errorf("recalages = %v, attendu [%v]", got, c.want)
			}
			if got, want := s.FilmsPending("halo_infinite"), cfg.Capabilities.FilmCatchUp; got != want {
				t.Errorf("FilmsPending = %v, attendu %v : l'écran d'état suit la même capacité", got, want)
			}
		})
	}
}
