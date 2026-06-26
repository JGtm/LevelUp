//go:build integration

package duckdb

import (
	"context"
	"testing"
)

// seedMedalDefsSchema crée le schéma minimal interrogé par
// MedalDefinitionsRepo.LookupByIDs : medal_definitions + medal_translations +
// citation_mappings (fallback).
func seedMedalDefsSchema(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()
	stmts := []string{
		`CREATE TABLE medal_definitions (
			medal_name_id  BIGINT PRIMARY KEY,
			name_en        VARCHAR DEFAULT '',
			name_fr        VARCHAR DEFAULT '',
			description_en VARCHAR DEFAULT '',
			description_fr VARCHAR DEFAULT '',
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
			medal_id              BIGINT,
			citation_name_display VARCHAR DEFAULT ''
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
//   - 100 : présent dans medal_definitions (name_en) → résolu directement ;
//   - 200 : absent de medal_definitions, présent dans citation_mappings → résolu
//     par le fallback (corrige Explorer top_medals + Squad) ;
//   - 300 : présent dans medal_definitions MAIS name_en vide, rattrapé par
//     citation_mappings (parité avec la vue Match).
func TestMedalDefinitionsRepo_LookupByIDs_CitationFallback(t *testing.T) {
	db := openMemDB(t)
	seedMedalDefsSchema(t, db)
	ctx := context.Background()

	if _, err := db.Exec(ctx,
		`INSERT INTO medal_definitions (medal_name_id, name_en, difficulty) VALUES (100, 'Killjoy', 'Heroic')`,
	); err != nil {
		t.Fatalf("insert medal 100: %v", err)
	}
	// 300 : dans medal_definitions avec un libellé VIDE (cas réel : référentiel
	// incomplet) — doit basculer sur le fallback citation_mappings.
	if _, err := db.Exec(ctx,
		`INSERT INTO medal_definitions (medal_name_id, name_en) VALUES (300, '')`,
	); err != nil {
		t.Fatalf("insert medal 300: %v", err)
	}
	if _, err := db.Exec(ctx,
		`INSERT INTO citation_mappings (medal_id, citation_name_display) VALUES (200, 'Perfection'), (300, 'Grand Slam')`,
	); err != nil {
		t.Fatalf("insert citation_mappings: %v", err)
	}

	repo := NewMedalDefinitionsRepo(&PlayerDB{Metadata: db})
	got, err := repo.LookupByIDs(ctx, []int64{100, 200, 300, 999}, "en")
	if err != nil {
		t.Fatalf("LookupByIDs: %v", err)
	}

	if got[100].Label != "Killjoy" {
		t.Errorf("medal 100 label = %q, want Killjoy (medal_definitions)", got[100].Label)
	}
	if got[200].Label != "Perfection" {
		t.Errorf("medal 200 label = %q, want Perfection (fallback citation_mappings)", got[200].Label)
	}
	if got[300].Label != "Grand Slam" {
		t.Errorf("medal 300 label = %q, want Grand Slam (fallback sur libellé vide)", got[300].Label)
	}
	// 999 : ni medal_definitions ni citation_mappings → absent (le front affiche l'ID).
	if row, ok := got[999]; ok && row.Label != "" {
		t.Errorf("medal 999 ne devrait pas être résolu, got %+v", row)
	}
}

// TestMedalReaders_FRParity_LookupByIDs_vs_ListMedalsByTitle est le garde-rail
// contre une 5e copie divergente du COALESCE médaille. Il vérifie que pour un même
// medal_id, le lecteur Explorer/Squad (MedalDefinitionsRepo.LookupByIDs, locale fr)
// et le lecteur du tab Assets (MetadataRepo.ListMedalsByTitle) renvoient le MÊME
// label ET la MÊME description FR — les deux passant désormais par le helper unique
// medalLabelDescCoalesceSQL.
//
// Régression couverte : avant centralisation, LookupByIDs OMETTAIT md.name_fr /
// md.description_fr (médailles non traduites sur Escouade, sur H5 ET Infinite).
// medal_translations est laissée VIDE ici : c'est la réalité prod (aucun INSERT Go),
// donc la vraie source FR est medal_definitions.name_fr/description_fr.
func TestMedalReaders_FRParity_LookupByIDs_vs_ListMedalsByTitle(t *testing.T) {
	db := openMemDB(t)
	seedMedalDefsSchema(t, db)
	ctx := context.Background()

	// 500 : FR + EN renseignés (name_fr/description_fr) — medal_translations VIDE.
	if _, err := db.Exec(ctx,
		`INSERT INTO medal_definitions
		    (medal_name_id, name_en, name_fr, description_en, description_fr)
		 VALUES (500, 'Perfect Kill', 'Tir Parfait', 'A flawless takedown', 'Une élimination sans faille')`,
	); err != nil {
		t.Fatalf("insert medal 500: %v", err)
	}

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
				labelFR: list[i].NameFR,
				descFR:  list[i].DescriptionFR,
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

	// PARITÉ : ListMedalsByTitle expose les MÊMES valeurs FR (colonnes name_fr /
	// description_fr de l'AssetMeta).
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
