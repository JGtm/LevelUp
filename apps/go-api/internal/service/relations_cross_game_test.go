package service

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// mockCrossGame implémente port.CrossGameCooccurrence : retourne une map fixe
// par xuid et capture les candidats reçus.
type mockCrossGame struct {
	hits     map[string]port.CrossGameHit
	gotXUIDs []string
	called   bool
}

func (m *mockCrossGame) CooccurrencesByXUID(_ context.Context, oppXUIDs []string) map[string]port.CrossGameHit {
	m.called = true
	m.gotXUIDs = oppXUIDs
	return m.hits
}

func crossGameRows() []domain.RelationRawRow {
	now := time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)
	return []domain.RelationRawRow{
		{XUID: "x1", Gamertag: "Ally", TotalMatches: 5, TeammateCount: 5, FirstSeen: now, LastSeen: now},
		{XUID: "x2", Gamertag: "Foe", TotalMatches: 4, EnemyCount: 4, FirstSeen: now, LastSeen: now},
	}
}

func crossGameBadgeCount(insight domain.RelationInsight) int {
	n := 0
	for _, b := range insight.Badges {
		if b.LabelKey == "narrative.encounter.cross_game" {
			n++
		}
	}
	return n
}

func findInsight(page domain.RelationsPageResponse, gt string) (domain.RelationInsight, bool) {
	for _, r := range page.Relations {
		if r.Gamertag == gt {
			return r, true
		}
	}
	return domain.RelationInsight{}, false
}

// Aucune dépendance injectée → chemin Phase 3a strictement inchangé (aucun badge cross-jeu).
func TestCrossGame_NoDependency_NoBadge(t *testing.T) {
	svc := NewRelationsService(&mockRelationsRepo{rows: crossGameRows()}).withNow(fixedNow())
	page, err := svc.GetRelationsPage(context.Background(), domain.FilterContextInput{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	for _, r := range page.Relations {
		if crossGameBadgeCount(r) != 0 {
			t.Fatalf("relation %s should have no cross-game badge", r.Gamertag)
		}
	}
}

// Autre titre présent, x1 >= seuil → badge ; x2 absent de la map → pas de badge.
func TestCrossGame_HitAboveThreshold_AddsBadge(t *testing.T) {
	cg := &mockCrossGame{hits: map[string]port.CrossGameHit{
		"x1": {TitleDisplayName: "Halo 5", MatchesTogether: 7},
	}}
	svc := NewRelationsService(&mockRelationsRepo{rows: crossGameRows()}).
		withNow(fixedNow()).WithCrossGame(cg)
	page, err := svc.GetRelationsPage(context.Background(), domain.FilterContextInput{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !cg.called {
		t.Fatal("crossGame dependency was not consulted")
	}
	ally, _ := findInsight(page, "Ally")
	if crossGameBadgeCount(ally) != 1 {
		t.Fatalf("Ally cross-game badges=%d want 1", crossGameBadgeCount(ally))
	}
	// Vérifie le contrat du badge (style + detail.game).
	var found bool
	for _, b := range ally.Badges {
		if b.LabelKey != "narrative.encounter.cross_game" {
			continue
		}
		found = true
		if b.Style != "solid" {
			t.Errorf("style=%q want solid", b.Style)
		}
		if b.Detail["game"] != "Halo 5" {
			t.Errorf("detail.game=%v want Halo 5", b.Detail["game"])
		}
	}
	if !found {
		t.Fatal("cross-game badge not found on Ally")
	}
	foe, _ := findInsight(page, "Foe")
	if crossGameBadgeCount(foe) != 0 {
		t.Fatalf("Foe should have no cross-game badge (not in hit map)")
	}
}

// Co-occurrence sous le seuil → le port l'aurait filtrée ; mais on durcit aussi
// CrossGameBadge (filet de sécurité côté analysis). Ici un hit < seuil n'ajoute pas de badge.
func TestCrossGame_BelowThreshold_NoBadge(t *testing.T) {
	cg := &mockCrossGame{hits: map[string]port.CrossGameHit{
		"x1": {TitleDisplayName: "Halo 5", MatchesTogether: 1}, // < seuil (3)
	}}
	svc := NewRelationsService(&mockRelationsRepo{rows: crossGameRows()}).
		withNow(fixedNow()).WithCrossGame(cg)
	page, err := svc.GetRelationsPage(context.Background(), domain.FilterContextInput{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	ally, _ := findInsight(page, "Ally")
	if crossGameBadgeCount(ally) != 0 {
		t.Fatalf("Ally cross-game badges=%d want 0 (below threshold)", crossGameBadgeCount(ally))
	}
}

// Aucun autre titre disponible (map vide, simulant l'erreur cross-titre déjà
// avalée par le port) → /relations dégrade gracieusement, zéro badge, zéro erreur.
func TestCrossGame_EmptyHits_Degrades(t *testing.T) {
	cg := &mockCrossGame{hits: map[string]port.CrossGameHit{}}
	svc := NewRelationsService(&mockRelationsRepo{rows: crossGameRows()}).
		withNow(fixedNow()).WithCrossGame(cg)
	page, err := svc.GetRelationsPage(context.Background(), domain.FilterContextInput{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(page.Relations) != 2 {
		t.Fatalf("relations len=%d want 2 (degraded but intact)", len(page.Relations))
	}
	for _, r := range page.Relations {
		if crossGameBadgeCount(r) != 0 {
			t.Fatalf("relation %s should have no cross-game badge", r.Gamertag)
		}
	}
}

// Empty-name hit (titre non résolu) → CrossGameBadge filtre → pas de badge.
func TestCrossGame_EmptyTitleName_NoBadge(t *testing.T) {
	cg := &mockCrossGame{hits: map[string]port.CrossGameHit{
		"x1": {TitleDisplayName: "", MatchesTogether: 9},
	}}
	svc := NewRelationsService(&mockRelationsRepo{rows: crossGameRows()}).
		withNow(fixedNow()).WithCrossGame(cg)
	page, err := svc.GetRelationsPage(context.Background(), domain.FilterContextInput{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	ally, _ := findInsight(page, "Ally")
	if crossGameBadgeCount(ally) != 0 {
		t.Fatalf("Ally cross-game badges=%d want 0 (empty title name)", crossGameBadgeCount(ally))
	}
}
