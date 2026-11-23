//go:build integration

package replayartifacts

// backlog_heberge_integration_test.go — HORS de la capacité FilmCatchUp (serveur hébergé), les bornes d'avant :
// au plus RejeuxParCycleHeberge rejeux par cycle, retard lu sur BacklogHorizonHeberge matchs. Les Deps portent
// ces valeurs comme le câblage de production (Hook.PerCycle, Hook.Horizon).

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

func TestRun_HorsCapaciteLePlafondEstLeNombreDAvant(t *testing.T) {
	d, enfiles := retardEnFile(t, 0) // 20 matchs au registre, placement ouvrier
	d.Budget = time.Minute
	hote := &Hook{}
	d.PerCycle, d.Horizon = hote.PerCycle(), hote.Horizon()

	Run(context.Background(), d, nil)

	if len(*enfiles) != RejeuxParCycleHeberge {
		t.Errorf("matchs mis en file = %d, attendu %d : hors capacité, le plafond du cycle est de cinq rejeux",
			len(*enfiles), RejeuxParCycleHeberge)
	}
	if got, want := filmbacklog.Lire(titlePkg.DefaultSlug).Rejeux, 20-RejeuxParCycleHeberge; got != want {
		t.Errorf("arriéré des rejeux = %d, attendu %d", got, want)
	}
}

func TestRun_HorsCapaciteLeRetardSeLitSurLHorizonDAvant(t *testing.T) {
	db := baseRegistre(t)
	repoRoot := racineAvecTitres(t, titlePkg.DefaultSlug)
	t0 := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	total := BacklogHorizonHeberge + 10
	for i := 0; i < total; i++ {
		inscrireAuRegistre(t, db, fmt.Sprintf("%08d-1111-4000-8000-000000000000", i),
			t0.Add(time.Duration(i)*time.Hour), 0)
	}
	hote := &Hook{}
	// Plafond large : seul l'horizon de lecture limite ce que le cycle voit.
	d := Deps{
		Placement: replaybuild.PlacementWorker,
		RepoRoot:  repoRoot,
		TitleSlug: titlePkg.DefaultSlug,
		WithRead:  func(ctx context.Context, _ string, fn func(*sql.DB)) { fn(db) },
		PerCycle:  maxPerCycle,
		Horizon:   hote.Horizon(),
		Budget:    time.Minute,
	}
	var enfiles []string
	d.Enqueue = func(_ context.Context, _, matchID string, _ int) error {
		poserArtefact(t, repoRoot, matchID)
		enfiles = append(enfiles, matchID)
		return nil
	}

	Run(context.Background(), d, nil)

	if len(enfiles) != BacklogHorizonHeberge {
		t.Errorf("matchs vus par le cycle = %d, attendu %d : hors capacité, la lecture du retard s'arrête à l'horizon",
			len(enfiles), BacklogHorizonHeberge)
	}

	// Sous la capacité, le même registre est lu en entier.
	d.Horizon = (&Hook{FilmCatchUp: true}).Horizon()
	Run(context.Background(), d, nil)
	if len(enfiles) != total {
		t.Errorf("matchs vus au total = %d, attendu %d : sous la capacité, l'horizon couvre tout le registre", len(enfiles), total)
	}
}
