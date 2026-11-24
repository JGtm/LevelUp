package duckdb

// player_read_open_test.go : la lecture d'une base joueur partage le handle de l'écrivain du processus
// (D-25) : l'écrivain qui ouvre pendant la lecture réutilise le handle au lieu d'échouer sur « different
// configuration », et l'écrivain qui rend le sien ne ferme pas la lecture.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// playerDBFile crée le dossier d'une base joueur (players/<gamertag>/stats.duckdb) et rend son chemin.
func playerDBFile(t *testing.T) string {
	t.Helper()
	CloseAll()
	t.Cleanup(CloseAll)
	path := filepath.Join(t.TempDir(), "titles", "halo_infinite", "players", "JGtm", "stats.duckdb")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// Témoin du défaut : une lecture en lecture seule ouverte la première fait échouer l'écrivain.
func TestOpenReadForQuery_LecturePremiere_EcrivainRefuse(t *testing.T) {
	path := playerDBFile(t)
	w, err := OpenReadWrite(path) // crée le fichier ; rendu aussitôt, plus aucun handle en cache
	if err != nil {
		t.Fatalf("création : %v", err)
	}
	_ = w.Close()
	_, release, err := OpenReadForQuery(path)
	if err != nil {
		t.Fatalf("OpenReadForQuery : %v", err)
	}
	defer release()
	w, err = OpenReadWrite(path)
	if err == nil {
		_ = w.Close()
		t.Fatal("OpenReadWrite pendant une lecture seule : attendu « different configuration »")
	}
	if !strings.Contains(err.Error(), "different configuration") {
		t.Fatalf("OpenReadWrite : %v, attendu « different configuration »", err)
	}
}

// La lecture ouverte la première (aucun handle en cache) : l'écrivain réutilise son handle.
func TestOpenPlayerReadForQuery_LecturePremiere_EcrivainReutilise(t *testing.T) {
	path := playerDBFile(t)
	rdb, release, err := OpenPlayerReadForQuery(path)
	if err != nil {
		t.Fatalf("OpenPlayerReadForQuery : %v", err)
	}
	w, err := OpenReadWrite(path)
	if err != nil {
		release()
		t.Fatalf("OpenReadWrite pendant la lecture : %v", err)
	}
	if w.SQLDb() != rdb {
		t.Error("l'écrivain doit partager le handle de la lecture")
	}
	if _, err := w.SQLDb().Exec(`CREATE TABLE t (x INTEGER); INSERT INTO t VALUES (1)`); err != nil {
		t.Fatalf("écriture : %v", err)
	}
	_ = w.Close()
	var n int
	if err := rdb.QueryRow(`SELECT count(*) FROM t`).Scan(&n); err != nil || n != 1 {
		t.Errorf("lecture après la fermeture de l'écrivain : (%d, %v), attendu 1", n, err)
	}
	release()
	release() // une seule référence rendue, quel que soit le nombre d'appels
	if left := DumpCachedLeaks(); len(left) != 0 {
		t.Errorf("handles encore en cache : %v", left)
	}
}

// L'écrivain ouvert le premier : la lecture prend une référence ; quand l'écrivain rend la sienne, la
// lecture continue sur un handle ouvert (plus de « sql: database is closed »).
func TestOpenPlayerReadForQuery_EcrivainPremier_PasFermeSousLaLecture(t *testing.T) {
	path := playerDBFile(t)
	w, err := OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite : %v", err)
	}
	if _, err := w.SQLDb().Exec(`CREATE TABLE t (x INTEGER)`); err != nil {
		t.Fatal(err)
	}
	rdb, release, err := OpenPlayerReadForQuery(path)
	if err != nil {
		t.Fatalf("OpenPlayerReadForQuery : %v", err)
	}
	defer release()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := rdb.QueryRow(`SELECT count(*) FROM t`).Scan(&n); err != nil {
		t.Fatalf("lecture après la fermeture de l'écrivain : %v", err)
	}
}

// Un handle lecture seule déjà tenu par un autre lecteur : la lecture le partage (aucune ouverture en
// écriture à côté, qui échouerait).
func TestOpenPlayerReadForQuery_LectureSeuleEnCache_Partagee(t *testing.T) {
	path := playerDBFile(t)
	w, err := OpenReadWrite(path)
	if err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	ro, err := OpenReadOnly(path)
	if err != nil {
		t.Fatalf("OpenReadOnly : %v", err)
	}
	defer func() { _ = ro.Close() }()
	rdb, release, err := OpenPlayerReadForQuery(path)
	if err != nil {
		t.Fatalf("OpenPlayerReadForQuery : %v", err)
	}
	defer release()
	if rdb != ro.SQLDb() {
		t.Error("la lecture doit partager le handle lecture seule déjà tenu")
	}
}

// Un chemin qui n'est pas une base joueur passe par OpenReadForQuery (aucune référence sur un handle
// RW d'une base partagée, que sa bascule doit pouvoir fermer).
func TestOpenPlayerReadForQuery_HorsBaseJoueur_CommeOpenReadForQuery(t *testing.T) {
	CloseAll()
	t.Cleanup(CloseAll)
	path := filepath.Join(t.TempDir(), "shared_matches_v2.duckdb")
	w, err := OpenReadWrite(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = w.Close() }()
	_, release, err := OpenPlayerReadForQuery(path)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	openDBsMu.Lock()
	refs := openDBs[rwCacheKeyPrefix+path].refCount
	openDBsMu.Unlock()
	if refs != 1 {
		t.Errorf("références sur le handle RW de la base partagée = %d, attendu 1 (emprunt sans référence)", refs)
	}
}
