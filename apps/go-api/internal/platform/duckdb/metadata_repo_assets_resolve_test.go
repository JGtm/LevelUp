package duckdb

import (
	"context"
	"testing"
)

// setupResolveFixtures crée la table asset_translations + insère un set
// représentatif (map "Shiro" en 14 langs comme prod, map "Catalyst" FR+EN,
// map "Forbidden" EN seul, map "Empty" sans aucune traduction).
func setupResolveFixtures(t *testing.T, db *DB, ctx context.Context) {
	t.Helper()
	if _, err := db.Exec(ctx, `
		CREATE TABLE asset_translations (
			asset_id    VARCHAR NOT NULL,
			asset_type  VARCHAR NOT NULL,
			lang        VARCHAR NOT NULL,
			name        VARCHAR NOT NULL DEFAULT '',
			description VARCHAR NOT NULL DEFAULT '',
			fetched_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (asset_id, asset_type, lang)
		)`); err != nil {
		t.Fatalf("DDL asset_translations: %v", err)
	}
	rows := []string{
		// Shiro : couverture multilingue complète (cas réel prod)
		`INSERT INTO asset_translations VALUES ('shiro-uuid','map','fr-FR','Shiro','',now())`,
		`INSERT INTO asset_translations VALUES ('shiro-uuid','map','en-US','Shiro','',now())`,
		`INSERT INTO asset_translations VALUES ('shiro-uuid','map','de-DE','Shiro','',now())`,
		`INSERT INTO asset_translations VALUES ('shiro-uuid','map','ja-JP','シロ','',now())`,
		// Catalyst : juste FR+EN
		`INSERT INTO asset_translations VALUES ('catalyst-uuid','map','fr-FR','Catalyst','',now())`,
		`INSERT INTO asset_translations VALUES ('catalyst-uuid','map','en-US','Catalyst','',now())`,
		// Forbidden : EN seul (cas où populate-assets n'a fait que en-US)
		`INSERT INTO asset_translations VALUES ('forbidden-uuid','map','en-US','Forbidden','',now())`,
		// Pair sample (asset_type différent — pour tester l'isolation par type)
		`INSERT INTO asset_translations VALUES ('pair-quickplay','pair','fr-FR','Partie rapide : Assassin','',now())`,
		`INSERT INTO asset_translations VALUES ('pair-quickplay','pair','en-US','Quick Play : Slayer','',now())`,
	}
	for _, q := range rows {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("fixture: %v", err)
		}
	}
}

func TestResolveAssetName_FRPreferenceHits(t *testing.T) {
	db := openAssetMemDB(t)
	repo := NewMetadataRepoFromDB(db)
	ctx := context.Background()
	setupResolveFixtures(t, db, ctx)

	name, lang, ok, err := repo.ResolveAssetName(ctx, "map", "shiro-uuid",
		[]string{"fr-FR", "fr", "en-US", "en"})
	if err != nil {
		t.Fatalf("ResolveAssetName: %v", err)
	}
	if !ok {
		t.Fatal("attendu ok=true")
	}
	if name != "Shiro" || lang != "fr-FR" {
		t.Errorf("attendu (Shiro, fr-FR), obtenu (%s, %s)", name, lang)
	}
}

func TestResolveAssetName_FallbackToEN_WhenFRMissing(t *testing.T) {
	db := openAssetMemDB(t)
	repo := NewMetadataRepoFromDB(db)
	ctx := context.Background()
	setupResolveFixtures(t, db, ctx)

	// Forbidden n'a que en-US ; demande locale fr → cascade EN
	name, lang, ok, err := repo.ResolveAssetName(ctx, "map", "forbidden-uuid",
		PreferredLangsForLocale("fr"))
	if err != nil {
		t.Fatalf("ResolveAssetName: %v", err)
	}
	if !ok {
		t.Fatal("attendu ok=true (fallback EN)")
	}
	if name != "Forbidden" || lang != "en-US" {
		t.Errorf("attendu (Forbidden, en-US), obtenu (%s, %s)", name, lang)
	}
}

func TestResolveAssetName_NotFound(t *testing.T) {
	db := openAssetMemDB(t)
	repo := NewMetadataRepoFromDB(db)
	ctx := context.Background()
	setupResolveFixtures(t, db, ctx)

	_, _, ok, err := repo.ResolveAssetName(ctx, "map", "missing-uuid",
		PreferredLangsForLocale("fr"))
	if err != nil {
		t.Fatalf("ResolveAssetName: %v", err)
	}
	if ok {
		t.Error("attendu ok=false pour asset inconnu")
	}
}

