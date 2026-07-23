//go:build cgo

// weapon_registry_reconcile_test.go — prouve le CHEMIN DE RÉCONCILIATION du
// registre d'armes au boot (ApplyWeaponRegistry, câblé dans cmd/server/main.go
// pour le titre par défaut ET les titres additionnels dont H5). Scénario réel :
// une metadata.duckdb migrée AVANT l'ajout des 7 stock_ids UGC H5 (h5_other_ugc,
// 2026-07-17) — la migration one-shot h5_add_weapon_registry ne les rejoue jamais.
// Le reconcile idempotent les fait converger, et reste stable au re-run.

package migrations

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// h5UGCStockIDs — les 7 stock_ids UGC H5 sans libellé, mappés h5_other_ugc.
var h5UGCStockIDs = []string{
	"2457457776", "390856427", "3541732101", "642449794",
	"2497647768", "2631958027", "2957796559",
}

const h5OtherUGCCount = `SELECT count(*) FROM weapon_ids
	WHERE title_slug='halo_5' AND id_kind='stock_id' AND weapon_key='h5_other_ugc'`

func TestWeaponRegistry_ReconcileConverges(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// 1. Seed initial (état nominal d'une DB migrée).
	if err := ApplyWeaponRegistry(db); err != nil {
		t.Fatalf("apply initial: %v", err)
	}

	// 2. Simule une DB migrée AVANT l'ajout des 7 UGC : on retire ces lignes.
	for _, id := range h5UGCStockIDs {
		if _, err := db.Exec(
			`DELETE FROM weapon_ids WHERE title_slug='halo_5' AND id_kind='stock_id' AND id_value=?`, id); err != nil {
			t.Fatalf("delete %s: %v", id, err)
		}
	}
	if got := queryCount(t, db, h5OtherUGCCount); got != 0 {
		t.Fatalf("préparation: h5_other_ugc = %d, want 0 (lignes retirées)", got)
	}

	// 3. Réconciliation (= appel boot) : les 7 reviennent, résolvant vers "other".
	if err := ApplyWeaponRegistry(db); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if got := queryCount(t, db, h5OtherUGCCount); got != 7 {
		t.Errorf("après reconcile: h5_other_ugc = %d, want 7", got)
	}
	for _, id := range h5UGCStockIDs {
		var role string
		if err := db.QueryRow(`
			SELECT COALESCE(w.role,'') FROM weapon_ids wi
			JOIN weapons w ON w.title_slug=wi.title_slug AND w.weapon_key=wi.weapon_key
			WHERE wi.title_slug='halo_5' AND wi.id_kind='stock_id' AND wi.id_value=?`, id).Scan(&role); err != nil {
			t.Fatalf("résolution %s après reconcile: %v", id, err)
		}
		if role != "other" {
			t.Errorf("stock_id %s → rôle %q après reconcile, want other", id, role)
		}
	}

	// 4. Re-run stable : total inchangé, aucun doublon (idempotence).
	if err := ApplyWeaponRegistry(db); err != nil {
		t.Fatalf("reconcile (2e): %v", err)
	}
	if got := queryCount(t, db, h5OtherUGCCount); got != 7 {
		t.Errorf("re-run: h5_other_ugc = %d, want 7 (stable)", got)
	}
	if got := queryCount(t, db, `
		SELECT count(*) FROM (
			SELECT id_value FROM weapon_ids
			WHERE title_slug='halo_5' AND id_kind='stock_id'
			GROUP BY id_value HAVING count(*) > 1)`); got != 0 {
		t.Errorf("re-run: %d stock_id H5 dupliqués (idempotence cassée)", got)
	}
}
