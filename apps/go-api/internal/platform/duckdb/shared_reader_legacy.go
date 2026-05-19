package duckdb

import (
	"context"
	"database/sql"
)

// LegacySharedReader wrap un *DB déjà ouvert dans l'interface SharedReader.
// Le release retourné est un no-op : l'appelant garde la propriété du *DB
// et le ferme lui-même via son lifecycle propre.
//
// Cas d'usage pendant la phase de migration (commits 6-8) vers
// sharedprovider.Provider :
//   - main.go : flag LEVELUP_USE_SHARED_PROVIDER=0 (default — comportement
//     identique à avant le sprint, le sync continue d'ouvrir RW directement
//     et la conn RO ici reste partagée par le serveur)
//   - tests : repo_test.go évite l'overhead d'instancier un Provider complet
//
// Sera retiré au commit 9 quand le Provider devient le seul chemin.
func LegacySharedReader(db *DB) SharedReader {
	return &legacySharedReader{db: db}
}

type legacySharedReader struct{ db *DB }

func (r *legacySharedReader) Get(_ context.Context) (*sql.DB, func(), error) {
	return r.db.SQLDb(), func() {}, nil
}
