//go:build integration

package killcollector

// postsync_budget_integration_test.go — LA LIMITE D UNE PASSE EST SON TEMPS, PAS UN NOMBRE DE FILMS.
//
// Un match synchronise l'est en totalite, film traite compris : la borne en nombre n'est qu'un
// garde-fou ([GardeFouFilmsParCycle]). Ces tests le verrouillent sur une base reelle :
//
//	budget large   la passe traite TOUS les matchs, bien au-dela des huit d'avant la borne relevee ;
//	budget court   la passe s'arrete ENTRE deux matchs sur l'horloge, avec du travail restant, et
//	               l'arriere publie pour l'ordonnanceur dit ce qui reste.
//
// MUTATION QUI DOIT LES FAIRE ROUGIR : remettre `perCycle` a 8 (le premier echoue) ou retirer
// `WithBudget(h.budget)` du collecteur (le second echoue : tout est traite malgre le budget).

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/sync/filmbacklog"
	"levelup/go-api/internal/sync/filmfetch"
)

// filmsLents : une source qui met `delai` a repondre « film absent » et compte ses appels.
type filmsLents struct {
	delai  time.Duration
	appels atomic.Int64
}

func (f *filmsLents) GetFilmChunks(ctx context.Context, _ string) ([]filmfetch.FilmChunk, bool, error) {
	f.appels.Add(1)
	select {
	case <-time.After(f.delai):
	case <-ctx.Done():
	}
	return nil, false, nil
}

func backlogDeVingt(t *testing.T) (deps PostSyncDeps, source *filmsLents) {
	t.Helper()
	db := baseBacklog(t)
	t0 := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 20; i++ {
		inscrireMatch(t, db, "m"+string(rune('a'+i)), t0.AddDate(0, 0, i), 0)
	}
	filmbacklog.Reset()
	source = &filmsLents{delai: 40 * time.Millisecond}
	deps = depsDeTest(db, &filmsTraces{}, nil)
	deps.Fetcher = source
	return deps, source
}

func TestRunPostSync_LeBudgetDeTempsEstLaSeuleLimite(t *testing.T) {
	deps, source := backlogDeVingt(t)
	h := NewPostSyncHook(racineDepot(t), 0)
	if h.perCycle != GardeFouFilmsParCycle || GardeFouFilmsParCycle != DefaultPostSyncPerCycle {
		t.Fatalf("perCycle = %d, attendu le garde-fou %d", h.perCycle, GardeFouFilmsParCycle)
	}
	h.budget = time.Minute // large : jamais atteint par vingt films de 40 ms

	RunPostSync(context.Background(), h, deps, nil)

	if got := source.appels.Load(); got != 20 {
		t.Errorf("films demandes = %d, attendu 20 : la passe doit aller au bout de son travail, la "+
			"borne en nombre ne doit pas la couper a huit", got)
	}
	if got := filmbacklog.Lire("halo_infinite").Frags; got != 0 {
		t.Errorf("arriere des frags = %d, attendu 0 apres une passe qui a tout traite", got)
	}
}

func TestRunPostSync_ArretSurLeBudgetDeTempsEtArrierePublie(t *testing.T) {
	deps, source := backlogDeVingt(t)
	h := NewPostSyncHook(racineDepot(t), 0)
	h.budget = 100 * time.Millisecond // ~2-3 films de 40 ms

	RunPostSync(context.Background(), h, deps, nil)

	traites := int(source.appels.Load())
	if traites < 1 || traites >= 20 {
		t.Fatalf("films demandes = %d : la passe devait s'arreter sur le budget de temps, entre 1 et 19", traites)
	}
	if got, want := filmbacklog.Lire("halo_infinite").Frags, 20-traites; got != want {
		t.Errorf("arriere des frags = %d, attendu %d (20 - %d traites) : l'arriere decrit l'etat APRES la passe",
			got, want, traites)
	}

	// La passe suivante, sans budget serre, reprend le solde — et lui seul (les traites ont quitte
	// le backlog) : l'arriere revient a zero.
	h.budget = time.Minute
	RunPostSync(context.Background(), h, deps, nil)
	if got := source.appels.Load(); got != 20 {
		t.Errorf("films demandes au total = %d, attendu 20 (chaque film une seule fois)", got)
	}
	if got := filmbacklog.Lire("halo_infinite").Frags; got != 0 {
		t.Errorf("arriere des frags = %d, attendu 0 une fois le solde repris", got)
	}
}
