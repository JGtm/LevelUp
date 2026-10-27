package duckdb

// weapon_names_read_test.go — lecture des noms d'armes par langue (plan i18n 8.4 et 8.7) pour le
// tiroir d'actifs : ListWeaponsByTitle (onglet Assets) et le chargeur de boot des titres
// additionnels. Les noms sont des lignes de weapon_label_names (une par arme et par langue) ; la
// langue servie se choisit par Text.Pick : langue demandée, puis `en`.

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/i18n"
)

// seedWeaponNamesDrawer : trois armes. 100 en fr et en, 200 en `en` seul, 300 en `fr` seul.
func seedWeaponNamesDrawer(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()
	if _, err := db.Exec(ctx, `CREATE TABLE weapon_labels (weapon_id UBIGINT PRIMARY KEY, icon_url VARCHAR)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO weapon_labels VALUES (100, 'https://cdn.example/br.png'), (200, NULL), (300, NULL)`); err != nil {
		t.Fatal(err)
	}
	createWeaponNameTables(t, db)
	putWeaponLabelNames(t, db, 100, frEn("Fusil BR75", "BR75 Battle Rifle"))
	putWeaponLabelNames(t, db, 200, i18n.Text{i18n.LocaleEN: "Skewer"})
	putWeaponLabelNames(t, db, 300, i18n.Text{i18n.LocaleFR: "Brochette"})
}

func weaponByID(t *testing.T, list []canonical.AssetMeta, id string) canonical.AssetMeta {
	t.Helper()
	for _, w := range list {
		if w.ID == id {
			return w
		}
	}
	t.Fatalf("arme %s absente de %+v", id, list)
	return canonical.AssetMeta{}
}

func TestListWeaponsByTitle_NomsParLangueEtRepli(t *testing.T) {
	db := openAssetMemDB(t)
	seedWeaponNamesDrawer(t, db)
	weapons, err := NewMetadataRepoFromDB(db).ListWeaponsByTitle(context.Background(), "halo_infinite", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(weapons) != 3 {
		t.Fatalf("%d armes, want 3 : %+v", len(weapons), weapons)
	}
	for _, c := range []struct {
		id, fr, en, inconnue string
	}{
		{"100", "Fusil BR75", "BR75 Battle Rifle", "BR75 Battle Rifle"},
		{"200", "Skewer", "Skewer", "Skewer"}, // `en` seul : repli sur `en` pour fr et pour une langue inconnue
		{"300", "Brochette", "", ""},          // `fr` seul : pas de repli croisé vers le français
	} {
		w := weaponByID(t, weapons, c.id)
		if got := w.Names.Pick(i18n.LocaleFR); got != c.fr {
			t.Errorf("arme %s fr = %q, want %q", c.id, got, c.fr)
		}
		if got := w.Names.Pick(i18n.LocaleEN); got != c.en {
			t.Errorf("arme %s en = %q, want %q", c.id, got, c.en)
		}
		if got := w.Names.Pick(i18n.Locale("zz")); got != c.inconnue {
			t.Errorf("arme %s langue inconnue = %q, want %q (repli en)", c.id, got, c.inconnue)
		}
	}
}

func TestListWeaponsByTitle_RechercheDansToutesLesLangues(t *testing.T) {
	db := openAssetMemDB(t)
	seedWeaponNamesDrawer(t, db)
	repo := NewMetadataRepoFromDB(db)
	for _, c := range []struct{ search, wantID string }{
		{"brochette", "300"}, // nom français
		{"SKEWER", "200"},    // nom anglais, casse ignorée
		{"fusil br75", "100"},
	} {
		got, err := repo.ListWeaponsByTitle(context.Background(), "halo_infinite", c.search)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].ID != c.wantID {
			t.Errorf("recherche %q = %+v, want la seule arme %s", c.search, got, c.wantID)
		}
		// L'arme trouvée par un nom garde TOUTES ses langues.
		if c.wantID == "100" && (got[0].Names[i18n.LocaleEN] != "BR75 Battle Rifle" || got[0].Names[i18n.LocaleFR] != "Fusil BR75") {
			t.Errorf("langues perdues par la recherche : %v", got[0].Names)
		}
	}
}

func TestLoadTitleAssetDrawerData_ArmesParLangueEtRepli(t *testing.T) {
	db := openAssetMemDB(t)
	seedWeaponNamesDrawer(t, db)
	_, weapons, _ := LoadTitleAssetDrawerData(context.Background(), db, "halo_5")
	// 300 n'a pas de nom anglais : omise (comme une médaille sans nom anglais).
	if len(weapons) != 2 {
		t.Fatalf("%d armes, want 2 (100 et 200) : %+v", len(weapons), weapons)
	}
	br := weaponByID(t, weapons, "100")
	if br.ImageURL != "https://cdn.example/br.png" {
		t.Errorf("icône = %q", br.ImageURL)
	}
	if got := br.Names.Pick(i18n.LocaleFR); got != "Fusil BR75" {
		t.Errorf("fr = %q", got)
	}
	if got := br.Names.Pick(i18n.LocaleEN); got != "BR75 Battle Rifle" {
		t.Errorf("en = %q", got)
	}
	sk := weaponByID(t, weapons, "200")
	if got := sk.Names.Pick(i18n.LocaleFR); got != "Skewer" {
		t.Errorf("200 fr = %q, want Skewer (repli en)", got)
	}
	if got := sk.Names.Pick(i18n.Locale("zz")); got != "Skewer" {
		t.Errorf("200 langue inconnue = %q, want Skewer (repli en)", got)
	}
	if weapons[0].Names[i18n.LocaleEN] > weapons[1].Names[i18n.LocaleEN] {
		t.Errorf("tri attendu par nom anglais : %q avant %q", weapons[0].Names[i18n.LocaleEN], weapons[1].Names[i18n.LocaleEN])
	}
}
