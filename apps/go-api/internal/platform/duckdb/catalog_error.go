package duckdb

import (
	"errors"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

// IsCatalogError dit si err est une erreur de catalogue DuckDB (« Catalog Error » : table,
// vue ou schéma absent de la base). C'est une absence durable de l'objet interrogé, pas une
// panne de la base : un appelant peut la traiter comme un résultat vide. Reconnue par le
// type d'erreur du pilote, jamais par le texte du message.
func IsCatalogError(err error) bool {
	var dbErr *duckdb.Error
	return errors.As(err, &dbErr) && dbErr.Type == duckdb.ErrorTypeCatalog
}
