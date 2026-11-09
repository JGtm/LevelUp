//go:build cgo

// name_labels_test.go — écritures des noms d'armes par langue (plan i18n 8.4) : réconciliation du
// TOML dans weapon_name_labels (une ligne par langue, SELECT puis UPDATE ou INSERT), seed
// ApplyLabels dans weapon_label_names, SetLabelName du fetch Halo 5.

package weapons

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/i18n"
)

func openWeaponsMem(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func writeNamesTOML(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "weapon_names.toml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func nameOf(t *testing.T, db *sql.DB, q string, args ...any) string {
	t.Helper()
	var s string
	if err := db.QueryRow(q, args...).Scan(&s); err != nil {
		t.Fatalf("%s %v: %v", q, args, err)
	}
	return s
}

func countOf(t *testing.T, db *sql.DB, q string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatalf("%s %v: %v", q, args, err)
	}
	return n
}

const twoWeaponsTOML = `[meta]
title_slug = "halo_infinite"
schema_version = 1

[weapons]
hinf_br75 = { en = "BR75", fr = "Fusil BR75" }
hinf_sword = { en = "Energy Sword", fr = "Épée à énergie" }
`

func TestReconcileNameLabels_UneLigneParLangue(t *testing.T) {
	db := openWeaponsMem(t)
	path := writeNamesTOML(t, twoWeaponsTOML)
	n, err := ReconcileNameLabels(db, "halo_infinite", path)
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 {
		t.Errorf("lignes du fichier = %d, want 4 (2 armes x 2 langues)", n)
	}
	if got := nameOf(t, db, `SELECT name FROM weapon_name_labels WHERE title_slug='halo_infinite' AND weapon_key='hinf_br75' AND lang='fr'`); got != "Fusil BR75" {
		t.Errorf("br75 fr = %q", got)
	}
	if got := nameOf(t, db, `SELECT name FROM weapon_name_labels WHERE title_slug='halo_infinite' AND weapon_key='hinf_sword' AND lang='en'`); got != "Energy Sword" {
		t.Errorf("sword en = %q", got)
	}
	if c := countOf(t, db, `SELECT COUNT(*) FROM weapon_name_labels`); c != 4 {
		t.Errorf("weapon_name_labels = %d lignes, want 4", c)
	}
}

func TestReconcileNameLabels_IdempotenteEtCorrigeSansEffacerUneAutreLangue(t *testing.T) {
	db := openWeaponsMem(t)
	path := writeNamesTOML(t, twoWeaponsTOML)
	if _, err := ReconcileNameLabels(db, "halo_infinite", path); err != nil {
		t.Fatal(err)
	}
	// Une langue posée hors du fichier, et une ligne d'un autre titre : jamais touchées.
	if _, err := db.Exec(`INSERT INTO weapon_name_labels VALUES ('halo_infinite', 'hinf_br75', 'es', 'Fusil BR75 (es)'), ('halo_5', 'h5_x', 'en', 'X')`); err != nil {
		t.Fatal(err)
	}
	// 2e passage identique : rien ne change.
	if _, err := ReconcileNameLabels(db, "halo_infinite", path); err != nil {
		t.Fatal(err)
	}
	if c := countOf(t, db, `SELECT COUNT(*) FROM weapon_name_labels`); c != 6 {
		t.Errorf("après 2 passages : %d lignes, want 6", c)
	}
	// Une correction du fichier se propage (UPDATE), la langue posée ailleurs survit.
	corrected := writeNamesTOML(t, `[meta]
title_slug = "halo_infinite"
schema_version = 1

[weapons]
hinf_br75 = { en = "BR75 Battle Rifle", fr = "Fusil BR75" }
hinf_sword = { en = "Energy Sword", fr = "Épée à énergie" }
`)
	if _, err := ReconcileNameLabels(db, "halo_infinite", corrected); err != nil {
		t.Fatal(err)
	}
	if got := nameOf(t, db, `SELECT name FROM weapon_name_labels WHERE weapon_key='hinf_br75' AND lang='en'`); got != "BR75 Battle Rifle" {
		t.Errorf("correction non propagée : %q", got)
	}
	if got := nameOf(t, db, `SELECT name FROM weapon_name_labels WHERE weapon_key='hinf_br75' AND lang='es'`); got != "Fusil BR75 (es)" {
		t.Errorf("langue posée hors fichier effacée : %q", got)
	}
	if c := countOf(t, db, `SELECT COUNT(*) FROM weapon_name_labels`); c != 6 {
		t.Errorf("après correction : %d lignes, want 6", c)
	}
}

func TestReconcileNameLabels_FichierAbsent_TableCreee(t *testing.T) {
	db := openWeaponsMem(t)
	n, err := ReconcileNameLabels(db, "halo_infinite", filepath.Join(t.TempDir(), "absent.toml"))
	if err != nil || n != 0 {
		t.Fatalf("fichier absent : n=%d err=%v, want 0 nil", n, err)
	}
	if c := countOf(t, db, `SELECT COUNT(*) FROM weapon_name_labels`); c != 0 {
		t.Errorf("table vide attendue, %d lignes", c)
	}
}

