package duckdb

import (
	"context"
	"database/sql"
)

// LegacySharedReader wrap un *DB déjà ouvert dans l'interface SharedReader.
// Le release retourné est un no-op : l'appelant garde la propriété du *DB
// et le ferme lui-même via son lifecycle propre.
//
// Le serveur n'en fait pas usage : il sert la base partagée par un
// sharedprovider.Provider (ADR 0016, B-swap). Ce wrapper sert les contextes
// SANS provider :
//
//  1. un pool joueur ouvert sans SharedReader (PlayerPoolConfig.SharedReader nil)
//     ou un PlayerDB construit à la main : PlayerDB.SharedReadDB() retombe sur
//     LegacySharedReader(pdb.Shared) ;
//  2. les tests du package duckdb et des couches au-dessus, qui instancient des
//     PlayerDB avec un *DB direct, sans plomberie Provider.
func LegacySharedReader(db *DB) SharedReader {
	return &legacySharedReader{db: db}
}

type legacySharedReader struct{ db *DB }

func (r *legacySharedReader) Get(_ context.Context) (*sql.DB, func(), error) {
	return r.db.SQLDb(), func() {}, nil
}
