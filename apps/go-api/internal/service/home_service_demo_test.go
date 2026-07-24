package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/ctxkeys"
)

// TestGetChallenges_DemoMode : en DemoMode, GetChallenges sert la fixture embarquée
// (bypass provider live + cache) avec des items renderables.
func TestGetChallenges_DemoMode(t *testing.T) {
	svc := NewHomeService(nil).WithDemoMode(true)

	resp := svc.GetChallenges(context.Background())

	if !resp.Available {
		t.Fatal("demo challenges: Available = false, want true")
	}
	if len(resp.Items) == 0 {
		t.Fatal("demo challenges: aucun item (la fixture doit peupler Items)")
	}
	if resp.Total == nil || *resp.Total != len(resp.Items) {
		t.Errorf("demo challenges: Total = %v, want %d", resp.Total, len(resp.Items))
	}
	if resp.SnapshotAt == nil {
		t.Error("demo challenges: SnapshotAt nil (fraîcheur attendue)")
	}
	// Chaque item doit avoir un titre non vide (rendu UI).
	for i, it := range resp.Items {
		if it.Title == "" {
			t.Errorf("demo challenges: item %d sans titre", i)
		}
	}
}

// TestGetChallenges_DemoMode_LocaleSelectsFixture : en démo, la fixture est servie
// dans la locale de la requête (ctxkeys.Locale). Régression du bug « défis en FR
// quand l'UI est en anglais » : sans sélection par locale, EN retombait sur la
// fixture FR-only.
func TestGetChallenges_DemoMode_LocaleSelectsFixture(t *testing.T) {
	svc := NewHomeService(nil).WithDemoMode(true)

	fr := svc.GetChallenges(ctxkeys.WithLocale(context.Background(), "fr"))
	en := svc.GetChallenges(ctxkeys.WithLocale(context.Background(), "en"))

	if len(fr.Items) == 0 || len(en.Items) == 0 {
		t.Fatalf("demo challenges: items vides (fr=%d, en=%d)", len(fr.Items), len(en.Items))
	}
	if len(fr.Items) != len(en.Items) {
		t.Fatalf("demo challenges: parité FR/EN rompue (fr=%d, en=%d)", len(fr.Items), len(en.Items))
	}
	// Le 1er défi doit différer entre FR et EN (libellés traduits, pas la même string).
	if fr.Items[0].Title == en.Items[0].Title {
		t.Errorf("demo challenges: titre identique FR/EN (%q) — la locale n'est pas prise en compte", fr.Items[0].Title)
	}
	if en.Items[0].Title != "Killing Spree" {
		t.Errorf("demo challenges EN: 1er titre attendu %q, got %q", "Killing Spree", en.Items[0].Title)
	}
	if fr.Items[0].Title != "Tueur en série" {
		t.Errorf("demo challenges FR: 1er titre attendu %q, got %q", "Tueur en série", fr.Items[0].Title)
	}
}
