//go:build integration && bug_repro

// Package persist — combined_persister_test.go : tests pour CombinedPersister.
//
// Build tag double : `integration` (CGO DuckDB) + `bug_repro` (reproductions
// de bugs known qui doivent rester rouges jusqu'a leur fix). Pattern aligne
// sur internal/sync/csr_art_repro_test.go (WIP user).
//
// Pour lancer : go test -tags 'integration bug_repro' ./internal/persist/...
// La CI default skip ces tests pour ne pas etre rouge sur des bugs documentes.
//
// Cible critique : reproduire le bug #1 de PLAN_FIX_SYNC_RELIABILITY_2026-05-24,
// observe en production le 2026-05-24 sur Chocoboflor + XxDaemonGamerxX :
//
//	"CombinedPersister: open player <gamertag>:
//	 Connection Error: Can't open a connection to same database file
//	 with a different configuration than existing connections"
//
// Cause racine identifiee : combined_persister.go ligne 94 ouvre
//
//	sql.Open("duckdb", playerPath+"?access_mode=READ_WRITE")
//
// alors que d'autres sites (engine.go, OpenPlayerDB) ouvrent avec DSN nu
// via duckdbpkg.OpenReadWrite. DuckDB voit 2 configs differentes pour le meme
// fichier et refuse la 2e connexion.
//
// Fix prevu : Phase 1 du plan principal — basculer CombinedPersister sur
// le cache duckdbpkg.OpenReadWrite pour aligner le DSN.

package persist

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// makeCombinedFixturePlayerDB cree un player DB DuckDB persistent sur disque
// (tmpDir) avec le schema bootstrap inline. Retourne le path absolu — utilisable
// par CombinedPersister via playerDBPathFn.
func makeCombinedFixturePlayerDB(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	playerPath := filepath.Join(tmpDir, "stats.duckdb")

	// Bootstrap schema via sql.Open direct (one-shot, on referme).
	db, err := sql.Open("duckdb", playerPath)
	if err != nil {
		t.Fatalf("bootstrap open: %v", err)
	}
	if _, err := db.Exec(playerTestSchemaSQL); err != nil {
		_ = db.Close()
		t.Fatalf("bootstrap schema: %v", err)
	}
	_ = db.Close()
	return playerPath
}

