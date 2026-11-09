package haloadapter

// client_career_test.go — rang carrière et apparence du client de synchronisation (tests du client
// d'avant la brique, portés sous leur nom).

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetCareerRank_EmptyXUID(t *testing.T) {
	c := NewClient("s", "c", 10)
	_, err := c.GetCareerRank(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty xuid")
	}
}

// TestGetSpartanCustomization_PublicViewFallback vérifie le fallback ajouté pour
// les joueurs TIERS (cas Explorer) : `/customization/appearance` est player-gated
// (403 pour un xuid non propriétaire) ; on retombe alors sur la vue publique
// `/customization?view=public`, qui expose le même bloc Appearance pour n'importe
// quel joueur, parsée par le même parseCustomizationAppearance.
func TestGetSpartanCustomization_PublicViewFallback(t *testing.T) {
	var appearanceCalls, publicCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/customization/appearance"):
			appearanceCalls++
			w.WriteHeader(http.StatusForbidden) // player-gated pour un xuid tiers
		case strings.HasSuffix(r.URL.Path, "/customization") && r.URL.Query().Get("view") == "public":
			publicCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Appearance": map[string]any{
					"ServiceTag":        "MELG",
					"BackdropImagePath": "Inventory/Spartan/BackdropImages/pub-backdrop.json",
					"Emblem": map[string]any{
						"EmblemPath":      "Inventory/Spartan/Emblems/pub-emblem.json",
						"ConfigurationId": 511768825,
					},
				},
			})
		case strings.Contains(r.URL.Path, "/hi/progression/file/Inventory/Spartan/BackdropImages/"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"CommonData": map[string]any{"DisplayPath": map[string]any{
					"Media": map[string]any{"MediaUrl": map[string]any{"Path": "progression/Backdrops/pub-backdrop.png"}}}},
			})
		case strings.Contains(r.URL.Path, "/hi/progression/file/Inventory/Spartan/Emblems/"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"CommonData": map[string]any{"DisplayPath": map[string]any{
					"Media": map[string]any{"MediaUrl": map[string]any{"Path": "progression/Emblems/pub-emblem.png"}}}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := &Client{
		http:           srv.Client(),
		spartanToken:   "s",
		clearanceToken: "c",
		economyBaseURL: srv.URL,
		gameCMSBaseURL: srv.URL,
		limiter:        fastLimiter(),
	}

	data, err := c.GetSpartanCustomization(context.Background(), "2535427927026623")
	if err != nil {
		t.Fatalf("GetSpartanCustomization: %v", err)
	}
	if data == nil {
		t.Fatal("customisation attendue non-nil via la vue publique")
	}
	if appearanceCalls == 0 {
		t.Error("/customization/appearance jamais tenté")
	}
	if publicCalls == 0 {
		t.Error("fallback /customization?view=public jamais déclenché")
	}
	if data.SpartanID != "MELG" {
		t.Errorf("SpartanID = %q, attendu MELG", data.SpartanID)
	}
	if data.EmblemImageURL == "" {
		t.Error("EmblemImageURL attendu non-vide (résolu via la vue publique)")
	}
	if data.BackdropImageURL == "" {
		t.Error("BackdropImageURL attendu non-vide (résolu via la vue publique)")
	}
}

func TestGetCareerRank_UsesCareerAndCustomizationEndpoints(t *testing.T) {
	var careerCalls, customizationCalls, progressionCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		// 2026-05-14 : nouveau path `/hi/careerranks/careerRank1?players=xuid(X)`
		// renvoie un RewardTrackResultContainer (parser gère les deux formes).
		case strings.HasPrefix(r.URL.Path, "/hi/careerranks/careerRank1"):
			careerCalls++
			if got := r.URL.Query().Get("players"); got == "" {
				t.Fatalf("missing players query param")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"RewardTracks": []map[string]any{
					{
						"Result": map[string]any{
							"CurrentProgress": map[string]any{
								"Rank":              174,
								"PartialProgress":   21840,
								"HasReachedMaxRank": false,
							},
						},
					},
				},
			})
		// 2026-05-14 : nouveau path `/customization/appearance` (sans query).
		case strings.HasSuffix(r.URL.Path, "/customization/appearance"):
			customizationCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Appearance": map[string]any{
					"ServiceTag":         "JGTM",
					"NameplateImagePath": "Inventory/Spartan/Nameplates/test-banner.json",
					"BackdropImagePath":  "Inventory/Spartan/BackdropImages/test-backdrop.json",
					"Emblem": map[string]any{
						"EmblemPath":      "Inventory/Spartan/Emblems/test-emblem.json",
						"ConfigurationId": 987654,
					},
				},
			})
		case strings.Contains(r.URL.Path, "/hi/progression/file/Inventory/Spartan/Nameplates/"):
			progressionCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"CommonData": map[string]any{
					"DisplayPath": map[string]any{
						"Media": map[string]any{
							"MediaUrl": map[string]any{"Path": "progression/Nameplates/test-banner.png"},
						},
					},
				},
			})
		case strings.Contains(r.URL.Path, "/hi/progression/file/Inventory/Spartan/BackdropImages/"):
			progressionCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"CommonData": map[string]any{
					"DisplayPath": map[string]any{
						"Media": map[string]any{
							"MediaUrl": map[string]any{"Path": "progression/Backdrops/test-backdrop.png"},
						},
					},
				},
			})
		case strings.Contains(r.URL.Path, "/hi/progression/file/Inventory/Spartan/Emblems/"):
			progressionCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"CommonData": map[string]any{
					"DisplayPath": map[string]any{
						"Media": map[string]any{
							"MediaUrl": map[string]any{"Path": "progression/Emblems/test-emblem.png"},
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := &Client{
		http:           srv.Client(),
		spartanToken:   "s",
		clearanceToken: "c",
		economyBaseURL: srv.URL,
		gameCMSBaseURL: srv.URL,
		limiter:        fastLimiter(),
	}

	data, err := c.GetCareerRank(context.Background(), "2535469190789936")
	if err != nil {
		t.Fatalf("GetCareerRank: %v", err)
	}
	if data == nil {
		t.Fatal("GetCareerRank returned nil")
	}
	if data.CurrentRank != 174 || data.CurrentXP != 21840 {
		t.Fatalf("unexpected career data: %+v", data)
	}
	if data.SpartanID != "JGTM" {
		t.Fatalf("SpartanID = %q", data.SpartanID)
	}
	if data.BannerImageURL != srv.URL+"/hi/images/file/progression/Nameplates/test-banner.png" {
		t.Fatalf("BannerImageURL = %q", data.BannerImageURL)
	}
	if data.EmblemImageURL != srv.URL+"/hi/images/file/progression/Emblems/test-emblem.png" {
		t.Fatalf("EmblemImageURL = %q", data.EmblemImageURL)
	}
	if data.BackdropImageURL != srv.URL+"/hi/images/file/progression/Backdrops/test-backdrop.png" {
		t.Fatalf("BackdropImageURL = %q", data.BackdropImageURL)
	}
	if careerCalls != 1 || customizationCalls != 1 || progressionCalls != 3 {
		t.Fatalf(
			"careerCalls=%d customizationCalls=%d progressionCalls=%d",
			careerCalls,
			customizationCalls,
			progressionCalls,
		)
	}
}

// TestGetCareerRank_BannerDerivedFromEmblemNameplate vérifie le comportement
// après le port Python (2026-05-20) : quand l'API ne retourne pas de
// BannerImagePath direct mais qu'un Emblem + ConfigurationId sont présents,
// `ResolveNameplateURL` construit une URL nameplate dérivée
// (`/hi/Waypoint/file/images/nameplates/<emblem_stem>_<cfg>.png`).
//
// Avant cette fix (commit 22cb84d5, 8 mai 2026) : BannerImageURL=="" (front
// affichait placeholder, suite à 403 upstream avec cfg négatif). Maintenant :
// si cfg>0, URL directe ; si cfg<=0, resolver fetche JSON emblem CMS pour
// trouver un cfg positif valide (port `resolve_positive_emblem_cfg` Python).
//
// L'EmblemImageURL reste résolu via le pattern Grunt strict (descriptor
// JSON → GameCms_GetProgressionImage).
func TestGetCareerRank_BannerDerivedFromEmblemNameplate(t *testing.T) {
	var customizationCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		// 2026-05-14 : nouveaux endpoints Halo Waypoint (cf. parent test).
		case strings.HasPrefix(r.URL.Path, "/hi/careerranks/careerRank1"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"RewardTracks": []map[string]any{
					{
						"Result": map[string]any{
							"CurrentProgress": map[string]any{
								"Rank":              177,
								"PartialProgress":   10100,
								"HasReachedMaxRank": false,
							},
						},
					},
				},
			})
		case strings.HasSuffix(r.URL.Path, "/customization/appearance"):
			customizationCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Appearance": map[string]any{
					"ServiceTag":      "OKLM",
					"PlayerTitlePath": nil,
					"Emblem": map[string]any{
						"EmblemPath":      "Inventory/Spartan/Emblems/104-001-343other-prop-79c5fbd5.json",
						"ConfigurationId": 372285867,
					},
				},
			})
		case strings.Contains(r.URL.Path, "/hi/progression/file/Inventory/Spartan/Emblems/"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"CommonData": map[string]any{
					"DisplayPath": map[string]any{
						"Media": map[string]any{
							"MediaUrl": map[string]any{"Path": "progression/Inventory/Emblems/343other_propaganda_emblem.png"},
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := &Client{
		http:           srv.Client(),
		spartanToken:   "s",
		clearanceToken: "c",
		economyBaseURL: srv.URL,
		gameCMSBaseURL: srv.URL,
		limiter:        fastLimiter(),
	}

	data, err := c.GetCareerRank(context.Background(), "2533274823110022")
	if err != nil {
		t.Fatalf("GetCareerRank: %v", err)
	}
	if data == nil {
		t.Fatal("GetCareerRank returned nil")
	}
	// cfg=372285867 > 0 → URL nameplate construite directement (pas d'appel
	// CMS pour resolve_positive_emblem_cfg). Le test cadenasse le format.
	wantBanner := "https://gamecms-hacs.svc.halowaypoint.com/hi/Waypoint/file/images/nameplates/104-001-343other-prop-79c5fbd5_372285867.png"
	if data.BannerImageURL != wantBanner {
		t.Errorf("BannerImageURL = %q, want %q (nameplate dérivé emblem+cfg)", data.BannerImageURL, wantBanner)
	}
	if data.EmblemImageURL != srv.URL+"/hi/images/file/progression/Inventory/Emblems/343other_propaganda_emblem.png" {
		t.Errorf("EmblemImageURL = %q (Grunt strict pattern attendu)", data.EmblemImageURL)
	}
	if customizationCalls != 1 {
		t.Errorf("customizationCalls=%d, want 1", customizationCalls)
	}
}
