// Package service — career_live_e2e_scenarios_test.go : tests E2E des
// scénarios API qui ont historiquement causé le flicker home (V2 §7).
//
// Chaque scénario simule un retour API live distinct et vérifie 2 contrats :
//  1. La DB ne se fait JAMAIS écraser par des valeurs vides venant du live
//  2. La home rend toujours quelque chose de cohérent (DB last-known-good
//     pour les champs que le live n'a pas pu actualiser)
package service

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
	syncpkg "levelup/go-api/internal/sync"
)

// e2eDBLast = état initial typique en DB (toutes les valeurs connues).
func e2eDBLast() *duckdb.CareerRankRow {
	return &duckdb.CareerRankRow{
		Rank:             180,
		CurrentXP:        4500,
		XPForNextRank:    5000,
		SpartanID:        "OKLM",
		BannerImageURL:   "https://db/banner.png",
		EmblemImageURL:   "https://db/emblem.png",
		BackdropImageURL: "https://db/backdrop.png",
	}
}

// runE2E : helper qui prépare service+mock, lance GetSpartanIdentityFor puis
// laisse 50ms au background goroutine pour finir avant inspection des inserts.
func runE2E(t *testing.T, fetcher *mockCareerFetcher, dbLast *duckdb.CareerRankRow) *mockCareerLiveRepo {
	t.Helper()
	repo := &mockCareerLiveRepo{last: dbLast, insertPartialOK: true}
	builder := &mockIdentityBuilder{}
	cache := NewCareerLiveCache(CareerLiveCacheConfig{ProgressTTL: 5 * time.Minute, CustomizationTTL: 6 * time.Hour}) // vide au boot
	factory := func(_ context.Context) CareerFetcher {
		if fetcher == nil {
			return nil
		}
		return fetcher
	}
	svc := NewCareerLiveService(repo, builder, factory, cache)
	ctx := ctxWithTokens(t, true)

	_, _ = svc.GetSpartanIdentityFor(ctx, ctxTokensXUID)
	// Le background refresh est détaché : on attend qu'il finisse.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		svc.bgInflightMu.Lock()
		done := len(svc.bgInflight) == 0
		svc.bgInflightMu.Unlock()
		if done {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	return repo
}

// Scénario 1 — cold start, live rend tout vide (API muette) :
// → rien n'est INSERT dans la DB (pas de pollution)
func TestE2E_LiveSilent_NoInsert(t *testing.T) {
	fetcher := &mockCareerFetcher{
		progress: nil, // API silencieuse
		custom:   nil,
	}
	repo := runE2E(t, fetcher, e2eDBLast())

	for _, p := range repo.insertedPartials {
		if p.Rank != nil || p.CurrentXP != nil || p.BannerImageURL != nil {
			t.Errorf("INSERT polluté avec data alors que live était muet: %+v", p)
		}
	}
	// On peut avoir un INSERT status-only (api_empty), c'est OK : ça trace l'essai
	for _, p := range repo.insertedPartials {
		if p.LastFetchStatus != nil && *p.LastFetchStatus != "api_empty" {
			t.Errorf("status devrait être api_empty: got %q", *p.LastFetchStatus)
		}
	}
}

// Scénario 2 — live rend custom complet seul (progress muet) :
// → INSERT contient SpartanID/banner/emblem/backdrop, rank/xp restent NULL
func TestE2E_LiveCustomOnly_NoRankPollution(t *testing.T) {
	fetcher := &mockCareerFetcher{
		progress: nil,
		custom: &syncpkg.SpartanCustomizationData{
			SpartanID:        "NEW",
			BannerImageURL:   "https://live/banner.png",
			EmblemImageURL:   "https://live/emblem.png",
			BackdropImageURL: "https://live/backdrop.png",
		},
	}
	repo := runE2E(t, fetcher, e2eDBLast())

	if len(repo.insertedPartials) == 0 {
		t.Fatal("attendait un INSERT (custom OK)")
	}
	found := false
	for _, p := range repo.insertedPartials {
		if p.BannerImageURL == nil {
			continue
		}
		found = true
		if p.Rank != nil {
			t.Errorf("Rank ne devrait PAS être set (live progress muet): %+v", p)
		}
		if p.CurrentXP != nil {
			t.Errorf("CurrentXP ne devrait PAS être set: %+v", p)
		}
		if p.SpartanID == nil || *p.SpartanID != "NEW" {
			t.Errorf("SpartanID: %v", p.SpartanID)
		}
		if *p.BannerImageURL != "https://live/banner.png" {
			t.Errorf("BannerImageURL: %v", *p.BannerImageURL)
		}
		if p.LastFetchStatus == nil || *p.LastFetchStatus != "ok" {
			t.Errorf("status: %v", p.LastFetchStatus)
		}
	}
	if !found {
		t.Errorf("aucun INSERT avec banner: %+v", repo.insertedPartials)
	}
}

// Scénario 3 — live rend progress complet seul (custom muet) :
// → INSERT contient Rank/CurrentXP, banner/emblem restent NULL
func TestE2E_LiveProgressOnly_NoCustomPollution(t *testing.T) {
	fetcher := &mockCareerFetcher{
		progress: &syncpkg.CareerRankData{
			CurrentRank: 185,
			CurrentXP:   1234,
		},
		custom: nil,
	}
	repo := runE2E(t, fetcher, e2eDBLast())

	if len(repo.insertedPartials) == 0 {
		t.Fatal("attendait un INSERT (progress OK)")
	}
	found := false
	for _, p := range repo.insertedPartials {
		if p.Rank == nil {
			continue
		}
		found = true
		if *p.Rank != 185 {
			t.Errorf("Rank: got %d, want 185", *p.Rank)
		}
		if p.BannerImageURL != nil {
			t.Errorf("BannerImageURL ne devrait PAS être set (custom muet): %+v", p)
		}
		if p.SpartanID != nil {
			t.Errorf("SpartanID ne devrait PAS être set: %+v", p)
		}
		if p.EmblemImageURL != nil {
			t.Errorf("EmblemImageURL ne devrait PAS être set: %+v", p)
		}
	}
	if !found {
		t.Errorf("aucun INSERT avec rank: %+v", repo.insertedPartials)
	}
}

// Scénario 4 — live rend Rank=0 explicite (API a renvoyé "0" pas "absent") :
// → on n'écrit PAS Rank=0 (progressHasRealData filtre)
func TestE2E_LiveRankZeroExplicit_NoInsert(t *testing.T) {
	fetcher := &mockCareerFetcher{
		progress: &syncpkg.CareerRankData{
			CurrentRank: 0,
			CurrentXP:   0,
			IsMaxRank:   false,
		},
		custom: nil,
	}
	repo := runE2E(t, fetcher, e2eDBLast())

	for _, p := range repo.insertedPartials {
		if p.Rank != nil {
			t.Errorf("Rank=0 explicite ne devrait pas être écrit: %+v", p)
		}
		if p.CurrentXP != nil {
			t.Errorf("CurrentXP ne devrait pas être écrit: %+v", p)
		}
	}
}

// Scénario 5 — joueur début de palier (Rank>0 + CurrentXP=0 explicite) :
// → on écrit Rank ET CurrentXP=0 (vraie valeur du joueur)
func TestE2E_LiveBeginningOfTier_PersistsZeroXP(t *testing.T) {
	fetcher := &mockCareerFetcher{
		progress: &syncpkg.CareerRankData{
			CurrentRank: 185, // tout nouveau palier
			CurrentXP:   0,
		},
		custom: nil,
	}
	repo := runE2E(t, fetcher, e2eDBLast())

	if len(repo.insertedPartials) == 0 {
		t.Fatal("attendait un INSERT")
	}
	found := false
	for _, p := range repo.insertedPartials {
		if p.Rank != nil && p.CurrentXP != nil {
			found = true
			if *p.Rank != 185 {
				t.Errorf("Rank: got %d, want 185", *p.Rank)
			}
			if *p.CurrentXP != 0 {
				t.Errorf("CurrentXP=0 doit être préservé: got %d", *p.CurrentXP)
			}
		}
	}
	if !found {
		t.Errorf("aucun INSERT avec rank+xp: %+v", repo.insertedPartials)
	}
}

// Scénario 6 — un seul champ custom rendu (banner uniquement) :
// → INSERT contient banner, emblem/backdrop/spartan_id restent NULL
func TestE2E_LiveSingleCustomField_OnlyThatFieldWritten(t *testing.T) {
	fetcher := &mockCareerFetcher{
		progress: nil,
		custom: &syncpkg.SpartanCustomizationData{
			BannerImageURL: "https://live/banner-only.png",
			// emblem / backdrop / spartan_id : vides
		},
	}
	repo := runE2E(t, fetcher, e2eDBLast())

	found := false
	for _, p := range repo.insertedPartials {
		if p.BannerImageURL == nil {
			continue
		}
		found = true
		if *p.BannerImageURL != "https://live/banner-only.png" {
			t.Errorf("BannerImageURL: %v", *p.BannerImageURL)
		}
		if p.EmblemImageURL != nil {
			t.Errorf("EmblemImageURL devrait être NULL: %+v", p)
		}
		if p.BackdropImageURL != nil {
			t.Errorf("BackdropImageURL devrait être NULL: %+v", p)
		}
		if p.SpartanID != nil {
			t.Errorf("SpartanID devrait être NULL: %+v", p)
		}
	}
	if !found {
		t.Errorf("INSERT avec banner-only manquant: %+v", repo.insertedPartials)
	}
}

// Scénario 7 — la home doit TOUJOURS afficher la banner connue, même si live
// échoue / rend partiel / rend vide. C'est le contrat UI-first (overlay
// fallback).
func TestE2E_HomeAlwaysShowsKnownBanner(t *testing.T) {
	cases := []struct {
		name    string
		fetcher *mockCareerFetcher
	}{
		{"live silent", &mockCareerFetcher{}},
		{"live progress only", &mockCareerFetcher{progress: &syncpkg.CareerRankData{CurrentRank: 185, CurrentXP: 100}}},
		{"live custom partiel (banner vide)", &mockCareerFetcher{custom: &syncpkg.SpartanCustomizationData{SpartanID: "X"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockCareerLiveRepo{last: e2eDBLast(), insertPartialOK: true}
			builder := &mockIdentityBuilder{}
			cache := NewCareerLiveCache(CareerLiveCacheConfig{ProgressTTL: 5 * time.Minute, CustomizationTTL: 6 * time.Hour})
			factory := func(_ context.Context) CareerFetcher {
				if tc.fetcher == nil {
					return nil
				}
				return tc.fetcher
			}
			svc := NewCareerLiveService(repo, builder, factory, cache)

			identity, err := svc.GetSpartanIdentityFor(ctxWithTokens(t, true), ctxTokensXUID)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if identity == nil {
				t.Fatal("identity should never be nil when DB has known banner")
			}
			if identity.BannerImageURL == nil {
				t.Errorf("BannerImageURL should be served from DB fallback: %+v", identity)
			}
		})
	}
}

// Scénario 8 — auth absente → status auth_missing inséré pour diag,
// pas de fetch live tenté
func TestE2E_NoAuth_StatusOnlyInsertOptional(t *testing.T) {
	// Sans auth tokens, fetchAndMerge ne déclenche pas kickoffBackgroundRefresh.
	// Donc pas d'INSERT du tout. C'est OK — le diag log "kickoff skipped reason=no_auth_tokens"
	// suffit pour tracer.
	repo := &mockCareerLiveRepo{last: e2eDBLast(), insertPartialOK: true}
	builder := &mockIdentityBuilder{}
	cache := NewCareerLiveCache(CareerLiveCacheConfig{ProgressTTL: 5 * time.Minute, CustomizationTTL: 6 * time.Hour})
	factory := func(_ context.Context) CareerFetcher { return nil }
	svc := NewCareerLiveService(repo, builder, factory, cache)
	// Pas de tokens dans le ctx
	ctx := ctxkeys.WithHaloAuth(context.Background(), nil, ctxTokensXUID)

	identity, err := svc.GetSpartanIdentityFor(ctx, ctxTokensXUID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if identity == nil {
		t.Fatal("identity should be served from DB fallback")
	}
	if identity.BannerImageURL == nil {
		t.Errorf("BannerImageURL should come from DB fallback")
	}
	// On peut avoir 0 ou 1 INSERT, peu importe — le contrat principal est que
	// la home affiche la banner DB connue.
	_ = repo
}

// Sentinel : on s'assure que mockCareerFetcher utilisé ici existe.
var _ = (*mockCareerFetcher)(nil)
var _ = (*duckdb.CareerProgressionPartial)(nil)
var _ = domain.HomeSpartanIdentityRow{}
