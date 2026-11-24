package duckdb

// player_read_open.go : la lecture d'une base joueur par un lecteur qui côtoie, dans le même processus,
// l'écrivain de cette base (worker de la file de persistance, post-sync, pool HTTP).
//
// Contrat : un fichier DuckDB n'a qu'une configuration d'ouverture par processus. L'écrivain d'une base
// joueur l'ouvre par [OpenReadWrite] (clé rwCacheKeyPrefix, DSN nu, une connexion). Un lecteur qui
// l'ouvre en lecture seule quand aucun handle n'est en cache (ce que fait [OpenReadForQuery]) publie la
// clé roCacheKeyPrefix : tant qu'il la tient, l'ouverture de l'écrivain échoue (« Can't open a connection
// to same database file with a different configuration »). À l'inverse, un handle RW emprunté sans
// référence par [OpenReadForQuery] se ferme sous la lecture quand l'écrivain rend le sien (« sql:
// database is closed »).
//
// [OpenPlayerReadForQuery] ferme les deux portes : il prend une RÉFÉRENCE sur le handle en cache (RW,
// sinon RO) et, faute de handle, ouvre le fichier dans la configuration de l'écrivain. Le lecteur et
// l'écrivain partagent alors un seul handle ; la dernière référence rendue le ferme.
//
// Mono-écrivain (ADR 0013) : le lecteur n'écrit jamais (SELECT seuls). L'ouverture en écriture ne crée
// aucun second processus ni second handle : c'est le handle que l'écrivain du processus aurait ouvert, et
// qu'il réutilise. Le bail d'écriture (dblease) reste à l'écrivain : le lecteur ne le prend pas, comme le
// pool HTTP, qui tient les bases joueur par la même clé. Le soin de l'ouverture physique en écriture
// (alignement des séquences, physical_open.go) s'applique quand le lecteur ouvre le premier : il
// s'exécute sous le verrou du cache, avant la publication du handle, quand aucun autre handle du
// processus n'existe sur le fichier.

import (
	"database/sql"
	"sync"

	"levelup/go-api/internal/domain/title"
)

// OpenPlayerReadForQuery rend un handle de LECTURE sur la base joueur path, partagé avec l'écrivain du
// processus, et la fonction qui rend la référence prise (à appeler une fois, après la lecture). Le
// handle reste ouvert tant que la référence est tenue, même si son ouvreur le rend entre-temps.
//
// Ordre : handle RW en cache (référence prise), sinon handle RO en cache (référence prise : un lecteur
// le tient déjà, l'écrivain attendra sa fermeture comme avant), sinon ouverture dans la configuration de
// l'écrivain (OpenReadWrite). Un chemin qui n'a pas la forme d'une base joueur (title.IsPlayerDBPath)
// est servi par [OpenReadForQuery] : les bases partagées ont leur propre bascule (ADR 0016), qu'une
// référence de lecteur sur le handle RW empêcherait de fermer.
func OpenPlayerReadForQuery(path string) (*sql.DB, func(), error) {
	if !title.IsPlayerDBPath(path) {
		return OpenReadForQuery(path)
	}
	if db, ok := retainCachedDB(path); ok {
		return db.SQLDb(), releaseOnce(db), nil
	}
	db, err := OpenReadWrite(path)
	if err != nil {
		// Un lecteur a publié la clé RO entre la recherche et l'ouverture : la référence se prend sur
		// son handle.
		if again, ok := retainCachedDB(path); ok {
			return again.SQLDb(), releaseOnce(again), nil
		}
		return nil, nil, err
	}
	return db.SQLDb(), releaseOnce(db), nil
}

// retainCachedDB prend une référence sur le handle ouvert de path (RW d'abord, puis RO), sous le verrou
// du cache. Le *DB rendu se rend par Close, qui décrémente la référence.
func retainCachedDB(path string) (*DB, bool) {
	openDBsMu.Lock()
	defer openDBsMu.Unlock()
	for _, key := range []string{rwCacheKeyPrefix + path, roCacheKeyPrefix + path} {
		cached, ok := openDBs[key]
		if !ok || cached.db == nil || cached.db.closed.Load() {
			continue
		}
		cached.refCount++
		return cached.db, true
	}
	return nil, false
}

// releaseOnce rend la référence de db une seule fois, quel que soit le nombre d'appels.
func releaseOnce(db *DB) func() {
	var once sync.Once
	return func() { once.Do(func() { _ = db.Close() }) }
}
