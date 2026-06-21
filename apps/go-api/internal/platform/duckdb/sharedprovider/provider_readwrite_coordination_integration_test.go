//go:build integration

package sharedprovider_test

import (
	"context"
	"testing"

	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	syncpkg "levelup/go-api/internal/sync"
)

// TestProvider_ReadWriteCoordination_SamePath_integration est LE test clé de
// l'Option A/C (lecture ET écriture d'un même fichier shared via le MÊME
// provider). Il reproduit le scénario multi-titre h5 :
//
//   - un writer (chemin livesync : AcquireSharedWriterStandalone avec provider
//     non-nil → provider.AcquireWriter → PreSwap RO→RW) écrit une ligne ;
//   - PUIS un reader (chemin pool joueur : provider.Get → handle RO) lit.
//
// Sur le MÊME provider/path. Prouve :
//  1. AUCUNE erreur DuckDB "different configuration" — c'est précisément ce que
//     le B-swap élimine en coordonnant RO+RW via le même provider (drain + swap),
//     vs. deux handles RO/RW non coordonnés sur le même fichier in-process.
//  2. La lecture post-write voit la ligne écrite (durabilité du swap RW→RO).
//
// Régression ciblée : si le writer h5 passait provider=nil (legacy OpenSharedDB
// RW direct) ALORS qu'un provider RO existe sur le même path, DuckDB lèverait
// "different configuration". Ce test verrouille le routage par provider unique.
func TestProvider_ReadWriteCoordination_SamePath_integration(t *testing.T) {
	path := setupSharedDB(t)
	ctx := context.Background()

	mgr := sharedprovider.NewManager()
	defer func() { _ = mgr.Close() }()

	// UN SEUL provider pour le path (lecture + écriture passent par lui).
	provider, err := mgr.For(path)
	if err != nil {
		t.Fatalf("For: %v", err)
	}

	// Ancrer un reader RO en vol AVANT le write : prouve que le provider draine
	// ce reader avant d'ouvrir RW (sinon RO+RW concurrents → "different
	// configuration"). On le relâche juste avant le write pour ne pas bloquer
	// le drain au-delà du timeout.
	dbPre, releasePre, err := provider.Get(ctx)
	if err != nil {
		t.Fatalf("Get (reader pré-write): %v", err)
	}
	var ver string
	if err := dbPre.QueryRowContext(ctx, "SELECT version()").Scan(&ver); err != nil {
		t.Fatalf("ping RO pré-write: %v", err)
	}
	releasePre()

	// --- WRITE via le chemin livesync (provider non-nil → PreSwap) ---
	const wantMatchID = "h5-coordination-match-0001"
	wdb, wrelease, err := syncpkg.AcquireSharedWriterStandalone(ctx, provider, path)
	if err != nil {
		t.Fatalf("AcquireSharedWriterStandalone (provider non-nil): %v", err)
	}
	if got := provider.State(); got != sharedprovider.StateRW {
		t.Errorf("pendant le write, State = %v, attendu StateRW", got)
	}
	// start_time est NOT NULL dans le schéma shared → le fournir.
	_, err = wdb.ExecContext(ctx,
		"INSERT INTO match_registry (match_id, start_time) VALUES (?, CURRENT_TIMESTAMP)", wantMatchID)
	if err != nil {
		wrelease()
		t.Fatalf("INSERT match_registry: %v", err)
	}
	wrelease() // RW→RO : draine, ferme RW, rouvre RO.

	// Après release, le provider doit être revenu en RO sans erreur de reopen.
	if got := provider.State(); got != sharedprovider.StateRO {
		t.Fatalf("après release writer, State = %v, attendu StateRO (reopen RO OK)", got)
	}

	// --- READ via le chemin pool joueur (provider.Get → handle RO) ---
	rdb, rrelease, err := provider.Get(ctx)
	if err != nil {
		t.Fatalf("Get (reader post-write): %v", err)
	}
	defer rrelease()

	var got string
	err = rdb.QueryRowContext(ctx,
		"SELECT match_id FROM match_registry WHERE match_id = ?", wantMatchID).Scan(&got)
	if err != nil {
		t.Fatalf("SELECT post-write (la lecture ne voit pas la ligne écrite ?): %v", err)
	}
	if got != wantMatchID {
		t.Errorf("match_id lu = %q, attendu %q", got, wantMatchID)
	}
}
