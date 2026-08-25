package duckdb

import (
	"bytes"
	"database/sql"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
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

// captureSlogText redirige le logger par défaut vers un buffer TEXTE pour la
// durée du test et retourne le buffer. Les logs du helper passent par le logger
// PACKAGE (slog.WarnContext / slog.ErrorContext), donc capturer le défaut suffit.
//
// Jumeau assumé : `captureSlog` (match_view_scoreboard_objective_degrade_test.go,
// buffer JSON) fait la même chose, mais ce fichier-là porte `//go:build
// integration` — ce test-ci doit tourner dans le gate PAR DÉFAUT, il ne peut donc
// pas s'en servir. Factoriser les deux demande de sortir le helper vers un
// fichier de test SANS tag (il serait alors compilé dans les deux configurations) :
// hors périmètre de ce lot, consigné en découverte.
func captureSlogText(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

// TestRecoveringReader_RecoversAfterOwnerClosesHandle reproduit la CAUSE RACINE
// des deux familles best-effort observées en prod le 2026-08-25 (citations
// pve_stat sur shared_pve, IndexMedia sur shared_matches_v2) :
//
//	OpenReadForQuery rend un INSTANTANÉ du *sql.DB. Quand il l'a EMPRUNTÉ au
//	cache process (LookupCachedDB, aucun refCount), le propriétaire — ici le
//	premier OpenReadForQuery, en prod le post-sync d'un autre joueur — peut le
//	fermer sous les pieds de l'emprunteur → « sql: database is closed » EN COURS
//	de requête.
//
// ReopenAllowed (profil shared_pve) : la reprise a le droit d'ouvrir un handle
// RO neuf, ce qui est indispensable ici puisque le Close du propriétaire a
// SUPPRIMÉ l'entrée de cache.
func TestRecoveringReader_RecoversAfterOwnerClosesHandle(t *testing.T) {
	path := seedRecoveringReaderDB(t, "shared_pve.duckdb")

	// Propriétaire : cache miss → refCount = 1, son release ferme le *sql.DB.
	ownerDB, ownerRelease, err := OpenReadForQuery(path)
	if err != nil {
		t.Fatalf("OpenReadForQuery (propriétaire): %v", err)
	}

	// Emprunteur : même instantané, aucun refCount (cf. LookupCachedDB).
	r, err := OpenRecoveringReader(path, ReopenAllowed)
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

	// Le reader doit se réparer seul (re-résolution + retry).
	n, err := readID(t, r)
	if err != nil {
		t.Fatalf("le reader devait récupérer après invalidation, got: %v", err)
	}
	if n != 7 {
		t.Errorf("id après récupération = %d, want 7", n)
	}
}

// TestRecoveringReader_CacheOnlyNeverOpensOnMiss verrouille le correctif P0 de la
// revue R1 : sur un chemin géré par un sharedprovider (profil shared_matches_v2),
// la reprise ne DOIT PAS ouvrir un handle RO neuf. `swapToRW` ferme le RO, notifie,
// puis ouvre le RW — un OpenReadOnly émis dans cette fenêtre peut faire échouer
// l'OpenReadWrite et envoyer le provider en StateError. Cache miss = abandon
// best-effort, pas ouverture.
func TestRecoveringReader_CacheOnlyNeverOpensOnMiss(t *testing.T) {
	path := seedRecoveringReaderDB(t, "shared_matches_v2.duckdb")

	ownerDB, ownerRelease, err := OpenReadForQuery(path)
	if err != nil {
		t.Fatalf("OpenReadForQuery (propriétaire): %v", err)
	}
	_ = ownerDB

	r, err := OpenRecoveringReader(path, ReopenCacheOnly)
	if err != nil {
		t.Fatalf("OpenRecoveringReader: %v", err)
	}
	defer r.Close()

	if n, err := readID(t, r); err != nil || n != 7 {
		t.Fatalf("lecture initiale = (%d, %v), want (7, nil)", n, err)
	}

	// Le propriétaire rend son handle : plus RIEN en cache pour ce chemin —
	// c'est l'état du fichier pendant la fenêtre de swap du provider.
	ownerRelease()
	if _, ok := LookupCachedDB(path); ok {
		t.Fatal("le cache devait être vide après le release du propriétaire (préalable du test)")
	}

	buf := captureSlogText(t)
	_, err = readID(t, r)
	if err == nil {
		t.Fatal("en ReopenCacheOnly sans handle en cache, la lecture doit ÉCHOUER (pas ouvrir un RO neuf)")
	}
	if !IsInvalidatedError(err) {
		t.Errorf("Do doit rendre l'erreur d'ORIGINE (invalidation), got: %v", err)
	}
	// Aucun handle neuf ne doit avoir été ouvert : le cache reste vide.
	if _, ok := LookupCachedDB(path); ok {
		t.Error("ReopenCacheOnly a ouvert un handle — l'invariant sharedprovider est violé")
	}
	if logs := buf.String(); !strings.Contains(logs, "aucun handle en cache") {
		t.Errorf("log d'abandon best-effort attendu, got: %s", logs)
	}

	// 2e lecture — couvre le chemin current(), DISTINCT de refresh() : la reprise
	// ayant échoué, le reader est désormais VIDE (r.db == nil), et c'est current()
	// qui re-résout. Sans cette passe, remettre current() sur openLocked() (au lieu
	// de resolveLocked()) laisse toute la suite verte : un Do ultérieur rouvrirait
	// alors un RO neuf, exactement ce que la reprise s'interdit — mutation prouvée
	// survivante en revue R2, d'où ce cliquet.
	_, err = readID(t, r)
	if !errors.Is(err, errNoCachedHandle) {
		t.Errorf("2e lecture (chemin current, reader vide) = %v, want errNoCachedHandle "+
			"— current() doit lui aussi respecter la ReopenPolicy", err)
	}
	if _, ok := LookupCachedDB(path); ok {
		t.Error("current() a ouvert un handle en ReopenCacheOnly — l'invariant sharedprovider est violé")
	}
}

// TestRecoveringReader_CacheOnlyBorrowsRWHandle : pendant une fenêtre RW du
// provider, l'entrée `rw:` du cache est servie en priorité par LookupCachedDB —
// une lecture marche sur un handle RW, donc la reprise cache-only récupère.
func TestRecoveringReader_CacheOnlyBorrowsRWHandle(t *testing.T) {
	path := seedRecoveringReaderDB(t, "shared_rw_window.duckdb")

	ownerDB, ownerRelease, err := OpenReadForQuery(path)
	if err != nil {
		t.Fatalf("OpenReadForQuery (propriétaire RO): %v", err)
	}
	_ = ownerDB

	r, err := OpenRecoveringReader(path, ReopenCacheOnly)
	if err != nil {
		t.Fatalf("OpenRecoveringReader: %v", err)
	}
	defer r.Close()
	if n, err := readID(t, r); err != nil || n != 7 {
		t.Fatalf("lecture initiale = (%d, %v), want (7, nil)", n, err)
	}

	// Le swap : le RO est rendu, puis un handle RW est posé dans le cache.
	ownerRelease()
	rw, err := OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite (fenêtre RW du provider): %v", err)
	}
	defer func() { _ = rw.Close() }()

	n, err := readID(t, r)
	if err != nil {
		t.Fatalf("la reprise cache-only devait emprunter le handle RW, got: %v", err)
	}
	if n != 7 {
		t.Errorf("id lu sur le handle RW = %d, want 7", n)
	}
}

// TestRecoveringReader_FatalClassIsNotReplayed verrouille le correctif P2 : sur la
// classe FATAL (« database has been invalidated », « Failed to delete all rows
// from index »), le *sql.DB n'est PAS fermé et le cache PAS purgé — la
// re-résolution rend LE MÊME pointeur mort. Rejouer serait un no-op trompeur :
// on rend l'erreur d'origine avec un log dédié, et fn n'est appelée qu'UNE fois.
func TestRecoveringReader_FatalClassIsNotReplayed(t *testing.T) {
	path := seedRecoveringReaderDB(t, "fatal.duckdb")

	// Un propriétaire garde l'entrée de cache VIVANTE pendant tout le test :
	// c'est la situation de la classe FATAL (rien n'est fermé ni purgé).
	_, ownerRelease, err := OpenReadForQuery(path)
	if err != nil {
		t.Fatalf("OpenReadForQuery (propriétaire): %v", err)
	}
	defer ownerRelease()

	r, err := OpenRecoveringReader(path, ReopenAllowed)
	if err != nil {
		t.Fatalf("OpenRecoveringReader: %v", err)
	}
	defer r.Close()

	buf := captureSlogText(t)
	fatal := errors.New("FATAL Error: database has been invalidated because of a previous fatal error")
	calls := 0
	err = r.Do(t.Context(), func(*sql.DB) error {
		calls++
		return fatal
	})
	if !errors.Is(err, fatal) {
		t.Fatalf("Do doit rendre l'erreur d'origine sur la classe FATAL, got: %v", err)
	}
	if calls != 1 {
		t.Errorf("fn appelée %d fois, want 1 (aucun rejeu sur le MÊME handle mort)", calls)
	}
	if logs := buf.String(); !strings.Contains(logs, "invalidation FATALE") {
		t.Errorf("log dédié classe FATAL attendu, got: %s", logs)
	}
}

// TestRecoveringReader_NoRetryOnBusinessError : une erreur métier (ici
// sql.ErrNoRows, le cas « match non-Firefight » de loadPveStats) ne doit
// déclencher AUCUNE re-résolution, et rester comparable par errors.Is.
func TestRecoveringReader_NoRetryOnBusinessError(t *testing.T) {
	path := seedRecoveringReaderDB(t, "business.duckdb")
	r, err := OpenRecoveringReader(path, ReopenAllowed)
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

// TestRecoveringReader_BusinessErrorAfterRecoveryIsNotAnError verrouille le
// correctif P1 : après une récupération RÉUSSIE, un sql.ErrNoRows rendu par le
// retry est un résultat métier NORMAL (match non-Firefight, cas majoritaire) —
// il ne doit pas produire d'ERROR, sinon on remplacerait 372 WARN/mois par
// autant d'ERROR. Seule l'invalidation PERSISTANTE mérite un ERROR.
func TestRecoveringReader_BusinessErrorAfterRecoveryIsNotAnError(t *testing.T) {
	path := seedRecoveringReaderDB(t, "postrecovery.duckdb")

	ownerDB, ownerRelease, err := OpenReadForQuery(path)
	if err != nil {
		t.Fatalf("OpenReadForQuery (propriétaire): %v", err)
	}
	_ = ownerDB

	r, err := OpenRecoveringReader(path, ReopenAllowed)
	if err != nil {
		t.Fatalf("OpenRecoveringReader: %v", err)
	}
	defer r.Close()
	if n, err := readID(t, r); err != nil || n != 7 {
		t.Fatalf("lecture initiale = (%d, %v), want (7, nil)", n, err)
	}
	ownerRelease() // invalide l'instantané emprunté

	buf := captureSlogText(t)
	err = r.Do(t.Context(), func(db *sql.DB) error {
		return db.QueryRowContext(t.Context(), `SELECT id FROM t WHERE id = 999`).Scan(new(int))
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("le retry post-récupération doit rendre sql.ErrNoRows, got: %v", err)
	}
	logs := buf.String()
	if !strings.Contains(logs, "level=WARN") {
		t.Errorf("la récupération doit être loguée en WARN, got: %s", logs)
	}
	if strings.Contains(logs, "level=ERROR") {
		t.Errorf("un sql.ErrNoRows au retry ne doit PAS produire d'ERROR (P1), got: %s", logs)
	}
}

// TestRecoveringReader_RetryBoundedToOne : sur invalidation PERSISTANTE (handle
// réellement re-résolu, mais fn qui rééchoue), fn est rejouée exactement une
// fois — jamais de boucle — et l'ERROR est bien émis.
func TestRecoveringReader_RetryBoundedToOne(t *testing.T) {
	path := seedRecoveringReaderDB(t, "persistent.duckdb")

	ownerDB, ownerRelease, err := OpenReadForQuery(path)
	if err != nil {
		t.Fatalf("OpenReadForQuery (propriétaire): %v", err)
	}
	_ = ownerDB

	r, err := OpenRecoveringReader(path, ReopenAllowed)
	if err != nil {
		t.Fatalf("OpenRecoveringReader: %v", err)
	}
	defer r.Close()
	if n, err := readID(t, r); err != nil || n != 7 {
		t.Fatalf("lecture initiale = (%d, %v), want (7, nil)", n, err)
	}
	ownerRelease() // le handle courant devient mort → la re-résolution ouvrira du neuf

	buf := captureSlogText(t)
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
	if logs := buf.String(); !strings.Contains(logs, "invalidation persistante") {
		t.Errorf("ERROR d'invalidation persistante attendu, got: %s", logs)
	}
}

// TestRecoveringReader_ClosedAndNil : après Close (et sur un reader nil), Do ne
// re-résout RIEN — sinon un handle serait acquis sans personne pour le rendre.
func TestRecoveringReader_ClosedAndNil(t *testing.T) {
	path := seedRecoveringReaderDB(t, "closed.duckdb")
	r, err := OpenRecoveringReader(path, ReopenAllowed)
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
}
