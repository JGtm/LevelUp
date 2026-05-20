// Package handlers — health_home_test.go : tests unitaires du smoke endpoint
// /api/v1/healthz/home. Couvre les 4 verdicts critiques :
//   - 400 si le param 'player' est absent
//   - 404 si player_slug inconnu (factory en erreur)
//   - 200 + ok=true quand toutes les sections sont peuplées
//   - 503 + empty_sections quand au moins une section critique est vide
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// fakeHomeService implémente port.HomeService minimalement pour les tests.
// Seules les méthodes appelées par le smoke endpoint (GetHomePage) sont
// implémentées ; les autres panic pour repérer rapidement un nouveau caller.
type fakeHomeService struct {
	resp *domain.HomePageResponse
	err  error
}

func (f *fakeHomeService) GetHomePage(_ context.Context, _, _ string) (*domain.HomePageResponse, error) {
	return f.resp, f.err
}
func (f *fakeHomeService) GetBattlePass(_ context.Context) domain.BattlePassResponse {
	panic("not used in healthz tests")
}
func (f *fakeHomeService) GetChallenges(_ context.Context) domain.ChallengesResponse {
	panic("not used in healthz tests")
}
func (f *fakeHomeService) SetSessionActive(_ bool)                  {}
func (f *fakeHomeService) RefreshTrack(_ context.Context, _ string) {}

func buildFakeFactory(svc port.HomeService, factoryErr error) HomeAuthFactory {
	return func(ctx context.Context, slug string) (port.HomeService, context.Context, string, string, error) {
		if factoryErr != nil {
			return nil, ctx, "", "", factoryErr
		}
		return svc, ctx, "xuid-" + slug, slug, nil
	}
}

func TestHealthHome_MissingPlayerParam_Returns400(t *testing.T) {
	h := NewHealthHomeHandler(buildFakeFactory(&fakeHomeService{}, nil))
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/healthz/home", nil)
	h.Check(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (missing player param)", w.Code)
	}
}

func TestHealthHome_UnknownPlayer_Returns404(t *testing.T) {
	h := NewHealthHomeHandler(buildFakeFactory(nil, errors.New("player_not_found_in_db_profiles")))
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/healthz/home?player=ghost", nil)
	h.Check(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHealthHome_AllSectionsPopulated_Returns200(t *testing.T) {
	bannerURL := "/api/v1/assets/spartan/banner/halo_infinite/x.png"
	zero := 0
	resp := &domain.HomePageResponse{
		Hero: domain.HomeHeroCard{
			KPIs: domain.HeroKPIs{FavoriteWeaponName: "BR75", FavoriteWeaponKills: 1500},
		},
		SpartanIdentity: &domain.HomeSpartanIdentity{
			BannerImageURL: &bannerURL,
			HighestCSR:     &domain.HomeSkillPeakSummary{RatingValue: 1450, MeasurementMatchesRemaining: &zero},
			HighestLUSR:    &domain.HomeSkillPeakSummary{RatingValue: 1750, MeasurementMatchesRemaining: &zero},
		},
		RecentPlaylistRanks: []domain.HomePlaylistRank{{PlaylistName: "Ranked Arena", IsRanked: true}},
	}

	h := NewHealthHomeHandler(buildFakeFactory(&fakeHomeService{resp: resp}, nil))
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/healthz/home?player=jgtm", nil)
	h.Check(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["ok"] != true {
		t.Errorf("ok = %v, want true", body["ok"])
	}
	if got := body["empty_sections"].([]any); len(got) != 0 {
		t.Errorf("empty_sections = %v, want empty", got)
	}
}

func TestHealthHome_EmptySections_Returns503WithList(t *testing.T) {
	// Cas régression : aucune section peuplée → 503 + 5 sections listées.
	resp := &domain.HomePageResponse{
		Hero:                domain.HomeHeroCard{KPIs: domain.HeroKPIs{FavoriteWeaponName: ""}},
		SpartanIdentity:     &domain.HomeSpartanIdentity{},
		RecentPlaylistRanks: []domain.HomePlaylistRank{},
	}

	h := NewHealthHomeHandler(buildFakeFactory(&fakeHomeService{resp: resp}, nil))
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/healthz/home?player=jgtm", nil)
	h.Check(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503, body=%s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["ok"] != false {
		t.Errorf("ok = %v, want false", body["ok"])
	}
	empty, _ := body["empty_sections"].([]any)
	expectedSections := map[string]bool{
		"banner":           true,
		"highest_csr":      true,
		"highest_lusr":     true,
		"recent_playlists": true,
		"favorite_weapon":  true,
	}
	if len(empty) != len(expectedSections) {
		t.Errorf("len(empty_sections) = %d, want %d. Got: %v", len(empty), len(expectedSections), empty)
	}
	for _, s := range empty {
		key, _ := s.(string)
		if !expectedSections[key] {
			t.Errorf("unexpected empty section reported: %q", key)
		}
	}
}

func TestHealthHome_PlacementCounts_AsOK(t *testing.T) {
	// En placement (measurement_matches_remaining > 0), le check doit OK :
	// le peak existe avec un badge unranked_N.png. Ce n'est pas un trou de
	// données, c'est la réalité du joueur.
	bannerURL := "/api/v1/assets/spartan/banner/halo_infinite/x.png"
	rem := 7
	resp := &domain.HomePageResponse{
		Hero: domain.HomeHeroCard{KPIs: domain.HeroKPIs{FavoriteWeaponName: "BR75"}},
		SpartanIdentity: &domain.HomeSpartanIdentity{
			BannerImageURL: &bannerURL,
			HighestCSR:     &domain.HomeSkillPeakSummary{RatingValue: 0, MeasurementMatchesRemaining: &rem},
			HighestLUSR:    &domain.HomeSkillPeakSummary{RatingValue: 0, MeasurementMatchesRemaining: &rem},
		},
		RecentPlaylistRanks: []domain.HomePlaylistRank{{PlaylistName: "Ranked Arena", IsRanked: true}},
	}

	h := NewHealthHomeHandler(buildFakeFactory(&fakeHomeService{resp: resp}, nil))
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/healthz/home?player=jgtm", nil)
	h.Check(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (placement = OK), body=%s", w.Code, w.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	checks, _ := body["checks"].(map[string]any)
	if got := checks["highest_csr"]; got != "ok (placement)" {
		t.Errorf("checks.highest_csr = %v, want 'ok (placement)'", got)
	}
}
