//go:build cgo

package wire

import (
	"context"
	"database/sql"
	"strconv"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games/weapons"
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
	if err := weapons.ApplyRegistry(db); err != nil {
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

	// Hors-arsenal H5 classé (2026-07-17) : un UGC sans libellé weapon_labels est
	// désormais RÉSOLU PAR LE REGISTRE (bucket h5_other_ugc) → il compte comme
	// couvert. C'est le mécanisme qui porte la couverture registre H5 à ~100 %
	// (66/66 stock_ids hors sentinelles). inLabel reste false (aucune ligne label).
	ugcH5 := int64(2457457776) // UGC H5, mappé à h5_other_ugc, absent de weapon_labels
	spartanH5 := int64(3168248199)
	inRegH5, inLabelH5 := map[int64]bool{}, map[int64]bool{}
	classifyWeaponCoverage(ctx, db, "halo_5", []int64{ugcH5, spartanH5}, inRegH5, inLabelH5)
	if !inRegH5[ugcH5] {
		t.Errorf("UGC H5 %d devrait être résolu par le registre (h5_other_ugc)", ugcH5)
	}
	if inLabelH5[ugcH5] {
		t.Errorf("UGC H5 %d ne devrait avoir aucun weapon_labels", ugcH5)
	}
	if !inRegH5[spartanH5] {
		t.Errorf("Spartan H5 %d devrait être résolu par le registre (h5_unattributed)", spartanH5)
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
