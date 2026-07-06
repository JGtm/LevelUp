//go:build cgo

package wire

import (
	"context"
	"database/sql"
	"strconv"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
)

// TestClassifyWeaponCoverage vérifie la classification registre / weapon_labels /
// non résolu, et le fallback weapon_labels-seul quand le registre est absent.
func TestClassifyWeaponCoverage(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `CREATE TABLE weapon_labels (weapon_id UBIGINT PRIMARY KEY, name_en VARCHAR, name_fr VARCHAR)`); err != nil {
		t.Fatalf("create: %v", err)
	}
	brID := uint64(0x2b1824d542c9679f) // BR75 (registre + label)
	labelOnly := uint64(999)           // dans weapon_labels seulement
	unknown := int64(424242)           // nulle part
	for _, q := range []string{
		"INSERT INTO weapon_labels VALUES (" + strconv.FormatUint(brID, 10) + ", 'BR75', 'BR75')",
		"INSERT INTO weapon_labels VALUES (999, 'Sandwich', 'Sandwich')",
	} {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	if err := halomigrations.ApplyWeaponRegistry(db); err != nil {
		t.Fatalf("registry: %v", err)
	}

	ids := []int64{int64(brID), int64(labelOnly), unknown}

	// Avec registre.
	inReg, inLabel := map[int64]bool{}, map[int64]bool{}
	classifyWeaponCoverage(ctx, db, "halo_infinite", ids, inReg, inLabel)
	if !inReg[int64(brID)] || !inLabel[int64(brID)] {
		t.Errorf("BR75 devrait être registre+label : reg=%v label=%v", inReg[int64(brID)], inLabel[int64(brID)])
	}
	if inReg[int64(labelOnly)] || !inLabel[int64(labelOnly)] {
		t.Errorf("Sandwich devrait être label seul : reg=%v label=%v", inReg[int64(labelOnly)], inLabel[int64(labelOnly)])
	}
	if inReg[unknown] || inLabel[unknown] {
		t.Errorf("id inconnu ne devrait être nulle part")
	}

	// Fallback : registre absent → labels seuls.
	if _, err := db.ExecContext(ctx, "DROP TABLE weapon_ids"); err != nil {
		t.Fatalf("drop: %v", err)
	}
	inReg2, inLabel2 := map[int64]bool{}, map[int64]bool{}
	classifyWeaponCoverage(ctx, db, "halo_infinite", ids, inReg2, inLabel2)
	if len(inReg2) != 0 {
		t.Errorf("sans registre, inReg devrait être vide : %v", inReg2)
	}
	if !inLabel2[int64(brID)] || !inLabel2[int64(labelOnly)] {
		t.Errorf("fallback labels : BR75 et Sandwich devraient être dans inLabel")
	}
}
