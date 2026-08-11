package halo_5

import (
	"testing"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

// Vérifie statiquement que AssetURLAdapter satisfait l'interface.
var _ games.TitleAssetURLAdapter = (*AssetURLAdapter)(nil)

func newTestAdapter() *AssetURLAdapter {
	return NewAssetURLAdapter().
		WithMaps([]canonical.AssetMeta{
			{ID: "guid-truth", NameEN: "The Rig", ImageURL: "https://cdn/maps/the-rig.png"},
			{ID: "guid-noimg", NameEN: "NoImage", ImageURL: ""},
		}).
		WithWeapons([]canonical.AssetMeta{
			{ID: "11", NameEN: "Battle Rifle", ImageURL: "https://cdn/weapons/br.png"},
			{ID: "12", NameEN: "NoIconWeapon", ImageURL: ""},
			{ID: "pas-un-entier", NameEN: "Broken ID", ImageURL: "https://cdn/weapons/x.png"},
		}).
		WithCSRResolver(func(designation string, subTier int) string {
			switch {
			// Onyx est stocké à tier_id=1 dans csr_designations (sous-paliers
			// 1-indexés) — le résolveur réel est keyé `onyx|1`, pas `onyx|0`.
			case designation == "Onyx" && subTier == 1:
				return "https://cdn/csr/onyx.png"
			case designation == "Diamond" && subTier == 3:
				return "https://cdn/csr/diamond-3.png"
			default:
				return ""
			}
		})
}

func TestAssetURLAdapter_TitleSlug(t *testing.T) {
	if got := NewAssetURLAdapter().TitleSlug(); got != "halo_5" {
		t.Fatalf("TitleSlug = %q, want halo_5", got)
	}
}

func TestAssetURLAdapter_MapImageURL(t *testing.T) {
	a := newTestAdapter()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"par GUID (identifiant réellement passé par le header)", "guid-truth", "https://cdn/maps/the-rig.png"},
		{"par nom canonique en repli", "The Rig", "https://cdn/maps/the-rig.png"},
		{"GUID inconnu", "guid-unknown", ""},
		{"nom inconnu", "Unknown Map", ""},
		{"vide", "", ""},
		{"espaces autour", "  guid-truth  ", "https://cdn/maps/the-rig.png"},
		{"map sans image ignorée (ID)", "guid-noimg", ""},
		{"map sans image ignorée (nom)", "NoImage", ""},
	}
	for _, tc := range tests {
		if got := a.MapImageURL(tc.in); got != tc.want {
			t.Errorf("%s: MapImageURL(%q) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
	}
}

func TestAssetURLAdapter_WeaponImageURL(t *testing.T) {
	a := newTestAdapter()
	if got := a.WeaponImageURL(11); got != "https://cdn/weapons/br.png" {
		t.Errorf("WeaponImageURL(11) = %q, want https://cdn/weapons/br.png", got)
	}
	if got := a.WeaponImageURL(999); got != "" {
		t.Errorf("WeaponImageURL(999) = %q, want \"\" (id inconnu)", got)
	}
	if got := a.WeaponImageURL(0); got != "" {
		t.Errorf("WeaponImageURL(0) = %q, want \"\" (aucune sentinelle Halo 5)", got)
	}
	if got := a.WeaponImageURL(12); got != "" {
		t.Errorf("WeaponImageURL(12) = %q, want \"\" (icon_url vide ignorée)", got)
	}
}

func TestAssetURLAdapter_MedalImageURL_Empty(t *testing.T) {
	// Halo 5 médailles = sprite, pas de PNG par-médaille → toujours "".
	if got := newTestAdapter().MedalImageURL(123); got != "" {
		t.Errorf("MedalImageURL = %q, want \"\" (sprite, chantier distinct)", got)
	}
}

func TestAssetURLAdapter_CSRRankImageURL(t *testing.T) {
	a := newTestAdapter()
	if got := a.CSRRankImageURL("Diamond", 3); got != "https://cdn/csr/diamond-3.png" {
		t.Errorf("CSRRankImageURL(Diamond,3) = %q, want https://cdn/csr/diamond-3.png", got)
	}
	if got := a.CSRRankImageURLOnyx(); got != "https://cdn/csr/onyx.png" {
		t.Errorf("CSRRankImageURLOnyx = %q, want https://cdn/csr/onyx.png", got)
	}
	if got := a.CSRRankImageURL("Bronze", 1); got != "" {
		t.Errorf("CSRRankImageURL(Bronze,1) = %q, want \"\" (inconnu)", got)
	}
}

func TestAssetURLAdapter_TeamName(t *testing.T) {
	a := NewAssetURLAdapter().WithTeamNameResolver(func(teamID int, locale string) string {
		names := map[int]map[string]string{
			0: {"en": "Red", "fr": "Rouge"},
			1: {"en": "Blue", "fr": "Bleu"},
		}
		if m, ok := names[teamID]; ok {
			return m[locale]
		}
		return ""
	})
	if got := a.TeamName(0, "fr"); got != "Rouge" {
		t.Errorf("TeamName(0,fr) = %q, want Rouge", got)
	}
	if got := a.TeamName(1, "en"); got != "Blue" {
		t.Errorf("TeamName(1,en) = %q, want Blue", got)
	}
	if got := a.TeamName(9, "fr"); got != "" {
		t.Errorf("TeamName(9,fr) = %q, want \"\" (team inconnu)", got)
	}
	// Sans résolveur injecté → "" (nil-safe, dégradation gracieuse).
	if got := NewAssetURLAdapter().TeamName(0, "fr"); got != "" {
		t.Errorf("TeamName sans résolveur = %q, want \"\"", got)
	}
}

func TestAssetURLAdapter_TeamColor(t *testing.T) {
	a := NewAssetURLAdapter().WithTeamColorResolver(func(teamID int) string {
		colors := map[int]string{0: "#b00000", 1: "#178dd8"}
		return colors[teamID]
	})
	if got := a.TeamColor(0); got != "#b00000" {
		t.Errorf("TeamColor(0) = %q, want #b00000", got)
	}
	if got := a.TeamColor(1); got != "#178dd8" {
		t.Errorf("TeamColor(1) = %q, want #178dd8", got)
	}
	if got := a.TeamColor(9); got != "" {
		t.Errorf("TeamColor(9) = %q, want \"\" (team inconnu)", got)
	}
	// Sans résolveur injecté → "" (nil-safe, dégradation gracieuse).
	if got := NewAssetURLAdapter().TeamColor(0); got != "" {
		t.Errorf("TeamColor sans résolveur = %q, want \"\"", got)
	}
}

func TestAssetURLAdapter_NilCSRResolver(t *testing.T) {
	// Sans résolveur injecté, les méthodes CSR dégradent en "" (nil-safe).
	a := NewAssetURLAdapter()
	if got := a.CSRRankImageURL("Diamond", 3); got != "" {
		t.Errorf("CSRRankImageURL sans résolveur = %q, want \"\"", got)
	}
	if got := a.CSRRankImageURLOnyx(); got != "" {
		t.Errorf("CSRRankImageURLOnyx sans résolveur = %q, want \"\"", got)
	}
}
