package migration

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// DuckDB column type names (utilises lors des ALTER TABLE ADD COLUMN).
// Centralisés ici pour réduire la duplication des littéraux dans les
// migrations additives (cf. lint goconst).
const (
	colDouble    = "DOUBLE"
	colFloat     = "FLOAT"
	colInteger   = "INTEGER"
	colSmallInt  = "SMALLINT"
	colVarchar   = "VARCHAR"
	colTimestamp = "TIMESTAMP"
	colTinyint   = "TINYINT"
	colBoolean   = "BOOLEAN"
)

// bootCtx retourne le contexte racine utilise par les migrations DDL boot-time.
// Les migrations tournent au demarrage du serveur (cmd/server/main.go) ou au
// boot d'un pool DuckDB (internal/platform/duckdb/pool.go) : il n'existe pas
// de scope HTTP/RPC pertinent, et l'annulation n'a pas de valeur metier (un
// rollback laisse la DB dans un etat incoherent). On utilise donc explicitement
// context.Background() pour satisfaire le lint noctx tout en preservant la
// semantique non-cancellable d'origine.
func bootCtx() context.Context { return context.Background() }

// columnExists verifie si une colonne existe dans une table.
func columnExists(db *sql.DB, table, column string) (bool, error) {
	var count int
	err := db.QueryRowContext(
		bootCtx(),
		"SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = 'main' AND table_name = ? AND column_name = ?",
		table, column,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// tableExists verifie si une table existe.
func tableExists(db *sql.DB, table string) (bool, error) {
	var count int
	err := db.QueryRowContext(
		bootCtx(),
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'main' AND table_name = ?",
		table,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// hasPrimaryKey indique si une table porte une contrainte PRIMARY KEY.
// Utilise duckdb_constraints() (catalogue DuckDB). Sert aux migrations
// correctives sur des tables creees historiquement via CREATE TABLE IF NOT
// EXISTS avant l'ajout d'une PK (la PK n'est alors jamais appliquee).
func hasPrimaryKey(db *sql.DB, table string) (bool, error) {
	var count int
	err := db.QueryRowContext(
		bootCtx(),
		"SELECT COUNT(*) FROM duckdb_constraints() WHERE table_name = ? AND constraint_type = 'PRIMARY KEY'",
		table,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// addColumnIfMissing ajoute une colonne si elle n'existe pas.
func addColumnIfMissing(db *sql.DB, table, column, colType string) error {
	exists, err := columnExists(db, table, column)
	if err != nil {
		return fmt.Errorf("addColumnIfMissing check %s.%s: %w", table, column, err)
	}
	if exists {
		return nil
	}
	_, err = db.ExecContext(bootCtx(), fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, colType))
	if err != nil {
		// Ignorer "already exists" au cas ou race condition
		if strings.Contains(err.Error(), "already exists") {
			return nil
		}
		return fmt.Errorf("addColumnIfMissing ALTER %s.%s: %w", table, column, err)
	}
	return nil
}

// createIndexSafe cree un index en ignorant les erreurs "already exists".
func createIndexSafe(db *sql.DB, ddl string) error {
	_, err := db.ExecContext(bootCtx(), ddl)
	if err != nil && (strings.Contains(err.Error(), "already exists") ||
		strings.Contains(err.Error(), "read only") ||
		strings.Contains(err.Error(), "does not exist")) {
		return nil
	}
	return err
}

// execScript execute un script SQL multi-statements.
func execScript(db *sql.DB, script string) error {
	for _, stmt := range splitSQL(script) {
		if _, err := db.ExecContext(bootCtx(), stmt); err != nil {
			return fmt.Errorf("execScript: %w (stmt=%.80s)", err, stmt)
		}
	}
	return nil
}

// splitSQL decoupe un script SQL en instructions individuelles. Le découpage est
// CONSCIENT des commentaires de ligne `--` : un `;` situé À L'INTÉRIEUR d'un
// commentaire `--` (jusqu'au saut de ligne) n'est PAS un séparateur d'instruction
// (sinon une note de doc contenant un `;` casserait le statement en deux moitiés
// dont l'une devient du SQL invalide). De plus, les fragments qui ne contiennent
// que des commentaires `--` et/ou des espaces (typiquement la note qui suit le
// dernier `;` d'un CREATE) sont ignorés : un commentaire-seul n'est pas une
// instruction exécutable (DuckDB la rejette en "empty query").
func splitSQL(script string) []string {
	var stmts []string
	var cur strings.Builder
	inLineComment := false
	flush := func() {
		if s := strings.TrimSpace(cur.String()); s != "" && !isCommentOnly(s) {
			stmts = append(stmts, s)
		}
		cur.Reset()
	}
	for i := 0; i < len(script); i++ {
		c := script[i]
		switch {
		case c == '\n':
			inLineComment = false
			cur.WriteByte(c)
		case !inLineComment && c == '-' && i+1 < len(script) && script[i+1] == '-':
			inLineComment = true
			cur.WriteByte(c)
		case c == ';' && !inLineComment:
			flush()
		default:
			cur.WriteByte(c)
		}
	}
	flush()
	return stmts
}

// isCommentOnly indique si un fragment SQL ne contient que des lignes de
// commentaire `--` et/ou des lignes vides (rien d'exécutable).
func isCommentOnly(fragment string) bool {
	for _, line := range strings.Split(fragment, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		return false
	}
	return true
}
