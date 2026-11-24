//go:build integration

package replayartifacts

// backlog_budget_integration_test.go — LA LIMITE D UNE PASSE EST SON TEMPS, PAS UN NOMBRE DE MATCHS.
//
// Un match synchronise l'est en totalite, film traite compris : `maxPerCycle` n'est qu'un garde-fou
// (cf. sa declaration). Sur une base reelle :
//
//	budget large   le cycle met en file TOUS les matchs en retard, bien au-dela des cinq d'avant ;
//	budget court   le cycle s'arrete ENTRE deux matchs sur l'horloge, et l'arriere publie pour
//	               l'ordonnanceur (`filmbacklog`) compte ce que la passe n'a pas atteint.
//
// MUTATION QUI DOIT LES FAIRE ROUGIR : remettre `maxPerCycle` a 5 (le premier echoue) ou retirer le
// controle du budget de `enqueueAll` (le second echoue).

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/replaybuild"
	"levelup/go-api/internal/sync/filmbacklog"
)

// retardEnFile : vingt matchs au registre, une mise en file qui met `delai` a repondre et depose
// l'artefact (l'ouvrier simule), et les Deps du cycle.
func retardEnFile(t *testing.T, delai time.Duration) (d Deps, enfiles *[]string) {
	t.Helper()
	db := baseRegistre(t)
	repoRoot := racineAvecTitres(t, titlePkg.DefaultSlug)
	t0 := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 20; i++ {
		inscrireAuRegistre(t, db, fmt.Sprintf("%08d-1111-4000-8000-000000000000", i),
			t0.Add(time.Duration(i)*time.Hour), 0)
	}
	filmbacklog.Reset()
	enfiles = new([]string)
	d = Deps{
		Placement: replaybuild.PlacementWorker,
		RepoRoot:  repoRoot,
		TitleSlug: titlePkg.DefaultSlug,
		WithRead:  func(ctx context.Context, _ string, fn func(*sql.DB)) { fn(db) },
		Enqueue: func(_ context.Context, _, matchID string, _ int) error {
			time.Sleep(delai)
			poserArtefact(t, repoRoot, matchID)
			*enfiles = append(*enfiles, matchID)
			return nil
		},
	}
	return d, enfiles
}

func TestRun_LeBudgetDeTempsEstLaSeuleLimiteDuLot(t *testing.T) {
	d, enfiles := retardEnFile(t, 5*time.Millisecond)
	d.Budget = time.Minute

	Run(context.Background(), d, nil)

	if len(*enfiles) != 20 {
		t.Errorf("matchs mis en file = %d, attendu 20 : la passe doit aller au bout, le garde-fou de %d ne "+
			"doit pas la couper a cinq", len(*enfiles), maxPerCycle)
	}
	if got := filmbacklog.Lire(titlePkg.DefaultSlug).Rejeux; got != 0 {
		t.Errorf("arriere des rejeux = %d, attendu 0 apres une passe qui a tout traite", got)
	}
}

func TestRun_ArretSurLeBudgetDeTempsEtArrierePublie(t *testing.T) {
	d, enfiles := retardEnFile(t, 40*time.Millisecond)
	d.Budget = 100 * time.Millisecond // ~3 mises en file de 40 ms

	Run(context.Background(), d, nil)

	traites := len(*enfiles)
	if traites < 1 || traites >= 20 {
		t.Fatalf("matchs mis en file = %d : la passe devait s'arreter sur le budget de temps, entre 1 et 19", traites)
	}
	if got, want := filmbacklog.Lire(titlePkg.DefaultSlug).Rejeux, 20-traites; got != want {
		t.Errorf("arriere des rejeux = %d, attendu %d (20 - %d mis en file) : l'arriere decrit l'etat "+
			"APRES la passe, budget de temps compris", got, want, traites)
	}

	// Le cycle suivant reprend le solde, et lui seul (les artefacts poses quittent le retard).
	d.Budget = time.Minute
	Run(context.Background(), d, nil)
	if len(*enfiles) != 20 {
		t.Errorf("mis en file au total = %d, attendu 20 (chaque match une seule fois)", len(*enfiles))
	}
	if got := filmbacklog.Lire(titlePkg.DefaultSlug).Rejeux; got != 0 {
		t.Errorf("arriere des rejeux = %d, attendu 0 une fois le solde repris", got)
	}
}
