//go:build integration

package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/i18n"
)

// seedMedalDefsSchema crée le schéma minimal interrogé par
// MedalDefinitionsRepo.LookupByIDs : medal_definitions (colonnes non textuelles) +
// medal_translations (les textes, une ligne par langue) + citation_mappings (fallback).
func seedMedalDefsSchema(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()
	stmts := []string{
		`CREATE TABLE medal_definitions (
			medal_name_id  BIGINT PRIMARY KEY,
			difficulty     VARCHAR DEFAULT '',
			medal_type     VARCHAR DEFAULT '',
			personal_score INTEGER DEFAULT 0
		)`,
		`CREATE TABLE medal_translations (
			medal_name_id BIGINT,
			lang          VARCHAR,
			name          VARCHAR DEFAULT '',
			description   VARCHAR DEFAULT ''
		)`,
		`CREATE TABLE citation_mappings (
			medal_id           BIGINT,
			citation_name_norm VARCHAR
		)`,
		`CREATE TABLE asset_translations (
			asset_id    VARCHAR,
			asset_type  VARCHAR,
			lang        VARCHAR,
			name        VARCHAR,
			description VARCHAR,
			fetched_at  TIMESTAMP,
			PRIMARY KEY (asset_id, asset_type, lang)
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(ctx, s); err != nil {
			t.Fatalf("seedMedalDefsSchema: %v", err)
		}
	}
}

// TestMedalDefinitionsRepo_LookupByIDs_CitationFallback couvre un dataset
// hétérogène :
//   - 100 : présent dans medal_definitions avec un nom en → résolu directement ;
//   - 200 : absent de medal_definitions, présent dans citation_mappings → résolu
//     par le fallback (corrige Explorer top_medals + Squad) ;
//   - 300 : présent dans medal_definitions MAIS sans aucun texte, rattrapé par
//     citation_mappings (parité avec la vue Match).
func TestMedalDefinitionsRepo_LookupByIDs_CitationFallback(t *testing.T) {
	db := openMemDB(t)
	seedMedalDefsSchema(t, db)
	ctx := context.Background()

	if _, err := db.Exec(ctx,
		`INSERT INTO medal_definitions (medal_name_id, difficulty) VALUES (100, 'Heroic')`,
	); err != nil {
		t.Fatalf("insert medal 100: %v", err)
	}
	insertMedalTexts(t, db, 100, i18n.Text{i18n.LocaleEN: "Killjoy"}, nil)
	// 300 : dans medal_definitions sans texte (cas réel : référentiel incomplet) — doit
	// basculer sur le fallback citation_mappings.
	if _, err := db.Exec(ctx,
		`INSERT INTO medal_definitions (medal_name_id) VALUES (300)`,
	); err != nil {
		t.Fatalf("insert medal 300: %v", err)
	}
	if _, err := db.Exec(ctx,
		`INSERT INTO citation_mappings (medal_id, citation_name_norm) VALUES (200, 'perfection'), (300, 'grand_slam')`,
	); err != nil {
		t.Fatalf("insert citation_mappings: %v", err)
	}
	insertCitationTexts(t, db, "perfection", i18n.Text{i18n.LocaleFR: "Perfection"}, nil)
	insertCitationTexts(t, db, "grand_slam", i18n.Text{i18n.LocaleFR: "Grand Slam"}, nil)

	repo := NewMedalDefinitionsRepo(&PlayerDB{Metadata: db})
	got, err := repo.LookupByIDs(ctx, []int64{100, 200, 300, 999}, "en")
	if err != nil {
		t.Fatalf("LookupByIDs: %v", err)
	}

	if got[100].Label != "Killjoy" || got[100].Difficulty != "Heroic" {
		t.Errorf("medal 100 = %+v, want Killjoy / Heroic (medal_translations + medal_definitions)", got[100])
	}
	if got[200].Label != "Perfection" {
		t.Errorf("medal 200 label = %q, want Perfection (fallback citation_mappings)", got[200].Label)
	}
	if got[300].Label != "Grand Slam" {
		t.Errorf("medal 300 label = %q, want Grand Slam (fallback sur médaille sans texte)", got[300].Label)
	}
	// 999 : ni medal_definitions ni citation_mappings → absent (le front affiche l'ID).
	if row, ok := got[999]; ok && row.Label != "" {
		t.Errorf("medal 999 ne devrait pas être résolu, got %+v", row)
	}
}

// TestMedalDefinitionsRepo_LookupByIDs_Langues : le texte vient de medal_translations dans la
// langue demandée ; une langue inconnue ou vide lit l'anglais, jamais le français ; un nom
// absent dans la langue de repli reste vide (aucune traduction inventée).
func TestMedalDefinitionsRepo_LookupByIDs_Langues(t *testing.T) {
	db := openMemDB(t)
	seedMedalDefsSchema(t, db)
	ctx := context.Background()
	if _, err := db.Exec(ctx,
		`INSERT INTO medal_definitions (medal_name_id, difficulty, medal_type, personal_score) VALUES
		 (500, 'Heroic', 'skill', 7), (501, 'Normal', '', 0)`,
	); err != nil {
		t.Fatalf("insert medals: %v", err)
	}
	insertMedalTexts(t, db, 500,
		frEn("Tir Parfait", "Perfect Kill"), frEn("Une élimination sans faille", "A flawless takedown"))
	// 501 : un nom français seul.
	insertMedalTexts(t, db, 501, i18n.Text{i18n.LocaleFR: "Seulement français"}, nil)

	repo := NewMedalDefinitionsRepo(&PlayerDB{Metadata: db})
	cases := []struct {
		locale, label, desc string
	}{
		{"fr", "Tir Parfait", "Une élimination sans faille"},
		{"en", "Perfect Kill", "A flawless takedown"},
		{"EN-gb", "Perfect Kill", "A flawless takedown"},
		{"de", "Perfect Kill", "A flawless takedown"}, // langue inconnue : repli en
		{"", "Perfect Kill", "A flawless takedown"},   // vide : repli en
	}
	for _, c := range cases {
		got, err := repo.LookupByIDs(ctx, []int64{500}, c.locale)
		if err != nil {
			t.Fatalf("LookupByIDs(%q): %v", c.locale, err)
		}
		if got[500].Label != c.label || got[500].Description != c.desc {
			t.Errorf("LookupByIDs(%q) = %q / %q, want %q / %q", c.locale, got[500].Label, got[500].Description, c.label, c.desc)
		}
		if got[500].Difficulty != "Heroic" || got[500].MedalType != "skill" || got[500].PersonalScore != 7 {
			t.Errorf("LookupByIDs(%q) : colonnes non textuelles = %+v", c.locale, got[500])
		}
	}
	got, err := repo.LookupByIDs(ctx, []int64{501}, "en")
	if err != nil {
		t.Fatalf("LookupByIDs(501, en): %v", err)
	}
	if got[501].Label != "" {
		t.Errorf("médaille 501 en en = %q, want vide (aucun texte anglais)", got[501].Label)
	}
	got, err = repo.LookupByIDs(ctx, []int64{501}, "fr")
	if err != nil {
		t.Fatalf("LookupByIDs(501, fr): %v", err)
	}
	if got[501].Label != "Seulement français" {
		t.Errorf("médaille 501 en fr = %q", got[501].Label)
	}
}

// TestMedalReaders_FRParity_LookupByIDs_vs_ListMedalsByTitle est le garde-rail
// contre une copie divergente de la résolution des textes de médaille. Il vérifie que pour un
// même medal_id, le lecteur Explorer/Squad (MedalDefinitionsRepo.LookupByIDs, langue fr) et le
// lecteur du tab Assets (MetadataRepo.ListMedalsByTitle) renvoient le MÊME label ET la MÊME
// description FR — tous deux lus dans medal_translations.
func TestMedalReaders_FRParity_LookupByIDs_vs_ListMedalsByTitle(t *testing.T) {
	db := openMemDB(t)
	seedMedalDefsSchema(t, db)
	ctx := context.Background()

	if _, err := db.Exec(ctx, `INSERT INTO medal_definitions (medal_name_id) VALUES (500)`); err != nil {
		t.Fatalf("insert medal 500: %v", err)
	}
	insertMedalTexts(t, db, 500,
		frEn("Tir Parfait", "Perfect Kill"), frEn("Une élimination sans faille", "A flawless takedown"))

	lookup := NewMedalDefinitionsRepo(&PlayerDB{Metadata: db})
	got, err := lookup.LookupByIDs(ctx, []int64{500}, "fr")
	if err != nil {
		t.Fatalf("LookupByIDs(fr): %v", err)
	}

	listRepo := NewMetadataRepoFromDB(db)
	list, err := listRepo.ListMedalsByTitle(ctx, "halo_infinite", "")
	if err != nil {
		t.Fatalf("ListMedalsByTitle: %v", err)
	}

	var listMedal *canonicalAssetMetaShim
	for i := range list {
		if list[i].ID == "500" {
			listMedal = &canonicalAssetMetaShim{
				labelFR: list[i].Names[i18n.LocaleFR],
				descFR:  list[i].Descriptions[i18n.LocaleFR],
			}
			break
		}
	}
	if listMedal == nil {
		t.Fatalf("ListMedalsByTitle: medal 500 absent du résultat")
	}

	// Le label/description résolus par LookupByIDs(fr) sont les valeurs FR.
	if got[500].Label != "Tir Parfait" {
		t.Errorf("LookupByIDs(fr) label = %q, want Tir Parfait", got[500].Label)
	}
	if got[500].Description != "Une élimination sans faille" {
		t.Errorf("LookupByIDs(fr) description = %q, want Une élimination sans faille", got[500].Description)
	}

	// PARITÉ : ListMedalsByTitle expose les MÊMES valeurs FR (textes par langue de l'AssetMeta).
	if listMedal.labelFR != got[500].Label {
		t.Errorf("parité label FR : ListMedalsByTitle=%q, LookupByIDs=%q", listMedal.labelFR, got[500].Label)
	}
	if listMedal.descFR != got[500].Description {
		t.Errorf("parité description FR : ListMedalsByTitle=%q, LookupByIDs=%q", listMedal.descFR, got[500].Description)
	}
}

// canonicalAssetMetaShim isole les champs FR comparés dans le test de parité
// (évite d'importer le package canonical dans ce fichier de test du package duckdb).
type canonicalAssetMetaShim struct {
	labelFR string
	descFR  string
}
