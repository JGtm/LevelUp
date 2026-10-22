package duckdb

import (
	"database/sql"
	"testing"
)

// Ponts vers le paquet de test externe (duckdb_test), qui seul peut importer `persist`.

// OuvrirSharedMigreePourTest ouvre un shared `:memory:` aux vraies migrations.
func OuvrirSharedMigreePourTest(t *testing.T) *sql.DB { return ouvrirSharedMigree(t) }

// EcrireResumesAide écrit, par l'aide de test, les résumés de la passe courante de chaque match.
func EcrireResumesAide(t *testing.T, db *sql.DB) { ecrireResumes(t, db) }
