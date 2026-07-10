//go:build integration

// Package sharedprovider_test (baseline) — documente la limite native du
// driver DuckDB-Go v2 que résout le futur SharedDBProvider (stratégie B-swap).
//
// Ce fichier est ajouté en premier (commit 1 de la roadmap) avant toute
// implémentation du provider : il sert d'ancrage régression pour la
// signature exacte du bug observé en prod.
package sharedprovider_test

import (
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/platform/duckdb"
	syncpkg "levelup/go-api/internal/sync"
)

// TestBaselineRedConflictExists_integration documente la limite native de
// DuckDB-Go v2 que résout le SharedDBProvider (B-swap).
//
// Symptôme observé en prod (gamertag Madina97294, 2026-05-18) :
//
//	auto_sync: RunDelta échoué — run OpenSharedDB: OpenSharedDB open ...
//	shared_matches_v2.duckdb: duckdb.OpenReadWrite(...): database/sql/driver:
//	could not connect to database: Connection Error: Can't open a connection
//	to same database file with a different configuration than existing
//	connections
//
// Topologie reproduite ici :
//  1. main.go ouvre shared_matches_v2.duckdb en RO au boot (cf.
//     cmd/server/main.go:209) — gardée tout le lifecycle serveur.
//  2. Le sync engine tente d'ouvrir le même fichier en RW lors d'un
//     RunDelta (cf. internal/sync/schema.go:280 + internal/sync/engine.go).
//
// Ce test est "rouge attendu" : il PASSE tant que duckdb-go conserve cette
// limite (état nominal du projet). Le jour où il ÉCHOUE — soit parce que
// le driver a été patché, soit parce qu'on a changé de version majeure —
// c'est un signal explicite que le sharedprovider B-swap peut être
// simplifié ou archivé.
//
// Inverse de l'usage habituel d'un test : on assert ici la PRÉSENCE d'un
// bug, pas son absence.
func TestBaselineRedConflictExists_integration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shared.duckdb")

	// Étape 0 : créer le fichier + schéma shared minimal pour avoir un état
	// initial réaliste (en prod, le fichier existe toujours et a un schéma).
	// On ferme tout de suite : ce n'est pas la conn qui doit déclencher le bug.
	bootstrap, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("bootstrap OpenReadWrite: %v", err)
	}
	if err := syncpkg.EnsureSharedSchema(t.Context(), bootstrap.SQLDb()); err != nil {
		_ = bootstrap.Close()
		t.Fatalf("EnsureSharedSchema: %v", err)
	}
	if err := bootstrap.Close(); err != nil {
		t.Fatalf("bootstrap Close: %v", err)
	}

	// Étape 1 : simuler le boot serveur — ouvre shared en RO et garde la conn
	// (équivalent main.go:209). La conn reste vivante via le defer.
	roDB, err := duckdb.OpenReadOnly(path)
	if err != nil {
		t.Fatalf("OpenReadOnly (simulant main.go boot): %v", err)
	}
	defer func() { _ = roDB.Close() }()

	// Sanity check : la conn RO doit fonctionner (sinon l'erreur n'a pas de
	// sens — on veut isoler la pathologie côté RW).
	var version string
	if err := roDB.SQLDb().QueryRow("SELECT version()").Scan(&version); err != nil {
		t.Fatalf("RO sanity ping: %v", err)
	}

	// Étape 2 : simuler le sync engine — tente d'ouvrir le même fichier en RW.
	// C'est exactement ce que fait sync.OpenSharedDB (schema.go:280).
	// CE CALL DOIT ÉCHOUER.
	rwDB, err := duckdb.OpenReadWrite(path)
	if rwDB != nil {
		// Si on arrive ici, le driver a accepté l'ouverture concurrente —
		// nettoyer avant d'asserter pour ne pas leaker.
		_ = rwDB.Close()
	}
	if err == nil {
		t.Fatal(
			"OpenReadWrite a réussi alors qu'une conn RO existe déjà sur le même fichier. " +
				"Cela signifie que duckdb-go a corrigé la limite \"different configuration\" — " +
				"le sharedprovider B-swap n'a plus de raison d'exister. " +
				"Voir docs/adr/0016-shared-db-provider-b-swap.md pour le plan d'archive.")
	}

	// La signature exacte du message doit être stable pour distinguer ce bug
	// d'autres erreurs DuckDB (lock OS, fichier corrompu, etc.).
	if !strings.Contains(err.Error(), "different configuration") {
		t.Errorf(
			"erreur attendue contenant \"different configuration\", "+
				"signature inattendue : %v", err)
	}
}
