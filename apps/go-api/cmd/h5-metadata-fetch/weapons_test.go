package main

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games/weapons"
	"levelup/go-api/internal/i18n"
)

// openWeaponsDB ouvre une metadata Halo 5 minimale : weapon_labels (avec icon_url et weapon_type,
// ajoutés par h5_weapon_labels_add_icon) et weapon_label_names.
func openWeaponsDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if err := weapons.EnsureLabelsTable(db); err != nil {
		t.Fatalf("EnsureLabelsTable: %v", err)
	}
	if _, err := db.Exec(`ALTER TABLE weapon_labels ADD COLUMN icon_url VARCHAR`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`ALTER TABLE weapon_labels ADD COLUMN weapon_type VARCHAR`); err != nil {
		t.Fatal(err)
	}
	return db
}

func nameFor(t *testing.T, db *sql.DB, id uint64, lang string) (string, bool) {
	t.Helper()
	var s string
	err := db.QueryRow(`SELECT name FROM weapon_label_names WHERE weapon_id = ? AND lang = ?`, id, lang).Scan(&s)
	if err == sql.ErrNoRows {
		return "", false
	}
	if err != nil {
		t.Fatalf("lecture du nom %d/%s : %v", id, lang, err)
	}
	return s, true
}

// storeWeapon écrit icône et type dans weapon_labels et UN nom par langue renseignée ; une langue
// sans nom n'écrit rien (jamais de nom anglais recopié dans le français) ; rejoué, il ne duplique
// rien et ne touche pas une langue posée ailleurs.
func TestStoreWeapon_UnNomParLangueRenseignee(t *testing.T) {
	db := openWeaponsDB(t)
	ctx := context.Background()
	w := apiWeapon{ID: "2511447508", Name: "Light Rifle", Type: "Rifle", LargeIconImageURL: "https://cdn.example/lr.png"}
	names := i18n.Text{i18n.Fallback: "Light Rifle", i18n.LocaleFR: "Fusil léger"}

	written, err := storeWeapon(ctx, db, 2511447508, w, names)
	if err != nil || written != 2 {
		t.Fatalf("storeWeapon : written=%d err=%v, want 2 nil", written, err)
	}
	if got, ok := nameFor(t, db, 2511447508, "fr"); !ok || got != "Fusil léger" {
		t.Errorf("fr = %q (%v)", got, ok)
	}
	if got, ok := nameFor(t, db, 2511447508, "en"); !ok || got != "Light Rifle" {
		t.Errorf("en = %q (%v)", got, ok)
	}
	var icon, typ string
	if err := db.QueryRow(`SELECT icon_url, weapon_type FROM weapon_labels WHERE weapon_id = 2511447508`).Scan(&icon, &typ); err != nil {
		t.Fatal(err)
	}
	if icon != "https://cdn.example/lr.png" || typ != "Rifle" {
		t.Errorf("icône/type = %q / %q", icon, typ)
	}

	// Une langue posée ailleurs (hors passe) survit ; rejouer ne change rien ; l'icône se met à jour.
	if _, err := db.Exec(`INSERT INTO weapon_label_names VALUES (2511447508, 'es', 'Fusil ligero')`); err != nil {
		t.Fatal(err)
	}
	w.LargeIconImageURL = "https://cdn.example/lr2.png"
	written, err = storeWeapon(ctx, db, 2511447508, w, names)
	if err != nil || written != 0 {
		t.Fatalf("rejeu : written=%d err=%v, want 0 nil", written, err)
	}
	if got, ok := nameFor(t, db, 2511447508, "es"); !ok || got != "Fusil ligero" {
		t.Errorf("langue posée ailleurs : %q (%v)", got, ok)
	}
	if err := db.QueryRow(`SELECT icon_url FROM weapon_labels WHERE weapon_id = 2511447508`).Scan(&icon); err != nil || icon != "https://cdn.example/lr2.png" {
		t.Errorf("icône non mise à jour : %q (%v)", icon, err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM weapon_labels`).Scan(&n); err != nil || n != 1 {
		t.Errorf("weapon_labels = %d lignes (%v), want 1", n, err)
	}
}

func TestStoreWeapon_LangueSansNomNEcritRien(t *testing.T) {
	db := openWeaponsDB(t)
	w := apiWeapon{ID: "42", Name: "Magnum", Type: "Pistol"}
	written, err := storeWeapon(context.Background(), db, 42, w, i18n.Text{i18n.Fallback: "Magnum", i18n.LocaleFR: ""})
	if err != nil || written != 1 {
		t.Fatalf("written=%d err=%v, want 1 nil", written, err)
	}
	if _, ok := nameFor(t, db, 42, "fr"); ok {
		t.Error("un nom français vide ne doit pas être écrit (le repli est l'anglais, pas un nom recopié)")
	}
}
