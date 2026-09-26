package migrations

// metadata_reclass_sword_hammer_test.go — LE STEP QUI RATTRAPE UN SEED INSERT-ONLY.
//
// POURQUOI CE TEST EXISTE, ET CE QU'IL AURAIT ATTRAPÉ. Le reclassement de l'épée et du
// marteau a d'abord été fait dans le seul registre Go — et il n'a RIEN changé sur une
// copie de la base de production. Cause : `weapons.ReconcileRegistry`, rejoué à chaque
// boot, est en `INSERT OR IGNORE`. Il propage une clé NOUVELLE, jamais une valeur
// MODIFIÉE. Le changement aurait donc été vrai dans le binaire et faux dans toutes les
// bases existantes, sans qu'aucun test ne le dise.
//
// Le test monte donc une base à l'ANCIEN état (classe `melee`, comme une base de prod
// déjà semée), applique le step, et exige le nouvel état.

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// baseRegistreArmes forge la table `weapons` à l'état d'une base de production semée
// AVANT le 2026-09-01 : l'épée et le marteau y sont en `melee`.
func baseRegistreArmes(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE weapons (
			weapon_key VARCHAR, title_slug VARCHAR, name VARCHAR,
			class VARCHAR, role VARCHAR, family_key VARCHAR,
			PRIMARY KEY (title_slug, weapon_key)
		);
		INSERT INTO weapons VALUES
			('hinf_energy_sword',   'halo_infinite', 'Energy Sword',   'melee',  'melee',  'energy_sword'),
			('hinf_gravity_hammer', 'halo_infinite', 'Gravity Hammer', 'melee',  'melee',  'gravity_hammer'),
			('hinf_s7_sniper',      'halo_infinite', 'S7 Sniper',      'heavy',  'sniper', 'sniper_rifle'),
			('h5_energy_sword',     'halo_5',        'Energy Sword',   'melee',  'melee',  'energy_sword');
	`); err != nil {
		t.Fatalf("forge weapons: %v", err)
	}
	return db
}

func classeEtRole(t *testing.T, db *sql.DB, slug, key string) (string, string) {
	t.Helper()
	var class, role string
	if err := db.QueryRow(
		`SELECT class, role FROM weapons WHERE title_slug = ? AND weapon_key = ?`, slug, key,
	).Scan(&class, &role); err != nil {
		t.Fatalf("lecture %s/%s: %v", slug, key, err)
	}
	return class, role
}

// TestReclassEpeeEtMarteau : les deux clés passent en heavy/power, et RIEN d'autre ne bouge
// — ni les autres armes de Halo Infinite, ni l'épée de Halo 5 (le registre est cross-titre,
// un UPDATE trop large aurait reclassé le second titre au passage).
func TestReclassEpeeEtMarteau(t *testing.T) {
	db := baseRegistreArmes(t)

	if err := reclasserEpeeEtMarteau(db); err != nil {
		t.Fatalf("reclasserEpeeEtMarteau: %v", err)
	}

	for _, key := range []string{"hinf_energy_sword", "hinf_gravity_hammer"} {
		class, role := classeEtRole(t, db, "halo_infinite", key)
		if class != "heavy" || role != "power" {
			t.Errorf("%s = %s/%s, attendu heavy/power — sans ce step le lecteur les écarte "+
				"et leurs 9 741 frags retombent dans « Non attribué »", key, class, role)
		}
	}
	if class, role := classeEtRole(t, db, "halo_infinite", "hinf_s7_sniper"); class != "heavy" || role != "sniper" {
		t.Errorf("hinf_s7_sniper = %s/%s, attendu heavy/sniper — le step a débordé de sa cible", class, role)
	}
	if class, role := classeEtRole(t, db, "halo_5", "h5_energy_sword"); class != "melee" || role != "melee" {
		t.Errorf("h5_energy_sword = %s/%s, attendu melee/melee — le registre est CROSS-TITRE, "+
			"le step ne doit toucher que Halo Infinite", class, role)
	}
}

// TestReclassEpeeEtMarteau_Idempotent : rejoué, il réécrit les mêmes valeurs. Et sur une
// base SANS table `weapons` (metadata vierge, le seed n'a pas encore tourné) il ne fait
// rien plutôt que d'échouer — le seed y posera directement les bonnes valeurs.
func TestReclassEpeeEtMarteau_Idempotent(t *testing.T) {
	db := baseRegistreArmes(t)
	for i := 0; i < 2; i++ {
		if err := reclasserEpeeEtMarteau(db); err != nil {
			t.Fatalf("passe %d: %v", i+1, err)
		}
	}
	if class, _ := classeEtRole(t, db, "halo_infinite", "hinf_energy_sword"); class != "heavy" {
		t.Errorf("class = %q apres deux passes, attendu heavy", class)
	}

	vierge, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = vierge.Close() })
	if err := reclasserEpeeEtMarteau(vierge); err != nil {
		t.Errorf("base sans table weapons: %v — le step doit degrader, pas echouer", err)
	}
}
