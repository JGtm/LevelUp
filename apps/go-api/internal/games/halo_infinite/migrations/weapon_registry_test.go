//go:build cgo

// weapon_registry_test.go — applique le seed du registre d'armes sur DuckDB
// :memory: et verrouille : cardinalités (64 armes / 46 familles / 76 ids),
// intégrité référentielle (family_key ∈ weapon_families, weapon_ids → weapons),
// enums class/faction, idempotence (double apply), et quelques résolutions
// (dont le long-tail H5 : grenades/mêlée mappés depuis v_weapon_kills).

package migrations

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openWeaponRegistryDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := ApplyWeaponRegistry(db); err != nil {
		t.Fatalf("ApplyWeaponRegistry: %v", err)
	}
	// Idempotence : un second apply (INSERT OR IGNORE) ne doit pas échouer ni dupliquer.
	if err := ApplyWeaponRegistry(db); err != nil {
		t.Fatalf("ApplyWeaponRegistry (2e): %v", err)
	}
	return db
}

func queryCount(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return n
}

func TestWeaponRegistry_SeedCardinalities(t *testing.T) {
	db := openWeaponRegistryDB(t)
	if got := queryCount(t, db, "SELECT count(*) FROM weapons"); got != 64 {
		t.Errorf("weapons = %d, want 64", got)
	}
	if got := queryCount(t, db, "SELECT count(*) FROM weapon_families"); got != 46 {
		t.Errorf("weapon_families = %d, want 46", got)
	}
	if got := queryCount(t, db, "SELECT count(*) FROM weapon_ids"); got != 76 {
		t.Errorf("weapon_ids = %d, want 76 (36 filmshell + 40 stock_id)", got)
	}
	if got := queryCount(t, db, "SELECT count(*) FROM weapon_ids WHERE id_kind='stock_id'"); got != 40 {
		t.Errorf("weapon_ids stock_id = %d, want 40", got)
	}
	if got := queryCount(t, db, "SELECT count(*) FROM weapons WHERE title_slug='halo_infinite'"); got != 29 {
		t.Errorf("weapons HINF = %d, want 29", got)
	}
	if got := queryCount(t, db, "SELECT count(*) FROM weapons WHERE title_slug='halo_5'"); got != 35 {
		t.Errorf("weapons H5 = %d, want 35", got)
	}
}

func TestWeaponRegistry_ReferentialIntegrity(t *testing.T) {
	db := openWeaponRegistryDB(t)
	// Toute family_key citée existe dans weapon_families.
	if got := queryCount(t, db, `SELECT count(*) FROM weapons w
		LEFT JOIN weapon_families f ON w.family_key = f.family_key
		WHERE w.family_key IS NOT NULL AND f.family_key IS NULL`); got != 0 {
		t.Errorf("%d armes pointent une family_key absente de weapon_families", got)
	}
	// Tout weapon_ids résout vers un weapon (même titre).
	if got := queryCount(t, db, `SELECT count(*) FROM weapon_ids i
		LEFT JOIN weapons w ON i.title_slug = w.title_slug AND i.weapon_key = w.weapon_key
		WHERE w.weapon_key IS NULL`); got != 0 {
		t.Errorf("%d weapon_ids pointent un weapon_key inexistant", got)
	}
}

func TestWeaponRegistry_Enums(t *testing.T) {
	db := openWeaponRegistryDB(t)
	if got := queryCount(t, db, `SELECT count(*) FROM weapons
		WHERE faction NOT IN ('human','covenant','forerunner','banished')`); got != 0 {
		t.Errorf("%d armes ont une faction hors enum", got)
	}
	if got := queryCount(t, db, `SELECT count(*) FROM weapons
		WHERE class NOT IN ('sidearm','shoulder','heavy','melee','grenade')`); got != 0 {
		t.Errorf("%d armes ont une class hors enum", got)
	}
	if got := queryCount(t, db, `SELECT count(*) FROM weapons
		WHERE role NOT IN ('automatic','precision','sniper','shotgun','sidearm','power','special','melee','grenade')`); got != 0 {
		t.Errorf("%d armes ont un role hors enum", got)
	}
	if got := queryCount(t, db, "SELECT count(*) FROM weapons WHERE role IS NULL OR role = ''"); got != 0 {
		t.Errorf("%d armes sans role", got)
	}
}

// TestWeaponRegistry_H5LongTailResolution — fige la couverture du long-tail H5
// (audit v_weapon_kills 2026-07-17) : chaque stock_id nouvellement mappé résout
// vers le rôle de combat attendu (grenade / melee). Sans ce gate, une régression
// du registre re-ouvrirait le trou du donut « Frags par type d'arme » H5.
func TestWeaponRegistry_H5LongTailResolution(t *testing.T) {
	db := openWeaponRegistryDB(t)
	cases := map[string]string{ // stock_id -> rôle attendu
		"4106030681": "grenade", // FRAG GRENADE
		"2460880172": "grenade", // PLASMA GRENADE
		"3190813201": "grenade", // SPLINTER GRENADE
		"409331533":  "melee",   // Golf Club
		"393532233":  "melee",   // Oddball
	}
	for id, want := range cases {
		var role string
		if err := db.QueryRow(`
			SELECT COALESCE(w.role, '')
			FROM weapon_ids wi
			JOIN weapons w ON w.title_slug = wi.title_slug AND w.weapon_key = wi.weapon_key
			WHERE wi.title_slug = 'halo_5' AND wi.id_kind = 'stock_id' AND wi.id_value = ?`, id,
		).Scan(&role); err != nil {
			t.Fatalf("résolution stock_id %s: %v", id, err)
		}
		if role != want {
			t.Errorf("stock_id %s → rôle %q, want %q", id, role, want)
		}
	}
	// Aucun stock_id H5 dupliqué (un id = un weapon_key unique).
	if got := queryCount(t, db, `
		SELECT count(*) FROM (
			SELECT id_value FROM weapon_ids
			WHERE title_slug = 'halo_5' AND id_kind = 'stock_id'
			GROUP BY id_value HAVING count(*) > 1
		)`); got != 0 {
		t.Errorf("%d stock_id H5 dupliqués", got)
	}
}

func TestWeaponRegistry_FilmshellResolution(t *testing.T) {
	db := openWeaponRegistryDB(t)
	// Multiplicités attendues (variantes d'arme partageant un weapon_key).
	cases := map[string]int{
		"hinf_br75":           1,
		"hinf_bandit":         2,
		"hinf_energy_sword":   4,
		"hinf_gravity_hammer": 3,
		"hinf_shock_rifle":    2,
	}
	for key, want := range cases {
		got := queryCount(t, db,
			"SELECT count(*) FROM weapon_ids WHERE id_kind='filmshell' AND weapon_key=?", key)
		if got != want {
			t.Errorf("filmshell ids pour %s = %d, want %d", key, got, want)
		}
	}
}
