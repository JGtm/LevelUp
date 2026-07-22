package handlers

// patterns_test.go — tests unitaires (sans DuckDB) de l'enrichissement des
// libellés de carte du endpoint GET /patterns (Lot A, item A1). Le repo est
// mocké : on vérifie que chaque pattern by_map servi porte un `label` non vide,
// résolu quand le référentiel connaît la carte, replié (« Carte inconnue » + id
// court, jamais le GUID nu) sinon.

import (
	"context"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/patterns"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/port"
)

const (
	mapKnownID   = "known-map-guid-aaaaaaaa-1111"
	mapUnknownID = "unknown-map-guid-bbbbbbbb-2222"
)

// mockPatternsRepo implémente port.PatternsRepository sans DB.
type mockPatternsRepo struct {
	rows   []patterns.MatchRow
	labels map[string]string
}

func (m *mockPatternsRepo) LoadRows(_ context.Context, _ int) ([]patterns.MatchRow, error) {
	return m.rows, nil
}

func (m *mockPatternsRepo) ResolveMapLabels(_ context.Context, _ []string) (map[string]string, error) {
	return m.labels, nil
}

var _ port.PatternsRepository = (*mockPatternsRepo)(nil)

// buildByMapRows produit 5 matchs sur chacune des deux cartes (>= MinMatchesPerGroup)
// pour garantir l'émission d'un pattern by_map par carte.
func buildByMapRows() []patterns.MatchRow {
	var rows []patterns.MatchRow
	mk := func(mapID string, outcome int) patterns.MatchRow {
		return patterns.MatchRow{Mode: "Slayer", MapID: mapID, Outcome: outcome}
	}
	for i := 0; i < 5; i++ {
		rows = append(rows, mk(mapKnownID, 2))   // wins
		rows = append(rows, mk(mapUnknownID, 3)) // losses
	}
	return rows
}

func newTestPatternsHandler(repo port.PatternsRepository) *PatternsHandler {
	return NewPatternsHandler(func(_ context.Context, _ string) (port.PatternsRepository, error) {
		return repo, nil
	}, "halo_infinite")
}

// byMapPatterns filtre les patterns by_map du rapport.
func byMapPatterns(report patterns.PatternReport) []patterns.ContextualPattern {
	var out []patterns.ContextualPattern
	for _, p := range report.ContextPatterns {
		if p.Type == patterns.ContextByMap {
			out = append(out, p)
		}
	}
	return out
}

func TestGetPatterns_ByMapLabelsResolvedOrFallback(t *testing.T) {
	repo := &mockPatternsRepo{
		rows:   buildByMapRows(),
		labels: map[string]string{mapKnownID: "Aquarius"},
	}
	h := newTestPatternsHandler(repo)

	out, err := h.GetPatterns(context.Background(), &patternsInput{PlayerSlug: "p"})
	if err != nil {
		t.Fatalf("GetPatterns: %v", err)
	}

	byMap := byMapPatterns(out.Body)
	if len(byMap) < 2 {
		t.Fatalf("attendu >= 2 patterns by_map, obtenu %d", len(byMap))
	}

	for _, p := range byMap {
		// Contrat A1 : chaque pattern by_map servi a un label NON VIDE.
		if strings.TrimSpace(p.Label) == "" {
			t.Errorf("pattern by_map %q sans label", p.Key)
		}
		// Jamais le GUID nu affiché tel quel.
		if p.Label == p.Key {
			t.Errorf("label == GUID nu pour %q", p.Key)
		}
		switch p.Key {
		case mapKnownID:
			if p.Label != "Aquarius" {
				t.Errorf("carte connue: label = %q, want Aquarius", p.Label)
			}
		case mapUnknownID:
			// Repli FR (locale par défaut) + id court, jamais le GUID complet.
			if !strings.HasPrefix(p.Label, "Carte inconnue (") {
				t.Errorf("carte inconnue: label = %q, want repli « Carte inconnue »", p.Label)
			}
			if strings.Contains(p.Label, mapUnknownID) {
				t.Errorf("le repli ne doit jamais contenir le GUID complet: %q", p.Label)
			}
		}
	}
}

func TestGetPatterns_MapLeverLabelHasNoGUID(t *testing.T) {
	// mapUnknownID est la faiblesse (0 % de victoires) -> génère un levier
	// map_avoidance dont le texte contenait le GUID. Le nom résolu doit le
	// remplacer dans la phrase (A2).
	repo := &mockPatternsRepo{
		rows:   buildByMapRows(),
		labels: map[string]string{mapUnknownID: "Behemoth"},
	}
	h := newTestPatternsHandler(repo)

	out, err := h.GetPatterns(context.Background(), &patternsInput{PlayerSlug: "p"})
	if err != nil {
		t.Fatalf("GetPatterns: %v", err)
	}

	var found bool
	for _, lev := range out.Body.Levers {
		if !strings.HasPrefix(lev.SourcePattern, "by_map:") {
			continue
		}
		found = true
		if strings.Contains(lev.Label, mapUnknownID) {
			t.Errorf("le texte du levier contient encore le GUID: %q", lev.Label)
		}
		if !strings.Contains(lev.Label, "Behemoth") {
			t.Errorf("le texte du levier devrait contenir le nom résolu: %q", lev.Label)
		}
	}
	if !found {
		t.Fatal("aucun levier by_map produit (le test ne couvre rien)")
	}
}

func TestGetPatterns_FallbackLocaleEN(t *testing.T) {
	repo := &mockPatternsRepo{rows: buildByMapRows(), labels: nil}
	h := newTestPatternsHandler(repo)

	ctx := ctxkeys.WithLocale(context.Background(), "en")
	out, err := h.GetPatterns(ctx, &patternsInput{PlayerSlug: "p"})
	if err != nil {
		t.Fatalf("GetPatterns: %v", err)
	}
	for _, p := range byMapPatterns(out.Body) {
		if !strings.HasPrefix(p.Label, "Unknown map (") {
			t.Errorf("repli EN attendu, obtenu %q", p.Label)
		}
	}
}
