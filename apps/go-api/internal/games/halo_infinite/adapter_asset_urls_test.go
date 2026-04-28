package halo_infinite

import "testing"

func TestAssetURLAdapter_TitleSlug(t *testing.T) {
	t.Parallel()
	a := NewAssetURLAdapterWithFlag(false)
	if a.TitleSlug() != "halo_infinite" {
		t.Errorf("TitleSlug = %q, want halo_infinite", a.TitleSlug())
	}
}

func TestAssetURLAdapter_MapImageURL_Flat(t *testing.T) {
	t.Parallel()
	a := NewAssetURLAdapterWithFlag(false)
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"png map known", "Aquarius", "/static/maps/Aquarius.png"},
		{"jpg map (default ext)", "Argyle", "/static/maps/Argyle.jpg"},
		{"png map with hyphen+space", "Streets - Ranked", "/static/maps/Streets%20-%20Ranked.png"},
		{"map with multiple spaces", "Highpower Sentry Defense", "/static/maps/Highpower%20Sentry%20Defense.png"},
		{"empty → empty", "", ""},
		{"whitespace only → empty", "   ", ""},
		{"UUID → empty", "12345678-abcd-1234-9876-1234567890ab", ""},
		{"UUID upper → empty", "12345678-ABCD-1234-9876-1234567890AB", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := a.MapImageURL(c.in); got != c.want {
				t.Errorf("MapImageURL(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestAssetURLAdapter_MapImageURL_TitleScoped(t *testing.T) {
	t.Parallel()
	a := NewAssetURLAdapterWithFlag(true)
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"png map known", "Aquarius", "/static/maps/halo_infinite/Aquarius.png"},
		{"jpg map (default ext)", "Argyle", "/static/maps/halo_infinite/Argyle.jpg"},
		{"png map with hyphen+space", "Streets - Ranked", "/static/maps/halo_infinite/Streets%20-%20Ranked.png"},
		{"empty → empty", "", ""},
		{"UUID → empty", "12345678-abcd-1234-9876-1234567890ab", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := a.MapImageURL(c.in); got != c.want {
				t.Errorf("MapImageURL(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestAssetURLAdapter_MedalImageURL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		titleScoped bool
		medalID     uint64
		want        string
	}{
		{"flat numeric", false, 12345, "/static/medals/icons/12345.png"},
		{"flat zero", false, 0, "/static/medals/icons/0.png"},
		{"title-scoped", true, 12345, "/static/medals/icons/halo_infinite/12345.png"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := NewAssetURLAdapterWithFlag(c.titleScoped)
			if got := a.MedalImageURL(c.medalID); got != c.want {
				t.Errorf("MedalImageURL(%d) = %q, want %q", c.medalID, got, c.want)
			}
		})
	}
}

func TestAssetURLAdapter_CSRRankImageURL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		titleScoped bool
		tier        string
		subTier     int
		want        string
	}{
		{"flat gold 3", false, "Gold", 3, "/static/ranks/120px-HINF-CSR_Gold3.png"},
		{"flat platinum 5", false, "Platinum", 5, "/static/ranks/120px-HINF-CSR_Platinum5.png"},
		{"title-scoped diamond 1", true, "Diamond", 1, "/static/ranks/halo_infinite/120px-HINF-CSR_Diamond1.png"},
		{"empty tier → empty", false, "", 1, ""},
		{"whitespace tier → empty", false, "  ", 1, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := NewAssetURLAdapterWithFlag(c.titleScoped)
			if got := a.CSRRankImageURL(c.tier, c.subTier); got != c.want {
				t.Errorf("CSRRankImageURL(%q, %d) = %q, want %q", c.tier, c.subTier, got, c.want)
			}
		})
	}
}

func TestAssetURLAdapter_CSRRankImageURLOnyx(t *testing.T) {
	t.Parallel()
	flat := NewAssetURLAdapterWithFlag(false)
	if got, want := flat.CSRRankImageURLOnyx(), "/static/ranks/120px-HINF-CSR_Onyx.png"; got != want {
		t.Errorf("flat CSRRankImageURLOnyx() = %q, want %q", got, want)
	}
	scoped := NewAssetURLAdapterWithFlag(true)
	if got, want := scoped.CSRRankImageURLOnyx(), "/static/ranks/halo_infinite/120px-HINF-CSR_Onyx.png"; got != want {
		t.Errorf("scoped CSRRankImageURLOnyx() = %q, want %q", got, want)
	}
}

func TestAssetURLAdapter_FlagFromEnv(t *testing.T) {
	// Default depuis Phase 6.5 : title-scoped activé sauf ENV explicit "false".
	t.Setenv(EnvTitleScopedFlag, "true")
	a := NewAssetURLAdapter()
	if !a.titleScoped {
		t.Errorf("titleScoped should be true when ENV=true")
	}

	t.Setenv(EnvTitleScopedFlag, "false")
	a2 := NewAssetURLAdapter()
	if a2.titleScoped {
		t.Errorf("titleScoped should be false when ENV=false (rollback path)")
	}

	t.Setenv(EnvTitleScopedFlag, "")
	a3 := NewAssetURLAdapter()
	if !a3.titleScoped {
		t.Errorf("titleScoped should default to true when ENV unset (Phase 6.5+)")
	}
}

func TestEncodeSpaces(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"Streets", "Streets"},
		{"Live Fire", "Live%20Fire"},
		{"a b c d", "a%20b%20c%20d"},
		{"", ""},
		{"   ", "%20%20%20"},
		{"hyphen-no-space", "hyphen-no-space"},
	}
	for _, c := range cases {
		if got := encodeSpaces(c.in); got != c.want {
			t.Errorf("encodeSpaces(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
