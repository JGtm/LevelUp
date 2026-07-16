//go:build cgo

// Package prestige — prestige_player_reopen_test.go : régression course de lecture
// sur pdb.Player pendant un Reopen concurrent (B-swap / recovery ART).
//
// Contexte (fix/prestige-player-db-closed-race) : les repos Prestige joueur
// partagent l'unique handle RW pdb.Player. Un writer concurrent qui déclenche
// db.Reopen() (invalidation ART, ou fermeture transitoire pendant un B-swap)
// ferme d'abord le *sql.DB sous-jacent (old.Close() dans Reopen). Une lecture
// Prestige en vol voyait alors "sql: database is closed" et — faute de passer par
// les variantes *Recovered — remontait l'erreur sans retry (bruit prod ~50/j +
// données Prestige transitoirement absentes ; op=OpenReadWrite, query=challenge).
//
// Ce test simule la fermeture via db.Close() (même simulacre que
// privacy_state_repo_test.go, dont l'UpsertPrivacyState survit déjà) AVANT chaque
// appel, et vérifie que List / Get / ListByUser / UpdateStatus récupèrent
// (Reopen + retry) au lieu d'échouer. Fichier réel (t.TempDir), car Reopen doit
// pouvoir rouvrir le DSN — :memory: ne survit pas à un close.
//
// Oracle : sur la handle fermée, chaque méthode doit réussir et rendre la ligne
// seedée. Le code pré-fix (Query/Exec/QueryRow plats) remonte "database is closed"
// → chaque appel FAIL. Le code post-fix (*Recovered) PASS.
package prestige

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/prestige"
)

// newPrestigePlayerReopenDB crée un stats.duckdb temporaire (fichier réel) avec le
// schéma player complet (migrations réelles) et retourne la handle RW ouverte via
// le wrapper duckdb (cache + pool), seule apte au Reopen auto-réparant.
func newPrestigePlayerReopenDB(t *testing.T) *duckdb.DB {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.duckdb")

	raw, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if err := migration.RunForDB(raw, migration.TargetPlayer); err != nil {
		raw.Close()
		t.Fatalf("RunForDB(player): %v", err)
	}
	raw.Close()

	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// seedRaceArcAndChallenge insère un arc + un défi actif sur la handle ouverte.
func seedRaceArcAndChallenge(t *testing.T, db *duckdb.DB, arcID, challengeID string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	if err := NewPrestigeArcRepo(db).Create(ctx, prestige.Arc{
		ID: arcID, UserID: "u1", TitleSlug: "halo_infinite", Title: "T", CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed arc Create: %v", err)
	}
	if err := NewPrestigeChallengeRepo(db).Create(ctx, prestige.Challenge{
		ID: challengeID, UserID: "u1", TitleSlug: "halo_infinite", ArcID: arcID,
		Metric: "FieldKDA", Target: 1.0,
		WindowType: prestige.WindowSession, Cadence: prestige.CadenceFree,
		EvalType: prestige.EvalThreshold, Mode: prestige.ModeLibre,
		DataTier: prestige.DataFull, Status: prestige.StatusActive, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed challenge Create: %v", err)
	}
}

// TestPrestigePlayerRepos_RecoverAfterHandleClose ferme la handle player AVANT
// chaque appel (simulacre du old.Close() transitoire d'un Reopen concurrent) puis
// vérifie que la lecture multi-lignes (List), la lecture mono-ligne (Get), la
// lecture d'arcs (ListByUser) et un write (UpdateStatus) récupèrent. Chaque close
// est ré-appliqué pour prouver la recovery de CHAQUE variante (QueryRecovered,
// QueryRowRecovered, ExecRecovered) indépendamment — pré-fix, chacune remonterait
// "sql: database is closed".
func TestPrestigePlayerRepos_RecoverAfterHandleClose(t *testing.T) {
	db := newPrestigePlayerReopenDB(t)
	seedRaceArcAndChallenge(t, db, "arc_race", "ch_race")

	chRepo := NewPrestigeChallengeRepo(db)
	arcRepo := NewPrestigeArcRepo(db)
	ctx := context.Background()
	active := prestige.StatusActive

	// 1. List (QueryRecovered) après fermeture.
	closeHandle(t, db, "List")
	list, err := chRepo.List(ctx, prestige.ChallengeFilter{
		UserID: "u1", TitleSlug: "halo_infinite", Status: &active,
	})
	if err != nil {
		t.Fatalf("List après close: attendu recovery, got %v", err)
	}
	if len(list) != 1 || list[0].ID != "ch_race" {
		t.Fatalf("List: got %d lignes, want 1 (ch_race)", len(list))
	}

	// 2. Get (QueryRowRecovered) après fermeture.
	closeHandle(t, db, "Get")
	got, err := chRepo.Get(ctx, "ch_race")
	if err != nil {
		t.Fatalf("Get après close: attendu recovery, got %v", err)
	}
	if got.ID != "ch_race" {
		t.Fatalf("Get: got %q want ch_race", got.ID)
	}

	// 3. ListByUser (QueryRecovered, table arc) après fermeture.
	closeHandle(t, db, "ListByUser")
	arcs, err := arcRepo.ListByUser(ctx, "u1", "halo_infinite")
	if err != nil {
		t.Fatalf("ListByUser après close: attendu recovery, got %v", err)
	}
	if len(arcs) != 1 || arcs[0].ID != "arc_race" {
		t.Fatalf("ListByUser: got %d arcs, want 1 (arc_race)", len(arcs))
	}

	// 4. UpdateStatus (ExecRecovered, write) après fermeture.
	closeHandle(t, db, "UpdateStatus")
	completedAt := time.Now().UTC().Truncate(time.Second)
	if err := chRepo.UpdateStatus(ctx, "ch_race", prestige.StatusCompleted, completedAt); err != nil {
		t.Fatalf("UpdateStatus après close: attendu recovery, got %v", err)
	}
	after, err := chRepo.Get(ctx, "ch_race")
	if err != nil {
		t.Fatalf("Get post-update: %v", err)
	}
	if after.Status != prestige.StatusCompleted {
		t.Fatalf("UpdateStatus non persisté après recovery: status=%s want completed", after.Status)
	}

	// 5. Not-found : la recovery ne doit pas masquer sql.ErrNoRows en succès —
	//    le mapping domaine ErrChallengeNotFound doit survivre au Reopen.
	closeHandle(t, db, "Get(absent)")
	if _, err := chRepo.Get(ctx, "ch_absent"); !errors.Is(err, prestige.ErrChallengeNotFound) {
		t.Fatalf("Get(absent) après close: want ErrChallengeNotFound, got %v", err)
	}
}

// closeHandle ferme la handle partagée pour simuler le old.Close() transitoire du
// Reopen concurrent. L'appel *Recovered suivant doit rouvrir le DSN et rejouer.
func closeHandle(t *testing.T, db *duckdb.DB, next string) {
	t.Helper()
	if err := db.Close(); err != nil {
		t.Fatalf("simulate handle close before %s: %v", next, err)
	}
}
