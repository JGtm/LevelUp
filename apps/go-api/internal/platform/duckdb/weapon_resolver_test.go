//go:build integration

// weapon_resolver_test.go — vérifie le PASSAGE PRINCIPAL P4 (resolveWeaponMeta) :
//   - PARITÉ : le nom reste celui de weapon_labels (« BR75 », PAS « Fusil de combat »).
//   - DIMENSIONS : role/family/faction/weapon_key viennent du registre.
//   - sentinel grenade (0) : nom weapon_labels, role vide.
//   - id inconnu : label vide (filtré en aval).
//   - fallback silencieux quand le registre est absent (parité conservée).

package duckdb

import (
	"context"
	"strconv"
	"testing"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
)

func resolverTestMeta(t *testing.T, withRegistry bool) *DB {
	t.Helper()
	meta := openMemDB(t)
	ctx := context.Background()
	if _, err := meta.Exec(ctx, `CREATE TABLE weapon_labels (weapon_id UBIGINT PRIMARY KEY, name_en VARCHAR, name_fr VARCHAR)`); err != nil {
		t.Fatalf("create weapon_labels: %v", err)
	}
	brID := uint64(0x2b1824d542c9679f) // BR75 filmshell
	if _, err := meta.Exec(ctx, "INSERT INTO weapon_labels VALUES ("+strconv.FormatUint(brID, 10)+", 'BR75', 'BR75')"); err != nil {
		t.Fatalf("seed BR75: %v", err)
	}
	if _, err := meta.Exec(ctx, "INSERT INTO weapon_labels VALUES (0, 'Grenade', 'Grenade')"); err != nil {
		t.Fatalf("seed sentinel: %v", err)
	}
	if withRegistry {
		if err := halomigrations.ApplyWeaponRegistry(meta.SQLDb()); err != nil {
			t.Fatalf("ApplyWeaponRegistry: %v", err)
		}
	}
	return meta
}

func TestResolveWeaponMeta_ParityAndDims(t *testing.T) {
	meta := resolverTestMeta(t, true)
	brID := int64(0x2b1824d542c9679f)
	res := resolveWeaponMeta(context.Background(), meta, "halo_infinite", []int64{brID, 0, 999999})

	br := res[brID]
	if br.label != "BR75" { // PARITÉ : nom = weapon_labels, surtout PAS "Fusil de combat"
		t.Errorf("BR75 label = %q, want \"BR75\" (parité)", br.label)
	}
	if br.role != "precision" || br.family != "battle_rifle" || br.faction != "human" || br.weaponKey != "hinf_br75" {
		t.Errorf("BR75 dims = %+v, want precision/battle_rifle/human/hinf_br75", br)
	}
	if s := res[0]; s.label != "Grenade" || s.role != "" {
		t.Errorf("sentinel 0 = %+v, want label=Grenade role=''", s)
	}
	if u := res[999999]; u.label != "" {
		t.Errorf("id inconnu label = %q, want vide", u.label)
	}
}

func TestResolveWeaponMeta_FallbackWhenNoRegistry(t *testing.T) {
	meta := resolverTestMeta(t, false) // registre absent
	brID := int64(0x2b1824d542c9679f)
	res := resolveWeaponMeta(context.Background(), meta, "halo_infinite", []int64{brID, 0})

	if br := res[brID]; br.label != "BR75" || br.role != "" {
		t.Errorf("BR75 sans registre = %+v, want label=BR75 role='' (parité, dims vides)", br)
	}
	if s := res[0]; s.label != "Grenade" {
		t.Errorf("sentinel sans registre = %+v, want label=Grenade", s)
	}
}
