//go:build integration

package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/i18n"
)

// seedMedalFallbackData peuple un dataset hétérogène partagé par les tests de
// fallback : 100 résoluble par medal_definitions, 200 SEULEMENT par
// citation_mappings. Réutilise seedMedalDefsSchema (medal_definitions_repo_test.go).
func seedMedalFallbackData(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()
	seedMedalDefsSchema(t, db)
	// Nom anglais seul (aucune ligne française) → label déterministe "Killjoy" quelle
	// que soit la locale : resolveMedalLabels et lookupMedalMeta sont locale-aware
	// mais une langue sans traduction retombe sur l'anglais, même en FR.
	if _, err := db.Exec(ctx,
		`INSERT INTO medal_definitions (medal_name_id, difficulty) VALUES (100, 'Heroic')`,
	); err != nil {
		t.Fatalf("seed medal 100: %v", err)
	}
	insertMedalTexts(t, db, 100, i18n.Text{i18n.LocaleEN: "Killjoy"}, nil)
	if _, err := db.Exec(ctx,
		`INSERT INTO citation_mappings (medal_id, citation_name_norm) VALUES (200, 'perfection')`,
	); err != nil {
		t.Fatalf("seed citation 200: %v", err)
	}
	insertCitationTexts(t, db, "perfection", i18n.Text{i18n.LocaleFR: "Perfection"}, nil)
}

// TestLookupMedalCitationLabels_Helper couvre le helper partagé (source unique
// du fallback, medal_citation_fallback.go) : résout les IDs présents dans
// citation_mappings, ignore les absents, et tolère db nil / ids vide.
func TestLookupMedalCitationLabels_Helper(t *testing.T) {
	db := openMemDB(t)
	seedMedalFallbackData(t, db)
	ctx := context.Background()

	got, err := lookupMedalCitationLabels(ctx, db, []int64{200, 999})
	if err != nil {
		t.Fatalf("helper: %v", err)
	}
	if got[200] != "Perfection" {
		t.Errorf("helper: medal 200 = %q, want Perfection", got[200])
	}
	if _, ok := got[999]; ok {
		t.Errorf("helper: medal 999 absent de citation_mappings ne doit pas être résolu")
	}

	// Robustesse du contrat : jamais nil, même en entrée dégénérée.
	if m, err := lookupMedalCitationLabels(ctx, nil, []int64{1}); m == nil || err != nil {
		t.Errorf("helper: db nil doit retourner une map vide sans erreur, obtenu (%v, %v)", m, err)
	}
	if m, err := lookupMedalCitationLabels(ctx, db, nil); m == nil || err != nil {
		t.Errorf("helper: ids vide doit retourner une map vide sans erreur, obtenu (%v, %v)", m, err)
	}
}

// TestResolveMedalLabels_CitationFallback couvre la tuile de match Home
// (resolveMedalLabels) : le fallback citation_mappings nouvellement ajouté
// résout les médailles absentes de medal_definitions (avant : libellé vide).
func TestResolveMedalLabels_CitationFallback(t *testing.T) {
	db := openMemDB(t)
	seedMedalFallbackData(t, db)
	ctx := context.Background()

	got := resolveMedalLabels(ctx, db, []int64{100, 200}, "fr")
	if got[100].label != "Killjoy" {
		t.Errorf("home: medal 100 = %q, want Killjoy (medal_definitions)", got[100].label)
	}
	if got[200].label != "Perfection" {
		t.Errorf("home: medal 200 = %q, want Perfection (fallback citation_mappings)", got[200].label)
	}
}

// TestResolveMedalLabels_LocaleAware prouve GH2-B6 : la tuile de match Home sert
// le nom/description de médaille dans la locale de requête. Sous UI EN, JAMAIS de
// texte français — parité avec la vue Match (GH-5b).
func TestResolveMedalLabels_LocaleAware(t *testing.T) {
	db := openMemDB(t)
	seedMedalDefsSchema(t, db)
	ctx := context.Background()
	if _, err := db.Exec(ctx,
		`INSERT INTO medal_definitions (medal_name_id, difficulty) VALUES (300, 'Heroic')`,
	); err != nil {
		t.Fatalf("seed medal 300: %v", err)
	}
	insertMedalTexts(t, db, 300, frEn("Tueur de joie", "Killjoy"), frEn("Fin d une serie", "Ended a spree"))

	fr := resolveMedalLabels(ctx, db, []int64{300}, "fr")
	if fr[300].label != "Tueur de joie" {
		t.Errorf("FR label = %q, want 'Tueur de joie'", fr[300].label)
	}
	if fr[300].description != "Fin d une serie" {
		t.Errorf("FR description = %q, want 'Fin d une serie'", fr[300].description)
	}

	en := resolveMedalLabels(ctx, db, []int64{300}, "en")
	if en[300].label != "Killjoy" {
		t.Errorf("EN label = %q, want 'Killjoy' (jamais de français sous EN)", en[300].label)
	}
	if en[300].description != "Ended a spree" {
		t.Errorf("EN description = %q, want 'Ended a spree' (jamais de français sous EN)", en[300].description)
	}

	// Langue inconnue : repli sur l'anglais, jamais sur la langue par défaut.
	de := resolveMedalLabels(ctx, db, []int64{300}, "de")
	if de[300].label != "Killjoy" || de[300].description != "Ended a spree" {
		t.Errorf("langue inconnue = %+v, want le texte anglais", de[300])
	}
}

