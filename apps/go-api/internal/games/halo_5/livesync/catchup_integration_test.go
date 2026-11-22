//go:build integration

// catchup_integration_test.go : les hooks de l'état de l'historique Halo 5 sur une vraie base joueur
// temporaire (DuckDB, CGO), seuls et branchés sur le runner (trois passes jusqu'à la fin).
package livesync

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
	syncpkg "levelup/go-api/internal/sync"
)

func TestHistoryHooks_BasePoueurTemporaire(t *testing.T) {
	playerPath := filepath.Join(t.TempDir(), "player", "stats.duckdb")
	ctx := context.Background()

	complete, err := playerHistoryComplete(ctx, playerPath)
	if err != nil || complete {
		t.Fatalf("base neuve : complet=%v err=%v, attendu faux sans erreur", complete, err)
	}
	if !markPlayerHistoryComplete(ctx, playerPath, "JGtm") {
		t.Fatal("la fin d'historique doit se ranger quand le bail est libre")
	}
	if complete, err = playerHistoryComplete(ctx, playerPath); err != nil || !complete {
		t.Fatalf("après rangement : complet=%v err=%v, attendu vrai", complete, err)
	}
}

// Le rangement de la fin passe par le bail d'écriture du joueur : bail tenu ailleurs = non rangé,
// sans rester bloqué au-delà du délai, et la base reste « incomplète » pour la passe suivante.
func TestHistoryHooks_BailTenuNeRangePas(t *testing.T) {
	playerPath := filepath.Join(t.TempDir(), "player", "stats.duckdb")
	db, err := syncpkg.OpenPlayerDB(playerPath)
	if err != nil {
		t.Fatalf("OpenPlayerDB : %v", err)
	}
	defer db.Close()
	held, err := dblease.AcquireWriter(db.SQLDb(), db.Path(), dblease.KindPlayer, dblease.PlayerLeaseTimeout)
	if err != nil {
		t.Fatalf("prise du bail : %v", err)
	}

	short, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	if markPlayerHistoryComplete(short, playerPath, "JGtm") {
		t.Fatal("fin rangée alors que le bail du joueur est tenu")
	}
	if waited := time.Since(start); waited > 2*time.Second {
		t.Fatalf("rangement resté bloqué %v au-delà de son délai", waited)
	}
	held.Release()

	if complete, err := playerHistoryComplete(context.Background(), playerPath); err != nil || complete {
		t.Fatalf("complet=%v err=%v, attendu faux (rien rangé)", complete, err)
	}
	if !markPlayerHistoryComplete(context.Background(), playerPath, "JGtm") {
		t.Fatal("bail libéré : la fin doit se ranger")
	}
}

// Runner branché sur les vrais hooks : trois passes rangent la fin dans la base joueur, la quatrième
// est un delta ordinaire.
func TestRunDelta_RattrapageSurBaseJoueurTemporaire(t *testing.T) {
	playerPath := filepath.Join(t.TempDir(), "player", "stats.duckdb")
	w := newHistoryWorld(450)
	deps := w.deps()
	deps.HistoryComplete = func(ctx context.Context) (bool, error) { return playerHistoryComplete(ctx, playerPath) }
	deps.MarkHistoryComplete = func(ctx context.Context) bool { return markPlayerHistoryComplete(ctx, playerPath, "JGtm") }
	r := NewRunner(deps, nil)

	for i, want := range []struct {
		inserted int
		pending  bool
	}{{200, true}, {200, true}, {50, false}} {
		res, err := r.RunDelta(context.Background(), domain.SyncOptions{})
		if err != nil || res.MatchesInserted != want.inserted || res.HistoryPending != want.pending {
			t.Fatalf("passe %d : err=%v insérés=%d pending=%v, attendu %d/%v", i+1, err, res.MatchesInserted, res.HistoryPending, want.inserted, want.pending)
		}
	}
	if complete, err := playerHistoryComplete(context.Background(), playerPath); err != nil || !complete {
		t.Fatalf("la clé doit être rangée dans la base joueur : complet=%v err=%v", complete, err)
	}
	res, err := r.RunDelta(context.Background(), domain.SyncOptions{MaxMatches: 25})
	if err != nil || w.deltaCalls != 1 || res.HistoryPending {
		t.Fatalf("passe 4 : err=%v delta=%d pending=%v, attendu un delta sans attente", err, w.deltaCalls, res.HistoryPending)
	}
}
