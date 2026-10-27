package duckdb

import (
	"context"
	"fmt"
	"testing"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/migration"
)

// createWeaponNameTables crée weapon_name_labels et weapon_label_names dans leur forme finale (le
// DDL est celui du step de migration : une seule définition).
func createWeaponNameTables(t *testing.T, db *DB) {
	t.Helper()
	if err := migration.ExecScript(db.SQLDb(), migration.WeaponTablesDDL()); err != nil {
		t.Fatalf("createWeaponNameTables: %v", err)
	}
}

// putWeaponLabelNames écrit les noms d'une arme dans weapon_label_names, une ligne par langue
// renseignée. L'identifiant est écrit en littéral (UBIGINT dont le bit 63 peut être posé).
func putWeaponLabelNames(t *testing.T, db *DB, id uint64, names i18n.Text) {
	t.Helper()
	for lang, name := range names {
		if name == "" {
			continue
		}
		q := fmt.Sprintf("INSERT INTO weapon_label_names (weapon_id, lang, name) VALUES (%d, ?, ?)", id)
		if _, err := db.Exec(context.Background(), q, string(lang), name); err != nil {
			t.Fatalf("putWeaponLabelNames(%d, %s): %v", id, lang, err)
		}
	}
}
