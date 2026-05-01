// catalog_adapter_test.go — tests Phase D plan catalogue.
package halo_infinite

import (
	"context"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/canonical"
)

// stubFetcher implémente AssetFetcher pour les tests sans appel HTTP réel.
type stubFetcher struct {
	asset *DiscoveryAssetRaw
	err   error
}

func (s *stubFetcher) FetchAsset(_ context.Context, _ AssetType, _, _, _, _ string) (*DiscoveryAssetRaw, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.asset, nil
}

// rulesPath retourne le chemin absolu vers experience_rules.toml en partant
// du package halo_infinite (4 niveaux au-dessus pour atteindre le repo root).
func rulesPath(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join(
		"..", "..", "..", "..", "..",
		"config", "titles", "halo_infinite", "catalog", "experience_rules.toml",
	))
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	return abs
}

func TestNewCatalogAdapter_LoadsRules(t *testing.T) {
	a, err := NewCatalogAdapter(&stubFetcher{}, rulesPath(t))
	if err != nil {
		t.Fatalf("NewCatalogAdapter: %v", err)
	}
	if a.TitleSlug() != "halo_infinite" {
		t.Errorf("TitleSlug = %q", a.TitleSlug())
	}
	if len(a.experienceRules) == 0 {
		t.Error("aucune règle chargée depuis le TOML")
	}
}

func TestClassifyExperience(t *testing.T) {
	a, err := NewCatalogAdapter(&stubFetcher{}, rulesPath(t))
	if err != nil {
		t.Fatalf("NewCatalogAdapter: %v", err)
	}

	cases := []struct {
		playlist canonical.CanonicalPlaylist
		want     canonical.Experience
	}{
		{canonical.CanonicalPlaylist{NameCanonical: "Ranked Arena"}, canonical.ExperienceRanked},
		{canonical.CanonicalPlaylist{NameCanonical: "Ranked Slayer"}, canonical.ExperienceRanked},
		{canonical.CanonicalPlaylist{NameCanonical: "Big Team Battle"}, canonical.ExperienceBTB},
		{canonical.CanonicalPlaylist{NameCanonical: "BTB"}, canonical.ExperienceBTB},
		{canonical.CanonicalPlaylist{NameCanonical: "BTB Heavies"}, canonical.ExperienceBTB},
		{canonical.CanonicalPlaylist{NameCanonical: "Firefight: King of the Hill"}, canonical.ExperienceFirefight},
		{canonical.CanonicalPlaylist{NameCanonical: "Husky Raid"}, canonical.ExperienceActionSack},
		{canonical.CanonicalPlaylist{NameCanonical: "Tactical Slayer"}, canonical.ExperienceActionSack},
		{canonical.CanonicalPlaylist{NameCanonical: "Super Fiesta"}, canonical.ExperienceLimitedTime},
		{canonical.CanonicalPlaylist{NameCanonical: "Quick Play"}, canonical.ExperienceSocial},
		{canonical.CanonicalPlaylist{NameCanonical: "Team Slayer"}, canonical.ExperienceSocial},
		{canonical.CanonicalPlaylist{NameCanonical: "<UUID-asset-non-résolu>"}, canonical.ExperienceUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.playlist.NameCanonical, func(t *testing.T) {
			got := a.ClassifyExperience(tc.playlist)
			if got != tc.want {
				t.Errorf("ClassifyExperience(%q) = %q, want %q",
					tc.playlist.NameCanonical, got, tc.want)
			}
		})
	}
}

func TestClassifyExperience_RankedFlag(t *testing.T) {
	a, err := NewCatalogAdapter(&stubFetcher{}, rulesPath(t))
	if err != nil {
		t.Fatalf("NewCatalogAdapter: %v", err)
	}
	// Une playlist non nommée "Ranked X" mais avec is_ranked=true doit aussi tomber sur ranked.
	pl := canonical.CanonicalPlaylist{NameCanonical: "Custom Squad", IsRanked: true}
	if got := a.ClassifyExperience(pl); got != canonical.ExperienceRanked {
		t.Errorf("ClassifyExperience(IsRanked=true) = %q, want ranked", got)
	}
}

func TestFetchPlaylist_UsesFetcherAndClassifies(t *testing.T) {
	fetcher := &stubFetcher{
		asset: &DiscoveryAssetRaw{
			AssetID: "p-1", VersionID: "v-1",
			PublicName: "Big Team Battle",
		},
	}
	a, err := NewCatalogAdapter(fetcher, rulesPath(t))
	if err != nil {
		t.Fatalf("NewCatalogAdapter: %v", err)
	}
	pl, err := a.FetchPlaylist(context.Background(), "p-1", "v-1")
	if err != nil {
		t.Fatalf("FetchPlaylist: %v", err)
	}
	if pl.AssetID != "p-1" {
		t.Errorf("AssetID = %q", pl.AssetID)
	}
	if pl.NameCanonical != "Big Team Battle" {
		t.Errorf("NameCanonical = %q", pl.NameCanonical)
	}
	if pl.Experience != canonical.ExperienceBTB {
		t.Errorf("Experience = %q, want btb", pl.Experience)
	}
	if pl.Names["en"] != "Big Team Battle" {
		t.Errorf("Names[en] = %q", pl.Names["en"])
	}
}

func TestFetchPair_ProducesModeCategoryAndLabel(t *testing.T) {
	fetcher := &stubFetcher{
		asset: &DiscoveryAssetRaw{
			AssetID: "pair-1", VersionID: "v-1",
			PublicName: "Arena:Slayer on Bazaar",
		},
	}
	a, err := NewCatalogAdapter(fetcher, rulesPath(t))
	if err != nil {
		t.Fatalf("NewCatalogAdapter: %v", err)
	}
	pair, err := a.FetchPair(context.Background(), "pair-1", "v-1")
	if err != nil {
		t.Fatalf("FetchPair: %v", err)
	}
	if pair.ModeCategory != ModeCategoryAssassin {
		t.Errorf("ModeCategory = %q, want %q", pair.ModeCategory, ModeCategoryAssassin)
	}
	if pair.ModeLabels["en"] == "" {
		t.Error("ModeLabels[en] vide — attendu non-vide via NormalizeModeLabel")
	}
}

func TestClassifyModeCanonical(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		want canonical.ModeCanonical
	}{
		{"Slayer", canonical.ModeSlayer},
		{"Team Slayer", canonical.ModeSlayer},
		{"CTF", canonical.ModeCTF},
		{"Capture the Flag", canonical.ModeCTF},
		{"Oddball", canonical.ModeOddball},
		{"King of the Hill", canonical.ModeKOTH},
		{"Strongholds", canonical.ModeStrongholds},
		{"Fiesta Slayer", canonical.ModeSlayer}, // slayer match avant fiesta (ordre du switch)
		{"Firefight: KOTR", canonical.ModeFirefightKOTR},
		{"Total Control", canonical.ModeTotalControl},
		{"Random Mode XYZ", canonical.ModeUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyModeCanonical(tc.name); got != tc.want {
				t.Errorf("classifyModeCanonical(%q) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}
