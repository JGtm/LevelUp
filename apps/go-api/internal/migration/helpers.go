package migration

import (
	"database/sql"
	"fmt"
	"strings"
)

// columnExists vérifie si une colonne existe dans une table.
func columnExists(db *sql.DB, table, column string) (bool, error) {
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = 'main' AND table_name = ? AND column_name = ?",
		table, column,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// tableExists vérifie si une table existe.
func tableExists(db *sql.DB, table string) (bool, error) {
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'main' AND table_name = ?",
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
	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, colType))
	if err != nil {
		// Ignorer "already exists" au cas où race condition
		if strings.Contains(err.Error(), "already exists") {
			return nil
		}
		return fmt.Errorf("addColumnIfMissing ALTER %s.%s: %w", table, column, err)
	}
	return nil
}

// createIndexSafe crée un index en ignorant les erreurs "already exists".
func createIndexSafe(db *sql.DB, ddl string) error {
	_, err := db.Exec(ddl)
	if err != nil && (strings.Contains(err.Error(), "already exists") ||
		strings.Contains(err.Error(), "read only") ||
		strings.Contains(err.Error(), "does not exist")) {
		return nil
	}
	return err
}

// execScript exécute un script SQL multi-statements.
func execScript(db *sql.DB, script string) error {
	for _, stmt := range splitSQL(script) {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("execScript: %w (stmt=%.80s)", err, stmt)
		}
	}
	return nil
}

// splitSQL découpe un script SQL en instructions individuelles.
func splitSQL(script string) []string {
	var stmts []string
	var cur strings.Builder
	for i := 0; i < len(script); i++ {
		if script[i] == ';' {
			s := strings.TrimSpace(cur.String())
			if s != "" {
				stmts = append(stmts, s)
			}
			cur.Reset()
		} else {
			cur.WriteByte(script[i])
		}
	}
	if s := strings.TrimSpace(cur.String()); s != "" {
		stmts = append(stmts, s)
	}
	return stmts
}
