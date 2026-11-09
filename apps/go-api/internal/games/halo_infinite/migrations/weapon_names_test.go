package migrations

// weapon_names_test.go — noms d'armes par langue (ADR 0040 D-5, plan i18n 8.4) sur la metadata
// Halo Infinite : une base neuve naît dans la forme finale avec le seed en deux langues, une base
// à l'ancienne forme est portée sans perte, rejouer ne change rien.

import (
	"testing"

	"levelup/go-api/internal/games/weapons"
)

func TestWeaponNames_BaseNeuveEtRejeu(t *testing.T) {
	db := ouvrirMemoire(t)
	jouerMetadata(t, db)
	count := func(q string) int {
		t.Helper()
		var n int
		if err := db.QueryRow(q).Scan(&n); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		return n
	}
	ids := count(`SELECT COUNT(*) FROM weapon_labels`)
	if ids == 0 {
		t.Fatal("weapon_labels vide : le seed Infinite n'a pas été posé")
	}
	if got := count(`SELECT COUNT(*) FROM weapon_label_names`); got != 2*ids {
		t.Errorf("weapon_label_names = %d, want %d (en + fr par arme)", got, 2*ids)
	}
	for _, col := range []string{"name_en", "name_fr"} {
		if n := count(`SELECT COUNT(*) FROM information_schema.columns WHERE table_name='weapon_labels' AND column_name='` + col + `'`); n != 0 {
			t.Errorf("weapon_labels.%s devrait être absente", col)
		}
	}
	jouerMetadata(t, db)
	if got := count(`SELECT COUNT(*) FROM weapon_label_names`); got != 2*ids {
		t.Errorf("après rejeu : %d lignes, want %d", got, 2*ids)
	}
	// Le reseed du CLI (cmd/seed-weapon-labels) reste idempotent sur la forme finale.
	if err := weapons.ApplyLabels(db); err != nil {
		t.Fatalf("ApplyLabels: %v", err)
	}
	if got := count(`SELECT COUNT(*) FROM weapon_label_names`); got != 2*ids {
		t.Errorf("après reseed : %d lignes, want %d", got, 2*ids)
	}
}
