//go:build integration

package duckdb

import (
	"context"
	"testing"
)

// seedMedalFallbackData peuple un dataset hétérogène partagé par les tests de
// fallback : 100 résoluble par medal_definitions, 200 SEULEMENT par
// citation_mappings. Réutilise seedMedalDefsSchema (medal_definitions_repo_test.go).
func seedMedalFallbackData(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()
	seedMedalDefsSchema(t, db)
	// name_en seul (name_fr laissé vide) → label déterministe "Killjoy" quelle
	// que soit la locale : resolveMedalLabels et lookupMedalMeta sont locale-aware
	// mais NULLIF(TRIM(name_fr),'') retombe sur name_en même en FR.
	if _, err := db.Exec(ctx,
		`INSERT INTO medal_definitions (medal_name_id, name_en, difficulty) VALUES (100, 'Killjoy', 'Heroic')`,
	); err != nil {
		t.Fatalf("seed medal 100: %v", err)
	}
	if _, err := db.Exec(ctx,
		`INSERT INTO citation_mappings (medal_id, citation_name_display) VALUES (200, 'Perfection')`,
	); err != nil {
		t.Fatalf("seed citation 200: %v", err)
	}
}

// TestLookupMedalCitationLabels_Helper couvre le helper partagé (source unique
// du fallback, medal_citation_fallback.go) : résout les IDs présents dans
// citation_mappings, ignore les absents, et tolère db nil / ids vide.
func TestLookupMedalCitationLabels_Helper(t *testing.T) {
	db := openMemDB(t)
	seedMedalFallbackData(t, db)
	ctx := context.Background()

	got := lookupMedalCitationLabels(ctx, db, []int64{200, 999})
	if got[200] != "Perfection" {
		t.Errorf("helper: medal 200 = %q, want Perfection", got[200])
	}
	if _, ok := got[999]; ok {
		t.Errorf("helper: medal 999 absent de citation_mappings ne doit pas être résolu")
	}

	// Robustesse du contrat : jamais nil, même en entrée dégénérée.
	if m := lookupMedalCitationLabels(ctx, nil, []int64{1}); m == nil {
		t.Error("helper: db nil doit retourner une map vide (pas nil)")
	}
	if m := lookupMedalCitationLabels(ctx, db, nil); m == nil {
		t.Error("helper: ids vide doit retourner une map vide (pas nil)")
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
// colonne FR (name_fr/description_fr) — parité avec la vue Match (GH-5b).
func TestResolveMedalLabels_LocaleAware(t *testing.T) {
	db := openMemDB(t)
	seedMedalDefsSchema(t, db)
	ctx := context.Background()
	if _, err := db.Exec(ctx,
		`INSERT INTO medal_definitions (medal_name_id, name_fr, name_en, description_fr, description_en, difficulty)
		 VALUES (300, 'Tueur de joie', 'Killjoy', 'Fin d une serie', 'Ended a spree', 'Heroic')`,
	); err != nil {
		t.Fatalf("seed medal 300: %v", err)
	}

	fr := resolveMedalLabels(ctx, db, []int64{300}, "fr")
	if fr[300].label != "Tueur de joie" {
		t.Errorf("FR label = %q, want 'Tueur de joie'", fr[300].label)
	}
	if fr[300].description != "Fin d une serie" {
		t.Errorf("FR description = %q, want 'Fin d une serie'", fr[300].description)
	}

	en := resolveMedalLabels(ctx, db, []int64{300}, "en")
	if en[300].label != "Killjoy" {
		t.Errorf("EN label = %q, want 'Killjoy' (jamais name_fr sous EN)", en[300].label)
	}
	if en[300].description != "Ended a spree" {
		t.Errorf("EN description = %q, want 'Ended a spree' (jamais description_fr sous EN)", en[300].description)
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
	got := repo.lookupMedalMeta(ctx, []int64{100, 200})
	if got[100].label != "Killjoy" {
		t.Errorf("match-view: medal 100 = %q, want Killjoy (medal_definitions)", got[100].label)
	}
	if got[200].label != "Perfection" {
		t.Errorf("match-view: medal 200 = %q, want Perfection (fallback citation_mappings)", got[200].label)
	}
}