func TestResolveAssetName_AssetTypeIsolation(t *testing.T) {
	db := openAssetMemDB(t)
	repo := NewMetadataRepoFromDB(db)
	ctx := context.Background()
	setupResolveFixtures(t, db, ctx)

	// pair-quickplay existe seulement en asset_type='pair'
	_, _, ok, _ := repo.ResolveAssetName(ctx, "map", "pair-quickplay",
		PreferredLangsForLocale("fr"))
	if ok {
		t.Error("attendu ok=false : asset_type='map' ne doit pas matcher un asset_type='pair'")
	}
	name, _, ok, err := repo.ResolveAssetName(ctx, "pair", "pair-quickplay",
		PreferredLangsForLocale("fr"))
	if err != nil {
		t.Fatalf("ResolveAssetName: %v", err)
	}
	if !ok || name != "Partie rapide : Assassin" {
		t.Errorf("attendu (Partie rapide : Assassin), obtenu (%s, ok=%v)", name, ok)
	}
}

func TestResolveAssetName_LocaleENPrefers_EN(t *testing.T) {
	db := openAssetMemDB(t)
	repo := NewMetadataRepoFromDB(db)
	ctx := context.Background()
	setupResolveFixtures(t, db, ctx)

	// Locale "en" → préfère en-US même si fr-FR existe (Shiro)
	name, lang, ok, err := repo.ResolveAssetName(ctx, "map", "shiro-uuid",
		PreferredLangsForLocale("en"))
	if err != nil {
		t.Fatalf("ResolveAssetName: %v", err)
	}
	if !ok {
		t.Fatal("ok=true attendu")
	}
	// fr-FR et en-US ont tous les deux "Shiro" — le test vérifie surtout que la
	// lang choisie est bien en-US (préférence respectée).
	if lang != "en-US" || name != "Shiro" {
		t.Errorf("attendu (Shiro, en-US), obtenu (%s, %s)", name, lang)
	}
}

func TestResolveAssetNamesBulk_MixedAvailability(t *testing.T) {
	db := openAssetMemDB(t)
	repo := NewMetadataRepoFromDB(db)
	ctx := context.Background()
	setupResolveFixtures(t, db, ctx)

	ids := []string{"shiro-uuid", "catalyst-uuid", "forbidden-uuid", "missing-uuid"}
	out, err := repo.ResolveAssetNamesBulk(ctx, "map", ids,
		PreferredLangsForLocale("fr"))
	if err != nil {
		t.Fatalf("ResolveAssetNamesBulk: %v", err)
	}
	expect := map[string]string{
		"shiro-uuid":     "Shiro",     // fr-FR direct
		"catalyst-uuid":  "Catalyst",  // fr-FR direct
		"forbidden-uuid": "Forbidden", // fallback en-US
		// missing-uuid : absent du résultat (aucune traduction)
	}
	for id, want := range expect {
		got, ok := out[id]
		if !ok {
			t.Errorf("%s manquant du résultat", id)
			continue
		}
		if got != want {
			t.Errorf("%s = %q, attendu %q", id, got, want)
		}
	}
	if _, ok := out["missing-uuid"]; ok {
		t.Errorf("missing-uuid ne devrait PAS être dans le résultat")
	}
}

func TestResolveAssetNamesBulk_EmptyInput(t *testing.T) {
	db := openAssetMemDB(t)
	repo := NewMetadataRepoFromDB(db)
	ctx := context.Background()
	setupResolveFixtures(t, db, ctx)

	out, err := repo.ResolveAssetNamesBulk(ctx, "map", nil,
		PreferredLangsForLocale("fr"))
	if err != nil {
		t.Fatalf("ResolveAssetNamesBulk: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("attendu map vide, obtenu %d entrées", len(out))
	}
}

func TestPreferredLangsForLocale(t *testing.T) {
	cases := []struct {
		locale string
		first  string
	}{
		{"fr", "fr-FR"},
		{"fr-FR", "fr-FR"},
		{"FR", "fr-FR"},
		{"en", "en-US"},
		{"en-US", "en-US"},
		{"de", "fr-FR"}, // locale inconnue → défaut FR-first
		{"", "fr-FR"},
	}
	for _, c := range cases {
		got := PreferredLangsForLocale(c.locale)
		if len(got) == 0 || got[0] != c.first {
			t.Errorf("PreferredLangsForLocale(%q): first=%v, attendu %q", c.locale, got, c.first)
		}
	}
}
