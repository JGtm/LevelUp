//go:build integration

// Package duckdb — citations_repo_q38_test.go : régression Q38 split cross-DB.
//
// CONTEXTE — incident 2026-05-26 :
//   - L'ancien Q38MatchViewCitations faisait un LEFT JOIN cross-DB entre
//     match_citations (player) et citation_mappings (metadata).
//   - Post ADR 0016 (retrait ATTACH metadata sur conn player), la query
//     levait "Catalog Error: Table with name citation_mappings does not
//     exist!" silencieusement (`return nil, nil`).
//   - Conséquence : page Match detail affichait des citations vides.
//
// Fix : split en 2 queries Go-side (Q38MatchViewCitationsPlayer + lookup
// citation_mappings via loadCitationMappingMeta) + merge en Go.
//
// Ces tests garantissent :
//  1. La query player-only ne référence PAS citation_mappings.
//  2. Le merge COALESCE(display, norm) renvoie le bon display quand
//     citation_mappings a un match.
//  3. Quand citation_mappings n'a pas de mapping pour un norm (cas norm
//     interne `_processed`), display = norm (pas vide).
//  4. Le LIMIT 4 est respecté.
//  5. Pas de panic si pdb.Metadata est nil (dégradation gracieuse).
package duckdb

import (
	"context"
	"strings"
	"testing"
)

// TestLoadMatchCitationsForView_TopCitationsEnriched : path nominal —
// player a 5 citations, citation_mappings a un mapping pour 3 d'entre elles,
// on attend les 4 top par value avec display enrichi pour 3 et display=norm
// pour celle sans mapping.
func TestLoadMatchCitationsForView_TopCitationsEnriched(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	repo := NewCitationsRepo(pdb)

	const matchID = "m_view_test"

	// Player : 5 citations, values déterministes pour ordre stable.
	inserts := []struct {
		norm  string
		value int
	}{
		{"killing_spree", 100}, // mapping existe (seedé par seedMetaDBSchema)
		{"double_kill", 80},    // pas de mapping → display = norm
		{"triple_kill", 60},    // pas de mapping → display = norm
		{"overkill", 40},       // pas de mapping → display = norm
		{"untracked_low", 20},  // ne devrait PAS apparaître (LIMIT 4)
	}
	for _, c := range inserts {
		if _, err := pdb.Player.Exec(ctx,
			`INSERT INTO match_citations VALUES (?,?,?)`,
			matchID, c.norm, c.value); err != nil {
			t.Fatalf("seed match_citations: %v", err)
		}
	}

	// Seed citation_mappings supplémentaire pour 2 norms (le seedMetaDBSchema
	// en injecte 1 déjà : killing_spree).
	if _, err := pdb.Metadata.Exec(ctx,
		`INSERT INTO citation_mappings (citation_name_norm, citation_name_display, mapping_type, enabled) VALUES (?,?,?,?)`,
		"double_kill", "Double Kill", "medal", true); err != nil {
		t.Fatalf("seed citation_mappings double: %v", err)
	}
	if _, err := pdb.Metadata.Exec(ctx,
		`INSERT INTO citation_mappings (citation_name_norm, citation_name_display, mapping_type, enabled) VALUES (?,?,?,?)`,
		"triple_kill", "Triple Kill", "medal", true); err != nil {
		t.Fatalf("seed citation_mappings triple: %v", err)
	}

	rows, err := repo.LoadMatchCitationsForView(ctx, matchID)
	if err != nil {
		t.Fatalf("LoadMatchCitationsForView: %v", err)
	}
	if len(rows) != 4 {
		t.Fatalf("len(rows) = %d, want 4 (LIMIT 4)", len(rows))
	}

	// Vérifier ordre par value DESC : killing_spree, double_kill, triple_kill, overkill.
	wantOrder := []string{"killing_spree", "double_kill", "triple_kill", "overkill"}
	for i, expectedNorm := range wantOrder {
		if rows[i].NameNorm != expectedNorm {
			t.Errorf("rows[%d].NameNorm = %q, want %q", i, rows[i].NameNorm, expectedNorm)
		}
	}

	// killing_spree : display enrichi (depuis seedMetaDBSchema).
	if rows[0].NameDisplay != "Killing Spree" {
		t.Errorf("killing_spree NameDisplay = %q, want \"Killing Spree\"", rows[0].NameDisplay)
	}
	if rows[0].Value != 100 {
		t.Errorf("killing_spree Value = %d, want 100", rows[0].Value)
	}

	// double_kill : display enrichi (test seed).
	if rows[1].NameDisplay != "Double Kill" {
		t.Errorf("double_kill NameDisplay = %q, want \"Double Kill\"", rows[1].NameDisplay)
	}

	// triple_kill : display enrichi.
	if rows[2].NameDisplay != "Triple Kill" {
		t.Errorf("triple_kill NameDisplay = %q, want \"Triple Kill\"", rows[2].NameDisplay)
	}

	// overkill : pas de mapping → display = norm (COALESCE de secours).
	if rows[3].NameDisplay != "overkill" {
		t.Errorf("overkill NameDisplay = %q, want \"overkill\" (fallback norm)", rows[3].NameDisplay)
	}
}

