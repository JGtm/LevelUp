//go:build integration

// catchup_integration_test.go : l'état du rattrapage de l'historique Halo 5 sur une vraie base joueur
// temporaire (DuckDB, CGO), seul et branché sur le runner (trois passes jusqu'à la fin).
package livesync

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/historystate"
)

func tempPlayerHistory(t *testing.T) (playerHistory, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "player", "stats.duckdb")
	return newPlayerHistory(path, "JGtm"), path
}

func TestPlayerHistory_BasePoueurTemporaire(t *testing.T) {
	h, _ := tempPlayerHistory(t)
	ctx := context.Background()

	if complete, err := h.Complete(ctx); err != nil || complete {
		t.Fatalf("base neuve : complet=%v err=%v, attendu faux sans erreur", complete, err)
	}
	if _, ok := h.LoadCursor(ctx); ok {
		t.Fatal("base neuve : aucun marque-page attendu")
	}
	cur := historystate.Cursor{Offset: 200, LastMatchID: "m-200"}
	if !h.SaveCursor(ctx, cur) {
		t.Fatal("le marque-page doit se ranger quand le bail est libre")
	}
	if got, ok := h.LoadCursor(ctx); !ok || got != cur {
		t.Fatalf("marque-page relu = (%+v, %v), attendu %+v", got, ok, cur)
	}
	st := historystate.Streak{Subject: "m-200", Attempts: 2, Skipped: 1, LastError: "500"}
	if !h.SaveStreak(ctx, st) {
		t.Fatal("le compte d'échecs doit se ranger")
	}
	if got := h.LoadStreak(ctx, "m-200"); got != st {
		t.Fatalf("compte relu = %+v, attendu %+v", got, st)
	}

	// La fin rangée prime : le marque-page et le compte d'échecs sont effacés.
	if !h.MarkComplete(ctx) {
		t.Fatal("la fin d'historique doit se ranger quand le bail est libre")
	}
	if complete, err := h.Complete(ctx); err != nil || !complete {
		t.Fatalf("après rangement : complet=%v err=%v, attendu vrai", complete, err)
	}
	if _, ok := h.LoadCursor(ctx); ok {
		t.Error("la fin rangée doit effacer le marque-page")
	}
	if got := h.LoadStreak(ctx, "m-200"); got.Attempts != 0 {
		t.Errorf("la fin rangée doit effacer le compte d'échecs : %+v", got)
	}
}

// Les écritures passent par le bail d'écriture du joueur : bail tenu ailleurs = non rangé, sans rester
// bloqué au-delà du délai, et la base reste « incomplète » pour la passe suivante.
func TestPlayerHistory_BailTenuNeRangePas(t *testing.T) {
	h, path := tempPlayerHistory(t)
	db, err := syncpkg.OpenPlayerDB(path)
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
	if h.MarkComplete(short) || h.SaveCursor(short, historystate.Cursor{Offset: 5, LastMatchID: "m"}) {
		t.Fatal("rangement réussi alors que le bail du joueur est tenu")
	}
	if waited := time.Since(start); waited > 2*time.Second {
		t.Fatalf("rangement resté bloqué %v au-delà de son délai", waited)
	}
	held.Release()

	if complete, err := h.Complete(context.Background()); err != nil || complete {
		t.Fatalf("complet=%v err=%v, attendu faux (rien rangé)", complete, err)
	}
	if !h.MarkComplete(context.Background()) {
		t.Fatal("bail libéré : la fin doit se ranger")
	}
}

// Runner branché sur le vrai magasin : trois passes rangent la fin dans la base joueur, la quatrième
// est un delta ordinaire ; le marque-page est bien lu et rangé entre les passes.
func TestRunDelta_RattrapageSurBaseJoueurTemporaire(t *testing.T) {
	h, _ := tempPlayerHistory(t)
	w := newHistoryWorld(450)
	deps := w.deps()
	deps.History = h
	r := NewRunner(deps, nil)
	ctx := context.Background()

	for i, want := range []struct {
		inserted int
		pending  bool
	}{{200, true}, {200, true}, {50, false}} {
		res, err := r.RunDelta(ctx, domain.SyncOptions{})
		if err != nil || res.MatchesInserted != want.inserted || res.HistoryPending != want.pending {
			t.Fatalf("passe %d : err=%v insérés=%d pending=%v, attendu %d/%v", i+1, err, res.MatchesInserted, res.HistoryPending, want.inserted, want.pending)
		}
		if i == 0 {
			if cur, ok := h.LoadCursor(ctx); !ok || cur.Offset != 200 || cur.LastMatchID != "m0199" {
				t.Fatalf("marque-page après la première passe = (%+v, %v), attendu {200 m0199}", cur, ok)
			}
		}
	}
	if complete, err := h.Complete(ctx); err != nil || !complete {
		t.Fatalf("la clé doit être rangée dans la base joueur : complet=%v err=%v", complete, err)
	}
	res, err := r.RunDelta(ctx, domain.SyncOptions{MaxMatches: 25})
	if err != nil || res.HistoryPending {
		t.Fatalf("passe 4 : err=%v pending=%v, attendu un delta sans attente", err, res.HistoryPending)
	}
}

// Page en échec durable sur la vraie base : le compte d'échecs survit entre les passes, la page est
// abandonnée à la troisième, la fin est rangée.
func TestRunDelta_PageAbandonneeSurBaseJoueurTemporaire(t *testing.T) {
	h, _ := tempPlayerHistory(t)
	w := newHistoryWorld(100)
	w.errAtStart[50] = errors.New("GetPlayerMatches: 500")
	deps := w.deps()
	deps.History = h
	r := NewRunner(deps, nil)
	ctx := context.Background()

	for pass := 1; pass <= 2; pass++ {
		res, err := r.RunDelta(ctx, domain.SyncOptions{})
		if err != nil || !res.HistoryPending {
			t.Fatalf("passe %d : err=%v pending=%v, attendu une passe en échec", pass, err, res.HistoryPending)
		}
		if got := h.LoadStreak(ctx, "m0049"); got.Attempts != pass {
			t.Fatalf("passe %d : compte d'échecs %+v, attendu %d", pass, got, pass)
		}
	}
	res, err := r.RunDelta(ctx, domain.SyncOptions{})
	if err != nil || res.HistoryPending {
		t.Fatalf("troisième passe : err=%v pending=%v, attendu la fin rangée", err, res.HistoryPending)
	}
	if complete, err := h.Complete(ctx); err != nil || !complete {
		t.Fatalf("complet=%v err=%v, attendu vrai", complete, err)
	}
}
