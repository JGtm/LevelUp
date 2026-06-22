// Package duckdb — round-trip persist→read des Défis avec colonnes de rendu.
//
// Garde-fou C4 (bout-en-bout) : vérifie que PersistChallengesSync ÉCRIT bien
// title/description/image_url (matching items↔snapshot par TrackingID) et que
// LoadCachedChallenges les RELIT depuis DuckDB pour reconstruire de vraies cartes —
// le maillon exact qui fait que les Défis s'affichent hors-ligne au lieu d'« indisponible ».
package duckdb

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
)

func strPtr(s string) *string { return &s }

func TestPersistThenLoadCachedChallenges_ReconstructsRenderedCards(t *testing.T) {
	ctx := context.Background()
	player := openBattlePassTestDB(t, "player_rt.duckdb", migration.TargetPlayer)

	sink := &PersistSink{PlayerPath: player.Path(), XUID: "xuid-rt"}

	body := []byte(`{"AssignedDecks":[{
		"Expiration":{"ISO8601Date":"2030-01-01T00:00:00Z"},
		"ActiveChallenges":[
			{"TrackingId":"t1","XPReward":500,"Threshold":10,"CurrentProgress":3},
			{"TrackingId":"t2","XPReward":250,"Threshold":4,"CurrentProgress":1}
		],
		"CompletedChallenges":[]
	}]}`)

	// Items rendus (titre/image résolus côté provider) — matchés par TrackingID.
	items := []domain.ChallengeItem{
		{TrackingID: strPtr("t1"), Title: "Tuer 10 Spartans", ImageURL: strPtr("/img/t1.png"), Description: strPtr("desc t1")},
		{TrackingID: strPtr("t2"), Title: "Gagner 4 parties"}, // sans image
	}

	if err := sink.PersistChallengesSync(ctx, body, items); err != nil {
		t.Fatalf("PersistChallengesSync: %v", err)
	}

	repo := NewHomeRepo(&PlayerDB{Player: player, XUID: "xuid-rt"})
	resp, hit, err := repo.LoadCachedChallenges(ctx, 24*time.Hour)
	if err != nil {
		t.Fatalf("LoadCachedChallenges: %v", err)
	}
	if !hit || resp == nil {
		t.Fatal("attendu un hit cache")
	}
	if len(resp.Items) != 2 {
		t.Fatalf("attendu 2 cartes reconstruites, got %d", len(resp.Items))
	}

	byTitle := map[string]domain.ChallengeItem{}
	for _, it := range resp.Items {
		byTitle[it.Title] = it
	}
	t1, ok := byTitle["Tuer 10 Spartans"]
	if !ok {
		t.Fatal("carte t1 (titre) absente — title non persisté/relu")
	}
	if t1.ImageURL == nil || *t1.ImageURL != "/img/t1.png" {
		t.Errorf("image t1 non persistée/relue, got %v", t1.ImageURL)
	}
	if t1.Description == nil || *t1.Description != "desc t1" {
		t.Errorf("description t1 non persistée/relue, got %v", t1.Description)
	}
	if t1.ProgressPercent == nil || *t1.ProgressPercent != 30 {
		t.Errorf("ProgressPercent t1 attendu 30 (3/10), got %v", t1.ProgressPercent)
	}
	if _, ok := byTitle["Gagner 4 parties"]; !ok {
		t.Error("carte t2 (titre sans image) absente")
	}
}
