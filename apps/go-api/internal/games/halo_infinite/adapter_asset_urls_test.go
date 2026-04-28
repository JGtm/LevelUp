package halo_infinite

import "testing"

func TestAssetURLAdapter_TitleSlug(t *testing.T) {
	t.Parallel()
	a := NewAssetURLAdapter()
	if a.TitleSlug() != "halo_infinite" {
		t.Errorf("TitleSlug = %q, want halo_infinite", a.TitleSlug())
	}
}

func TestAssetURLAdapter_MapImageURL(t *testing.T) {
	t.Parallel()
	a := NewAssetURLAdapter()
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"png map known", "Aquarius", "/static/maps/halo_infinite/Aquarius.png"},
		{"jpg map (default ext)", "Argyle", "/static/maps/halo_infinite/Argyle.jpg"},
		{"png map with hyphen+space", "Streets - Ranked", "/static/maps/halo_infinite/Streets%20-%20Ranked.png"},
		{"map with multiple spaces", "Highpower Sentry Defense", "/static/maps/halo_infinite/Highpower%20Sentry%20Defense.png"},
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

func TestAssetURLAdapter_MedalImageURL(t *testing.T) {
	t.Parallel()
	a := NewAssetURLAdapter()
	cases := []struct {
		name    string
		medalID uint64
		want    string
	}{
		{"numeric", 12345, "/static/medals/icons/halo_infinite/12345.png"},
		{"zero", 0, "/static/medals/icons/halo_infinite/0.png"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := a.MedalImageURL(c.medalID); got != c.want {
				t.Errorf("MedalImageURL(%d) = %q, want %q", c.medalID, got, c.want)
			}
		})
	}
}

func TestAssetURLAdapter_CSRRankImageURL(t *testing.T) {
	t.Parallel()
	a := NewAssetURLAdapter()
	cases := []struct {
		name    string
		tier    string
		subTier int
		want    string
	}{
		{"gold 3", "Gold", 3, "/static/ranks/halo_infinite/120px-HINF-CSR_Gold3.png"},
		{"platinum 5", "Platinum", 5, "/static/ranks/halo_infinite/120px-HINF-CSR_Platinum5.png"},
		{"diamond 1", "Diamond", 1, "/static/ranks/halo_infinite/120px-HINF-CSR_Diamond1.png"},
		{"empty tier → empty", "", 1, ""},
		{"whitespace tier → empty", "  ", 1, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := a.CSRRankImageURL(c.tier, c.subTier); got != c.want {
				t.Errorf("CSRRankImageURL(%q, %d) = %q, want %q", c.tier, c.subTier, got, c.want)
			}
		})
	}
}

func TestAssetURLAdapter_CSRRankImageURLOnyx(t *testing.T) {
	t.Parallel()
	a := NewAssetURLAdapter()
	if got, want := a.CSRRankImageURLOnyx(), "/static/ranks/halo_infinite/120px-HINF-CSR_Onyx.png"; got != want {
		t.Errorf("CSRRankImageURLOnyx() = %q, want %q", got, want)
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
