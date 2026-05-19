package duckdb

import (
	"context"
	"database/sql"
)

// LegacySharedReader wrap un *DB déjà ouvert dans l'interface SharedReader.
// Le release retourné est un no-op : l'appelant garde la propriété du *DB
// et le ferme lui-même via son lifecycle propre.
//
// **Statut post-sprint B1 (commits 1-12) : kill-switch d'urgence.**
//
// Le Provider B-swap (sharedprovider.Provider) est désormais le chemin par
// défaut en production (LEVELUP_USE_SHARED_PROVIDER non défini ou = "1").
// LegacySharedReader survit pour deux raisons :
//
//  1. **Kill-switch prod** : si une régression critique apparaît avec le
//     Provider (e.g. swap RW qui bloque, deadlock, fuite de file handle),
//     un opérateur peut basculer LEVELUP_USE_SHARED_PROVIDER=0 sans
//     redéploiement de code. Le serveur retombe alors sur le comportement
//     pré-sprint (OpenReadOnly direct, sync RW non coordonné — bug
//     "different configuration" théoriquement possible mais survivant).
//
//  2. **Tests** : les tests integration `_test.go` du package duckdb
//     instancient des PlayerDB avec un `*DB` direct, sans plomberie
//     Provider/Subscribe. LegacySharedReader leur permet de satisfaire
//     l'interface SharedReader sans changer le setup test.
//
// Plan de retrait : si après un trimestre en prod (≥ 2026-Q3) aucun
// rollback `=0` n'a été activé et les compteurs
// `shared_provider_swap_failures_total` restent à zéro, supprimer ce type
// et migrer les tests vers le Provider via Manager.For(memPath). ADR à
// rédiger avant suppression.
func LegacySharedReader(db *DB) SharedReader {
	return &legacySharedReader{db: db}
}

type legacySharedReader struct{ db *DB }

func (r *legacySharedReader) Get(_ context.Context) (*sql.DB, func(), error) {
	return r.db.SQLDb(), func() {}, nil
}