// TestLookupMedalMeta_CitationFallback verrouille le refactor de la vue Match
// (lookupMedalMeta délègue désormais au helper partagé) : le fallback
// citation_mappings préexistant fonctionne toujours après extraction.
func TestLookupMedalMeta_CitationFallback(t *testing.T) {
	db := openMemDB(t)
	seedMedalFallbackData(t, db)
	ctx := context.Background()

	repo := NewMatchViewRepo(&PlayerDB{Metadata: db}, pTestXUID)
	got, err := repo.lookupMedalMeta(ctx, []int64{100, 200})
	if err != nil {
		t.Fatalf("lookupMedalMeta: %v", err)
	}
	if got[100].label != "Killjoy" {
		t.Errorf("match-view: medal 100 = %q, want Killjoy (medal_definitions)", got[100].label)
	}
	if got[200].label != "Perfection" {
		t.Errorf("match-view: medal 200 = %q, want Perfection (fallback citation_mappings)", got[200].label)
	}
}

// TestLookupMedalMeta_Langues : la vue d'un match sert label et description dans la langue de la
// requête (contexte), difficulty vient de medal_definitions ; une médaille sans traduction française
// est servie en anglais, jamais vide.
func TestLookupMedalMeta_Langues(t *testing.T) {
	db := openMemDB(t)
	seedMedalDefsSchema(t, db)
	ctx := context.Background()
	if _, err := db.Exec(ctx,
		`INSERT INTO medal_definitions (medal_name_id, difficulty) VALUES (300, 'Heroic'), (301, 'Normal')`,
	); err != nil {
		t.Fatalf("seed medals: %v", err)
	}
	insertMedalTexts(t, db, 300, frEn("Tueur de joie", "Killjoy"), frEn("Fin d une serie", "Ended a spree"))
	insertMedalTexts(t, db, 301, i18n.Text{i18n.LocaleEN: "Only English"}, nil)

	repo := NewMatchViewRepo(&PlayerDB{Metadata: db}, pTestXUID)
	cases := []struct {
		locale, label, desc, only string
	}{
		{"fr", "Tueur de joie", "Fin d une serie", "Only English"},
		{"en", "Killjoy", "Ended a spree", "Only English"},
	}
	for _, c := range cases {
		got, err := repo.lookupMedalMeta(ctxkeys.WithLocale(ctx, c.locale), []int64{300, 301})
		if err != nil {
			t.Fatalf("lookupMedalMeta(%s): %v", c.locale, err)
		}
		if got[300].label != c.label || got[300].description != c.desc || got[300].difficulty != "Heroic" {
			t.Errorf("lookupMedalMeta(%s) 300 = %+v", c.locale, got[300])
		}
		if got[301].label != c.only {
			t.Errorf("lookupMedalMeta(%s) 301 = %q, want %q (repli en)", c.locale, got[301].label, c.only)
		}
	}
}

// TestLookupMedalMetaByName_Langues : le nom anglais du film retrouve la médaille, et son label est
// celui de la langue de la requête.
func TestLookupMedalMetaByName_Langues(t *testing.T) {
	db := openMemDB(t)
	seedMedalDefsSchema(t, db)
	ctx := context.Background()
	if _, err := db.Exec(ctx, `INSERT INTO medal_definitions (medal_name_id) VALUES (300), (400)`); err != nil {
		t.Fatalf("seed medals: %v", err)
	}
	insertMedalTexts(t, db, 300, frEn("Tueur de joie", "Killjoy"), frEn("Fin d une serie", "Ended a spree"))
	insertMedalTexts(t, db, 400, i18n.Text{i18n.LocaleFR: "Seulement français"}, nil)

	repo := NewMatchViewRepo(&PlayerDB{Metadata: db}, pTestXUID)
	for locale, want := range map[string]string{"fr": "Tueur de joie", "en": "Killjoy"} {
		got, err := repo.LookupMedalMetaByName(ctxkeys.WithLocale(ctx, locale), []string{"Killjoy", "Seulement français", ""})
		if err != nil {
			t.Fatalf("LookupMedalMetaByName(%s): %v", locale, err)
		}
		if got["Killjoy"].MedalNameID != 300 || got["Killjoy"].Label != want {
			t.Errorf("LookupMedalMetaByName(%s) Killjoy = %+v, want 300 / %q", locale, got["Killjoy"], want)
		}
		// Seul le nom ANGLAIS est la clé du film : un nom français n'identifie rien.
		if _, ok := got["Seulement français"]; ok {
			t.Errorf("LookupMedalMetaByName(%s) : un nom non anglais ne doit pas résoudre", locale)
		}
	}
}

