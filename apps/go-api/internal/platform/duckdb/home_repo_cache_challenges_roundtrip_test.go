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

	"levelup/go-api/internal/ctxkeys"
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

	// Items rendus (titre/image/vrai path résolus côté provider) — matchés par TrackingID.
	// ChallengePath = vrai chemin GameCMS (porte le marqueur de cadence DailyChallenges).
	items := []domain.ChallengeItem{
		{TrackingID: strPtr("t1"), ChallengePath: "ChallengeContent/Csv/DailyChallenges/d1.json",
			Title: "Tuer 10 Spartans", ImageURL: strPtr("/img/t1.png"), Description: strPtr("desc t1")},
		{TrackingID: strPtr("t2"), ChallengePath: "ChallengeContent/Csv/WeeklyChallenges/w1.json",
			Title: "Gagner 4 parties"}, // sans image
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
	// display_path : le vrai chemin GameCMS doit être relu (le front en dérive la cadence).
	if t1.ChallengePath != "ChallengeContent/Csv/DailyChallenges/d1.json" {
		t.Errorf("ChallengePath (display_path) attendu = vrai path GameCMS, got %q", t1.ChallengePath)
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

// TestPersistThenLoadCachedChallenges_LocaleCoexistence — PREUVE du fix Cause B :
// deux fetches de la MÊME liste de défis (corps /decks identique → state_hash
// langue-indépendant) mais rendus en FR puis en EN doivent COEXISTER en cache, et le
// reader doit servir la langue demandée. Sans la locale dans la clé de dédup, l'INSERT
// EN serait vu « inchangé » et silencieusement sauté (seule la dernière langue
// survivrait) ; sans le filtre locale au reader, on resservirait n'importe quelle langue.
func TestPersistThenLoadCachedChallenges_LocaleCoexistence(t *testing.T) {
	player := openBattlePassTestDB(t, "player_locale_rt.duckdb", migration.TargetPlayer)
	sink := &PersistSink{PlayerPath: player.Path(), XUID: "xuid-loc"}

	// Corps /decks IDENTIQUE pour les deux locales → même state_hash (langue-indépendant).
	body := []byte(`{"AssignedDecks":[{
		"Expiration":{"ISO8601Date":"2030-01-01T00:00:00Z"},
		"ActiveChallenges":[
			{"TrackingId":"t1","XPReward":500,"Threshold":10,"CurrentProgress":3}
		],
		"CompletedChallenges":[]
	}]}`)

	frItems := []domain.ChallengeItem{{
		TrackingID: strPtr("t1"), ChallengePath: "ChallengeContent/Csv/DailyChallenges/d1.json",
		Title: "Tuer 10 Spartans", Description: strPtr("Éliminez 10 Spartans"),
	}}
	enItems := []domain.ChallengeItem{{
		TrackingID: strPtr("t1"), ChallengePath: "ChallengeContent/Csv/DailyChallenges/d1.json",
		Title: "Kill 10 Spartans", Description: strPtr("Eliminate 10 Spartans"),
	}}

	ctxFR := ctxkeys.WithLocale(context.Background(), "fr")
	ctxEN := ctxkeys.WithLocale(context.Background(), "en")

	if err := sink.PersistChallengesSync(ctxFR, body, frItems); err != nil {
		t.Fatalf("PersistChallengesSync FR: %v", err)
	}
	if err := sink.PersistChallengesSync(ctxEN, body, enItems); err != nil {
		t.Fatalf("PersistChallengesSync EN: %v", err)
	}

	repo := NewHomeRepo(&PlayerDB{Player: player, XUID: "xuid-loc"})

	frResp, hitFR, err := repo.LoadCachedChallenges(ctxFR, 24*time.Hour)
	if err != nil || !hitFR || frResp == nil {
		t.Fatalf("LoadCachedChallenges FR: hit=%v err=%v", hitFR, err)
	}
	enResp, hitEN, err := repo.LoadCachedChallenges(ctxEN, 24*time.Hour)
	if err != nil || !hitEN || enResp == nil {
		t.Fatalf("LoadCachedChallenges EN: hit=%v err=%v", hitEN, err)
	}

	if len(frResp.Items) != 1 || len(enResp.Items) != 1 {
		t.Fatalf("attendu 1 carte par locale, got fr=%d en=%d", len(frResp.Items), len(enResp.Items))
	}
	if frResp.Items[0].Title != "Tuer 10 Spartans" {
		t.Errorf("FR: titre attendu %q, got %q — l'insert FR a-t-il été écrasé ?", "Tuer 10 Spartans", frResp.Items[0].Title)
	}
	if enResp.Items[0].Title != "Kill 10 Spartans" {
		t.Errorf("EN: titre attendu %q, got %q — insert EN droppé (locale absente de la dédup) ou reader non filtré", "Kill 10 Spartans", enResp.Items[0].Title)
	}
}
