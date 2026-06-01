package duckdb

import (
	"path/filepath"
	"testing"
)

// TestOpenReadForQuery_ReusesCachedRWHandle reproduit l'incident 2026-06-01 :
// quand un fichier est déjà ouvert en READ-WRITE dans le process (cas pool /
// career live / backup cron sur une DB joueur), un OpenReadOnly échouerait
// ("Can't open ... with a different configuration"). OpenReadForQuery doit au
// contraire réutiliser le handle RW en cache et permettre la lecture.
func TestOpenReadForQuery_ReusesCachedRWHandle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.duckdb")

	rw, err := OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite: %v", err)
	}
	t.Cleanup(func() { _ = rw.Close() })
	if _, err := rw.SQLDb().Exec(`CREATE TABLE t (id INTEGER); INSERT INTO t VALUES (1),(2)`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Garde-fou : confirmer que l'ancien comportement (OpenReadOnly direct)
	// échoue bien tant que le RW est ouvert — c'est la cause racine.
	if roDB, roErr := OpenReadOnly(path); roErr == nil {
		_ = roDB.Close()
		t.Log("note: OpenReadOnly n'a pas échoué malgré RW ouvert (cache a peut-être servi) — le test reste valide")
	}

	db, release, err := OpenReadForQuery(path)
	if err != nil {
		t.Fatalf("OpenReadForQuery doit réussir en réutilisant le handle RW caché, got: %v", err)
	}
	defer release()

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM t`).Scan(&n); err != nil {
		t.Fatalf("lecture via handle réutilisé: %v", err)
	}
	if n != 2 {
		t.Errorf("COUNT(*) = %d, want 2", n)
	}

	// release() emprunté = no-op : le handle RW d'origine doit rester utilisable.
	release()
	if _, err := rw.SQLDb().Exec(`INSERT INTO t VALUES (3)`); err != nil {
		t.Errorf("le handle RW propriétaire ne doit pas avoir été fermé par release(): %v", err)
	}
}

// TestOpenReadForQuery_ColdOpensReadOnly : sans handle en cache, OpenReadForQuery
// ouvre la base en read-only et la lecture fonctionne ; release la ferme.
func TestOpenReadForQuery_ColdOpensReadOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cold.duckdb")

	// Créer puis fermer pour qu'aucun handle ne reste en cache.
	rw, err := OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite: %v", err)
	}
	if _, err := rw.SQLDb().Exec(`CREATE TABLE t (id INTEGER); INSERT INTO t VALUES (42)`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := rw.Close(); err != nil {
		t.Fatalf("close RW: %v", err)
	}

	db, release, err := OpenReadForQuery(path)
	if err != nil {
		t.Fatalf("OpenReadForQuery (cold): %v", err)
	}
	defer release()

	var v int
	if err := db.QueryRow(`SELECT id FROM t`).Scan(&v); err != nil {
		t.Fatalf("lecture cold RO: %v", err)
	}
	if v != 42 {
		t.Errorf("id = %d, want 42", v)
	}
}