// TestListMedalsByTitle_LanguesEtRecherche : la liste du tiroir porte les textes de chaque langue ;
// la recherche trouve une médaille par son nom dans n'importe quelle langue ; le tri suit le nom
// anglais ; une langue inconnue se replie sur l'anglais au choix du texte.
func TestListMedalsByTitle_LanguesEtRecherche(t *testing.T) {
	db := openMemDB(t)
	seedMedalDefsSchema(t, db)
	ctx := context.Background()
	if _, err := db.Exec(ctx, `INSERT INTO medal_definitions (medal_name_id) VALUES (1), (2), (3)`); err != nil {
		t.Fatalf("seed medals: %v", err)
	}
	insertMedalTexts(t, db, 1, frEn("Zèbre", "Zebra"), frEn("Rayé", "Striped"))
	insertMedalTexts(t, db, 2, frEn("Abeille", "Bee"), nil)
	insertMedalTexts(t, db, 3, i18n.Text{i18n.LocaleFR: "Sans anglais"}, nil)
	repo := NewMetadataRepoFromDB(db)

	all, err := repo.ListMedalsByTitle(ctx, "halo_infinite", "")
	if err != nil {
		t.Fatalf("ListMedalsByTitle: %v", err)
	}
	if len(all) != 3 || all[0].ID != "2" || all[1].ID != "1" || all[2].ID != "3" {
		t.Fatalf("ordre (nom anglais, puis sans nom anglais) = %+v", all)
	}
	for _, c := range []struct {
		loc  i18n.Locale
		name string
		desc string
	}{
		{i18n.LocaleFR, "Zèbre", "Rayé"},
		{i18n.LocaleEN, "Zebra", "Striped"},
		{i18n.Locale("zz"), "Zebra", "Striped"}, // langue inconnue : repli en
	} {
		got := all[1].Localized(c.loc)
		if got.Name != c.name || got.Description != c.desc {
			t.Errorf("langue %s : %q / %q, want %q / %q", c.loc, got.Name, got.Description, c.name, c.desc)
		}
	}

	// Recherche : un nom français retrouve la médaille, un nom anglais aussi, casse ignorée.
	for _, search := range []string{"zèb", "ZEBRA", "abeille"} {
		got, err := repo.ListMedalsByTitle(ctx, "halo_infinite", search)
		if err != nil {
			t.Fatalf("ListMedalsByTitle(%q): %v", search, err)
		}
		if len(got) != 1 {
			t.Errorf("recherche %q : %d résultat(s), want 1", search, len(got))
		}
	}
}

// TestLoadTitleMedals_TextesParLangue : le tiroir d'actifs des titres à sprites charge les textes de
// chaque langue depuis medal_translations, omet une médaille sans nom anglais et trie par nom anglais.
func TestLoadTitleMedals_TextesParLangue(t *testing.T) {
	db := openMemDB(t)
	seedMedalDefsSchema(t, db)
	ctx := context.Background()
	for _, q := range []string{
		`ALTER TABLE medal_definitions ADD COLUMN sprite_sheet_url VARCHAR`,
		`ALTER TABLE medal_definitions ADD COLUMN sprite_left INTEGER`,
		`ALTER TABLE medal_definitions ADD COLUMN sprite_top INTEGER`,
		`ALTER TABLE medal_definitions ADD COLUMN sprite_width INTEGER`,
		`ALTER TABLE medal_definitions ADD COLUMN sprite_height INTEGER`,
		`INSERT INTO medal_definitions (medal_name_id, sprite_sheet_url, sprite_left, sprite_top, sprite_width, sprite_height)
		 VALUES (1, 'sheet.png', 10, 20, 30, 40), (2, NULL, NULL, NULL, NULL, NULL), (3, NULL, NULL, NULL, NULL, NULL)`,
	} {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("setup %q: %v", q, err)
		}
	}
	insertMedalTexts(t, db, 1, frEn("Zèbre", "Zebra"), frEn("Rayé", "Striped"))
	insertMedalTexts(t, db, 2, frEn("Abeille", "Bee"), nil)
	insertMedalTexts(t, db, 3, i18n.Text{i18n.LocaleFR: "Sans anglais"}, nil)

	got := loadTitleMedals(ctx, db)
	if len(got) != 2 || got[0].ID != "2" || got[1].ID != "1" {
		t.Fatalf("médailles = %+v, want [2 1] (la 3 n'a pas de nom anglais)", got)
	}
	z := got[1].Localized(i18n.LocaleFR)
	if z.Name != "Zèbre" || z.Description != "Rayé" || z.SpriteSheet != "sheet.png" || z.SpriteLeft != 10 || z.SpriteHeight != 40 {
		t.Errorf("médaille 1 en fr = %+v", z)
	}
	if e := got[1].Localized(i18n.LocaleEN); e.Name != "Zebra" || e.Description != "Striped" {
		t.Errorf("médaille 1 en en = %+v", e)
	}
}
