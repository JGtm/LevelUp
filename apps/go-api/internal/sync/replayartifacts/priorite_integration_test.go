//go:build integration

package replayartifacts

// priorite_integration_test.go — LA PRIORITÉ POSÉE PAR LA SÉLECTION, sur une vraie base migrée :
// le match inséré du cycle part devant le match récent du rattrapage, qui part devant
// l'historique (domain.BuildPriority*).

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/replaybuild"
)

func inscrireSynchronise(t *testing.T, db *sql.DB, id string, debut, synchro time.Time) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO match_registry (match_id, start_time, start_time_utc, first_sync_at, map_name)
		 VALUES (?, ?, ?, ?, 'Recharge')`, id, debut, debut, synchro)
	if err != nil {
		t.Fatalf("insert registre %s: %v", id, err)
	}
}

func TestRun_PrioriteInsereRecentRattrapage(t *testing.T) {
	db := baseRegistre(t)
	repoRoot := racineAvecTitres(t, titlePkg.DefaultSlug)
	now := time.Now().UTC()
	const (
		insere = "11111111-1111-4000-8000-000000000000"
		recent = "22222222-2222-4000-8000-000000000000"
		ancien = "33333333-3333-4000-8000-000000000000"
	)
	inscrireSynchronise(t, db, insere, now.Add(-15*time.Minute), now.Add(-time.Minute))
	inscrireSynchronise(t, db, recent, now.Add(-2*time.Hour), now.Add(-90*time.Minute))
	inscrireSynchronise(t, db, ancien, now.Add(-72*time.Hour), now.Add(-71*time.Hour))

	priorites := map[string]int{}
	Run(context.Background(), Deps{
		Placement: replaybuild.PlacementWorker,
		RepoRoot:  repoRoot,
		TitleSlug: titlePkg.DefaultSlug,
		WithRead:  func(ctx context.Context, _ string, fn func(*sql.DB)) { fn(db) },
		Enqueue: func(_ context.Context, _, matchID string, priority int) error {
			priorites[matchID] = priority
			return nil
		},
	}, []string{insere})

	attendu := map[string]int{
		insere: domain.BuildPriorityInsere,
		recent: domain.BuildPriorityRecent,
		ancien: domain.BuildPriorityRattrapage,
	}
	for id, p := range attendu {
		if got, ok := priorites[id]; !ok || got != p {
			t.Errorf("priorité de %s = %d (mis en file : %v), attendu %d", id, got, ok, p)
		}
	}
}

// TestSelection_LocalSansRattrapageQuandInseres : en construction locale, un cycle qui a inséré
// un match ne cuit pas l'historique à sa suite — et la jauge de retard est quand même mesurée.
func TestSelection_LocalSansRattrapageQuandInseres(t *testing.T) {
	db := baseRegistre(t)
	now := time.Now().UTC()
	const insere = "44444444-4444-4000-8000-000000000000"
	inscrireSynchronise(t, db, insere, now.Add(-15*time.Minute), now.Add(-time.Minute))
	inscrireSynchronise(t, db, "55555555-5555-4000-8000-000000000000", now.Add(-72*time.Hour), now.Add(-71*time.Hour))
	d := Deps{
		Placement: replaybuild.PlacementLocal,
		RepoRoot:  t.TempDir(),
		TitleSlug: titlePkg.DefaultSlug,
		WithRead:  func(ctx context.Context, _ string, fn func(*sql.DB)) { fn(db) },
	}

	work, retard := selectionnerLeTravail(context.Background(), d, []string{insere})
	if got := ids(work); len(got) != 1 || got[0] != insere {
		t.Fatalf("travail = %v, attendu le seul match inséré", got)
	}
	if retard != 1 {
		t.Fatalf("retard = %d, attendu 1 : la jauge se mesure même sans place", retard)
	}

	work, _ = selectionnerLeTravail(context.Background(), d, nil)
	// Les deux matchs sont sans rejeu : sans insertion, le rattrapage les reprend tous les deux.
	if len(work) != 2 {
		t.Fatalf("sans insertion, le rattrapage reprend : travail = %v", ids(work))
	}
}
