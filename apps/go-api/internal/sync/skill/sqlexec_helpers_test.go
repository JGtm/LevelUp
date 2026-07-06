package skill

// sqlexec_helpers_test.go — helpers SQL de test (execScript/splitSQL/truncate) copiés depuis
// sync/schema.go lors de l'extraction du package skill (K3c) : skill est un package FEUILLE et
// ne peut pas importer sync (le parent) sans créer un cycle. Duplication tolérée (test-only,
// helpers triviaux).

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func execScript(ctx context.Context, db *sql.DB, script string) error {
	for _, stmt := range splitSQL(script) {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("execScript: %w (stmt=%q)", err, truncate(stmt, 80))
		}
	}
	return nil
}

func splitSQL(script string) []string {
	var stmts []string
	for _, part := range strings.Split(script, ";") {
		if s := strings.TrimSpace(part); s != "" {
			stmts = append(stmts, s)
		}
	}
	return stmts
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
