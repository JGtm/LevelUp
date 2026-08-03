//go:build integration

// weapon_resolver_test.go — vérifie le PASSAGE PRINCIPAL P4 (resolveWeaponMeta) :
//   - SOURCE UNIQUE (V72-06) : le nom vient de weapon_name_labels, keyé par weapon_key
//     (weapon_id → weapon_ids → weapon_key → {en,fr}). BR75 = « BR75 » (parité).
//   - DIMENSIONS : role/family/faction/weapon_key viennent du registre (plus de nom).
//   - sentinel grenade (0) : nom weapon_labels (pas de weapon_key), role vide.
//   - id inconnu : label vide (filtré en aval).
//   - fallback silencieux quand le registre/la source de noms sont absents.

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
		// V72-06 : SOURCE UNIQUE des noms keyée par weapon_key. Le resolver joint
		// weapon_name_labels (weapon_id → weapon_ids → weapon_key → {en,fr}). On seede
		// les 3 clés exercées par les tests ; les autres ids retombent sur weapon_labels
		// (sentinelles) ou "" (inconnus).
		if _, err := meta.Exec(ctx, `CREATE TABLE weapon_name_labels (
			title_slug VARCHAR, weapon_key VARCHAR, name_en VARCHAR, name_fr VARCHAR,
			PRIMARY KEY (title_slug, weapon_key))`); err != nil {
			t.Fatalf("create weapon_name_labels: %v", err)
		}
		for _, ins := range []string{
			"INSERT INTO weapon_name_labels VALUES ('halo_infinite', 'hinf_br75', 'BR75', 'BR75')",
			"INSERT INTO weapon_name_labels VALUES ('halo_5', 'h5_light_rifle', 'Light Rifle', 'Fusil léger')",
			"INSERT INTO weapon_name_labels VALUES ('halo_5', 'h5_frag_grenade', 'Frag Grenade', 'Grenade à fragmentation')",
		} {
			if _, err := meta.Exec(ctx, ins); err != nil {
				t.Fatalf("seed weapon_name_labels: %v", err)
			}
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

// TestResolveWeaponMeta_WeaponKeyNameSourceH5 fige la SOURCE UNIQUE keyée par
// weapon_key (V72-06, ordre wnl.name_fr > wnl.name_en > wl.name_fr > wl.name_en) :
//
//	(a) parité Infinite : BR75 résout le MÊME nom qu'avant (« BR75 ») via
//	    weapon_name_labels ;
//	(b) H5, aucune ligne weapon_labels : h5_light_rifle (id 2511447508) récupère le nom
//	    depuis weapon_name_labels (weapon_key h5_light_rifle) au lieu de rester vide ;
//	(c) H5, label EN-ONLY POURRI : une arme dont le weapon_labels porte un name_en NON
//	    VIDE (« FRAG GRENADE », casse API) mais un name_fr vide doit résoudre le nom de
//	    weapon_name_labels (« Grenade à fragmentation »), PAS le label EN. C'est le bug
//	    d'origine : l'EN brut de weapon_labels fuyait sur le Match view H5. La source
//	    keyée par weapon_key le prime désormais.
func TestResolveWeaponMeta_WeaponKeyNameSourceH5(t *testing.T) {
	meta := resolverTestMeta(t, true)
	ctx := context.Background()

	// (a) Parité Infinite : BR75 (weapon_name_labels hinf_br75) → byte-inchangé.
	brID := int64(0x2b1824d542c9679f)
	if br := resolveWeaponMeta(ctx, meta, "halo_infinite", []int64{brID})[brID]; br.label != "BR75" {
		t.Errorf("BR75 label = %q, want \"BR75\" (parité, source keyée weapon_key)", br.label)
	}

	// (b) H5, aucune ligne weapon_labels : h5_light_rifle (id 2511447508) → nom résolu
	// via weapon_ids → weapon_key h5_light_rifle → weapon_name_labels (« Fusil léger »).
	const lightRifleID = int64(2511447508)
	if lr := resolveWeaponMeta(ctx, meta, "halo_5", []int64{lightRifleID})[lightRifleID]; lr.label != "Fusil léger" {
		t.Errorf("h5 light rifle label = %q, want \"Fusil léger\" (source weapon_key)", lr.label)
	}

	// (c) H5, label EN-only POURRI : on seede une ligne weapon_labels avec name_en NON
	// VIDE et name_fr VIDE (« FRAG GRENADE », comme en prod H5) pour l'id frag grenade
	// (stock_id 4106030681 → h5_frag_grenade). La source weapon_name_labels
	// (« Grenade à fragmentation ») doit primer l'EN brut de weapon_labels.
	const fragGrenadeID = int64(4106030681)
	if _, err := meta.Exec(ctx,
		"INSERT INTO weapon_labels VALUES ("+strconv.FormatInt(fragGrenadeID, 10)+", 'FRAG GRENADE', '')"); err != nil {
		t.Fatalf("seed h5 frag grenade EN-only label: %v", err)
	}
	if fg := resolveWeaponMeta(ctx, meta, "halo_5", []int64{fragGrenadeID})[fragGrenadeID]; fg.label != "Grenade à fragmentation" {
		t.Errorf("h5 frag grenade label = %q, want \"Grenade à fragmentation\" (weapon_name_labels prime l'EN brut)", fg.label)
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

// TestResolveWeaponMeta_H5VehiclesDistinctPerEngine fige le SOCLE de la ventilation par
// ENGIN (V73-3.2) et documente pourquoi elle ne peut pas passer par le rôle :
//
//	(a) les stock_ids RÉELS des véhicules H5 (relevés en base le 2026-08-02 :
//	    Warthog 4028516791, Ghost 3010146366) résolvent chacun un weapon_key DISTINCT —
//	    ils ne s'écrasent PAS sur le sentinel véhicule (id 2), qui ne porte aucun kill H5 ;
//	(b) leur class vaut « vehicle » pour les deux, tout comme role et family : ventiler le
//	    niveau 2 par rôle donnerait donc un arc unique sans information. Seul weapon_key
//	    distingue un Warthog d'un Ghost — d'où domain.IsPerWeaponFragClass.
func TestResolveWeaponMeta_H5VehiclesDistinctPerEngine(t *testing.T) {
	meta := resolverTestMeta(t, true)
	ctx := context.Background()
	for _, ins := range []string{
		"INSERT INTO weapon_name_labels VALUES ('halo_5', 'h5_vehicle_warthog', 'Warthog', 'Warthog')",
		"INSERT INTO weapon_name_labels VALUES ('halo_5', 'h5_vehicle_ghost', 'Ghost', 'Ghost')",
	} {
		if _, err := meta.Exec(ctx, ins); err != nil {
			t.Fatalf("seed weapon_name_labels véhicules: %v", err)
		}
	}
	const warthogID, ghostID = int64(4028516791), int64(3010146366)
	res := resolveWeaponMeta(ctx, meta, "halo_5", []int64{warthogID, ghostID})

	warthog, ghost := res[warthogID], res[ghostID]
	if warthog.weaponKey != "h5_vehicle_warthog" || ghost.weaponKey != "h5_vehicle_ghost" {
		t.Fatalf("weapon_key par engin = %q / %q, want h5_vehicle_warthog / h5_vehicle_ghost",
			warthog.weaponKey, ghost.weaponKey)
	}
	if warthog.weaponKey == ghost.weaponKey {
		t.Error("deux engins distincts partagent une clé : le sous-niveau ne pourrait pas les séparer")
	}
	if warthog.label != "Warthog" || ghost.label != "Ghost" {
		t.Errorf("libellés registre = %q / %q, want Warthog / Ghost (source weapon_names.toml)",
			warthog.label, ghost.label)
	}
	// (b) class/role/family confondus → le rôle ne distingue aucun engin.
	for name, m := range map[string]weaponResolved{"warthog": warthog, "ghost": ghost} {
		if m.class != "vehicle" {
			t.Errorf("%s class = %q, want vehicle", name, m.class)
		}
		if m.role != m.class {
			t.Errorf("%s : role=%q class=%q — le test suppose role == class (socle de la ventilation par engin)",
				name, m.role, m.class)
		}
	}
}