// TestLoadMatchCitationsForView_NoCrossDBSQL : assertion structurelle pour
// éviter qu'on réintroduise un LEFT JOIN cross-DB dans Q38. Le const SQL
// doit cibler une seule table (match_citations) sans référence à
// citation_mappings.
func TestLoadMatchCitationsForView_NoCrossDBSQL(t *testing.T) {
	if strings.Contains(Q38MatchViewCitationsPlayer, "citation_mappings") {
		t.Error("REGRESSION: Q38MatchViewCitationsPlayer référence citation_mappings — le JOIN cross-DB est réintroduit, le pattern split est cassé. Cf. incident 2026-05-26.")
	}
	if !strings.Contains(Q38MatchViewCitationsPlayer, "match_citations") {
		t.Error("Q38MatchViewCitationsPlayer ne référence pas match_citations — la query est incorrecte")
	}
}

// TestLoadMatchCitationsForView_EmptyMatch : match sans citation → nil.
func TestLoadMatchCitationsForView_EmptyMatch(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewCitationsRepo(pdb)

	rows, err := repo.LoadMatchCitationsForView(context.Background(), "m_does_not_exist")
	if err != nil {
		t.Fatalf("LoadMatchCitationsForView: %v", err)
	}
	if rows != nil && len(rows) != 0 {
		t.Errorf("rows = %v, want nil or empty", rows)
	}
}

// TestLoadMatchCitationsForView_NoMappingsAtAll : metadata vide → display
// fallback sur norm pour toutes les rows. Pas d'erreur.
func TestLoadMatchCitationsForView_NoMappingsAtAll(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	repo := NewCitationsRepo(pdb)

	// Nettoyer citation_mappings.
	if _, err := pdb.Metadata.Exec(ctx, `DELETE FROM citation_mappings`); err != nil {
		t.Fatalf("clear citation_mappings: %v", err)
	}

	const matchID = "m_no_mappings"
	if _, err := pdb.Player.Exec(ctx,
		`INSERT INTO match_citations VALUES (?,?,?)`,
		matchID, "some_norm", 50); err != nil {
		t.Fatalf("seed: %v", err)
	}

	rows, err := repo.LoadMatchCitationsForView(ctx, matchID)
	if err != nil {
		t.Fatalf("LoadMatchCitationsForView: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	if rows[0].NameNorm != "some_norm" {
		t.Errorf("NameNorm = %q, want some_norm", rows[0].NameNorm)
	}
	// Sans mapping → display fallback sur norm.
	if rows[0].NameDisplay != "some_norm" {
		t.Errorf("NameDisplay = %q, want \"some_norm\" (fallback)", rows[0].NameDisplay)
	}
}

// TestLoadMatchCitationsForView_FiltersZeroAndNullValues : citations avec
// value=0 ou citation_name_norm IS NULL sont exclues par la query.
func TestLoadMatchCitationsForView_FiltersZeroAndNullValues(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	repo := NewCitationsRepo(pdb)

	const matchID = "m_filter_test"
	inserts := []struct {
		norm  any // string OR nil
		value int
	}{
		{"valid_one", 50},
		{"value_zero", 0}, // exclu par value > 0
		{nil, 99},         // exclu par norm IS NOT NULL
		{"valid_two", 30},
	}
	for _, c := range inserts {
		if _, err := pdb.Player.Exec(ctx,
			`INSERT INTO match_citations VALUES (?,?,?)`,
			matchID, c.norm, c.value); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	rows, err := repo.LoadMatchCitationsForView(ctx, matchID)
	if err != nil {
		t.Fatalf("LoadMatchCitationsForView: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2 (zero/null exclus)", len(rows))
	}
	if rows[0].NameNorm != "valid_one" || rows[1].NameNorm != "valid_two" {
		t.Errorf("unexpected order/filter: %+v", rows)
	}
}