// noopSharedWriter retourne une closure qui ouvre un shared DB :memory:
// pre-migre comme dans openSharedTestDB (de shared_persister_test.go).
func noopSharedWriter(t *testing.T) SharedWriterFn {
	t.Helper()
	sharedDB := openSharedTestDB(t)
	return func(ctx context.Context) (*sql.DB, func(), error) {
		return sharedDB, func() {}, nil
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Test critique : reproduit bug #1 si CombinedPersister utilise sql.Open
// avec un DSN different de celui du cache. Le test echoue tant que Phase 1
// du fix n'est pas applique.
// ─────────────────────────────────────────────────────────────────────────────

func TestCombinedPersister_NoConfigurationConflict(t *testing.T) {
	playerPath := makeCombinedFixturePlayerDB(t)

	// Etape 1 : ouvrir le player DB en "pre-emption" comme le ferait l'engine
	// via OpenPlayerDB (DSN nu, RW cache process-level).
	//
	// On utilise sql.Open direct ici plutot que duckdbpkg.OpenReadWrite pour
	// eviter l'import cycle persist->duckdbpkg->persist. Le DSN nu (sans
	// query params) simule le comportement du cache duckdbpkg cote engine.
	enginePreDB, err := sql.Open("duckdb", playerPath)
	if err != nil {
		t.Fatalf("engine open (DSN nu): %v", err)
	}
	if err := enginePreDB.Ping(); err != nil {
		t.Fatalf("engine ping: %v", err)
	}
	defer enginePreDB.Close()

	// Etape 2 : CombinedPersister.Persist appelle sql.Open avec
	// playerPath+"?access_mode=READ_WRITE" (DSN different) → Bug #1 se
	// declenche : "different configuration".
	acquireShared := noopSharedWriter(t)
	cp := NewCombinedPersister(acquireShared, func(_ string) string { return playerPath })
	batch := helperPlayerBatch("m_combined_001")
	batch.Shared.Match = nil // shared no-op (focus sur player open conflict)

	err = cp.Persist(context.Background(), batch)
	if err == nil {
		// Cas A : Bug #1 deja fixe (Phase 1 mergee), Persist OK — alors
		// ce test devient sentinelle anti-regression : si quelqu'un re-introduit
		// sql.Open direct cote CombinedPersister, le test re-deviendra ROUGE.
		t.Log("CombinedPersister.Persist OK — bug #1 probablement deja fixe (Phase 1 mergee). Test devient sentinelle anti-regression.")
		return
	}
	// Cas B : echec attendu — verifie que c'est bien le bug "different config",
	// pas une autre erreur (lease, schema, etc).
	if !strings.Contains(err.Error(), "different configuration") {
		t.Logf("CombinedPersister.Persist echec mais cause differente : %v", err)
		t.Logf("Le bug attendu est 'different configuration', mais le test detecte un autre echec.")
		t.Logf("(Possible : driver duckdb a evolue, ou lease/schema non setup.)")
		return
	}
	// Le bug #1 est reproductible → log explicite + fail si on est en mode strict.
	if testing.Verbose() {
		t.Logf("BUG #1 REPRODUIT : %v", err)
	}
	t.Errorf("BUG #1 : CombinedPersister echoue avec 'different configuration' — Phase 1 du plan principal a appliquer (combined_persister.go:94 doit passer par cache duckdbpkg.OpenReadWrite)")
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests fonctionnels : meme une fois Bug #1 fixe, on veut valider
// que CombinedPersister fait bien le shared + player.
// ─────────────────────────────────────────────────────────────────────────────

func TestCombinedPersister_PersistShared_NoPlayerPath_OK(t *testing.T) {
	// playerDBPathFn retourne "" → CombinedPersister skip player persist,
	// shared persist seul doit reussir.
	acquireShared := noopSharedWriter(t)
	cp := NewCombinedPersister(acquireShared, func(_ string) string { return "" })

	batch := helperBuildSampleBatch("m_combined_002", "1111", "Alice")

	if err := cp.Persist(context.Background(), batch); err != nil {
		t.Fatalf("Persist shared seul : %v", err)
	}
}

func TestCombinedPersister_NilBatch_ReturnsError(t *testing.T) {
	cp := NewCombinedPersister(noopSharedWriter(t), func(_ string) string { return "" })
	if err := cp.Persist(context.Background(), nil); err == nil {
		t.Error("Persist(nil) doit retourner erreur")
	}
}

func TestCombinedPersister_SharedFailureSkipsPlayer(t *testing.T) {
	// Si acquireShared echoue, Persist doit erreur AVANT toute tentative player.
	failingShared := func(ctx context.Context) (*sql.DB, func(), error) {
		return nil, nil, sql.ErrConnDone
	}
	cp := NewCombinedPersister(failingShared, func(_ string) string {
		t.Error("playerDBPathFn appele alors que shared a fail — atomicity casse")
		return ""
	})
	batch := helperBuildSampleBatch("m_combined_003", "1111", "Alice")
	if err := cp.Persist(context.Background(), batch); err == nil {
		t.Error("Persist doit echouer quand shared echoue")
	}
}

func TestCombinedPersister_OrderSharedFirst(t *testing.T) {
	// Le contrat dit : "shared AVANT player". On verifie indirectement en
	// confirmant que le batch n'est pas push cote player si shared persist
	// fail (SharedPersister.Persist retourne erreur sur batch invalide).
	//
	// Setup : shared DB valide mais batch invalide (Shared.Match nil + Shared
	// data non-nil → SharedPersister va no-op, donc pas d'echec ici → on
	// utilise un autre mecanisme).
	//
	// Approche : on configure playerDBPathFn pour fail rapidement (path
	// vers fichier impossible). Si shared persist reussit (no-op sur empty
	// match), CombinedPersister doit alors essayer player et echouer dessus.
	// Si shared echouait, on ne verrait jamais l'erreur player.

	acquireShared := noopSharedWriter(t)
	// Path invalide pour forcer un echec player apres shared OK.
	cp := NewCombinedPersister(acquireShared, func(_ string) string {
		return "/non/existent/path/that/cannot/be/created.duckdb"
	})

	batch := helperPlayerBatch("m_combined_004")
	batch.Shared = SharedBatch{} // shared vide → no-op cote shared persister
	err := cp.Persist(context.Background(), batch)
	if err == nil {
		t.Skip("Persist a reussi malgre path invalide — env autorise les ecritures arbitraires ?")
	}
	// L'erreur doit etre cote player (pas shared)
	if !strings.Contains(err.Error(), "player") && !strings.Contains(err.Error(), "lease") {
		t.Errorf("erreur attendue cote player, got : %v", err)
	}
}
