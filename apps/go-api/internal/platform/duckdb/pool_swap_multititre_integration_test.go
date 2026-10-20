//go:build integration

// pool_swap_multititre_integration_test.go — LA PRISE D'ÉCRITURE D'UN TITRE NE DÉPEND PAS DES
// ÉCRITURES D'UN AUTRE TITRE (D-15, 2026-10-19).
//
// Topologie de production reproduite : un provider par base partagée de titre (le Manager en ouvre
// un par chemin), un PlayerDB du second titre dans le pool global en mode B-swap (SharedReader =
// provider de SON titre, comme config.sharedReaderForTitle). Le serveur n'abonne plus rien aux
// bascules d'un provider (cmd/server) : le test non plus. Ce test tient l'INVARIANT (base B
// inscriptible et tenue une seule fois après une écriture A) ; il a été rouge sur le code fautif
// avec une copie du câblage du serveur d'alors, qui n'existe plus. La résurrection d'un crochet du
// pool qui rouvrirait la base est interdite par archlint/pool_no_shared_handle_test.go.
//
// Le symptôme du serveur local : après une écriture sur la base du titre par défaut, toute prise
// d'écriture de la base Halo 5 échouait (« open RW after RO close : … different configuration »)
// jusqu'au redémarrage, sans autre processus sur la machine. Cause : le pool rouvrait, à chaque
// retour en lecture du provider du titre par défaut, un handle lecture seule sur la base partagée
// de CHAQUE PlayerDB du pool, tous titres confondus ; ces références au handle ref-compté de la
// base Halo 5 survivaient à la fermeture du handle de son provider (ADR 0016, amendement).
package duckdb_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	syncpkg "levelup/go-api/internal/sync"
)

// baseTitre prépare la base partagée (schéma) et la base metadata d'un titre, dans dir.
func baseTitre(t *testing.T, dir string) (sharedPath, metaPath string) {
	t.Helper()
	sharedPath = filepath.Join(dir, "shared_matches_v2.duckdb")
	metaPath = filepath.Join(dir, "metadata.duckdb")
	sb, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		t.Fatalf("OpenReadWrite shared: %v", err)
	}
	if err := syncpkg.EnsureSharedSchema(t.Context(), sb.SQLDb()); err != nil {
		_ = sb.Close()
		t.Fatalf("EnsureSharedSchema: %v", err)
	}
	_ = sb.Close()
	mb, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		t.Fatalf("OpenReadWrite metadata: %v", err)
	}
	_ = mb.Close()
	return sharedPath, metaPath
}

// cycleEcriture prend l'écriture du provider, écrit une ligne, la relâche.
func cycleEcriture(ctx context.Context, p sharedprovider.Provider, matchID string) error {
	w, err := p.AcquireWriter(ctx)
	if err != nil {
		return err
	}
	defer w.Release()
	_, err = w.DB().ExecContext(ctx,
		"INSERT INTO match_registry (match_id, start_time) VALUES (?, NOW())", matchID)
	return err
}

// attendreLecture attend que le provider serve de nouveau les lectures (StateRO), borné.
func attendreLecture(t *testing.T, p sharedprovider.Provider) {
	t.Helper()
	limite := time.Now().Add(10 * time.Second)
	for p.State() != sharedprovider.StateRO {
		if time.Now().After(limite) {
			t.Fatalf("provider %s : état %s, toujours pas revenu en lecture", p.Path(), p.State())
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestPool_EcritureDUnTitreNeBloquePasUnAutre_integration : après une écriture sur la base du
// titre par défaut, la base du second titre reste inscriptible, deux fois de suite (aucun état
// collant), et aucun handle du processus ne la tient hors de son provider.
func TestPool_EcritureDUnTitreNeBloquePasUnAutre_integration(t *testing.T) {
	duckdb.CloseAll()
	t.Cleanup(duckdb.CloseAll)
	ctx := context.Background()

	sharedA, _ := baseTitre(t, t.TempDir())
	dirB := t.TempDir()
	sharedB, metaB := baseTitre(t, dirB)

	provA, err := sharedprovider.New(sharedA)
	if err != nil {
		t.Fatalf("provider A: %v", err)
	}
	t.Cleanup(func() { _ = provA.Close() })
	provB, err := sharedprovider.New(sharedB)
	if err != nil {
		t.Fatalf("provider B: %v", err)
	}
	t.Cleanup(func() { _ = provB.Close() })

	// Joueur du second titre dans le pool, en mode B-swap sur le provider de SON titre.
	if _, err := duckdb.GetOrOpen(ctx, duckdb.PlayerPoolConfig{
		Gamertag: "joueur-b", XUID: "2", TitleSlug: "halo_5",
		PlayerDBPath: filepath.Join(dirB, "player.duckdb"),
		SharedDBPath: sharedB, MetaDBPath: metaB, SharedReader: provB,
	}); err != nil {
		t.Fatalf("GetOrOpen joueur B: %v", err)
	}

	if err := cycleEcriture(ctx, provB, "b-avant"); err != nil {
		t.Fatalf("écriture B avant toute écriture A : %v", err)
	}
	if err := cycleEcriture(ctx, provA, "a-1"); err != nil {
		t.Fatalf("écriture A : %v", err)
	}

	for i, id := range []string{"b-apres-1", "b-apres-2"} {
		attendreLecture(t, provB)
		if err := cycleEcriture(ctx, provB, id); err != nil {
			t.Errorf("écriture B n°%d après une écriture A : %v", i+1, err)
		}
	}

	attendreLecture(t, provB)
	if n := duckdb.CachedReadOnlyRefs(sharedB); n != 1 {
		t.Errorf("base B tenue %d fois en lecture dans le processus, attendu 1 (son provider seul)", n)
	}
}
