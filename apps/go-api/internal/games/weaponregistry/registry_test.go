//go:build cgo

// registry_test.go — seede le référentiel (migration add_weapon_registry) sur
// DuckDB :memory:, le charge via LoadFromDB, puis vérifie ByKey/ByID/Family +
// la dégradation gracieuse (id/clé inconnu → false).

package weaponregistry

import (
	"context"
	"database/sql"
	"strconv"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games/halo_infinite/migrations"
)

func loadTestRegistry(t *testing.T) *MemRegistry {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrations.ApplyWeaponRegistry(db); err != nil {
		t.Fatalf("seed: %v", err)
	}
	reg, err := LoadFromDB(context.Background(), db)
	if err != nil {
		t.Fatalf("LoadFromDB: %v", err)
	}
	return reg
}

func TestRegistry_Counts(t *testing.T) {
	reg := loadTestRegistry(t)
	w, i, f := reg.Counts()
	if w != 84 || i != 102 || f != 51 {
		t.Errorf("Counts = (%d,%d,%d), want (84,102,51)", w, i, f)
	}
}

func TestRegistry_ByKey(t *testing.T) {
	reg := loadTestRegistry(t)
	w, ok := reg.ByKey("hinf_br75")
	if !ok {
		t.Fatal("hinf_br75 introuvable")
	}
	if w.Name != "BR75" || w.Class != "shoulder" || w.Role != "precision" ||
		w.FamilyKey != "battle_rifle" || w.Faction != "human" {
		t.Errorf("BR75 mal résolu: %+v", w)
	}
	if _, ok := reg.ByKey("nope"); ok {
		t.Error("clé inconnue devrait être false")
	}
}

func TestRegistry_ByID_Filmshell(t *testing.T) {
	reg := loadTestRegistry(t)
	// 0x2b1824d542c9679f = id filmshell du BR75 (cf. weapon_registry.go).
	brFilmshell := strconv.FormatUint(0x2b1824d542c9679f, 10)
	w, ok := reg.ByID("halo_infinite", "filmshell", brFilmshell)
	if !ok || w.Key != "hinf_br75" {
		t.Errorf("ByID filmshell BR75 = (%+v, %v), want hinf_br75", w, ok)
	}
	// Une variante (Energy Sword Duelist) résout vers la même arme.
	swordDuelist := strconv.FormatUint(0x4ff3937e8978aa7a, 10)
	if w, ok := reg.ByID("halo_infinite", "filmshell", swordDuelist); !ok || w.Key != "hinf_energy_sword" {
		t.Errorf("ByID variante épée = (%+v, %v), want hinf_energy_sword", w, ok)
	}
	// id inconnu → false (dégradation).
	if _, ok := reg.ByID("halo_infinite", "filmshell", "999999999"); ok {
		t.Error("id filmshell inconnu devrait être false")
	}
	// H5 stock_id résout (P2bis) : 907086443 = Retro Beam Rifle (variante) → h5_beam_rifle.
	if w, ok := reg.ByID("halo_5", "stock_id", "907086443"); !ok || w.Key != "h5_beam_rifle" {
		t.Errorf("ByID stock_id 907086443 = (%+v, %v), want h5_beam_rifle", w, ok)
	}
	// SAW 2278207101 → h5_saw (role automatic).
	if w, ok := reg.ByID("halo_5", "stock_id", "2278207101"); !ok || w.Key != "h5_saw" || w.Role != "automatic" {
		t.Errorf("ByID stock_id SAW = (%+v, %v), want h5_saw/automatic", w, ok)
	}
}

func TestRegistry_Family(t *testing.T) {
	reg := loadTestRegistry(t)
	f, ok := reg.Family("battle_rifle")
	if !ok || f.NameFR != "Fusil de combat" || f.NameEN != "Battle Rifle" {
		t.Errorf("Family battle_rifle = (%+v, %v)", f, ok)
	}
	if _, ok := reg.Family("inexistante"); ok {
		t.Error("famille inconnue devrait être false")
	}
}
