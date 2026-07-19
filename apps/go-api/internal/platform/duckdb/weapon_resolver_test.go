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

// TestResolveWeaponMeta_RegistryFRFallbackH5 fige la politique « FR d'abord (label
// puis registre), EN en dernier repli » (décision user 2026-07-19, ordre
// wl.name_fr > w.name_fr > wl.name_en > w.name) :
//
//	(a) parité Infinite : BR75, dont le weapon_labels porte déjà un name_fr, résout
//	    le MÊME nom qu'avant (« BR75 ») — le registre n'intervient jamais ;
//	(b) repli H5, label ABSENT : une arme absente de weapon_labels récupère le nom FR
//	    du registre (w.name_fr) au lieu de rester vide ;
//	(c) repli H5, label EN-ONLY : une arme dont le weapon_labels porte un name_en NON
//	    VIDE mais un name_fr vide (cas RÉEL H5 : « lightrifle », « FRAG GRENADE ») doit
//	    résoudre le nom FR du REGISTRE, PAS le label EN. C'est précisément le bug que
//	    l'ancien ordre (wl.name_en avant w.name_fr) laissait passer : sans ligne label,
//	    (b) réussissait déjà même avec l'ancien ordre → il ne captait pas la régression.
func TestResolveWeaponMeta_RegistryFRFallbackH5(t *testing.T) {
	meta := resolverTestMeta(t, true)
	ctx := context.Background()

	// (a) Parité Infinite : BR75 (name_fr de label présent) → byte-inchangé.
	brID := int64(0x2b1824d542c9679f)
	if br := resolveWeaponMeta(ctx, meta, "halo_infinite", []int64{brID})[brID]; br.label != "BR75" {
		t.Errorf("BR75 label = %q, want \"BR75\" (parité, registre non consulté)", br.label)
	}

	// (b) Repli H5, label absent : h5_light_rifle (id 2511447508) n'a PAS de ligne
	// weapon_labels → wl.name_fr/wl.name_en NULL → COALESCE tombe sur w.name_fr.
	const lightRifleID = int64(2511447508)
	if lr := resolveWeaponMeta(ctx, meta, "halo_5", []int64{lightRifleID})[lightRifleID]; lr.label != "Fusil léger" {
		t.Errorf("h5 light rifle label = %q, want \"Fusil léger\" (repli registre FR)", lr.label)
	}

	// (c) Repli H5, label EN-only : on seede une ligne weapon_labels avec name_en NON
	// VIDE et name_fr VIDE (« FRAG GRENADE », comme en prod H5) pour l'id frag grenade
	// (stock_id 4106030681 → h5_frag_grenade, w.name_fr = « Grenade à fragmentation »).
	// Sous l'ANCIEN ordre, wl.name_en (« FRAG GRENADE ») gagnait → le registre FR n'était
	// jamais atteint. Le nouvel ordre place w.name_fr AVANT wl.name_en.
	const fragGrenadeID = int64(4106030681)
	if _, err := meta.Exec(ctx,
		"INSERT INTO weapon_labels VALUES ("+strconv.FormatInt(fragGrenadeID, 10)+", 'FRAG GRENADE', '')"); err != nil {
		t.Fatalf("seed h5 frag grenade EN-only label: %v", err)
	}
	if fg := resolveWeaponMeta(ctx, meta, "halo_5", []int64{fragGrenadeID})[fragGrenadeID]; fg.label != "Grenade à fragmentation" {
		t.Errorf("h5 frag grenade label = %q, want \"Grenade à fragmentation\" (registre FR AVANT label EN)", fg.label)
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