func TestReconcileNameLabels_PorteUneBaseALAncienneForme(t *testing.T) {
	db := openWeaponsMem(t)
	if _, err := db.Exec(`CREATE TABLE weapon_name_labels (title_slug VARCHAR NOT NULL, weapon_key VARCHAR NOT NULL,
		name_en VARCHAR NOT NULL, name_fr VARCHAR NOT NULL, PRIMARY KEY (title_slug, weapon_key));
		INSERT INTO weapon_name_labels VALUES ('halo_5', 'h5_old', 'Old', 'Ancien')`); err != nil {
		t.Fatal(err)
	}
	if _, err := ReconcileNameLabels(db, "halo_infinite", writeNamesTOML(t, twoWeaponsTOML)); err != nil {
		t.Fatal(err)
	}
	if got := nameOf(t, db, `SELECT name FROM weapon_name_labels WHERE weapon_key='h5_old' AND lang='fr'`); got != "Ancien" {
		t.Errorf("ligne héritée perdue : %q", got)
	}
	if c := countOf(t, db, `SELECT COUNT(*) FROM weapon_name_labels`); c != 6 {
		t.Errorf("%d lignes, want 6 (2 héritées + 4 du fichier)", c)
	}
}

func TestApplyLabels_NomsParLangueSansEcraser(t *testing.T) {
	db := openWeaponsMem(t)
	if err := ApplyLabels(db); err != nil {
		t.Fatal(err)
	}
	seeds := len(weaponLabelSeeds())
	if c := countOf(t, db, `SELECT COUNT(*) FROM weapon_labels`); c != seeds {
		t.Errorf("weapon_labels = %d, want %d", c, seeds)
	}
	if c := countOf(t, db, `SELECT COUNT(*) FROM weapon_label_names`); c != 2*seeds {
		t.Errorf("weapon_label_names = %d, want %d (en + fr)", c, 2*seeds)
	}
	for _, col := range []string{"name_en", "name_fr"} {
		if c := countOf(t, db, `SELECT COUNT(*) FROM information_schema.columns WHERE table_name='weapon_labels' AND column_name=?`, col); c != 0 {
			t.Errorf("weapon_labels.%s ne devrait pas exister", col)
		}
	}
	if got := nameOf(t, db, `SELECT name FROM weapon_label_names WHERE weapon_id = 1 AND lang = 'fr'`); got != "Corps à corps" {
		t.Errorf("melee fr = %q", got)
	}
	// Un nom posé ailleurs n'est pas écrasé par un re-seed, et le re-seed est idempotent.
	if _, err := db.Exec(`UPDATE weapon_label_names SET name = 'API melee' WHERE weapon_id = 1 AND lang = 'en'`); err != nil {
		t.Fatal(err)
	}
	if err := ApplyLabels(db); err != nil {
		t.Fatal(err)
	}
	if got := nameOf(t, db, `SELECT name FROM weapon_label_names WHERE weapon_id = 1 AND lang = 'en'`); got != "API melee" {
		t.Errorf("nom écrasé par le re-seed : %q", got)
	}
	if c := countOf(t, db, `SELECT COUNT(*) FROM weapon_label_names`); c != 2*seeds {
		t.Errorf("après re-seed : %d lignes, want %d", c, 2*seeds)
	}
}

func TestApplyLabels_PorteUneBaseALAncienneForme(t *testing.T) {
	db := openWeaponsMem(t)
	if _, err := db.Exec(`CREATE TABLE weapon_labels (weapon_id UBIGINT PRIMARY KEY, name_en VARCHAR NOT NULL, name_fr VARCHAR NOT NULL);
		INSERT INTO weapon_labels VALUES (777, 'Prototype', 'Prototype FR')`); err != nil {
		t.Fatal(err)
	}
	if err := ApplyLabels(db); err != nil {
		t.Fatal(err)
	}
	if got := nameOf(t, db, `SELECT name FROM weapon_label_names WHERE weapon_id = 777 AND lang = 'fr'`); got != "Prototype FR" {
		t.Errorf("nom hérité perdu : %q", got)
	}
	if c := countOf(t, db, `SELECT COUNT(*) FROM weapon_labels`); c != len(weaponLabelSeeds())+1 {
		t.Errorf("weapon_labels = %d lignes", c)
	}
}

func TestSetLabelName(t *testing.T) {
	db := openWeaponsMem(t)
	if err := EnsureLabelsTable(db); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	const bit63 = uint64(0xf5c335dfe7232c0f)
	steps := []struct {
		lang    i18n.Locale
		name    string
		changed bool
	}{
		{i18n.LocaleEN, " MA5K ", true},  // INSERT, espaces de bord retirés
		{i18n.LocaleEN, "MA5K", false},   // identique : aucune écriture
		{i18n.LocaleEN, "MA5K AR", true}, // UPDATE
		{i18n.LocaleFR, "", false},       // nom vide : ignoré
		{i18n.LocaleFR, "MA5K FR", true}, // autre langue : INSERT, l'anglais reste
	}
	for i, s := range steps {
		changed, err := SetLabelName(ctx, db, bit63, s.lang, s.name)
		if err != nil {
			t.Fatalf("étape %d : %v", i, err)
		}
		if changed != s.changed {
			t.Errorf("étape %d (%s %q) : changed=%v, want %v", i, s.lang, s.name, changed, s.changed)
		}
	}
	q := fmt.Sprintf(`SELECT name FROM weapon_label_names WHERE weapon_id = %d AND lang = ?`, bit63)
	if got := nameOf(t, db, q, "en"); got != "MA5K AR" {
		t.Errorf("en = %q", got)
	}
	if got := nameOf(t, db, q, "fr"); got != "MA5K FR" {
		t.Errorf("fr = %q", got)
	}
	if c := countOf(t, db, `SELECT COUNT(*) FROM weapon_label_names WHERE name = ''`); c != 0 {
		t.Errorf("%d nom(s) vide(s) écrit(s)", c)
	}
}
