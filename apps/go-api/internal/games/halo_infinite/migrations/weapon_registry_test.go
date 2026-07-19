//go:build cgo

// weapon_registry_test.go — applique le seed du registre d'armes sur DuckDB
// :memory: et verrouille : cardinalités (84 armes / 51 familles / 102 ids),
// intégrité référentielle (family_key ∈ weapon_families, weapon_ids → weapons),
// enums class/faction, idempotence (double apply), et quelques résolutions
// (dont le long-tail H5 : grenades/mêlée + hors-arsenal non-combat mappés depuis
// v_weapon_kills).

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
	if got := queryCount(t, db, "SELECT count(*) FROM weapons"); got != 84 {
		t.Errorf("weapons = %d, want 84", got)
	}
	if got := queryCount(t, db, "SELECT count(*) FROM weapon_families"); got != 51 {
		t.Errorf("weapon_families = %d, want 51", got)
	}
	if got := queryCount(t, db, "SELECT count(*) FROM weapon_ids"); got != 102 {
		t.Errorf("weapon_ids = %d, want 102 (36 filmshell + 66 stock_id)", got)
	}
	if got := queryCount(t, db, "SELECT count(*) FROM weapon_ids WHERE id_kind='stock_id'"); got != 66 {
		t.Errorf("weapon_ids stock_id = %d, want 66", got)
	}
	if got := queryCount(t, db, "SELECT count(*) FROM weapons WHERE title_slug='halo_infinite'"); got != 29 {
		t.Errorf("weapons HINF = %d, want 29", got)
	}
	if got := queryCount(t, db, "SELECT count(*) FROM weapons WHERE title_slug='halo_5'"); got != 55 {
		t.Errorf("weapons H5 = %d, want 55", got)
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
	// Les buckets hors-arsenal non-combat (véhicule/tourelle/environnement/…) n'ont
	// pas de faction pertinente → faction vide tolérée. Toute faction NON vide doit
	// rester dans l'enum (garde-fou anti-typo pour les vraies armes).
	if got := queryCount(t, db, `SELECT count(*) FROM weapons
		WHERE faction <> '' AND faction NOT IN ('human','covenant','forerunner','banished')`); got != 0 {
		t.Errorf("%d armes ont une faction hors enum", got)
	}
	if got := queryCount(t, db, `SELECT count(*) FROM weapons
		WHERE class NOT IN ('sidearm','shoulder','heavy','melee','grenade',
			'vehicle','turret','environmental','unattributed','other')`); got != 0 {
		t.Errorf("%d armes ont une class hors enum", got)
	}
	// Rôles : 9 rôles de combat + 5 rôles non-combat (donut hors-arsenal H5).
	if got := queryCount(t, db, `SELECT count(*) FROM weapons
		WHERE role NOT IN ('automatic','precision','sniper','shotgun','sidearm','power','special','melee','grenade',
			'vehicle','turret','environmental','unattributed','other')`); got != 0 {
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

// TestWeaponRegistry_H5NonCombatResolution — fige le classement des 26 stock_ids
// hors-arsenal (décision produit 2026-07-17) : chaque id résout vers son rôle
// non-combat dédié, et les 7 UGC sans libellé convergent tous vers l'unique
// bucket h5_other_ugc. Sans ce gate, une régression re-sortirait « Spartan »
// (~8.8k frags) du bon rôle et fausserait le donut / l'insight coach.
func TestWeaponRegistry_H5NonCombatResolution(t *testing.T) {
	db := openWeaponRegistryDB(t)
	cases := map[string]string{ // stock_id -> rôle attendu
		"3010146366": "vehicle",       // Ghost
		"1063919886": "vehicle",       // Mongoose
		"4028516791": "vehicle",       // Warthog
		"419783896":  "vehicle",       // Banshee
		"3227919741": "vehicle",       // Mantis
		"1730553442": "vehicle",       // Scorpion
		"1206711506": "vehicle",       // Wraith
		"3207900961": "vehicle",       // Wasp
		"3394982816": "vehicle",       // Phaeton
		"2988661926": "turret",        // Chaingun Turret
		"1749823285": "turret",        // Splinter Turret
		"2907783784": "turret",        // Rocket Pod Turret
		"4233134183": "turret",        // Gauss Turret
		"698769165":  "turret",        // Shade Plasma Turret
		"244872079":  "turret",        // Scorpion Anti-Infantry Turret
		"2023669721": "turret",        // Plasma Turret
		"1351500565": "turret",        // Hunter Arm Turret
		"47178948":   "environmental", // Environmental Explosives
		"3168248199": "unattributed",  // Spartan (irréductible)
		"2457457776": "other",         // UGC
		"390856427":  "other",         // UGC
		"3541732101": "other",         // UGC
		"642449794":  "other",         // UGC
		"2497647768": "other",         // UGC
		"2631958027": "other",         // UGC
		"2957796559": "other",         // UGC
	}
	for id, want := range cases {
		var role, key string
		if err := db.QueryRow(`
			SELECT COALESCE(w.role, ''), w.weapon_key
			FROM weapon_ids wi
			JOIN weapons w ON w.title_slug = wi.title_slug AND w.weapon_key = wi.weapon_key
			WHERE wi.title_slug = 'halo_5' AND wi.id_kind = 'stock_id' AND wi.id_value = ?`, id,
		).Scan(&role, &key); err != nil {
			t.Fatalf("résolution stock_id %s: %v", id, err)
		}
		if role != want {
			t.Errorf("stock_id %s → rôle %q, want %q (key %q)", id, role, want, key)
		}
	}
	// Les 7 UGC convergent tous vers l'unique weapon_key h5_other_ugc.
	if got := queryCount(t, db, `SELECT count(*) FROM weapon_ids
		WHERE title_slug='halo_5' AND id_kind='stock_id' AND weapon_key='h5_other_ugc'`); got != 7 {
		t.Errorf("h5_other_ugc mappe %d stock_ids, want 7", got)
	}
	// Aucun stock_id H5 dupliqué (invariant global maintenu après ajout).
	if got := queryCount(t, db, `
		SELECT count(*) FROM (
			SELECT id_value FROM weapon_ids
			WHERE title_slug = 'halo_5' AND id_kind = 'stock_id'
			GROUP BY id_value HAVING count(*) > 1
		)`); got != 0 {
		t.Errorf("%d stock_id H5 dupliqués", got)
	}
}

// TestReconcileWeaponRegistry_Idempotent — fige l'auto-guérison de boot (fix Q2) :
// sur une DB au seed PARTIEL (lignes ajoutées au seed APRÈS que le runner a marqué
// add_weapon_registry « done », jamais réinsérées), ReconcileWeaponRegistry ré-insère
// EXACTEMENT les lignes manquantes et rend leur nombre ; un 2e passage n'insère plus
// rien (INSERT OR IGNORE → zéro doublon).
func TestReconcileWeaponRegistry_Idempotent(t *testing.T) {
	db := openWeaponRegistryDB(t) // seed complet (+ 2e apply : déjà idempotent)

	fullWeapons := queryCount(t, db, "SELECT count(*) FROM weapons")
	fullIDs := queryCount(t, db, "SELECT count(*) FROM weapon_ids")

	// Simule une DB prod au schéma antérieur : on retire le bucket non-combat ajouté
	// « après coup » (h5_unattributed « Spartan » + son stock_id) — le trou EXACT que
	// le fix Q2 doit combler.
	if _, err := db.Exec(`DELETE FROM weapon_ids WHERE weapon_key = 'h5_unattributed'`); err != nil {
		t.Fatalf("delete stock id: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM weapons WHERE weapon_key = 'h5_unattributed' AND title_slug = 'halo_5'`); err != nil {
		t.Fatalf("delete weapon: %v", err)
	}
	if got := queryCount(t, db, "SELECT count(*) FROM weapons"); got != fullWeapons-1 {
		t.Fatalf("pré-reconcile weapons = %d, want %d", got, fullWeapons-1)
	}

	// 1er reconcile : ré-insère les 2 lignes manquantes (1 weapon + 1 weapon_id).
	n, err := ReconcileWeaponRegistry(db, "halo_5")
	if err != nil {
		t.Fatalf("ReconcileWeaponRegistry: %v", err)
	}
	if n != 2 {
		t.Errorf("reconciled rows = %d, want 2 (1 weapon + 1 weapon_id)", n)
	}
	if got := queryCount(t, db, "SELECT count(*) FROM weapons"); got != fullWeapons {
		t.Errorf("post-reconcile weapons = %d, want %d (trou comblé)", got, fullWeapons)
	}
	if got := queryCount(t, db, "SELECT count(*) FROM weapon_ids"); got != fullIDs {
		t.Errorf("post-reconcile weapon_ids = %d, want %d", got, fullIDs)
	}
	if got := queryCount(t, db,
		"SELECT count(*) FROM weapon_ids WHERE id_value = '3168248199'"); got != 1 {
		t.Errorf("stock_id Spartan réinséré = %d, want 1", got)
	}

	// 2e reconcile : plus rien à insérer (idempotent, zéro doublon).
	n2, err := ReconcileWeaponRegistry(db, "halo_5")
	if err != nil {
		t.Fatalf("ReconcileWeaponRegistry (2e): %v", err)
	}
	if n2 != 0 {
		t.Errorf("2e reconcile rows = %d, want 0 (idempotent)", n2)
	}
	if got := queryCount(t, db, "SELECT count(*) FROM weapons"); got != fullWeapons {
		t.Errorf("weapons après 2e reconcile = %d, want %d (aucun doublon)", got, fullWeapons)
	}
	if got := queryCount(t, db, "SELECT count(*) FROM weapon_ids"); got != fullIDs {
		t.Errorf("weapon_ids après 2e reconcile = %d, want %d (aucun doublon)", got, fullIDs)
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
