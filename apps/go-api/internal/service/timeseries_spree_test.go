package service

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
)

// TestEnrichMatchesMaxKillingSpree : la folie meurtrière est calculée depuis les events
// kill/death pour les matchs SANS valeur native (Halo 5), sans écraser une valeur native
// existante (Infinite), et reste nil si aucun event pour le match.
func TestEnrichMatchesMaxKillingSpree(t *testing.T) {
	const xuid = "player1"
	// m1 : 3 kills d'affilée → mort → 1 kill ⇒ spree max = 3.
	events := []canonical.HighlightEvent{
		{MatchID: "m1", EventType: string(canonical.EventKill), TimeMS: 10, XUID: xuid},
		{MatchID: "m1", EventType: string(canonical.EventKill), TimeMS: 20, XUID: xuid},
		{MatchID: "m1", EventType: string(canonical.EventKill), TimeMS: 30, XUID: xuid},
		{MatchID: "m1", EventType: string(canonical.EventDeath), TimeMS: 40, XUID: xuid},
		{MatchID: "m1", EventType: string(canonical.EventKill), TimeMS: 50, XUID: xuid},
		// kill d'un AUTRE joueur sur le même match — ignoré (filtre XUID).
		{MatchID: "m1", EventType: string(canonical.EventKill), TimeMS: 60, XUID: "other"},
	}
	seven := 7
	matches := []legacymatch.StatsMatchRow{
		{MatchID: "m1"},                          // nil → calculé = 3
		{MatchID: "m2", MaxKillingSpree: &seven}, // valeur native → préservée
		{MatchID: "m3"},                          // pas d'events → reste nil
	}

	enrichMatchesMaxKillingSpree(matches, events, xuid)

	if matches[0].MaxKillingSpree == nil || *matches[0].MaxKillingSpree != 3 {
		t.Errorf("m1 spree = %v, want 3", matches[0].MaxKillingSpree)
	}
	if matches[1].MaxKillingSpree == nil || *matches[1].MaxKillingSpree != 7 {
		t.Errorf("m2 spree native = %v, want 7 préservé (non écrasé)", matches[1].MaxKillingSpree)
	}
	if matches[2].MaxKillingSpree != nil {
		t.Errorf("m3 spree = %v, want nil (aucun event)", matches[2].MaxKillingSpree)
	}
}

// TestEnrichMatchesMaxKillingSpree_NoOpGuards : xuid vide ou events vides ⇒ no-op.
func TestEnrichMatchesMaxKillingSpree_NoOpGuards(t *testing.T) {
	matches := []legacymatch.StatsMatchRow{{MatchID: "m1"}}
	enrichMatchesMaxKillingSpree(matches, nil, "x")
	if matches[0].MaxKillingSpree != nil {
		t.Errorf("events nil → spree doit rester nil, obtenu %v", matches[0].MaxKillingSpree)
	}
	enrichMatchesMaxKillingSpree(matches, []canonical.HighlightEvent{
		{MatchID: "m1", EventType: string(canonical.EventKill), TimeMS: 1, XUID: "x"},
	}, "")
	if matches[0].MaxKillingSpree != nil {
		t.Errorf("xuid vide → spree doit rester nil, obtenu %v", matches[0].MaxKillingSpree)
	}
}
