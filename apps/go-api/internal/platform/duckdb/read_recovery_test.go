package duckdb

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
)

// seedRecoveringReaderDB crée une base de test peuplée puis referme tous les
// handles, pour que le cache process reparte VIDE (le premier OpenReadForQuery
// du test soit bien celui qui possède le refCount).
func seedRecoveringReaderDB(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	rw, err := OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite: %v", err)
	}
	if _, err := rw.SQLDb().Exec(`CREATE TABLE t (id INTEGER); INSERT INTO t VALUES (7)`); err != nil {
		_ = rw.Close()
		t.Fatalf("seed: %v", err)
	}
	if err := rw.Close(); err != nil {
		t.Fatalf("close RW seed: %v", err)
	}
	return path
}

// readID lit l'unique ligne de la table de test via le reader.
func readID(t *testing.T, r *RecoveringReader) (int, error) {
	t.Helper()
	var n int
	err := r.Do(t.Context(), func(db *sql.DB) error {
		return db.QueryRowContext(t.Context(), `SELECT id FROM t`).Scan(&n)
	})
	return n, err
}

// TestRecoveringReader_RecoversAfterOwnerClosesHandle reproduit la CAUSE RACINE
// des deux familles best-effort observées en prod le 2026-08-25 (citations
// pve_stat sur shared_pve, IndexMedia sur shared_matches_v2) :
//
//	OpenReadForQuery rend un INSTANTANÉ du *sql.DB. Quand il l'a EMPRUNTÉ au
//	cache process (LookupCachedDB, aucun refCount), le propriétaire — ici le
//	premier OpenReadForQuery, en prod le pool RO fermé par le B-swap ou le
//	post-sync d'un autre joueur — peut le fermer sous les pieds de l'emprunteur
//	→ « sql: database is closed » EN COURS de requête.
//
// Le reader doit ré-ouvrir et rejouer la lecture une fois, sans intervention.
func TestRecoveringReader_RecoversAfterOwnerClosesHandle(t *testing.T) {
	path := seedRecoveringReaderDB(t, "shared_pve.duckdb")

	// Propriétaire : cache miss → refCount = 1, son release ferme le *sql.DB.
	ownerDB, ownerRelease, err := OpenReadForQuery(path)
	if err != nil {
		t.Fatalf("OpenReadForQuery (propriétaire): %v", err)
	}

	// Emprunteur : même instantané, aucun refCount (cf. LookupCachedDB).
	r, err := OpenRecoveringReader(path)
	if err != nil {
		t.Fatalf("OpenRecoveringReader: %v", err)
	}
	defer r.Close()

	if n, err := readID(t, r); err != nil || n != 7 {
		t.Fatalf("lecture initiale = (%d, %v), want (7, nil)", n, err)
	}

	// Le propriétaire rend son handle → entrée de cache supprimée + Close().
	ownerRelease()

	// Preuve directe de la cause racine : l'instantané est mort.
	if err := ownerDB.QueryRow(`SELECT id FROM t`).Scan(new(int)); err == nil {
		t.Fatal("l'instantané emprunté devait être mort après le release du propriétaire")
	} else if !IsInvalidatedError(err) {
		t.Fatalf("erreur inattendue sur l'instantané mort: %v", err)
	}

	// Le reader doit se réparer seul (ré-ouverture + retry).
	n, err := readID(t, r)
	if err != nil {
		t.Fatalf("le reader devait récupérer après invalidation, got: %v", err)
	}
	if n != 7 {
		t.Errorf("id après récupération = %d, want 7", n)
	}
}

// TestRecoveringReader_NoRetryOnBusinessError : une erreur métier (ici
// sql.ErrNoRows, le cas « match non-Firefight » de loadPveStats) ne doit
// déclencher AUCUNE ré-ouverture, et rester comparable par errors.Is.
func TestRecoveringReader_NoRetryOnBusinessError(t *testing.T) {
	path := seedRecoveringReaderDB(t, "business.duckdb")
	r, err := OpenRecoveringReader(path)
	if err != nil {
		t.Fatalf("OpenRecoveringReader: %v", err)
	}
	defer r.Close()

	calls := 0
	err = r.Do(t.Context(), func(db *sql.DB) error {
		calls++
		return db.QueryRowContext(t.Context(), `SELECT id FROM t WHERE id = 999`).Scan(new(int))
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("Do doit propager sql.ErrNoRows tel quel, got: %v", err)
	}
	if calls != 1 {
		t.Errorf("fn appelée %d fois, want 1 (aucun retry sur erreur métier)", calls)
	}
}

// TestRecoveringReader_RetryBoundedToOne : sur invalidation PERSISTANTE, fn est
// rejouée exactement une fois — jamais de boucle.
func TestRecoveringReader_RetryBoundedToOne(t *testing.T) {
	path := seedRecoveringReaderDB(t, "persistent.duckdb")
	r, err := OpenRecoveringReader(path)
	if err != nil {
		t.Fatalf("OpenRecoveringReader: %v", err)
	}
	defer r.Close()

	sentinel := errors.New("sql: database is closed")
	calls := 0
	err = r.Do(t.Context(), func(*sql.DB) error {
		calls++
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("Do doit remonter l'erreur du 2e essai, got: %v", err)
	}
	if calls != 2 {
		t.Errorf("fn appelée %d fois, want 2 (1 essai + 1 retry borné)", calls)
	}
}

// TestRecoveringReader_ClosedAndNil : après Close (et sur un reader nil), Do ne
// ré-ouvre RIEN — sinon un handle serait acquis sans personne pour le rendre.
func TestRecoveringReader_ClosedAndNil(t *testing.T) {
	path := seedRecoveringReaderDB(t, "closed.duckdb")
	r, err := OpenRecoveringReader(path)
	if err != nil {
		t.Fatalf("OpenRecoveringReader: %v", err)
	}
	r.Close()
	r.Close() // idempotent

	if _, err := readID(t, r); !errors.Is(err, ErrReaderClosed) {
		t.Errorf("Do après Close = %v, want ErrReaderClosed", err)
	}

	var nilReader *RecoveringReader
	nilReader.Close() // ne doit pas paniquer
	if err := nilReader.Do(t.Context(), func(*sql.DB) error { return nil }); !errors.Is(err, ErrReaderClosed) {
		t.Errorf("Do sur reader nil = %v, want ErrReaderClosed", err)
	}
	if got := nilReader.Path(); got != "" {
		t.Errorf("Path() sur reader nil = %q, want \"\"", got)
	}
}
