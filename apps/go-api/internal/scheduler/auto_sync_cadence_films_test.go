// auto_sync_cadence_films_test.go — la cadence de rattrapage tient aussi tant qu'un arriéré de films
// (source des frags, artefacts de rejeu) reste à traiter sur un titre du cycle : historique complet +
// arriéré non vide = 10 minutes ; arriéré vide = intervalle réglé. Les deux chemins de cycle sont joués :
// l'orchestrateur V2 et le filet de boot (syncPlayer). Halo 5 n'a aucune étape film : son titre ne publie
// jamais d'arriéré.
package scheduler

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// fakeFilmBacklog : l'arriéré de films lu par titre ; absent = vide.
type fakeFilmBacklog map[string]port.FilmBacklog

func (f fakeFilmBacklog) FilmBacklog(titleSlug string) port.FilmBacklog { return f[titleSlug] }

func TestCadence_ArriereDeFilmsTientLaCadenceCourte(t *testing.T) {
	cases := []struct {
		name    string
		backlog port.FilmBacklogReader
		want    time.Duration
	}{
		{"arriéré de frags", fakeFilmBacklog{"halo_infinite": {Frags: 395}}, catchUpCycleInterval},
		{"arriéré de rejeux", fakeFilmBacklog{"halo_infinite": {Rejeux: 49}}, catchUpCycleInterval},
		{"les deux", fakeFilmBacklog{"halo_infinite": {Frags: 3, Rejeux: 2}}, catchUpCycleInterval},
		{"arriéré vide", fakeFilmBacklog{"halo_infinite": {}}, 6 * time.Hour},
		{"aucun arriéré publié", fakeFilmBacklog{}, 6 * time.Hour},
		{"arriéré d'un titre absent du cycle", fakeFilmBacklog{"halo_5": {Frags: 10}}, 6 * time.Hour},
		{"sans lecteur câblé", nil, 6 * time.Hour},
	}
	for _, c := range cases {
		t.Run("V2 : "+c.name, func(t *testing.T) {
			s := newCadenceScheduler(t, settings6h, false)
			if c.backlog != nil {
				s.WithFilmBacklogReader(c.backlog)
			}
			// Historique COMPLET : seul l'arriéré de films peut ramener la cadence à 10 minutes.
			s.cycleOrchestrator = &scriptedOrch{outcome: okOutcome(false)}
			tk := newFakeTicker()
			runOneTick(t, s, &tickState{configured: 6 * time.Hour, period: 6 * time.Hour}, tk)
			if got := tk.Resets(); len(got) != 1 || got[0] != c.want {
				t.Errorf("recalages = %v, attendu [%v]", got, c.want)
			}
		})
		t.Run("filet de boot : "+c.name, func(t *testing.T) {
			s := newCadenceScheduler(t, settings6h, false)
			if c.backlog != nil {
				s.WithFilmBacklogReader(c.backlog)
			}
			s.cycleOrchestrator = nil // filet : syncPlayer route tous les joueurs
			s.RunnerFactory = func(context.Context, string, string) DeltaRunner {
				return stubResultRunner{res: domain.SyncResult{}} // historique complet
			}
			tk := newFakeTicker()
			runOneTick(t, s, &tickState{configured: 6 * time.Hour, period: 6 * time.Hour}, tk)
			if got := tk.Resets(); len(got) != 1 || got[0] != c.want {
				t.Errorf("recalages = %v, attendu [%v]", got, c.want)
			}
		})
	}
}

// L'arriéré se somme sur les titres du cycle, chaque titre compté une fois ; le journal du cycle suivant
// en rend le total.
func TestFilmBacklogOf_SommeParTitreSansDoublon(t *testing.T) {
	s := newCadenceScheduler(t, settings6h, true)
	s.WithFilmBacklogReader(fakeFilmBacklog{
		"halo_infinite": {Frags: 3, Rejeux: 2},
		"halo_5":        {Frags: 10},
	})
	players, err := s.cfg.LoadPlayers()
	if err != nil {
		t.Fatalf("LoadPlayers: %v", err)
	}
	players = append(players, players...) // un titre vu deux fois : compté une fois
	got := s.filmBacklogOf(players)
	if got.Frags != 13 || got.Rejeux != 2 || got.Total() != 15 {
		t.Errorf("arriéré = %+v, attendu 13 frags et 2 rejeux", got)
	}
}

// L'écran d'état de l'historique lit le même arriéré, par titre.
func TestFilmsPending_LitLArriereDuTitre(t *testing.T) {
	s := newCadenceScheduler(t, settings6h, false)
	if s.FilmsPending("halo_infinite") {
		t.Error("sans lecteur câblé, aucun arriéré n'est connu")
	}
	s.WithFilmBacklogReader(fakeFilmBacklog{"halo_infinite": {Rejeux: 1}})
	if !s.FilmsPending("halo_infinite") || s.FilmsPending("halo_5") {
		t.Error("FilmsPending doit suivre l'arriéré du titre demandé, et de lui seul")
	}
}
